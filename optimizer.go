package main

import (
	"fmt"
	"math"
	"os"
)

// optimizer.go - Compiler optimization passes
//
// This file contains all optimization transformations applied to the AST
// before code generation. Optimizations include:
// - Constant folding and propagation
// - Strength reduction (expensive ops → cheaper ops)
// - Dead code elimination
// - Function inlining
// - Purity analysis
// - Closure analysis

func optimizeProgram(program *Program) *Program {
	// Pass 1: Constant folding (2 + 3 → 5)
	for i, stmt := range program.Statements {
		program.Statements[i] = foldConstants(stmt)
	}

	// Pass 2: Constant propagation (x = 5; y = x + 1 → y = 6)
	constMap := make(map[string]*NumberExpr)
	for i, stmt := range program.Statements {
		program.Statements[i] = propagateConstants(stmt, constMap)
	}

	// Pass 3: Dead code elimination (remove unused variables, unreachable code)
	// DISABLED: This was removing unused definitions before sibling files were loaded
	// DCE now runs in the WPO phase (optimizer.go) after all files are combined
	// usedVars := make(map[string]bool)
	// for _, stmt := range program.Statements {
	// 	collectUsedVariables(stmt, usedVars)
	// }
	// newStmts := make([]Statement, 0, len(program.Statements))
	// for _, stmt := range program.Statements {
	// 	if keep := eliminateDeadCode(stmt, usedVars); keep != nil {
	// 		newStmts = append(newStmts, keep)
	// 	}
	// }
	// program.Statements = newStmts

	// Pass 4: Analyze lambda purity (for future memoization)
	pureFunctions := make(map[string]bool) // Track which named functions are pure
	for _, stmt := range program.Statements {
		analyzePurity(stmt, pureFunctions)
	}

	// Pass 5: Function inlining (substitute small function calls with their bodies)
	inlineCandidates := make(map[string]*LambdaExpr) // Functions that can be inlined
	callCounts := make(map[string]int)               // Number of times each function is called

	// Identify inline candidates
	for _, stmt := range program.Statements {
		collectInlineCandidates(stmt, inlineCandidates)
	}

	// Count call sites for each candidate
	for _, stmt := range program.Statements {
		countCalls(stmt, callCounts)
	}

	// Inline function calls
	for i, stmt := range program.Statements {
		program.Statements[i] = inlineFunctions(stmt, inlineCandidates, callCounts)
	}

	// Pass 6: Constant folding after inlining (fold inlined expressions)
	for i, stmt := range program.Statements {
		program.Statements[i] = foldConstants(stmt)
	}

	return program
}

// foldConstants performs constant folding on statements
func foldConstants(stmt Statement) Statement {
	switch s := stmt.(type) {
	case *AssignStmt:
		s.Value = foldConstantExpr(s.Value)
		return s
	case *ExpressionStmt:
		s.Expr = foldConstantExpr(s.Expr)
		return s
	case *LoopStmt:
		s.Iterable = foldConstantExpr(s.Iterable)
		for i, st := range s.Body {
			s.Body[i] = foldConstants(st)
		}
		return s
	default:
		return stmt
	}
}

// foldConstantExpr performs constant folding on expressions
func foldConstantExpr(expr Expression) Expression {
	switch e := expr.(type) {
	case *BinaryExpr:
		// Fold left and right first
		e.Left = foldConstantExpr(e.Left)
		e.Right = foldConstantExpr(e.Right)

		// Check if both operands are now constants
		leftNum, leftOk := e.Left.(*NumberExpr)
		rightNum, rightOk := e.Right.(*NumberExpr)

		if leftOk && rightOk {
			// Both are constants - fold them
			var result float64
			switch e.Operator {
			case "+":
				result = leftNum.Value + rightNum.Value
			case "-":
				result = leftNum.Value - rightNum.Value
			case "*":
				result = leftNum.Value * rightNum.Value
			case "/":
				if rightNum.Value != 0 {
					result = leftNum.Value / rightNum.Value
				} else {
					return e // Don't fold division by zero
				}
			case "mod", "%":
				if rightNum.Value != 0 {
					result = math.Mod(leftNum.Value, rightNum.Value)
				} else {
					return e // Don't fold modulo by zero
				}
			default:
				return e // Don't fold comparisons
			}
			return &NumberExpr{Value: result}
		}
		return e

	case *CallExpr:
		// Fold arguments
		for i, arg := range e.Args {
			e.Args[i] = foldConstantExpr(arg)
		}
		return e

	case *RangeExpr:
		// Fold range start and end
		e.Start = foldConstantExpr(e.Start)
		e.End = foldConstantExpr(e.End)
		return e

	case *ListExpr:
		// Fold list elements
		for i, elem := range e.Elements {
			e.Elements[i] = foldConstantExpr(elem)
		}
		return e

	case *MapExpr:
		for i := range e.Keys {
			e.Keys[i] = foldConstantExpr(e.Keys[i])
			e.Values[i] = foldConstantExpr(e.Values[i])
		}
		return e
	case *IndexExpr:
		e.List = foldConstantExpr(e.List)
		e.Index = foldConstantExpr(e.Index)
		return e

	case *LambdaExpr:
		e.Body = foldConstantExpr(e.Body)
		return e

	case *ParallelExpr:
		e.List = foldConstantExpr(e.List)
		e.Operation = foldConstantExpr(e.Operation)
		return e

	case *PipeExpr:
		e.Left = foldConstantExpr(e.Left)
		e.Right = foldConstantExpr(e.Right)
		return e

	case *InExpr:
		e.Value = foldConstantExpr(e.Value)
		e.Container = foldConstantExpr(e.Container)
		return e

	case *LengthExpr:
		e.Operand = foldConstantExpr(e.Operand)
		return e

	case *MatchExpr:
		e.Condition = foldConstantExpr(e.Condition)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				clause.Guard = foldConstantExpr(clause.Guard)
			}
			clause.Result = foldConstantExpr(clause.Result)
		}
		if e.DefaultExpr != nil {
			e.DefaultExpr = foldConstantExpr(e.DefaultExpr)
		}
		return e

	default:
		return expr
	}
}

// isPowerOfTwo checks if a float64 value is a power of 2
func isPowerOfTwo(x float64) bool {
	if x <= 0 {
		return false
	}
	// Check if x is an integer
	if x != math.Floor(x) {
		return false
	}
	// Check if it's a power of 2: x & (x-1) == 0
	ix := int64(x)
	return (ix & (ix - 1)) == 0
}

// strengthReduceExpr performs strength reduction optimization on expressions
// Replaces expensive operations with cheaper equivalent ones:
// - x * 2^n → x << n (multiply by power of 2 → left shift)
// - x / 2^n → x >> n (divide by power of 2 → right shift)
// - x * 0 → 0, x * 1 → x (identity elimination)
// - x + 0, x - 0 → x (identity elimination)
// - x % 2^n → x & (2^n - 1) (modulo by power of 2 → bitwise AND)
func strengthReduceExpr(expr Expression) Expression {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		// Recursively apply strength reduction to operands first
		e.Left = strengthReduceExpr(e.Left)
		e.Right = strengthReduceExpr(e.Right)

		// Check for patterns we can optimize
		leftNum, leftIsNum := e.Left.(*NumberExpr)
		rightNum, rightIsNum := e.Right.(*NumberExpr)

		switch e.Operator {
		case "*":
			// x * 0 → 0
			if (leftIsNum && leftNum.Value == 0) || (rightIsNum && rightNum.Value == 0) {
				return &NumberExpr{Value: 0}
			}

			// x * 1 → x
			if rightIsNum && rightNum.Value == 1 {
				return e.Left
			}
			if leftIsNum && leftNum.Value == 1 {
				return e.Right
			}

			// x * -1 → -x
			if rightIsNum && rightNum.Value == -1 {
				return &UnaryExpr{Operator: "-", Operand: e.Left}
			}
			if leftIsNum && leftNum.Value == -1 {
				return &UnaryExpr{Operator: "-", Operand: e.Right}
			}

			// x * 2^n → x << n (only for positive integer powers of 2)
			// DISABLED: This optimization is only valid for integers, but Flap uses float64 by default.
			// Applying this to floating-point operations breaks codegen because << is an integer operation.
			// TODO: Re-enable this optimization only in integer contexts (unsafe blocks, explicit int types).
			/*
				if rightIsNum && rightNum.Value > 0 && isPowerOfTwo(rightNum.Value) {
					shift := math.Log2(rightNum.Value)
					return &BinaryExpr{
						Left:     e.Left,
						Operator: "<<",
						Right:    &NumberExpr{Value: shift},
					}
				}
				if leftIsNum && leftNum.Value > 0 && isPowerOfTwo(leftNum.Value) {
					shift := math.Log2(leftNum.Value)
					return &BinaryExpr{
						Left:     e.Right,
						Operator: "<<",
						Right:    &NumberExpr{Value: shift},
					}
				}
			*/

		case "/":
			// x / 1 → x
			if rightIsNum && rightNum.Value == 1 {
				return e.Left
			}

			// x / -1 → -x
			if rightIsNum && rightNum.Value == -1 {
				return &UnaryExpr{Operator: "-", Operand: e.Left}
			}

			// x / 2^n → x >> n (only for positive powers of 2)
			// DISABLED: This optimization is only valid for integers, but Flap uses float64 by default.
			// Applying this to floating-point operations breaks codegen because >> is an integer operation.
			// TODO: Re-enable this optimization only in integer contexts (unsafe blocks, explicit int types).
			/*
				if rightIsNum && rightNum.Value > 0 && isPowerOfTwo(rightNum.Value) {
					shift := math.Log2(rightNum.Value)
					return &BinaryExpr{
						Left:     e.Left,
						Operator: ">>",
						Right:    &NumberExpr{Value: shift},
					}
				}
			*/

		case "+":
			// x + 0 → x
			if rightIsNum && rightNum.Value == 0 {
				return e.Left
			}
			if leftIsNum && leftNum.Value == 0 {
				return e.Right
			}

		case "-":
			// x - 0 → x
			if rightIsNum && rightNum.Value == 0 {
				return e.Left
			}

			// 0 - x → -x
			if leftIsNum && leftNum.Value == 0 {
				return &UnaryExpr{Operator: "-", Operand: e.Right}
			}

		case "&":
			// x & 0 → 0
			if (leftIsNum && leftNum.Value == 0) || (rightIsNum && rightNum.Value == 0) {
				return &NumberExpr{Value: 0}
			}

		case "|":
			// x | 0 → x
			if rightIsNum && rightNum.Value == 0 {
				return e.Left
			}
			if leftIsNum && leftNum.Value == 0 {
				return e.Right
			}

		case "^":
			// x ^ 0 → x
			if rightIsNum && rightNum.Value == 0 {
				return e.Left
			}
			if leftIsNum && leftNum.Value == 0 {
				return e.Right
			}

		case "<<", ">>":
			// x << 0 → x, x >> 0 → x
			if rightIsNum && rightNum.Value == 0 {
				return e.Left
			}

			// 0 << x → 0, 0 >> x → 0
			if leftIsNum && leftNum.Value == 0 {
				return &NumberExpr{Value: 0}
			}

		case "mod", "%":
			// x % 1 → 0
			if rightIsNum && rightNum.Value == 1 {
				return &NumberExpr{Value: 0}
			}

			// 0 % x → 0
			if leftIsNum && leftNum.Value == 0 {
				return &NumberExpr{Value: 0}
			}

			// x % 2^n → x & (2^n - 1) for positive powers of 2
			// DISABLED: This optimization is only valid for integers, but Flap uses float64 by default.
			// Applying this to floating-point operations breaks codegen because & is an integer operation.
			// TODO: Re-enable this optimization only in integer contexts (unsafe blocks, explicit int types).
			/*
				if rightIsNum && rightNum.Value > 0 && isPowerOfTwo(rightNum.Value) {
					mask := rightNum.Value - 1
					return &BinaryExpr{
						Left:     e.Left,
						Operator: "&",
						Right:    &NumberExpr{Value: mask},
					}
				}
			*/
		}

		return e

	case *UnaryExpr:
		e.Operand = strengthReduceExpr(e.Operand)

		// Double negation: -(-x) → x
		if e.Operator == "-" {
			if inner, ok := e.Operand.(*UnaryExpr); ok && inner.Operator == "-" {
				return inner.Operand
			}
		}

		return e

	case *CallExpr:
		for i, arg := range e.Args {
			e.Args[i] = strengthReduceExpr(arg)
		}
		return e

	case *ListExpr:
		for i, elem := range e.Elements {
			e.Elements[i] = strengthReduceExpr(elem)
		}
		return e

	case *MapExpr:
		for i := range e.Keys {
			e.Keys[i] = strengthReduceExpr(e.Keys[i])
			e.Values[i] = strengthReduceExpr(e.Values[i])
		}
		return e

	case *IndexExpr:
		e.List = strengthReduceExpr(e.List)
		e.Index = strengthReduceExpr(e.Index)
		return e

	case *LambdaExpr:
		e.Body = strengthReduceExpr(e.Body)
		return e

	case *RangeExpr:
		e.Start = strengthReduceExpr(e.Start)
		e.End = strengthReduceExpr(e.End)
		return e

	case *MatchExpr:
		e.Condition = strengthReduceExpr(e.Condition)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				clause.Guard = strengthReduceExpr(clause.Guard)
			}
			clause.Result = strengthReduceExpr(clause.Result)
		}
		if e.DefaultExpr != nil {
			e.DefaultExpr = strengthReduceExpr(e.DefaultExpr)
		}
		return e

	case *BlockExpr:
		for i, stmt := range e.Statements {
			e.Statements[i] = strengthReduceStmt(stmt)
		}
		return e

	case *LoopExpr:
		e.Iterable = strengthReduceExpr(e.Iterable)
		for i, stmt := range e.Body {
			e.Body[i] = strengthReduceStmt(stmt)
		}
		return e

	case *PipeExpr:
		e.Left = strengthReduceExpr(e.Left)
		e.Right = strengthReduceExpr(e.Right)
		return e

	case *ParallelExpr:
		e.List = strengthReduceExpr(e.List)
		e.Operation = strengthReduceExpr(e.Operation)
		return e

	case *InExpr:
		e.Value = strengthReduceExpr(e.Value)
		e.Container = strengthReduceExpr(e.Container)
		return e

	case *LengthExpr:
		e.Operand = strengthReduceExpr(e.Operand)
		return e

	default:
		return expr
	}
}

// strengthReduceStmt applies strength reduction to statements
func strengthReduceStmt(stmt Statement) Statement {
	if stmt == nil {
		return nil
	}

	switch s := stmt.(type) {
	case *AssignStmt:
		s.Value = strengthReduceExpr(s.Value)
		return s

	case *ExpressionStmt:
		s.Expr = strengthReduceExpr(s.Expr)
		return s

	case *LoopStmt:
		s.Iterable = strengthReduceExpr(s.Iterable)
		for i, bodyStmt := range s.Body {
			s.Body[i] = strengthReduceStmt(bodyStmt)
		}
		return s

	case *JumpStmt:
		if s.Value != nil {
			s.Value = strengthReduceExpr(s.Value)
		}
		return s

	default:
		return stmt
	}
}

// propagateConstants performs constant propagation on statements
// Tracks immutable variables assigned constant values and substitutes them
func propagateConstants(stmt Statement, constMap map[string]*NumberExpr) Statement {
	switch s := stmt.(type) {
	case *AssignStmt:
		// First propagate constants in the value expression
		s.Value = propagateConstantsExpr(s.Value, constMap)

		// Then fold constants in case propagation enabled new folding opportunities
		s.Value = foldConstantExpr(s.Value)

		// Apply strength reduction after constant folding
		s.Value = strengthReduceExpr(s.Value)

		// If this is an immutable assignment to a number literal, track it
		if !s.Mutable && !s.IsUpdate {
			if numExpr, ok := s.Value.(*NumberExpr); ok {
				// Clone the number expression to avoid mutation issues
				constMap[s.Name] = &NumberExpr{Value: numExpr.Value}
			} else {
				// Variable is not assigned a constant, remove from map
				delete(constMap, s.Name)
			}
		} else {
			// Mutable or update - can't track as constant
			delete(constMap, s.Name)
		}
		return s

	case *ExpressionStmt:
		s.Expr = propagateConstantsExpr(s.Expr, constMap)
		s.Expr = foldConstantExpr(s.Expr)
		s.Expr = strengthReduceExpr(s.Expr)
		return s

	case *LoopStmt:
		s.Iterable = propagateConstantsExpr(s.Iterable, constMap)
		s.Iterable = foldConstantExpr(s.Iterable)
		s.Iterable = strengthReduceExpr(s.Iterable)

		// Loop body creates a new scope - clone const map
		bodyConstMap := make(map[string]*NumberExpr)
		for k, v := range constMap {
			bodyConstMap[k] = v
		}
		// Remove iterator variable from constants (it changes each iteration)
		delete(bodyConstMap, s.Iterator)

		for i, bodyStmt := range s.Body {
			s.Body[i] = propagateConstants(bodyStmt, bodyConstMap)
		}
		return s

	default:
		return stmt
	}
}

// propagateConstantsExpr substitutes variable references with known constant values
func propagateConstantsExpr(expr Expression, constMap map[string]*NumberExpr) Expression {
	switch e := expr.(type) {
	case *IdentExpr:
		// Check if this variable has a known constant value
		if constVal, exists := constMap[e.Name]; exists {
			// Substitute with the constant value
			return &NumberExpr{Value: constVal.Value}
		}
		return e

	case *BinaryExpr:
		e.Left = propagateConstantsExpr(e.Left, constMap)
		e.Right = propagateConstantsExpr(e.Right, constMap)
		return e

	case *CallExpr:
		for i, arg := range e.Args {
			e.Args[i] = propagateConstantsExpr(arg, constMap)
		}
		return e

	case *RangeExpr:
		e.Start = propagateConstantsExpr(e.Start, constMap)
		e.End = propagateConstantsExpr(e.End, constMap)
		return e

	case *ListExpr:
		for i, elem := range e.Elements {
			e.Elements[i] = propagateConstantsExpr(elem, constMap)
		}
		return e

	case *MapExpr:
		for i := range e.Keys {
			e.Keys[i] = propagateConstantsExpr(e.Keys[i], constMap)
			e.Values[i] = propagateConstantsExpr(e.Values[i], constMap)
		}
		return e

	case *IndexExpr:
		e.List = propagateConstantsExpr(e.List, constMap)
		e.Index = propagateConstantsExpr(e.Index, constMap)
		return e

	case *LambdaExpr:
		// Lambda creates new scope - don't propagate outer constants into lambda body
		// (More sophisticated analysis could handle this)
		return e

	case *ParallelExpr:
		e.List = propagateConstantsExpr(e.List, constMap)
		e.Operation = propagateConstantsExpr(e.Operation, constMap)
		return e

	case *PipeExpr:
		e.Left = propagateConstantsExpr(e.Left, constMap)
		e.Right = propagateConstantsExpr(e.Right, constMap)
		return e

	case *InExpr:
		e.Value = propagateConstantsExpr(e.Value, constMap)
		e.Container = propagateConstantsExpr(e.Container, constMap)
		return e

	case *LengthExpr:
		e.Operand = propagateConstantsExpr(e.Operand, constMap)
		return e

	case *MatchExpr:
		e.Condition = propagateConstantsExpr(e.Condition, constMap)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				clause.Guard = propagateConstantsExpr(clause.Guard, constMap)
			}
			clause.Result = propagateConstantsExpr(clause.Result, constMap)
		}
		if e.DefaultExpr != nil {
			e.DefaultExpr = propagateConstantsExpr(e.DefaultExpr, constMap)
		}
		return e

	case *BlockExpr:
		// Block creates new scope - clone const map
		blockConstMap := make(map[string]*NumberExpr)
		for k, v := range constMap {
			blockConstMap[k] = v
		}
		for i, stmt := range e.Statements {
			e.Statements[i] = propagateConstants(stmt, blockConstMap)
		}
		return e

	case *MoveExpr:
		// Don't propagate constants into move expressions
		// The variable must exist at runtime for move semantics to work
		return e

	default:
		return expr
	}
}

// collectUsedVariables walks the AST and tracks which variables are referenced
func collectUsedVariables(stmt Statement, usedVars map[string]bool) {
	switch s := stmt.(type) {
	case *AssignStmt:
		collectUsedVariablesExpr(s.Value, usedVars)
	case *ExpressionStmt:
		collectUsedVariablesExpr(s.Expr, usedVars)
	case *LoopStmt:
		collectUsedVariablesExpr(s.Iterable, usedVars)
		// Mark iterator as used (even if not explicitly referenced)
		usedVars[s.Iterator] = true
		for _, bodyStmt := range s.Body {
			collectUsedVariables(bodyStmt, usedVars)
		}
	}
}

// collectUsedVariablesExpr tracks variable references in expressions
func collectUsedVariablesExpr(expr Expression, usedVars map[string]bool) {
	switch e := expr.(type) {
	case *IdentExpr:
		usedVars[e.Name] = true
	case *BinaryExpr:
		collectUsedVariablesExpr(e.Left, usedVars)
		collectUsedVariablesExpr(e.Right, usedVars)
	case *CallExpr:
		// Mark the function being called as used
		usedVars[e.Function] = true
		for _, arg := range e.Args {
			collectUsedVariablesExpr(arg, usedVars)
		}
	case *RangeExpr:
		collectUsedVariablesExpr(e.Start, usedVars)
		collectUsedVariablesExpr(e.End, usedVars)
	case *ListExpr:
		for _, elem := range e.Elements {
			collectUsedVariablesExpr(elem, usedVars)
		}
	case *MapExpr:
		for i := range e.Keys {
			collectUsedVariablesExpr(e.Keys[i], usedVars)
			collectUsedVariablesExpr(e.Values[i], usedVars)
		}
	case *IndexExpr:
		collectUsedVariablesExpr(e.List, usedVars)
		collectUsedVariablesExpr(e.Index, usedVars)
	case *LambdaExpr:
		collectUsedVariablesExpr(e.Body, usedVars)
	case *ParallelExpr:
		collectUsedVariablesExpr(e.List, usedVars)
		collectUsedVariablesExpr(e.Operation, usedVars)
	case *PipeExpr:
		collectUsedVariablesExpr(e.Left, usedVars)
		collectUsedVariablesExpr(e.Right, usedVars)
	case *InExpr:
		collectUsedVariablesExpr(e.Value, usedVars)
		collectUsedVariablesExpr(e.Container, usedVars)
	case *LengthExpr:
		collectUsedVariablesExpr(e.Operand, usedVars)
	case *MatchExpr:
		collectUsedVariablesExpr(e.Condition, usedVars)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				collectUsedVariablesExpr(clause.Guard, usedVars)
			}
			collectUsedVariablesExpr(clause.Result, usedVars)
		}
		if e.DefaultExpr != nil {
			collectUsedVariablesExpr(e.DefaultExpr, usedVars)
		}
	case *BlockExpr:
		for _, stmt := range e.Statements {
			collectUsedVariables(stmt, usedVars)
		}
	case *CastExpr:
		collectUsedVariablesExpr(e.Expr, usedVars)
	case *SliceExpr:
		collectUsedVariablesExpr(e.List, usedVars)
		if e.Start != nil {
			collectUsedVariablesExpr(e.Start, usedVars)
		}
		if e.End != nil {
			collectUsedVariablesExpr(e.End, usedVars)
		}
	case *UnaryExpr:
		collectUsedVariablesExpr(e.Operand, usedVars)
	case *NamespacedIdentExpr:
		// Namespace access like sdl.SDL_Init or data.field
		// For data.field, "data" is a variable that should be marked as used
		// For sdl.SDL_Init, "sdl" is an imported namespace, not a variable
		// We mark it as used - the compiler will handle whether it's a variable or namespace
		usedVars[e.Namespace] = true
	case *FStringExpr:
		// FStringExpr.Parts is []Expression, each part is either StringExpr or an expression
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: FStringExpr with %d parts\n", len(e.Parts))
			for i, part := range e.Parts {
				fmt.Fprintf(os.Stderr, "  Part %d: %T\n", i, part)
			}
		}
		for _, part := range e.Parts {
			collectUsedVariablesExpr(part, usedVars)
		}
	case *DirectCallExpr:
		collectUsedVariablesExpr(e.Callee, usedVars)
		for _, arg := range e.Args {
			collectUsedVariablesExpr(arg, usedVars)
		}
	case *PostfixExpr:
		collectUsedVariablesExpr(e.Operand, usedVars)
	case *VectorExpr:
		for _, comp := range e.Components {
			collectUsedVariablesExpr(comp, usedVars)
		}
	case *ArenaExpr:
		// ArenaExpr has Body []Statement
		for _, stmt := range e.Body {
			collectUsedVariables(stmt, usedVars)
		}
	case *MultiLambdaExpr:
		// For multi-lambda, collect variables from all lambda bodies
		for _, lambda := range e.Lambdas {
			collectUsedVariablesExpr(lambda.Body, usedVars)
		}
	case *SendExpr:
		// SendExpr has Target and Message
		collectUsedVariablesExpr(e.Target, usedVars)
		collectUsedVariablesExpr(e.Message, usedVars)
	case *UnsafeExpr:
		// UnsafeExpr has architecture-specific blocks
		for _, stmt := range e.X86_64Block {
			collectUsedVariables(stmt, usedVars)
		}
		for _, stmt := range e.ARM64Block {
			collectUsedVariables(stmt, usedVars)
		}
		for _, stmt := range e.RISCV64Block {
			collectUsedVariables(stmt, usedVars)
		}
	case *LoopExpr:
		for _, stmt := range e.Body {
			collectUsedVariables(stmt, usedVars)
		}
	case *LoopStateExpr:
		// LoopStateExpr doesn't reference variables
	case *JumpExpr:
		// JumpExpr doesn't reference variables directly
	}
}

// eliminateDeadCode removes assignments to unused variables
// Returns nil if statement should be removed entirely
func eliminateDeadCode(stmt Statement, usedVars map[string]bool) Statement {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Keep assignments if:
		// 1. Variable is used somewhere
		// 2. Assignment has side effects (contains function call)
		if usedVars[s.Name] || hasSideEffects(s.Value) {
			return s
		}
		// Dead assignment - remove it
		return nil

	case *ExpressionStmt:
		// Always keep expression statements (they might have side effects like printf)
		return s

	case *LoopStmt:
		// Keep loop but eliminate dead code in body
		newBody := make([]Statement, 0, len(s.Body))
		for _, bodyStmt := range s.Body {
			if keep := eliminateDeadCode(bodyStmt, usedVars); keep != nil {
				newBody = append(newBody, keep)
			}
		}
		s.Body = newBody
		return s

	default:
		return stmt
	}
}

// hasSideEffects checks if an expression contains function calls or other side effects
func hasSideEffects(expr Expression) bool {
	switch e := expr.(type) {
	case *CallExpr:
		return true // Function calls have side effects
	case *BinaryExpr:
		return hasSideEffects(e.Left) || hasSideEffects(e.Right)
	case *ListExpr:
		for _, elem := range e.Elements {
			if hasSideEffects(elem) {
				return true
			}
		}
		return false
	case *MapExpr:
		for i := range e.Keys {
			if hasSideEffects(e.Keys[i]) || hasSideEffects(e.Values[i]) {
				return true
			}
		}
		return false
	case *IndexExpr:
		return hasSideEffects(e.List) || hasSideEffects(e.Index)
	case *ParallelExpr:
		return true // Parallel operations have side effects
	case *PipeExpr:
		return hasSideEffects(e.Left) || hasSideEffects(e.Right)
	case *MatchExpr:
		if hasSideEffects(e.Condition) {
			return true
		}
		for _, clause := range e.Clauses {
			if clause.Guard != nil && hasSideEffects(clause.Guard) {
				return true
			}
			if hasSideEffects(clause.Result) {
				return true
			}
		}
		if e.DefaultExpr != nil && hasSideEffects(e.DefaultExpr) {
			return true
		}
		return false
	case *BlockExpr:
		// Blocks can have side effects if any statement does
		return true
	default:
		return false // Literals, identifiers, etc. have no side effects
	}
}

// analyzePurity walks AST and marks lambdas as pure (no side effects, no captured mutables)
func analyzePurity(stmt Statement, pureFunctions map[string]bool) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Analyze value expression for lambdas
		if lambda, ok := s.Value.(*LambdaExpr); ok {
			// Check if this lambda is pure
			lambda.IsPure = isLambdaPure(lambda, pureFunctions)
			if !s.Mutable {
				// Track named pure functions for call analysis
				pureFunctions[s.Name] = lambda.IsPure
			}
		}
		analyzePurityExpr(s.Value, pureFunctions)
	case *ExpressionStmt:
		analyzePurityExpr(s.Expr, pureFunctions)
	case *LoopStmt:
		analyzePurityExpr(s.Iterable, pureFunctions)
		for _, bodyStmt := range s.Body {
			analyzePurity(bodyStmt, pureFunctions)
		}
	}
}

// analyzePurityExpr recursively analyzes expressions for lambdas
func analyzePurityExpr(expr Expression, pureFunctions map[string]bool) {
	switch e := expr.(type) {
	case *LambdaExpr:
		e.IsPure = isLambdaPure(e, pureFunctions)
	case *BinaryExpr:
		analyzePurityExpr(e.Left, pureFunctions)
		analyzePurityExpr(e.Right, pureFunctions)
	case *CallExpr:
		for _, arg := range e.Args {
			analyzePurityExpr(arg, pureFunctions)
		}
	case *ListExpr:
		for _, elem := range e.Elements {
			analyzePurityExpr(elem, pureFunctions)
		}
	case *MapExpr:
		for i := range e.Keys {
			analyzePurityExpr(e.Keys[i], pureFunctions)
			analyzePurityExpr(e.Values[i], pureFunctions)
		}
	case *IndexExpr:
		analyzePurityExpr(e.List, pureFunctions)
		analyzePurityExpr(e.Index, pureFunctions)
	case *ParallelExpr:
		analyzePurityExpr(e.List, pureFunctions)
		analyzePurityExpr(e.Operation, pureFunctions)
	case *PipeExpr:
		analyzePurityExpr(e.Left, pureFunctions)
		analyzePurityExpr(e.Right, pureFunctions)
	case *MatchExpr:
		analyzePurityExpr(e.Condition, pureFunctions)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				analyzePurityExpr(clause.Guard, pureFunctions)
			}
			analyzePurityExpr(clause.Result, pureFunctions)
		}
		if e.DefaultExpr != nil {
			analyzePurityExpr(e.DefaultExpr, pureFunctions)
		}
	case *BlockExpr:
		for _, stmt := range e.Statements {
			analyzePurity(stmt, pureFunctions)
		}
	}
}

// isLambdaPure determines if a lambda is pure (safe to memoize)
// A pure lambda:
// 1. Has no side effects (no I/O, no global state mutation)
// 2. Doesn't capture mutable variables
// 3. Only calls other pure functions
// 4. Is deterministic (same inputs → same outputs)
func isLambdaPure(lambda *LambdaExpr, pureFunctions map[string]bool) bool {
	// Check for basic side effects
	if hasSideEffects(lambda.Body) {
		return false
	}

	// Check if lambda calls any impure functions
	if callsImpureFunctions(lambda.Body, pureFunctions) {
		return false
	}

	// Check if lambda captures external variables (conservatively mark as impure)
	// More sophisticated analysis could track whether captured vars are mutable
	capturedVars := make(map[string]bool)
	collectCapturedVariables(lambda.Body, lambda.Params, capturedVars)
	if len(capturedVars) > 0 {
		// Lambda captures external variables - conservatively mark as impure
		// (Could be enhanced to allow capturing immutable constants)
		return false
	}

	return true
}

// callsImpureFunctions checks if expression calls any functions marked as impure
func callsImpureFunctions(expr Expression, pureFunctions map[string]bool) bool {
	switch e := expr.(type) {
	case *CallExpr:
		// Check if called function is known to be impure
		// Known impure built-ins
		impureBuiltins := map[string]bool{
			"printf": true, "println": true, "print": true,
			"scanf": true, "read": true, "write": true,
		}
		if impureBuiltins[e.Function] {
			return true
		}
		// Check if it's a user function we know is impure
		if isPure, known := pureFunctions[e.Function]; known && !isPure {
			return true
		}
		// Check arguments
		for _, arg := range e.Args {
			if callsImpureFunctions(arg, pureFunctions) {
				return true
			}
		}
		return false
	case *BinaryExpr:
		return callsImpureFunctions(e.Left, pureFunctions) || callsImpureFunctions(e.Right, pureFunctions)
	case *ListExpr:
		for _, elem := range e.Elements {
			if callsImpureFunctions(elem, pureFunctions) {
				return true
			}
		}
		return false
	case *MatchExpr:
		if callsImpureFunctions(e.Condition, pureFunctions) {
			return true
		}
		for _, clause := range e.Clauses {
			if clause.Guard != nil && callsImpureFunctions(clause.Guard, pureFunctions) {
				return true
			}
			if callsImpureFunctions(clause.Result, pureFunctions) {
				return true
			}
		}
		if e.DefaultExpr != nil && callsImpureFunctions(e.DefaultExpr, pureFunctions) {
			return true
		}
		return false
	case *BlockExpr:
		// Conservative: blocks might have impure statements
		return true
	default:
		return false
	}
}

// collectCapturedVariables finds variables used but not defined in lambda params
func collectCapturedVariables(expr Expression, params []string, captured map[string]bool) {
	// Create param set for quick lookup
	paramSet := make(map[string]bool)
	for _, p := range params {
		paramSet[p] = true
	}

	collectCapturedVarsExpr(expr, paramSet, captured)
}

func collectCapturedVarsExpr(expr Expression, paramSet map[string]bool, captured map[string]bool) {
	switch e := expr.(type) {
	case *IdentExpr:
		// If variable is not a parameter, it's captured from outer scope
		if !paramSet[e.Name] {
			captured[e.Name] = true
		}
	case *LambdaExpr:
		// Nested lambda: extend paramSet with nested lambda's parameters
		// and recursively collect from its body
		nestedParamSet := make(map[string]bool)
		for k, v := range paramSet {
			nestedParamSet[k] = v
		}
		for _, param := range e.Params {
			nestedParamSet[param] = true
		}
		collectCapturedVarsExpr(e.Body, nestedParamSet, captured)
	case *BinaryExpr:
		collectCapturedVarsExpr(e.Left, paramSet, captured)
		collectCapturedVarsExpr(e.Right, paramSet, captured)
	case *CallExpr:
		for _, arg := range e.Args {
			collectCapturedVarsExpr(arg, paramSet, captured)
		}
	case *ListExpr:
		for _, elem := range e.Elements {
			collectCapturedVarsExpr(elem, paramSet, captured)
		}
	case *MapExpr:
		for i := range e.Keys {
			collectCapturedVarsExpr(e.Keys[i], paramSet, captured)
			collectCapturedVarsExpr(e.Values[i], paramSet, captured)
		}
	case *IndexExpr:
		collectCapturedVarsExpr(e.List, paramSet, captured)
		collectCapturedVarsExpr(e.Index, paramSet, captured)
	case *MatchExpr:
		collectCapturedVarsExpr(e.Condition, paramSet, captured)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				collectCapturedVarsExpr(clause.Guard, paramSet, captured)
			}
			collectCapturedVarsExpr(clause.Result, paramSet, captured)
		}
		if e.DefaultExpr != nil {
			collectCapturedVarsExpr(e.DefaultExpr, paramSet, captured)
		}
	case *JumpExpr:
		// Process the value expression of return/jump statements
		if e.Value != nil {
			collectCapturedVarsExpr(e.Value, paramSet, captured)
		}
	case *BlockExpr:
		// For blocks, we need to track locally defined variables
		// so they aren't treated as captured
		localParamSet := make(map[string]bool)
		for k, v := range paramSet {
			localParamSet[k] = v
		}

		// Process each statement in the block
		for _, stmt := range e.Statements {
			switch s := stmt.(type) {
			case *AssignStmt:
				// Recursively check the assignment value (with current param set)
				collectCapturedVarsExpr(s.Value, localParamSet, captured)
				// Then add locally defined variable to param set
				localParamSet[s.Name] = true
			case *ExpressionStmt:
				collectCapturedVarsExpr(s.Expr, localParamSet, captured)
			}
		}
	}
}

// analyzeClosure detects and marks closures (lambdas that capture variables from outer scope)
// This must be called during compilation to populate CapturedVars field
func analyzeClosures(stmt Statement, availableVars map[string]bool) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Add this variable to available vars
		newAvailableVars := make(map[string]bool)
		for k, v := range availableVars {
			newAvailableVars[k] = v
		}
		newAvailableVars[s.Name] = true

		// Analyze the value expression
		analyzeClosuresExpr(s.Value, availableVars)

	case *ExpressionStmt:
		analyzeClosuresExpr(s.Expr, availableVars)

	case *LoopStmt:
		// Add iterator to available vars for loop body
		newAvailableVars := make(map[string]bool)
		for k, v := range availableVars {
			newAvailableVars[k] = v
		}
		newAvailableVars[s.Iterator] = true

		analyzeClosuresExpr(s.Iterable, availableVars)
		for _, bodyStmt := range s.Body {
			analyzeClosures(bodyStmt, newAvailableVars)
		}

	case *JumpStmt:
		// Analyze the value expression of return/jump statements
		if s.Value != nil {
			analyzeClosuresExpr(s.Value, availableVars)
		}
	}
}

func analyzeClosuresExpr(expr Expression, availableVars map[string]bool) {
	switch e := expr.(type) {
	case *LambdaExpr:
		// This is a lambda - check if it captures any variables
		captured := make(map[string]bool)
		collectCapturedVariables(e.Body, e.Params, captured)

		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Lambda analysis - raw captured: %v, availableVars: %v\n", captured, availableVars)
		}

		// Filter captured vars to only include those available in outer scope
		var capturedList []string
		for varName := range captured {
			if availableVars[varName] {
				capturedList = append(capturedList, varName)
			}
		}

		e.CapturedVars = capturedList
		e.IsNestedLambda = len(capturedList) > 0

		if VerboseMode && len(capturedList) > 0 {
			fmt.Fprintf(os.Stderr, "DEBUG: Found closure with %d captured vars: %v\n", len(capturedList), capturedList)
		}

		// Recursively analyze the lambda body with params added to available vars
		newAvailableVars := make(map[string]bool)
		for k, v := range availableVars {
			newAvailableVars[k] = v
		}
		for _, param := range e.Params {
			newAvailableVars[param] = true
		}
		analyzeClosuresExpr(e.Body, newAvailableVars)

	case *BinaryExpr:
		analyzeClosuresExpr(e.Left, availableVars)
		analyzeClosuresExpr(e.Right, availableVars)
	case *CallExpr:
		for _, arg := range e.Args {
			analyzeClosuresExpr(arg, availableVars)
		}
	case *ListExpr:
		for _, elem := range e.Elements {
			analyzeClosuresExpr(elem, availableVars)
		}
	case *MapExpr:
		for i := range e.Keys {
			analyzeClosuresExpr(e.Keys[i], availableVars)
			analyzeClosuresExpr(e.Values[i], availableVars)
		}
	case *IndexExpr:
		analyzeClosuresExpr(e.List, availableVars)
		analyzeClosuresExpr(e.Index, availableVars)
	case *MatchExpr:
		analyzeClosuresExpr(e.Condition, availableVars)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				analyzeClosuresExpr(clause.Guard, availableVars)
			}
			analyzeClosuresExpr(clause.Result, availableVars)
		}
		if e.DefaultExpr != nil {
			analyzeClosuresExpr(e.DefaultExpr, availableVars)
		}
	case *JumpExpr:
		// Analyze the value expression of return/jump statements
		if e.Value != nil {
			analyzeClosuresExpr(e.Value, availableVars)
		}
	case *BlockExpr:
		// Create a new scope for the block, accumulating available vars
		blockAvailableVars := make(map[string]bool)
		for k, v := range availableVars {
			blockAvailableVars[k] = v
		}

		// Process each statement, threading through newly defined variables
		for _, stmt := range e.Statements {
			analyzeClosures(stmt, blockAvailableVars)
			// If it's an assignment, add the variable to available vars for subsequent statements
			if assign, ok := stmt.(*AssignStmt); ok {
				blockAvailableVars[assign.Name] = true
			}
		}
	case *UnaryExpr:
		analyzeClosuresExpr(e.Operand, availableVars)
	case *ParallelExpr:
		analyzeClosuresExpr(e.List, availableVars)
		analyzeClosuresExpr(e.Operation, availableVars)
	case *PipeExpr:
		analyzeClosuresExpr(e.Left, availableVars)
		analyzeClosuresExpr(e.Right, availableVars)
	}
}

// collectInlineCandidates identifies lambdas suitable for inlining
// Criteria: immutable, small body (single expression), not in a loop
func collectInlineCandidates(stmt Statement, candidates map[string]*LambdaExpr) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Only inline immutable assignments to lambdas
		if !s.Mutable && !s.IsUpdate {
			if lambda, ok := s.Value.(*LambdaExpr); ok {
				// Only inline simple lambdas (single expression body, no blocks)
				if !isComplexExpression(lambda.Body) {
					// Store a copy to avoid mutation
					candidates[s.Name] = &LambdaExpr{
						Params: lambda.Params,
						Body:   lambda.Body,
						IsPure: lambda.IsPure,
					}
				}
			}
		}
	case *LoopStmt:
		// Recursively check loop bodies (but don't inline loop vars)
		for _, bodyStmt := range s.Body {
			collectInlineCandidates(bodyStmt, candidates)
		}
	}
}

// isComplexExpression checks if an expression is too complex to inline
func isComplexExpression(expr Expression) bool {
	switch e := expr.(type) {
	case *BlockExpr:
		return true // Don't inline blocks
	case *MatchExpr:
		return true // Don't inline match expressions (can be large)
	case *ParallelExpr:
		return true // Don't inline parallel operations
	case *CallExpr:
		// Allow simple function calls, but not nested complex calls
		for _, arg := range e.Args {
			if isComplexExpression(arg) {
				return true
			}
		}
		return false
	case *BinaryExpr:
		// Allow binary operations
		return isComplexExpression(e.Left) || isComplexExpression(e.Right)
	case *ListExpr:
		// Allow small lists
		if len(e.Elements) > 5 {
			return true
		}
		for _, elem := range e.Elements {
			if isComplexExpression(elem) {
				return true
			}
		}
		return false
	default:
		return false // Simple expressions (numbers, idents, etc.) are OK
	}
}

// countCalls counts how many times each function is called in the program
func countCalls(stmt Statement, counts map[string]int) {
	switch s := stmt.(type) {
	case *AssignStmt:
		countCallsExpr(s.Value, counts)
	case *ExpressionStmt:
		countCallsExpr(s.Expr, counts)
	case *LoopStmt:
		countCallsExpr(s.Iterable, counts)
		for _, bodyStmt := range s.Body {
			countCalls(bodyStmt, counts)
		}
	}
}

func countCallsExpr(expr Expression, counts map[string]int) {
	switch e := expr.(type) {
	case *CallExpr:
		counts[e.Function]++
		for _, arg := range e.Args {
			countCallsExpr(arg, counts)
		}
	case *BinaryExpr:
		countCallsExpr(e.Left, counts)
		countCallsExpr(e.Right, counts)
	case *ListExpr:
		for _, elem := range e.Elements {
			countCallsExpr(elem, counts)
		}
	case *MapExpr:
		for i := range e.Keys {
			countCallsExpr(e.Keys[i], counts)
			countCallsExpr(e.Values[i], counts)
		}
	case *IndexExpr:
		countCallsExpr(e.List, counts)
		countCallsExpr(e.Index, counts)
	case *ParallelExpr:
		countCallsExpr(e.List, counts)
		countCallsExpr(e.Operation, counts)
	case *PipeExpr:
		countCallsExpr(e.Left, counts)
		countCallsExpr(e.Right, counts)
	case *MatchExpr:
		countCallsExpr(e.Condition, counts)
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				countCallsExpr(clause.Guard, counts)
			}
			countCallsExpr(clause.Result, counts)
		}
		if e.DefaultExpr != nil {
			countCallsExpr(e.DefaultExpr, counts)
		}
	case *BlockExpr:
		for _, stmt := range e.Statements {
			countCalls(stmt, counts)
		}
	case *LambdaExpr:
		countCallsExpr(e.Body, counts)
	}
}

// inlineFunctions substitutes function calls with their bodies
func inlineFunctions(stmt Statement, candidates map[string]*LambdaExpr, callCounts map[string]int) Statement {
	switch s := stmt.(type) {
	case *AssignStmt:
		s.Value = inlineFunctionsExpr(s.Value, candidates, callCounts)
		return s
	case *ExpressionStmt:
		s.Expr = inlineFunctionsExpr(s.Expr, candidates, callCounts)
		return s
	case *LoopStmt:
		s.Iterable = inlineFunctionsExpr(s.Iterable, candidates, callCounts)
		for i, bodyStmt := range s.Body {
			s.Body[i] = inlineFunctions(bodyStmt, candidates, callCounts)
		}
		return s
	default:
		return stmt
	}
}

func inlineFunctionsExpr(expr Expression, candidates map[string]*LambdaExpr, callCounts map[string]int) Expression {
	switch e := expr.(type) {
	case *CallExpr:
		// First, recursively inline in arguments (process innermost calls first)
		for i, arg := range e.Args {
			e.Args[i] = inlineFunctionsExpr(arg, candidates, callCounts)
		}

		// Then check if this function itself is an inline candidate
		if lambda, isCandidate := candidates[e.Function]; isCandidate {
			// Only inline if:
			// 1. Parameter count matches
			// 2. Called at least once
			if len(e.Args) == len(lambda.Params) && callCounts[e.Function] > 0 {
				// Inline by substituting parameters with arguments (which may now be inlined)
				inlinedBody := substituteParams(lambda.Body, lambda.Params, e.Args)
				return inlinedBody
			}
		}
		return e
	case *BinaryExpr:
		e.Left = inlineFunctionsExpr(e.Left, candidates, callCounts)
		e.Right = inlineFunctionsExpr(e.Right, candidates, callCounts)
		return e
	case *ListExpr:
		for i, elem := range e.Elements {
			e.Elements[i] = inlineFunctionsExpr(elem, candidates, callCounts)
		}
		return e
	case *MapExpr:
		for i := range e.Keys {
			e.Keys[i] = inlineFunctionsExpr(e.Keys[i], candidates, callCounts)
			e.Values[i] = inlineFunctionsExpr(e.Values[i], candidates, callCounts)
		}
		return e
	case *IndexExpr:
		e.List = inlineFunctionsExpr(e.List, candidates, callCounts)
		e.Index = inlineFunctionsExpr(e.Index, candidates, callCounts)
		return e
	case *ParallelExpr:
		e.List = inlineFunctionsExpr(e.List, candidates, callCounts)
		e.Operation = inlineFunctionsExpr(e.Operation, candidates, callCounts)
		return e
	case *PipeExpr:
		e.Left = inlineFunctionsExpr(e.Left, candidates, callCounts)
		e.Right = inlineFunctionsExpr(e.Right, candidates, callCounts)
		return e
	case *MatchExpr:
		e.Condition = inlineFunctionsExpr(e.Condition, candidates, callCounts)
		for i := range e.Clauses {
			if e.Clauses[i].Guard != nil {
				e.Clauses[i].Guard = inlineFunctionsExpr(e.Clauses[i].Guard, candidates, callCounts)
			}
			e.Clauses[i].Result = inlineFunctionsExpr(e.Clauses[i].Result, candidates, callCounts)
		}
		if e.DefaultExpr != nil {
			e.DefaultExpr = inlineFunctionsExpr(e.DefaultExpr, candidates, callCounts)
		}
		return e
	case *BlockExpr:
		for i, stmt := range e.Statements {
			e.Statements[i] = inlineFunctions(stmt, candidates, callCounts)
		}
		return e
	case *LambdaExpr:
		e.Body = inlineFunctionsExpr(e.Body, candidates, callCounts)
		return e
	default:
		return expr
	}
}

// deepCopyExpr creates a deep copy of an expression to avoid AST node sharing
func deepCopyExpr(expr Expression) Expression {
	switch e := expr.(type) {
	case *NumberExpr:
		return &NumberExpr{Value: e.Value}
	case *StringExpr:
		return &StringExpr{Value: e.Value}
	case *IdentExpr:
		return &IdentExpr{Name: e.Name}
	case *BinaryExpr:
		return &BinaryExpr{
			Left:     deepCopyExpr(e.Left),
			Operator: e.Operator,
			Right:    deepCopyExpr(e.Right),
		}
	case *CallExpr:
		newArgs := make([]Expression, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i] = deepCopyExpr(arg)
		}
		return &CallExpr{
			Function: e.Function,
			Args:     newArgs,
		}
	case *ListExpr:
		newElements := make([]Expression, len(e.Elements))
		for i, elem := range e.Elements {
			newElements[i] = deepCopyExpr(elem)
		}
		return &ListExpr{Elements: newElements}
	case *MapExpr:
		newKeys := make([]Expression, len(e.Keys))
		newValues := make([]Expression, len(e.Values))
		for i := range e.Keys {
			newKeys[i] = deepCopyExpr(e.Keys[i])
			newValues[i] = deepCopyExpr(e.Values[i])
		}
		return &MapExpr{Keys: newKeys, Values: newValues}
	case *IndexExpr:
		return &IndexExpr{
			List:  deepCopyExpr(e.List),
			Index: deepCopyExpr(e.Index),
		}
	case *LambdaExpr:
		paramsCopy := make([]string, len(e.Params))
		copy(paramsCopy, e.Params)
		return &LambdaExpr{
			Params: paramsCopy,
			Body:   deepCopyExpr(e.Body),
			IsPure: e.IsPure,
		}
	default:
		// For other types, return as-is (may need to extend this)
		return expr
	}
}

// substituteParams replaces parameter references with actual arguments
func substituteParams(body Expression, params []string, args []Expression) Expression {
	// Create substitution map
	substMap := make(map[string]Expression)
	for i, param := range params {
		substMap[param] = args[i]
	}

	return substituteParamsExpr(body, substMap)
}

func substituteParamsExpr(expr Expression, substMap map[string]Expression) Expression {
	switch e := expr.(type) {
	case *IdentExpr:
		// Replace parameter with argument (must deep copy to avoid sharing!)
		if replacement, found := substMap[e.Name]; found {
			return deepCopyExpr(replacement)
		}
		return e
	case *BinaryExpr:
		return &BinaryExpr{
			Left:     substituteParamsExpr(e.Left, substMap),
			Operator: e.Operator,
			Right:    substituteParamsExpr(e.Right, substMap),
		}
	case *CallExpr:
		newArgs := make([]Expression, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i] = substituteParamsExpr(arg, substMap)
		}
		return &CallExpr{
			Function: e.Function,
			Args:     newArgs,
		}
	case *ListExpr:
		newElements := make([]Expression, len(e.Elements))
		for i, elem := range e.Elements {
			newElements[i] = substituteParamsExpr(elem, substMap)
		}
		return &ListExpr{Elements: newElements}
	case *MapExpr:
		newKeys := make([]Expression, len(e.Keys))
		newValues := make([]Expression, len(e.Values))
		for i := range e.Keys {
			newKeys[i] = substituteParamsExpr(e.Keys[i], substMap)
			newValues[i] = substituteParamsExpr(e.Values[i], substMap)
		}
		return &MapExpr{Keys: newKeys, Values: newValues}
	case *IndexExpr:
		return &IndexExpr{
			List:  substituteParamsExpr(e.List, substMap),
			Index: substituteParamsExpr(e.Index, substMap),
		}
	case *MatchExpr:
		newClauses := make([]*MatchClause, len(e.Clauses))
		for i, clause := range e.Clauses {
			newClause := &MatchClause{
				Guard:  nil,
				Result: substituteParamsExpr(clause.Result, substMap),
			}
			if clause.Guard != nil {
				newClause.Guard = substituteParamsExpr(clause.Guard, substMap)
			}
			newClauses[i] = newClause
		}
		var newDefault Expression
		if e.DefaultExpr != nil {
			newDefault = substituteParamsExpr(e.DefaultExpr, substMap)
		}
		return &MatchExpr{
			Condition:   substituteParamsExpr(e.Condition, substMap),
			Clauses:     newClauses,
			DefaultExpr: newDefault,
		}
	case *LambdaExpr:
		// Don't substitute inside nested lambdas' parameters
		// But do substitute in the body (closure)
		return &LambdaExpr{
			Params: e.Params,
			Body:   substituteParamsExpr(e.Body, substMap),
			IsPure: e.IsPure,
		}
	case *BlockExpr:
		// Substitute in block statements
		newStatements := make([]Statement, len(e.Statements))
		for i, stmt := range e.Statements {
			newStatements[i] = substituteParamsStmt(stmt, substMap)
		}
		return &BlockExpr{Statements: newStatements}
	default:
		// Literals (NumberExpr, StringExpr, etc.) are returned as-is
		return expr
	}
}

func substituteParamsStmt(stmt Statement, substMap map[string]Expression) Statement {
	switch s := stmt.(type) {
	case *AssignStmt:
		return &AssignStmt{
			Name:     s.Name,
			Value:    substituteParamsExpr(s.Value, substMap),
			Mutable:  s.Mutable,
			IsUpdate: s.IsUpdate,
		}
	case *ExpressionStmt:
		return &ExpressionStmt{
			Expr: substituteParamsExpr(s.Expr, substMap),
		}
	case *LoopStmt:
		newBody := make([]Statement, len(s.Body))
		for i, bodyStmt := range s.Body {
			newBody[i] = substituteParamsStmt(bodyStmt, substMap)
		}
		return &LoopStmt{
			Iterator:      s.Iterator,
			Iterable:      substituteParamsExpr(s.Iterable, substMap),
			Body:          newBody,
			MaxIterations: s.MaxIterations,
			NeedsMaxCheck: s.NeedsMaxCheck,
			NumThreads:    s.NumThreads,
		}
	default:
		return stmt
	}
}
