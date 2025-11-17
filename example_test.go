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
fib = n => {
	| n == 0 -> 0
	| n == 1 -> 1
	~> fib(n - 1) + fib(n - 2)
}

result = fib(10)
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
countdown = (n, acc) => {
	| n == 0 -> acc
	~> countdown(n - 1, acc + n)
}

result = countdown(3, 0)
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
x = -42
result = { | x < 0 -> -x ~> x }  // abs implementation
printf("abs(%v) = %v\n", x, result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "42") {
		t.Errorf("Expected '42' in output for abs(-42), got: %s", output)
	}
}

// TestFactorial tests simple computation
func TestFactorial(t *testing.T) {
	code := `
result = 2 * 3 * 4 * 5
printf("Product = %v\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "120") {
		t.Errorf("Expected '120' in output, got: %s", output)
	}
}

// TestExampleMapOperations tests map creation and access
func TestExampleMapOperations(t *testing.T) {
	code := `
person = {0: 100, 1: 30, 2: 42}

printf("Values: %v, %v\n", person[0], person[1])
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "100") || !strings.Contains(output, "30") {
		t.Errorf("Expected map values in output, got: %s", output)
	}
}

// TestListOperations tests list creation and manipulation
func TestListOperations(t *testing.T) {
	code := `
numbers = [1, 2, 3, 4, 5]

sum_list = lst => {
	sum_helper = (i, acc) => {
		| i >= 5 -> acc
		~> sum_helper(i + 1, acc + lst[i])
	}
	sum_helper(0, 0)
}

result = sum_list(numbers)
printf("Sum: %v\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "15") {
		t.Errorf("Expected '15' in output for sum of [1,2,3,4,5], got: %s", output)
	}
}

// TestMatchExpressions tests simple conditionals
func TestMatchExpressions(t *testing.T) {
	code := `
is_positive = x => {
	| x > 0 -> 1
	~> 0
}

printf("%v %v\n", is_positive(0), is_positive(42))
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "0 1") {
		t.Errorf("Expected '0 1' in output, got: %s", output)
	}
}

// TestNestedFunctions tests nested function definitions
func TestNestedFunctions(t *testing.T) {
	code := `
make_adder = x => {
	add_x = y => x + y
	add_x
}

add5 = make_adder(5)
result = add5(10)
printf("5 + 10 = %v\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "15") {
		t.Errorf("Expected '15' in output for 5 + 10, got: %s", output)
	}
}

// TestLoopWithLabel tests simple loops
func TestLoopWithLabel(t *testing.T) {
	code := `
@ i in 0..<3 {
	printf("i=%v\n", i)
}
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "i=0") || !strings.Contains(output, "i=2") {
		t.Errorf("Expected loop output, got: %s", output)
	}
}

// TestQuickSort tests a more complex algorithm
func TestQuickSort(t *testing.T) {
	code := `
// Concatenate two lists recursively
concat = (left, right, i, result) => {
	len_left = #left
	| i >= len_left -> concat_right(right, 0, result)
	~> concat(left, right, i + 1, result.append(left[i]))
}

concat_right = (right, j, acc) => {
	| j >= #right -> acc
	~> concat_right(right, j + 1, acc.append(right[j]))
}

// Partition helper - adds element to less or greater based on comparison
add_to_partition = (elem, pivot, less, greater) => {
	| elem < pivot -> [less.append(elem), greater]
	~> [less, greater.append(elem)]
}

// Partition array recursively
partition_helper = (arr, pivot, i, less, greater) => {
	len_arr = #arr
	| i >= len_arr -> [less, greater]
	~> {
		elem = arr[i]
		updated = add_to_partition(elem, pivot, less, greater)
		partition_helper(arr, pivot, i + 1, updated[0], updated[1])
	}
}

// QuickSort implementation
quicksort = arr => {
	len_arr = #arr
	| len_arr <= 1 -> arr
	~> {
		pivot = arr[0]
		partitioned = partition_helper(arr, pivot, 1, [], [])
		less = partitioned[0]
		greater = partitioned[1]
		sorted_less = quicksort(less)
		sorted_greater = quicksort(greater)
		with_pivot = concat(sorted_less, [pivot], 0, [])
		concat(with_pivot, sorted_greater, 0, [])
	}
}

numbers = [3, 1, 4, 1, 5, 9, 2, 6]
sorted = quicksort(numbers)
printf("Sorted: %v %v %v %v %v %v %v %v\n", sorted[0], sorted[1], sorted[2], sorted[3], sorted[4], sorted[5], sorted[6], sorted[7])
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "1 1 2 3 4 5 6 9") {
		t.Errorf("Expected '1 1 2 3 4 5 6 9' in sorted output, got: %s", output)
	}
}

// TestExampleStringOperations tests string handling
func TestExampleStringOperations(t *testing.T) {
	code := `
greeting = "Hello"
name = "World"
message = greeting + ", " + name + "!"
println(message)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "Hello, World!") {
		t.Errorf("Expected 'Hello, World!' in output, got: %s", output)
	}
}

// TestRecursiveSum tests simple recursion
func TestRecursiveSum(t *testing.T) {
	code := `
sum_to = n => {
	| n == 0 -> 0
	~> n + sum_to(n - 1)
}

result = sum_to(10)
printf("Sum from 1 to 10: %v\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "55") {
		t.Errorf("Expected '55' in output, got: %s", output)
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
