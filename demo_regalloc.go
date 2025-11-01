//go:build demo
// +build demo

package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Println("=== Register Allocator Demonstration ===\n")

	// Example 1: Simple loop with 3 variables
	ra := NewRegisterAllocator(ArchX86_64)

	fmt.Println("Example 1: Loop with 3 variables")
	fmt.Println("for i := 0; i < 10; i++ { sum += i; temp = i * 2 }")
	fmt.Println()

	ra.BeginVariable("i")
	ra.AdvancePosition()
	ra.BeginVariable("sum")
	ra.AdvancePosition()
	ra.BeginVariable("temp")
	ra.AdvancePosition()

	for iter := 0; iter < 10; iter++ {
		ra.UseVariable("i")
		ra.UseVariable("sum")
		ra.UseVariable("temp")
		ra.AdvancePosition()
	}

	ra.EndVariable("i")
	ra.EndVariable("sum")
	ra.EndVariable("temp")

	ra.AllocateRegisters()
	ra.PrintAllocation()

	// Example 2: Spilling
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("Example 2: Too many variables (forces spilling)")
	fmt.Println(strings.Repeat("=", 60) + "\n")

	ra2 := NewRegisterAllocator(ArchX86_64)

	varNames := []string{"a", "b", "c", "d", "e", "f", "g"}
	for _, name := range varNames {
		ra2.BeginVariable(name)
		ra2.AdvancePosition()
	}

	for _, name := range varNames {
		ra2.UseVariable(name)
	}
	ra2.AdvancePosition()

	for _, name := range varNames {
		ra2.EndVariable(name)
	}

	ra2.AllocateRegisters()
	ra2.PrintAllocation()
}
