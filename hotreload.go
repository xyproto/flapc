package main

import (
	"debug/elf"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// CodePage represents an allocated executable memory page
type CodePage struct {
	addr      uintptr
	size      int
	code      []byte
	allocated time.Time
}

// HotReloadManager manages hot code reloading
type HotReloadManager struct {
	activePages  map[string]*CodePage  // function name -> active code page
	oldPages     []*CodePage           // pages pending cleanup
	gracePeriod  time.Duration         // grace period before freeing old code
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager() *HotReloadManager {
	return &HotReloadManager{
		activePages: make(map[string]*CodePage),
		oldPages:    make([]*CodePage, 0),
		gracePeriod: 1 * time.Second, // 1 second grace period
	}
}

// AllocateExecutablePage allocates a memory page with read+write+execute permissions
func (hrm *HotReloadManager) AllocateExecutablePage(size int) (*CodePage, error) {
	// Round up to page size (4KB)
	pageSize := 4096
	allocSize := ((size + pageSize - 1) / pageSize) * pageSize

	// Allocate memory with mmap
	// PROT_READ | PROT_WRITE | PROT_EXEC = 1 | 2 | 4 = 7
	// MAP_PRIVATE | MAP_ANONYMOUS = 2 | 32 = 34
	addr, _, errno := syscall.Syscall6(
		syscall.SYS_MMAP,
		0,                           // addr (let kernel choose)
		uintptr(allocSize),          // length
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC, // prot
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS, // flags
		0,                                         // fd
		0,                                         // offset
	)

	if errno != 0 {
		return nil, fmt.Errorf("mmap failed: %v", errno)
	}

	page := &CodePage{
		addr:      addr,
		size:      allocSize,
		code:      make([]byte, 0, size),
		allocated: time.Now(),
	}

	return page, nil
}

// CopyCode copies machine code to the executable page
func (page *CodePage) CopyCode(code []byte) error {
	if len(code) > page.size {
		return fmt.Errorf("code size %d exceeds page size %d", len(code), page.size)
	}

	// Copy code to the executable page
	dst := unsafe.Slice((*byte)(unsafe.Pointer(page.addr)), page.size)
	copy(dst, code)
	page.code = code

	return nil
}

// GetAddress returns the address of the code page
func (page *CodePage) GetAddress() uintptr {
	return page.addr
}

// LoadHotFunction loads new code for a hot function
func (hrm *HotReloadManager) LoadHotFunction(name string, code []byte) (uintptr, error) {
	// Allocate new executable page
	newPage, err := hrm.AllocateExecutablePage(len(code))
	if err != nil {
		return 0, fmt.Errorf("failed to allocate page: %v", err)
	}

	// Copy code to page
	if err := newPage.CopyCode(code); err != nil {
		hrm.FreePage(newPage)
		return 0, fmt.Errorf("failed to copy code: %v", err)
	}

	// If there's an old version, mark it for cleanup
	if oldPage, exists := hrm.activePages[name]; exists {
		hrm.oldPages = append(hrm.oldPages, oldPage)
	}

	// Set as active page
	hrm.activePages[name] = newPage

	// Clean up old pages after grace period
	go hrm.cleanupOldPages()

	return newPage.GetAddress(), nil
}

// cleanupOldPages removes old code pages after grace period
func (hrm *HotReloadManager) cleanupOldPages() {
	time.Sleep(hrm.gracePeriod)

	now := time.Now()
	remaining := make([]*CodePage, 0)

	for _, page := range hrm.oldPages {
		if now.Sub(page.allocated) >= hrm.gracePeriod {
			// Safe to free this page
			hrm.FreePage(page)
		} else {
			// Keep for next cleanup
			remaining = append(remaining, page)
		}
	}

	hrm.oldPages = remaining
}

// FreePage frees an allocated code page
func (hrm *HotReloadManager) FreePage(page *CodePage) error {
	if page.addr == 0 {
		return nil
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_MUNMAP,
		page.addr,
		uintptr(page.size),
		0,
	)

	if errno != 0 {
		return fmt.Errorf("munmap failed: %v", errno)
	}

	page.addr = 0
	return nil
}

// UpdateFunctionPointer atomically updates a function pointer in the hot function table
func UpdateFunctionPointer(tableAddr uintptr, index int, newAddr uintptr) {
	// Calculate the address of the function pointer in the table
	ptrAddr := tableAddr + uintptr(index*8) // 8 bytes per pointer

	// Atomic write (8-byte aligned write is atomic on x86-64)
	ptr := (*uintptr)(unsafe.Pointer(ptrAddr))
	*ptr = newAddr
}

// ExtractFunctionCode extracts machine code for a specific function from an ELF binary
func ExtractFunctionCode(elfPath string, functionName string) ([]byte, error) {
	// Open ELF file
	elfFile, err := elf.Open(elfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ELF: %v", err)
	}
	defer elfFile.Close()

	// Get symbol table
	symbols, err := elfFile.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to read symbols: %v", err)
	}

	// Find the function symbol
	var funcSym *elf.Symbol
	for _, sym := range symbols {
		if sym.Name == functionName && elf.ST_TYPE(sym.Info) == elf.STT_FUNC {
			funcSym = &sym
			break
		}
	}

	if funcSym == nil {
		return nil, fmt.Errorf("function '%s' not found in symbol table", functionName)
	}

	// Get the .text section
	textSection := elfFile.Section(".text")
	if textSection == nil {
		return nil, fmt.Errorf(".text section not found")
	}

	// Read the entire .text section
	textData, err := textSection.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read .text section: %v", err)
	}

	// Calculate function offset within .text section
	funcOffset := funcSym.Value - textSection.Addr
	funcSize := funcSym.Size

	// Validate bounds
	if funcOffset < 0 || funcOffset+funcSize > uint64(len(textData)) {
		return nil, fmt.Errorf("function bounds invalid: offset=%d, size=%d, text_size=%d",
			funcOffset, funcSize, len(textData))
	}

	// Extract function code
	code := make([]byte, funcSize)
	copy(code, textData[funcOffset:funcOffset+funcSize])

	return code, nil
}

// ReloadHotFunction is the main entry point for hot reloading a function
func (hrm *HotReloadManager) ReloadHotFunction(name string, code []byte, tableAddr uintptr, tableIndex int) error {
	// Load the new code into executable memory
	newAddr, err := hrm.LoadHotFunction(name, code)
	if err != nil {
		return err
	}

	// Atomically update the function pointer table
	UpdateFunctionPointer(tableAddr, tableIndex, newAddr)

	if VerboseMode {
		fmt.Printf("Hot reloaded function '%s' at address 0x%x\n", name, newAddr)
	}

	return nil
}
