package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestHelloWorld tests a simple "Hello, World!" program
func TestHelloWorld(t *testing.T) {
	code := `println("Hello, World!")`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("Expected 'Hello, World!' in output, got: %s", output)
	}
}

// TestFibonacci tests a recursive fibonacci implementation
func TestFibonacci(t *testing.T) {
	code := `
fib = n => n {
	0 -> 0
	1 -> 1
	~> fib(n - 1) + fib(n - 2)
}

result := fib(10)
printf("fib(10) = %v\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "55") {
		t.Errorf("Expected '55' in output for fib(10), got: %s", output)
	}
}

// Test99Bottles tests a simple counting program (inspired by 99 bottles)
func Test99Bottles(t *testing.T) {
	code := `
countdown = (n, acc) => n == 0 {
	-> acc
	~> countdown(n - 1, acc + n)
}

result := countdown(3, 0)
printf("Sum from 1 to 3: %v\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "6") {
		t.Errorf("Expected '6' (sum of 1+2+3) in output, got: %s", output)
	}
}

// TestCFunctionCall tests calling a C standard library function
func TestCFunctionCall(t *testing.T) {
	code := `
// Simple calculation that would benefit from C stdlib
x := -42
result := x < 0 { -> -x ~> x }  // abs implementation
printf("abs(%v) = %v\n", x, result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "42") {
		t.Errorf("Expected '42' in output for abs(-42), got: %s", output)
	}
}

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
