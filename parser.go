package main

import (
	"fmt"
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
	TOKEN_NEWLINE
	TOKEN_LT       // <
	TOKEN_GT       // >
	TOKEN_LE       // <=
	TOKEN_GE       // >=
	TOKEN_EQ       // ==
	TOKEN_NE       // !=
	TOKEN_IF       // if keyword
	TOKEN_ELSE     // else keyword
	TOKEN_END      // end keyword
	TOKEN_AT       // @
	TOKEN_IN       // in keyword
	TOKEN_LBRACE   // {
	TOKEN_RBRACE   // }
	TOKEN_LBRACKET // [
	TOKEN_RBRACKET // ]
	TOKEN_ARROW    // ->
	TOKEN_PIPE     // |
	TOKEN_PIPEPIPE // ||
	TOKEN_HASH     // #
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
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
		case "if":
			return Token{Type: TOKEN_IF, Value: value, Line: l.line}
		case "else":
			return Token{Type: TOKEN_ELSE, Value: value, Line: l.line}
		case "end":
			return Token{Type: TOKEN_END, Value: value, Line: l.line}
		case "in":
			return Token{Type: TOKEN_IN, Value: value, Line: l.line}
		}

		return Token{Type: TOKEN_IDENT, Value: value, Line: l.line}
	}

	// Operators and punctuation
	switch ch {
	case '+':
		l.pos++
		return Token{Type: TOKEN_PLUS, Value: "+", Line: l.line}
	case '-':
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
		// Otherwise, just skip it (part of type annotation, handled separately)
		l.pos++
		return l.NextToken()
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
		// Check for ||
		if l.peek() == '|' {
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

type IfStmt struct {
	Condition Expression
	ThenBlock []Statement
	ElseBlock []Statement
}

func (i *IfStmt) String() string {
	var out strings.Builder
	out.WriteString("if ")
	out.WriteString(i.Condition.String())
	out.WriteString("\n")
	for _, stmt := range i.ThenBlock {
		out.WriteString("  ")
		out.WriteString(stmt.String())
		out.WriteString("\n")
	}
	if len(i.ElseBlock) > 0 {
		out.WriteString("else\n")
		for _, stmt := range i.ElseBlock {
			out.WriteString("  ")
			out.WriteString(stmt.String())
			out.WriteString("\n")
		}
	}
	out.WriteString("end")
	return out.String()
}
func (i *IfStmt) statementNode() {}

type LoopStmt struct {
	Iterator string     // Variable name (e.g., "i")
	Iterable Expression // Expression to iterate over (e.g., range(10))
	Body     []Statement
}

func (l *LoopStmt) String() string {
	var out strings.Builder
	out.WriteString("@ ")
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

type LengthExpr struct {
	Operand Expression
}

func (l *LengthExpr) String() string {
	return "#" + l.Operand.String()
}
func (l *LengthExpr) expressionNode() {}

// Parser for Flap language
type Parser struct {
	lexer   *Lexer
	current Token
	peek    Token
}

func NewParser(input string) *Parser {
	p := &Parser{lexer: NewLexer(input)}
	p.nextToken()
	p.nextToken()
	return p
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

	return program
}

func (p *Parser) parseStatement() Statement {
	// Check for loop statement
	if p.current.Type == TOKEN_AT {
		return p.parseLoopStatement()
	}

	// Check for if statement
	if p.current.Type == TOKEN_IF {
		return p.parseIfStatement()
	}

	// Check for assignment (both = and :=)
	if p.current.Type == TOKEN_IDENT && (p.peek.Type == TOKEN_EQUALS || p.peek.Type == TOKEN_COLON_EQUALS) {
		return p.parseAssignment()
	}

	// Otherwise, it's an expression statement
	expr := p.parseExpression()
	if expr != nil {
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

func (p *Parser) parseIfStatement() *IfStmt {
	p.nextToken() // skip 'if'

	condition := p.parseExpression()

	// Skip newlines after condition
	for p.peek.Type == TOKEN_NEWLINE {
		p.nextToken()
	}

	// Parse then block
	var thenBlock []Statement
	for p.peek.Type != TOKEN_ELSE && p.peek.Type != TOKEN_END && p.peek.Type != TOKEN_EOF {
		p.nextToken()
		if p.current.Type == TOKEN_NEWLINE {
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			thenBlock = append(thenBlock, stmt)
		}
	}

	// Parse else block if present
	var elseBlock []Statement
	if p.peek.Type == TOKEN_ELSE {
		p.nextToken() // skip to 'else'

		// Skip newlines after else keyword (check peek, not current)
		for p.peek.Type == TOKEN_NEWLINE {
			p.nextToken()
		}

		for p.peek.Type != TOKEN_END && p.peek.Type != TOKEN_EOF {
			p.nextToken()
			if p.current.Type == TOKEN_NEWLINE {
				continue
			}
			stmt := p.parseStatement()
			if stmt != nil {
				elseBlock = append(elseBlock, stmt)
			}
		}
	}

	// Skip to 'end'
	for p.peek.Type != TOKEN_END && p.peek.Type != TOKEN_EOF {
		p.nextToken()
	}
	p.nextToken() // skip to 'end'

	return &IfStmt{
		Condition: condition,
		ThenBlock: thenBlock,
		ElseBlock: elseBlock,
	}
}

func (p *Parser) parseLoopStatement() *LoopStmt {
	p.nextToken() // skip '@'

	// Expect identifier for iterator variable
	if p.current.Type != TOKEN_IDENT {
		fmt.Fprintf(os.Stderr, "Error: expected identifier after @ in loop\n")
		return nil
	}
	iterator := p.current.Value

	p.nextToken() // skip identifier

	// Expect 'in' keyword
	if p.current.Type != TOKEN_IN {
		fmt.Fprintf(os.Stderr, "Error: expected 'in' in loop statement\n")
		return nil
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
		fmt.Fprintf(os.Stderr, "Error: expected '{' in loop statement\n")
		return nil
	}
	p.nextToken() // advance to '{'

	// Skip newlines after '{'
	for p.peek.Type == TOKEN_NEWLINE {
		p.nextToken()
	}

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
		Iterator: iterator,
		Iterable: iterable,
		Body:     body,
	}
}

func (p *Parser) parseExpression() Expression {
	return p.parseParallel()
}

func (p *Parser) parseParallel() Expression {
	left := p.parseComparison()

	for p.peek.Type == TOKEN_PIPEPIPE {
		p.nextToken() // skip current
		p.nextToken() // skip '||'
		right := p.parseComparison()
		left = &ParallelExpr{List: left, Operation: right}
	}

	return left
}

func (p *Parser) parseComparison() Expression {
	left := p.parseAdditive()

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

func (p *Parser) parseAdditive() Expression {
	left := p.parseMultiplicative()

	for p.peek.Type == TOKEN_PLUS || p.peek.Type == TOKEN_MINUS {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseMultiplicative()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseMultiplicative() Expression {
	left := p.parsePostfix()

	for p.peek.Type == TOKEN_STAR || p.peek.Type == TOKEN_SLASH {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parsePostfix()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parsePostfix() Expression {
	expr := p.parsePrimary()

	// Handle postfix operations like indexing
	for p.peek.Type == TOKEN_LBRACKET {
		p.nextToken() // skip current expr
		p.nextToken() // skip '['
		index := p.parseExpression()
		p.nextToken() // move to ']'
		expr = &IndexExpr{List: expr, Index: index}
	}

	return expr
}

func (p *Parser) parsePrimary() Expression {
	switch p.current.Type {
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
				body := p.parseExpression()
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
					body := p.parseExpression()
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
						fmt.Fprintf(os.Stderr, "Error: expected parameter name in lambda\n")
						os.Exit(1)
					}
					params = append(params, p.current.Value)
					p.nextToken() // skip param
				}

				// current should be ')'
				if p.current.Type != TOKEN_RPAREN {
					fmt.Fprintf(os.Stderr, "Error: expected ')' after lambda parameters\n")
					os.Exit(1)
				}

				// peek should be '->'
				if p.peek.Type != TOKEN_ARROW {
					fmt.Fprintf(os.Stderr, "Error: expected '->' after lambda parameters\n")
					os.Exit(1)
				}

				p.nextToken() // skip ')'
				p.nextToken() // skip '->'
				body := p.parseExpression()
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
	}

	return nil
}

// Code Generator for Flap
type FlapCompiler struct {
	eb            *ExecutableBuilder
	out           *Out
	variables     map[string]int  // variable name -> stack offset
	mutableVars   map[string]bool // variable name -> is mutable
	sourceCode    string          // Store source for recompilation
	usedFunctions map[string]bool // Track which functions are called
	callOrder     []string        // Track order of function calls
	stringCounter int             // Counter for unique string labels
	stackOffset   int             // Current stack offset for variables
	labelCounter  int             // Counter for unique labels (if/else, loops, etc)
	lambdaCounter int             // Counter for unique lambda function names
	lambdaFuncs   []LambdaFunc    // List of lambda functions to generate
	lambdaOffsets map[string]int  // Lambda name -> offset in .text
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
		eb:            eb,
		out:           out,
		variables:     make(map[string]int),
		mutableVars:   make(map[string]bool),
		usedFunctions: make(map[string]bool),
		callOrder:     []string{},
		lambdaOffsets: make(map[string]int),
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

	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

	// Generate lambda functions
	fc.generateLambdaFunctions()

	// Write ELF using existing infrastructure
	return fc.writeELF(outputPath)
}

func (fc *FlapCompiler) writeELF(outputPath string) error {
	// WORKAROUND: Always use printf and exit in fixed order for PLT
	// to maintain consistent PLT size and avoid _start jump offset bugs
	// We'll generate the correct calls based on callOrder
	pltFunctions := []string{"printf", "exit"}

	// Build mapping from actual calls to PLT indices
	callToPLT := make(map[string]int)
	for i, f := range pltFunctions {
		callToPLT[f] = i
	}

	// Set up dynamic sections
	ds := NewDynamicSections()
	ds.AddNeeded("libc.so.6")

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

	estimatedRodataAddr := uint64(0x403000 + 0x100)
	currentAddr := estimatedRodataAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]
		fc.eb.WriteRodata([]byte(value))
		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
	}

	// Write complete dynamic ELF with fixed PLT functions
	gotBase, rodataBaseAddr, textAddr, pltBase, err := fc.eb.WriteCompleteDynamicELF(ds, pltFunctions)
	if err != nil {
		return err
	}

	// Update rodata addresses using same sorted order
	currentAddr = rodataBaseAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]
		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
	}

	// Regenerate code with correct addresses
	fc.eb.text.Reset()
	fc.eb.pcRelocations = []PCRelocation{}  // Reset PC relocations for recompilation
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

	// Recompile with correct addresses
	parser := NewParser(fc.sourceCode)
	program := parser.ParseProgram()
	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

	// Generate lambda functions
	fc.generateLambdaFunctions()

	// Set lambda function addresses
	for lambdaName, offset := range fc.lambdaOffsets {
		lambdaAddr := textAddr + uint64(offset)
		fc.eb.DefineAddr(lambdaName, lambdaAddr)
	}

	// Patch PLT calls using callOrder (actual calls) mapped to pltFunctions positions
	fc.eb.patchPLTCalls(ds, textAddr, pltBase, fc.callOrder)

	// Patch PC-relative relocations
	rodataSize := fc.eb.rodata.Len()
	fc.eb.PatchPCRelocations(textAddr, rodataBaseAddr, rodataSize)

	// Update ELF with regenerated code
	fc.eb.patchTextInELF()

	// Output the executable file
	if err := os.WriteFile(outputPath, fc.eb.Bytes(), 0o755); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Final GOT base: 0x%x\n", gotBase)
	return nil
}

func (fc *FlapCompiler) compileStatement(stmt Statement) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Check if variable already exists
		offset, exists := fc.variables[s.Name]

		if exists {
			// Variable exists - check if it's mutable
			if !fc.mutableVars[s.Name] {
				fmt.Fprintf(os.Stderr, "Error: cannot reassign const variable '%s'\n", s.Name)
				os.Exit(1)
			}
			// It's mutable, allow reassignment
		} else {
			// First assignment - allocate stack space (16 bytes for alignment)
			fc.stackOffset += 16
			offset = fc.stackOffset
			fc.variables[s.Name] = offset
			fc.mutableVars[s.Name] = s.Mutable
			// Actually allocate the stack space (16 bytes to maintain alignment)
			fc.out.SubImmFromReg("rsp", 16)
		}

		// Evaluate expression into xmm0
		fc.compileExpression(s.Value)
		// Store xmm0 to stack at variable's offset
		// movsd [rbp - offset], xmm0
		fc.out.MovXmmToMem("xmm0", "rbp", -offset)

	case *IfStmt:
		fc.compileIfStatement(s)

	case *LoopStmt:
		fc.compileLoopStatement(s)

	case *ExpressionStmt:
		fc.compileExpression(s.Expr)
	}
}

func (fc *FlapCompiler) compileIfStatement(stmt *IfStmt) {
	// Compile condition - this will set flags via ucomisd for comparisons
	fc.compileExpression(stmt.Condition)

	// Increment label counter for uniqueness
	fc.labelCounter++

	// Extract the comparison operator from the condition
	// For floating-point comparisons (ucomisd), use unsigned conditions
	var jumpCond JumpCondition
	if binExpr, ok := stmt.Condition.(*BinaryExpr); ok {
		switch binExpr.Operator {
		case "<":
			jumpCond = JumpAboveOrEqual // Jump to else if NOT below (i.e., >=)
		case "<=":
			jumpCond = JumpAbove // Jump to else if NOT below or equal (i.e., >)
		case ">":
			jumpCond = JumpBelowOrEqual // Jump to else if NOT above (i.e., <=)
		case ">=":
			jumpCond = JumpBelow // Jump to else if NOT above or equal (i.e., <)
		case "==":
			jumpCond = JumpNotEqual // Jump to else if NOT equal
		case "!=":
			jumpCond = JumpEqual // Jump to else if equal (NOT not-equal)
		default:
			jumpCond = JumpEqual // Default fallback
		}
	} else {
		jumpCond = JumpEqual
	}

	// Emit conditional jump with placeholder offset
	elseJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(jumpCond, 0) // Use 0 as placeholder

	// Compile then block
	for _, s := range stmt.ThenBlock {
		fc.compileStatement(s)
	}

	// If there's an else block, emit unconditional jump to end
	var endJumpPos int
	if len(stmt.ElseBlock) > 0 {
		endJumpPos = fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Placeholder
	}

	// Mark else label position (where we are now)
	elseLabelPos := fc.eb.text.Len()

	// NOW patch the conditional jump - we know where else starts
	targetPos := elseLabelPos
	if len(stmt.ElseBlock) == 0 {
		// No else block, so conditional jump goes to end (which is here)
		targetPos = elseLabelPos
	}
	elseOffset := int32(targetPos - (elseJumpPos + 6)) // 6 bytes for conditional jump
	fmt.Fprintf(os.Stderr, "DEBUG: Patching conditional jump at %d to target %d, offset=%d\n", elseJumpPos, targetPos, elseOffset)
	fc.patchJumpImmediate(elseJumpPos+2, elseOffset) // +2 to skip 0F 8x

	// Compile else block if present
	if len(stmt.ElseBlock) > 0 {
		for _, s := range stmt.ElseBlock {
			fc.compileStatement(s)
		}
	}

	// Mark end label position
	endLabelPos := fc.eb.text.Len()

	// Patch the unconditional jump to end (if we had an else block)
	if len(stmt.ElseBlock) > 0 {
		endOffset := int32(endLabelPos - (endJumpPos + 5)) // 5 bytes for unconditional jump
		fmt.Fprintf(os.Stderr, "DEBUG: Patching unconditional jump at %d to target %d, offset=%d\n", endJumpPos, endLabelPos, endOffset)
		fc.patchJumpImmediate(endJumpPos+1, endOffset) // +1 to skip E9
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

	// Patch the conditional jump to loop end
	endOffset := int32(loopEndPos - (loopEndJumpPos + 6)) // 6 bytes for conditional jump
	fmt.Fprintf(os.Stderr, "DEBUG LOOP: Patching conditional jump at %d to target %d, offset=%d\n", loopEndJumpPos, loopEndPos, endOffset)
	fc.patchJumpImmediate(loopEndJumpPos+2, endOffset) // +2 to skip 0F 8x
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

	// Load index: mov rax, [rbp - indexOffset]
	fc.out.MovMemToReg("rax", "rbp", -indexOffset)

	// Load length: mov rdi, [rbp - lengthOffset]
	fc.out.MovMemToReg("rdi", "rbp", -lengthOffset)

	// Compare index with length: cmp rax, rdi
	fc.out.CmpRegToReg("rax", "rdi")

	// Jump to loop end if index >= length
	loopEndJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0) // Placeholder

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

	// Patch the conditional jump to loop end
	endOffset := int32(loopEndPos - (loopEndJumpPos + 6)) // 6 bytes for conditional jump
	fc.patchJumpImmediate(loopEndJumpPos+2, endOffset)    // +2 to skip 0F 8x
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

func (fc *FlapCompiler) compileExpression(expr Expression) {
	switch e := expr.(type) {
	case *NumberExpr:
		// Flap uses float64 foundation - all values are float64
		// Convert to int64 first, then to float64 in xmm0
		val := int64(e.Value)
		fc.out.MovImmToReg("rax", strconv.FormatInt(val, 10))
		// Convert integer to float64: cvtsi2sd xmm0, rax
		fc.out.Cvtsi2sd("xmm0", "rax")

	case *StringExpr:
		// Store string and load address (strings still use pointers for now)
		labelName := fmt.Sprintf("str_%d", fc.stringCounter)
		fc.stringCounter++
		fc.eb.Define(labelName, e.Value+"\x00")
		fc.out.LeaSymbolToReg("rax", labelName)

	case *IdentExpr:
		// Load variable from stack into xmm0
		offset, exists := fc.variables[e.Name]
		if !exists {
			fmt.Fprintf(os.Stderr, "Error: undefined variable '%s'\n", e.Name)
			os.Exit(1)
		}
		// movsd xmm0, [rbp - offset]
		fc.out.MovMemToXmm("xmm0", "rbp", -offset)

	case *BinaryExpr:
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
		}

	case *CallExpr:
		fc.compileCall(e)

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
					fmt.Fprintf(os.Stderr, "Error: list elements must be constant numbers for now\n")
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

	case *IndexExpr:
		// Compile list expression (returns pointer as float64 in xmm0)
		fc.compileExpression(e.List)
		// Save list pointer to stack
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Compile index expression (returns index as float64 in xmm0)
		fc.compileExpression(e.Index)

		// Convert index from float64 to integer in rax
		fc.out.Cvttsd2si("rax", "xmm0") // truncate float to int

		// Load list pointer from stack to rbx
		fc.out.MovMemToXmm("xmm1", "rsp", 0)
		// Convert pointer from float64 back to integer in rbx
		fc.out.MovXmmToMem("xmm1", "rsp", 8)
		fc.out.MovMemToReg("rbx", "rsp", 8)
		fc.out.AddImmToReg("rsp", 16)

		// Skip the length prefix (first 8 bytes)
		fc.out.AddImmToReg("rbx", 8)

		// Calculate offset: rax * 8 (each float64 is 8 bytes)
		fc.out.MulRegWithImm("rax", 8) // rax = rax * 8

		// Add offset to base pointer: rbx = rbx + rax
		fc.out.AddRegToReg("rbx", "rax")

		// Load float64 from [rbx] into xmm0
		fc.out.MovMemToXmm("xmm0", "rbx", 0)

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

	case *ParallelExpr:
		fc.compileParallelExpr(e)
	}
}

func (fc *FlapCompiler) compileParallelExpr(expr *ParallelExpr) {
	// For now, only support: list || lambda
	lambda, ok := expr.Operation.(*LambdaExpr)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: parallel operator currently only supports lambda expressions\n")
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
	fc.out.XorRegWithReg("r12", "r12")
	finalizeJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	finalizeJumpEnd := fc.eb.text.Len()

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

	// Finalize: remove lambda/list pointers and return the result pointer as float64
	finalizeLabel := fc.eb.text.Len()

	// Clean up: remove lambda pointer and list pointer from stack
	fc.out.AddImmToReg("rsp", 16)

	// Return result list pointer as float64 in xmm0
	fc.out.SubImmFromReg("rsp", 8)
	fc.out.MovRegToMem("r12", "rsp", 0)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.AddImmToReg("rsp", 8)

	// Note: result list (parallelResultAlloc bytes) is left on stack
	// This is a memory leak for now, but works for simple programs

	// Patch jumps for the null-input fast path
	nonNullOffset := int32(nonNullListStart - (nonNullJumpPos + 6))
	fc.patchJumpImmediate(nonNullJumpPos+2, nonNullOffset)

	finalizeOffset := int32(finalizeLabel - finalizeJumpEnd)
	fc.patchJumpImmediate(finalizeJumpPos+1, finalizeOffset)
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

		// Compile lambda body (result in xmm0)
		fc.compileExpression(lambda.Body)

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
	// then SubImmFromReg 16 for each arg, then AddImmFromReg 16 for each arg,
	// then AddImmFromReg 16 for func ptr, so we're back to original alignment

	// Call the function pointer in r11
	fc.out.CallRegister("r11")

	// Result is in xmm0
}

func (fc *FlapCompiler) compileCall(call *CallExpr) {
	// Check if this is a stored function (variable containing function pointer)
	if _, isVariable := fc.variables[call.Function]; isVariable {
		fc.compileStoredFunctionCall(call)
		return
	}

	// Otherwise, handle builtin functions
	switch call.Function {
	case "println":
		if len(call.Args) == 0 {
			return
		}

		arg := call.Args[0]
		if strExpr, ok := arg.(*StringExpr); ok {
			// Print string with newline
			labelName := fmt.Sprintf("str_%d", fc.stringCounter)
			fc.stringCounter++
			fc.eb.Define(labelName, strExpr.Value+"\n\x00")
			fc.out.LeaSymbolToReg("rdi", labelName)
			fc.out.XorRegWithReg("rax", "rax")
			fc.trackFunctionCall("printf")
			fc.eb.GenerateCallInstruction("printf")
		} else {
			// Print number with newline
			fc.compileExpression(arg)
			// xmm0 contains float64 value
			// For printf %f, float64 goes in xmm0, and rax=1 (1 vector register used)
			fc.out.LeaSymbolToReg("rdi", "fmt_float")
			fc.out.MovImmToReg("rax", "1") // 1 vector register used
			fc.trackFunctionCall("printf")
			fc.eb.GenerateCallInstruction("printf")
		}

	case "len":
		if len(call.Args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: len() requires exactly one argument\n")
			os.Exit(1)
		}

		// Compile the list expression (returns pointer as float64 in xmm0)
		fc.compileExpression(call.Args[0])

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
	}
}

func (fc *FlapCompiler) trackFunctionCall(funcName string) {
	if !fc.usedFunctions[funcName] {
		fc.usedFunctions[funcName] = true
	}
	fc.callOrder = append(fc.callOrder, funcName)
}

func CompileFlap(inputPath string, outputPath string) error {
	// Read input file
	content, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", inputPath, err)
	}

	// Parse
	parser := NewParser(string(content))
	program := parser.ParseProgram()

	fmt.Fprintf(os.Stderr, "Parsed program:\n%s\n", program.String())

	// Compile
	compiler, err := NewFlapCompiler(MachineX86_64)
	if err != nil {
		return fmt.Errorf("failed to create compiler: %v", err)
	}
	compiler.sourceCode = string(content)

	err = compiler.Compile(program, outputPath)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	return nil
}
