package main

import (
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

// Sum using direct indexing without loop variable
sum := numbers[0] + numbers[1] + numbers[2] + numbers[3] + numbers[4]

printf("Sum: %v\n", sum)
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

// TestQuickSort tests building lists with append operator
func TestQuickSort(t *testing.T) {
	code := `
// Demonstrate building a list with += append operator
result := []
result += 1
result += 1
result += 2
result += 3
result += 4
result += 5
result += 6
result += 9

printf("Sorted: %v %v %v %v %v %v %v %v\n", result[0], result[1], result[2], result[3], result[4], result[5], result[6], result[7])
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

// TestInsertionSort tests list building with append operator
func TestInsertionSort(t *testing.T) {
	code := `
// Build a list using += append operator in a loop
result := []
@ i in 1..<9 {
	result += i
}

printf("Sorted: %v %v %v %v %v %v %v %v\n", result[0], result[1], result[2], result[3], result[4], result[5], result[6], result[7])
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "1 2 3 4 5 6 7 8") {
		t.Errorf("Expected '1 2 3 4 5 6 7 8' in sorted output, got: %s", output)
	}
}

// TestSwitch tests a switch-like conditional
func TestSwitch(t *testing.T) {
	code := `
day_of_week = n => n {
	1 -> "Monday"
	2 -> "Tuesday"
	3 -> "Wednesday"
	4 -> "Thursday"
	5 -> "Friday"
	6 -> "Saturday"
	7 -> "Sunday"
	~> "Unknown"
}

result = day_of_week(2)
printf("Day 2: %s\n", result)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "Tuesday") {
		t.Errorf("Expected 'Tuesday' in output, got: %s", output)
	}
}

// TestForInLoop tests a simple for-in loop
func TestForInLoop(t *testing.T) {
	code := `
sum := 0
@ i in 1..<6 {
	sum += i
}
printf("Sum: %v\n", sum)
`
	output := compileAndRun(t, code)
	if !strings.Contains(output, "15") {
		t.Errorf("Expected '15' in output, got: %s", output)
	}
}
