package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Confidence that this function is working: 95%
// compileAndRun is a helper function that compiles and runs Flap code,
// returning the output
func compileAndRun(t *testing.T, code string) string {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Write source file
	srcFile := filepath.Join(tmpDir, "test.flap")
	if err := os.WriteFile(srcFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Compile
	exePath := filepath.Join(tmpDir, "test")
	cmd := exec.Command("./flapc", "-o", exePath, srcFile)
	compileOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Compilation failed: %v\nOutput: %s", err, compileOutput)
	}

	// Run
	cmd = exec.Command(exePath)
	runOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Execution failed: %v\nOutput: %s", err, runOutput)
	}

	return string(runOutput)
}
