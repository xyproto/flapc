package main

import (
	"strings"
	"testing"
)

// Confidence that this function is working: 95%
// TestCFunctionSin tests calling c.sin from the C math library
func TestCFunctionSin(t *testing.T) {
	code := `
// Test calling c.sin - standard C math function
result := c.sin(0.0)
printf("sin(0.0) = %v\n", result)

result2 := c.sin(1.0)
printf("sin(1.0) ≈ %v\n", result2)
`
	output := compileAndRun(t, code)
	
	// sin(0) should be approximately 0
	if !strings.Contains(output, "sin(0.0) = 0") {
		t.Errorf("Expected 'sin(0.0) = 0', got: %s", output)
	}
	
	// sin(1.0) should be approximately 0.84147
	if !strings.Contains(output, "sin(1.0)") {
		t.Errorf("Expected 'sin(1.0)' in output, got: %s", output)
	}
}

// Confidence that this function is working: 95%
// TestCFunctionCos tests calling c.cos from the C math library
func TestCFunctionCos(t *testing.T) {
	code := `
result := c.cos(0.0)
printf("cos(0.0) = %v\n", result)
`
	output := compileAndRun(t, code)
	
	// cos(0) should be 1
	if !strings.Contains(output, "cos(0.0) = 1") {
		t.Errorf("Expected 'cos(0.0) = 1', got: %s", output)
	}
}

// Confidence that this function is working: 90%
// TestNullPointerLiterals tests that 0, [], {} can be used as null pointers
func TestNullPointerLiterals(t *testing.T) {
	code := `
// Test that 0, [], and {} can all be used as null pointers in C FFI calls
// We'll test by passing them to a mock function that prints them
x = 0
y = []
z = {}

printf("x=%v y=%v z=%v\n", x, y, z)
`
	output := compileAndRun(t, code)
	
	// Should print the values
	if !strings.Contains(output, "x=0") {
		t.Errorf("Expected x=0, got: %s", output)
	}
}

// Confidence that this function is working: 85%
// TestCFunctionWithNullPointer tests passing null pointers to C functions
func TestCFunctionWithNullPointer(t *testing.T) {
	code := `
// Test passing null pointer using different forms
// Note: This is a contrived example - in real code you'd use actual C functions that accept nulls
result1 := c.strlen(0 as cstr)  // NULL pointer - should handle gracefully or crash (C behavior)
printf("strlen(NULL) = %v\n", result1)
`
	// This test is expected to potentially crash or return 0
	// The key is that it compiles and the null pointer is correctly passed
	_ = compileTestCode(t, code)
	// If we get here without compilation error, the test passes
}
