package main

import (
	"fmt"
	"os"
	"time"
)

// OptimizationPass represents a single optimization transformation
type OptimizationPass interface {
	Name() string
	Run(program *Program) (changed bool, err error)
}

// Optimizer manages all optimization passes
type Optimizer struct {
	passes  []OptimizationPass
	maxIter int
	timeout time.Duration
}

// NewOptimizer creates an optimizer with default passes
func NewOptimizer(timeoutSeconds float64) *Optimizer {
	return &Optimizer{
		passes: []OptimizationPass{
			&ConstantPropagation{},
			&DeadCodeElimination{},
			&FunctionInlining{},
			&LoopUnrolling{},
		},
		maxIter: 10,
		timeout: time.Duration(timeoutSeconds * float64(time.Second)),
	}
}

// Optimize runs all optimization passes until fixed point or timeout
func (opt *Optimizer) Optimize(program *Program) error {
	if opt.timeout <= 0 {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "-> Skipping WPO (disabled via --opt-timeout=0)\n")
		}
		return nil
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Starting whole program optimization (timeout: %.1fs)\n", opt.timeout.Seconds())
	}

	startTime := time.Now()

	for i := 0; i < opt.maxIter; i++ {
		// Check timeout
		if time.Since(startTime) > opt.timeout {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "-> Optimization stopped: timeout reached (%.1fs)\n", opt.timeout.Seconds())
			}
			break
		}

		anyChanged := false
		for _, pass := range opt.passes {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "   Running %s (iteration %d)\n", pass.Name(), i+1)
			}
			changed, err := pass.Run(program)
			if err != nil {
				return fmt.Errorf("%s failed: %v", pass.Name(), err)
			}
			if changed {
				anyChanged = true
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "   %s made changes\n", pass.Name())
				}
			}
		}
		if !anyChanged {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "-> Optimization converged after %d iterations (%.3fs)\n", i+1, time.Since(startTime).Seconds())
			}
			break
		}
	}

	return nil
}

// ConstantPropagation replaces variables with known constant values
type ConstantPropagation struct {
	constants map[string]Expression
	mutated   map[string]bool
}

func (cp *ConstantPropagation) Name() string {
	return "Constant Propagation"
}

func (cp *ConstantPropagation) Run(program *Program) (bool, error) {
	cp.constants = make(map[string]Expression)
	cp.mutated = make(map[string]bool)

	// First pass: identify all mutated variables
	for _, stmt := range program.Statements {
		cp.findMutationsWithDepth(stmt, 0)
	}

	// Second pass: propagate constants (skipping mutated variables)
	changed := false
	for _, stmt := range program.Statements {
		if cp.propagateInStmt(stmt) {
			changed = true
		}
	}

	return changed, nil
}

const maxRecursionDepth = 100

func (cp *ConstantPropagation) findMutationsWithDepth(stmt Statement, depth int) {
	if depth > maxRecursionDepth {
		return
	}
	switch s := stmt.(type) {
	case *AssignStmt:
		if s.IsUpdate {
			cp.mutated[s.Name] = true
		}
		cp.findMutationsInExprWithDepth(s.Value, depth+1)
	case *LoopStmt:
		for _, bodyStmt := range s.Body {
			cp.findMutationsWithDepth(bodyStmt, depth+1)
		}
	case *ExpressionStmt:
		cp.findMutationsInExprWithDepth(s.Expr, depth+1)
	}
}

func (cp *ConstantPropagation) findMutationsInExprWithDepth(expr Expression, depth int) {
	if depth > maxRecursionDepth {
		return
	}
	switch e := expr.(type) {
	case *MatchExpr:
		for _, clause := range e.Clauses {
			cp.findMutationsInExprWithDepth(clause.Result, depth+1)
		}
		if e.DefaultExpr != nil {
			cp.findMutationsInExprWithDepth(e.DefaultExpr, depth+1)
		}
	case *BlockExpr:
		for _, stmt := range e.Statements {
			cp.findMutationsWithDepth(stmt, depth+1)
		}
	case *BinaryExpr:
		if e.Operator == "<-" {
			if ident, ok := e.Left.(*IdentExpr); ok {
				cp.mutated[ident.Name] = true
			}
		}
		cp.findMutationsInExprWithDepth(e.Left, depth+1)
		cp.findMutationsInExprWithDepth(e.Right, depth+1)
	case *LambdaExpr:
		cp.findMutationsInExprWithDepth(e.Body, depth+1)
	case *CallExpr:
		for _, arg := range e.Args {
			cp.findMutationsInExprWithDepth(arg, depth+1)
		}
	case *PostfixExpr:
		if ident, ok := e.Operand.(*IdentExpr); ok {
			cp.mutated[ident.Name] = true
		}
	}
}

func (cp *ConstantPropagation) propagateInStmt(stmt Statement) bool {
	changed := false

	switch s := stmt.(type) {
	case *AssignStmt:
		// Track constant assignments (but not for mutated variables)
		if !cp.mutated[s.Name] && !s.IsUpdate && cp.isConstant(s.Value) {
			cp.constants[s.Name] = s.Value
		}
		// Propagate in value expression
		if newExpr, ok := cp.propagateInExpr(s.Value); ok {
			s.Value = newExpr
			changed = true
		}

	case *ExpressionStmt:
		if newExpr, ok := cp.propagateInExpr(s.Expr); ok {
			s.Expr = newExpr
			changed = true
		}

	case *LoopStmt:
		if newExpr, ok := cp.propagateInExpr(s.Iterable); ok {
			s.Iterable = newExpr
			changed = true
		}
		for _, bodyStmt := range s.Body {
			if cp.propagateInStmt(bodyStmt) {
				changed = true
			}
		}

	case *JumpStmt:
		if s.Value != nil {
			if newExpr, ok := cp.propagateInExpr(s.Value); ok {
				s.Value = newExpr
				changed = true
			}
		}
	}

	return changed
}

func (cp *ConstantPropagation) propagateInExpr(expr Expression) (Expression, bool) {
	switch e := expr.(type) {
	case *IdentExpr:
		// Don't propagate if the variable is mutated
		if !cp.mutated[e.Name] {
			if constant, ok := cp.constants[e.Name]; ok {
				return constant, true
			}
		}

	case *MoveExpr:
		// Don't propagate constants into move expressions
		// The variable must exist at runtime for move semantics to work
		// Return the move expression unchanged
		return e, false

	case *BinaryExpr:
		changed := false
		newLeft, leftChanged := cp.propagateInExpr(e.Left)
		if leftChanged {
			e.Left = newLeft
			changed = true
		}
		newRight, rightChanged := cp.propagateInExpr(e.Right)
		if rightChanged {
			e.Right = newRight
			changed = true
		}

		// Constant folding: evaluate if both operands are constants
		if leftNum, leftOk := e.Left.(*NumberExpr); leftOk {
			if rightNum, rightOk := e.Right.(*NumberExpr); rightOk {
				if folded := cp.foldConstants(leftNum, rightNum, e.Operator); folded != nil {
					return folded, true
				}
			}
		}

		return e, changed

	case *CallExpr:
		changed := false
		for i, arg := range e.Args {
			if newArg, argChanged := cp.propagateInExpr(arg); argChanged {
				e.Args[i] = newArg
				changed = true
			}
		}
		return e, changed

	case *LambdaExpr:
		changed := false

		savedConstants := make(map[string]Expression)
		for _, param := range e.Params {
			if oldVal, existed := cp.constants[param]; existed {
				savedConstants[param] = oldVal
				delete(cp.constants, param)
			}
		}

		if newBody, bodyChanged := cp.propagateInExpr(e.Body); bodyChanged {
			e.Body = newBody
			changed = true
		}

		for param, oldVal := range savedConstants {
			cp.constants[param] = oldVal
		}

		return e, changed

	case *MatchExpr:
		changed := false
		if newCond, condChanged := cp.propagateInExpr(e.Condition); condChanged {
			e.Condition = newCond
			changed = true
		}
		for i := range e.Clauses {
			if e.Clauses[i].Guard != nil {
				if newGuard, guardChanged := cp.propagateInExpr(e.Clauses[i].Guard); guardChanged {
					e.Clauses[i].Guard = newGuard
					changed = true
				}
			}
			if newResult, resultChanged := cp.propagateInExpr(e.Clauses[i].Result); resultChanged {
				e.Clauses[i].Result = newResult
				changed = true
			}
		}
		if e.DefaultExpr != nil {
			if newDefault, defaultChanged := cp.propagateInExpr(e.DefaultExpr); defaultChanged {
				e.DefaultExpr = newDefault
				changed = true
			}
		}
		return e, changed
	}

	return expr, false
}

func (cp *ConstantPropagation) isConstant(expr Expression) bool {
	switch expr.(type) {
	case *NumberExpr, *StringExpr:
		return true
	default:
		return false
	}
}

func (cp *ConstantPropagation) foldConstants(left, right *NumberExpr, op string) *NumberExpr {
	switch op {
	case "+":
		return &NumberExpr{Value: left.Value + right.Value}
	case "-":
		return &NumberExpr{Value: left.Value - right.Value}
	case "*":
		return &NumberExpr{Value: left.Value * right.Value}
	case "/":
		if right.Value != 0 {
			return &NumberExpr{Value: left.Value / right.Value}
		}
	case "%":
		if right.Value != 0 {
			return &NumberExpr{Value: float64(int64(left.Value) % int64(right.Value))}
		}
	}
	return nil
}

// DeadCodeElimination removes unreachable code
type DeadCodeElimination struct{}

func (dce *DeadCodeElimination) Name() string {
	return "Dead Code Elimination"
}

func (dce *DeadCodeElimination) Run(program *Program) (bool, error) {
	// Track which functions/variables are actually used
	used := make(map[string]bool)
	dce.markUsed(program, used)

	// Remove unused assignments
	changed := false
	newStmts := make([]Statement, 0, len(program.Statements))
	for _, stmt := range program.Statements {
		if assign, ok := stmt.(*AssignStmt); ok {
			if !used[assign.Name] {
				// Never remove hot functions - they must exist for hot reload
				if assign.IsHot {
					if VerboseMode {
						fmt.Fprintf(os.Stderr, "   DCE: Preserving hot function: %s\n", assign.Name)
					}
				} else {
					changed = true
					if VerboseMode {
						fmt.Fprintf(os.Stderr, "   DCE: Removing unused assignment: %s\n", assign.Name)
					}
					continue
				}
			}
		}
		newStmts = append(newStmts, stmt)
	}

	if changed {
		program.Statements = newStmts
	}

	return changed, nil
}

func (dce *DeadCodeElimination) markUsed(program *Program, used map[string]bool) {
	for _, stmt := range program.Statements {
		dce.markUsedInStmt(stmt, used)
	}
}

func (dce *DeadCodeElimination) markUsedInStmt(stmt Statement, used map[string]bool) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Mark the variable as used if this is a mutation (<-)
		if s.IsUpdate {
			used[s.Name] = true
		}
		dce.markUsedInExpr(s.Value, used)

	case *ExpressionStmt:
		dce.markUsedInExpr(s.Expr, used)

	case *LoopStmt:
		dce.markUsedInExpr(s.Iterable, used)
		for _, bodyStmt := range s.Body {
			dce.markUsedInStmt(bodyStmt, used)
		}

	case *JumpStmt:
		if s.Value != nil {
			dce.markUsedInExpr(s.Value, used)
		}
	}
}

func (dce *DeadCodeElimination) markUsedInExpr(expr Expression, used map[string]bool) {
	switch e := expr.(type) {
	case *IdentExpr:
		used[e.Name] = true

	case *MoveExpr:
		// Mark the moved expression as used
		dce.markUsedInExpr(e.Expr, used)

	case *BinaryExpr:
		dce.markUsedInExpr(e.Left, used)
		dce.markUsedInExpr(e.Right, used)

	case *CallExpr:
		used[e.Function] = true
		for _, arg := range e.Args {
			dce.markUsedInExpr(arg, used)
		}

	case *LambdaExpr:
		dce.markUsedInExpr(e.Body, used)

	case *MatchExpr:
		dce.markUsedInExpr(e.Condition, used)
		for _, c := range e.Clauses {
			if c.Guard != nil {
				dce.markUsedInExpr(c.Guard, used)
			}
			dce.markUsedInExpr(c.Result, used)
		}
		if e.DefaultExpr != nil {
			dce.markUsedInExpr(e.DefaultExpr, used)
		}

	case *MapExpr:
		for _, key := range e.Keys {
			dce.markUsedInExpr(key, used)
		}
		for _, val := range e.Values {
			dce.markUsedInExpr(val, used)
		}

	case *PipeExpr:
		dce.markUsedInExpr(e.Left, used)
		dce.markUsedInExpr(e.Right, used)

	case *LengthExpr:
		dce.markUsedInExpr(e.Operand, used)

	case *ListExpr:
		for _, elem := range e.Elements {
			dce.markUsedInExpr(elem, used)
		}

	case *IndexExpr:
		dce.markUsedInExpr(e.List, used)
		dce.markUsedInExpr(e.Index, used)

	case *SliceExpr:
		dce.markUsedInExpr(e.List, used)
		if e.Start != nil {
			dce.markUsedInExpr(e.Start, used)
		}
		if e.End != nil {
			dce.markUsedInExpr(e.End, used)
		}
		if e.Step != nil {
			dce.markUsedInExpr(e.Step, used)
		}

	case *UnaryExpr:
		dce.markUsedInExpr(e.Operand, used)

	case *BlockExpr:
		for _, stmt := range e.Statements {
			dce.markUsedInStmt(stmt, used)
		}

	case *RangeExpr:
		dce.markUsedInExpr(e.Start, used)
		dce.markUsedInExpr(e.End, used)

	case *InExpr:
		dce.markUsedInExpr(e.Value, used)
		dce.markUsedInExpr(e.Container, used)

	case *CastExpr:
		dce.markUsedInExpr(e.Expr, used)

	case *UnsafeExpr:
		for _, stmt := range e.X86_64Block {
			dce.markUsedInStmt(stmt, used)
		}
		for _, stmt := range e.ARM64Block {
			dce.markUsedInStmt(stmt, used)
		}
		for _, stmt := range e.RISCV64Block {
			dce.markUsedInStmt(stmt, used)
		}

	case *ArenaExpr:
		for _, stmt := range e.Body {
			dce.markUsedInStmt(stmt, used)
		}

	case *ParallelExpr:
		dce.markUsedInExpr(e.List, used)
		dce.markUsedInExpr(e.Operation, used)

	case *FStringExpr:
		for _, part := range e.Parts {
			dce.markUsedInExpr(part, used)
		}

	case *DirectCallExpr:
		dce.markUsedInExpr(e.Callee, used)
		for _, arg := range e.Args {
			dce.markUsedInExpr(arg, used)
		}

	case *NamespacedIdentExpr:
		used[e.Namespace] = true

	case *PostfixExpr:
		dce.markUsedInExpr(e.Operand, used)

	case *VectorExpr:
		for _, comp := range e.Components {
			dce.markUsedInExpr(comp, used)
		}

	case *LoopExpr:
		for _, stmt := range e.Body {
			dce.markUsedInStmt(stmt, used)
		}

	case *MultiLambdaExpr:
		for _, lambda := range e.Lambdas {
			dce.markUsedInExpr(lambda.Body, used)
		}
	}
}

// FunctionInlining replaces small function calls with function bodies
type FunctionInlining struct {
	functions map[string]*LambdaExpr
}

func (fi *FunctionInlining) Name() string {
	return "Function Inlining"
}

func (fi *FunctionInlining) Run(program *Program) (bool, error) {
	fi.functions = make(map[string]*LambdaExpr)

	// Collect simple lambda assignments
	for _, stmt := range program.Statements {
		if assign, ok := stmt.(*AssignStmt); ok {
			if lambda, ok := assign.Value.(*LambdaExpr); ok {
				// Only inline simple lambdas (no closures, single expression)
				if len(lambda.CapturedVars) == 0 && fi.isSimpleBody(lambda.Body) {
					fi.functions[assign.Name] = lambda
				}
			}
		}
	}

	// Inline calls to these functions
	changed := false
	for _, stmt := range program.Statements {
		if fi.inlineInStmt(stmt) {
			changed = true
		}
	}

	return changed, nil
}

func (fi *FunctionInlining) isSimpleBody(expr Expression) bool {
	// Simple: single expression, no loops, no nested lambdas
	switch e := expr.(type) {
	case *NumberExpr, *StringExpr, *IdentExpr:
		return true
	case *BinaryExpr:
		return fi.isSimpleBody(e.Left) && fi.isSimpleBody(e.Right)
	case *LambdaExpr:
		return false
	default:
		return false
	}
}

func (fi *FunctionInlining) inlineInStmt(stmt Statement) bool {
	changed := false

	switch s := stmt.(type) {
	case *AssignStmt:
		if newExpr, ok := fi.inlineInExpr(s.Value); ok {
			s.Value = newExpr
			changed = true
		}

	case *ExpressionStmt:
		if newExpr, ok := fi.inlineInExpr(s.Expr); ok {
			s.Expr = newExpr
			changed = true
		}

	case *LoopStmt:
		for _, bodyStmt := range s.Body {
			if fi.inlineInStmt(bodyStmt) {
				changed = true
			}
		}

	case *JumpStmt:
		if s.Value != nil {
			if newExpr, ok := fi.inlineInExpr(s.Value); ok {
				s.Value = newExpr
				changed = true
			}
		}
	}

	return changed
}

func (fi *FunctionInlining) inlineInExpr(expr Expression) (Expression, bool) {
	switch e := expr.(type) {
	case *CallExpr:
		// Check if calling an inlineable function directly by name
		if lambda, exists := fi.functions[e.Function]; exists {
			// Simple inlining: substitute parameters with arguments
			if len(lambda.Params) == len(e.Args) {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "   Inlining: %s\n", e.Function)
				}
				return fi.substitute(lambda.Body, lambda.Params, e.Args), true
			}
		}

		// Inline arguments
		changed := false
		for i, arg := range e.Args {
			if newArg, argChanged := fi.inlineInExpr(arg); argChanged {
				e.Args[i] = newArg
				changed = true
			}
		}
		return e, changed

	case *BinaryExpr:
		changed := false
		if newLeft, leftChanged := fi.inlineInExpr(e.Left); leftChanged {
			e.Left = newLeft
			changed = true
		}
		if newRight, rightChanged := fi.inlineInExpr(e.Right); rightChanged {
			e.Right = newRight
			changed = true
		}
		return e, changed

	case *MoveExpr:
		// Process the inner expression
		if newExpr, changed := fi.inlineInExpr(e.Expr); changed {
			e.Expr = newExpr
			return e, true
		}
		return e, false
	}

	return expr, false
}

func (fi *FunctionInlining) substitute(body Expression, params []string, args []Expression) Expression {
	subst := make(map[string]Expression)
	for i, param := range params {
		if i < len(args) {
			subst[param] = args[i]
		}
	}
	return fi.substituteExpr(body, subst)
}

func (fi *FunctionInlining) substituteExpr(expr Expression, subst map[string]Expression) Expression {
	switch e := expr.(type) {
	case *IdentExpr:
		if replacement, ok := subst[e.Name]; ok {
			return replacement
		}
		return e

	case *BinaryExpr:
		return &BinaryExpr{
			Left:     fi.substituteExpr(e.Left, subst),
			Operator: e.Operator,
			Right:    fi.substituteExpr(e.Right, subst),
		}

	case *MoveExpr:
		return &MoveExpr{
			Expr: fi.substituteExpr(e.Expr, subst),
		}

	case *CallExpr:
		newArgs := make([]Expression, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i] = fi.substituteExpr(arg, subst)
		}
		return &CallExpr{
			Function:            e.Function,
			Args:                newArgs,
			MaxRecursionDepth:   e.MaxRecursionDepth,
			NeedsRecursionCheck: e.NeedsRecursionCheck,
		}

	default:
		return expr
	}
}

type LoopUnrolling struct{}

func (lu *LoopUnrolling) Name() string {
	return "Loop Unrolling"
}

func (lu *LoopUnrolling) Run(program *Program) (bool, error) {
	changed := false
	newStatements := []Statement{}

	for _, stmt := range program.Statements {
		unrolled, wasUnrolled := lu.tryUnrollLoop(stmt)
		if wasUnrolled {
			changed = true
			newStatements = append(newStatements, unrolled...)
		} else {
			newStatements = append(newStatements, stmt)
		}
	}

	if changed {
		program.Statements = newStatements
	}
	return changed, nil
}

func (lu *LoopUnrolling) tryUnrollLoop(stmt Statement) ([]Statement, bool) {
	loopStmt, ok := stmt.(*LoopStmt)
	if !ok {
		return nil, false
	}

	// Don't unroll parallel loops (@@  or N @)
	// These need to remain as LoopStmt for parallel code generation
	if loopStmt.NumThreads != 0 {
		return nil, false
	}

	rangeExpr, ok := loopStmt.Iterable.(*RangeExpr)
	if !ok {
		return nil, false
	}

	startNum, ok := rangeExpr.Start.(*NumberExpr)
	if !ok {
		return nil, false
	}

	endNum, ok := rangeExpr.End.(*NumberExpr)
	if !ok {
		return nil, false
	}

	start := int(startNum.Value)
	end := int(endNum.Value)

	if rangeExpr.Inclusive {
		end++
	}

	iterations := end - start
	if iterations <= 0 || iterations > 8 {
		return nil, false
	}

	// Check if this loop contains nested loops
	hasNestedLoops := lu.containsNestedLoops(loopStmt.Body)

	unrolled := make([]Statement, 0, iterations*len(loopStmt.Body))
	for i := start; i < end; i++ {
		for _, bodyStmt := range loopStmt.Body {
			substituted := lu.substituteIterator(bodyStmt, loopStmt.Iterator, float64(i), hasNestedLoops)
			unrolled = append(unrolled, substituted)
		}
	}

	// Convert duplicate variable definitions to updates (recursively)
	seenDefs := make(map[string]bool)
	lu.fixDuplicateDefinitions(unrolled, seenDefs)

	return unrolled, true
}

func (lu *LoopUnrolling) containsNestedLoops(stmts []Statement) bool {
	for _, stmt := range stmts {
		if _, isLoop := stmt.(*LoopStmt); isLoop {
			return true
		}
	}
	return false
}

func (lu *LoopUnrolling) fixDuplicateDefinitions(stmts []Statement, seenDefs map[string]bool) {
	for _, stmt := range stmts {
		switch s := stmt.(type) {
		case *AssignStmt:
			if !s.IsUpdate && s.Mutable {
				if seenDefs[s.Name] {
					s.IsUpdate = true
				} else {
					seenDefs[s.Name] = true
				}
			}
		case *LoopStmt:
			lu.fixDuplicateDefinitions(s.Body, seenDefs)
		}
	}
}

func (lu *LoopUnrolling) substituteIterator(stmt Statement, iterator string, value float64, hasNestedLoops bool) Statement {
	switch s := stmt.(type) {
	case *AssignStmt:
		return &AssignStmt{
			Name:      s.Name,
			Value:     lu.substituteIteratorInExpr(s.Value, iterator, value, hasNestedLoops),
			Mutable:   s.Mutable,
			IsUpdate:  s.IsUpdate,
			Precision: s.Precision,
			IsHot:     s.IsHot,
		}
	case *ExpressionStmt:
		return &ExpressionStmt{
			Expr: lu.substituteIteratorInExpr(s.Expr, iterator, value, hasNestedLoops),
		}
	case *LoopStmt:
		// Substitute iterator in nested loop's iterable expression
		newBody := make([]Statement, len(s.Body))
		for i, bodyStmt := range s.Body {
			newBody[i] = lu.substituteIterator(bodyStmt, iterator, value, hasNestedLoops)
		}
		return &LoopStmt{
			Iterator:      s.Iterator,
			Iterable:      lu.substituteIteratorInExpr(s.Iterable, iterator, value, hasNestedLoops),
			Body:          newBody,
			MaxIterations: s.MaxIterations,
			NeedsMaxCheck: s.NeedsMaxCheck,
			BaseOffset:    s.BaseOffset,
		}
	default:
		return stmt
	}
}

func (lu *LoopUnrolling) substituteIteratorInExpr(expr Expression, iterator string, value float64, hasNestedLoops bool) Expression {
	switch e := expr.(type) {
	case *IdentExpr:
		if e.Name == iterator {
			return &NumberExpr{Value: value}
		}
		return e
	case *BinaryExpr:
		return &BinaryExpr{
			Left:     lu.substituteIteratorInExpr(e.Left, iterator, value, hasNestedLoops),
			Operator: e.Operator,
			Right:    lu.substituteIteratorInExpr(e.Right, iterator, value, hasNestedLoops),
		}
	case *CallExpr:
		newArgs := make([]Expression, len(e.Args))
		for i, arg := range e.Args {
			newArgs[i] = lu.substituteIteratorInExpr(arg, iterator, value, hasNestedLoops)
		}
		return &CallExpr{
			Function:            e.Function,
			Args:                newArgs,
			MaxRecursionDepth:   e.MaxRecursionDepth,
			NeedsRecursionCheck: e.NeedsRecursionCheck,
		}
	case *CastExpr:
		return &CastExpr{
			Expr: lu.substituteIteratorInExpr(e.Expr, iterator, value, hasNestedLoops),
			Type: e.Type,
		}
	case *LoopStateExpr:
		if e.Type == "i" {
			if e.LoopLevel == 1 {
				// Always replace @i1 (outermost loop reference) with the value
				return &NumberExpr{Value: value}
			} else if e.LoopLevel > 1 {
				// Decrement loop levels deeper than outermost
				return &LoopStateExpr{
					Type:      e.Type,
					LoopLevel: e.LoopLevel - 1,
				}
			} else if e.LoopLevel == 0 && !hasNestedLoops {
				// Only replace @i (current loop) if there are no nested loops
				return &NumberExpr{Value: value}
			}
		}
		return e
	default:
		return expr
	}
}
