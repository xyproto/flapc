package main

import (
	"os"
	"os/exec"
	"testing"
)

// TestPrintfWithStringLiteral tests printf with string literals
func TestPrintfWithStringLiteral(t *testing.T) {
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
			cmd := exec.Command("./flapc", "-c", tt.code)
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			// Run the binary
			cmd = exec.Command("/tmp/flapc_inline")
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
