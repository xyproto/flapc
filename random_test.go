package main

import (
	"strings"
	"testing"
)

func TestRandomOperator(t *testing.T) {
	tests := []struct {
		name    string
		program string
		check   func(result FlapResult) bool
	}{
		{
			name: "basic random returns value in range",
			program: `
x := ???
x >= 0.0 and x < 1.0 {
	-> println(1)
	~> println(0)
}
`,
			check: func(result FlapResult) bool {
				return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "1"
			},
		},
		{
			name: "multiple random calls produce different values",
			program: `
a := ???
b := ???
c := ???
d := ???
e := ???
a == b and b == c and c == d and d == e {
	-> println(0)
	~> println(1)
}
`,
			check: func(result FlapResult) bool {
				return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "1"
			},
		},
		{
			name: "random in arithmetic expression",
			program: `
roll := (??? * 6) | 0 + 1
roll >= 1 and roll <= 6 {
	-> println(1)
	~> println(0)
}
`,
			check: func(result FlapResult) bool {
				return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "1"
			},
		},
		{
			name: "random in loop generates different values",
			program: `
first := ???
all_same := 1
@ i in 0..<100 {
	??? != first {
		-> all_same <- 0
	}
}
println(all_same)
`,
			check: func(result FlapResult) bool {
				// With 100 samples, probability of all being equal is effectively 0
				return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "0"
			},
		},
		{
			name: "random with list selection",
			program: `
items := [10, 20, 30, 40, 50]
idx := (??? * #items) as int64
chosen := items[idx]
chosen == 10 or chosen == 20 or chosen == 30 or chosen == 40 or chosen == 50 {
	-> println(1)
	~> println(0)
}
`,
			check: func(result FlapResult) bool {
				return result.ExitCode == 0 && strings.TrimSpace(result.Stdout) == "1"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := runFlapProgram(t, tt.program)
			if result.CompileError != "" {
				t.Fatalf("Compilation failed: %s", result.CompileError)
			}
			if !tt.check(result) {
				t.Errorf("Test failed:\nstdout: %s\nstderr: %s\nexitCode: %d", result.Stdout, result.Stderr, result.ExitCode)
			}
		})
	}
}

func TestRandomStatisticalProperties(t *testing.T) {
	program := `
sum := 0.0
@ i in 0..<100 {
	sum <- sum + ???
}
avg := sum / 100.0
avg >= 0.3 and avg <= 0.7 {
	println(1)
	~> println(0)
}
`
	result := runFlapProgram(t, program)
	if result.CompileError != "" {
		t.Fatalf("Compilation failed: %s", result.CompileError)
	}
	if result.ExitCode != 0 || strings.TrimSpace(result.Stdout) != "1" {
		t.Errorf("Statistical test failed:\nstdout: %s\nstderr: %s\nexitCode: %d", result.Stdout, result.Stderr, result.ExitCode)
	}
}
