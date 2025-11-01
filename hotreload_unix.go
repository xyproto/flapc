//go:build linux
// +build linux

package main

import (
	"debug/elf"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

type CodePage struct {
	addr      uintptr
	size      int
	code      []byte
	allocated time.Time
}

type HotReloadManager struct {
	activePages map[string]*CodePage
	oldPages    []*CodePage
	gracePeriod time.Duration
}

func NewHotReloadManager() *HotReloadManager {
	return &HotReloadManager{
		activePages: make(map[string]*CodePage),
		oldPages:    make([]*CodePage, 0),
		gracePeriod: 1 * time.Second,
	}
}

func (hrm *HotReloadManager) AllocateExecutablePage(size int) (*CodePage, error) {
	pageSize := 4096
	allocSize := ((size + pageSize - 1) / pageSize) * pageSize

	addr, _, errno := syscall.Syscall6(
		syscall.SYS_MMAP,
		0,
		uintptr(allocSize),
		syscall.PROT_READ|syscall.PROT_WRITE|syscall.PROT_EXEC,
		syscall.MAP_PRIVATE|syscall.MAP_ANON,
		0,
		0,
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

func (page *CodePage) CopyCode(code []byte) error {
	if len(code) > page.size {
		return fmt.Errorf("code size %d exceeds page size %d", len(code), page.size)
	}

	dst := unsafe.Slice((*byte)(unsafe.Pointer(page.addr)), page.size)
	copy(dst, code)
	page.code = code

	return nil
}

func (page *CodePage) GetAddress() uintptr {
	return page.addr
}

func (hrm *HotReloadManager) LoadHotFunction(name string, code []byte) (uintptr, error) {
	newPage, err := hrm.AllocateExecutablePage(len(code))
	if err != nil {
		return 0, fmt.Errorf("failed to allocate page: %v", err)
	}

	if err := newPage.CopyCode(code); err != nil {
		hrm.FreePage(newPage)
		return 0, fmt.Errorf("failed to copy code: %v", err)
	}

	if oldPage, exists := hrm.activePages[name]; exists {
		hrm.oldPages = append(hrm.oldPages, oldPage)
	}

	hrm.activePages[name] = newPage

	go hrm.cleanupOldPages()

	return newPage.GetAddress(), nil
}

func (hrm *HotReloadManager) cleanupOldPages() {
	time.Sleep(hrm.gracePeriod)

	now := time.Now()
	remaining := make([]*CodePage, 0)

	for _, page := range hrm.oldPages {
		if now.Sub(page.allocated) >= hrm.gracePeriod {
			hrm.FreePage(page)
		} else {
			remaining = append(remaining, page)
		}
	}

	hrm.oldPages = remaining
}

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

func UpdateFunctionPointer(tableAddr uintptr, index int, newAddr uintptr) {
	ptrAddr := tableAddr + uintptr(index*8)
	ptr := (*uintptr)(unsafe.Pointer(ptrAddr))
	*ptr = newAddr
}

func ExtractFunctionCode(elfPath string, functionName string) ([]byte, error) {
	elfFile, err := elf.Open(elfPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open ELF: %v", err)
	}
	defer elfFile.Close()

	symbols, err := elfFile.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to read symbols: %v", err)
	}

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

	textSection := elfFile.Section(".text")
	if textSection == nil {
		return nil, fmt.Errorf(".text section not found")
	}

	textData, err := textSection.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read .text section: %v", err)
	}

	funcOffset := funcSym.Value - textSection.Addr
	funcSize := funcSym.Size

	if funcOffset < 0 || funcOffset+funcSize > uint64(len(textData)) {
		return nil, fmt.Errorf("function bounds invalid: offset=%d, size=%d, text_size=%d",
			funcOffset, funcSize, len(textData))
	}

	code := make([]byte, funcSize)
	copy(code, textData[funcOffset:funcOffset+funcSize])

	return code, nil
}

func (hrm *HotReloadManager) ReloadHotFunction(name string, code []byte, tableAddr uintptr, tableIndex int) error {
	newAddr, err := hrm.LoadHotFunction(name, code)
	if err != nil {
		return err
	}

	UpdateFunctionPointer(tableAddr, tableIndex, newAddr)

	if VerboseMode {
		fmt.Printf("Hot reloaded function '%s' at address 0x%x\n", name, newAddr)
	}

	return nil
}
