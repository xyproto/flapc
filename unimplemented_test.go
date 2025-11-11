package main

import (
	"testing"
)

// TestReducePipe tests the ||| (reduce) operator - NOT YET IMPLEMENTED
func TestReducePipe(t *testing.T) {
	t.Skip("Reduce pipe ||| not yet implemented")

	source := `numbers := [1, 2, 3, 4, 5]
result := numbers ||| (acc, x) => acc + x
println(result)
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "15\n")
}

// TestRandomOperator tests the ??? (random) operator - NOT YET IMPLEMENTED
func TestRandomOperator(t *testing.T) {
	t.Skip("Random operator ??? not yet implemented")

	source := `x := ???
y := ???
(x >= 0.0 && x < 1.0 && y >= 0.0 && y < 1.0) {
    -> println("PASS")
    ~> println("FAIL")
}
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "PASS\n")
}
