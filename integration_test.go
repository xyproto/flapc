package main

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// Programs that are expected to fail compilation
var compileExpectations = map[string]string{
	"const":                  "cannot update immutable variable",
	"lambda_bad_syntax_test": "lambda definitions must use '=>'",
}

// Programs to skip entirely
var skipPrograms = map[string]bool{}

// Programs to compile but not run (require external libraries beyond libc/libm)
var compileOnlyPrograms = map[string]bool{
	// SDL3 C FFI tests compile successfully but may crash in headless environments
	// SDL3's library constructors attempt to initialize display subsystems even before main()
	// This occurs before SDL_Init(0) can specify headless mode
	// Works fine with a display; headless execution requires SDL_VIDEODRIVER=dummy or X virtual framebuffer
	"c_auto_cast_test":   true,
	"c_ffi_test":         true,
	"c_string_test":      true,
	"sdl3_texture_demo":  true,
	"snake_cstruct_test": true,
	"snake_simple":       true,

	// Network server tests that block waiting for input
	"test_receive_simple": true,
	"snake_visual_demo":  true,
	"snakegame":          true,
	// Raylib tests are compile-only (Raylib is optional, may not be available)
}

// Programs to skip on ARM64 (macOS) - C import not yet implemented
var skipOnARM64 = map[string]bool{
	"c_ffi_test":       true,
	"c_string_test":    true,
	"c_auto_cast_test": true,
	"c_constants_test": true,
	"c_getpid_test":    true,
	"c_import_test":    true,
	"c_simple_test":    true,
}

// Expected exit codes (default is 0 if not specified)
var expectedExitCodes = map[string]int{
	"first":         0,
	"div_zero_test": 1, // Tests division-by-zero error handling
}

// TestFlapPrograms is an integration test that compiles and runs all .flap testprograms
// and compares their output against .result files
func TestFlapPrograms(t *testing.T) {
	// Find all .flap files
	matches, err := filepath.Glob("testprograms/*.flap")
	if err != nil {
		t.Fatalf("Failed to find .flap files: %v", err)
	}

	if len(matches) == 0 {
		t.Fatal("No .flap files found in testprograms/ directory")
	}

	// Test each program
	for _, srcPath := range matches {
		base := strings.TrimSuffix(filepath.Base(srcPath), ".flap")

		// Skip experimental testprograms
		if skipPrograms[base] {
			continue
		}

		t.Run(base, func(t *testing.T) {
			// t.Parallel() // DISABLED: compiler has race conditions with parallel execution

			// Skip ARM64-incompatible testprograms on macOS
			if runtime.GOARCH == "arm64" && skipOnARM64[base] {
				t.Skipf("Skipping %s on ARM64 (C import not yet implemented)", base)
				return
			}

			// Use t.TempDir() for thread-safe temporary directory
			buildDir := t.TempDir()

			testFlapProgram(t, base, srcPath, buildDir)
		})
	}
}

func testFlapProgram(t *testing.T, name, srcPath, buildDir string) {
	executable := filepath.Join(buildDir, name)
	expectedPattern, shouldFailCompile := compileExpectations[name]

	// Compile the program using Go API directly
	// Use current platform for testing
	platform := GetDefaultPlatform()
	compileErr := CompileFlap(srcPath, executable, platform)

	// Check compilation result
	if compileErr != nil {
		if !shouldFailCompile {
			t.Fatalf("Compilation failed unexpectedly: %v", compileErr)
		}
		// Compilation was expected to fail - check for expected error pattern
		if expectedPattern != "" && !strings.Contains(compileErr.Error(), expectedPattern) {
			t.Errorf("Expected error pattern %q not found in error: %v", expectedPattern, compileErr)
		}
		return // Don't try to run if compilation failed
	}

	if shouldFailCompile {
		t.Fatalf("Expected compilation to fail, but it succeeded")
	}

	// Check if this is a compile-only program (e.g., requires runtime libraries)
	if compileOnlyPrograms[name] {
		t.Logf("Program %s compiled successfully (compile-only test, not run)", name)
		return
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

	// Check line counts match first
	if len(expectedLines) != len(actualLines) {
		t.Errorf("Expected %d lines but found %d\nExpected output:\n%s\nActual output:\n%s",
			len(expectedLines), len(actualLines), string(expected), string(actual))
		return
	}

	// Compare line by line in order
	for i, expectedLine := range expectedLines {
		actualLine := actualLines[i]

		// Check for wildcard pattern: * matches any number
		// Supported patterns: "value = *", "timestamp: *", or just "*"
		// But not "6 * 7" (multiplication) or "i=0:   *" (ASCII art with multiple spaces)
		hasWildcardSuffix := false
		if strings.HasSuffix(expectedLine, " *") && !strings.HasSuffix(expectedLine, "  *") {
			// Ends with exactly one space before *, not multiple spaces
			prefix := strings.TrimSuffix(expectedLine, " *")
			hasWildcardSuffix = strings.TrimSpace(prefix) != ""
		}
		isWildcard := hasWildcardSuffix || expectedLine == "*" ||
			(strings.Contains(expectedLine, "* ") && !strings.Contains(expectedLine, " * "))

		if isWildcard {
			// Pattern matching: * matches any number (integer or float)
			pattern := regexp.QuoteMeta(expectedLine)
			pattern = strings.ReplaceAll(pattern, "\\*", "[-+]?[0-9]+(?:\\.[0-9]+)?(?:[eE][-+]?[0-9]+)?")
			re := regexp.MustCompile("^" + pattern + "$")
			if !re.MatchString(actualLine) {
				t.Errorf("Line %d pattern mismatch:\nPattern: %q\nActual:  %q",
					i+1, expectedLine, actualLine)
			}
			continue
		}

		// Direct comparison - if bytes are identical, lines match
		if expectedLine != actualLine {
			t.Errorf("Line %d mismatch:\nExpected: %q\nActual:   %q\nExpected bytes: %v\nActual bytes:   %v",
				i+1, expectedLine, actualLine, []byte(expectedLine), []byte(actualLine))
		}
	}
}

func splitLines(data []byte) []string {
	var lines []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		// Trim trailing whitespace from each line for comparison
		lines = append(lines, strings.TrimRight(scanner.Text(), " \t"))
	}
	return lines
}
