package main

import (
	"fmt"
)

func TestAST() {
	code := `x := 5.0
y := !x
println(y)`

	parser := NewParser(code)
	program := parser.ParseProgram()

	fmt.Printf("Program has %d statements\n", len(program.Statements))
	for i, stmt := range program.Statements {
		fmt.Printf("Statement %d: %T\n", i, stmt)
		if assign, ok := stmt.(*AssignStmt); ok {
			fmt.Printf("  Name: %s\n", assign.Name)
			fmt.Printf("  Value type: %T\n", assign.Value)
			if assign.Value == nil {
				fmt.Printf("  Value: nil\n")
			} else {
				fmt.Printf("  Value: %+v\n", assign.Value)
			}
		}
	}
}
