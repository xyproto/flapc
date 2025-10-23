package main

import (
	"debug/macho"
	"os"
	"path/filepath"
	"testing"
)

// TestARM64BasicCompilation tests that ARM64 code can be compiled
// Note: We can't execute the binaries on Linux, but we can verify they compile and have correct structure
func TestARM64BasicCompilation(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		wantText int // expected minimum text size in bytes
	}{
		{
			name:     "exit_zero",
			code:     "exit(0)",
			wantText: 40, // prologue + exit syscall + epilogue
		},
		{
			name:     "exit_code",
			code:     "exit(42)",
			wantText: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(os.TempDir(), "test_arm64_"+tt.name+".flap")
			outFile := filepath.Join(os.TempDir(), "test_arm64_"+tt.name)
			defer os.Remove(tmpFile)
			defer os.Remove(outFile)

			// Write test program
			if err := os.WriteFile(tmpFile, []byte(tt.code), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Compile for ARM64 macOS
			platform := Platform{Arch: ArchARM64, OS: OSDarwin}
			err := CompileFlap(tmpFile, outFile, platform)
			if err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Verify file exists
			info, err := os.Stat(outFile)
			if err != nil {
				t.Fatalf("Output file not created: %v", err)
			}

			// Verify file is executable
			if info.Mode()&0111 == 0 {
				t.Errorf("Output file is not executable")
			}

			machoFile, err := macho.Open(outFile)
			if err != nil {
				t.Fatalf("failed to parse Mach-O: %v", err)
			}
			defer machoFile.Close()

			if machoFile.Type != macho.TypeExec {
				t.Errorf("unexpected Mach-O type: got %v, want %v", machoFile.Type, macho.TypeExec)
			}
			if machoFile.Cpu != macho.CpuArm64 {
				t.Errorf("unexpected Mach-O CPU: got %v, want %v", machoFile.Cpu, macho.CpuArm64)
			}
		})
	}
}
