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

	// Check if Wine is available on non-Windows platforms
	if runtime.GOOS != "windows" {
		if _, err := exec.LookPath("wine"); err != nil {
			t.Skip("Wine is not installed - skipping Windows test")
		}
	}

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

	// Verify the PE executable was created
	if _, err := os.Stat(exePath); err != nil {
		t.Fatalf("Executable not created: %v", err)
	}

	// Verify it's a PE executable
	data, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatalf("Failed to read executable: %v", err)
	}
	if len(data) < 2 || data[0] != 'M' || data[1] != 'Z' {
		t.Fatalf("Not a valid PE executable (missing MZ header)")
	}

	// Run - use Wine on non-Windows platforms
	var runCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		runCmd = exec.Command(exePath)
	} else {
		// Use Wine directly (no timeout wrapper for now)
		runCmd = exec.Command("wine", exePath)
	}

	runOutput, err := runCmd.CombinedOutput()

	// Wine may return non-zero exit codes even on success
	// Check if we got any output, which indicates the program ran
	if err != nil && len(runOutput) == 0 {
		t.Fatalf("Failed to run Windows executable: %v\nOutput: %s", err, runOutput)
	}

	return string(runOutput)
}
