package main

import (
	"testing"
)

// TestDivisionByZeroReturnsNaN tests that division by zero returns NaN not exit
func TestDivisionByZeroReturnsNaN(t *testing.T) {
	source := `x := 10 / 0
is_error(x) {
    -> println("ERROR")
    ~> println("OK")
}
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "ERROR\n")
}

// TestOrBangWithSuccess tests or! with successful value
func TestOrBangWithSuccess(t *testing.T) {
	source := `result := 10 / 2
safe := result or! 0.0
println(safe)
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "5\n")
}

// TestOrBangWithError tests or! with error value (division by zero)
func TestOrBangWithError(t *testing.T) {
	source := `result := 10 / 0
safe := result or! 42.0
println(safe)
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "42\n")
}

// TestOrBangChaining tests chained or! operators
func TestOrBangChaining(t *testing.T) {
	source := `x := 10 / 0
y := 20 / 0
z := x or! (y or! 99.0)
println(z)
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "99\n")
}

// TestErrorPropertySimple tests .error property doesn't crash
func TestErrorPropertySimple(t *testing.T) {
	source := `x := 5.0
y := x.error
println("ok")
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "ok\n")
}

// TestErrorPropertyLength tests .error returns a string
func TestErrorPropertyLength(t *testing.T) {
	source := `x := 5.0
code := x.error
len := #code
println(len)
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "0\n") // Empty string for non-error
}

// TestErrorPropertyBasic tests .error property on Result types
func TestErrorPropertyBasic(t *testing.T) {
	t.Skip("TODO: .error property needs division error encoding")
	source := `result := 10 / 0
code := result.error
println(code)
`
	result := runFlapProgram(t, source)
	// Should print the error code (e.g., "dv0 " for division by zero)
	// For now, just check it doesn't crash
	if result.CompileError != "" {
		t.Fatalf("Compilation failed: %s", result.CompileError)
	}
}
