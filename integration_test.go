package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Programs that are expected to fail compilation
var compileExpectations = map[string]string{
	"const":            "cannot reassign immutable variable",
	"hash_length_test": "panic: runtime error",
}

// Programs to skip - none, we want to see all failures
var skipPrograms = map[string]bool{}

// Expected exit codes (default is 0 if not specified)
var expectedExitCodes = map[string]int{
	"first": 0,
}

// TestFlapPrograms is an integration test that compiles and runs all .flap programs
// and compares their output against .result files
func TestFlapPrograms(t *testing.T) {
	// Build flapc first
	buildCmd := exec.Command("go", "build", "-o", "flapc", ".")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build flapc: %v\n%s", err, string(output))
	}
	defer os.Remove("flapc")

	// Find all .flap files
	matches, err := filepath.Glob("programs/*.flap")
	if err != nil {
		t.Fatalf("Failed to find .flap files: %v", err)
	}

	if len(matches) == 0 {
		t.Fatal("No .flap files found in programs/ directory")
	}

	// Create build directory
	buildDir := "build"
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("Failed to create build directory: %v", err)
	}
	defer os.RemoveAll(buildDir)

	// Test each program
	for _, srcPath := range matches {
		base := strings.TrimSuffix(filepath.Base(srcPath), ".flap")

		// Skip experimental programs
		if skipPrograms[base] {
			continue
		}

		t.Run(base, func(t *testing.T) {
			testFlapProgram(t, base, srcPath, buildDir)
		})
	}
}

func testFlapProgram(t *testing.T, name, srcPath, buildDir string) {
	executable := filepath.Join(buildDir, name)
	expectedPattern, shouldFailCompile := compileExpectations[name]

	// Compile the program
	compileCmd := exec.Command("./flapc", "-o", executable, srcPath)
	compileOutput, compileErr := compileCmd.CombinedOutput()

	// Check compilation result
	if compileErr != nil {
		if !shouldFailCompile {
			t.Fatalf("Compilation failed unexpectedly: %v\nOutput: %s", compileErr, string(compileOutput))
		}
		// Compilation was expected to fail - check for expected error pattern
		if expectedPattern != "" && !strings.Contains(string(compileOutput), expectedPattern) {
			t.Errorf("Expected error pattern %q not found in output: %s", expectedPattern, string(compileOutput))
		}
		return // Don't try to run if compilation failed
	}

	if shouldFailCompile {
		t.Fatalf("Expected compilation to fail, but it succeeded")
	}

	// Run the program
	expectedExitCode := expectedExitCodes[name]
	runCmd := exec.Command(executable)
	actualOutput, runErr := runCmd.CombinedOutput()

	// Check exit code
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			if exitErr.ExitCode() != expectedExitCode {
				t.Errorf("Expected exit code %d, got %d\nOutput: %s", expectedExitCode, exitErr.ExitCode(), string(actualOutput))
			}
		} else if expectedExitCode == 0 {
			t.Errorf("Program failed to run: %v\nOutput: %s", runErr, string(actualOutput))
		}
	}

	// Compare output with .result file if it exists
	resultPath := strings.TrimSuffix(srcPath, ".flap") + ".result"
	if _, err := os.Stat(resultPath); os.IsNotExist(err) {
		t.Logf("No .result file found at %s - skipping output comparison", resultPath)
		return
	}

	expectedOutput, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("Failed to read .result file: %v", err)
	}

	// Compare outputs line by line
	compareOutputs(t, expectedOutput, actualOutput, name)
}

func compareOutputs(t *testing.T, expected, actual []byte, programName string) {
	// Handle empty expected output
	if len(expected) == 0 {
		if len(actual) != 0 {
			t.Errorf("Expected no output but got: %s", string(actual))
		}
		return
	}

	// Split into lines
	expectedLines := splitLines(expected)
	actualLines := splitLines(actual)

	// Check each expected line appears in actual output
	actualMap := make(map[string]bool)
	for _, line := range actualLines {
		actualMap[line] = true
	}

	for _, expectedLine := range expectedLines {
		if !actualMap[expectedLine] {
			t.Errorf("Missing expected line: %q\nExpected output:\n%s\nActual output:\n%s",
				expectedLine, string(expected), string(actual))
		}
	}

	// Check line counts match
	if len(expectedLines) != len(actualLines) {
		t.Errorf("Expected %d lines but found %d\nExpected output:\n%s\nActual output:\n%s",
			len(expectedLines), len(actualLines), string(expected), string(actual))
	}
}

func splitLines(data []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
