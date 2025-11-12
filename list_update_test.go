package main

import (
	"testing"
)

// TestListUpdateMinimal tests most basic list update
func TestListUpdateMinimal(t *testing.T) {
	source := `nums := [5]
nums[0] <- 10
println(nums[0])
`
	result := runFlapProgram(t, source)
	if result.ExitCode != 0 {
		t.Logf("Exit code: %d", result.ExitCode)
		t.Logf("Stderr: %s", result.Stderr)
		t.Logf("Stdout: %s", result.Stdout)
		t.FailNow()
	}
	result.expectOutput(t, "10\n")
}

// TestListUpdateBasic tests basic list element update
func TestListUpdateBasic(t *testing.T) {
	source := `arr := [5, 10, 15]
println(arr[0])
arr[0] <- 99
println(arr[0])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "5\n99\n")
}

// TestListUpdateSingleElement tests updating a single-element list
func TestListUpdateSingleElement(t *testing.T) {
	source := `arr := [42]
arr[0] <- 100
println(arr[0])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "100\n")
}

// TestListUpdateMiddleElement tests updating a middle element
func TestListUpdateMiddleElement(t *testing.T) {
	source := `arr := [1, 2, 3, 4, 5]
arr[2] <- 99
println(arr[2])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "99\n")
}

// TestListUpdateLastElement tests updating the last element
func TestListUpdateLastElement(t *testing.T) {
	source := `arr := [10, 20, 30]
arr[2] <- 999
println(arr[2])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "999\n")
}

// TestListUpdateMultiple tests multiple updates
func TestListUpdateMultiple(t *testing.T) {
	source := `nums := [1, 2, 3]
nums[0] <- 10
nums[1] <- 20
nums[2] <- 30
println(nums[0])
println(nums[1])
println(nums[2])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "10\n20\n30\n")
}

// TestListUpdatePreservesOtherElements tests that other elements are unchanged
func TestListUpdatePreservesOtherElements(t *testing.T) {
	source := `arr := [100, 200, 300, 400]
arr[1] <- 999
println(arr[0])
println(arr[1])
println(arr[2])
println(arr[3])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "100\n999\n300\n400\n")
}

// TestConsOperator tests the cons operator (::)
func TestConsOperator(t *testing.T) {
	source := `list1 := [2, 3, 4]
list2 := 1 :: list1
println(list2[0])
println(list2[1])
println(list2[2])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "1\n2\n3\n")
}

// TestTailOperatorBasic tests the tail operator (_)
func TestTailOperatorBasic(t *testing.T) {
	t.Skip("TODO: tail operator needs implementation")
	source := `list := [1, 2, 3, 4]
rest := _list
println(rest[0])
println(rest[1])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "2\n3\n")
}
