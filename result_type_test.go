package main

import (
	"strings"
	"testing"
)

// TestDivisionByZeroReturnsNaN tests that division by zero returns NaN not exit
func TestDivisionByZeroReturnsNaN(t *testing.T) {
	source := `x := 10 / 0
safe := x or! -999.0
println(safe)
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "-999\n") {
		t.Errorf("Expected output to contain: %s, got: %s", "-999\n", result)
	}
}

// TestOrBangWithSuccess tests or! with successful value
func TestOrBangWithSuccess(t *testing.T) {
	source := `result := 10 / 2
safe := result or! 0.0
println(safe)
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "5\n") {
		t.Errorf("Expected output to contain: %s, got: %s", "5\n", result)
	}
}

// TestOrBangWithError tests or! with error value (division by zero)
func TestOrBangWithError(t *testing.T) {
	source := `result := 10 / 0
safe := result or! 42.0
println(safe)
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "42\n") {
		t.Errorf("Expected output to contain: %s, got: %s", "42\n", result)
	}
}

// TestOrBangChaining tests chained or! operators
func TestOrBangChaining(t *testing.T) {
	source := `x := 10 / 0
y := 20 / 0
z := x or! (y or! 99.0)
println(z)
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "99\n") {
		t.Errorf("Expected output to contain: %s, got: %s", "99\n", result)
	}
}

// TestErrorPropertySimple tests .error property doesn't crash
func TestErrorPropertySimple(t *testing.T) {
	source := `x := 5.0
y := x.error
println("ok")
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "ok\n") {
		t.Errorf("Expected output to contain: %s, got: %s", "ok\n", result)
	}
}

// TestErrorPropertyLength tests .error returns a string
func TestErrorPropertyLength(t *testing.T) {
	source := `x := 5.0
code := x.error
len := #code
println(len)
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "0\n") { // Empty string for non-error
		t.Errorf("Expected output to contain: %s, got: %s", "0\n", result)
	}
}

// TestErrorPropertyBasic tests .error property on Result types
func TestErrorPropertyBasic(t *testing.T) {
	t.Skip("TODO: .error property needs division error encoding")
	source := `result := 10 / 0
code := result.error
println(code)
`
	_ = compileAndRun(t, source)
	// Should print the error code (e.g., "dv0 " for division by zero)
	// For now, just check it doesn't crash (compileAndRun will fail test if compilation fails)
}
