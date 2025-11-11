package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestListPrograms tests list operations
func TestListPrograms(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		expected string
	}{
		{
			name: "list_literal",
			source: `items := [1, 2, 3, 4, 5]
println(#items)
`,
			expected: "5\n",
		},
		{
			name: "list_indexing",
			source: `items := [10, 20, 30]
println(items[0])
println(items[1])
println(items[2])
`,
			expected: "10\n20\n30\n",
		},
		{
			name: "list_update",
			source: `items := [1, 2, 3]
items[1] <- 99
println(items[0])
println(items[1])
println(items[2])
`,
			expected: "1\n99\n3\n",
		},
		{
			name: "empty_list",
			source: `empty := []
println(#empty)
`,
			expected: "0\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testInlineFlap(t, tt.name, tt.source, tt.expected)
		})
	}
}

// TestExistingListPrograms runs existing list test programs
func TestExistingListPrograms(t *testing.T) {
	tests := []string{
		"list_test",
		"list_test2",
		"list_simple",
		"list_index_test",
		"list_iter_test",
		"manual_list_test",
		"len_test",
		"len_simple",
		"len_empty",
	}

	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			srcPath := filepath.Join("testprograms", name+".flap")
			resultPath := filepath.Join("testprograms", name+".result")

			if _, err := os.Stat(srcPath); os.IsNotExist(err) {
				t.Skipf("Source file %s not found", srcPath)
				return
			}

			var expected string
			if data, err := os.ReadFile(resultPath); err == nil {
				expected = string(data)
			}

			tmpDir := t.TempDir()
			exePath := filepath.Join(tmpDir, name)

			platform := GetDefaultPlatform()
			if err := CompileFlap(srcPath, exePath, platform); err != nil {
				t.Fatalf("Compilation failed: %v", err)
			}

			cmd := exec.Command("timeout", "5s", exePath)
			output, err := cmd.CombinedOutput()
			if err != nil {
				if _, ok := err.(*exec.ExitError); !ok {
					t.Fatalf("Execution failed: %v", err)
				}
			}

			if expected != "" {
				actual := string(output)
				if actual != expected {
					t.Errorf("Output mismatch:\nExpected:\n%s\nActual:\n%s",
						expected, actual)
				}
			}
		})
	}
}

// TestHeadOperator tests the ^ (head) operator
func TestHeadOperator(t *testing.T) {
	source := `list := [1, 2, 3, 4]
head := ^list
println(head)
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "1\n")
}

// TestTailOperator tests the & (tail) operator
func TestTailOperator(t *testing.T) {
	source := `list := [1, 2, 3, 4]
tail := &list
println(tail[0])
println(tail[1])
println(tail[2])
`
	result := runFlapProgram(t, source)
	result.expectOutput(t, "2\n3\n4\n")
}
