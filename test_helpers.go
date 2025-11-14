package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// FlapResult holds the result of running a Flap program
type FlapResult struct {
	Stdout       string
	Stderr       string
	ExitCode     int
	CompileError string
}

// TestResult is an alias for backwards compatibility
type TestResult = FlapResult

// expectOutput checks if the stdout matches expected output
func (r *FlapResult) expectOutput(t *testing.T, expected string) {
	t.Helper()
	if r.Stdout != expected {
		t.Errorf("Output mismatch:\nExpected: %q\nGot:      %q", expected, r.Stdout)
	}
}

// runFlapProgram compiles and runs a Flap program from source code
func runFlapProgram(t *testing.T, source string) FlapResult {
	t.Helper()

	// Create temporary directory for test
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "test.flap")
	exeFile := filepath.Join(tmpDir, "test")

	// Write source to file
	if err := os.WriteFile(srcFile, []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write source file: %v", err)
	}

	// Compile using Go API
	platform := GetDefaultPlatform()
	compileErr := CompileFlap(srcFile, exeFile, platform)

	result := FlapResult{
		Stdout:       "",
		Stderr:       "",
		ExitCode:     0,
		CompileError: "",
	}

	if compileErr != nil {
		result.ExitCode = 1
		result.CompileError = compileErr.Error()
		result.Stderr = compileErr.Error()
		return result
	}

	// Run the executable
	cmd := exec.Command(exeFile)
	output, err := cmd.CombinedOutput()

	result.Stdout = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			// Failed to run at all
			t.Fatalf("Failed to execute program: %v", err)
		}
	} else {
		result.ExitCode = 0
	}

	// Split stdout/stderr if needed (combined output doesn't separate them)
	// For now, all output goes to Stdout
	if strings.Contains(result.Stdout, "error") || strings.Contains(result.Stdout, "Error") {
		result.Stderr = result.Stdout
	}

	return result
}
