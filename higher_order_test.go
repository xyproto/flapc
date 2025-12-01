package main

import (
	"testing"
)

func TestHigherOrderBasic(t *testing.T) {
	code := `
		add_one := x -> x + 1
		apply := (f, x) -> f(x)
		result := apply(add_one, 5)
		println(result)
	`
	result := compileAndRun(t, code)
	if result != "6\n" {
		t.Fatalf("Expected '6\\n', got %q", result)
	}
}

func TestHigherOrderMap(t *testing.T) {
	code := `
		map := (f, xs) -> match xs {
			[] -> []
			_ -> [f(head(xs))] ++ map(f, tail(xs))
		}
		double := x -> x * 2
		result := map(double, [1, 2, 3])
		println(result[0])
		println(result[1])
		println(result[2])
	`
	result := compileAndRun(t, code)
	if result != "2\n4\n6\n" {
		t.Fatalf("Expected '2\\n4\\n6\\n', got %q", result)
	}
}

func TestHigherOrderFilter(t *testing.T) {
	code := `
		filter := (pred, xs) -> match xs {
			[] -> []
			_ -> match pred(head(xs)) {
				0 -> filter(pred, tail(xs))
				_ -> [head(xs)] ++ filter(pred, tail(xs))
			}
		}
		is_positive := x -> x > 0
		result := filter(is_positive, [-1, 2, -3, 4])
		println(result[0])
		println(result[1])
	`
	result := compileAndRun(t, code)
	if result != "2\n4\n" {
		t.Fatalf("Expected '2\\n4\\n', got %q", result)
	}
}

func TestHigherOrderCompose(t *testing.T) {
	code := `
		compose := (f, g) => x => f(g(x))
		add_one := x => x + 1
		double := x => x * 2
		add_then_double := compose(double, add_one)
		result := add_then_double(5)
		println(result)
	`
	result := compileAndRun(t, code)
	if result != "12\n" {
		t.Fatalf("Expected '12\\n', got %q", result)
	}
}

func TestHigherOrderReduce(t *testing.T) {
	code := `
		reduce := (f, acc, xs) => match xs {
			[] => acc
			_ => reduce(f, f(acc, head(xs)), tail(xs))
		}
		add := (a, b) => a + b
		result := reduce(add, 0, [1, 2, 3, 4])
		println(result)
	`
	result := compileAndRun(t, code)
	if result != "10\n" {
		t.Fatalf("Expected '10\\n', got %q", result)
	}
}

func TestHigherOrderPartialApplication(t *testing.T) {
	code := `
		add := (a, b) => a + b
		add_five := x => add(5, x)
		result := add_five(3)
		println(result)
	`
	result := compileAndRun(t, code)
	if result != "8\n" {
		t.Fatalf("Expected '8\\n', got %q", result)
	}
}

func TestHigherOrderCallback(t *testing.T) {
	code := `
		repeat := (n, f) => match n {
			0 => 0
			_ => {
				f()
				repeat(n - 1, f)
			}
		}
		counter := 0
		increment := () => counter := counter + 1
		repeat(3, increment)
		println(counter)
	`
	result := compileAndRun(t, code)
	if result != "3\n" {
		t.Fatalf("Expected '3\\n', got %q", result)
	}
}
