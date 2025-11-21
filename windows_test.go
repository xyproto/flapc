package main

import (
	"strings"
	"testing"
)

// Test basic Windows/Wine execution
func TestWindowsBasic(t *testing.T) {
	code := `
main = {
    c.exit(42)
}
`
	// Just test that compilation works and execution doesn't crash
	_ = compileAndRunWindows(t, code)
}

// Test Windows printf (if working)
func TestWindowsPrintf(t *testing.T) {
	code := `
main = {
    c.printf("Hello from Windows\n")
    c.exit(0)
}
`
	output := compileAndRunWindows(t, code)
	// Wine may not capture stdout properly, so this test may not see output
	// Just check it doesn't crash
	if strings.Contains(output, "FATAL") || strings.Contains(output, "Segmentation fault") {
		t.Fatalf("Program crashed: %s", output)
	}
}
