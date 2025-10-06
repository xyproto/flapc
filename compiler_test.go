package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestBasicCompilation tests that flapc can compile and generate an executable
func TestBasicCompilation(t *testing.T) {
	eb, err := New("x86_64")
	if err != nil {
		t.Fatalf("Failed to create ExecutableBuilder: %v", err)
	}

	eb.Define("hello", "Hello, World!\n\x00")

	// Generate ELF
	eb.WriteELFHeader()

	bytes := eb.Bytes()
	if len(bytes) == 0 {
		t.Fatal("Generated ELF is empty")
	}

	// Check for ELF magic number
	if bytes[0] != 0x7f || bytes[1] != 'E' || bytes[2] != 'L' || bytes[3] != 'F' {
		t.Fatal("Invalid ELF magic number")
	}
}

// TestArchitectures tests that all supported architectures can be initialized
func TestArchitectures(t *testing.T) {
	archs := []string{"x86_64", "amd64", "aarch64", "arm64", "riscv64"}

	for _, arch := range archs {
		t.Run(arch, func(t *testing.T) {
			eb, err := New(arch)
			if err != nil {
				t.Fatalf("Failed to create ExecutableBuilder for %s: %v", arch, err)
			}

			if eb == nil {
				t.Fatalf("ExecutableBuilder is nil for %s", arch)
			}
		})
	}
}

// TestDynamicLinking tests basic dynamic linking setup
func TestDynamicLinking(t *testing.T) {
	eb, err := New("x86_64")
	if err != nil {
		t.Fatalf("Failed to create ExecutableBuilder: %v", err)
	}

	// Add a library
	glibc := eb.AddLibrary("glibc", "libc.so.6")
	if glibc == nil {
		t.Fatal("Failed to add library")
	}

	// Add a function
	glibc.AddFunction("printf", CTypeInt,
		Parameter{Name: "format", Type: CTypePointer},
	)

	// Import the function
	err = eb.ImportFunction("glibc", "printf")
	if err != nil {
		t.Fatalf("Failed to import function: %v", err)
	}

	// Check that the function is imported
	if len(eb.dynlinker.ImportedFuncs) == 0 {
		t.Fatal("No functions imported")
	}
}

// TestELFSections tests that dynamic sections can be created
func TestELFSections(t *testing.T) {
	ds := NewDynamicSections()

	// Add needed library
	ds.AddNeeded("libc.so.6")

	// Add symbols
	ds.AddSymbol("printf", STB_GLOBAL, STT_FUNC)
	ds.AddSymbol("exit", STB_GLOBAL, STT_FUNC)

	// Build tables
	ds.buildSymbolTable()
	ds.buildHashTable()

	// Check sizes
	if ds.dynsym.Len() == 0 {
		t.Fatal("Symbol table is empty")
	}
	if ds.dynstr.Len() == 0 {
		t.Fatal("String table is empty")
	}
	if ds.hash.Len() == 0 {
		t.Fatal("Hash table is empty")
	}
}

// TestPLTGOT tests PLT and GOT generation
func TestPLTGOT(t *testing.T) {
	ds := NewDynamicSections()

	functions := []string{"printf", "exit"}

	// Add symbols
	for _, fn := range functions {
		ds.AddSymbol(fn, STB_GLOBAL, STT_FUNC)
	}

	// Generate PLT and GOT
	pltBase := uint64(0x402000)
	gotBase := uint64(0x403000)
	dynamicAddr := uint64(0x403000)

	ds.GeneratePLT(functions, gotBase, pltBase)
	ds.GenerateGOT(functions, dynamicAddr, pltBase)

	// Check sizes
	// PLT[0] is 16 bytes, plus 16 bytes per function
	expectedPLTSize := 16 + len(functions)*16
	if ds.plt.Len() != expectedPLTSize {
		t.Errorf("PLT size = %d, want %d", ds.plt.Len(), expectedPLTSize)
	}

	// GOT has 3 reserved entries (24 bytes) plus one per function (8 bytes each)
	expectedGOTSize := 24 + len(functions)*8
	if ds.got.Len() != expectedGOTSize {
		t.Errorf("GOT size = %d, want %d", ds.got.Len(), expectedGOTSize)
	}
}

// TestExecutableGeneration tests full executable generation
func TestExecutableGeneration(t *testing.T) {
	// Only run if we have write permissions
	tmpfile := "/tmp/flapc_test_output"
	defer os.Remove(tmpfile)

	eb, err := New("x86_64")
	if err != nil {
		t.Fatalf("Failed to create ExecutableBuilder: %v", err)
	}

	eb.Define("hello", "Test\n\x00")
	eb.useDynamicLinking = false

	// Generate static ELF
	eb.WriteELFHeader()

	// Write to file
	err = os.WriteFile(tmpfile, eb.Bytes(), 0755)
	if err != nil {
		t.Fatalf("Failed to write executable: %v", err)
	}

	// Check that it's a valid ELF
	cmd := exec.Command("file", tmpfile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("file output: %s", output)
	}

	// Just verify the file was created with executable permissions
	info, err := os.Stat(tmpfile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// Check that owner has execute permission (umask may affect group/other)
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("File not executable: permissions = %o", info.Mode().Perm())
	}
}

// TestCTypeSize tests CType size calculations
func TestCTypeSize(t *testing.T) {
	tests := []struct {
		ctype    CType
		expected int
	}{
		{CTypeVoid, 0},
		{CTypeInt, 4},
		{CTypeUInt, 4},
		{CTypeChar, 1},
		{CTypeFloat, 4},
		{CTypeDouble, 8},
		{CTypeLong, 8},
		{CTypeULong, 8},
		{CTypePointer, 8},
	}

	for _, tt := range tests {
		t.Run(tt.ctype.String(), func(t *testing.T) {
			size := tt.ctype.Size(MachineX86_64)
			if size != tt.expected {
				t.Errorf("Size(%s) = %d, want %d", tt.ctype, size, tt.expected)
			}
		})
	}
}
