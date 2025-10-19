package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// findFlapc tries to find the flapc binary
func findFlapc(t *testing.T) string {
	// Try current directory first
	if _, err := os.Stat("./flapc"); err == nil {
		abs, _ := filepath.Abs("./flapc")
		return abs
	}

	// Get the directory where the test file is located
	// When go test runs, it may change directory, so we need to find
	// the source directory
	if wd, err := os.Getwd(); err == nil {
		// If we're in a Go test temp directory, go back to find the source
		for dir := wd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
			flapPath := filepath.Join(dir, "flapc")
			if info, err := os.Stat(flapPath); err == nil && !info.IsDir() {
				return flapPath
			}
		}
	}

	// Try in PATH
	if path, err := exec.LookPath("flapc"); err == nil {
		return path
	}

	t.Skip("flapc binary not found - run 'make flapc' first")
	return ""
}

// TestPrintfWithStringLiteral tests printf with string literals
func TestPrintfWithStringLiteral(t *testing.T) {
	flapcPath := findFlapc(t)

	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "simple string",
			code:     `printf("Hello, World!\n")`,
			expected: "Hello, World!\n",
		},
		{
			name:     "string with %s format",
			code:     `printf("Test: %s\n", "hello")`,
			expected: "Test: hello\n",
		},
		{
			name:     "number with %g format",
			code:     `printf("Number: %.15g\n", 42)`,
			expected: "Number: 42\n",
		},
		{
			name:     "boolean with %b format",
			code:     `printf("Bool: %b\n", 1)`,
			expected: "Bool: yes\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compile the code
			cmd := exec.Command(flapcPath, "-c", tt.code)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Run the binary
			cmd = exec.Command(filepath.Join(os.TempDir(), "flapc_inline"))
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Execution failed: %v\nOutput: %s", err, output)
			}

			got := string(output)
			if got != tt.expected {
				t.Errorf("Output mismatch:\nGot:      %q\nExpected: %q", got, tt.expected)
			}
		})
	}
}
