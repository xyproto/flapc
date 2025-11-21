package main

import (
	"strings"
	"testing"
)

// Test basic Windows/Wine execution
func TestWindowsBasic(t *testing.T) {
	code := `
c.exit(42)
`
	// Just test that compilation works and execution doesn't crash
	_ = compileAndRunWindows(t, code)
}

// Test Windows printf
func TestWindowsPrintf(t *testing.T) {
	code := `
c.printf("Hello from Windows\n")
c.exit(0)
`
	output := compileAndRunWindows(t, code)
	// Check that printf output is captured
	if !strings.Contains(output, "Hello from Windows") {
		t.Errorf("Expected 'Hello from Windows' in output, got: %s", output)
	}
}
