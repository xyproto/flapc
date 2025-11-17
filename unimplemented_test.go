package main

import (
	"strings"
	"testing"
)

// TestReducePipe tests the ||| (reduce) operator - NOT YET IMPLEMENTED
func TestReducePipe(t *testing.T) {
	t.Skip("Reduce pipe ||| not yet implemented")

	source := `numbers := [1, 2, 3, 4, 5]
result := numbers ||| (acc, x) => acc + x
println(result)
`
	result := compileAndRun(t, source)
	if !strings.Contains(result, "15\n") {
		t.Errorf("Expected output to contain: %s, got: %s", "15\n", result)
	}
}
