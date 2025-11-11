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
