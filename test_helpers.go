package main

import (
	"bytes"
	"fmt"
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

// runFlapProgram compiles and runs a Flap program, returning full output
// Executables are created in isolated temp directories and cleaned up automatically
func runFlapProgram(t *testing.T, source string) FlapResult {
	t.Helper()

	// Try /dev/shm first (faster on Linux), fall back to os.TempDir()
	tmpBase := "/dev/shm"
	if _, err := os.Stat(tmpBase); os.IsNotExist(err) {
		tmpBase = os.TempDir()
	}

	// Create a unique test directory to isolate from sibling .flap files
	// This prevents the compiler from loading other test files as dependencies
	tmpDir, err := os.MkdirTemp(tmpBase, "flapc_test_*")
	if err != nil {
		return FlapResult{CompileError: fmt.Sprintf("Failed to create test dir: %v", err)}
	}
	defer os.RemoveAll(tmpDir)

	// Create test files in isolated directory
	tmpSrc := filepath.Join(tmpDir, "test.flap")
	tmpExe := filepath.Join(tmpDir, "test")

	// Write source
	if err := os.WriteFile(tmpSrc, []byte(source), 0644); err != nil {
		return FlapResult{CompileError: fmt.Sprintf("Failed to write source: %v", err)}
	}

	// Compile
	platform := GetDefaultPlatform()
	if err := CompileFlap(tmpSrc, tmpExe, platform); err != nil {
		return FlapResult{CompileError: err.Error()}
	}

	// Run with timeout, capturing stdout and stderr separately
	cmd := exec.Command("timeout", "5s", tmpExe)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Other error (timeout, etc)
			exitCode = -1
			stderr.WriteString(fmt.Sprintf("\nExecution error: %v", err))
		}
	}

	return FlapResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
	}
}

// expectOutput checks stdout matches expected
func (r FlapResult) expectOutput(t *testing.T, expected string) {
	t.Helper()
	if r.CompileError != "" {
		t.Fatalf("Compilation failed: %s", r.CompileError)
	}
	if r.ExitCode != 0 {
		t.Errorf("Program exited with code %d\nStderr: %s\nStdout: %s", r.ExitCode, r.Stderr, r.Stdout)
	}
	if r.Stdout != expected {
		t.Errorf("Output mismatch:\nExpected:\n%s\nActual:\n%s", expected, r.Stdout)
	}
}

// expectExitCode checks the exit code
func (r FlapResult) expectExitCode(t *testing.T, expected int) {
	t.Helper()
	if r.CompileError != "" {
		t.Fatalf("Compilation failed: %s", r.CompileError)
	}
	if r.ExitCode != expected {
		t.Errorf("Exit code mismatch: expected %d, got %d\nStderr: %s", expected, r.ExitCode, r.Stderr)
	}
}

// expectStderr checks stderr contains expected text
func (r FlapResult) expectStderr(t *testing.T, expected string) {
	t.Helper()
	if r.CompileError != "" {
		t.Fatalf("Compilation failed: %s", r.CompileError)
	}
	if !strings.Contains(r.Stderr, expected) {
		t.Errorf("Stderr doesn't contain expected text:\nExpected substring: %s\nActual stderr: %s", expected, r.Stderr)
	}
}

// expectCompilationError checks that compilation fails with expected error
func (r FlapResult) expectCompilationError(t *testing.T, expectedErr string) {
	t.Helper()
	if r.CompileError == "" {
		t.Fatalf("Expected compilation to fail with '%s', but it succeeded", expectedErr)
	}
	if !strings.Contains(r.CompileError, expectedErr) {
		t.Errorf("Compilation error doesn't match:\nExpected substring: %s\nActual error: %s", expectedErr, r.CompileError)
	}
}

// runTestProgram is a helper for running test programs from files
func runTestProgram(t *testing.T, name, path string) {
	t.Helper()

	// Read source
	source, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("Cannot read %s: %v", path, err)
		return
	}

	// Read expected output
	resultPath := strings.TrimSuffix(path, ".flap") + ".result"
	expected := ""
	if data, err := os.ReadFile(resultPath); err == nil {
		expected = string(data)
	}

	// Run test
	result := runFlapProgram(t, string(source))
	if expected != "" {
		result.expectOutput(t, expected)
	}
}
