package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"unicode"
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

	// Skip comments (lines starting with #)
	if l.pos < len(l.input) && l.input[l.pos] == '#' {
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

	// Identifier
	if unicode.IsLetter(rune(ch)) || ch == '_' {
		start := l.pos
		for l.pos < len(l.input) && (unicode.IsLetter(rune(l.input[l.pos])) || unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '_') {
			l.pos++
		}
		value := l.input[start:l.pos]
		return Token{Type: TOKEN_IDENT, Value: value, Line: l.line}
	}

	// Operators and punctuation
	switch ch {
	case '+':
		l.pos++
		return Token{Type: TOKEN_PLUS, Value: "+", Line: l.line}
	case '-':
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
		l.pos++
		return Token{Type: TOKEN_EQUALS, Value: "=", Line: l.line}
	case '(':
		l.pos++
		return Token{Type: TOKEN_LPAREN, Value: "(", Line: l.line}
	case ')':
		l.pos++
		return Token{Type: TOKEN_RPAREN, Value: ")", Line: l.line}
	case ',':
		l.pos++
		return Token{Type: TOKEN_COMMA, Value: ",", Line: l.line}
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

func (p *Parser) parseExpression() Expression {
	return p.parseAdditive()
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
	left := p.parsePrimary()

	for p.peek.Type == TOKEN_STAR || p.peek.Type == TOKEN_SLASH {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parsePrimary()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parsePrimary() Expression {
	switch p.current.Type {
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
		p.nextToken() // skip '('
		expr := p.parseExpression()
		p.nextToken() // skip ')'
		return expr
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
	}, nil
}

func (fc *FlapCompiler) Compile(program *Program, outputPath string) error {
	// Add format strings for printf
	fc.eb.Define("fmt_str", "%s\x00")
	fc.eb.Define("fmt_int", "%ld\n\x00")
	fc.eb.Define("fmt_float", "%.0f\n\x00") // Print float without decimal places

	// Generate code
	// Initialize registers
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rdi", "rdi")
	fc.out.XorRegWithReg("rsi", "rsi")

	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

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
	fc.callOrder = []string{} // Clear call order for recompilation
	fc.stringCounter = 0      // Reset string counter for recompilation
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rdi", "rdi")
	fc.out.XorRegWithReg("rsi", "rsi")

	// Recompile with correct addresses
	parser := NewParser(fc.sourceCode)
	program := parser.ParseProgram()
	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
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
		_, exists := fc.variables[s.Name]

		if exists {
			// Variable exists - check if it's mutable
			if !fc.mutableVars[s.Name] {
				fmt.Fprintf(os.Stderr, "Error: cannot reassign const variable '%s'\n", s.Name)
				os.Exit(1)
			}
			// It's mutable, allow reassignment
		} else {
			// First assignment - record mutability
			fc.mutableVars[s.Name] = s.Mutable
		}

		// Evaluate expression into xmm0
		fc.compileExpression(s.Value)
		// For now, keep in xmm0 (in full compiler, would push to stack)
		fc.variables[s.Name] = 0

	case *ExpressionStmt:
		fc.compileExpression(s.Expr)
	}
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
		// Variable is in xmm0 from previous assignment (float64)
		// No operation needed, value already there

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
		// Perform SIMD operation (operates on float64 values)
		switch e.Operator {
		case "+":
			fc.out.AddpdXmm("xmm0", "xmm1") // addpd xmm0, xmm1
		case "-":
			fc.out.SubpdXmm("xmm0", "xmm1") // subpd xmm0, xmm1
		case "*":
			fc.out.MulpdXmm("xmm0", "xmm1") // mulpd xmm0, xmm1
		case "/":
			fc.out.DivpdXmm("xmm0", "xmm1") // divpd xmm0, xmm1
		}

	case *CallExpr:
		fc.compileCall(e)
	}
}

func (fc *FlapCompiler) compileCall(call *CallExpr) {
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
