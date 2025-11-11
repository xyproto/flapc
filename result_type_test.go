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
