package main

import (
	"fmt"
	"strings"
)

// AST Nodes
type Node interface {
	String() string
}

type Program struct {
	Statements []Statement
}

func (p *Program) String() string {
	var out strings.Builder
	for _, stmt := range p.Statements {
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}
	return out.String()
}

type Statement interface {
	Node
	statementNode()
}

type AssignStmt struct {
	Name      string
	Value     Expression
	Mutable   bool   // true for := or <-, false for =
	IsUpdate  bool   // true for <-, false for = and :=
	Precision string // Type annotation: "b64", "f32", etc. (empty if none)
}

func (a *AssignStmt) String() string {
	op := "="
	if a.IsUpdate {
		op = "<-"
	} else if a.Mutable {
		op = ":="
	}
	result := a.Name
	if a.Precision != "" {
		result += ":" + a.Precision
	}
	if a.Value == nil {
		return result + " " + op + " <nil>"
	}
	return result + " " + op + " " + a.Value.String()
}
func (a *AssignStmt) statementNode() {}

type UseStmt struct {
	Path string // Import path: "./file.flap" or "package_name"
}

func (u *UseStmt) String() string { return "use " + u.Path }
func (u *UseStmt) statementNode() {}

type ImportStmt struct {
	URL     string // Git URL: "github.com/owner/repo"
	Version string // Git ref: "v1.0.0", "HEAD", "latest", "commit-hash", or "" for latest
	Alias   string // Namespace alias: "xmath" or "*" for wildcard
}

func (i *ImportStmt) String() string {
	url := i.URL
	if i.Version != "" {
		url += "@" + i.Version
	}
	return "import " + url + " as " + i.Alias
}
func (i *ImportStmt) statementNode() {}

type ExpressionStmt struct {
	Expr Expression
}

func (e *ExpressionStmt) String() string { return e.Expr.String() }
func (e *ExpressionStmt) statementNode() {}

type LoopStmt struct {
	// No explicit label - determined by nesting depth when created with @+
	Iterator string     // Variable name (e.g., "i")
	Iterable Expression // Expression to iterate over (e.g., range(10))
	Body     []Statement
}

type LoopExpr struct {
	// No explicit label - determined by nesting depth when created with @+
	Iterator string      // Variable name (e.g., "i")
	Iterable Expression  // Expression to iterate over (e.g., range(10))
	Body     []Statement // Body statements
}

func (l *LoopExpr) String() string {
	return fmt.Sprintf("@+ %s in %s { ... }", l.Iterator, l.Iterable.String())
}
func (l *LoopExpr) expressionNode() {}

func (l *LoopStmt) String() string {
	var out strings.Builder
	out.WriteString("@+ ")
	out.WriteString(l.Iterator)
	out.WriteString(" in ")
	out.WriteString(l.Iterable.String())
	out.WriteString(" {\n")
	for _, stmt := range l.Body {
		out.WriteString("  ")
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}
	out.WriteString("}")
	return out.String()
}
func (l *LoopStmt) statementNode() {}

// JumpStmt represents a ret statement or loop continue
// ret (Label=0) = return from function
// ret @N (Label=N) = exit loop N and all inner loops
// @N (without ret) = continue loop N (IsBreak=false)
type JumpStmt struct {
	IsBreak bool       // true for ret (return/exit loop), false for continue (@N without ret)
	Label   int        // 0 for function return, N for loop label
	Value   Expression // Optional value to return
}

func (j *JumpStmt) String() string {
	keyword := "@"
	if j.IsBreak {
		keyword = "ret"
	}

	if j.Label > 0 {
		if j.Value != nil {
			return fmt.Sprintf("%s @%d %s", keyword, j.Label, j.Value.String())
		}
		return fmt.Sprintf("%s @%d", keyword, j.Label)
	}

	if j.Value != nil {
		return fmt.Sprintf("%s %s", keyword, j.Value.String())
	}
	return keyword
}
func (j *JumpStmt) statementNode() {}

type Expression interface {
	Node
	expressionNode()
}

type NumberExpr struct {
	Value float64
}

func (n *NumberExpr) String() string  { return fmt.Sprintf("%g", n.Value) }
func (n *NumberExpr) expressionNode() {}

type StringExpr struct {
	Value string
}

func (s *StringExpr) String() string  { return fmt.Sprintf("\"%s\"", s.Value) }
func (s *StringExpr) expressionNode() {}

// FStringExpr represents an f-string with interpolated expressions
// Parts alternates between string literals and expressions
// Example: f"Hello {name}" -> Parts = [StringExpr("Hello "), IdentExpr("name")]
type FStringExpr struct {
	Parts []Expression // Alternating string literals and expressions
}

func (f *FStringExpr) String() string { return "f\"...\"" }
func (f *FStringExpr) expressionNode() {}

type IdentExpr struct {
	Name string
}

func (i *IdentExpr) String() string  { return i.Name }
func (i *IdentExpr) expressionNode() {}

// LoopStateExpr represents special loop variables: @first, @last, @counter, @i
type LoopStateExpr struct {
	Type string // "first", "last", "counter", "i"
}

func (l *LoopStateExpr) String() string {
	return "@" + l.Type
}
func (l *LoopStateExpr) expressionNode() {}

// JumpExpr represents a label jump used as an expression (e.g., in match blocks)
type JumpExpr struct {
	Label   int        // Target label (0 = outer scope, N = loop label)
	Value   Expression // Optional value to return (for @0 value syntax)
	IsBreak bool       // true for ret @N (exit loop), false for @N (continue loop)
}

func (j *JumpExpr) String() string {
	prefix := "@"
	if j.IsBreak {
		prefix = "ret @"
	}
	if j.Value != nil {
		return fmt.Sprintf("%s%d %s", prefix, j.Label, j.Value.String())
	}
	return fmt.Sprintf("%s%d", prefix, j.Label)
}
func (j *JumpExpr) expressionNode() {}

type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpr) String() string {
	return "(" + b.Left.String() + " " + b.Operator + " " + b.Right.String() + ")"
}
func (b *BinaryExpr) expressionNode() {}

// UnaryExpr represents a unary operation: not, -, #, ++expr, --expr
type UnaryExpr struct {
	Operator string
	Operand  Expression
}

func (u *UnaryExpr) String() string {
	return "(" + u.Operator + u.Operand.String() + ")"
}
func (u *UnaryExpr) expressionNode() {}

// PostfixExpr: expr++, expr-- (increment/decrement after evaluation)
type PostfixExpr struct {
	Operator string // "++", "--"
	Operand  Expression
}

func (p *PostfixExpr) String() string {
	return "(" + p.Operand.String() + p.Operator + ")"
}
func (p *PostfixExpr) expressionNode() {}

type InExpr struct {
	Value     Expression // Value to search for
	Container Expression // List or map to search in
}

func (i *InExpr) String() string {
	return "(" + i.Value.String() + " in " + i.Container.String() + ")"
}
func (i *InExpr) expressionNode() {}

type MatchClause struct {
	Guard  Expression
	Result Expression
}

type MatchExpr struct {
	Condition       Expression
	Clauses         []*MatchClause
	DefaultExpr     Expression
	DefaultExplicit bool
}

func (m *MatchExpr) String() string {
	var parts []string
	for _, clause := range m.Clauses {
		if clause.Guard != nil {
			if clause.Result != nil {
				parts = append(parts, clause.Guard.String()+" -> "+clause.Result.String())
			} else {
				parts = append(parts, clause.Guard.String()+" -> <statement>")
			}
		} else {
			if clause.Result != nil {
				parts = append(parts, "-> "+clause.Result.String())
			} else {
				parts = append(parts, "-> <statement>")
			}
		}
	}
	if m.DefaultExpr != nil && (m.DefaultExplicit || len(m.Clauses) == 0) {
		parts = append(parts, "~> "+m.DefaultExpr.String())
	}
	return m.Condition.String() + " { " + strings.Join(parts, " ") + " }"
}
func (m *MatchExpr) expressionNode() {}

type BlockExpr struct {
	Statements []Statement
}

func (b *BlockExpr) String() string {
	var parts []string
	for _, stmt := range b.Statements {
		parts = append(parts, stmt.String())
	}
	return "{ " + strings.Join(parts, "; ") + " }"
}
func (b *BlockExpr) expressionNode() {}

type CallExpr struct {
	Function string
	Args     []Expression
}

func (c *CallExpr) String() string {
	args := make([]string, len(c.Args))
	for i, arg := range c.Args {
		if arg == nil {
			args[i] = "<nil>"
		} else {
			args[i] = arg.String()
		}
	}
	return c.Function + "(" + strings.Join(args, ", ") + ")"
}
func (c *CallExpr) expressionNode() {}

type DirectCallExpr struct {
	Callee Expression // The expression being called (e.g., a lambda)
	Args   []Expression
}

func (d *DirectCallExpr) String() string {
	args := make([]string, len(d.Args))
	for i, arg := range d.Args {
		args[i] = arg.String()
	}
	return "(" + d.Callee.String() + ")(" + strings.Join(args, ", ") + ")"
}
func (d *DirectCallExpr) expressionNode() {}

type ListExpr struct {
	Elements []Expression
}

func (l *ListExpr) String() string {
	elements := make([]string, len(l.Elements))
	for i, elem := range l.Elements {
		elements[i] = elem.String()
	}
	return "[" + strings.Join(elements, ", ") + "]"
}
func (l *ListExpr) expressionNode() {}

type MapExpr struct {
	Keys   []Expression
	Values []Expression
}

func (m *MapExpr) String() string {
	var pairs []string
	for i := range m.Keys {
		pairs = append(pairs, m.Keys[i].String()+": "+m.Values[i].String())
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}
func (m *MapExpr) expressionNode() {}

type IndexExpr struct {
	List  Expression
	Index Expression
}

func (i *IndexExpr) String() string {
	if i.List == nil || i.Index == nil {
		return fmt.Sprintf("IndexExpr{List=%v, Index=%v}", i.List, i.Index)
	}
	return i.List.String() + "[" + i.Index.String() + "]"
}
func (i *IndexExpr) expressionNode() {}

// SliceExpr: list[start:end:step] or string[start:end:step] (Python-style slicing)
type SliceExpr struct {
	List  Expression
	Start Expression // nil means start from beginning
	End   Expression // nil means go to end
	Step  Expression // nil means step of 1, negative means reverse
}

func (s *SliceExpr) String() string {
	start := ""
	if s.Start != nil {
		start = s.Start.String()
	}
	end := ""
	if s.End != nil {
		end = s.End.String()
	}
	result := s.List.String() + "[" + start + ":" + end
	if s.Step != nil {
		result += ":" + s.Step.String()
	}
	result += "]"
	return result
}
func (s *SliceExpr) expressionNode() {}

type LambdaExpr struct {
	Params []string
	Body   Expression
}

func (l *LambdaExpr) String() string {
	return "(" + strings.Join(l.Params, ", ") + ") -> " + l.Body.String()
}
func (l *LambdaExpr) expressionNode() {}

// MultiLambdaExpr: multiple lambda dispatch based on argument count
// Example: f = (x) -> x, (x, y) -> x + y
type MultiLambdaExpr struct {
	Lambdas []*LambdaExpr
}

func (m *MultiLambdaExpr) String() string {
	parts := make([]string, len(m.Lambdas))
	for i, lambda := range m.Lambdas {
		parts[i] = lambda.String()
	}
	return strings.Join(parts, ", ")
}
func (m *MultiLambdaExpr) expressionNode() {}

type ParallelExpr struct {
	List      Expression // The list/data to operate on
	Operation Expression // The lambda or function to apply
}

func (p *ParallelExpr) String() string {
	return p.List.String() + " || " + p.Operation.String()
}
func (p *ParallelExpr) expressionNode() {}

type PipeExpr struct {
	Left  Expression // Input to the pipe
	Right Expression // Operation to apply
}

func (p *PipeExpr) String() string {
	return p.Left.String() + " | " + p.Right.String()
}
func (p *PipeExpr) expressionNode() {}

type ConcurrentGatherExpr struct {
	Left  Expression // Input to the concurrent gather
	Right Expression // Operation to apply concurrently
}

func (c *ConcurrentGatherExpr) String() string {
	return c.Left.String() + " ||| " + c.Right.String()
}
func (c *ConcurrentGatherExpr) expressionNode() {}

type LengthExpr struct {
	Operand Expression
}

func (l *LengthExpr) String() string {
	return "#" + l.Operand.String()
}
func (l *LengthExpr) expressionNode() {}

type CastExpr struct {
	Expr Expression
	Type string // "i8", "i32", "u64", "f32", "f64", "cstr", "ptr", "number", "string", "list"
}

func (c *CastExpr) String() string {
	if c.Expr == nil {
		return "<nil> as " + c.Type
	}
	return c.Expr.String() + " as " + c.Type
}
func (c *CastExpr) expressionNode() {}
