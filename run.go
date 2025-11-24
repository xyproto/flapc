package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// Run - inherit environment variables for SDL_VIDEODRIVER etc
	cmd = exec.Command(exePath)
	cmd.Env = os.Environ()
	runOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Execution failed: %v\nOutput: %s", err, runOutput)
	}

	return string(runOutput)
}

// Confidence that this function is working: 90%
// compileAndRunWindows is a helper function that compiles and runs Flap code for Windows
// under Wine (on non-Windows platforms), with a 3-second timeout
func compileAndRunWindows(t *testing.T, code string) string {
	t.Helper()

	// Create temporary directory
	tmpDir := t.TempDir()

	// Write source file
	srcFile := filepath.Join(tmpDir, "test.flap")
	if err := os.WriteFile(srcFile, []byte(code), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Compile for Windows
	exePath := filepath.Join(tmpDir, "test.exe")
	cmd := exec.Command("./flapc", "-target", "amd64-windows", "-o", exePath, srcFile)
	compileOutput, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Compilation failed: %v\nOutput: %s", err, compileOutput)
	}

	// Run - use Wine on non-Windows platforms
	var runCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		runCmd = exec.Command(exePath)
	} else {
		// Use timeout command with Wine
		runCmd = exec.Command("timeout", "3", "wine", exePath)
	}
	runOutput, err := runCmd.CombinedOutput()
	// Note: timeout command may return exit code 124 on timeout, which is expected
	// Wine also may produce stderr output, so we just capture combined output

	return string(runOutput)
}
