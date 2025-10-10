package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unsafe"
)

// Token types for Flap language
type TokenType int

const (
	TOKEN_EOF TokenType = iota
	TOKEN_IDENT
	TOKEN_NUMBER
	TOKEN_STRING
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_STAR
	TOKEN_SLASH
	TOKEN_EQUALS
	TOKEN_COLON_EQUALS
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_COMMA
	TOKEN_COLON
	TOKEN_NEWLINE
	TOKEN_LT            // <
	TOKEN_GT            // >
	TOKEN_LE            // <=
	TOKEN_GE            // >=
	TOKEN_EQ            // ==
	TOKEN_NE            // !=
	TOKEN_TILDE         // ~
	TOKEN_DEFAULT_ARROW // ~>
	TOKEN_AT            // @
	TOKEN_IN            // in keyword
	TOKEN_FOR           // for keyword
	TOKEN_BREAK         // break keyword
	TOKEN_CONTINUE      // continue keyword
	TOKEN_LBRACE        // {
	TOKEN_RBRACE        // }
	TOKEN_LBRACKET      // [
	TOKEN_RBRACKET      // ]
	TOKEN_ARROW         // ->
	TOKEN_PIPE          // |
	TOKEN_PIPEPIPE      // ||
	TOKEN_PIPEPIPEPIPE  // |||
	TOKEN_HASH          // #
	TOKEN_AND           // and keyword
	TOKEN_OR            // or keyword
	TOKEN_NOT           // not keyword
	TOKEN_XOR           // xor keyword
	TOKEN_SHL           // shl keyword
	TOKEN_SHR           // shr keyword
	TOKEN_ROL           // rol keyword
	TOKEN_ROR           // ror keyword
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// processEscapeSequences converts escape sequences in a string to their actual characters
func processEscapeSequences(s string) string {
	var result strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'n':
				result.WriteByte('\n')
			case 't':
				result.WriteByte('\t')
			case 'r':
				result.WriteByte('\r')
			case '\\':
				result.WriteByte('\\')
			case '"':
				result.WriteByte('"')
			default:
				// Unknown escape sequence - keep backslash and the character
				result.WriteByte(s[i])
				result.WriteByte(s[i+1])
			}
			i++ // Skip the escaped character
		} else {
			result.WriteByte(s[i])
		}
	}
	return result.String()
}

// Lexer for Flap language
type Lexer struct {
	input string
	pos   int
	line  int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input, pos: 0, line: 1}
}

func (l *Lexer) peek() byte {
	if l.pos+1 < len(l.input) {
		return l.input[l.pos+1]
	}
	return 0
}

func (l *Lexer) advance() {
	if l.pos < len(l.input) {
		l.pos++
	}
}

func (l *Lexer) NextToken() Token {
	// Skip whitespace (except newlines)
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t' || l.input[l.pos] == '\r') {
		l.pos++
	}

	// Skip comments (lines starting with //)
	if l.pos < len(l.input)-1 && l.input[l.pos] == '/' && l.input[l.pos+1] == '/' {
		for l.pos < len(l.input) && l.input[l.pos] != '\n' {
			l.pos++
		}
	}

	if l.pos >= len(l.input) {
		return Token{Type: TOKEN_EOF, Line: l.line}
	}

	ch := l.input[l.pos]

	// Newline
	if ch == '\n' {
		l.pos++
		l.line++
		return Token{Type: TOKEN_NEWLINE, Line: l.line - 1}
	}

	// String literal
	if ch == '"' {
		l.pos++
		start := l.pos
		for l.pos < len(l.input) && l.input[l.pos] != '"' {
			l.pos++
		}
		value := l.input[start:l.pos]
		l.pos++ // skip closing "
		return Token{Type: TOKEN_STRING, Value: value, Line: l.line}
	}

	// Number
	if unicode.IsDigit(rune(ch)) {
		start := l.pos
		for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.') {
			l.pos++
		}
		return Token{Type: TOKEN_NUMBER, Value: l.input[start:l.pos], Line: l.line}
	}

	// Identifier or keyword
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		start := l.pos
		for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_') {
			l.pos++
		}
		value := l.input[start:l.pos]

		// Check for keywords
		switch value {
		case "in":
			return Token{Type: TOKEN_IN, Value: value, Line: l.line}
		case "for":
			return Token{Type: TOKEN_FOR, Value: value, Line: l.line}
		case "break":
			return Token{Type: TOKEN_BREAK, Value: value, Line: l.line}
		case "continue":
			return Token{Type: TOKEN_CONTINUE, Value: value, Line: l.line}
		case "and":
			return Token{Type: TOKEN_AND, Value: value, Line: l.line}
		case "or":
			return Token{Type: TOKEN_OR, Value: value, Line: l.line}
		case "not":
			return Token{Type: TOKEN_NOT, Value: value, Line: l.line}
		case "xor":
			return Token{Type: TOKEN_XOR, Value: value, Line: l.line}
		case "shl":
			return Token{Type: TOKEN_SHL, Value: value, Line: l.line}
		case "shr":
			return Token{Type: TOKEN_SHR, Value: value, Line: l.line}
		case "rol":
			return Token{Type: TOKEN_ROL, Value: value, Line: l.line}
		case "ror":
			return Token{Type: TOKEN_ROR, Value: value, Line: l.line}
		}

		return Token{Type: TOKEN_IDENT, Value: value, Line: l.line}
	}

	// Operators and punctuation
	switch ch {
	case '+':
		l.pos++
		return Token{Type: TOKEN_PLUS, Value: "+", Line: l.line}
	case '-':
		// Check for negative number literal
		if l.peek() != '>' && l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.peek())) {
			start := l.pos
			l.pos++ // skip '-'
			for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.') {
				l.pos++
			}
			return Token{Type: TOKEN_NUMBER, Value: l.input[start:l.pos], Line: l.line}
		}
		// Check for ->
		if l.peek() == '>' {
			l.pos += 2
			return Token{Type: TOKEN_ARROW, Value: "->", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_MINUS, Value: "-", Line: l.line}
	case '*':
		l.pos++
		return Token{Type: TOKEN_STAR, Value: "*", Line: l.line}
	case '/':
		l.pos++
		return Token{Type: TOKEN_SLASH, Value: "/", Line: l.line}
	case ':':
		// Check for := before advancing
		if l.peek() == '=' {
			l.pos += 2 // skip both ':' and '='
			return Token{Type: TOKEN_COLON_EQUALS, Value: ":=", Line: l.line}
		}
		// Standalone : for map literals
		l.pos++
		return Token{Type: TOKEN_COLON, Value: ":", Line: l.line}
	case '=':
		// Check for ==
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_EQ, Value: "==", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_EQUALS, Value: "=", Line: l.line}
	case '<':
		// Check for <=
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_LE, Value: "<=", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_LT, Value: "<", Line: l.line}
	case '>':
		// Check for >=
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_GE, Value: ">=", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_GT, Value: ">", Line: l.line}
	case '!':
		// Check for !=
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_NE, Value: "!=", Line: l.line}
		}
		// Just ! is not supported, skip
		l.pos++
		return l.NextToken()
	case '~':
		// Check for ~>
		if l.peek() == '>' {
			l.pos += 2
			return Token{Type: TOKEN_DEFAULT_ARROW, Value: "~>", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_TILDE, Value: "~", Line: l.line}
	case '(':
		l.pos++
		return Token{Type: TOKEN_LPAREN, Value: "(", Line: l.line}
	case ')':
		l.pos++
		return Token{Type: TOKEN_RPAREN, Value: ")", Line: l.line}
	case ',':
		l.pos++
		return Token{Type: TOKEN_COMMA, Value: ",", Line: l.line}
	case '@':
		l.pos++
		return Token{Type: TOKEN_AT, Value: "@", Line: l.line}
	case '{':
		l.pos++
		return Token{Type: TOKEN_LBRACE, Value: "{", Line: l.line}
	case '}':
		l.pos++
		return Token{Type: TOKEN_RBRACE, Value: "}", Line: l.line}
	case '[':
		l.pos++
		return Token{Type: TOKEN_LBRACKET, Value: "[", Line: l.line}
	case ']':
		l.pos++
		return Token{Type: TOKEN_RBRACKET, Value: "]", Line: l.line}
	case '|':
		// Check for ||| first, then ||, then |
		if l.peek() == '|' {
			if l.pos+2 < len(l.input) && l.input[l.pos+2] == '|' {
				l.pos += 3
				return Token{Type: TOKEN_PIPEPIPEPIPE, Value: "|||", Line: l.line}
			}
			l.pos += 2
			return Token{Type: TOKEN_PIPEPIPE, Value: "||", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_PIPE, Value: "|", Line: l.line}
	case '#':
		l.pos++
		return Token{Type: TOKEN_HASH, Value: "#", Line: l.line}
	}

	return Token{Type: TOKEN_EOF, Line: l.line}
}

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
	Name    string
	Value   Expression
	Mutable bool // true for :=, false for =
}

func (a *AssignStmt) String() string {
	op := "="
	if a.Mutable {
		op = ":="
	}
	return a.Name + " " + op + " " + a.Value.String()
}
func (a *AssignStmt) statementNode() {}

type ExpressionStmt struct {
	Expr Expression
}

func (e *ExpressionStmt) String() string { return e.Expr.String() }
func (e *ExpressionStmt) statementNode() {}

type LoopStmt struct {
	Label    int        // Loop label number (e.g., 1 for @1)
	Iterator string     // Variable name (e.g., "i")
	Iterable Expression // Expression to iterate over (e.g., range(10))
	Body     []Statement
}

func (l *LoopStmt) String() string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("@%d ", l.Label))
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

// JumpStmt represents a jump to a loop label (@N)
// @0 = break out to outer scope
// @N = continue/break to loop with label N
type JumpStmt struct {
	Label int // Target label (0 = outer scope, N = loop label)
}

func (j *JumpStmt) String() string {
	return fmt.Sprintf("@%d", j.Label)
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

type IdentExpr struct {
	Name string
}

func (i *IdentExpr) String() string  { return i.Name }
func (i *IdentExpr) expressionNode() {}

type BinaryExpr struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpr) String() string {
	return "(" + b.Left.String() + " " + b.Operator + " " + b.Right.String() + ")"
}
func (b *BinaryExpr) expressionNode() {}

// UnaryExpr represents a unary operation (currently just unary minus)
type UnaryExpr struct {
	Operator string
	Operand  Expression
}

func (u *UnaryExpr) String() string {
	return "(" + u.Operator + u.Operand.String() + ")"
}
func (u *UnaryExpr) expressionNode() {}

type InExpr struct {
	Value     Expression // Value to search for
	Container Expression // List or map to search in
}

func (i *InExpr) String() string {
	return "(" + i.Value.String() + " in " + i.Container.String() + ")"
}
func (i *InExpr) expressionNode() {}

type MatchExpr struct {
	Condition   Expression
	TrueExpr    Expression
	DefaultExpr Expression
}

func (m *MatchExpr) String() string {
	return "~(" + m.Condition.String() + ") { yes -> " + m.TrueExpr.String() + " ~> " + m.DefaultExpr.String() + " }"
}
func (m *MatchExpr) expressionNode() {}

type CallExpr struct {
	Function string
	Args     []Expression
}

func (c *CallExpr) String() string {
	args := make([]string, len(c.Args))
	for i, arg := range c.Args {
		args[i] = arg.String()
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
	return i.List.String() + "[" + i.Index.String() + "]"
}
func (i *IndexExpr) expressionNode() {}

type LambdaExpr struct {
	Params []string
	Body   Expression
}

func (l *LambdaExpr) String() string {
	return "(" + strings.Join(l.Params, ", ") + ") -> " + l.Body.String()
}
func (l *LambdaExpr) expressionNode() {}

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

// Parser for Flap language
type Parser struct {
	lexer     *Lexer
	current   Token
	peek      Token
	filename  string
	source    string
	loopDepth int // Current loop nesting level (0 = not in loop, 1 = outer loop, etc.)
}

func NewParser(input string) *Parser {
	p := &Parser{
		lexer:    NewLexer(input),
		filename: "<input>",
		source:   input,
	}
	p.nextToken()
	p.nextToken()
	return p
}

func NewParserWithFilename(input, filename string) *Parser {
	p := &Parser{
		lexer:    NewLexer(input),
		filename: filename,
		source:   input,
	}
	p.nextToken()
	p.nextToken()
	return p
}

// formatError creates a nicely formatted error message with source context
func (p *Parser) formatError(line int, msg string) string {
	lines := strings.Split(p.source, "\n")
	if line < 1 || line > len(lines) {
		return fmt.Sprintf("%s:%d: %s", p.filename, line, msg)
	}

	sourceLine := lines[line-1]
	lineNum := fmt.Sprintf("%4d | ", line)
	marker := strings.Repeat(" ", len(lineNum)) + strings.Repeat("^", len(sourceLine))

	return fmt.Sprintf("%s:%d: error: %s\n%s%s\n%s",
		p.filename, line, msg, lineNum, sourceLine, marker)
}

// error prints a formatted error and exits
func (p *Parser) error(msg string) {
	fmt.Fprintln(os.Stderr, p.formatError(p.current.Line, msg))
	os.Exit(1)
}

func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) skipNewlines() {
	for p.current.Type == TOKEN_NEWLINE {
		p.nextToken()
	}
}

func (p *Parser) ParseProgram() *Program {
	program := &Program{}

	p.skipNewlines()
	for p.current.Type != TOKEN_EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
		p.skipNewlines()
	}

	// Apply optimizations
	program = optimizeProgram(program)

	return program
}

// optimizeProgram applies optimization passes to the AST
func optimizeProgram(program *Program) *Program {
	// Apply constant folding
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
		e.TrueExpr = foldConstantExpr(e.TrueExpr)
		e.DefaultExpr = foldConstantExpr(e.DefaultExpr)
		return e

	default:
		return expr
	}
}

func (p *Parser) parseStatement() Statement {
	// Check for 'for' keyword (alias for @(N+1) loop)
	if p.current.Type == TOKEN_FOR {
		return p.parseForLoop()
	}

	// Check for 'break' keyword (alias for @(N-1) jump out)
	if p.current.Type == TOKEN_BREAK {
		if p.loopDepth == 0 {
			p.error("'break' used outside of loop")
		}
		p.nextToken() // skip 'break'
		// break = @(N-1) where N is current loop label
		// Since loop labels start at 1, loopDepth-1 gives us the outer scope or previous loop
		return &JumpStmt{Label: p.loopDepth - 1}
	}

	// Check for 'continue' keyword (alias for @N jump to start)
	if p.current.Type == TOKEN_CONTINUE {
		if p.loopDepth == 0 {
			p.error("'continue' used outside of loop")
		}
		p.nextToken() // skip 'continue'
		// continue = @N where N is current loop label
		return &JumpStmt{Label: p.loopDepth}
	}

	// Check for @ (either loop @N or jump @N)
	if p.current.Type == TOKEN_AT {
		// Look ahead to distinguish loop vs jump
		// Loop: @N identifier in ...
		// Jump: @N (followed by newline, semicolon, or })
		if p.peek.Type == TOKEN_NUMBER {
			// We need to peek further to distinguish loop from jump
			// For now, let's just parse as loop if it matches the pattern
			// Otherwise treat as jump

			// Simple heuristic: if @ NUMBER IDENTIFIER, it's a loop
			// We can't easily look 2 tokens ahead, so we'll just try parsing as loop first
			return p.parseLoopStatement()
		}
		p.error("expected number after @ (e.g., @1, @2, @0)")
	}

	// Check for assignment (both = and :=)
	if p.current.Type == TOKEN_IDENT && (p.peek.Type == TOKEN_EQUALS || p.peek.Type == TOKEN_COLON_EQUALS) {
		return p.parseAssignment()
	}

	// Otherwise, it's an expression statement (or match expression)
	expr := p.parseExpression()
	if expr != nil {
		// Check for match syntax: CONDITION { -> ... ~> ... }
		if p.peek.Type == TOKEN_LBRACE {
			p.nextToken() // move to '{'
			p.nextToken() // skip '{'
			p.skipNewlines()

			// Must start with '->' for match expression
			if p.current.Type == TOKEN_ARROW {
				p.nextToken() // skip '->'
				trueExpr := p.parseExpression()
				p.nextToken() // move past the expression
				p.skipNewlines()

				// Parse "~> expr" (optional - defaults to 0)
				var defaultExpr Expression
				if p.current.Type == TOKEN_DEFAULT_ARROW {
					p.nextToken() // skip '~>'
					defaultExpr = p.parseExpression()
					p.nextToken() // move past the expression
					p.skipNewlines()
				} else {
					// Default to 0 if no default case provided
					defaultExpr = &NumberExpr{Value: 0}
				}

				// Should be at '}'
				if p.current.Type != TOKEN_RBRACE {
					p.error("expected '}' after match expression")
				}

				matchExpr := &MatchExpr{Condition: expr, TrueExpr: trueExpr, DefaultExpr: defaultExpr}
				return &ExpressionStmt{Expr: matchExpr}
			}
			// Not a match expression - this is a syntax error
			p.error("unexpected '{' after expression")
		}

		return &ExpressionStmt{Expr: expr}
	}

	return nil
}

func (p *Parser) parseAssignment() *AssignStmt {
	name := p.current.Value
	p.nextToken() // skip identifier
	mutable := p.current.Type == TOKEN_COLON_EQUALS
	p.nextToken() // skip '=' or ':='
	value := p.parseExpression()
	return &AssignStmt{Name: name, Value: value, Mutable: mutable}
}

func (p *Parser) parseLoopStatement() Statement {
	p.nextToken() // skip '@'

	// Expect number for loop label
	if p.current.Type != TOKEN_NUMBER {
		p.error("expected number after @ (e.g., @1, @2, @0)")
	}

	labelNum, err := strconv.ParseFloat(p.current.Value, 64)
	if err != nil {
		p.error("invalid loop label number")
	}
	label := int(labelNum)

	p.nextToken() // skip label number

	// Check if next is identifier (loop) or not (jump)
	if p.current.Type != TOKEN_IDENT {
		// It's a jump statement: @N
		if label < 0 {
			p.error("jump label must be >= 0 (use @0, @1, @2, etc.)")
		}
		return &JumpStmt{Label: label}
	}

	// It's a loop statement
	if label < 1 {
		p.error("loop label must be >= 1 (use @1, @2, @3, etc.)")
	}

	iterator := p.current.Value

	p.nextToken() // skip identifier

	// Expect 'in' keyword
	if p.current.Type != TOKEN_IN {
		p.error("expected 'in' in loop statement")
	}

	p.nextToken() // skip 'in'

	// Parse iterable expression
	iterable := p.parseExpression()

	// Skip newlines before '{'
	for p.peek.Type == TOKEN_NEWLINE {
		p.nextToken()
	}

	// Expect '{'
	if p.peek.Type != TOKEN_LBRACE {
		p.error("expected '{' in loop statement")
	}
	p.nextToken() // advance to '{'

	// Skip newlines after '{'
	for p.peek.Type == TOKEN_NEWLINE {
		p.nextToken()
	}

	// Track loop depth for nested loops (for break/continue)
	oldDepth := p.loopDepth
	p.loopDepth = label
	defer func() { p.loopDepth = oldDepth }()

	// Parse loop body
	var body []Statement
	for p.peek.Type != TOKEN_RBRACE && p.peek.Type != TOKEN_EOF {
		p.nextToken()
		if p.current.Type == TOKEN_NEWLINE {
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	// Skip to '}'
	for p.peek.Type != TOKEN_RBRACE && p.peek.Type != TOKEN_EOF {
		p.nextToken()
	}
	p.nextToken() // skip to '}'

	return &LoopStmt{
		Label:    label,
		Iterator: iterator,
		Iterable: iterable,
		Body:     body,
	}
}

// parseForLoop handles 'for' keyword which is an alias for @(N+1)
// Supports:
// - for x in expr { ... } - normal loop with auto-increment label
// - for x in N { ... } - loop N times (x from 0 to N-1)
// - for { ... } - infinite loop
func (p *Parser) parseForLoop() Statement {
	p.nextToken() // skip 'for'

	// Check for infinite loop: for { ... }
	if p.current.Type == TOKEN_LBRACE {
		// Infinite loop: for { ... }
		// This is equivalent to: @(N+1) _ in range(∞) { ... }
		// We'll implement this as a special case later
		p.error("infinite loops (for { }) not yet implemented")
	}

	// Calculate auto-increment label: @(N+1)
	label := p.loopDepth + 1

	// Expect identifier for iterator variable
	if p.current.Type != TOKEN_IDENT {
		p.error("expected identifier after 'for'")
	}
	iterator := p.current.Value

	p.nextToken() // skip identifier

	// Expect 'in' keyword
	if p.current.Type != TOKEN_IN {
		p.error("expected 'in' in for loop")
	}

	p.nextToken() // skip 'in'

	// Parse iterable expression
	iterable := p.parseExpression()

	// Skip newlines before '{'
	for p.peek.Type == TOKEN_NEWLINE {
		p.nextToken()
	}

	// Expect '{'
	if p.peek.Type != TOKEN_LBRACE {
		p.error("expected '{' in for loop")
	}
	p.nextToken() // advance to '{'

	// Skip newlines after '{'
	for p.peek.Type == TOKEN_NEWLINE {
		p.nextToken()
	}

	// Track loop depth for nested loops (for break/continue)
	oldDepth := p.loopDepth
	p.loopDepth = label
	defer func() { p.loopDepth = oldDepth }()

	// Parse loop body
	var body []Statement
	for p.peek.Type != TOKEN_RBRACE && p.peek.Type != TOKEN_EOF {
		p.nextToken()
		if p.current.Type == TOKEN_NEWLINE {
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
	}

	// Skip to '}'
	for p.peek.Type != TOKEN_RBRACE && p.peek.Type != TOKEN_EOF {
		p.nextToken()
	}
	p.nextToken() // skip to '}'

	return &LoopStmt{
		Label:    label,
		Iterator: iterator,
		Iterable: iterable,
		Body:     body,
	}
}

func (p *Parser) parseExpression() Expression {
	expr := p.parseConcurrentGather()
	return expr
}

func (p *Parser) parseConcurrentGather() Expression {
	left := p.parsePipe()

	for p.peek.Type == TOKEN_PIPEPIPEPIPE {
		p.nextToken() // skip current
		p.nextToken() // skip '|||'
		right := p.parsePipe()
		left = &ConcurrentGatherExpr{Left: left, Right: right}
	}

	return left
}

func (p *Parser) parsePipe() Expression {
	left := p.parseParallel()

	for p.peek.Type == TOKEN_PIPE {
		p.nextToken() // skip current
		p.nextToken() // skip '|'
		right := p.parseParallel()
		left = &PipeExpr{Left: left, Right: right}
	}

	return left
}

func (p *Parser) parseParallel() Expression {
	left := p.parseLogicalOr()

	for p.peek.Type == TOKEN_PIPEPIPE {
		p.nextToken() // skip current
		p.nextToken() // skip '||'
		right := p.parseLogicalOr()
		left = &ParallelExpr{List: left, Operation: right}
	}

	return left
}

func (p *Parser) parseLogicalOr() Expression {
	left := p.parseLogicalAnd()

	for p.peek.Type == TOKEN_OR || p.peek.Type == TOKEN_XOR {
		p.nextToken() // skip current
		op := p.current.Value
		p.nextToken() // skip operator
		right := p.parseLogicalAnd()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseLogicalAnd() Expression {
	left := p.parseComparison()

	for p.peek.Type == TOKEN_AND {
		p.nextToken() // skip current
		op := p.current.Value
		p.nextToken() // skip 'and'
		right := p.parseComparison()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseComparison() Expression {
	left := p.parseAdditive()

	// Check for 'in' operator (membership testing)
	if p.peek.Type == TOKEN_IN {
		p.nextToken() // move to left expr
		p.nextToken() // skip 'in'
		right := p.parseAdditive()
		return &InExpr{Value: left, Container: right}
	}

	for p.peek.Type == TOKEN_LT || p.peek.Type == TOKEN_GT ||
		p.peek.Type == TOKEN_LE || p.peek.Type == TOKEN_GE ||
		p.peek.Type == TOKEN_EQ || p.peek.Type == TOKEN_NE {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseAdditive()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseLambdaBody() Expression {
	// Parse the body expression
	expr := p.parseExpression()

	// Check if it's followed by a match block: { -> ... ~> ... }
	if p.peek.Type == TOKEN_LBRACE {
		p.nextToken() // move to '{'
		p.nextToken() // skip '{'
		p.skipNewlines()

		// Check if it starts with '->' (match expression)
		if p.current.Type == TOKEN_ARROW {
			p.nextToken() // skip '->'
			trueExpr := p.parseExpression()
			p.nextToken() // move past the expression
			p.skipNewlines()

			// Parse "~> expr" (optional - defaults to 0)
			var defaultExpr Expression
			if p.current.Type == TOKEN_DEFAULT_ARROW {
				p.nextToken() // skip '~>'
				defaultExpr = p.parseExpression()
				p.nextToken() // move past the expression
				p.skipNewlines()
			} else {
				// Default to 0 if no default case provided
				defaultExpr = &NumberExpr{Value: 0}
			}

			// Should be at '}'
			if p.current.Type != TOKEN_RBRACE {
				p.error("expected '}' after match expression")
			}

			return &MatchExpr{Condition: expr, TrueExpr: trueExpr, DefaultExpr: defaultExpr}
		}
		// Not a match, backtrack is not possible, this is an error
		p.error("unexpected '{' after lambda body expression")
	}

	return expr
}

func (p *Parser) parseAdditive() Expression {
	left := p.parseBitwise()

	for p.peek.Type == TOKEN_PLUS || p.peek.Type == TOKEN_MINUS {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseBitwise()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseBitwise() Expression {
	left := p.parseMultiplicative()

	for p.peek.Type == TOKEN_SHL || p.peek.Type == TOKEN_SHR ||
		p.peek.Type == TOKEN_ROL || p.peek.Type == TOKEN_ROR {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseMultiplicative()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseMultiplicative() Expression {
	left := p.parseUnary()

	for p.peek.Type == TOKEN_STAR || p.peek.Type == TOKEN_SLASH {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseUnary()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseUnary() Expression {
	// Handle unary operators (not, unary minus)
	if p.current.Type == TOKEN_NOT {
		p.nextToken() // skip 'not'
		operand := p.parseUnary()
		return &UnaryExpr{Operator: "not", Operand: operand}
	}

	// Unary minus handled in parsePrimary for simplicity
	return p.parsePostfix()
}

func (p *Parser) parsePostfix() Expression {
	expr := p.parsePrimary()

	// Handle postfix operations like indexing and function calls
	for {
		if p.peek.Type == TOKEN_LBRACKET {
			p.nextToken() // skip current expr
			p.nextToken() // skip '['
			index := p.parseExpression()
			p.nextToken() // move to ']'
			expr = &IndexExpr{List: expr, Index: index}
		} else if p.peek.Type == TOKEN_LPAREN {
			// Handle direct lambda calls: ((x) -> x * 2)(5)
			// or chained calls: f(1)(2)
			p.nextToken() // skip current expr
			p.nextToken() // skip '('
			args := []Expression{}

			if p.current.Type != TOKEN_RPAREN {
				args = append(args, p.parseExpression())
				for p.peek.Type == TOKEN_COMMA {
					p.nextToken() // skip current
					p.nextToken() // skip ','
					args = append(args, p.parseExpression())
				}
			}

			// current should be on last arg or on '('
			// peek should be ')'
			p.nextToken() // move to ')'

			// Wrap the expression in a CallExpr
			// If expr is a LambdaExpr, it will be compiled and called
			// If expr is an IdentExpr, it will be looked up and called
			if ident, ok := expr.(*IdentExpr); ok {
				expr = &CallExpr{Function: ident.Name, Args: args}
			} else {
				// For lambda expressions or other callable expressions,
				// create a special call expression that compiles the lambda inline
				expr = &DirectCallExpr{Callee: expr, Args: args}
			}
		} else {
			break
		}
	}

	return expr
}

func (p *Parser) parsePrimary() Expression {
	switch p.current.Type {
	case TOKEN_MINUS:
		// Unary minus: -expr
		p.nextToken() // skip '-'
		expr := p.parsePrimary()
		return &UnaryExpr{Operator: "-", Operand: expr}

	case TOKEN_HASH:
		// Length operator: #list
		p.nextToken() // skip '#'
		expr := p.parsePrimary()
		return &LengthExpr{Operand: expr}

	case TOKEN_NUMBER:
		val, _ := strconv.ParseFloat(p.current.Value, 64)
		return &NumberExpr{Value: val}

	case TOKEN_STRING:
		return &StringExpr{Value: p.current.Value}

	case TOKEN_IDENT:
		name := p.current.Value
		// Check for function call
		if p.peek.Type == TOKEN_LPAREN {
			p.nextToken() // skip identifier
			p.nextToken() // skip '('
			args := []Expression{}

			if p.current.Type != TOKEN_RPAREN {
				args = append(args, p.parseExpression())
				for p.peek.Type == TOKEN_COMMA {
					p.nextToken() // skip current
					p.nextToken() // skip ','
					args = append(args, p.parseExpression())
				}
			}

			// current should be on last arg or on '('
			// peek should be ')'
			p.nextToken() // move to ')'
			return &CallExpr{Function: name, Args: args}
		}
		return &IdentExpr{Name: name}

	case TOKEN_LPAREN:
		// Could be lambda (params) -> expr or parenthesized expression (expr)
		p.nextToken() // skip '('

		// Check for empty parameter list: () ->
		if p.current.Type == TOKEN_RPAREN {
			if p.peek.Type == TOKEN_ARROW {
				p.nextToken() // skip ')'
				p.nextToken() // skip '->'
				body := p.parseLambdaBody()
				return &LambdaExpr{Params: []string{}, Body: body}
			}
			// Empty parens without arrow is an error, but skip for now
			p.nextToken()
			return nil
		}

		// Try to parse as parameter list (identifiers separated by commas)
		// or as an expression
		if p.current.Type == TOKEN_IDENT {
			// Peek ahead to determine if it's a lambda or expression
			// If we see: ident ) -> or ident , -> it's a lambda
			firstIdent := p.current.Value

			if p.peek.Type == TOKEN_RPAREN {
				// Could be (x) -> expr or (x)
				p.nextToken() // move to ')'
				if p.peek.Type == TOKEN_ARROW {
					// It's a lambda: (x) -> expr
					p.nextToken() // skip ')'
					p.nextToken() // skip '->'
					body := p.parseLambdaBody()
					return &LambdaExpr{Params: []string{firstIdent}, Body: body}
				}
				// It's (x) parenthesized identifier
				p.nextToken() // skip ')'
				return &IdentExpr{Name: firstIdent}
			}

			if p.peek.Type == TOKEN_COMMA {
				// Definitely a lambda with multiple params: (x, y, ...) -> expr
				params := []string{firstIdent}
				p.nextToken() // skip first ident

				for p.current.Type == TOKEN_COMMA {
					p.nextToken() // skip ','
					if p.current.Type != TOKEN_IDENT {
						p.error("expected parameter name in lambda")
					}
					params = append(params, p.current.Value)
					p.nextToken() // skip param
				}

				// current should be ')'
				if p.current.Type != TOKEN_RPAREN {
					p.error("expected ')' after lambda parameters")
				}

				// peek should be '->'
				if p.peek.Type != TOKEN_ARROW {
					p.error("expected '->' after lambda parameters")
				}

				p.nextToken() // skip ')'
				p.nextToken() // skip '->'
				body := p.parseLambdaBody()
				return &LambdaExpr{Params: params, Body: body}
			}
		}

		// Not a lambda, parse as parenthesized expression
		expr := p.parseExpression()
		p.nextToken() // skip ')'
		return expr

	case TOKEN_LBRACKET:
		p.nextToken() // skip '['
		elements := []Expression{}

		if p.current.Type != TOKEN_RBRACKET {
			elements = append(elements, p.parseExpression())
			for p.peek.Type == TOKEN_COMMA {
				p.nextToken() // skip current
				p.nextToken() // skip ','
				elements = append(elements, p.parseExpression())
			}
		}

		// current should be on last element or on '['
		// peek should be ']'
		p.nextToken() // move to ']'
		return &ListExpr{Elements: elements}

	case TOKEN_LBRACE:
		// Map literal: {key: value, key2: value2, ...}
		p.nextToken() // skip '{'
		keys := []Expression{}
		values := []Expression{}

		if p.current.Type != TOKEN_RBRACE {
			// Parse first key
			key := p.parseExpression()
			p.nextToken() // move past key

			// Must have ':'
			if p.current.Type != TOKEN_COLON {
				p.error("expected ':' in map literal")
			}
			p.nextToken() // skip ':'

			// Parse value
			value := p.parseExpression()
			keys = append(keys, key)
			values = append(values, value)

			// Parse additional key:value pairs
			for p.peek.Type == TOKEN_COMMA {
				p.nextToken() // skip current value
				p.nextToken() // skip ','

				key := p.parseExpression()
				p.nextToken() // move past key

				if p.current.Type != TOKEN_COLON {
					p.error("expected ':' in map literal")
				}
				p.nextToken() // skip ':'

				value := p.parseExpression()
				keys = append(keys, key)
				values = append(values, value)
			}
		}

		// current should be on last value or on '{'
		// peek should be '}'
		p.nextToken() // move to '}'
		return &MapExpr{Keys: keys, Values: values}
	}

	return nil
}

// LoopInfo tracks information about an active loop during compilation
type LoopInfo struct {
	Label      int   // Loop label (@1, @2, @3, etc.)
	StartPos   int   // Code position of loop start (for continue/jump back)
	EndPatches []int // Positions that need to be patched to jump to loop end
}

// Code Generator for Flap
type FlapCompiler struct {
	eb               *ExecutableBuilder
	out              *Out
	variables        map[string]int    // variable name -> stack offset
	mutableVars      map[string]bool   // variable name -> is mutable
	varTypes         map[string]string // variable name -> "map" or "list"
	sourceCode       string            // Store source for recompilation
	usedFunctions    map[string]bool   // Track which functions are called
	unknownFunctions map[string]bool   // Track functions called but not defined
	callOrder        []string          // Track order of function calls
	stringCounter    int               // Counter for unique string labels
	stackOffset      int               // Current stack offset for variables
	labelCounter     int               // Counter for unique labels (if/else, loops, etc)
	lambdaCounter    int               // Counter for unique lambda function names
	activeLoops      []LoopInfo        // Stack of active loops (for @N jump resolution)
	lambdaFuncs      []LambdaFunc      // List of lambda functions to generate
	lambdaOffsets    map[string]int    // Lambda name -> offset in .text
	currentLambda    *LambdaFunc       // Currently compiling lambda (for "me" self-reference)
	lambdaBodyStart  int               // Offset where lambda body starts (for tail recursion)
}

type LambdaFunc struct {
	Name   string
	Params []string
	Body   Expression
}

func NewFlapCompiler(machine Machine) (*FlapCompiler, error) {
	// Create ExecutableBuilder
	eb, err := New(machine.String())
	if err != nil {
		return nil, err
	}

	// Enable dynamic linking
	eb.useDynamicLinking = true
	// Don't set neededFunctions yet - we'll build it dynamically

	// Create Out wrapper
	out := &Out{
		machine: eb.machine,
		writer:  eb.TextWriter(),
		eb:      eb,
	}

	return &FlapCompiler{
		eb:               eb,
		out:              out,
		variables:        make(map[string]int),
		mutableVars:      make(map[string]bool),
		varTypes:         make(map[string]string),
		usedFunctions:    make(map[string]bool),
		unknownFunctions: make(map[string]bool),
		callOrder:        []string{},
		lambdaOffsets:    make(map[string]int),
	}, nil
}

func (fc *FlapCompiler) Compile(program *Program, outputPath string) error {
	// Add format strings for printf
	fc.eb.Define("fmt_str", "%s\x00")
	fc.eb.Define("fmt_int", "%ld\n\x00")
	fc.eb.Define("fmt_float", "%.0f\n\x00") // Print float without decimal places

	// Generate code
	// Set up stack frame
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.SubImmFromReg("rsp", 8) // Align stack to 16 bytes (push rbp made it 8-byte aligned)

	// Initialize registers
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rdi", "rdi")
	fc.out.XorRegWithReg("rsi", "rsi")

	// ===== AVX-512 CPU DETECTION =====
	// Check CPUID for AVX-512 support and store result
	// Required for safe use of AVX-512 instructions in map lookups
	fc.eb.Define("cpu_has_avx512", "\x00") // 1 byte: 0=no, 1=yes

	// Check CPUID leaf 7, subleaf 0, EBX bit 16 (AVX512F)
	fc.out.MovImmToReg("rax", "7")     // CPUID leaf 7
	fc.out.XorRegWithReg("rcx", "rcx") // subleaf 0
	fc.out.Emit([]byte{0x0f, 0xa2})    // cpuid

	// Test EBX bit 16 (AVX512F - foundation)
	fc.out.Emit([]byte{0xf6, 0xc3, 0x01}) // test bl, 1 (bit 0 after shift)
	// Actually test bit 16 of ebx: bt ebx, 16
	fc.out.Emit([]byte{0x0f, 0xba, 0xe3, 0x10}) // bt ebx, 16

	// Set carry flag if supported
	// setc al (set AL to 1 if carry flag set)
	fc.out.Emit([]byte{0x0f, 0x92, 0xc0}) // setc al

	// Store result to cpu_has_avx512 (only write AL, not full RAX!)
	fc.out.LeaSymbolToReg("rbx", "cpu_has_avx512")
	fc.out.MovByteRegToMem("rax", "rbx", 0) // Write only the low byte (AL)

	// Clear registers used for CPUID
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rbx", "rbx")
	fc.out.XorRegWithReg("rcx", "rcx")
	// ===== END AVX-512 DETECTION =====

	// Two-pass compilation: First pass collects all variable declarations
	// so that function/constant order doesn't matter
	for _, stmt := range program.Statements {
		fc.collectSymbols(stmt)
	}

	// Second pass: Generate actual code with all symbols known
	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

	// Automatically call exit(0) at program end
	fc.out.XorRegWithReg("rdi", "rdi") // exit code 0
	fc.trackFunctionCall("exit")
	fc.eb.GenerateCallInstruction("exit")

	// Generate lambda functions
	fc.generateLambdaFunctions()

	// Generate runtime helper functions
	fc.generateRuntimeHelpers()

	// Write ELF using existing infrastructure
	return fc.writeELF(outputPath)
}

func (fc *FlapCompiler) writeELF(outputPath string) error {
	// WORKAROUND: Always use printf and exit in fixed order for PLT
	// to maintain consistent PLT size and avoid _start jump offset bugs
	// We'll generate the correct calls based on callOrder
	// All trig functions use x87 hardware instructions:
	// sin/cos/tan: FSIN, FCOS, FPTAN
	// atan: FPATAN, asin/acos: FPATAN + Fsqrt + x87 arithmetic
	// malloc is needed for runtime string concatenation
	// Always include malloc since runtime helpers (_flap_string_concat) need it
	pltFunctions := []string{"printf", "exit", "malloc"}

	// Build mapping from actual calls to PLT indices
	callToPLT := make(map[string]int)
	for i, f := range pltFunctions {
		callToPLT[f] = i
	}

	// Set up dynamic sections
	ds := NewDynamicSections()
	ds.AddNeeded("libc.so.6")
	// Note: libm.so.6 not needed - all math functions use x87 FPU instructions

	// Add symbols for PLT functions
	for _, funcName := range pltFunctions {
		ds.AddSymbol(funcName, STB_GLOBAL, STT_FUNC)
	}

	// Write rodata - get symbols and sort for consistent ordering
	rodataSymbols := fc.eb.RodataSection()

	// Create sorted list of symbol names for deterministic ordering
	var symbolNames []string
	for name := range rodataSymbols {
		symbolNames = append(symbolNames, name)
	}
	sort.Strings(symbolNames)

	// DEBUG: Print what symbols we're writing
	fmt.Fprintf(os.Stderr, "DEBUG: rodataSymbols contains %d symbols: %v\n", len(symbolNames), symbolNames)

	estimatedRodataAddr := uint64(0x403000 + 0x100)
	currentAddr := estimatedRodataAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]
		fc.eb.WriteRodata([]byte(value))
		fc.eb.DefineAddr(symbol, currentAddr)
		fmt.Fprintf(os.Stderr, "DEBUG: Writing %s at 0x%x, len=%d\n", symbol, currentAddr, len(value))
		currentAddr += uint64(len(value))
	}

	// Write complete dynamic ELF with unique PLT functions
	// Note: We pass pltFunctions (unique) for building PLT/GOT structure
	// We'll use fc.callOrder (with duplicates) later for patching actual call sites
	fmt.Fprintf(os.Stderr, "\n=== First compilation callOrder: %v ===\n", fc.callOrder)
	fmt.Fprintf(os.Stderr, "=== pltFunctions (unique): %v ===\n", pltFunctions)
	gotBase, rodataBaseAddr, textAddr, pltBase, err := fc.eb.WriteCompleteDynamicELF(ds, pltFunctions)
	if err != nil {
		return err
	}

	// Update rodata addresses using same sorted order
	currentAddr = rodataBaseAddr
	fmt.Fprintf(os.Stderr, "DEBUG: Updating addresses with actual rodata base=0x%x\n", rodataBaseAddr)
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]
		fc.eb.DefineAddr(symbol, currentAddr)
		fmt.Fprintf(os.Stderr, "DEBUG: Updated %s to 0x%x, len=%d\n", symbol, currentAddr, len(value))
		currentAddr += uint64(len(value))
	}

	// Regenerate code with correct addresses
	fc.eb.text.Reset()
	fc.eb.pcRelocations = []PCRelocation{}  // Reset PC relocations for recompilation
	fc.eb.callPatches = []CallPatch{}       // Reset call patches for recompilation
	fc.eb.labels = make(map[string]int)     // Reset labels for recompilation
	fc.callOrder = []string{}               // Clear call order for recompilation
	fc.stringCounter = 0                    // Reset string counter for recompilation
	fc.labelCounter = 0                     // Reset label counter for recompilation
	fc.lambdaCounter = 0                    // Reset lambda counter for recompilation
	fc.lambdaFuncs = []LambdaFunc{}         // Clear lambda functions list
	fc.lambdaOffsets = make(map[string]int) // Reset lambda offsets
	fc.variables = make(map[string]int)     // Reset variables map
	fc.mutableVars = make(map[string]bool)  // Reset mutability tracking
	fc.stackOffset = 0                      // Reset stack offset
	// Set up stack frame
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.SubImmFromReg("rsp", 8) // Align stack to 16 bytes
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rdi", "rdi")
	fc.out.XorRegWithReg("rsi", "rsi")

	// Re-define format strings for printf
	fc.eb.Define("fmt_str", "%s\x00")
	fc.eb.Define("fmt_int", "%ld\n\x00")
	fc.eb.Define("fmt_float", "%.0f\n\x00")

	// ===== AVX-512 CPU DETECTION (regenerated) =====
	fc.eb.Define("cpu_has_avx512", "\x00")      // 1 byte: 0=no, 1=yes
	fc.out.MovImmToReg("rax", "7")              // CPUID leaf 7
	fc.out.XorRegWithReg("rcx", "rcx")          // subleaf 0
	fc.out.Emit([]byte{0x0f, 0xa2})             // cpuid
	fc.out.Emit([]byte{0xf6, 0xc3, 0x01})       // test bl, 1
	fc.out.Emit([]byte{0x0f, 0xba, 0xe3, 0x10}) // bt ebx, 16
	fc.out.Emit([]byte{0x0f, 0x92, 0xc0})       // setc al
	fc.out.LeaSymbolToReg("rbx", "cpu_has_avx512")
	fc.out.MovByteRegToMem("rax", "rbx", 0) // Write only AL, not full RAX
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rbx", "rbx")
	fc.out.XorRegWithReg("rcx", "rcx")
	// ===== END AVX-512 DETECTION =====

	// Recompile with correct addresses
	parser := NewParser(fc.sourceCode)
	program := parser.ParseProgram()
	// Collect symbols again (two-pass compilation for second regeneration)
	for _, stmt := range program.Statements {
		fc.collectSymbols(stmt)
	}
	// Generate code with symbols collected
	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

	// Automatically call exit(0) at program end
	fc.out.XorRegWithReg("rdi", "rdi") // exit code 0
	fc.trackFunctionCall("exit")
	fc.eb.GenerateCallInstruction("exit")

	// Generate lambda functions
	fc.generateLambdaFunctions()

	// Generate runtime helper functions
	fc.generateRuntimeHelpers()

	// Collect rodata symbols again (lambda/runtime functions may have created new ones)
	rodataSymbols = fc.eb.RodataSection()

	// Find any NEW symbols that weren't in the original list
	var newSymbols []string
	for symbol := range rodataSymbols {
		found := false
		for _, existingSym := range symbolNames {
			if symbol == existingSym {
				found = true
				break
			}
		}
		if !found {
			newSymbols = append(newSymbols, symbol)
		}
	}

	if len(newSymbols) > 0 {
		fmt.Fprintf(os.Stderr, "DEBUG: Found %d new rodata symbols after lambda generation: %v\n", len(newSymbols), newSymbols)
		sort.Strings(newSymbols)

		// Append new symbols to rodata and assign addresses
		for _, symbol := range newSymbols {
			value := rodataSymbols[symbol]
			fc.eb.WriteRodata([]byte(value))
			fc.eb.DefineAddr(symbol, currentAddr)
			fmt.Fprintf(os.Stderr, "DEBUG: Added new symbol %s at 0x%x, len=%d\n", symbol, currentAddr, len(value))
			currentAddr += uint64(len(value))
			symbolNames = append(symbolNames, symbol)
		}
	}

	// Set lambda function addresses
	for lambdaName, offset := range fc.lambdaOffsets {
		lambdaAddr := textAddr + uint64(offset)
		fc.eb.DefineAddr(lambdaName, lambdaAddr)
	}

	// Patch PLT calls using callOrder (actual sequence of calls)
	// patchPLTCalls will look up each function name in the PLT to get its offset
	// This handles duplicate calls (e.g., two calls to exit) correctly
	fmt.Fprintf(os.Stderr, "\n=== Second compilation callOrder: %v ===\n", fc.callOrder)
	fc.eb.patchPLTCalls(ds, textAddr, pltBase, fc.callOrder)

	// Patch PC-relative relocations
	rodataSize := fc.eb.rodata.Len()
	fc.eb.PatchPCRelocations(textAddr, rodataBaseAddr, rodataSize)

	// Patch function calls in regenerated code
	fmt.Fprintf(os.Stderr, "\n=== Patching function calls (regenerated code) ===\n")
	fc.eb.PatchCallSites(textAddr)

	// Update ELF with regenerated code
	fc.eb.patchTextInELF()

	// Output the executable file
	if err := os.WriteFile(outputPath, fc.eb.Bytes(), 0o755); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Final GOT base: 0x%x\n", gotBase)
	return nil
}

// collectSymbols performs the first pass: collect all variable declarations
// without generating any code. This allows forward references.
func (fc *FlapCompiler) collectSymbols(stmt Statement) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Check if variable already exists
		_, exists := fc.variables[s.Name]
		if !exists {
			// First assignment - allocate stack space (16 bytes for alignment)
			fc.stackOffset += 16
			offset := fc.stackOffset
			fc.variables[s.Name] = offset
			fc.mutableVars[s.Name] = s.Mutable

			// Track type if we can determine it from the expression
			exprType := fc.getExprType(s.Value)
			if exprType != "number" && exprType != "unknown" {
				fc.varTypes[s.Name] = exprType
			}
		}
	case *LoopStmt:
		// Recursively collect symbols from loop body
		for _, bodyStmt := range s.Body {
			fc.collectSymbols(bodyStmt)
		}
	case *ExpressionStmt:
		// No symbols to collect from expression statements
	}
}

func (fc *FlapCompiler) compileStatement(stmt Statement) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Variable already registered in collectSymbols pass
		offset := fc.variables[s.Name]

		// Check mutability on reassignment
		if s.Mutable && !fc.mutableVars[s.Name] {
			fmt.Fprintf(os.Stderr, "Error: cannot reassign immutable variable '%s'\n", s.Name)
			os.Exit(1)
		}

		// Allocate actual stack space (was only registered in first pass)
		fc.out.SubImmFromReg("rsp", 16)

		// Evaluate expression into xmm0
		fc.compileExpression(s.Value)
		// Store xmm0 to stack at variable's offset
		fc.out.MovXmmToMem("xmm0", "rbp", -offset)

	case *LoopStmt:
		fc.compileLoopStatement(s)

	case *JumpStmt:
		fc.compileJumpStatement(s)

	case *ExpressionStmt:
		fc.compileExpression(s.Expr)
	}
}

func (fc *FlapCompiler) compileLoopStatement(stmt *LoopStmt) {
	// Check if iterating over range() or a list
	funcCall, isRangeCall := stmt.Iterable.(*CallExpr)
	if isRangeCall && funcCall.Function == "range" && len(funcCall.Args) == 1 {
		// Range loop
		fc.compileRangeLoop(stmt, funcCall)
	} else {
		// List iteration
		fc.compileListLoop(stmt)
	}
}

func (fc *FlapCompiler) compileRangeLoop(stmt *LoopStmt, funcCall *CallExpr) {
	// Increment label counter for uniqueness
	fc.labelCounter++

	// Evaluate the loop limit (argument to range())
	fc.compileExpression(funcCall.Args[0])

	// Convert to integer and store in a temp variable for the limit
	// cvttsd2si rax, xmm0
	fc.out.Cvttsd2si("rax", "xmm0")

	// Allocate stack space for loop limit (16 bytes for alignment)
	fc.stackOffset += 16
	limitOffset := fc.stackOffset
	fc.out.SubImmFromReg("rsp", 16)

	// Store limit: mov [rbp - limitOffset], rax
	fc.out.MovRegToMem("rax", "rbp", -limitOffset)

	// Allocate stack space for iterator variable (16 bytes for alignment)
	fc.stackOffset += 16
	iterOffset := fc.stackOffset
	fc.variables[stmt.Iterator] = iterOffset
	fc.mutableVars[stmt.Iterator] = true
	fc.out.SubImmFromReg("rsp", 16)

	// Initialize iterator to 0.0
	// xor rax, rax
	fc.out.XorRegWithReg("rax", "rax")
	// cvtsi2sd xmm0, rax (convert 0 to float64)
	fc.out.Cvtsi2sd("xmm0", "rax")
	// movsd [rbp - iterOffset], xmm0
	fc.out.MovXmmToMem("xmm0", "rbp", -iterOffset)

	// Loop start label
	loopStartPos := fc.eb.text.Len()

	// Register this loop on the active loop stack
	loopInfo := LoopInfo{
		Label:      stmt.Label,
		StartPos:   loopStartPos,
		EndPatches: []int{},
	}
	fc.activeLoops = append(fc.activeLoops, loopInfo)

	// Load iterator value as float: movsd xmm0, [rbp - iterOffset]
	fc.out.MovMemToXmm("xmm0", "rbp", -iterOffset)

	// Convert iterator to integer for comparison: cvttsd2si rax, xmm0
	fc.out.Cvttsd2si("rax", "xmm0")

	// Load limit value: mov rdi, [rbp - limitOffset]
	fc.out.MovMemToReg("rdi", "rbp", -limitOffset)

	// Compare iterator with limit: cmp rax, rdi
	fc.out.CmpRegToReg("rax", "rdi")

	// Jump to loop end if iterator >= limit
	loopEndJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0) // Placeholder

	// Add this to the loop's end patches
	fc.activeLoops[len(fc.activeLoops)-1].EndPatches = append(
		fc.activeLoops[len(fc.activeLoops)-1].EndPatches,
		loopEndJumpPos+2, // +2 to skip to the offset field
	)

	// Compile loop body
	for _, s := range stmt.Body {
		fc.compileStatement(s)
	}

	// Increment iterator (add 1.0 to float64 value)
	// movsd xmm0, [rbp - iterOffset]
	fc.out.MovMemToXmm("xmm0", "rbp", -iterOffset)
	// Load 1.0 into xmm1
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.IncReg("rax") // rax = 1
	fc.out.Cvtsi2sd("xmm1", "rax")
	// addsd xmm0, xmm1 (scalar addition)
	fc.out.AddsdXmm("xmm0", "xmm1")
	// movsd [rbp - iterOffset], xmm0
	fc.out.MovXmmToMem("xmm0", "rbp", -iterOffset)

	// Jump back to loop start
	loopBackJumpPos := fc.eb.text.Len()
	backOffset := int32(loopStartPos - (loopBackJumpPos + 5)) // 5 bytes for unconditional jump
	fc.out.JumpUnconditional(backOffset)

	// Loop end label
	loopEndPos := fc.eb.text.Len()

	// Patch all end jumps (conditional jump + any @0 breaks)
	for _, patchPos := range fc.activeLoops[len(fc.activeLoops)-1].EndPatches {
		endOffset := int32(loopEndPos - (patchPos + 4)) // 4 bytes for 32-bit offset
		fc.patchJumpImmediate(patchPos, endOffset)
	}

	// Pop loop from active stack
	fc.activeLoops = fc.activeLoops[:len(fc.activeLoops)-1]
}

func (fc *FlapCompiler) compileListLoop(stmt *LoopStmt) {
	// Increment label counter for uniqueness
	fc.labelCounter++

	// Evaluate the list expression (returns pointer as float64 in xmm0)
	fc.compileExpression(stmt.Iterable)

	// Save list pointer to stack (16 bytes for alignment)
	fc.stackOffset += 16
	listPtrOffset := fc.stackOffset
	fc.out.SubImmFromReg("rsp", 16)
	fc.out.MovXmmToMem("xmm0", "rbp", -listPtrOffset)

	// Convert pointer from float64 to integer in rax
	fc.out.MovMemToXmm("xmm1", "rbp", -listPtrOffset)
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovXmmToMem("xmm1", "rsp", 0)
	fc.out.MovMemToReg("rax", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Load list length from [rax] (first 8 bytes)
	fc.out.MovMemToXmm("xmm0", "rax", 0)

	// Convert length to integer
	fc.out.Cvttsd2si("rax", "xmm0")

	// Store length in stack
	fc.stackOffset += 16
	lengthOffset := fc.stackOffset
	fc.out.SubImmFromReg("rsp", 16)
	fc.out.MovRegToMem("rax", "rbp", -lengthOffset)

	// Allocate stack space for index variable
	fc.stackOffset += 16
	indexOffset := fc.stackOffset
	fc.out.SubImmFromReg("rsp", 16)

	// Initialize index to 0
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.MovRegToMem("rax", "rbp", -indexOffset)

	// Allocate stack space for iterator variable (the actual value from the list)
	fc.stackOffset += 16
	iterOffset := fc.stackOffset
	fc.variables[stmt.Iterator] = iterOffset
	fc.mutableVars[stmt.Iterator] = true
	fc.out.SubImmFromReg("rsp", 16)

	// Loop start label
	loopStartPos := fc.eb.text.Len()

	// Register this loop on the active loop stack
	loopInfo := LoopInfo{
		Label:      stmt.Label,
		StartPos:   loopStartPos,
		EndPatches: []int{},
	}
	fc.activeLoops = append(fc.activeLoops, loopInfo)

	// Load index: mov rax, [rbp - indexOffset]
	fc.out.MovMemToReg("rax", "rbp", -indexOffset)

	// Load length: mov rdi, [rbp - lengthOffset]
	fc.out.MovMemToReg("rdi", "rbp", -lengthOffset)

	// Compare index with length: cmp rax, rdi
	fc.out.CmpRegToReg("rax", "rdi")

	// Jump to loop end if index >= length
	loopEndJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0) // Placeholder

	// Add this to the loop's end patches
	fc.activeLoops[len(fc.activeLoops)-1].EndPatches = append(
		fc.activeLoops[len(fc.activeLoops)-1].EndPatches,
		loopEndJumpPos+2, // +2 to skip to the offset field
	)

	// Load list pointer from stack to rbx
	fc.out.MovMemToXmm("xmm1", "rbp", -listPtrOffset)
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovXmmToMem("xmm1", "rsp", 0)
	fc.out.MovMemToReg("rbx", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Skip length prefix: rbx += 8
	fc.out.AddImmToReg("rbx", 8)

	// Load index into rax
	fc.out.MovMemToReg("rax", "rbp", -indexOffset)

	// Calculate offset: rax * 8
	fc.out.MulRegWithImm("rax", 8)

	// Add offset to base: rbx = rbx + rax
	fc.out.AddRegToReg("rbx", "rax")

	// Load element from list: movsd xmm0, [rbx]
	fc.out.MovMemToXmm("xmm0", "rbx", 0)

	// Store in iterator variable
	fc.out.MovXmmToMem("xmm0", "rbp", -iterOffset)

	// Compile loop body
	for _, s := range stmt.Body {
		fc.compileStatement(s)
	}

	// Increment index
	fc.out.MovMemToReg("rax", "rbp", -indexOffset)
	fc.out.IncReg("rax")
	fc.out.MovRegToMem("rax", "rbp", -indexOffset)

	// Jump back to loop start
	loopBackJumpPos := fc.eb.text.Len()
	backOffset := int32(loopStartPos - (loopBackJumpPos + 5)) // 5 bytes for unconditional jump
	fc.out.JumpUnconditional(backOffset)

	// Loop end label
	loopEndPos := fc.eb.text.Len()

	// Patch all end jumps (conditional jump + any @0 breaks)
	for _, patchPos := range fc.activeLoops[len(fc.activeLoops)-1].EndPatches {
		endOffset := int32(loopEndPos - (patchPos + 4)) // 4 bytes for 32-bit offset
		fc.patchJumpImmediate(patchPos, endOffset)
	}

	// Pop loop from active stack
	fc.activeLoops = fc.activeLoops[:len(fc.activeLoops)-1]
}

func (fc *FlapCompiler) compileJumpStatement(stmt *JumpStmt) {
	if stmt.Label == 0 {
		// @0 = break to outer scope (jump to end of innermost loop)
		if len(fc.activeLoops) == 0 {
			fmt.Fprintf(os.Stderr, "Error: @0 used outside of loop\n")
			os.Exit(1)
		}
		// Add this jump position to the innermost loop's end patches
		jumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Placeholder
		fc.activeLoops[len(fc.activeLoops)-1].EndPatches = append(
			fc.activeLoops[len(fc.activeLoops)-1].EndPatches,
			jumpPos+1, // +1 to skip the opcode byte
		)
	} else {
		// @N = jump back to loop N (continue)
		// Find the loop with matching label
		var targetLoop *LoopInfo
		for i := len(fc.activeLoops) - 1; i >= 0; i-- {
			if fc.activeLoops[i].Label == stmt.Label {
				targetLoop = &fc.activeLoops[i]
				break
			}
		}
		if targetLoop == nil {
			fmt.Fprintf(os.Stderr, "Error: @%d references undefined loop label\n", stmt.Label)
			os.Exit(1)
		}
		// Jump back to loop start
		jumpPos := fc.eb.text.Len()
		backOffset := int32(targetLoop.StartPos - (jumpPos + 5)) // 5 bytes for unconditional jump
		fc.out.JumpUnconditional(backOffset)
	}
}

func (fc *FlapCompiler) patchJumpImmediate(pos int, offset int32) {
	// Get the current bytes from buffer
	// This is safe because we're patching backwards into already-written code
	bytes := fc.eb.text.Bytes()

	fmt.Fprintf(os.Stderr, "DEBUG PATCH: Before patching at pos %d: %02x %02x %02x %02x\n", pos, bytes[pos], bytes[pos+1], bytes[pos+2], bytes[pos+3])

	// Write 32-bit little-endian offset at position
	bytes[pos] = byte(offset)
	bytes[pos+1] = byte(offset >> 8)
	bytes[pos+2] = byte(offset >> 16)
	bytes[pos+3] = byte(offset >> 24)

	fmt.Fprintf(os.Stderr, "DEBUG PATCH: After patching at pos %d: %02x %02x %02x %02x (offset=%d)\n", pos, bytes[pos], bytes[pos+1], bytes[pos+2], bytes[pos+3], offset)
}

// getExprType returns the type of an expression at compile time
// Returns: "string", "number", "list", "map", or "unknown"
func (fc *FlapCompiler) getExprType(expr Expression) string {
	switch e := expr.(type) {
	case *StringExpr:
		return "string"
	case *NumberExpr:
		return "number"
	case *ListExpr:
		return "list"
	case *MapExpr:
		return "map"
	case *IdentExpr:
		// Look up in varTypes
		if typ, exists := fc.varTypes[e.Name]; exists {
			return typ
		}
		// Default to number if not tracked (most variables are numbers)
		return "number"
	case *BinaryExpr:
		// Binary expressions between strings return strings if operator is "+"
		if e.Operator == "+" {
			leftType := fc.getExprType(e.Left)
			rightType := fc.getExprType(e.Right)
			if leftType == "string" && rightType == "string" {
				return "string"
			}
		}
		return "number"
	default:
		return "unknown"
	}
}

func (fc *FlapCompiler) compileExpression(expr Expression) {
	switch e := expr.(type) {
	case *NumberExpr:
		// Flap uses float64 foundation - all values are float64
		// For whole numbers, use integer conversion; for decimals, load from .rodata
		if e.Value == float64(int64(e.Value)) {
			// Whole number - can use integer path
			val := int64(e.Value)
			fc.out.MovImmToReg("rax", strconv.FormatInt(val, 10))
			fc.out.Cvtsi2sd("xmm0", "rax")
		} else {
			// Decimal number - store in .rodata and load
			labelName := fmt.Sprintf("float_%d", fc.stringCounter)
			fc.stringCounter++

			// Convert float64 to 8 bytes (little-endian)
			bits := uint64(0)
			*(*float64)(unsafe.Pointer(&bits)) = e.Value
			var floatData []byte
			for i := 0; i < 8; i++ {
				floatData = append(floatData, byte((bits>>(i*8))&0xFF))
			}
			fc.eb.Define(labelName, string(floatData))

			// Load from .rodata
			fc.out.LeaSymbolToReg("rax", labelName)
			fc.out.MovMemToXmm("xmm0", "rax", 0)
		}

	case *StringExpr:
		// Strings are represented as map[uint64]float64 where keys are indices
		// and values are character codes
		// Map format: [count][key0][val0][key1][val1]...
		// Following Lisp philosophy: even empty strings are objects (count=0), not null

		labelName := fmt.Sprintf("str_%d", fc.stringCounter)
		fc.stringCounter++

		// Build map data: count followed by key-value pairs
		var mapData []byte

		// Count (number of characters) - can be 0 for empty strings
		count := float64(len(e.Value))
		countBits := uint64(0)
		*(*float64)(unsafe.Pointer(&countBits)) = count
		for i := 0; i < 8; i++ {
			mapData = append(mapData, byte((countBits>>(i*8))&0xFF))
		}

		// Add each character as a key-value pair (none for empty strings)
		for idx, ch := range e.Value {
			// Key: character index as float64
			keyVal := float64(idx)
			keyBits := uint64(0)
			*(*float64)(unsafe.Pointer(&keyBits)) = keyVal
			for i := 0; i < 8; i++ {
				mapData = append(mapData, byte((keyBits>>(i*8))&0xFF))
			}

			// Value: character code as float64
			charVal := float64(ch)
			charBits := uint64(0)
			*(*float64)(unsafe.Pointer(&charBits)) = charVal
			for i := 0; i < 8; i++ {
				mapData = append(mapData, byte((charBits>>(i*8))&0xFF))
			}
		}

		fc.eb.Define(labelName, string(mapData))
		fc.out.LeaSymbolToReg("rax", labelName)
		// Convert pointer to float64
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case *IdentExpr:
		// Load variable from stack into xmm0
		offset, exists := fc.variables[e.Name]
		if !exists {
			fmt.Fprintf(os.Stderr, "Error: undefined variable '%s' at line %d\n", e.Name, 0)
			os.Exit(1)
		}
		// movsd xmm0, [rbp - offset]
		fc.out.MovMemToXmm("xmm0", "rbp", -offset)

	case *UnaryExpr:
		// Compile the operand first (result in xmm0)
		fc.compileExpression(e.Operand)

		switch e.Operator {
		case "-":
			// Unary minus: negate the value
			// Create -1.0 constant and multiply
			labelName := fmt.Sprintf("negone_%d", fc.stringCounter)
			fc.stringCounter++

			// Store -1.0 as float64 bytes
			negOne := -1.0
			bits := uint64(0)
			*(*float64)(unsafe.Pointer(&bits)) = negOne
			var floatData []byte
			for i := 0; i < 8; i++ {
				floatData = append(floatData, byte((bits>>(i*8))&0xFF))
			}
			fc.eb.Define(labelName, string(floatData))

			// Load -1.0 into xmm1 and multiply
			fc.out.LeaSymbolToReg("rax", labelName)
			fc.out.MovMemToXmm("xmm1", "rax", 0)
			fc.out.MulsdXmm("xmm0", "xmm1") // xmm0 = xmm0 * -1.0
		case "not":
			// Logical NOT: returns 1.0 if operand is 0.0, else 0.0
			// Compare xmm0 with 0
			fc.out.XorpdXmm("xmm1", "xmm1") // xmm1 = 0.0
			fc.out.Ucomisd("xmm0", "xmm1")
			// Set rax to 1 if xmm0 == 0, else 0
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			fc.out.Cmove("rax", "rcx") // rax = (xmm0 == 0) ? 1 : 0
			// Convert to float64
			fc.out.Cvtsi2sd("xmm0", "rax")
		}

	case *BinaryExpr:
		// Check for string/list/map operations with + operator
		if e.Operator == "+" {
			leftType := fc.getExprType(e.Left)
			rightType := fc.getExprType(e.Right)

			if leftType == "string" && rightType == "string" {
				// String concatenation (strings are maps, so merge with offset keys)
				leftStr, leftIsLiteral := e.Left.(*StringExpr)
				rightStr, rightIsLiteral := e.Right.(*StringExpr)

				if leftIsLiteral && rightIsLiteral {
					// Compile-time concatenation - just create new string map
					result := leftStr.Value + rightStr.Value

					// Build concatenated string map
					labelName := fmt.Sprintf("str_%d", fc.stringCounter)
					fc.stringCounter++

					var mapData []byte
					count := float64(len(result))
					countBits := uint64(0)
					*(*float64)(unsafe.Pointer(&countBits)) = count
					for i := 0; i < 8; i++ {
						mapData = append(mapData, byte((countBits>>(i*8))&0xFF))
					}

					for idx, ch := range result {
						// Key: index
						keyVal := float64(idx)
						keyBits := uint64(0)
						*(*float64)(unsafe.Pointer(&keyBits)) = keyVal
						for i := 0; i < 8; i++ {
							mapData = append(mapData, byte((keyBits>>(i*8))&0xFF))
						}

						// Value: char code
						charVal := float64(ch)
						charBits := uint64(0)
						*(*float64)(unsafe.Pointer(&charBits)) = charVal
						for i := 0; i < 8; i++ {
							mapData = append(mapData, byte((charBits>>(i*8))&0xFF))
						}
					}

					fc.eb.Define(labelName, string(mapData))
					fc.out.LeaSymbolToReg("rax", labelName)
					fc.out.SubImmFromReg("rsp", 8)
					fc.out.MovRegToMem("rax", "rsp", 0)
					fc.out.MovMemToXmm("xmm0", "rsp", 0)
					fc.out.AddImmToReg("rsp", 8)
					break
				} else {
					// Runtime string concatenation
					// Evaluate left string (result pointer in xmm0)
					fc.compileExpression(e.Left)
					// Save left pointer to stack
					fc.out.SubImmFromReg("rsp", 16)
					fc.out.MovXmmToMem("xmm0", "rsp", 0)

					// Evaluate right string (result pointer in xmm0)
					fc.compileExpression(e.Right)
					// Save right pointer to stack
					fc.out.SubImmFromReg("rsp", 16)
					fc.out.MovXmmToMem("xmm0", "rsp", 0)

					// Call _flap_string_concat(left_ptr, right_ptr)
					// Load arguments into registers following x86-64 calling convention
					fc.out.MovMemToReg("rdi", "rsp", 16) // left ptr (first arg)
					fc.out.MovMemToReg("rsi", "rsp", 0)  // right ptr (second arg)
					fc.out.AddImmToReg("rsp", 32)        // clean up stack

					// Align stack for call (must be at 16n+8 before CALL)
					fc.out.SubImmFromReg("rsp", 8)

					// Call the helper function (direct call, not through PLT)
					fc.out.CallSymbol("_flap_string_concat")

					// Restore stack alignment
					fc.out.AddImmToReg("rsp", 8)

					// Result pointer is in rax, convert to xmm0
					fc.out.SubImmFromReg("rsp", 8)
					fc.out.MovRegToMem("rax", "rsp", 0)
					fc.out.MovMemToXmm("xmm0", "rsp", 0)
					fc.out.AddImmToReg("rsp", 8)
					break
				}
			}

			if leftType == "list" && rightType == "list" {
				// List concatenation - TODO
				fmt.Fprintf(os.Stderr, "Error: list concatenation not yet implemented\n")
				os.Exit(1)
			}

			if leftType == "map" && rightType == "map" {
				// Map union - TODO
				fmt.Fprintf(os.Stderr, "Error: map union not yet implemented\n")
				os.Exit(1)
			}
		}

		// Default: numeric binary operation
		// Compile left into xmm0
		fc.compileExpression(e.Left)
		// Save xmm0 to stack
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		// Compile right into xmm0
		fc.compileExpression(e.Right)
		// Move right operand to xmm1
		fc.out.MovRegToReg("xmm1", "xmm0")
		// Load left operand from stack to xmm0
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 16)
		// Perform scalar floating-point operation
		switch e.Operator {
		case "+":
			fc.out.AddsdXmm("xmm0", "xmm1") // addsd xmm0, xmm1
		case "-":
			fc.out.SubsdXmm("xmm0", "xmm1") // subsd xmm0, xmm1
		case "*":
			fc.out.MulsdXmm("xmm0", "xmm1") // mulsd xmm0, xmm1
		case "/":
			fc.out.DivsdXmm("xmm0", "xmm1") // divsd xmm0, xmm1
		case "<", "<=", ">", ">=", "==", "!=":
			// Compare xmm0 with xmm1, sets flags
			fc.out.Ucomisd("xmm0", "xmm1")
			// For now, don't convert to boolean - leave flags set for conditional jump
		case "and":
			// Logical AND: returns 1.0 if both non-zero, else 0.0
			// Compare xmm0 with 0
			fc.out.XorpdXmm("xmm2", "xmm2") // xmm2 = 0.0
			fc.out.Ucomisd("xmm0", "xmm2")
			// Set rax to 1 if xmm0 != 0
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			fc.out.Cmovne("rax", "rcx") // rax = (xmm0 != 0) ? 1 : 0
			// Compare xmm1 with 0
			fc.out.Ucomisd("xmm1", "xmm2")
			// Set rcx to 1 if xmm1 != 0
			fc.out.MovImmToReg("rcx", "0")
			fc.out.MovImmToReg("rdx", "1")
			fc.out.Cmovne("rcx", "rdx") // rcx = (xmm1 != 0) ? 1 : 0
			// AND the results: rax = rax & rcx
			fc.out.AndRegWithReg("rax", "rcx")
			// Convert to float64
			fc.out.Cvtsi2sd("xmm0", "rax")
		case "or":
			// Logical OR: returns 1.0 if either non-zero, else 0.0
			// Compare xmm0 with 0
			fc.out.XorpdXmm("xmm2", "xmm2") // xmm2 = 0.0
			fc.out.Ucomisd("xmm0", "xmm2")
			// Set rax to 1 if xmm0 != 0
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			fc.out.Cmovne("rax", "rcx") // rax = (xmm0 != 0) ? 1 : 0
			// Compare xmm1 with 0
			fc.out.Ucomisd("xmm1", "xmm2")
			// Set rcx to 1 if xmm1 != 0
			fc.out.MovImmToReg("rcx", "0")
			fc.out.MovImmToReg("rdx", "1")
			fc.out.Cmovne("rcx", "rdx") // rcx = (xmm1 != 0) ? 1 : 0
			// OR the results: rax = rax | rcx
			fc.out.OrRegWithReg("rax", "rcx")
			// Convert to float64
			fc.out.Cvtsi2sd("xmm0", "rax")
		case "xor":
			// Logical XOR: returns 1.0 if exactly one non-zero, else 0.0
			// Compare xmm0 with 0
			fc.out.XorpdXmm("xmm2", "xmm2") // xmm2 = 0.0
			fc.out.Ucomisd("xmm0", "xmm2")
			// Set rax to 1 if xmm0 != 0
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			fc.out.Cmovne("rax", "rcx") // rax = (xmm0 != 0) ? 1 : 0
			// Compare xmm1 with 0
			fc.out.Ucomisd("xmm1", "xmm2")
			// Set rcx to 1 if xmm1 != 0
			fc.out.MovImmToReg("rcx", "0")
			fc.out.MovImmToReg("rdx", "1")
			fc.out.Cmovne("rcx", "rdx") // rcx = (xmm1 != 0) ? 1 : 0
			// XOR the results: rax = rax ^ rcx
			fc.out.XorRegWithReg("rax", "rcx")
			// Convert to float64
			fc.out.Cvtsi2sd("xmm0", "rax")
		case "shl":
			// Shift left: convert to int64, shift, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.ShlClReg("rax", "cl")    // rax <<= cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case "shr":
			// Shift right: convert to int64, shift, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.ShrClReg("rax", "cl")    // rax >>= cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case "rol":
			// Rotate left: convert to int64, rotate, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.RolClReg("rax", "cl")    // rol rax, cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case "ror":
			// Rotate right: convert to int64, rotate, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.RorClReg("rax", "cl")    // ror rax, cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		}

	case *CallExpr:
		fc.compileCall(e)

	case *DirectCallExpr:
		fc.compileDirectCall(e)

	case *ListExpr:
		// For now, create list data in .rodata and return pointer
		// TODO: Implement proper list representation with length/capacity
		if len(e.Elements) == 0 {
			// Empty list - return null pointer (0) as float64
			fc.out.XorRegWithReg("rax", "rax")
			fc.out.Cvtsi2sd("xmm0", "rax")
		} else {
			// Allocate list data in .rodata
			labelName := fmt.Sprintf("list_%d", fc.stringCounter)
			fc.stringCounter++

			// Store list as: [length (8 bytes)] [element1] [element2] ...
			var listData []byte

			// First, add length as float64
			length := float64(len(e.Elements))
			lengthBits := uint64(0)
			*(*float64)(unsafe.Pointer(&lengthBits)) = length
			listData = append(listData, byte(lengthBits&0xFF))
			listData = append(listData, byte((lengthBits>>8)&0xFF))
			listData = append(listData, byte((lengthBits>>16)&0xFF))
			listData = append(listData, byte((lengthBits>>24)&0xFF))
			listData = append(listData, byte((lengthBits>>32)&0xFF))
			listData = append(listData, byte((lengthBits>>40)&0xFF))
			listData = append(listData, byte((lengthBits>>48)&0xFF))
			listData = append(listData, byte((lengthBits>>56)&0xFF))

			// Then add elements
			for _, elem := range e.Elements {
				// Evaluate element to get float64 value
				// For now, only support number literals
				if numExpr, ok := elem.(*NumberExpr); ok {
					val := numExpr.Value
					// Convert float64 to 8 bytes (little-endian)
					bits := uint64(0)
					*(*float64)(unsafe.Pointer(&bits)) = val
					listData = append(listData, byte(bits&0xFF))
					listData = append(listData, byte((bits>>8)&0xFF))
					listData = append(listData, byte((bits>>16)&0xFF))
					listData = append(listData, byte((bits>>24)&0xFF))
					listData = append(listData, byte((bits>>32)&0xFF))
					listData = append(listData, byte((bits>>40)&0xFF))
					listData = append(listData, byte((bits>>48)&0xFF))
					listData = append(listData, byte((bits>>56)&0xFF))
				} else {
					fmt.Fprintf(os.Stderr, "Error: list literal elements must be constant numbers\n")
					os.Exit(1)
				}
			}

			fc.eb.Define(labelName, string(listData))
			fc.out.LeaSymbolToReg("rax", labelName)
			// Convert pointer to float64: reinterpret rax as xmm0
			// Push rax to stack, then load as float64 into xmm0
			fc.out.SubImmFromReg("rsp", 8)
			fc.out.MovRegToMem("rax", "rsp", 0)
			fc.out.MovMemToXmm("xmm0", "rsp", 0)
			fc.out.AddImmToReg("rsp", 8)
		}

	case *InExpr:
		// Membership testing: value in container
		// Returns 1.0 if found, 0.0 if not found

		// Compile value to search for
		fc.compileExpression(e.Value)
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Compile container
		fc.compileExpression(e.Container)
		fc.out.MovXmmToMem("xmm0", "rsp", 8)
		fc.out.MovMemToReg("rbx", "rsp", 8) // rbx = container pointer

		// Check if null
		fc.out.CmpRegToImm("rbx", 0)
		fc.labelCounter++
		notNullJump := fc.eb.text.Len()
		fc.out.JumpConditional(JumpNotEqual, 0)
		notNullEnd := fc.eb.text.Len()

		// Null: return 0.0
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm0", "rax")
		endJump1 := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)
		endJump1End := fc.eb.text.Len()

		// Not null: load count and search
		notNullPos := fc.eb.text.Len()
		fc.patchJumpImmediate(notNullJump+2, int32(notNullPos-notNullEnd))

		fc.out.MovMemToXmm("xmm1", "rbx", 0)
		fc.out.Cvttsd2si("rcx", "xmm1")      // rcx = count
		fc.out.MovMemToXmm("xmm2", "rsp", 0) // xmm2 = search value

		// Loop: rdi = index
		fc.out.XorRegWithReg("rdi", "rdi")
		loopStart := fc.eb.text.Len()
		fc.out.CmpRegToReg("rdi", "rcx")
		loopEndJump := fc.eb.text.Len()
		fc.out.JumpConditional(JumpGreaterOrEqual, 0)
		loopEndJumpEnd := fc.eb.text.Len()

		// Load element at index
		fc.out.MovRegToReg("rax", "rdi")
		fc.out.MulRegWithImm("rax", 8)
		fc.out.AddImmToReg("rax", 8)
		fc.out.AddRegToReg("rax", "rbx")
		fc.out.MovMemToXmm("xmm3", "rax", 0)

		// Compare
		fc.out.Ucomisd("xmm2", "xmm3")
		notEqualJump := fc.eb.text.Len()
		fc.out.JumpConditional(JumpNotEqual, 0)
		notEqualEnd := fc.eb.text.Len()

		// Found! Return 1.0
		fc.out.MovImmToReg("rax", "1")
		fc.out.Cvtsi2sd("xmm0", "rax")
		fc.out.AddImmToReg("rsp", 16)
		foundJump := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)
		foundJumpEnd := fc.eb.text.Len()

		// Not equal: next iteration
		notEqualPos := fc.eb.text.Len()
		fc.patchJumpImmediate(notEqualJump+2, int32(notEqualPos-notEqualEnd))
		fc.out.AddImmToReg("rdi", 1)
		fc.out.JumpUnconditional(int32(loopStart - (fc.eb.text.Len() + 5)))

		// Not found: return 0.0
		loopEndPos := fc.eb.text.Len()
		fc.patchJumpImmediate(loopEndJump+2, int32(loopEndPos-loopEndJumpEnd))
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm0", "rax")
		fc.out.AddImmToReg("rsp", 16)

		// Patch end jumps
		endPos := fc.eb.text.Len()
		fc.patchJumpImmediate(endJump1+1, int32(endPos-endJump1End))
		fc.patchJumpImmediate(foundJump+1, int32(endPos-foundJumpEnd))

	case *MapExpr:
		// Map literal stored as: [count (float64)] [key1] [value1] [key2] [value2] ...
		// Even empty maps need a proper data structure with count = 0
		labelName := fmt.Sprintf("map_%d", fc.stringCounter)
		fc.stringCounter++
		var mapData []byte

		// Add count
		count := float64(len(e.Keys))
		countBits := uint64(0)
		*(*float64)(unsafe.Pointer(&countBits)) = count
		for i := 0; i < 8; i++ {
			mapData = append(mapData, byte((countBits>>(i*8))&0xFF))
		}

		// Add key-value pairs (if any)
		for i := range e.Keys {
			if keyNum, ok := e.Keys[i].(*NumberExpr); ok {
				keyBits := uint64(0)
				*(*float64)(unsafe.Pointer(&keyBits)) = keyNum.Value
				for j := 0; j < 8; j++ {
					mapData = append(mapData, byte((keyBits>>(j*8))&0xFF))
				}
			}
			if valNum, ok := e.Values[i].(*NumberExpr); ok {
				valBits := uint64(0)
				*(*float64)(unsafe.Pointer(&valBits)) = valNum.Value
				for j := 0; j < 8; j++ {
					mapData = append(mapData, byte((valBits>>(j*8))&0xFF))
				}
			}
		}

		fc.eb.Define(labelName, string(mapData))
		fc.out.LeaSymbolToReg("rax", labelName)
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)
	case *IndexExpr:
		// Determine if we're indexing a map/string or list
		// Strings are map[uint64]float64, so use map indexing
		isMap := false
		if identExpr, ok := e.List.(*IdentExpr); ok {
			varType := fc.varTypes[identExpr.Name]
			if varType == "map" || varType == "string" {
				isMap = true
			}
		}

		// Compile container expression (returns pointer as float64 in xmm0)
		fc.compileExpression(e.List)
		// Save container pointer to stack
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Compile index/key expression (returns value as float64 in xmm0)
		fc.compileExpression(e.Index)
		// Save key/index to stack
		fc.out.MovXmmToMem("xmm0", "rsp", 8)

		// Load container pointer from stack to rbx
		fc.out.MovMemToXmm("xmm1", "rsp", 0)
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm1", "rsp", 0)
		fc.out.MovMemToReg("rbx", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

		if isMap {
			// SIMD-OPTIMIZED MAP INDEXING
			// =============================
			// Three-tier approach for optimal performance:
			// 1. AVX-512: Process 8 keys/iteration (8× throughput)
			// 2. SSE2:    Process 2 keys/iteration (2× throughput)
			// 3. Scalar:  Process 1 key/iteration (baseline)
			//
			// Map format: [count (float64)][key1][value1][key2][value2]...
			// Keys are interleaved with values at 16-byte strides
			//
			// Load key to search for from stack into xmm2
			fc.out.MovMemToXmm("xmm2", "rsp", 8)

			// Load count from [rbx]
			fc.out.MovMemToXmm("xmm1", "rbx", 0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = count

			// Check if count is 0
			fc.out.CmpRegToImm("rcx", 0)
			notFoundJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpEqual, 0)
			notFoundEnd := fc.eb.text.Len()

			// Start at first key-value pair (skip 8-byte count)
			fc.out.AddImmToReg("rbx", 8)

			// ============ AVX-512 PATH (8 keys/iteration) ============
			// Runtime CPU detection: check if AVX-512 is supported
			// AVX-512 is available on Intel Xeon Scalable and some high-end desktop CPUs
			// Requires: AVX512F, AVX512DQ for VGATHERQPD and VCMPPD with k-registers

			// Check cpu_has_avx512 flag
			fc.out.LeaSymbolToReg("r15", "cpu_has_avx512")
			fc.out.Emit([]byte{0x41, 0x80, 0x3f, 0x00}) // cmp byte [r15], 0
			avx512NotSupportedJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpEqual, 0) // Jump to SSE2 if not supported
			avx512NotSupportedEnd := fc.eb.text.Len()

			// Check if we can process 8 at a time (count >= 8)
			fc.out.CmpRegToImm("rcx", 8)
			avx512SkipJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpLess, 0)
			avx512SkipEnd := fc.eb.text.Len()

			// Broadcast search key to all 8 lanes of zmm3
			// vbroadcastsd zmm3, xmm2
			fc.out.Emit([]byte{0x62, 0xf2, 0xfd, 0x48, 0x19, 0xda}) // EVEX.512.66.0F38.W1 19 /r

			// Set up gather indices for keys at 16-byte strides
			// Keys are at offsets: 0, 16, 32, 48, 64, 80, 96, 112 from rbx
			// Store indices in ymm4 (we need 8 x 64-bit indices for VGATHERQPD)
			// Using stack to construct index vector
			fc.out.SubImmFromReg("rsp", 64) // Space for 8 indices
			for i := 0; i < 8; i++ {
				fc.out.MovImmToReg("rax", fmt.Sprintf("%d", i*16))
				fc.out.MovRegToMem("rax", "rsp", i*8)
			}
			// Load indices into zmm4
			// vmovdqu64 zmm4, [rsp]
			fc.out.Emit([]byte{0x62, 0xf1, 0xfe, 0x48, 0x6f, 0x24, 0x24}) // EVEX.512.F3.0F.W1 6F /r

			// AVX-512 loop
			avx512LoopStart := fc.eb.text.Len()

			// Gather 8 keys using VGATHERQPD
			// vgatherqpd zmm0{k1}, [rbx + zmm4*1]
			// First, set mask k1 to all 1s (we want all 8 values)
			fc.out.Emit([]byte{0xc5, 0xf8, 0x92, 0xc9}) // kmovb k1, ecx (set to 0xFF)
			// Actually, let's use kxnorb k1, k1, k1 to set all bits to 1
			fc.out.Emit([]byte{0xc5, 0xfc, 0x46, 0xc9}) // kxnorb k1, k0, k1 -> k1 = 0xFF

			// vgatherqpd zmm0{k1}, [rbx + zmm4*1]
			// EVEX.512.66.0F38.W1 92 /r
			// This is complex - we need rbx as base, zmm4 as index, scale=1
			fc.out.Emit([]byte{0x62, 0xf2, 0xfd, 0x49, 0x92, 0x04, 0xe3}) // [rbx + zmm4*1]

			// Compare all 8 keys with search key
			// vcmppd k2{k1}, zmm0, zmm3, 0 (EQ_OQ)
			fc.out.Emit([]byte{0x62, 0xf1, 0xfd, 0x49, 0xc2, 0xd3, 0x00}) // EVEX.512.66.0F.W1 C2 /r ib

			// Extract mask to GPR
			// kmovb eax, k2
			fc.out.Emit([]byte{0xc5, 0xf9, 0x90, 0xc2}) // kmovb eax, k2

			// Test if any key matched
			fc.out.Emit([]byte{0x85, 0xc0}) // test eax, eax
			avx512FoundJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpNotEqual, 0)
			avx512FoundEnd := fc.eb.text.Len()

			// No match - advance by 128 bytes (8 key-value pairs)
			fc.out.AddImmToReg("rbx", 128)
			fc.out.SubImmFromReg("rcx", 8)
			// Continue if count >= 8
			fc.out.CmpRegToImm("rcx", 8)
			fc.out.JumpConditional(JumpGreaterOrEqual, int32(avx512LoopStart-(fc.eb.text.Len()+6)))

			// Clean up indices from stack and fall through to SSE2
			fc.out.AddImmToReg("rsp", 64)
			avx512ToSse2Jump := fc.eb.text.Len()
			fc.out.JumpUnconditional(0)
			avx512ToSse2End := fc.eb.text.Len()

			// AVX-512 match found - determine which key matched
			avx512FoundPos := fc.eb.text.Len()
			fc.patchJumpImmediate(avx512FoundJump+2, int32(avx512FoundPos-avx512FoundEnd))

			// Use BSF (bit scan forward) to find first set bit
			// bsf edx, eax
			fc.out.Emit([]byte{0x0f, 0xbc, 0xd0}) // bsf edx, eax

			// edx now contains index (0-7) of matched key
			// Calculate offset: base_rbx + (edx * 16) + 8 for value
			// shl edx, 4  (multiply by 16)
			fc.out.Emit([]byte{0xc1, 0xe2, 0x04}) // shl edx, 4
			// add edx, 8 (offset to value)
			fc.out.Emit([]byte{0x83, 0xc2, 0x08}) // add edx, 8
			// Load value at [rbx + rdx]
			// movsd xmm0, [rbx + rdx]
			fc.out.Emit([]byte{0xf2, 0x48, 0x0f, 0x10, 0x04, 0x13}) // movsd xmm0, [rbx+rdx]

			// Clean up and jump to end
			fc.out.AddImmToReg("rsp", 64)
			avx512DoneJump := fc.eb.text.Len()
			fc.out.JumpUnconditional(0)
			avx512DoneEnd := fc.eb.text.Len()

			// ============ SSE2 PATH (2 keys/iteration) ============
			avx512SkipPos := fc.eb.text.Len()
			fc.patchJumpImmediate(avx512NotSupportedJump+2, int32(avx512SkipPos-avx512NotSupportedEnd))
			fc.patchJumpImmediate(avx512SkipJump+2, int32(avx512SkipPos-avx512SkipEnd))
			fc.patchJumpImmediate(avx512ToSse2Jump+1, int32(avx512SkipPos-avx512ToSse2End))

			// Broadcast search key to both lanes of xmm3 for SSE2 comparison
			// unpcklpd xmm3, xmm2, xmm2 duplicates xmm2 into both 64-bit lanes
			fc.out.MovXmmToXmm("xmm3", "xmm2")
			fc.out.Emit([]byte{0x66, 0x0f, 0x14, 0xda}) // unpcklpd xmm3, xmm2

			// Check if we can process 2 at a time (count >= 2)
			fc.out.CmpRegToImm("rcx", 2)
			scalarLoopJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpLess, 0)
			scalarLoopEnd := fc.eb.text.Len()

			// SIMD loop: process 2 key-value pairs at a time
			simdLoopStart := fc.eb.text.Len()
			// Load key1 from [rbx] into low lane of xmm0
			fc.out.MovMemToXmm("xmm0", "rbx", 0)
			// Load key2 from [rbx+16] into low lane of xmm1
			fc.out.MovMemToXmm("xmm1", "rbx", 16)
			// Pack both keys into xmm0: [key1 | key2]
			fc.out.Emit([]byte{0x66, 0x0f, 0x14, 0xc1}) // unpcklpd xmm0, xmm1

			// Compare both keys with search key in parallel
			// cmpeqpd xmm0, xmm3 (sets all bits in lane to 1 if equal)
			fc.out.Emit([]byte{0x66, 0x0f, 0xc2, 0xc3, 0x00}) // cmpeqpd xmm0, xmm3, 0

			// Extract comparison mask to eax
			// movmskpd eax, xmm0 (bit 0 = key1 match, bit 1 = key2 match)
			fc.out.Emit([]byte{0x66, 0x0f, 0x50, 0xc0}) // movmskpd eax, xmm0

			// Test if any key matched (eax != 0)
			fc.out.Emit([]byte{0x85, 0xc0}) // test eax, eax
			simdFoundJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpNotEqual, 0)
			simdFoundEnd := fc.eb.text.Len()

			// No match - advance by 32 bytes (2 key-value pairs)
			fc.out.AddImmToReg("rbx", 32)
			fc.out.SubImmFromReg("rcx", 2)
			// Continue if count >= 2
			fc.out.CmpRegToImm("rcx", 2)
			fc.out.JumpConditional(JumpGreaterOrEqual, int32(simdLoopStart-(fc.eb.text.Len()+6)))

			// Fall through to scalar loop if count < 2
			scalarFallThrough := fc.eb.text.Len()
			fc.out.JumpUnconditional(0)
			scalarFallThroughEnd := fc.eb.text.Len()

			// SIMD match found - determine which key matched
			simdFoundPos := fc.eb.text.Len()
			fc.patchJumpImmediate(simdFoundJump+2, int32(simdFoundPos-simdFoundEnd))

			// Test bit 0 (key1 match)
			fc.out.Emit([]byte{0xa8, 0x01}) // test al, 1
			key1MatchJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpNotEqual, 0)
			key1MatchEnd := fc.eb.text.Len()

			// Bit 0 not set, must be bit 1 (key2 match) - load value at [rbx+24]
			fc.out.MovMemToXmm("xmm0", "rbx", 24)
			simdDoneJump := fc.eb.text.Len()
			fc.out.JumpUnconditional(0)
			simdDoneEnd := fc.eb.text.Len()

			// Key1 matched - load value at [rbx+8]
			key1MatchPos := fc.eb.text.Len()
			fc.patchJumpImmediate(key1MatchJump+2, int32(key1MatchPos-key1MatchEnd))
			fc.out.MovMemToXmm("xmm0", "rbx", 8)

			// Patch SIMD done jump to skip scalar loop
			allDoneJump := fc.eb.text.Len()
			fc.out.JumpUnconditional(0)
			allDoneEnd := fc.eb.text.Len()

			simdDonePos := fc.eb.text.Len()
			fc.patchJumpImmediate(simdDoneJump+1, int32(simdDonePos-simdDoneEnd))
			fc.out.JumpUnconditional(int32(allDoneJump - fc.eb.text.Len() - 5))

			// SCALAR loop: handle remaining keys (when count < 2 or remainder)
			scalarLoopPos := fc.eb.text.Len()
			fc.patchJumpImmediate(scalarLoopJump+2, int32(scalarLoopPos-scalarLoopEnd))
			fc.patchJumpImmediate(scalarFallThrough+1, int32(scalarLoopPos-scalarFallThroughEnd))

			// Check if any keys remain
			fc.out.CmpRegToImm("rcx", 0)
			scalarDoneJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpEqual, 0)
			scalarDoneEnd := fc.eb.text.Len()

			scalarLoopStart := fc.eb.text.Len()
			// Load current key from [rbx] into xmm1
			fc.out.MovMemToXmm("xmm1", "rbx", 0)
			// Compare key with search key (xmm1 vs xmm2)
			fc.out.Ucomisd("xmm1", "xmm2")

			// If equal, jump to found
			foundJump := fc.eb.text.Len()
			fc.out.JumpConditional(JumpEqual, 0)
			foundEnd := fc.eb.text.Len()

			// Not equal - advance to next pair (16 bytes)
			fc.out.AddImmToReg("rbx", 16)
			fc.out.SubImmFromReg("rcx", 1)
			// If counter > 0, continue loop
			fc.out.CmpRegToImm("rcx", 0)
			fc.out.JumpConditional(JumpNotEqual, int32(scalarLoopStart-(fc.eb.text.Len()+6)))

			// Not found - return 0.0
			scalarDonePos := fc.eb.text.Len()
			fc.patchJumpImmediate(scalarDoneJump+2, int32(scalarDonePos-scalarDoneEnd))
			notFoundPos := fc.eb.text.Len()
			fc.patchJumpImmediate(notFoundJump+2, int32(notFoundPos-notFoundEnd))
			fc.out.XorRegWithReg("rax", "rax")
			fc.out.Cvtsi2sd("xmm0", "rax")
			notFoundDoneJump := fc.eb.text.Len()
			fc.out.JumpUnconditional(0)
			notFoundDoneEnd := fc.eb.text.Len()

			// Scalar found - load value at [rbx + 8]
			foundPos := fc.eb.text.Len()
			fc.patchJumpImmediate(foundJump+2, int32(foundPos-foundEnd))
			fc.out.MovMemToXmm("xmm0", "rbx", 8)

			// All done - patch final jumps
			allDonePos := fc.eb.text.Len()
			fc.patchJumpImmediate(allDoneJump+1, int32(allDonePos-allDoneEnd))
			fc.patchJumpImmediate(avx512DoneJump+1, int32(allDonePos-avx512DoneEnd))
			fc.patchJumpImmediate(notFoundDoneJump+1, int32(allDonePos-notFoundDoneEnd))

		} else {
			// LIST INDEXING: Position-based indexing
			// Load index from stack
			fc.out.MovMemToXmm("xmm0", "rsp", 8)
			// Convert index from float64 to integer in rax
			fc.out.Cvttsd2si("rax", "xmm0")

			// Skip the length prefix (first 8 bytes)
			fc.out.AddImmToReg("rbx", 8)

			// Calculate offset: rax * 8 (each float64 is 8 bytes)
			fc.out.MulRegWithImm("rax", 8)

			// Add offset to base pointer: rbx = rbx + rax
			fc.out.AddRegToReg("rbx", "rax")

			// Load float64 from [rbx] into xmm0
			fc.out.MovMemToXmm("xmm0", "rbx", 0)
		}

		// Clean up stack (remove saved key/index)
		fc.out.AddImmToReg("rsp", 16)

	case *LambdaExpr:
		// Generate a unique function name for this lambda
		fc.lambdaCounter++
		funcName := fmt.Sprintf("lambda_%d", fc.lambdaCounter)

		// Store lambda for later code generation
		fc.lambdaFuncs = append(fc.lambdaFuncs, LambdaFunc{
			Name:   funcName,
			Params: e.Params,
			Body:   e.Body,
		})

		// Return function pointer as float64 in xmm0
		// Use LEA to get function address, then convert to float64
		fc.out.LeaSymbolToReg("rax", funcName)
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case *LengthExpr:
		// Compile the operand (should be a list, returns pointer as float64 in xmm0)
		fc.compileExpression(e.Operand)

		// Convert pointer from float64 to integer in rax
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rax", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

		// Check if pointer is null (empty list)
		fc.out.CmpRegToImm("rax", 0)
		skipJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpNotEqual, 0) // Jump if not null
		skipJumpEnd := fc.eb.text.Len()

		// Empty list case: return 0.0
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm0", "rax")

		endJumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Jump to end
		endJumpEnd := fc.eb.text.Len()

		// Non-null case: load length from list
		notNullPos := fc.eb.text.Len()
		fc.out.MovMemToXmm("xmm0", "rax", 0)

		// Patch the skip jump
		skipOffset := int32(notNullPos - skipJumpEnd)
		fc.patchJumpImmediate(skipJumpPos+2, skipOffset)

		// Patch end jump
		finalPos := fc.eb.text.Len()
		endOffset := int32(finalPos - endJumpEnd)
		fc.patchJumpImmediate(endJumpPos+1, endOffset)

		// Length is now in xmm0 as float64

	case *MatchExpr:
		fc.compileMatchExpr(e)

	case *ParallelExpr:
		fc.compileParallelExpr(e)

	case *PipeExpr:
		fc.compilePipeExpr(e)

	case *ConcurrentGatherExpr:
		fc.compileConcurrentGatherExpr(e)
	}
}

func (fc *FlapCompiler) compileMatchExpr(expr *MatchExpr) {
	// Compile condition - this will set flags via ucomisd for comparisons
	fc.compileExpression(expr.Condition)

	// Increment label counter for uniqueness
	fc.labelCounter++

	// Extract the comparison operator from the condition
	var jumpCond JumpCondition
	needsZeroCompare := false

	if binExpr, ok := expr.Condition.(*BinaryExpr); ok {
		switch binExpr.Operator {
		case "<":
			jumpCond = JumpAboveOrEqual
		case "<=":
			jumpCond = JumpAbove
		case ">":
			jumpCond = JumpBelowOrEqual
		case ">=":
			jumpCond = JumpBelow
		case "==":
			jumpCond = JumpNotEqual
		case "!=":
			jumpCond = JumpEqual
		default:
			needsZeroCompare = true
		}
	} else {
		// For InExpr and other value-returning expressions
		needsZeroCompare = true
	}

	var defaultJumpPos int

	if needsZeroCompare {
		// Compare result in xmm0 against 0.0
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm1", "rax")
		fc.out.Ucomisd("xmm0", "xmm1")
		defaultJumpPos = fc.eb.text.Len()
		fc.out.JumpConditional(JumpEqual, 0) // Jump to default if 0.0
	} else {
		defaultJumpPos = fc.eb.text.Len()
		fc.out.JumpConditional(jumpCond, 0)
	}

	// Compile true expression (result in xmm0)
	fc.compileExpression(expr.TrueExpr)

	// Jump over default expression
	endJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // Placeholder offset

	// Mark default expression position
	defaultPos := fc.eb.text.Len()

	// Patch conditional jump to default
	defaultOffset := int32(defaultPos - (defaultJumpPos + 6))
	fc.patchJumpImmediate(defaultJumpPos+2, defaultOffset)

	// Compile default expression (result in xmm0)
	fc.compileExpression(expr.DefaultExpr)

	// Mark end position
	endPos := fc.eb.text.Len()

	// Patch unconditional jump to end
	endOffset := int32(endPos - (endJumpPos + 5))
	fc.patchJumpImmediate(endJumpPos+1, endOffset)
}

func (fc *FlapCompiler) compileParallelExpr(expr *ParallelExpr) {
	// For now, only support: list || lambda
	lambda, ok := expr.Operation.(*LambdaExpr)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: parallel operator (||) currently only supports lambda expressions\n")
		os.Exit(1)
	}

	if len(lambda.Params) != 1 {
		fmt.Fprintf(os.Stderr, "Error: parallel operator lambda must have exactly one parameter\n")
		os.Exit(1)
	}

	const (
		parallelResultAlloc    = 2080
		lambdaScratchOffset    = parallelResultAlloc - 8
		savedLambdaSpillOffset = parallelResultAlloc + 8
	)

	// Compile the lambda to get its function pointer (result in xmm0)
	fc.compileExpression(expr.Operation)

	// Save lambda function pointer (currently in xmm0) to stack and convert once to raw pointer bits
	fc.out.SubImmFromReg("rsp", 16)
	fc.out.MovXmmToMem("xmm0", "rsp", 8) // Store at rsp+8
	fc.out.MovMemToReg("r11", "rsp", 8)  // Reinterpret float64 bits as pointer
	fc.out.MovRegToMem("r11", "rsp", 8)  // Keep integer pointer for later loads

	// Compile the input list expression (returns pointer as float64 in xmm0)
	fc.compileExpression(expr.List)

	// Save list pointer to stack (reuse reserved slot) and load as integer pointer
	fc.out.MovXmmToMem("xmm0", "rsp", 0) // Store at rsp+0
	fc.out.MovMemToReg("r13", "rsp", 0)

	// Handle empty lists early (null pointer - nothing to map)
	fc.out.CmpRegToImm("r13", 0)
	nonNullJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	// Null case: return 0.0 as float64 and clean up stack
	fc.out.AddImmToReg("rsp", 16) // Clean up lambda/list pointers
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.Cvtsi2sd("xmm0", "rax")
	nullReturnJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	nullReturnJumpEnd := fc.eb.text.Len()

	// Non-null input list continues here
	nonNullListStart := fc.eb.text.Len()

	// Load list length from [r13] into r14
	fc.out.MovMemToXmm("xmm0", "r13", 0)
	fc.out.Cvttsd2si("r14", "xmm0") // r14 = length as integer

	// Allocate result list on stack: 8 bytes (length) + length * 8 bytes (elements)
	// Reserve an extra 16 bytes at the end to keep the lambda pointer reachable for future vector paths
	// parallelResultAlloc keeps the stack aligned once the initial 16-byte spill area is considered
	fc.out.SubImmFromReg("rsp", parallelResultAlloc)

	// Store result list pointer in r12
	fc.out.MovRegToReg("r12", "rsp") // r12 = result list base

	// Move the saved lambda pointer into the reserved scratch slot inside the result buffer
	fc.out.MovMemToReg("r10", "r12", savedLambdaSpillOffset)
	fc.out.MovRegToMem("r10", "r12", lambdaScratchOffset)

	// Store length in result list
	fc.out.MovMemToXmm("xmm0", "r13", 0) // Reload length as float64
	fc.out.MovXmmToMem("xmm0", "r12", 0)

	// Initialize loop counter to 0
	fc.out.XorRegWithReg("r15", "r15") // r15 = index

	// Loop start
	loopStart := fc.eb.text.Len()

	// Check if index >= length
	fc.out.CmpRegToReg("r15", "r14")
	loopEndJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	// Load element from input list: input_list[index]
	// Element address = r13 + 8 + (r15 * 8)
	fc.out.MovRegToReg("rax", "r15")
	fc.out.MulRegWithImm("rax", 8)
	fc.out.AddImmToReg("rax", 8)     // skip length
	fc.out.AddRegToReg("rax", "r13") // rax = address of element

	// Load element into xmm0 (this is the argument to the lambda)
	fc.out.MovMemToXmm("xmm0", "rax", 0)

	// Load lambda function pointer (stored in the reserved scratch slot) and call it
	fc.out.MovMemToReg("r11", "r12", lambdaScratchOffset)

	// Call the lambda function with element in xmm0
	fc.out.CallRegister("r11")

	// Result is in xmm0, store it in output list: result_list[index]
	fc.out.MovRegToReg("rax", "r15")
	fc.out.MulRegWithImm("rax", 8)
	fc.out.AddImmToReg("rax", 8)     // skip length
	fc.out.AddRegToReg("rax", "r12") // rax = address in result list
	fc.out.MovXmmToMem("xmm0", "rax", 0)

	// Increment index
	fc.out.IncReg("r15")

	// Jump back to loop start
	loopBackJumpPos := fc.eb.text.Len()
	backOffset := int32(loopStart - (loopBackJumpPos + 5))
	fc.out.JumpUnconditional(backOffset)

	// Loop end
	loopEndPos := fc.eb.text.Len()

	// Patch conditional jump
	endOffset := int32(loopEndPos - (loopEndJumpPos + 6))
	fc.patchJumpImmediate(loopEndJumpPos+2, endOffset)

	// Clean up only the lambda/list pointer spill area (16 bytes)
	// Leave result buffer on stack since we're returning a pointer to it
	// Note: The result buffer (parallelResultAlloc bytes) remains on stack
	// This trades memory for simplicity - acceptable for short programs
	fc.out.AddImmToReg("rsp", 16) // Clean up lambda+list pointers

	// Return result list pointer as float64 in xmm0
	// r12 still points to the result buffer on stack
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovRegToMem("r12", "rsp", 0)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Adjust stack pointer to account for result buffer still being there
	// The calling code must use the result before further stack operations
	fc.out.AddImmToReg("rsp", parallelResultAlloc)

	// End of parallel operator - xmm0 contains result pointer as float64
	endLabel := fc.eb.text.Len()

	// Patch jumps for the null-input fast path
	nonNullOffset := int32(nonNullListStart - (nonNullJumpPos + 6))
	fc.patchJumpImmediate(nonNullJumpPos+2, nonNullOffset)

	// Patch jump for null-input return - skip directly to end
	nullReturnOffset := int32(endLabel - nullReturnJumpEnd)
	fc.patchJumpImmediate(nullReturnJumpPos+1, nullReturnOffset)
}

func (fc *FlapCompiler) generateLambdaFunctions() {
	for _, lambda := range fc.lambdaFuncs {
		// Record the offset of this lambda function in .text
		fc.lambdaOffsets[lambda.Name] = fc.eb.text.Len()

		// Mark the start of the lambda function with a label
		fc.eb.MarkLabel(lambda.Name)

		// Function prologue
		fc.out.PushReg("rbp")
		fc.out.MovRegToReg("rbp", "rsp")
		fc.out.SubImmFromReg("rsp", 8) // Align stack

		// Save previous state
		oldVariables := fc.variables
		oldMutableVars := fc.mutableVars
		oldStackOffset := fc.stackOffset

		// Create new scope for lambda
		fc.variables = make(map[string]int)
		fc.mutableVars = make(map[string]bool)
		fc.stackOffset = 0

		// Store parameters from xmm registers to stack
		// Parameters come in xmm0, xmm1, xmm2, ...
		xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5"}
		for i, paramName := range lambda.Params {
			if i >= len(xmmRegs) {
				fmt.Fprintf(os.Stderr, "Error: lambda has too many parameters (max 6)\n")
				os.Exit(1)
			}

			// Allocate stack space for parameter
			fc.stackOffset += 16
			paramOffset := fc.stackOffset
			fc.variables[paramName] = paramOffset
			fc.mutableVars[paramName] = false

			// Allocate stack space
			fc.out.SubImmFromReg("rsp", 16)

			// Store parameter from xmm register to stack
			fc.out.MovXmmToMem(xmmRegs[i], "rbp", -paramOffset)
		}

		// Set current lambda context for "me" self-reference and tail recursion
		fc.currentLambda = &lambda
		fc.lambdaBodyStart = fc.eb.text.Len()

		// Compile lambda body (result in xmm0)
		fc.compileExpression(lambda.Body)

		// Clear lambda context
		fc.currentLambda = nil

		// Function epilogue
		// Clean up stack
		fc.out.MovRegToReg("rsp", "rbp")
		fc.out.PopReg("rbp")
		fc.out.Ret()

		// Restore previous state
		fc.variables = oldVariables
		fc.mutableVars = oldMutableVars
		fc.stackOffset = oldStackOffset
	}
}

func (fc *FlapCompiler) generateRuntimeHelpers() {
	// Generate _flap_string_concat(left_ptr, right_ptr) -> new_ptr
	// Arguments: rdi = left_ptr, rsi = right_ptr
	// Returns: rax = pointer to new concatenated string

	fc.eb.MarkLabel("_flap_string_concat")

	// Function prologue
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")

	// Save callee-saved registers
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")

	// Align stack to 16 bytes for malloc call
	// After call (8) + push rbp (8) + push 5 regs (40) = 56 bytes
	// We need to subtract 8 more to get 16-byte alignment
	fc.out.SubImmFromReg("rsp", 8)

	// Save arguments
	fc.out.MovRegToReg("r12", "rdi") // r12 = left_ptr
	fc.out.MovRegToReg("r13", "rsi") // r13 = right_ptr

	// Get left string length
	fc.out.MovMemToXmm("xmm0", "r12", 0) // load count as float64
	// Convert float64 to integer using cvttsd2si
	fc.out.Emit([]byte{0xf2, 0x4c, 0x0f, 0x2c, 0xf0}) // cvttsd2si r14, xmm0

	// Get right string length
	fc.out.MovMemToXmm("xmm0", "r13", 0) // load count as float64
	// Convert float64 to integer
	fc.out.Emit([]byte{0xf2, 0x4c, 0x0f, 0x2c, 0xf8}) // cvttsd2si r15, xmm0

	// Calculate total length: rbx = r14 + r15
	fc.out.MovRegToReg("rbx", "r14")
	fc.out.Emit([]byte{0x4c, 0x01, 0xfb}) // add rbx, r15

	// Calculate allocation size: rax = 8 + rbx * 16
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.Emit([]byte{0x48, 0xc1, 0xe0, 0x04}) // shl rax, 4 (multiply by 16)
	fc.out.Emit([]byte{0x48, 0x83, 0xc0, 0x08}) // add rax, 8

	// Align to 16 bytes for safety
	fc.out.Emit([]byte{0x48, 0x83, 0xc0, 0x0f}) // add rax, 15
	fc.out.Emit([]byte{0x48, 0x83, 0xe0, 0xf0}) // and rax, ~15

	// Call malloc(rax)
	fc.out.MovRegToReg("rdi", "rax")
	fc.trackFunctionCall("malloc") // Track for PLT patching
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r10", "rax") // r10 = result pointer

	// Write total count to result
	fc.out.Emit([]byte{0xf2, 0x48, 0x0f, 0x2a, 0xc3}) // cvtsi2sd xmm0, rbx
	fc.out.MovXmmToMem("xmm0", "r10", 0)

	// Copy left string entries
	// memcpy(r10 + 8, r12 + 8, r14 * 16)
	fc.out.Emit([]byte{0x4d, 0x89, 0xf1})             // mov r9, r14 (counter)
	fc.out.Emit([]byte{0x49, 0x8d, 0x74, 0x24, 0x08}) // lea rsi, [r12 + 8]
	fc.out.Emit([]byte{0x49, 0x8d, 0x7a, 0x08})       // lea rdi, [r10 + 8]

	// Loop to copy left entries
	fc.eb.MarkLabel("_concat_copy_left_loop")
	fc.out.Emit([]byte{0x4d, 0x85, 0xc9}) // test r9, r9
	// jz to skip copying if zero length - skip entire loop body (22 + 8 + 3 + 2 = 35 bytes)
	fc.out.Emit([]byte{0x74, 0x23}) // jz +35 bytes (skip the entire loop)

	fc.out.MovMemToXmm("xmm0", "rsi", 0)        // load key
	fc.out.MovXmmToMem("xmm0", "rdi", 0)        // store key
	fc.out.MovMemToXmm("xmm0", "rsi", 8)        // load value
	fc.out.MovXmmToMem("xmm0", "rdi", 8)        // store value
	fc.out.Emit([]byte{0x48, 0x83, 0xc6, 0x10}) // add rsi, 16
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x10}) // add rdi, 16
	fc.out.Emit([]byte{0x49, 0xff, 0xc9})       // dec r9
	fc.out.Emit([]byte{0xeb, 0xd8})             // jmp back to test (-40 bytes)

	// Now copy right string entries with offset keys
	// r15 = right_len (counter), r14 = offset for keys
	fc.out.Emit([]byte{0x49, 0x8d, 0x75, 0x08}) // lea rsi, [r13 + 8]
	// rdi already points to correct position

	fc.eb.MarkLabel("_concat_copy_right_loop")
	fc.out.Emit([]byte{0x4d, 0x85, 0xff}) // test r15, r15
	fc.out.Emit([]byte{0x74, 0x2c})       // jz +44 bytes (skip entire second loop)

	fc.out.MovMemToXmm("xmm0", "rsi", 0)              // load key
	fc.out.Emit([]byte{0xf2, 0x49, 0x0f, 0x2a, 0xce}) // cvtsi2sd xmm1, r14 (offset)
	fc.out.Emit([]byte{0xf2, 0x0f, 0x58, 0xc1})       // addsd xmm0, xmm1 (key += offset)
	fc.out.MovXmmToMem("xmm0", "rdi", 0)              // store adjusted key
	fc.out.MovMemToXmm("xmm0", "rsi", 8)              // load value
	fc.out.MovXmmToMem("xmm0", "rdi", 8)              // store value
	fc.out.Emit([]byte{0x48, 0x83, 0xc6, 0x10})       // add rsi, 16
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x10})       // add rdi, 16
	fc.out.Emit([]byte{0x49, 0xff, 0xcf})             // dec r15
	fc.out.Emit([]byte{0xeb, 0xcf})                   // jmp back to test (-49 bytes)

	// Return result pointer in rax
	fc.out.MovRegToReg("rax", "r10")

	// Restore stack alignment
	fc.out.AddImmToReg("rsp", 8)

	// Restore callee-saved registers
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")

	// Function epilogue
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate flap_string_to_cstr(flap_string_ptr) -> cstr_ptr
	// Converts a Flap string (map format) to a null-terminated C string
	// Argument: xmm0 = Flap string pointer (as float64)
	// Returns: rax = C string pointer
	fc.eb.MarkLabel("flap_string_to_cstr")

	// Function prologue
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")

	// Save callee-saved registers
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")

	// Align stack
	fc.out.SubImmFromReg("rsp", 8)

	// Convert float64 pointer to integer pointer in r12
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)
	fc.out.MovMemToReg("r12", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Get string length from map: count = [r12+0]
	fc.out.MovMemToXmm("xmm0", "r12", 0)
	fc.out.Emit([]byte{0xf2, 0x4c, 0x0f, 0x2c, 0xd8}) // cvttsd2si r11, xmm0 (r11 = count)

	// Allocate memory: malloc(count + 1) for null terminator
	fc.out.MovRegToReg("rdi", "r11")
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x01}) // add rdi, 1
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax") // r13 = C string buffer

	// Initialize: rbx = current index, r12 = map ptr, r13 = cstr ptr, r11 = count
	fc.out.XorRegWithReg("rbx", "rbx") // rbx = 0 (current index)

	// Loop through map entries to extract characters
	fc.eb.MarkLabel("_cstr_convert_loop")
	fc.out.Emit([]byte{0x4c, 0x39, 0xdb}) // cmp rbx, r11
	fc.out.Emit([]byte{0x74, 0x28})       // je +40 bytes (exit loop)

	// Calculate map entry offset: 8 + (rbx * 16) for [count][key0][val0][key1][val1]...
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.Emit([]byte{0x48, 0xc1, 0xe0, 0x04}) // shl rax, 4 (multiply by 16)
	fc.out.Emit([]byte{0x48, 0x83, 0xc0, 0x08}) // add rax, 8

	// Load character code: xmm0 = [r12 + rax + 8] (value field)
	fc.out.Emit([]byte{0xf2, 0x49, 0x0f, 0x10, 0x04, 0x04})       // movsd xmm0, [r12 + rax]
	fc.out.Emit([]byte{0xf2, 0x49, 0x0f, 0x10, 0x44, 0x04, 0x08}) // movsd xmm0, [r12 + rax + 8]

	// Convert character code to byte
	fc.out.Emit([]byte{0xf2, 0x48, 0x0f, 0x2c, 0xc0}) // cvttsd2si rax, xmm0

	// Store character: [r13 + rbx] = al
	fc.out.Emit([]byte{0x41, 0x88, 0x04, 0x1d}) // mov [r13 + rbx], al

	// Increment index
	fc.out.Emit([]byte{0x48, 0xff, 0xc3}) // inc rbx
	fc.out.Emit([]byte{0xeb, 0xd4})       // jmp _cstr_convert_loop (-44 bytes)

	// Add null terminator: [r13 + r11] = 0
	fc.out.Emit([]byte{0x43, 0xc6, 0x04, 0x1d, 0x00}) // mov byte [r13 + r11], 0

	// Return C string pointer in rax
	fc.out.MovRegToReg("rax", "r13")

	// Restore stack alignment
	fc.out.AddImmToReg("rsp", 8)

	// Restore callee-saved registers
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")

	// Function epilogue
	fc.out.PopReg("rbp")
	fc.out.Ret()
}

func (fc *FlapCompiler) compileStoredFunctionCall(call *CallExpr) {
	// Load function pointer from variable
	offset, _ := fc.variables[call.Function]
	fc.out.MovMemToXmm("xmm0", "rbp", -offset)

	// Convert function pointer from float64 to integer in rax
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)
	fc.out.MovMemToReg("rax", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Compile arguments and put them in xmm registers
	xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5"}
	if len(call.Args) > len(xmmRegs) {
		fmt.Fprintf(os.Stderr, "Error: too many arguments to stored function (max 6)\n")
		os.Exit(1)
	}

	// Save function pointer to stack (rax might get clobbered)
	fc.out.SubImmFromReg("rsp", 16)
	fc.out.MovRegToMem("rax", "rsp", 0)

	// Evaluate all arguments and save to stack
	for _, arg := range call.Args {
		fc.compileExpression(arg) // Result in xmm0
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
	}

	// Load arguments from stack into xmm registers (in reverse order)
	for i := len(call.Args) - 1; i >= 0; i-- {
		fc.out.MovMemToXmm(xmmRegs[i], "rsp", 0)
		fc.out.AddImmToReg("rsp", 16)
	}

	// Load function pointer from stack to r11
	fc.out.MovMemToReg("r11", "rsp", 0)
	fc.out.AddImmToReg("rsp", 16)

	// Ensure stack is 16-byte aligned before call
	// After the moves above, rsp should already be aligned
	// but let's verify: original rsp was aligned, we did SubImmFromReg 16 for func ptr
	// then SubImmFromReg 16 for each arg, then AddImmToReg 16 for each arg,
	// then AddImmToReg 16 for func ptr, so we're back to original alignment

	// Call the function pointer in r11
	fc.out.CallRegister("r11")

	// Result is in xmm0
}

func (fc *FlapCompiler) compileDirectCall(call *DirectCallExpr) {
	// Compile the callee expression (e.g., a lambda) to get function pointer
	fc.compileExpression(call.Callee) // Result in xmm0 (function pointer as float64)

	// Convert function pointer from float64 to integer in rax
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)
	fc.out.MovMemToReg("rax", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Compile arguments and put them in xmm registers
	xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5"}
	if len(call.Args) > len(xmmRegs) {
		fmt.Fprintf(os.Stderr, "Error: too many arguments to direct call (max 6)\n")
		os.Exit(1)
	}

	// Save function pointer to stack (rax might get clobbered)
	fc.out.SubImmFromReg("rsp", 16)
	fc.out.MovRegToMem("rax", "rsp", 0)

	// Evaluate all arguments and save to stack
	for _, arg := range call.Args {
		fc.compileExpression(arg) // Result in xmm0
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
	}

	// Load arguments from stack into xmm registers (in reverse order)
	for i := len(call.Args) - 1; i >= 0; i-- {
		fc.out.MovMemToXmm(xmmRegs[i], "rsp", 0)
		fc.out.AddImmToReg("rsp", 16)
	}

	// Load function pointer from stack to r11
	fc.out.MovMemToReg("r11", "rsp", 0)
	fc.out.AddImmToReg("rsp", 16)

	// Call the function pointer in r11
	fc.out.CallRegister("r11")

	// Result is in xmm0
}

// compileMapToCString converts a string map (map[uint64]float64) to a CString
// Input: mapPtr (register name) = pointer to string map
// Output: cstrPtr (register name) = pointer to first character of CString
// CString format: [length_byte][char0][char1]...[charn][newline][null]
//
//	^-- returned pointer points here
func (fc *FlapCompiler) compileMapToCString(mapPtr, cstrPtr string) {
	// Allocate space on stack for CString (max 256 bytes + length + newline + null)
	fc.out.SubImmFromReg("rsp", 260) // 1 (length) + 256 (chars) + 1 (newline) + 1 (null) + padding

	// Check if map pointer is null
	fc.out.CmpRegToImm(mapPtr, 0)
	nullJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	nullEnd := fc.eb.text.Len()

	// Load count from map[0]
	fc.out.MovMemToXmm("xmm0", mapPtr, 0)
	fc.out.Cvttsd2si("rcx", "xmm0") // rcx = character count

	// Store length byte at [rsp]
	fc.out.MovRegToMem("rcx", "rsp", 0) // Just store lower byte

	// rsi = write position (starts at rsp+1, after length byte)
	fc.out.LeaMemToReg("rsi", "rsp", 1)

	// rbx = map pointer (start after count)
	fc.out.MovRegToReg("rbx", mapPtr)
	fc.out.AddImmToReg("rbx", 8) // Skip count field

	// rdi = character index (0, 1, 2, ...)
	fc.out.XorRegWithReg("rdi", "rdi")

	// Loop through each character
	loopStart := fc.eb.text.Len()

	// Check if done (rdi >= rcx)
	fc.out.CmpRegToReg("rdi", "rcx")
	loopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)
	loopEndEnd := fc.eb.text.Len()

	// Find character at index rdi in the map
	// For simplicity, use linear search through map pairs
	// TODO: This is O(n²) - optimize later

	// r8 = current map position
	fc.out.MovRegToReg("r8", "rbx")

	// r9 = remaining keys to check
	fc.out.MovRegToReg("r9", "rcx")

	// Inner loop: search for key == rdi
	innerLoopStart := fc.eb.text.Len()

	// Check if any keys remain
	fc.out.CmpRegToImm("r9", 0)
	innerLoopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	innerLoopEndEnd := fc.eb.text.Len()

	// Load key from [r8]
	fc.out.MovMemToXmm("xmm1", "r8", 0)
	fc.out.Cvttsd2si("r10", "xmm1") // r10 = key as integer

	// Compare with rdi (target index)
	fc.out.CmpRegToReg("r10", "rdi")
	keyMatchJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	keyMatchEnd := fc.eb.text.Len()

	// Not a match, advance to next pair
	fc.out.AddImmToReg("r8", 16) // Skip key+value pair
	fc.out.SubImmFromReg("r9", 1)
	fc.out.JumpUnconditional(int32(innerLoopStart - (fc.eb.text.Len() + 5)))

	// Key matched - load value (character code)
	keyMatchPos := fc.eb.text.Len()
	fc.patchJumpImmediate(keyMatchJump+2, int32(keyMatchPos-keyMatchEnd))

	fc.out.MovMemToXmm("xmm2", "r8", 8) // Load value at [r8+8]
	fc.out.Cvttsd2si("r10", "xmm2")     // r10 = character code

	// Store character byte at [rsi]
	fc.out.MovByteRegToMem("r10", "rsi", 0)

	// Advance write position
	fc.out.AddImmToReg("rsi", 1)

	// Advance character index
	fc.out.AddImmToReg("rdi", 1)

	// Continue outer loop
	fc.out.JumpUnconditional(int32(loopStart - (fc.eb.text.Len() + 5)))

	// Inner loop end (key not found - shouldn't happen for valid strings)
	innerLoopEndPos := fc.eb.text.Len()
	fc.patchJumpImmediate(innerLoopEndJump+2, int32(innerLoopEndPos-innerLoopEndEnd))

	// Store '?' for missing character (shouldn't happen)
	fc.out.MovImmToReg("r10", "63") // ASCII '?'
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)
	fc.out.AddImmToReg("rdi", 1)
	fc.out.JumpUnconditional(int32(loopStart - (fc.eb.text.Len() + 5)))

	// Loop end - all characters processed
	loopEndPos := fc.eb.text.Len()
	fc.patchJumpImmediate(loopEndJump+2, int32(loopEndPos-loopEndEnd))

	// Add newline character
	fc.out.MovImmToReg("r10", "10") // ASCII '\n'
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	// Add null terminator
	fc.out.XorRegWithReg("r10", "r10")
	fc.out.MovByteRegToMem("r10", "rsi", 0)

	// Return pointer to first character (skip length byte)
	fc.out.LeaMemToReg(cstrPtr, "rsp", 1)

	doneJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	doneEnd := fc.eb.text.Len()

	// Null map case - return pointer to empty string
	nullPos := fc.eb.text.Len()
	fc.patchJumpImmediate(nullJump+2, int32(nullPos-nullEnd))

	// Create empty string: length=0, newline, null
	fc.out.XorRegWithReg("r10", "r10")
	fc.out.MovByteRegToMem("r10", "rsp", 0) // length = 0
	fc.out.MovImmToReg("r10", "10")         // newline
	fc.out.MovByteRegToMem("r10", "rsp", 1)
	fc.out.XorRegWithReg("r10", "r10") // null
	fc.out.MovByteRegToMem("r10", "rsp", 2)
	fc.out.LeaMemToReg(cstrPtr, "rsp", 1)

	// Done
	donePos := fc.eb.text.Len()
	fc.patchJumpImmediate(doneJump+1, int32(donePos-doneEnd))

	// Note: Stack not cleaned up here - caller must handle
}

// compilePrintMapAsString converts a string map to bytes for printing via syscall
// Input: mapPtr (register) = pointer to string map, bufPtr (register) = buffer start
// Output: rsi = pointer to string data, rdx = length (including newline)
func (fc *FlapCompiler) compilePrintMapAsString(mapPtr, bufPtr string) {
	// Check if map pointer is null (empty string)
	fc.out.CmpRegToImm(mapPtr, 0)
	nullJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	nullEnd := fc.eb.text.Len()

	// Load count from map[0] (first float64 is the count)
	fc.out.MovMemToXmm("xmm0", mapPtr, 0)
	fc.out.Cvttsd2si("rcx", "xmm0") // rcx = character count

	// rsi = write position (buffer start)
	fc.out.MovRegToReg("rsi", bufPtr)

	// rbx = map data pointer (start after count at offset 8)
	fc.out.MovRegToReg("rbx", mapPtr)
	fc.out.AddImmToReg("rbx", 8)

	// rdi = character index
	fc.out.XorRegWithReg("rdi", "rdi")

	// Loop through each character
	loopStart := fc.eb.text.Len()

	// Check if done (rdi >= rcx)
	fc.out.CmpRegToReg("rdi", "rcx")
	loopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)
	loopEndEnd := fc.eb.text.Len()

	// Linear search for key == rdi
	fc.out.MovRegToReg("r8", "rbx")
	fc.out.MovRegToReg("r9", "rcx")

	innerLoopStart := fc.eb.text.Len()
	fc.out.CmpRegToImm("r9", 0)
	innerLoopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	innerLoopEndEnd := fc.eb.text.Len()

	// Load and compare key
	fc.out.MovMemToXmm("xmm1", "r8", 0)
	fc.out.Cvttsd2si("r10", "xmm1")
	fc.out.CmpRegToReg("r10", "rdi")
	keyMatchJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	keyMatchEnd := fc.eb.text.Len()

	// Not a match, advance
	fc.out.AddImmToReg("r8", 16)
	fc.out.SubImmFromReg("r9", 1)
	fc.out.JumpUnconditional(int32(innerLoopStart - (fc.eb.text.Len() + 5)))

	// Key matched - store character
	keyMatchPos := fc.eb.text.Len()
	fc.patchJumpImmediate(keyMatchJump+2, int32(keyMatchPos-keyMatchEnd))

	fc.out.MovMemToXmm("xmm2", "r8", 8)
	fc.out.Cvttsd2si("r10", "xmm2")
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	// Inner loop end
	innerLoopEndPos := fc.eb.text.Len()
	fc.patchJumpImmediate(innerLoopEndJump+2, int32(innerLoopEndPos-innerLoopEndEnd))

	// Advance character index
	fc.out.AddImmToReg("rdi", 1)
	fc.out.JumpUnconditional(int32(loopStart - (fc.eb.text.Len() + 5)))

	// Loop end - add newline
	loopEndPos := fc.eb.text.Len()
	fc.patchJumpImmediate(loopEndJump+2, int32(loopEndPos-loopEndEnd))

	// Store newline
	fc.out.MovImmToReg("r10", "10") // '\n' = 10
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	// Calculate length: rsi - bufPtr
	fc.out.MovRegToReg("rdx", "rsi")
	fc.out.SubRegFromReg("rdx", bufPtr)

	// Set rsi back to buffer start
	fc.out.MovRegToReg("rsi", bufPtr)

	// Jump to end
	normalEndJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	normalEndEnd := fc.eb.text.Len()

	// Null case - just print newline
	nullPos := fc.eb.text.Len()
	fc.patchJumpImmediate(nullJump+2, int32(nullPos-nullEnd))
	fc.out.MovImmToReg("r10", "10") // '\n'
	fc.out.MovByteRegToMem("r10", bufPtr, 0)
	fc.out.MovRegToReg("rsi", bufPtr)
	fc.out.MovImmToReg("rdx", "1")

	// End
	normalEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(normalEndJump+1, int32(normalEnd-normalEndEnd))
}

// compileFloatToString converts a float64 to ASCII string representation
// Input: xmmReg = XMM register with float64, bufPtr = buffer pointer (register)
// Output: rsi = string start, rdx = length (including newline)
func (fc *FlapCompiler) compileFloatToString(xmmReg, bufPtr string) {
	// Allocate stack space: 16 bytes for float + 32 bytes for output buffer
	fc.out.SubImmFromReg("rsp", 32)
	// Save the float value at rsp+16 (above the output buffer)
	fc.out.MovXmmToMem(xmmReg, "rsp", 16)

	// Check if negative by testing sign bit
	// We'll load 0.0 by converting integer 0
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.Cvtsi2sd("xmm2", "rax") // xmm2 = 0.0
	fc.out.Ucomisd(xmmReg, "xmm2")
	negativeJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpBelow, 0)
	negativeEnd := fc.eb.text.Len()

	// Positive path
	positiveSkipJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	positiveSkipEnd := fc.eb.text.Len()

	// Negative path - add minus sign and negate
	negativePos := fc.eb.text.Len()
	fc.patchJumpImmediate(negativeJump+2, int32(negativePos-negativeEnd))
	fc.out.MovImmToReg("r10", "45") // '-'
	fc.out.MovByteRegToMem("r10", bufPtr, 0)
	fc.out.LeaMemToReg("rsi", bufPtr, 1)

	// Negate the float: multiply by -1
	fc.out.MovMemToXmm("xmm0", "rsp", 16)
	fc.loadFloatConstant("xmm3", -1.0)
	fc.out.MulsdXmm("xmm0", "xmm3")
	fc.out.MovXmmToMem("xmm0", "rsp", 16)

	negativeSkipJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	negativeSkipEnd := fc.eb.text.Len()

	// Positive path target
	positiveSkip := fc.eb.text.Len()
	fc.patchJumpImmediate(positiveSkipJump+1, int32(positiveSkip-positiveSkipEnd))
	fc.out.MovRegToReg("rsi", bufPtr)

	// Negative skip target
	negativeSkip := fc.eb.text.Len()
	fc.patchJumpImmediate(negativeSkipJump+1, int32(negativeSkip-negativeSkipEnd))

	// Now rsi points to where we write, load the (now positive) float
	fc.out.MovMemToXmm("xmm0", "rsp", 16)

	// Check if it's a whole number
	fc.out.Cvttsd2si("rax", "xmm0")
	fc.out.Cvtsi2sd("xmm1", "rax")
	fc.out.Ucomisd("xmm0", "xmm1")

	notWholeJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	notWholeEnd := fc.eb.text.Len()

	// Whole number path - print as integer
	fc.compileIntToStringAtPos("rax", "rsi")

	// If we wrote a '-' sign, we need to adjust rsi to include it
	// Check if byte [bufPtr] == '-' (ASCII 45)
	fc.out.MovMemToReg("r10", bufPtr, 0) // load 8 bytes from bufPtr
	// Emit AND r10, 0xFF manually to mask to low byte
	fc.out.Write(0x49) // REX.W prefix for r10
	fc.out.Write(0x81) // AND r/m64, imm32
	fc.out.Write(0xE2) // ModR/M byte for r10 (11 100 010)
	fc.out.Write(0xFF) // immediate value (low byte)
	fc.out.Write(0x00) // immediate value (next 3 bytes)
	fc.out.Write(0x00)
	fc.out.Write(0x00)
	fc.out.CmpRegToImm("r10", 45) // compare with '-'
	noMinusJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	noMinusEnd := fc.eb.text.Len()

	// Has minus sign - adjust rsi and rdx
	fc.out.MovRegToReg("rsi", bufPtr)
	fc.out.AddImmToReg("rdx", 1) // include the '-' in length

	noMinusPos := fc.eb.text.Len()
	fc.patchJumpImmediate(noMinusJump+2, int32(noMinusPos-noMinusEnd))

	fc.out.AddImmToReg("rsp", 32) // cleanup

	wholeEndJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	wholeEndEnd := fc.eb.text.Len()

	// Float path - print with decimal point
	notWholePos := fc.eb.text.Len()
	fc.patchJumpImmediate(notWholeJump+2, int32(notWholePos-notWholeEnd))

	// Extract integer part (rax already has it from above)
	fc.out.Cvttsd2si("rax", "xmm0")

	// Save int part as float in xmm1 BEFORE printing (printing will clobber rax)
	fc.out.Cvtsi2sd("xmm1", "rax")

	// Print integer part
	fc.compileIntToStringAtPosNoNewline("rax", "rsi")
	// rsi now points after the integer part

	// Add decimal point
	fc.out.MovImmToReg("r10", "46") // '.'
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	// Get fractional part: frac = num - int_part
	fc.out.MovMemToXmm("xmm0", "rsp", 16)
	// xmm1 already has int part as float from above
	fc.out.SubsdXmm("xmm0", "xmm1") // xmm0 = fractional part

	// Print up to 6 decimal digits
	fc.out.MovImmToReg("r11", "6") // digit counter
	fc.loadFloatConstant("xmm3", 10.0)

	fracLoopStart := fc.eb.text.Len()

	// Check if done
	fc.out.CmpRegToImm("r11", 0)
	fracLoopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	fracLoopEndEnd := fc.eb.text.Len()

	// Multiply by 10
	fc.out.MulsdXmm("xmm0", "xmm3")

	// Extract digit (save it first before converting to ASCII)
	fc.out.Cvttsd2si("r10", "xmm0")

	// Convert integer digit back to float for subtraction
	fc.out.Cvtsi2sd("xmm1", "r10")
	fc.out.SubsdXmm("xmm0", "xmm1")

	// Convert digit to ASCII and store
	fc.out.AddImmToReg("r10", 48) // to ASCII
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	fc.out.SubImmFromReg("r11", 1)
	fc.out.JumpUnconditional(int32(fracLoopStart - (fc.eb.text.Len() + 5)))

	fracLoopEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(fracLoopEndJump+2, int32(fracLoopEnd-fracLoopEndEnd))

	// Add newline
	fc.out.MovImmToReg("r10", "10") // '\n'
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	// Calculate length
	fc.out.MovRegToReg("rdx", "rsi")
	fc.out.SubRegFromReg("rdx", bufPtr)
	fc.out.MovRegToReg("rsi", bufPtr)

	fc.out.AddImmToReg("rsp", 32) // cleanup

	// End
	wholeEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(wholeEndJump+1, int32(wholeEnd-wholeEndEnd))
}

// loadFloatConstant loads a float constant into an XMM register
func (fc *FlapCompiler) loadFloatConstant(xmmReg string, value float64) {
	// Create a constant label for this float value
	labelName := fmt.Sprintf("float_const_%d", fc.stringCounter)
	fc.stringCounter++

	// Convert float64 to bytes
	bits := math.Float64bits(value)
	bytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(bytes, bits)
	fc.eb.Define(labelName, string(bytes))

	// Load the address into a temp register, then load the value
	fc.out.LeaSymbolToReg("rax", labelName)
	fc.out.MovMemToXmm(xmmReg, "rax", 0)
}

// compileIntToStringAtPos is like compileIntToString but writes at rsi position
func (fc *FlapCompiler) compileIntToStringAtPos(intReg, posReg string) {
	fc.compileWholeNumberToStringAtPos(intReg, posReg, true)
}

// compileIntToStringAtPosNoNewline writes integer without newline
func (fc *FlapCompiler) compileIntToStringAtPosNoNewline(intReg, posReg string) {
	fc.compileWholeNumberToStringAtPos(intReg, posReg, false)
}

// compileWholeNumberToStringAtPos converts a whole number to ASCII at a given position
// Input: intReg = register with int64, posReg = write position register
// If addNewline is true, adds '\n' and sets rsi/rdx; otherwise just updates posReg
func (fc *FlapCompiler) compileWholeNumberToStringAtPos(intReg, posReg string, addNewline bool) {
	// Store the starting position
	startPosReg := "r15"
	fc.out.MovRegToReg(startPosReg, posReg)

	// Convert digits (rax = number, posReg = write position)
	fc.out.MovRegToReg("rax", intReg)
	fc.out.LeaMemToReg("rdi", posReg, 20) // digit storage area
	fc.out.MovImmToReg("rcx", "10")       // divisor

	digitLoopStart := fc.eb.text.Len()

	// Divide rax by 10
	fc.out.DivRegByReg("rax", "rcx")

	// Convert remainder to ASCII
	fc.out.AddImmToReg("rdx", 48) // '0' = 48
	fc.out.MovByteRegToMem("rdx", "rdi", 0)
	fc.out.AddImmToReg("rdi", 1)

	// Continue if quotient > 0
	fc.out.CmpRegToImm("rax", 0)
	digitLoopJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreater, 0)
	digitLoopEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(digitLoopJump+2, int32(digitLoopStart-(digitLoopEnd)))

	// Copy digits back in reverse
	fc.out.SubImmFromReg("rdi", 1)
	fc.out.LeaMemToReg("r11", posReg, 20)

	copyLoopStart := fc.eb.text.Len()
	fc.out.CmpRegToReg("rdi", "r11")
	copyLoopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLess, 0)
	copyLoopEndEnd := fc.eb.text.Len()

	fc.out.MovMemToReg("r10", "rdi", 0)
	fc.out.MovByteRegToMem("r10", posReg, 0)
	fc.out.AddImmToReg(posReg, 1)
	fc.out.SubImmFromReg("rdi", 1)
	fc.out.JumpUnconditional(int32(copyLoopStart - (fc.eb.text.Len() + 5)))

	copyLoopEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(copyLoopEndJump+2, int32(copyLoopEnd-copyLoopEndEnd))

	if addNewline {
		// Add newline
		fc.out.MovImmToReg("r10", "10")
		fc.out.MovByteRegToMem("r10", posReg, 0)
		fc.out.AddImmToReg(posReg, 1)

		// Calculate length
		fc.out.MovRegToReg("rdx", posReg)
		fc.out.SubRegFromReg("rdx", startPosReg)
		fc.out.MovRegToReg("rsi", startPosReg)
	}
}

// compileWholeNumberToString converts a whole number (truncated float) to ASCII string
// Input: intReg = register with int64, bufPtr = buffer pointer (register)
// Output: rsi = string start, rdx = length (including newline)
func (fc *FlapCompiler) compileWholeNumberToString(intReg, bufPtr string) {
	// Special case: zero
	fc.out.CmpRegToImm(intReg, 0)
	zeroJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	zeroEnd := fc.eb.text.Len()

	// Handle negative numbers
	fc.out.CmpRegToImm(intReg, 0)
	negativeJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLess, 0)
	negativeEnd := fc.eb.text.Len()

	// Positive path
	fc.out.MovRegToReg("rax", intReg)
	positiveSkipJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	positiveSkipEnd := fc.eb.text.Len()

	// Negative path
	negativePos := fc.eb.text.Len()
	fc.patchJumpImmediate(negativeJump+2, int32(negativePos-negativeEnd))
	fc.out.MovRegToReg("rax", intReg)
	fc.out.NegReg("rax")

	// Store negative sign
	fc.out.MovImmToReg("r10", "45") // '-' = 45
	fc.out.MovByteRegToMem("r10", bufPtr, 0)
	fc.out.LeaMemToReg("rsi", bufPtr, 1)

	negativeSkipJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	negativeSkipEnd := fc.eb.text.Len()

	// Positive skip target
	positiveSkip := fc.eb.text.Len()
	fc.patchJumpImmediate(positiveSkipJump+1, int32(positiveSkip-positiveSkipEnd))
	fc.out.MovRegToReg("rsi", bufPtr)

	// Negative skip target
	negativeSkip := fc.eb.text.Len()
	fc.patchJumpImmediate(negativeSkipJump+1, int32(negativeSkip-negativeSkipEnd))

	// Convert digits (rax = number, rsi = buffer position)
	// Store digits in reverse, then copy forward
	fc.out.LeaMemToReg("rdi", bufPtr, 20) // digit storage area
	fc.out.MovImmToReg("rcx", "10")       // divisor

	digitLoopStart := fc.eb.text.Len()

	// Divide rax by 10: rax = quotient, rdx = remainder
	fc.out.DivRegByReg("rax", "rcx")

	// Convert remainder to ASCII ('0' + digit)
	fc.out.AddImmToReg("rdx", 48) // '0' = 48
	fc.out.MovByteRegToMem("rdx", "rdi", 0)
	fc.out.AddImmToReg("rdi", 1)

	// Continue if quotient > 0
	fc.out.CmpRegToImm("rax", 0)
	digitLoopJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreater, 0)
	digitLoopEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(digitLoopJump+2, int32(digitLoopStart-(digitLoopEnd)))

	// Copy digits back in reverse order
	fc.out.SubImmFromReg("rdi", 1)        // point to last digit
	fc.out.LeaMemToReg("r11", bufPtr, 20) // r11 = start of digit storage

	copyLoopStart := fc.eb.text.Len()

	// Check if done (rdi < r11 means we've copied all digits)
	fc.out.CmpRegToReg("rdi", "r11")
	copyLoopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLess, 0)
	copyLoopEndEnd := fc.eb.text.Len()

	// Copy byte
	fc.out.MovMemToReg("r10", "rdi", 0)
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)
	fc.out.SubImmFromReg("rdi", 1)
	fc.out.JumpUnconditional(int32(copyLoopStart - (fc.eb.text.Len() + 5)))

	copyLoopEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(copyLoopEndJump+2, int32(copyLoopEnd-copyLoopEndEnd))

	// Add newline
	fc.out.MovImmToReg("r10", "10") // '\n'
	fc.out.MovByteRegToMem("r10", "rsi", 0)
	fc.out.AddImmToReg("rsi", 1)

	// Calculate length
	fc.out.MovRegToReg("rdx", "rsi")
	fc.out.SubRegFromReg("rdx", bufPtr)
	fc.out.MovRegToReg("rsi", bufPtr)

	// Jump to end
	normalEndJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	normalEndEnd := fc.eb.text.Len()

	// Zero case
	zeroPos := fc.eb.text.Len()
	fc.patchJumpImmediate(zeroJump+2, int32(zeroPos-zeroEnd))
	fc.out.MovImmToReg("r10", "48") // '0' = 48
	fc.out.MovByteRegToMem("r10", bufPtr, 0)
	fc.out.MovImmToReg("r10", "10") // '\n'
	fc.out.MovByteRegToMem("r10", bufPtr, 1)
	fc.out.MovRegToReg("rsi", bufPtr)
	fc.out.MovImmToReg("rdx", "2") // length = 2 ("0\n")

	// End
	normalEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(normalEndJump+1, int32(normalEnd-normalEndEnd))
}

func (fc *FlapCompiler) compileTailCall(call *CallExpr) {
	// Tail recursion optimization for "me" self-reference
	// Instead of calling, we update parameters and jump to function start

	if len(call.Args) != len(fc.currentLambda.Params) {
		fmt.Fprintf(os.Stderr, "Error: tail call to 'me' has %d args but function has %d params\n",
			len(call.Args), len(fc.currentLambda.Params))
		os.Exit(1)
	}

	// Step 1: Evaluate all arguments and save to temporary stack locations
	// We need temporaries because arguments may reference current parameters
	tempOffsets := make([]int, len(call.Args))
	for i, arg := range call.Args {
		// Evaluate argument
		fc.compileExpression(arg) // Result in xmm0

		// Save to temporary stack location
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		tempOffsets[i] = fc.stackOffset + 16*(i+1)
	}

	// Step 2: Copy temporary values to parameter locations
	// Parameters are at [rbp - offset] where offset is in fc.variables
	for i, paramName := range fc.currentLambda.Params {
		paramOffset := fc.variables[paramName]
		tempStackPos := 16 * (len(call.Args) - 1 - i)

		// Load from temporary location
		fc.out.MovMemToXmm("xmm0", "rsp", tempStackPos)

		// Store to parameter location
		fc.out.MovXmmToMem("xmm0", "rbp", -paramOffset)
	}

	// Step 3: Clean up temporary stack space
	fc.out.AddImmToReg("rsp", int64(16*len(call.Args)))

	// Step 4: Jump back to lambda body start (tail recursion!)
	jumpOffset := int32(fc.lambdaBodyStart - (fc.eb.text.Len() + 5))
	fc.out.JumpUnconditional(jumpOffset)
}

func (fc *FlapCompiler) compileCall(call *CallExpr) {
	// Check for "me" self-reference (tail recursion candidate)
	if call.Function == "me" && fc.currentLambda != nil {
		fc.compileTailCall(call)
		return
	}

	// Check if this is a stored function (variable containing function pointer)
	if _, isVariable := fc.variables[call.Function]; isVariable {
		fc.compileStoredFunctionCall(call)
		return
	}

	// Otherwise, handle builtin functions
	switch call.Function {
	case "println":
		if len(call.Args) == 0 {
			// Just print a newline
			newlineLabel := fmt.Sprintf("newline_%d", fc.stringCounter)
			fc.stringCounter++
			fc.eb.Define(newlineLabel, "\n")

			// Write newline using syscall: write(1, str, 1)
			fc.out.LeaSymbolToReg("rsi", newlineLabel)
			fc.out.MovImmToReg("rdx", "1") // length = 1
			fc.eb.SysWrite("rsi", "rdx")
			return
		}

		arg := call.Args[0]
		argType := fc.getExprType(arg)

		if strExpr, ok := arg.(*StringExpr); ok {
			// String literal - use direct syscall write
			labelName := fmt.Sprintf("str_%d", fc.stringCounter)
			fc.stringCounter++
			strWithNewline := strExpr.Value + "\n"
			fc.eb.Define(labelName, strWithNewline)

			// Write using syscall: write(1, str, len)
			fc.out.LeaSymbolToReg("rsi", labelName)
			fc.out.MovImmToReg("rdx", fmt.Sprintf("%d", len(strWithNewline))) // length
			fc.eb.SysWrite("rsi", "rdx")
		} else if argType == "string" {
			// String variable - convert map[uint64]float64 to bytes and write with syscall

			// Compile the string expression (returns map pointer as float64 in xmm0)
			fc.compileExpression(arg)

			// Convert xmm0 (string map pointer) to rax
			fc.out.SubImmFromReg("rsp", 8)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)
			fc.out.MovMemToReg("rax", "rsp", 0)
			fc.out.AddImmToReg("rsp", 8)

			// Allocate buffer on stack (260 bytes: length + 256 chars + newline + null)
			fc.out.SubImmFromReg("rsp", 260)

			// Convert map to string buffer
			// rax = map pointer, rsp = buffer start
			fc.compilePrintMapAsString("rax", "rsp")

			// rsi now points to string start, rdx has length (including newline)
			// Write using syscall
			fc.eb.SysWrite("rsi", "rdx")

			// Clean up stack
			fc.out.AddImmToReg("rsp", 260)
		} else {
			// Print number - convert to string and use syscall
			fc.compileExpression(arg)
			// xmm0 contains float64 value

			// Allocate 32 bytes on stack for number string
			fc.out.SubImmFromReg("rsp", 32)

			// Convert float64 in xmm0 to string at rsp
			// Result: rsi = string pointer, rdx = length
			fc.compileFloatToString("xmm0", "rsp")

			// Write using syscall
			fc.eb.SysWrite("rsi", "rdx")

			// Clean up stack
			fc.out.AddImmToReg("rsp", 32)
		}

	case "printf":
		if len(call.Args) == 0 {
			fmt.Fprintf(os.Stderr, "Error: printf() requires at least a format string\n")
			os.Exit(1)
		}

		// First argument must be a string (format string)
		formatArg := call.Args[0]
		strExpr, ok := formatArg.(*StringExpr)
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: printf() first argument must be a string literal\n")
			os.Exit(1)
		}

		// Process format string: %v -> %g (smart float), %b -> %s (boolean), %s -> string
		processedFormat := processEscapeSequences(strExpr.Value)
		boolPositions := make(map[int]bool)   // Track which args are %b (boolean)
		stringPositions := make(map[int]bool) // Track which args are %s (string)

		argPos := 0
		result := ""
		i := 0
		for i < len(processedFormat) {
			if processedFormat[i] == '%' && i+1 < len(processedFormat) {
				next := processedFormat[i+1]
				if next == '%' {
					// Escaped %% - keep as is
					result += "%%"
					i += 2
					continue
				} else if next == 'v' {
					// %v = smart value format (uses %.15g for precision with no trailing zeros)
					result += "%.15g"
					argPos++
					i += 2
					continue
				} else if next == 'b' {
					// %b = boolean (yes/no)
					result += "%s"
					boolPositions[argPos] = true
					argPos++
					i += 2
					continue
				} else if next == 's' {
					// %s = string pointer
					stringPositions[argPos] = true
					argPos++
				} else if next == 'f' || next == 'd' || next == 'g' {
					argPos++
				}
			}
			result += string(processedFormat[i])
			i++
		}

		// Create "yes" and "no" string labels for %b
		yesLabel := fmt.Sprintf("bool_yes_%d", fc.stringCounter)
		noLabel := fmt.Sprintf("bool_no_%d", fc.stringCounter)
		fc.eb.Define(yesLabel, "yes\x00")
		fc.eb.Define(noLabel, "no\x00")

		// Create label for processed format string
		labelName := fmt.Sprintf("str_%d", fc.stringCounter)
		fc.stringCounter++
		fc.eb.Define(labelName, result+"\x00")

		numArgs := len(call.Args) - 1
		if numArgs > 8 {
			fmt.Fprintf(os.Stderr, "Error: printf() supports max 8 arguments (got %d)\n", numArgs)
			os.Exit(1)
		}

		// x86-64 ABI: integers/pointers in rsi,rdx,rcx,r8,r9; floats in xmm0-7
		intRegs := []string{"rsi", "rdx", "rcx", "r8", "r9"}
		xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7"}

		intArgCount := 0
		xmmArgCount := 0

		// Evaluate all arguments
		for i := 1; i < len(call.Args); i++ {
			argIdx := i - 1

			// Special case: string literal arguments for %s
			if stringPositions[argIdx] {
				if strExpr, ok := call.Args[i].(*StringExpr); ok {
					// String literal - load as C string pointer directly
					labelName := fmt.Sprintf("str_%d", fc.stringCounter)
					fc.stringCounter++
					fc.eb.Define(labelName, strExpr.Value+"\x00")
					fc.out.LeaSymbolToReg(intRegs[intArgCount], labelName)
					intArgCount++
					continue
				}
			}

			fc.compileExpression(call.Args[i])

			if boolPositions[argIdx] {
				// %b: Convert float to yes/no string pointer
				fc.out.XorRegWithReg("rax", "rax")
				fc.out.Cvtsi2sd("xmm1", "rax") // xmm1 = 0.0
				fc.out.Ucomisd("xmm0", "xmm1") // Compare with 0.0

				fc.labelCounter++
				yesJump := fc.eb.text.Len()
				fc.out.JumpConditional(JumpNotEqual, 0) // Jump if != 0.0
				yesJumpEnd := fc.eb.text.Len()

				// 0.0 -> "no"
				fc.out.LeaSymbolToReg(intRegs[intArgCount], noLabel)
				noJump := fc.eb.text.Len()
				fc.out.JumpUnconditional(0)
				noJumpEnd := fc.eb.text.Len()

				// Non-zero -> "yes"
				yesPos := fc.eb.text.Len()
				fc.patchJumpImmediate(yesJump+2, int32(yesPos-yesJumpEnd))
				fc.out.LeaSymbolToReg(intRegs[intArgCount], yesLabel)

				endPos := fc.eb.text.Len()
				fc.patchJumpImmediate(noJump+1, int32(endPos-noJumpEnd))

				intArgCount++
			} else if stringPositions[argIdx] {
				// %s: Flap string -> C string conversion
				// xmm0 contains pointer to Flap string map [count][key0][val0][key1][val1]...
				// Call helper function to convert to null-terminated C string
				fc.out.CallSymbol("flap_string_to_cstr")
				// Result in rax is C string pointer
				fc.out.MovRegToReg(intRegs[intArgCount], "rax")
				intArgCount++
			} else {
				// Regular float argument (%v, %f, %g, etc)
				fc.out.SubImmFromReg("rsp", 16)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)
			}
		}

		// Load float arguments from stack into xmm registers (reverse order)
		for i := numArgs - 1; i >= 0; i-- {
			if !boolPositions[i] && !stringPositions[i] {
				fc.out.MovMemToXmm(xmmRegs[xmmArgCount], "rsp", 0)
				fc.out.AddImmToReg("rsp", 16)
				xmmArgCount++
			}
		}

		// Load format string to rdi
		fc.out.LeaSymbolToReg("rdi", labelName)

		// Set rax = number of vector registers used
		fc.out.MovImmToReg("rax", fmt.Sprintf("%d", xmmArgCount))

		fc.trackFunctionCall("printf")
		fc.eb.GenerateCallInstruction("printf")

	case "exit":
		if len(call.Args) > 0 {
			fc.compileExpression(call.Args[0])
			// Convert float64 in xmm0 to int64 in rdi
			fc.out.Cvttsd2si("rdi", "xmm0") // truncate float to int
		} else {
			fc.out.XorRegWithReg("rdi", "rdi")
		}
		fc.trackFunctionCall("exit")
		fc.eb.GenerateCallInstruction("exit")

	case "syscall":
		// Raw Linux syscall: syscall(number, arg1, arg2, arg3, arg4, arg5, arg6)
		// x86-64 syscall convention: rax=number, rdi, rsi, rdx, r10, r8, r9
		if len(call.Args) < 1 || len(call.Args) > 7 {
			fmt.Fprintf(os.Stderr, "Error: syscall() requires 1-7 arguments (syscall number + up to 6 args)\n")
			os.Exit(1)
		}

		// Syscall registers in x86-64: rdi, rsi, rdx, r10, r8, r9
		// Note: r10 is used instead of rcx for syscalls
		argRegs := []string{"rdi", "rsi", "rdx", "r10", "r8", "r9"}

		// Evaluate all arguments and save to stack (in reverse order)
		for i := len(call.Args) - 1; i >= 0; i-- {
			fc.compileExpression(call.Args[i]) // Result in xmm0
			// Convert float64 to int64 and save
			fc.out.Cvttsd2si("rax", "xmm0")
			fc.out.PushReg("rax")
		}

		// Pop syscall number into rax
		fc.out.PopReg("rax")

		// Pop arguments into registers
		numArgs := len(call.Args) - 1 // Exclude syscall number
		for i := 0; i < numArgs && i < 6; i++ {
			fc.out.PopReg(argRegs[i])
		}

		// Execute syscall instruction (0x0f 0x05 for x86-64)
		fc.out.Emit([]byte{0x0f, 0x05})

		// Convert result from rax (int64) to xmm0 (float64)
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "getpid":
		// Call getpid() from libc via PLT
		// getpid() takes no arguments and returns pid_t in rax
		if len(call.Args) != 0 {
			fmt.Fprintf(os.Stderr, "Error: getpid() takes no arguments\n")
			os.Exit(1)
		}
		fc.trackFunctionCall("getpid")
		fc.eb.GenerateCallInstruction("getpid")
		// Convert result from rax to xmm0
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "sqrt":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: sqrt() requires exactly 1 argument\n")
			os.Exit(1)
		}
		// Compile argument (result in xmm0)
		fc.compileExpression(call.Args[0])
		// Use x86-64 SQRTSD instruction (hardware sqrt)
		// sqrtsd xmm0, xmm0 - sqrt of xmm0, result in xmm0
		fc.out.Sqrtsd("xmm0", "xmm0")

	case "sin":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: sin() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FSIN instruction
		// xmm0 -> memory -> ST(0) -> FSIN -> memory -> xmm0
		fc.out.SubImmFromReg("rsp", 8) // Allocate 8 bytes on stack
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0)
		fc.out.Fsin()
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8) // Restore stack

	case "cos":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: cos() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FCOS instruction
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0)
		fc.out.Fcos()
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "tan":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: tan() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FPTAN instruction
		// FPTAN computes tan and pushes 1.0, so we need to pop the 1.0
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0)
		fc.out.Fptan()
		fc.out.Fpop() // Pop the 1.0 that FPTAN pushes
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "atan":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: atan() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FPATAN: atan(x) = atan2(x, 1.0)
		// FPATAN expects ST(1)=y, ST(0)=x, computes atan2(y,x)
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.Fld1()           // ST(0) = 1.0, ST(1) = x
		fc.out.Fpatan()         // ST(0) = atan2(x, 1.0) = atan(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "asin":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: asin() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// asin(x) = atan2(x, sqrt(1 - x²))
		// FPATAN needs ST(1)=x, ST(0)=sqrt(1-x²)
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.FldSt0()         // ST(0) = x, ST(1) = x
		fc.out.FmulSelf()       // ST(0) = x²
		fc.out.Fld1()           // ST(0) = 1.0, ST(1) = x²
		fc.out.Fsubrp()         // ST(0) = 1 - x²
		fc.out.Fsqrt()          // ST(0) = sqrt(1 - x²)
		fc.out.FldMem("rsp", 0) // ST(0) = x, ST(1) = sqrt(1 - x²)
		// Now swap: need ST(1)=x, ST(0)=sqrt(1-x²) but have reverse
		// Solution: save sqrt to mem, reload in reverse order
		fc.out.FstpMem("rsp", 0) // Store x to [rsp], pop, ST(0) = sqrt(1-x²)
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.FstpMem("rsp", 0) // Store sqrt to [rsp+0]
		fc.out.FldMem("rsp", 8)  // Load x: ST(0) = x
		fc.out.FldMem("rsp", 0)  // Load sqrt: ST(0) = sqrt, ST(1) = x
		fc.out.Fpatan()          // ST(0) = atan2(x, sqrt(1-x²)) = asin(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 16) // Restore both allocations

	case "acos":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: acos() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// acos(x) = atan2(sqrt(1-x²), x)
		// FPATAN needs ST(1)=sqrt(1-x²), ST(0)=x
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.FldSt0()         // ST(0) = x, ST(1) = x
		fc.out.FmulSelf()       // ST(0) = x²
		fc.out.Fld1()           // ST(0) = 1.0, ST(1) = x²
		fc.out.Fsubrp()         // ST(0) = 1 - x²
		fc.out.Fsqrt()          // ST(0) = sqrt(1 - x²)
		fc.out.FldMem("rsp", 0) // ST(0) = x, ST(1) = sqrt(1 - x²)
		fc.out.Fpatan()         // ST(0) = atan2(sqrt(1-x²), x) = acos(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "abs":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: abs() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// abs(x) using FABS
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.Fabs()           // ST(0) = |x|
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "floor":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: floor() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// floor(x): round toward -∞
		// FPU control word: set rounding mode to 01 (round down)
		fc.out.SubImmFromReg("rsp", 16) // Need space for control word + value
		fc.out.MovXmmToMem("xmm0", "rsp", 8)

		// Save current FPU control word
		fc.out.FstcwMem("rsp", 0)

		// Load control word, modify to set RC=01 (bits 10-11)
		// Emit 16-bit MOV manually: mov ax, [rsp]
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x8B) // MOV r16, r/m16
		fc.out.Write(0x04) // ModR/M: [rsp]
		fc.out.Write(0x24) // SIB: [rsp]
		// OR ax, 0x0400 (set bit 10 for round down)
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x81) // OR r/m16, imm16
		fc.out.Write(0xC8) // ModR/M for ax
		fc.out.Write(0x00) // Low byte
		fc.out.Write(0x04) // High byte: 0x0400 = bit 10 set (round down)
		// AND ax, 0xF7FF (clear bit 11, keep bit 10)
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x81) // AND r/m16, imm16
		fc.out.Write(0xE0) // ModR/M for ax
		fc.out.Write(0xFF) // Low byte
		fc.out.Write(0xF7) // High byte: 0xF7FF = clear bit 11, keep bit 10
		// Store modified control word: mov [rsp+2], ax
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x89) // MOV r/m16, r16
		fc.out.Write(0x44) // ModR/M: [rsp+disp8]
		fc.out.Write(0x24) // SIB: [rsp]
		fc.out.Write(0x02) // disp8: +2

		// Load modified control word
		fc.out.FldcwMem("rsp", 2)

		// Perform rounding
		fc.out.FldMem("rsp", 8)
		fc.out.Frndint()
		fc.out.FstpMem("rsp", 8)

		// Restore original control word
		fc.out.FldcwMem("rsp", 0)

		fc.out.MovMemToXmm("xmm0", "rsp", 8)
		fc.out.AddImmToReg("rsp", 16)

	case "ceil":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: ceil() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// ceil(x): round toward +∞
		// FPU control word: set rounding mode to 10 (round up)
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 8)

		// Save current FPU control word
		fc.out.FstcwMem("rsp", 0)

		// Load control word, modify to set RC=10 (bits 10-11)
		// Emit 16-bit MOV manually: mov ax, [rsp]
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x8B) // MOV r16, r/m16
		fc.out.Write(0x04) // ModR/M: [rsp]
		fc.out.Write(0x24) // SIB: [rsp]
		// OR ax, 0x0800 (set bit 11 for round up)
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x81) // OR r/m16, imm16
		fc.out.Write(0xC8) // ModR/M for ax
		fc.out.Write(0x00) // Low byte
		fc.out.Write(0x08) // High byte: 0x0800 = bit 11 set (round up)
		// AND ax, 0xFBFF (clear bit 10, keep bit 11)
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x81) // AND r/m16, imm16
		fc.out.Write(0xE0) // ModR/M for ax
		fc.out.Write(0xFF) // Low byte
		fc.out.Write(0xFB) // High byte: 0xFBFF = clear bit 10, keep bit 11
		// Store modified control word: mov [rsp+2], ax
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x89) // MOV r/m16, r16
		fc.out.Write(0x44) // ModR/M: [rsp+disp8]
		fc.out.Write(0x24) // SIB: [rsp]
		fc.out.Write(0x02) // disp8: +2

		fc.out.FldcwMem("rsp", 2)
		fc.out.FldMem("rsp", 8)
		fc.out.Frndint()
		fc.out.FstpMem("rsp", 8)
		fc.out.FldcwMem("rsp", 0) // Restore

		fc.out.MovMemToXmm("xmm0", "rsp", 8)
		fc.out.AddImmToReg("rsp", 16)

	case "round":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: round() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// round(x): round to nearest (even)
		// FPU control word: set rounding mode to 00 (round to nearest)
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 8)

		// Save current FPU control word
		fc.out.FstcwMem("rsp", 0)

		// Load control word, modify to set RC=00 (clear bits 10-11)
		// Emit 16-bit MOV manually: mov ax, [rsp]
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x8B) // MOV r16, r/m16
		fc.out.Write(0x04) // ModR/M: [rsp]
		fc.out.Write(0x24) // SIB: [rsp]
		// AND ax, 0xF3FF (clear bits 10-11 for round to nearest)
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x81) // AND r/m16, imm16
		fc.out.Write(0xE0) // ModR/M for ax
		fc.out.Write(0xFF) // Low byte
		fc.out.Write(0xF3) // High byte: 0xF3FF = clear bits 10-11
		// Store modified control word: mov [rsp+2], ax
		fc.out.Write(0x66) // 16-bit operand prefix
		fc.out.Write(0x89) // MOV r/m16, r16
		fc.out.Write(0x44) // ModR/M: [rsp+disp8]
		fc.out.Write(0x24) // SIB: [rsp]
		fc.out.Write(0x02) // disp8: +2

		fc.out.FldcwMem("rsp", 2)
		fc.out.FldMem("rsp", 8)
		fc.out.Frndint()
		fc.out.FstpMem("rsp", 8)
		fc.out.FldcwMem("rsp", 0) // Restore

		fc.out.MovMemToXmm("xmm0", "rsp", 8)
		fc.out.AddImmToReg("rsp", 16)

	case "log":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: log() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// log(x) = ln(x) = log2(x) / log2(e) = log2(x) * ln(2) / (ln(2) / ln(e))
		// FYL2X computes ST(1) * log2(ST(0))
		// So: log(x) = ln(2) * log2(x) = FYL2X with ST(1)=ln(2), ST(0)=x
		// But we want ln(x), not log2(x)
		// ln(x) = log2(x) * ln(2)
		// Actually: FYL2X gives us: ST(1) * log2(ST(0))
		// So if ST(1) = ln(2) and ST(0) = x, we get: ln(2) * log2(x) = ln(x) ✓
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.Fldln2()         // ST(0) = ln(2)
		fc.out.FldMem("rsp", 0) // ST(0) = x, ST(1) = ln(2)
		fc.out.Fyl2x()          // ST(0) = ln(2) * log2(x) = ln(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "exp":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: exp() requires exactly 1 argument\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0])
		// exp(x) = e^x = 2^(x * log2(e))
		// Steps:
		// 1. Multiply x by log2(e): x' = x * log2(e)
		// 2. Split x' = n + f where n is integer, -1 <= f <= 1
		// 3. Compute 2^f using F2XM1: 2^f = 1 + F2XM1(f)
		// 4. Scale by 2^n using FSCALE
		fc.out.SubImmFromReg("rsp", 8)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.Fldl2e()         // ST(0) = log2(e), ST(1) = x
		fc.out.Fmulp()          // ST(0) = x * log2(e)

		// Now split into integer and fractional parts
		fc.out.FldSt0()    // ST(0) = x', ST(1) = x'
		fc.out.Frndint()   // ST(0) = n (integer part)
		fc.out.FldSt0()    // ST(0) = n, ST(1) = n, ST(2) = x'
		fc.out.Write(0xD9) // FXCH st(2) - exchange ST(0) and ST(2)
		fc.out.Write(0xCA)
		fc.out.Fsubrp() // ST(0) = x' - n = f, ST(1) = n

		// Compute 2^f - 1 using F2XM1
		fc.out.F2xm1() // ST(0) = 2^f - 1, ST(1) = n
		fc.out.Fld1()  // ST(0) = 1, ST(1) = 2^f - 1, ST(2) = n
		fc.out.Faddp() // ST(0) = 2^f, ST(1) = n

		// Scale by 2^n
		fc.out.Fscale() // ST(0) = 2^f * 2^n = 2^(n+f) = e^x, ST(1) = n
		// Discard n (ST(1)) while keeping result in ST(0)
		fc.out.Write(0xDD) // FSTP st(1) - stores ST(0) to st(1), pops stack
		fc.out.Write(0xD9)

		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 8)

	case "pow":
		if len(call.Args) != 2 {
			fmt.Fprintf(os.Stderr, "Error: pow() requires exactly 2 arguments\n")
			os.Exit(1)
		}
		fc.compileExpression(call.Args[0]) // x in xmm0
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.compileExpression(call.Args[1]) // y in xmm0
		fc.out.MovXmmToMem("xmm0", "rsp", 8)

		// pow(x, y) = x^y = 2^(y * log2(x))
		// Steps:
		// 1. Compute log2(x) using FYL2X
		// 2. Multiply by y
		// 3. Split into integer and fractional parts
		// 4. Use F2XM1 and FSCALE like in exp

		fc.out.Fld1()           // ST(0) = 1.0
		fc.out.FldMem("rsp", 0) // ST(0) = x, ST(1) = 1.0
		fc.out.Fyl2x()          // ST(0) = 1 * log2(x) = log2(x)
		fc.out.FldMem("rsp", 8) // ST(0) = y, ST(1) = log2(x)
		fc.out.Fmulp()          // ST(0) = y * log2(x)

		// Split into n + f
		fc.out.FldSt0()    // ST(0) = y*log2(x), ST(1) = y*log2(x)
		fc.out.Frndint()   // ST(0) = n
		fc.out.FldSt0()    // ST(0) = n, ST(1) = n, ST(2) = y*log2(x)
		fc.out.Write(0xD9) // FXCH st(2)
		fc.out.Write(0xCA)
		fc.out.Fsubrp() // ST(0) = f, ST(1) = n

		// Compute 2^f
		fc.out.F2xm1() // ST(0) = 2^f - 1
		fc.out.Fld1()
		fc.out.Faddp()  // ST(0) = 2^f, ST(1) = n
		fc.out.Fscale() // ST(0) = 2^f * 2^n = x^y, ST(1) = n
		// Discard n (ST(1)) while keeping result in ST(0)
		fc.out.Write(0xDD) // FSTP st(1) - stores ST(0) to st(1), pops stack
		fc.out.Write(0xD9)

		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 16)

	default:
		// Unknown function - track it for dependency resolution
		fc.unknownFunctions[call.Function] = true
		fc.trackFunctionCall(call.Function)

		// For now, generate a call instruction hoping it will be resolved
		// In the future, this will be resolved by loading from dependency repos

		// Arguments are passed in xmm0-xmm5 (up to 6 args)
		// Compile arguments in order
		for i, arg := range call.Args {
			fc.compileExpression(arg)
			if i < len(call.Args)-1 {
				// Save result to stack if not the last arg
				fc.out.SubImmFromReg("rsp", 8)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)
			}
		}

		// Restore arguments from stack in reverse order to registers
		// Last arg is already in xmm0
		for i := len(call.Args) - 2; i >= 0; i-- {
			regName := fmt.Sprintf("xmm%d", i)
			fc.out.MovMemToXmm(regName, "rsp", 0)
			fc.out.AddImmToReg("rsp", 8)
		}

		// Generate call instruction
		fc.eb.GenerateCallInstruction(call.Function)
	}
}

func (fc *FlapCompiler) compilePipeExpr(expr *PipeExpr) {
	// Pipe operator: left | right
	// Semantics: Execute left, pass result to right
	// For now, this is a simple sequential composition:
	// 1. Evaluate left expression
	// 2. Pass result (in xmm0) to right expression

	// Compile left side (result will be in xmm0)
	fc.compileExpression(expr.Left)

	// Right side should be a function/lambda that takes the result
	// For now, if right is a lambda or function call, we can evaluate it
	// The result from left is already in xmm0, which is the first parameter

	switch right := expr.Right.(type) {
	case *LambdaExpr:
		// Compile the lambda and call it with the value in xmm0
		// First save the input value
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Compile the lambda to get its function pointer
		fc.compileExpression(right)

		// Convert function pointer from float64 to integer
		fc.out.MovXmmToMem("xmm0", "rsp", 8)
		fc.out.MovMemToReg("r11", "rsp", 8)

		// Restore input value to xmm0
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 16)

		// Call the lambda
		fc.out.CallRegister("r11")

	case *CallExpr:
		// For function calls, the value in xmm0 becomes the first argument
		// This is a simplified implementation
		fc.compileExpression(right)

	case *IdentExpr:
		// Variable reference - could be a lambda stored in a variable
		// Save the input value
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Load the variable (function pointer as float64)
		fc.compileExpression(right)

		// Convert function pointer from float64 to integer
		fc.out.MovXmmToMem("xmm0", "rsp", 8)
		fc.out.MovMemToReg("r11", "rsp", 8)

		// Restore input value to xmm0
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 16)

		// Call the lambda
		fc.out.CallRegister("r11")

	default:
		// For other expressions, just evaluate them
		// This may not be the correct semantics but is a placeholder
		fc.compileExpression(expr.Right)
	}
}

func (fc *FlapCompiler) compileConcurrentGatherExpr(expr *ConcurrentGatherExpr) {
	// Concurrent gather operator: left ||| right
	// Semantics: Gather results concurrently
	// This requires goroutines or threads for true concurrency

	// For now, print an error as this is not yet implemented
	fmt.Fprintf(os.Stderr, "Error: concurrent gather operator ||| is not yet implemented\n")
	fmt.Fprintf(os.Stderr, "This feature requires runtime support for concurrency\n")
	os.Exit(1)
}

func (fc *FlapCompiler) trackFunctionCall(funcName string) {
	if !fc.usedFunctions[funcName] {
		fc.usedFunctions[funcName] = true
	}
	fc.callOrder = append(fc.callOrder, funcName)
}

// collectFunctionCalls walks an expression and collects all function calls
func collectFunctionCalls(expr Expression, calls map[string]bool) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *CallExpr:
		calls[e.Function] = true
		for _, arg := range e.Args {
			collectFunctionCalls(arg, calls)
		}
	case *BinaryExpr:
		collectFunctionCalls(e.Left, calls)
		collectFunctionCalls(e.Right, calls)
	case *PipeExpr:
		collectFunctionCalls(e.Left, calls)
		collectFunctionCalls(e.Right, calls)
	case *ConcurrentGatherExpr:
		collectFunctionCalls(e.Left, calls)
		collectFunctionCalls(e.Right, calls)
	case *MatchExpr:
		collectFunctionCalls(e.Condition, calls)
		collectFunctionCalls(e.TrueExpr, calls)
		collectFunctionCalls(e.DefaultExpr, calls)
	case *LambdaExpr:
		collectFunctionCalls(e.Body, calls)
	case *ListExpr:
		for _, elem := range e.Elements {
			collectFunctionCalls(elem, calls)
		}
	case *MapExpr:
		for i := range e.Keys {
			collectFunctionCalls(e.Keys[i], calls)
			collectFunctionCalls(e.Values[i], calls)
		}
	case *IndexExpr:
		collectFunctionCalls(e.List, calls)
		collectFunctionCalls(e.Index, calls)
	case *LengthExpr:
		collectFunctionCalls(e.Operand, calls)
	}
}

// collectFunctionCallsFromStmt walks a statement and collects all function calls
func collectFunctionCallsFromStmt(stmt Statement, calls map[string]bool) {
	if stmt == nil {
		return
	}

	switch s := stmt.(type) {
	case *AssignStmt:
		collectFunctionCalls(s.Value, calls)
	case *ExpressionStmt:
		collectFunctionCalls(s.Expr, calls)
	case *LoopStmt:
		collectFunctionCalls(s.Iterable, calls)
		for _, bodyStmt := range s.Body {
			collectFunctionCallsFromStmt(bodyStmt, calls)
		}
	}
}

// collectDefinedFunctions returns a set of function names defined in a program
func collectDefinedFunctions(program *Program) map[string]bool {
	defined := make(map[string]bool)

	for _, stmt := range program.Statements {
		if assign, ok := stmt.(*AssignStmt); ok {
			// Check if the value is a lambda (function definition)
			if _, isLambda := assign.Value.(*LambdaExpr); isLambda {
				defined[assign.Name] = true
			}
		}
	}

	return defined
}

// getUnknownFunctions determines which functions are called but not defined
func getUnknownFunctions(program *Program) []string {
	// Builtin functions that are always available (implemented in compiler)
	builtins := map[string]bool{
		"printf": true, "exit": true, "syscall": true,
		"getpid": true, "range": true, "me": true,
		"println": true, // println is a builtin optimization, not a dependency
	}

	// Collect all function calls
	calls := make(map[string]bool)
	for _, stmt := range program.Statements {
		collectFunctionCallsFromStmt(stmt, calls)
	}

	// Collect all defined functions
	defined := collectDefinedFunctions(program)

	// Find unknown functions (called but not builtin and not defined)
	var unknown []string
	for funcName := range calls {
		if !builtins[funcName] && !defined[funcName] {
			unknown = append(unknown, funcName)
		}
	}

	return unknown
}

func CompileFlap(inputPath string, outputPath string) error {
	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", inputPath, err)
	}

	// Parse main file
	parser := NewParserWithFilename(string(content), inputPath)
	program := parser.ParseProgram()

	fmt.Fprintf(os.Stderr, "Parsed program:\n%s\n", program.String())

	// Check for unknown functions and resolve dependencies
	// Build combined source code (dependencies + main)
	var combinedSource string
	unknownFuncs := getUnknownFunctions(program)
	if len(unknownFuncs) > 0 {
		fmt.Fprintf(os.Stderr, "Resolving dependencies for: %v\n", unknownFuncs)

		// Resolve dependencies
		repos := ResolveDependencies(unknownFuncs)
		if len(repos) > 0 {
			fmt.Fprintf(os.Stderr, "Loading dependencies from %d repositories\n", len(repos))

			// Ensure all repositories are cloned/updated
			for _, repoURL := range repos {
				repoPath, err := EnsureRepoCloned(repoURL, UpdateDepsFlag)
				if err != nil {
					return fmt.Errorf("failed to fetch dependency %s: %v", repoURL, err)
				}

				// Find all .flap files in the repository
				flapFiles, err := FindFlapFiles(repoPath)
				if err != nil {
					return fmt.Errorf("failed to find .flap files in %s: %v", repoPath, err)
				}

				// Parse and merge each .flap file
				for _, flapFile := range flapFiles {
					depContent, err := os.ReadFile(flapFile)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", flapFile, err)
						continue
					}

					depParser := NewParserWithFilename(string(depContent), flapFile)
					depProgram := depParser.ParseProgram()

					// Prepend dependency program to main program (dependencies must be defined before use)
					program.Statements = append(depProgram.Statements, program.Statements...)
					// Prepend dependency source to combined source
					combinedSource = string(depContent) + "\n" + combinedSource
					fmt.Fprintf(os.Stderr, "Loaded %s from %s\n", flapFile, repoURL)
				}
			}
		}
	}
	// Append main file source
	combinedSource = combinedSource + string(content)

	// Compile
	compiler, err := NewFlapCompiler(MachineX86_64)
	if err != nil {
		return fmt.Errorf("failed to create compiler: %v", err)
	}
	compiler.sourceCode = combinedSource

	err = compiler.Compile(program, outputPath)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	return nil
}
