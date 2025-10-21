package main

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"unsafe"
)

// hashStringKey hashes a string identifier to a uint64 for use as a map key.
// Uses FNV-1a hash algorithm for deterministic, collision-resistant hashing.
// Currently limited to 30-bit hash due to compiler integer literal limitations.
// Sets bit 30 to distinguish symbolic keys from typical numeric indices.
func hashStringKey(s string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(s))
	// Use FNV-1a 32-bit variant for now, mask to 30 bits (0x3FFFFFFF)
	// Then set bit 30 (0x40000000) to distinguish symbolic keys
	// This gives us range 0x40000000 to 0x7FFFFFFF (1073741824 to 2147483647)
	h32 := fnv.New32a()
	h32.Write([]byte(s))
	return uint64((h32.Sum32() & 0x3FFFFFFF) | 0x40000000)
}

// isUppercase checks if an identifier is all uppercase (constant naming convention)
func isUppercase(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch >= 'a' && ch <= 'z' {
			return false
		}
	}
	return true
}

// parseNumberLiteral parses a number literal which can be decimal, hex (0x...), or binary (0b...)
func (p *Parser) parseNumberLiteral(s string) float64 {
	if len(s) >= 2 {
		prefix := s[0:2]
		if prefix == "0x" || prefix == "0X" {
			// Hexadecimal
			val, err := strconv.ParseUint(s[2:], 16, 64)
			if err != nil {
				p.error(fmt.Sprintf("invalid hexadecimal literal: %s", s))
			}
			return float64(val)
		} else if prefix == "0b" || prefix == "0B" {
			// Binary
			val, err := strconv.ParseUint(s[2:], 2, 64)
			if err != nil {
				p.error(fmt.Sprintf("invalid binary literal: %s", s))
			}
			return float64(val)
		}
	}
	// Regular decimal number
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

type Parser struct {
	lexer     *Lexer
	current   Token
	peek      Token
	filename  string
	source    string
	loopDepth int                   // Current loop nesting level (0 = not in loop, 1 = outer loop, etc.)
	constants map[string]Expression // Compile-time constants (immutable literals)
}

func NewParser(input string) *Parser {
	p := &Parser{
		lexer:     NewLexer(input),
		filename:  "<input>",
		source:    input,
		constants: make(map[string]Expression),
	}
	p.nextToken()
	p.nextToken()
	return p
}

func NewParserWithFilename(input, filename string) *Parser {
	p := &Parser{
		lexer:     NewLexer(input),
		filename:  filename,
		source:    input,
		constants: make(map[string]Expression),
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

// error prints a formatted error and panics (to be recovered by CompileFlap)
func (p *Parser) error(msg string) {
	errMsg := p.formatError(p.current.Line, msg)
	if VerboseMode {
		fmt.Fprintln(os.Stderr, errMsg)
	}
	panic(fmt.Errorf("%s", errMsg))
}

// compilerError prints an error message and panics (to be recovered by CompileFlap)
// Use this instead of fmt.Fprintf + os.Exit in code generation
func compilerError(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if VerboseMode {
		fmt.Fprintln(os.Stderr, "Error:", msg)
	}
	panic(fmt.Errorf("%s", msg))
}

func (p *Parser) nextToken() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

func (p *Parser) skipNewlines() {
	for p.current.Type == TOKEN_NEWLINE || p.current.Type == TOKEN_SEMICOLON {
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

	// Don't add automatic exit(0) statement - the compiler will emit exit code
	// after processing deferred statements (see lines 2658-2669 in compileStatement)

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

func (p *Parser) parseImport() Statement {
	p.nextToken() // skip 'import'

	// Auto-detect import type:
	// - String ending with ".so" -> C library .so file: import "/path/to/lib.so" as alias
	// - String with "/" -> Flap package (Git): import "github.com/user/pkg" as alias
	// - Identifier -> C library: import sdl3 as sdl, import raylib as rl

	if p.current.Type == TOKEN_STRING {
		value := p.current.Value

		// Check if this is a .so file import
		if strings.HasSuffix(value, ".so") || strings.Contains(value, ".so.") {
			// C library .so file import: import "/tmp/libmylib.so" as mylib
			p.nextToken()

			if p.current.Type != TOKEN_AS {
				p.error("expected 'as' after .so file path")
			}
			p.nextToken()

			if p.current.Type != TOKEN_IDENT {
				p.error("expected alias after 'as'")
			}
			alias := p.current.Value
			p.nextToken()

			// Extract just the filename from the path
			soPath := value
			soFilename := soPath
			if lastSlash := strings.LastIndex(soPath, "/"); lastSlash != -1 {
				soFilename = soPath[lastSlash+1:]
			}

			return &CImportStmt{Library: soFilename, Alias: alias, SoPath: soPath}
		}

		// Git import: import "url@version" as alias
		urlWithVersion := value
		p.nextToken()

		// Parse URL and optional version (URL@version)
		url := urlWithVersion
		version := ""
		if atIndex := strings.LastIndex(urlWithVersion, "@"); atIndex != -1 {
			url = urlWithVersion[:atIndex]
			version = urlWithVersion[atIndex+1:]
		}

		if p.current.Type != TOKEN_AS {
			p.error("expected 'as' after import URL")
		}
		p.nextToken()

		if p.current.Type != TOKEN_IDENT && p.current.Type != TOKEN_STAR {
			p.error("expected alias or '*' after 'as'")
		}
		alias := p.current.Value
		p.nextToken()

		return &ImportStmt{URL: url, Version: version, Alias: alias}
	}

	if p.current.Type == TOKEN_IDENT {
		// C library import: import sdl3 as sdl, import raylib as rl
		libName := p.current.Value
		p.nextToken()

		if p.current.Type != TOKEN_AS {
			p.error("expected 'as' after library name")
		}
		p.nextToken()

		if p.current.Type != TOKEN_IDENT {
			p.error("expected alias after 'as'")
		}
		alias := p.current.Value
		p.nextToken()

		return &CImportStmt{Library: libName, Alias: alias}
	}

	p.error("expected library name or git URL string after 'import'")
	return nil
}

func (p *Parser) parseArenaStmt() *ArenaStmt {
	p.nextToken() // skip 'arena'

	if p.current.Type != TOKEN_LBRACE {
		p.error("expected '{' after 'arena'")
	}
	p.nextToken() // skip '{'
	p.skipNewlines()

	var body []Statement
	for p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
		p.nextToken()
		p.skipNewlines()
	}

	if p.current.Type != TOKEN_RBRACE {
		p.error("expected '}' at end of arena block")
	}

	return &ArenaStmt{Body: body}
}

func (p *Parser) parseDeferStmt() *DeferStmt {
	p.nextToken() // skip 'defer'

	// Parse the expression to be deferred (typically a function call)
	expr := p.parseExpression()
	if expr == nil {
		p.error("expected expression after 'defer'")
	}

	return &DeferStmt{Call: expr}
}

func (p *Parser) parseStatement() Statement {
	// Check for use keyword (imports)
	if p.current.Type == TOKEN_USE {
		p.nextToken() // skip 'use'
		if p.current.Type != TOKEN_STRING {
			p.error("expected string after 'use'")
		}
		path := p.current.Value
		return &UseStmt{Path: path}
	}

	// Check for import keyword (git URL imports)
	if p.current.Type == TOKEN_IMPORT {
		return p.parseImport()
	}

	// Check for arena keyword
	if p.current.Type == TOKEN_ARENA {
		return p.parseArenaStmt()
	}

	// Check for defer keyword
	if p.current.Type == TOKEN_DEFER {
		return p.parseDeferStmt()
	}

	// Check for ret keyword
	if p.current.Type == TOKEN_RET {
		return p.parseJumpStatement()
	}

	// Check for @- (jump to @(N-1))
	if p.current.Type == TOKEN_AT_MINUS {
		return p.parseLoopStatement()
	}

	// Check for @= (continue current loop)
	if p.current.Type == TOKEN_AT_EQUALS {
		return p.parseLoopStatement()
	}

	// Check for @ (loop)

	// Check for @ (either loop @N, loop @ ident, or jump @N)
	if p.current.Type == TOKEN_AT {
		// Look ahead to distinguish loop vs jump
		// Loop: @N identifier in ... or @ identifier in ...
		// Jump: @N (followed by newline, semicolon, or })
		if p.peek.Type == TOKEN_NUMBER || p.peek.Type == TOKEN_IDENT {
			// We need to peek further to distinguish loop from jump
			// For now, let's just parse as loop if it matches the pattern
			// Otherwise treat as jump

			// Simple heuristic: if @ NUMBER IDENTIFIER or @ IDENTIFIER, it's a loop
			// We can't easily look 2 tokens ahead, so we'll just try parsing as loop first
			return p.parseLoopStatement()
		}
		p.error("expected number or identifier after @ (e.g., @1 i in..., @ i in...)")
	}

	// Check for assignment (=, :=, ==>, <-, with optional type annotation, and compound assignments)
	if p.current.Type == TOKEN_IDENT && (p.peek.Type == TOKEN_EQUALS || p.peek.Type == TOKEN_EQUALS_FAT_ARROW || p.peek.Type == TOKEN_COLON_EQUALS || p.peek.Type == TOKEN_LEFT_ARROW || p.peek.Type == TOKEN_COLON ||
		p.peek.Type == TOKEN_PLUS_EQUALS || p.peek.Type == TOKEN_MINUS_EQUALS ||
		p.peek.Type == TOKEN_STAR_EQUALS || p.peek.Type == TOKEN_SLASH_EQUALS || p.peek.Type == TOKEN_MOD_EQUALS) {
		return p.parseAssignment()
	}

	// Otherwise, it's an expression statement (or match expression)
	expr := p.parseExpression()
	if expr != nil {
		if p.peek.Type == TOKEN_LBRACE {
			p.nextToken() // move to '{'
			p.nextToken() // skip '{'
			p.skipNewlines()
			matchExpr := p.parseMatchBlock(expr)
			return &ExpressionStmt{Expr: matchExpr}
		}

		return &ExpressionStmt{Expr: expr}
	}

	return nil
}

// tryParseNonParenLambda attempts to parse a lambda without parentheses: x => expr or x, y => expr
// Returns nil if current position doesn't look like a lambda
func (p *Parser) tryParseNonParenLambda() Expression {
	if p.current.Type != TOKEN_IDENT {
		return nil
	}

	// Single param: x =>
	firstParam := p.current.Value
	if p.peek.Type == TOKEN_FAT_ARROW {
		p.nextToken() // skip param
		p.nextToken() // skip '=>'
		body := p.parseLambdaBody()
		return &LambdaExpr{Params: []string{firstParam}, Body: body}
	}

	// Error if using -> instead of =>
	if p.peek.Type == TOKEN_ARROW {
		p.error("lambda definitions must use '=>' not '->' (e.g., x => x * 2)")
	}

	// Multi param: x, y, z =>
	// Parameters are comma-separated
	if p.peek.Type != TOKEN_COMMA {
		return nil
	}

	// Collect parameters until we find => or something else
	params := []string{firstParam}

	for p.peek.Type == TOKEN_COMMA {
		p.nextToken() // skip current param
		p.nextToken() // skip ','

		if p.current.Type != TOKEN_IDENT {
			p.error("expected parameter name after ','")
		}

		params = append(params, p.current.Value)

		if p.peek.Type == TOKEN_FAT_ARROW {
			// Found the fat arrow! This is a lambda
			p.nextToken() // skip last param
			p.nextToken() // skip '=>'
			body := p.parseLambdaBody()
			return &LambdaExpr{Params: params, Body: body}
		}

		// Error if using -> instead of =>
		if p.peek.Type == TOKEN_ARROW {
			p.error("lambda definitions must use '=>' not '->' (e.g., x, y => x + y)")
		}
	}

	// We have multiple identifiers separated by commas but no arrow following
	p.error(fmt.Sprintf("expected '=>' after lambda parameters (%s), got %v", strings.Join(params, ", "), p.peek.Type))
	return nil
}

// parseFString parses an f-string and returns an FStringExpr
// F-strings have the format: f"text {expr} more text {expr2}"
// We convert this to alternating string literals and expressions
func (p *Parser) parseFString() Expression {
	raw := p.current.Value // Raw f-string content without f" and "

	var parts []Expression
	currentPos := 0

	for currentPos < len(raw) {
		// Find next {
		nextBrace := -1
		for i := currentPos; i < len(raw); i++ {
			if raw[i] == '{' {
				// Check if it's escaped {{
				if i+1 < len(raw) && raw[i+1] == '{' {
					i++ // Skip the second {
					continue
				}
				nextBrace = i
				break
			}
		}

		// If no more braces, add remaining text as string literal
		if nextBrace == -1 {
			if currentPos < len(raw) {
				text := raw[currentPos:]
				// Process escape sequences and unescape {{  }}
				text = strings.ReplaceAll(text, "{{", "{")
				text = strings.ReplaceAll(text, "}}", "}")
				text = processEscapeSequences(text)
				parts = append(parts, &StringExpr{Value: text})
			}
			break
		}

		// Add text before { as string literal
		if nextBrace > currentPos {
			text := raw[currentPos:nextBrace]
			// Process escape sequences and unescape {{ }}
			text = strings.ReplaceAll(text, "{{", "{")
			text = strings.ReplaceAll(text, "}}", "}")
			text = processEscapeSequences(text)
			parts = append(parts, &StringExpr{Value: text})
		}

		// Find matching }
		braceDepth := 1
		exprStart := nextBrace + 1
		exprEnd := exprStart
		for exprEnd < len(raw) && braceDepth > 0 {
			if raw[exprEnd] == '{' {
				braceDepth++
			} else if raw[exprEnd] == '}' {
				braceDepth--
			}
			if braceDepth > 0 {
				exprEnd++
			}
		}

		if braceDepth != 0 {
			p.error("unclosed { in f-string")
			return &StringExpr{Value: raw}
		}

		// Parse the expression inside {...}
		exprCode := raw[exprStart:exprEnd]
		exprLexer := NewLexer(exprCode)
		exprParser := NewParser(exprCode)
		exprParser.lexer = exprLexer
		exprParser.current = exprLexer.NextToken()
		exprParser.peek = exprLexer.NextToken()

		expr := exprParser.parseExpression()

		parts = append(parts, expr)

		currentPos = exprEnd + 1 // Skip past the }
	}

	// If only one part and it's a string, return a regular StringExpr
	if len(parts) == 1 {
		if strExpr, ok := parts[0].(*StringExpr); ok {
			return strExpr
		}
	}

	return &FStringExpr{Parts: parts}
}

func (p *Parser) parseAssignment() *AssignStmt {
	name := p.current.Value
	p.nextToken() // skip identifier

	// Check for type annotation: name:bNN or name:fNN
	var precision string
	if p.current.Type == TOKEN_COLON && p.peek.Type == TOKEN_IDENT {
		p.nextToken() // skip ':'
		precision = p.current.Value
		// Validate precision format (bNN or fNN where NN is a number)
		if len(precision) < 2 || (precision[0] != 'b' && precision[0] != 'f') {
			p.error("invalid precision format: expected bNN or fNN (e.g., b64, f32)")
		}
		p.nextToken() // skip precision identifier
	}

	// Check for compound assignment operators (+=, -=, *=, /=, %=)
	var compoundOp string
	switch p.current.Type {
	case TOKEN_PLUS_EQUALS:
		compoundOp = "+"
	case TOKEN_MINUS_EQUALS:
		compoundOp = "-"
	case TOKEN_STAR_EQUALS:
		compoundOp = "*"
	case TOKEN_SLASH_EQUALS:
		compoundOp = "/"
	case TOKEN_MOD_EQUALS:
		compoundOp = "%"
	}

	// Determine assignment type
	// := - mutable definition
	// = - immutable definition
	// ==> - immutable definition with lambda (shorthand for = =>)
	// <- - update (requires existing mutable variable)
	isUpdate := p.current.Type == TOKEN_LEFT_ARROW
	mutable := p.current.Type == TOKEN_COLON_EQUALS || isUpdate

	// Handle ==> as shorthand for = =>
	isEqualsArrow := p.current.Type == TOKEN_EQUALS_FAT_ARROW

	p.nextToken() // skip '=' or ':=' or '<-' or '==>' or compound operator

	// Check for non-parenthesized lambda: x -> expr or x y -> expr
	var value Expression

	// If we have ==>, expect lambda parameters or body
	if isEqualsArrow {
		// For ==>, we parse the lambda body directly (no params before =>)
		// Syntax: main ==> { body } is equivalent to main = => { body }
		value = &LambdaExpr{Params: []string{}, Body: p.parseLambdaBody()}
	} else if p.current.Type == TOKEN_IDENT {
		value = p.tryParseNonParenLambda()
		if value == nil {
			value = p.parseExpression()
		}
	} else {
		value = p.parseExpression()
	}

	// Check for match block after expression
	if p.peek.Type == TOKEN_LBRACE {
		p.nextToken() // move to expression
		p.nextToken() // skip '{'
		p.skipNewlines()
		value = p.parseMatchBlock(value)
	}

	// Check for multiple lambda dispatch: f = (x) -> x, (y) -> y + 1
	if lambda, ok := value.(*LambdaExpr); ok && p.peek.Type == TOKEN_COMMA {
		lambdas := []*LambdaExpr{lambda}

		for p.peek.Type == TOKEN_COMMA {
			p.nextToken() // move to comma
			p.nextToken() // skip comma

			// Try non-parenthesized lambda first
			var nextExpr Expression
			if p.current.Type == TOKEN_IDENT {
				nextExpr = p.tryParseNonParenLambda()
				if nextExpr == nil {
					nextExpr = p.parseExpression()
				}
			} else {
				nextExpr = p.parseExpression()
			}

			if nextLambda, ok := nextExpr.(*LambdaExpr); ok {
				lambdas = append(lambdas, nextLambda)
			} else {
				p.error("expected lambda expression after comma in multiple lambda dispatch")
			}
		}

		// Wrap in MultiLambdaExpr
		value = &MultiLambdaExpr{Lambdas: lambdas}
	}

	// Transform compound assignment: x += 5  =>  x = x + 5
	if compoundOp != "" {
		value = &BinaryExpr{
			Left:     &IdentExpr{Name: name},
			Operator: compoundOp,
			Right:    value,
		}
		// Compound assignments are updates
		isUpdate = true
		mutable = true
	}

	// Check if this is a constant definition (uppercase immutable with literal value)
	// Store compile-time constants for substitution
	// Only uppercase identifiers are true constants (cannot be shadowed in practice)
	if !mutable && !isUpdate && isUppercase(name) {
		// Store numbers, strings, and lists as compile-time constants
		switch v := value.(type) {
		case *NumberExpr:
			p.constants[name] = v
		case *StringExpr:
			p.constants[name] = v
		case *ListExpr:
			// Only store lists that contain only literal values
			isLiteral := true
			for _, elem := range v.Elements {
				switch elem.(type) {
				case *NumberExpr, *StringExpr:
					// These are literals, OK
				default:
					// Contains expressions, not a pure literal list
					isLiteral = false
					break
				}
			}
			if isLiteral {
				p.constants[name] = v
			}
		}
	}

	return &AssignStmt{Name: name, Value: value, Mutable: mutable, IsUpdate: isUpdate, Precision: precision}
}

func (p *Parser) parseMatchBlock(condition Expression) *MatchExpr {
	clauses := []*MatchClause{}
	defaultExpr := Expression(&NumberExpr{Value: 0})
	defaultExplicit := false
	sawBareClause := false

	for {
		p.skipNewlines()

		if p.current.Type == TOKEN_RBRACE {
			break
		}

		if p.current.Type == TOKEN_DEFAULT_ARROW {
			if defaultExplicit {
				p.error("duplicate default clause in match block")
			}
			defaultExplicit = true
			p.nextToken() // skip '~>'
			p.skipNewlines()
			defaultExpr = p.parseMatchTarget()
			p.skipNewlines()
			continue
		}

		clause, bare := p.parseMatchClause()
		if bare {
			if sawBareClause || len(clauses) > 0 || defaultExplicit {
				p.error("bare match clause must be the only entry in the block")
			}
			sawBareClause = true
		}
		clauses = append(clauses, clause)
	}

	if p.current.Type != TOKEN_RBRACE {
		p.error("expected '}' after match block")
	}

	if len(clauses) == 0 && !defaultExplicit {
		p.error("match block must contain a clause or default")
	}

	return &MatchExpr{
		Condition:       condition,
		Clauses:         clauses,
		DefaultExpr:     defaultExpr,
		DefaultExplicit: defaultExplicit,
	}
}

func (p *Parser) parseMatchClause() (*MatchClause, bool) {
	// Guardless clause starting with '->'
	if p.current.Type == TOKEN_ARROW {
		p.nextToken() // skip '->'
		p.skipNewlines()
		result := p.parseMatchTarget()
		p.skipNewlines()
		return &MatchClause{Result: result}, false
	}

	guard := p.parseExpression()
	p.nextToken()
	p.skipNewlines()

	if p.current.Type == TOKEN_ARROW {
		p.nextToken() // skip '->'
		p.skipNewlines()
		result := p.parseMatchTarget()
		p.skipNewlines()
		return &MatchClause{Guard: guard, Result: result}, false
	}

	// Bare expression clause (sugar for '-> expr')
	return &MatchClause{Result: guard}, true
}

func (p *Parser) parseMatchTarget() Expression {
	switch p.current.Type {
	case TOKEN_LBRACE:
		// Parse a block of statements as the match target
		// This allows multi-statement match arms like:
		//   condition {
		//       { stmt1; stmt2; stmt3 }
		//   }
		p.nextToken() // skip '{'
		p.skipNewlines()

		var statements []Statement
		for p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
			stmt := p.parseStatement()
			if stmt != nil {
				statements = append(statements, stmt)
			}

			// Skip separators between statements
			if p.peek.Type == TOKEN_NEWLINE || p.peek.Type == TOKEN_SEMICOLON {
				p.nextToken()
				p.skipNewlines()
			} else if p.peek.Type == TOKEN_RBRACE || p.peek.Type == TOKEN_EOF {
				p.nextToken() // move to '}'
				break
			} else {
				p.nextToken()
				p.skipNewlines()
			}
		}

		if p.current.Type != TOKEN_RBRACE {
			p.error("expected '}' at end of match block")
		}

		// Consume the closing '}'
		p.nextToken()

		return &BlockExpr{Statements: statements}

	case TOKEN_RET:
		// ret or ret @N or ret value or ret @N value
		p.nextToken() // skip 'ret'

		label := 0 // 0 means return from function
		var value Expression

		// Check for optional @N
		if p.current.Type == TOKEN_AT {
			p.nextToken() // skip '@'
			if p.current.Type != TOKEN_NUMBER {
				p.error("expected number after @ in ret statement")
			}
			labelNum, err := strconv.ParseFloat(p.current.Value, 64)
			if err != nil {
				p.error("invalid loop label number")
			}
			label = int(labelNum)
			if label < 1 {
				p.error("loop label must be >= 1 (use @1, @2, @3, etc.)")
			}
			p.nextToken() // skip number
		}

		// Check for optional value
		if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF && p.current.Type != TOKEN_DEFAULT_ARROW {
			value = p.parseExpression()
			p.nextToken()
		}

		// Return a JumpExpr with IsBreak semantics (ret exits loop)
		return &JumpExpr{Label: label, Value: value, IsBreak: true}
	case TOKEN_AT_MINUS:
		if p.loopDepth < 2 {
			p.error("@- requires at least 2 nested loops")
		}
		p.nextToken() // skip '@-'
		// Check for optional return value: @- value
		var value Expression
		if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
			value = p.parseExpression()
			p.nextToken()
		}
		return &JumpExpr{Label: p.loopDepth - 1, Value: value, IsBreak: true}
	case TOKEN_AT_EQUALS:
		if p.loopDepth < 1 {
			p.error("@= requires at least 1 loop")
		}
		p.nextToken() // skip '@='
		// Check for optional return value: @= value
		var value Expression
		if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
			value = p.parseExpression()
			p.nextToken()
		}
		return &JumpExpr{Label: p.loopDepth, Value: value, IsBreak: false}
	case TOKEN_AT:
		p.nextToken() // skip '@'
		if p.current.Type != TOKEN_NUMBER {
			p.error("expected number after @ in match block")
		}
		labelNum, err := strconv.ParseFloat(p.current.Value, 64)
		if err != nil {
			p.error("invalid label number")
		}
		label := int(labelNum)
		p.nextToken() // skip label number
		// Check for optional return value: @N value
		var value Expression
		if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
			value = p.parseExpression()
			p.nextToken()
		}
		// @N is continue (jump to top of loop N), not break
		return &JumpExpr{Label: label, Value: value, IsBreak: false}
	default:
		expr := p.parseExpression()

		// Check if this expression has a match block attached
		if p.peek.Type == TOKEN_LBRACE {
			p.nextToken() // move to expr
			p.nextToken() // move to '{'
			p.nextToken() // skip '{'
			p.skipNewlines()
			return p.parseMatchBlock(expr)
		}

		p.nextToken()
		return expr
	}
}

func (p *Parser) parseLoopStatement() Statement {
	// Handle @- token (jump to outer loop)
	if p.current.Type == TOKEN_AT_MINUS {
		// @- means jump to @(N-1) where N is current loop depth (break semantics)
		if p.loopDepth < 2 {
			p.error("@- requires at least 2 nested loops")
		}
		// Check for optional return value: @- value
		var value Expression
		if p.peek.Type != TOKEN_NEWLINE && p.peek.Type != TOKEN_RBRACE && p.peek.Type != TOKEN_EOF {
			p.nextToken() // move to value
			value = p.parseExpression()
		}
		return &JumpStmt{IsBreak: true, Label: p.loopDepth - 1, Value: value}
	}

	// Handle @= token (continue current loop)
	if p.current.Type == TOKEN_AT_EQUALS {
		// @= means continue current loop (jump to @N where N is current loop depth)
		if p.loopDepth < 1 {
			p.error("@= requires at least 1 loop")
		}
		// @= is continue semantics (not break)
		return &JumpStmt{IsBreak: false, Label: p.loopDepth, Value: nil}
	}

	// Handle @ token (start loop at @(N+1))
	if p.current.Type == TOKEN_AT {
		// @ means start a loop at @(N+1) where N is current loop depth
		label := p.loopDepth + 1
		p.nextToken() // skip '@'

		// Check if this is @N (numbered loop) or @ ident (simple loop)
		if p.current.Type == TOKEN_NUMBER {
			// This is @N syntax, handle it in the jump statement section below
			// by re-parsing this TOKEN_AT
			p.current.Type = TOKEN_AT // restore token type (it's already @, but for clarity)
			// Fall through to the jump statement section
			goto handleJump
		}

		// Expect identifier for loop variable
		if p.current.Type != TOKEN_IDENT {
			p.error("expected identifier after @")
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

		// Track loop depth for nested loops
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

		// Expect and consume '}'
		if p.peek.Type != TOKEN_RBRACE {
			p.error("expected '}' at end of loop body")
		}
		p.nextToken() // consume the '}'

		return &LoopStmt{
			Iterator: iterator,
			Iterable: iterable,
			Body:     body,
		}
	}

handleJump:
	// If we reach here, must be @N for a jump statement
	p.nextToken() // skip '@'

	// Expect number for jump label
	if p.current.Type != TOKEN_NUMBER {
		p.error("expected number after @ (e.g., @0, @1, @2)")
	}

	labelNum, err := strconv.ParseFloat(p.current.Value, 64)
	if err != nil {
		p.error("invalid jump label number")
	}
	label := int(labelNum)

	p.nextToken() // skip label number

	// It's a jump statement: @N or @N value
	if label < 0 {
		p.error("jump label must be >= 0 (use @0, @1, @2, etc.)")
	}
	// Check for optional return value: @0 value
	var value Expression
	if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
		value = p.parseExpression()
	}
	return &JumpStmt{IsBreak: true, Label: label, Value: value}
}

// parseJumpStatement parses ret statements
// ret - return from function
// ret value - return value from function
// ret @N - exit loop N and all inner loops
// ret @N value - exit loop N and return value
func (p *Parser) parseJumpStatement() Statement {
	p.nextToken() // skip 'ret'

	label := 0 // 0 means return from function
	var value Expression

	// Check for optional @N label (for loop exit)
	if p.current.Type == TOKEN_AT {
		p.nextToken() // skip '@'
		if p.current.Type != TOKEN_NUMBER {
			p.error("expected number after @ in ret statement")
		}
		labelNum, err := strconv.ParseFloat(p.current.Value, 64)
		if err != nil {
			p.error("invalid loop label number")
		}
		label = int(labelNum)
		if label < 1 {
			p.error("loop label must be >= 1 (use @1, @2, @3, etc.)")
		}
		p.nextToken() // skip number
	}

	// Check for optional value
	if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
		value = p.parseExpression()
	}

	// ret is always a break/return (IsBreak=true)
	// label=0 means return from function, label>0 means exit loop N
	return &JumpStmt{IsBreak: true, Label: label, Value: value}
}

func (p *Parser) parseExpression() Expression {
	return p.parseErrorHandling()
}

// parseErrorHandling handles the or! operator (lowest precedence, right-associative)
func (p *Parser) parseErrorHandling() Expression {
	left := p.parseConcurrentGather()

	// or! is right-associative and very low precedence
	if p.peek.Type == TOKEN_OR_BANG {
		p.nextToken()                   // move to left
		p.nextToken()                   // skip 'or!'
		right := p.parseErrorHandling() // right-associative recursion
		return &BinaryExpr{Left: left, Operator: "or!", Right: right}
	}

	return left
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
	left := p.parseRange()

	// Check for 'in' operator (membership testing)
	if p.peek.Type == TOKEN_IN {
		p.nextToken() // move to left expr
		p.nextToken() // skip 'in'
		right := p.parseRange()
		return &InExpr{Value: left, Container: right}
	}

	for p.peek.Type == TOKEN_LT || p.peek.Type == TOKEN_GT ||
		p.peek.Type == TOKEN_LE || p.peek.Type == TOKEN_GE ||
		p.peek.Type == TOKEN_EQ || p.peek.Type == TOKEN_NE {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseRange()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

// parseRange handles range expressions (0..<10 or 0..=10)
func (p *Parser) parseRange() Expression {
	left := p.parseAdditive()

	// Check for range operators
	if p.peek.Type == TOKEN_DOTDOTLT || p.peek.Type == TOKEN_DOTDOTEQ {
		p.nextToken() // move to left expr
		inclusive := p.current.Type == TOKEN_DOTDOTEQ
		p.nextToken() // skip range operator
		right := p.parseAdditive()
		return &RangeExpr{Start: left, End: right, Inclusive: inclusive}
	}

	return left
}

func (p *Parser) parseLambdaBody() Expression {
	// Check if lambda body is a block { ... }
	if p.current.Type == TOKEN_LBRACE {
		p.nextToken() // skip '{'
		p.skipNewlines()

		// Parse statements until we hit '}'
		var statements []Statement
		for p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
			stmt := p.parseStatement()
			if stmt != nil {
				statements = append(statements, stmt)
			}

			// Need to advance to the next statement
			// Skip newlines and semicolons between statements
			if p.peek.Type == TOKEN_NEWLINE || p.peek.Type == TOKEN_SEMICOLON {
				p.nextToken() // move to separator
				p.skipNewlines()
			} else if p.peek.Type == TOKEN_RBRACE || p.peek.Type == TOKEN_EOF {
				// At end of block
				p.nextToken() // move to '}'
				break
			} else {
				// No separator found - might be at end
				p.nextToken()
				p.skipNewlines()
			}
		}

		if p.current.Type != TOKEN_RBRACE {
			p.error("expected '}' at end of lambda block")
		}
		// Don't skip the '}' - let the caller handle it

		// Return a BlockExpr containing the statements
		return &BlockExpr{Statements: statements}
	}

	// Otherwise, parse the body expression
	expr := p.parseExpression()

	if p.peek.Type == TOKEN_LBRACE {
		p.nextToken() // move to '{'
		p.nextToken() // skip '{'
		p.skipNewlines()
		return p.parseMatchBlock(expr)
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
		p.peek.Type == TOKEN_ROL || p.peek.Type == TOKEN_ROR ||
		p.peek.Type == TOKEN_PIPE_B || p.peek.Type == TOKEN_AMP_B ||
		p.peek.Type == TOKEN_CARET_B || p.peek.Type == TOKEN_LT_B ||
		p.peek.Type == TOKEN_GT_B || p.peek.Type == TOKEN_LTLT_B ||
		p.peek.Type == TOKEN_GTGT_B {
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

	for p.peek.Type == TOKEN_STAR || p.peek.Type == TOKEN_SLASH || p.peek.Type == TOKEN_MOD || p.peek.Type == TOKEN_FMA {
		p.nextToken()
		op := p.current.Value
		p.nextToken()
		right := p.parseUnary()
		left = &BinaryExpr{Left: left, Operator: op, Right: right}
	}

	return left
}

func (p *Parser) parseUnary() Expression {
	// Handle unary operators (not, ++, --, ~b, ^, &)
	if p.current.Type == TOKEN_NOT {
		p.nextToken() // skip 'not'
		operand := p.parseUnary()
		return &UnaryExpr{Operator: "not", Operand: operand}
	}

	// Handle prefix increment/decrement: ++x, --x
	if p.current.Type == TOKEN_INCREMENT || p.current.Type == TOKEN_DECREMENT {
		op := p.current.Value
		p.nextToken() // skip ++ or --
		operand := p.parseUnary()
		return &UnaryExpr{Operator: op, Operand: operand}
	}

	// Handle bitwise NOT: ~b
	if p.current.Type == TOKEN_TILDE_B {
		p.nextToken() // skip '~b'
		operand := p.parseUnary()
		return &UnaryExpr{Operator: "~b", Operand: operand}
	}

	// Handle head operator: ^
	if p.current.Type == TOKEN_CARET {
		p.nextToken() // skip '^'
		operand := p.parseUnary()
		return &UnaryExpr{Operator: "^", Operand: operand}
	}

	// Handle tail operator: &
	if p.current.Type == TOKEN_AMP {
		p.nextToken() // skip '&'
		operand := p.parseUnary()
		return &UnaryExpr{Operator: "&", Operand: operand}
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

			// Check for empty indexing (syntax error)
			if p.current.Type == TOKEN_RBRACKET {
				p.error("empty indexing [] is not allowed")
			}

			// Check for slice syntax: [start:end], [:end], [start:], [:]
			// Parse the first expression (could be start or index)
			var firstExpr Expression
			var isSlice bool
			if p.current.Type == TOKEN_COLON {
				// Case: [:end] or [::step]
				firstExpr = nil
				isSlice = true
				p.nextToken() // skip ':'
			} else {
				firstExpr = p.parseExpression()
				// Check if this is a slice (has colon)
				isSlice = p.peek.Type == TOKEN_COLON
				if isSlice {
					p.nextToken() // move to colon
					p.nextToken() // skip ':'
				}
			}

			if isSlice {
				var endExpr Expression
				if p.current.Type == TOKEN_RBRACKET {
					// Case: [start:] or [:]
					endExpr = nil
				} else if p.current.Type == TOKEN_COLON {
					// Case: [start::step] or [::step]
					endExpr = nil
					// Don't skip the colon yet - let step handling do it
				} else {
					endExpr = p.parseExpression()
				}

				// Check for step parameter (second colon)
				var stepExpr Expression
				if p.peek.Type == TOKEN_COLON || p.current.Type == TOKEN_COLON {
					if p.current.Type != TOKEN_COLON {
						p.nextToken() // move to second colon
					}
					p.nextToken() // skip ':'

					if p.current.Type == TOKEN_RBRACKET {
						// Case: [start:end:] - step is nil
						stepExpr = nil
					} else {
						stepExpr = p.parseExpression()
						p.nextToken() // move to ']'
					}
				} else if endExpr != nil {
					// We parsed an end expression, need to move to ']'
					p.nextToken()
				}

				expr = &SliceExpr{List: expr, Start: firstExpr, End: endExpr, Step: stepExpr}
			} else {
				// Regular indexing
				p.nextToken() // move to ']'
				expr = &IndexExpr{List: expr, Index: firstExpr}
			}
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
				// current should be on last arg, peek should be ')'
				p.nextToken() // move to ')'
			}
			// current is now on ')', whether we had args or not

			// Wrap the expression in a CallExpr
			// If expr is a LambdaExpr, it will be compiled and called
			// If expr is an IdentExpr, it will be looked up and called
			if ident, ok := expr.(*IdentExpr); ok {
				// Special handling for vector constructors
				if ident.Name == "vec2" {
					if len(args) != 2 {
						p.error("vec2 requires exactly 2 arguments")
					}
					expr = &VectorExpr{Components: args, Size: 2}
				} else if ident.Name == "vec4" {
					if len(args) != 4 {
						p.error("vec4 requires exactly 4 arguments")
					}
					expr = &VectorExpr{Components: args, Size: 4}
				} else {
					expr = &CallExpr{Function: ident.Name, Args: args}
				}
			} else {
				// For lambda expressions or other callable expressions,
				// create a special call expression that compiles the lambda inline
				expr = &DirectCallExpr{Callee: expr, Args: args}
			}
		} else if p.peek.Type == TOKEN_INCREMENT || p.peek.Type == TOKEN_DECREMENT {
			// Handle postfix increment/decrement: x++, x--
			p.nextToken() // skip current expr
			op := p.current.Value
			expr = &PostfixExpr{Operator: op, Operand: expr}
		} else if p.peek.Type == TOKEN_AS {
			// Handle type cast: expr as type
			p.nextToken() // skip current expr
			p.nextToken() // skip 'as'

			// Parse the cast type
			var castType string
			if p.current.Type == TOKEN_IDENT {
				// All type keywords are contextual - check if this identifier is a valid type name
				validTypes := map[string]bool{
					"i8": true, "i16": true, "i32": true, "i64": true,
					"u8": true, "u16": true, "u32": true, "u64": true,
					"f32": true, "f64": true, "cstr": true, "cstring": true,
					"ptr": true, "pointer": true,
					"int": true, "uint": true, "uint32": true, "int32": true,
					"number": true, "string": true, "list": true,
				}
				if validTypes[p.current.Value] {
					castType = p.current.Value
				} else {
					p.error("expected type after 'as'")
				}
			} else {
				p.error("expected type after 'as'")
			}

			expr = &CastExpr{Expr: expr, Type: castType}
		} else if p.peek.Type == TOKEN_DOT {
			// Handle dot notation: obj.field, namespace.func(), or namespace.CONSTANT
			p.nextToken() // skip current expr
			p.nextToken() // skip '.'

			if p.current.Type != TOKEN_IDENT {
				p.error("expected field name after '.'")
			}

			fieldName := p.current.Value

			// Check if this is a namespaced function call or constant: namespace.func() or namespace.CONSTANT
			// This requires expr to be an IdentExpr
			if ident, ok := expr.(*IdentExpr); ok {
				if p.peek.Type == TOKEN_LPAREN {
					// Namespaced function call - combine identifiers
					namespacedName := ident.Name + "." + fieldName
					p.nextToken() // skip second identifier
					p.nextToken() // skip '('
					args := []Expression{}

					if p.current.Type != TOKEN_RPAREN {
						args = append(args, p.parseExpression())
						for p.peek.Type == TOKEN_COMMA {
							p.nextToken() // skip current
							p.nextToken() // skip ','
							args = append(args, p.parseExpression())
						}
						p.nextToken() // move to ')'
					}
					expr = &CallExpr{Function: namespacedName, Args: args}
				} else {
					// Could be a C constant (sdl.SDL_INIT_VIDEO) or field access
					// We'll create a special NamespacedIdentExpr to distinguish at compile time
					expr = &NamespacedIdentExpr{Namespace: ident.Name, Name: fieldName}
				}
			} else {
				// Regular field access - hash the field name and create index expression
				hashValue := hashStringKey(fieldName)
				expr = &IndexExpr{
					List:  expr,
					Index: &NumberExpr{Value: float64(hashValue)},
				}
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
		val := p.parseNumberLiteral(p.current.Value)
		return &NumberExpr{Value: val}

	case TOKEN_STRING:
		return &StringExpr{Value: p.current.Value}

	case TOKEN_FSTRING:
		return p.parseFString()

	case TOKEN_IDENT:
		name := p.current.Value

		// Check if this is a constant reference (substitute with value)
		if expr, isConst := p.constants[name]; isConst {
			// Return a copy of the stored expression to avoid mutation issues
			switch e := expr.(type) {
			case *NumberExpr:
				return &NumberExpr{Value: e.Value}
			case *StringExpr:
				return &StringExpr{Value: e.Value}
			case *ListExpr:
				// Deep copy the list
				elements := make([]Expression, len(e.Elements))
				for i, elem := range e.Elements {
					switch el := elem.(type) {
					case *NumberExpr:
						elements[i] = &NumberExpr{Value: el.Value}
					case *StringExpr:
						elements[i] = &StringExpr{Value: el.Value}
					default:
						elements[i] = elem
					}
				}
				return &ListExpr{Elements: elements}
			}
			return expr
		}

		// Check for lambda: x => expr or x, y => expr
		if p.peek.Type == TOKEN_FAT_ARROW {
			// Try to parse as non-parenthesized lambda
			if lambda := p.tryParseNonParenLambda(); lambda != nil {
				return lambda
			}
		}

		// Error if using -> instead of =>
		if p.peek.Type == TOKEN_ARROW {
			p.error("lambda definitions must use '=>' not '->' (e.g., x => x * 2)")
		}

		// Dot notation is now handled entirely in parsePostfix
		// This includes both field access (obj.field) and namespaced calls (namespace.func())

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
				// current should be on last arg, peek should be ')'
				p.nextToken() // move to ')'
			}
			// current is now on ')', whether we had args or not
			// Special handling for vector constructors
			if name == "vec2" {
				if len(args) != 2 {
					p.error("vec2 requires exactly 2 arguments")
				}
				return &VectorExpr{Components: args, Size: 2}
			} else if name == "vec4" {
				if len(args) != 4 {
					p.error("vec4 requires exactly 4 arguments")
				}
				return &VectorExpr{Components: args, Size: 4}
			}

			return &CallExpr{Function: name, Args: args}
		}
		return &IdentExpr{Name: name}

	case TOKEN_ME:
		// "me" is a special identifier for self-reference
		return &IdentExpr{Name: "me"}

	case TOKEN_CME:
		// "cme" is a special identifier for cached/memoized self-reference
		return &IdentExpr{Name: "cme"}

	case TOKEN_AT_FIRST:
		// @first is true on the first iteration of a loop
		return &LoopStateExpr{Type: "first"}

	case TOKEN_AT_LAST:
		// @last is true on the last iteration of a loop
		return &LoopStateExpr{Type: "last"}

	case TOKEN_AT_COUNTER:
		// @counter is the loop iteration counter
		return &LoopStateExpr{Type: "counter"}

	case TOKEN_AT_I:
		// @i is the current loop, @i1 is outermost loop, @i2 is second loop, etc.
		value := p.current.Value
		level := 0
		if len(value) > 2 { // @iN where N is a number
			// Parse the number after @i
			numStr := value[2:] // Skip "@i"
			if num, err := strconv.Atoi(numStr); err == nil {
				level = num
			}
		}
		return &LoopStateExpr{Type: "i", LoopLevel: level}

	case TOKEN_LPAREN:
		// Could be lambda (params) -> expr or parenthesized expression (expr)
		p.nextToken() // skip '('

		// Check for empty parameter list: () =>
		if p.current.Type == TOKEN_RPAREN {
			if p.peek.Type == TOKEN_FAT_ARROW {
				p.nextToken() // skip ')'
				p.nextToken() // skip '=>'
				body := p.parseLambdaBody()
				return &LambdaExpr{Params: []string{}, Body: body}
			}
			// Error if using -> instead of =>
			if p.peek.Type == TOKEN_ARROW {
				p.error("lambda definitions must use '=>' not '->' (e.g., () => expr)")
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
				// Could be (x) => expr or (x)
				p.nextToken() // move to ')'
				if p.peek.Type == TOKEN_FAT_ARROW {
					// It's a lambda: (x) => expr
					p.nextToken() // skip ')'
					p.nextToken() // skip '=>'
					body := p.parseLambdaBody()
					return &LambdaExpr{Params: []string{firstIdent}, Body: body}
				}
				// Error if using -> instead of =>
				if p.peek.Type == TOKEN_ARROW {
					p.error("lambda definitions must use '=>' not '->' (e.g., (x) => x * 2 or just x => x * 2)")
				}
				// It's (x) parenthesized identifier
				p.nextToken() // skip ')'
				return &IdentExpr{Name: firstIdent}
			}

			if p.peek.Type == TOKEN_COMMA {
				// Definitely a lambda with multiple params: (x, y, ...) => expr
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

				// peek should be '=>'
				if p.peek.Type != TOKEN_FAT_ARROW {
					if p.peek.Type == TOKEN_ARROW {
						p.error("lambda definitions must use '=>' not '->' (e.g., (x, y) => x + y or just x, y => x + y)")
					}
					p.error("expected '=>' after lambda parameters")
				}

				p.nextToken() // skip ')'
				p.nextToken() // skip '=>'
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
			// current should be on last element
			// peek should be ']'
			p.nextToken() // move to ']'
		}
		// For empty list, current is already on ']' after first nextToken()
		return &ListExpr{Elements: elements}

	case TOKEN_LBRACE:
		// Map literal: {key: value, key2: value2, ...}
		// Supports both numeric keys and string identifier keys
		// String identifiers are automatically hashed to uint64
		p.nextToken() // skip '{'
		keys := []Expression{}
		values := []Expression{}

		if p.current.Type != TOKEN_RBRACE {
			// Parse first key
			var key Expression
			if p.current.Type == TOKEN_IDENT && p.peek.Type == TOKEN_COLON {
				// String key: hash identifier to uint64
				hashValue := hashStringKey(p.current.Value)
				key = &NumberExpr{Value: float64(hashValue)}
				p.nextToken() // move past identifier
			} else {
				// Numeric key or expression
				key = p.parseExpression()
				p.nextToken() // move past key
			}

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

				// Parse key (string or numeric)
				if p.current.Type == TOKEN_IDENT && p.peek.Type == TOKEN_COLON {
					// String key: hash identifier to uint64
					hashValue := hashStringKey(p.current.Value)
					key = &NumberExpr{Value: float64(hashValue)}
					p.nextToken() // move past identifier
				} else {
					// Numeric key or expression
					key = p.parseExpression()
					p.nextToken() // move past key
				}

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

	case TOKEN_AT:
		// Could be loop expression (@ i in...) or jump expression (@N)
		// Look ahead to decide
		if p.peek.Type == TOKEN_NUMBER {
			// Jump expression: @N [value]
			// Returns JumpExpr for continuing loops (IsBreak=false)
			p.nextToken() // skip '@'
			if p.current.Type != TOKEN_NUMBER {
				p.error("expected number after @")
			}
			labelNum, _ := strconv.ParseFloat(p.current.Value, 64)
			label := int(labelNum)
			p.nextToken() // skip number
			var value Expression
			if p.current.Type != TOKEN_NEWLINE && p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
				value = p.parseExpression()
				p.nextToken()
			}
			return &JumpExpr{Label: label, Value: value, IsBreak: false}
		}
		// Must be loop expression: @ ident in...
		return p.parseLoopExpr()

	case TOKEN_UNSAFE:
		// unsafe { x86_64 } { arm64 } { riscv64 }
		return p.parseUnsafeExpr()

	case TOKEN_ARENA:
		// arena { ... }
		return p.parseArenaExpr()
	}

	return nil
}

func (p *Parser) parseArenaExpr() Expression {
	p.nextToken() // skip 'arena'

	if p.current.Type != TOKEN_LBRACE {
		p.error("expected '{' after 'arena'")
	}
	p.nextToken() // skip '{'
	p.skipNewlines()

	var body []Statement
	for p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
		p.nextToken()
		p.skipNewlines()
	}

	if p.current.Type != TOKEN_RBRACE {
		p.error("expected '}' at end of arena block")
	}

	return &ArenaExpr{Body: body}
}

// isLoopExpr checks if current position looks like a loop expression
// Pattern: @ ident in
func (p *Parser) isLoopExpr() bool {
	// Loop expressions start with @
	return p.current.Type == TOKEN_AT
}

// parseLoopExpr parses a loop expression: @ i in iterable { body }
func (p *Parser) parseLoopExpr() Expression {
	// Must be @
	if p.current.Type != TOKEN_AT {
		p.error("expected @ to start loop expression")
	}

	label := p.loopDepth + 1
	p.nextToken() // skip '@'

	// Expect identifier for loop variable
	if p.current.Type != TOKEN_IDENT {
		p.error("expected identifier after @")
	}
	iterator := p.current.Value
	p.nextToken() // skip iterator

	// Expect 'in' keyword
	if p.current.Type != TOKEN_IN {
		p.error("expected 'in' keyword in loop expression")
	}
	p.nextToken() // skip 'in'

	iterable := p.parseExpression()
	p.nextToken() // move past iterable

	// Expect '{'
	if p.current.Type != TOKEN_LBRACE {
		p.error("expected '{' to start loop body")
	}
	p.nextToken() // skip '{'

	// Parse loop body
	oldDepth := p.loopDepth
	p.loopDepth = label
	defer func() { p.loopDepth = oldDepth }()

	var body []Statement
	for p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
		if p.current.Type == TOKEN_NEWLINE {
			p.nextToken()
			continue
		}
		stmt := p.parseStatement()
		if stmt != nil {
			body = append(body, stmt)
		}
		p.nextToken()
	}

	if p.current.Type != TOKEN_RBRACE {
		p.error("expected '}' at end of loop body")
	}

	return &LoopExpr{
		Iterator: iterator,
		Iterable: iterable,
		Body:     body,
	}
}

// parseUnsafeExpr parses: unsafe { x86_64 block } { arm64 block } { riscv64 block }
func (p *Parser) parseUnsafeExpr() Expression {
	p.nextToken() // skip 'unsafe'

	// Parse x86_64 block
	if p.current.Type != TOKEN_LBRACE {
		p.error("expected '{' for x86_64 block in unsafe expression")
	}
	x86_64Block := p.parseUnsafeBlock()

	// Parse arm64 block
	if p.current.Type != TOKEN_LBRACE {
		p.error("expected '{' for arm64 block in unsafe expression")
	}
	arm64Block := p.parseUnsafeBlock()

	// Parse riscv64 block
	if p.current.Type != TOKEN_LBRACE {
		p.error("expected '{' for riscv64 block in unsafe expression")
	}
	riscv64Block := p.parseUnsafeBlock()

	return &UnsafeExpr{
		X86_64Block:  x86_64Block,
		ARM64Block:   arm64Block,
		RISCV64Block: riscv64Block,
	}
}

// parseUnsafeBlock parses a single architecture block with extended syntax
func (p *Parser) parseUnsafeBlock() []Statement {
	p.nextToken() // skip '{'
	p.skipNewlines()

	statements := []Statement{}

	for p.current.Type != TOKEN_RBRACE && p.current.Type != TOKEN_EOF {
		// Check for syscall
		if p.current.Type == TOKEN_SYSCALL {
			statements = append(statements, &SyscallStmt{})
			p.nextToken() // skip 'syscall'
			p.skipNewlines()
			continue
		}

		// Check for memory store: [rax] <- value or [rax] <- value as uint8
		if p.current.Type == TOKEN_LBRACKET {
			// Parse: [rax + offset] <- value as type
			p.nextToken() // skip '['

			if p.current.Type != TOKEN_IDENT {
				p.error("expected register name in memory address")
			}
			storeAddr := p.current.Value
			p.nextToken() // skip register name

			// Check for offset: [rax + 16]
			var storeOffset int64 = 0
			if p.current.Type == TOKEN_PLUS {
				p.nextToken() // skip '+'
				if p.current.Type != TOKEN_NUMBER {
					p.error("expected number after '+' in memory address")
				}
				storeOffset = int64(p.parseNumberLiteral(p.current.Value))
				p.nextToken() // skip number
			}

			if p.current.Type != TOKEN_RBRACKET {
				p.error("expected ']' after memory address")
			}
			p.nextToken() // skip ']'

			if p.current.Type != TOKEN_LEFT_ARROW {
				p.error("expected '<-' after memory address")
			}
			p.nextToken() // skip '<-'

			// Parse value
			var value interface{}
			if p.current.Type == TOKEN_NUMBER {
				val := p.parseNumberLiteral(p.current.Value)
				value = &NumberExpr{Value: val}
				p.nextToken()
			} else if p.current.Type == TOKEN_IDENT {
				value = p.current.Value
				p.nextToken()
			} else {
				p.error("expected number or register after '<-' in memory store")
			}

			// Check for size cast: [rax] <- value as uint8
			storeSize := "uint64" // default to 64-bit
			if p.current.Type == TOKEN_AS {
				p.nextToken() // skip 'as'
				if p.current.Type != TOKEN_IDENT {
					p.error("expected type name after 'as'")
				}
				storeSize = p.current.Value
				p.nextToken() // skip type name
			}

			statements = append(statements, &MemoryStore{
				Size:    storeSize,
				Address: storeAddr,
				Offset:  storeOffset,
				Value:   value,
			})
			p.skipNewlines()
			continue
		}

		// Regular register assignment
		if p.current.Type != TOKEN_IDENT {
			p.error("expected register name, memory address, or syscall in unsafe block")
		}

		regName := p.current.Value
		p.nextToken() // skip register name

		if p.current.Type != TOKEN_LEFT_ARROW {
			p.error(fmt.Sprintf("expected '<-' after register %s in unsafe block", regName))
		}
		p.nextToken() // skip '<-'

		// Parse the right-hand side
		value := p.parseUnsafeValue()

		statements = append(statements, &RegisterAssignStmt{
			Register: regName,
			Value:    value,
		})

		p.skipNewlines()
	}

	if p.current.Type != TOKEN_RBRACE {
		p.error("expected '}' to close unsafe block")
	}
	p.nextToken() // skip '}'

	return statements
}

// parseUnsafeValue parses the RHS of a register assignment in unsafe blocks
func (p *Parser) parseUnsafeValue() interface{} {
	// Check for memory load: [rax] or [rax + offset]
	// Followed optionally by: as uint8, as int16, etc.
	if p.current.Type == TOKEN_LBRACKET {
		// [rax] or [rax + offset]
		p.nextToken() // skip '['
		if p.current.Type != TOKEN_IDENT {
			p.error("expected register name in memory load")
		}
		addrReg := p.current.Value
		p.nextToken() // skip register

		var offset int64 = 0
		if p.current.Type == TOKEN_PLUS {
			p.nextToken() // skip '+'
			if p.current.Type != TOKEN_NUMBER {
				p.error("expected number after '+' in memory address")
			}
			offset = int64(p.parseNumberLiteral(p.current.Value))
			p.nextToken() // skip number
		}

		if p.current.Type != TOKEN_RBRACKET {
			p.error("expected ']' after memory address")
		}
		p.nextToken() // skip ']'

		// Check for size cast: [rbx] as uint8
		size := "uint64" // default to 64-bit
		if p.current.Type == TOKEN_AS {
			p.nextToken() // skip 'as'
			if p.current.Type != TOKEN_IDENT {
				p.error("expected type name after 'as'")
			}
			size = p.current.Value
			p.nextToken() // skip type name
		}

		return &MemoryLoad{Size: size, Address: addrReg, Offset: offset}
	}

	// Check for unary operation: ~b rax (bitwise NOT)
	if p.current.Type == TOKEN_TILDE_B {
		p.nextToken() // skip '~b'
		if p.current.Type != TOKEN_IDENT {
			p.error("expected register name after '~b'")
		}
		reg := p.current.Value
		p.nextToken() // skip register
		return &RegisterOp{Left: "", Operator: "~b", Right: reg}
	}

	// Parse left operand (register or immediate)
	var left string
	var leftIsImmediate bool
	var leftValue *NumberExpr

	if p.current.Type == TOKEN_NUMBER {
		val := p.parseNumberLiteral(p.current.Value)
		leftValue = &NumberExpr{Value: val}
		leftIsImmediate = true
		p.nextToken() // skip number

		// Check for cast: 42 as uint8
		if p.current.Type == TOKEN_AS {
			p.nextToken() // skip 'as'
			if p.current.Type == TOKEN_IDENT {
				castType := p.current.Value
				p.nextToken() // skip type
				// Wrap in cast expression
				return &CastExpr{Expr: leftValue, Type: castType}
			} else {
				p.error("expected type after 'as'")
			}
		}
	} else if p.current.Type == TOKEN_IDENT {
		left = p.current.Value
		p.nextToken() // skip register name

		// Check for cast: rax as pointer
		if p.current.Type == TOKEN_AS {
			p.nextToken() // skip 'as'
			if p.current.Type == TOKEN_IDENT {
				castType := p.current.Value
				p.nextToken() // skip type
				// Return cast of variable reference
				return &CastExpr{Expr: &IdentExpr{Name: left}, Type: castType}
			} else {
				p.error("expected type after 'as'")
			}
		}
	} else {
		p.error("expected number, register, memory load, or unary operator")
	}

	// Check for binary operator
	var op string
	switch p.current.Type {
	case TOKEN_PLUS:
		op = "+"
	case TOKEN_MINUS:
		op = "-"
	case TOKEN_STAR:
		op = "*"
	case TOKEN_SLASH:
		op = "/"
	case TOKEN_MOD:
		op = "%"
	case TOKEN_AMP:
		op = "&"
	case TOKEN_PIPE:
		op = "|"
	case TOKEN_CARET_B:
		op = "^b"
	case TOKEN_LT:
		// Check if it's << (shift left)
		if p.peek.Type == TOKEN_LT {
			p.nextToken() // skip first '<'
			op = "<<"
		}
	case TOKEN_GT:
		// Check if it's >> (shift right)
		if p.peek.Type == TOKEN_GT {
			p.nextToken() // skip first '>'
			op = ">>"
		}
	}

	if op != "" {
		// Binary operation
		p.nextToken() // skip operator

		// Parse right operand
		var right interface{}
		if p.current.Type == TOKEN_NUMBER {
			val := p.parseNumberLiteral(p.current.Value)
			right = &NumberExpr{Value: val}
			p.nextToken()
		} else if p.current.Type == TOKEN_IDENT {
			right = p.current.Value
			p.nextToken()
		} else {
			p.error("expected number or register after operator")
		}

		if leftIsImmediate {
			p.error("left operand of binary operation must be a register")
		}

		return &RegisterOp{Left: left, Operator: op, Right: right}
	}

	// No operator - just a simple value
	if leftIsImmediate {
		return leftValue
	}
	return left
}

// LoopInfo tracks information about an active loop during compilation
type LoopInfo struct {
	Label           int   // Loop label (@1, @2, @3, etc.)
	StartPos        int   // Code position of loop start (condition check)
	ContinuePos     int   // Code position for continue (increment step)
	EndPatches      []int // Positions that need to be patched to jump to loop end
	ContinuePatches []int // Positions that need to be patched to jump to continue position

	// Special loop variables support
	IteratorOffset   int  // Stack offset for iterator variable (loop variable)
	IndexOffset      int  // Stack offset for index counter (list loops only)
	UpperBoundOffset int  // Stack offset for limit (range) or length (list)
	ListPtrOffset    int  // Stack offset for list pointer (list loops only)
	IsRangeLoop      bool // True for range loops, false for list loops
}

// Code Generator for Flap
type FlapCompiler struct {
	eb               *ExecutableBuilder
	out              *Out
	variables        map[string]int               // variable name -> stack offset
	mutableVars      map[string]bool              // variable name -> is mutable
	varTypes         map[string]string            // variable name -> "map" or "list"
	sourceCode       string                       // Store source for recompilation
	usedFunctions    map[string]bool              // Track which functions are called
	unknownFunctions map[string]bool              // Track functions called but not defined
	callOrder        []string                     // Track order of function calls
	cImports         map[string]string            // Track C imports: alias -> library name
	cLibHandles      map[string]string            // Track library handles: library -> handle var name
	cConstants       map[string]*CHeaderConstants // Track C constants: alias -> constants
	stringCounter    int                          // Counter for unique string labels
	stackOffset      int                          // Current stack offset for variables
	labelCounter     int                          // Counter for unique labels (if/else, loops, etc)
	lambdaCounter    int                          // Counter for unique lambda function names
	activeLoops      []LoopInfo                   // Stack of active loops (for @N jump resolution)
	lambdaFuncs      []LambdaFunc                 // List of lambda functions to generate
	lambdaOffsets    map[string]int               // Lambda name -> offset in .text
	currentLambda    *LambdaFunc                  // Currently compiling lambda (for "me" self-reference)
	lambdaBodyStart  int                          // Offset where lambda body starts (for tail recursion)
	hasExplicitExit  bool                         // Track if program contains explicit exit() call
	debug            bool                         // Enable debug output (set via DEBUG_FLAP env var)
	cContext         bool                         // When true, compile expressions for C FFI (affects strings, pointers, ints)
	currentArena     int                          // Arena depth (0=none, 1=first arena, 2=nested, etc.)
	usesArenas       bool                         // Track if program uses any arena blocks
	cacheEnabledLambdas map[string]bool           // Track which lambdas use cme
	deferredExprs    [][]Expression               // Stack of deferred expressions per scope (LIFO order)

	metaArenaGrowthErrorJump      int
	firstMetaArenaMallocErrorJump int
}

type LambdaFunc struct {
	Name   string
	Params []string
	Body   Expression
}

func NewFlapCompiler(platform Platform) (*FlapCompiler, error) {
	// Create ExecutableBuilder
	eb, err := NewWithPlatform(platform)
	if err != nil {
		return nil, err
	}

	// Enable dynamic linking
	eb.useDynamicLinking = true
	// Don't set neededFunctions yet - we'll build it dynamically

	// Create Out wrapper
	out := &Out{
		machine: eb.platform,
		writer:  eb.TextWriter(),
		eb:      eb,
	}

	// Check if debug mode is enabled
	debugEnabled := os.Getenv("DEBUG_FLAP") != ""

	return &FlapCompiler{
		eb:                  eb,
		out:                 out,
		variables:           make(map[string]int),
		mutableVars:         make(map[string]bool),
		varTypes:            make(map[string]string),
		usedFunctions:       make(map[string]bool),
		unknownFunctions:    make(map[string]bool),
		callOrder:           []string{},
		cImports:            make(map[string]string),
		cLibHandles:         make(map[string]string),
		cConstants:          make(map[string]*CHeaderConstants),
		lambdaOffsets:       make(map[string]int),
		cacheEnabledLambdas: make(map[string]bool),
		debug:               debugEnabled,
		currentArena:        -1,
	}, nil
}

func (fc *FlapCompiler) Compile(program *Program, outputPath string) error {
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG Compile: starting compilation with %d statements\n", len(program.Statements))
		}
	}
	// Use ARM64 code generator if target is ARM64
	if VerboseMode {
	}
	if fc.eb.platform.Arch == ArchARM64 {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "-> Using ARM64 code generator\n")
		}
		return fc.compileARM64(program, outputPath)
	}
	// Use RISC-V64 code generator if target is RISC-V64
	if fc.eb.platform.Arch == ArchRiscv64 {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "-> Using RISC-V64 code generator\n")
		}
		return fc.compileRiscv64(program, outputPath)
	}

	// Add format strings for printf
	fc.eb.Define("fmt_str", "%s\x00")
	fc.eb.Define("fmt_int", "%ld\n\x00")
	fc.eb.Define("fmt_float", "%.0f\n\x00") // Print float without decimal places

	// Generate code
	// Set up stack frame
	// Note: After _start JMPs here, RSP is 16-byte aligned (kernel guarantee)
	// After PUSH RBP, RSP = (16n - 8), which is correct for making C function calls
	// (CALL will push return address, making RSP = 16n - 16, then function prologue adjusts)
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	// Do NOT subtract from RSP here - we want it at (16n - 8) for C ABI compliance

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

	// Pre-pass: Collect C imports to set up library handles and extract constants
	for _, stmt := range program.Statements {
		if cImport, ok := stmt.(*CImportStmt); ok {
			fc.cImports[cImport.Alias] = cImport.Library
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Registered C import: %s -> %s\n", cImport.Alias, cImport.Library)
			}

			// For .so file imports, extract symbols and function signatures
			if cImport.SoPath != "" {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Extracting symbols from %s...\n", cImport.SoPath)
				}
				symbols, err := ExtractSymbolsFromSo(cImport.SoPath)
				if err != nil {
					// Non-fatal: symbol extraction is optional
					if VerboseMode {
						fmt.Fprintf(os.Stderr, "Warning: failed to extract symbols from %s: %v\n", cImport.SoPath, err)
					}
				} else if VerboseMode && len(symbols) > 0 {
					fmt.Fprintf(os.Stderr, "Found %d symbols in %s\n", len(symbols), cImport.Library)
					if len(symbols) <= 20 {
						for _, sym := range symbols {
							fmt.Fprintf(os.Stderr, "  - %s\n", sym)
						}
					}
				}

				// Extract function signatures from DWARF debug info
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Extracting function signatures from DWARF info...\n")
				}
				signatures, err := ExtractFunctionSignatures(cImport.SoPath)
				if err != nil {
					if VerboseMode {
						fmt.Fprintf(os.Stderr, "Warning: failed to extract signatures: %v\n", err)
					}
				} else if len(signatures) > 0 {
					// Store signatures for this library
					if fc.cConstants[cImport.Alias] == nil {
						fc.cConstants[cImport.Alias] = &CHeaderConstants{
							Constants: make(map[string]int64),
							Macros:    make(map[string]string),
							Functions: make(map[string]*CFunctionSignature),
						}
					}
					// Merge DWARF signatures into the constants map
					for funcName, sig := range signatures {
						fc.cConstants[cImport.Alias].Functions[funcName] = sig
					}
					if VerboseMode {
						fmt.Fprintf(os.Stderr, "Extracted %d function signatures from DWARF\n", len(signatures))
						if len(signatures) <= 10 {
							for name, sig := range signatures {
								paramTypes := make([]string, len(sig.Params))
								for i, p := range sig.Params {
									paramTypes[i] = p.Type
								}
								fmt.Fprintf(os.Stderr, "  - %s(%s) -> %s\n", name, strings.Join(paramTypes, ", "), sig.ReturnType)
							}
						}
					}
				} else if VerboseMode {
					fmt.Fprintf(os.Stderr, "No DWARF debug info found in %s\n", cImport.SoPath)
				}
			}

			// Extract constants from C headers
			constants, err := ExtractConstantsFromLibrary(cImport.Library)
			if err != nil {
				// Non-fatal: constants extraction is optional
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Warning: failed to extract constants from %s: %v\n", cImport.Library, err)
				}
			} else {
				// Merge with existing data (don't overwrite DWARF signatures!)
				if fc.cConstants[cImport.Alias] == nil {
					fc.cConstants[cImport.Alias] = constants
				} else {
					// Merge constants and macros, but don't overwrite existing functions from DWARF
					for k, v := range constants.Constants {
						fc.cConstants[cImport.Alias].Constants[k] = v
					}
					for k, v := range constants.Macros {
						fc.cConstants[cImport.Alias].Macros[k] = v
					}
					// Merge functions (header signatures don't overwrite DWARF)
					for k, v := range constants.Functions {
						if _, exists := fc.cConstants[cImport.Alias].Functions[k]; !exists {
							fc.cConstants[cImport.Alias].Functions[k] = v
						}
					}
				}
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Extracted %d constants from %s\n", len(constants.Constants), cImport.Library)
				}
			}
		}
	}

	// Two-pass compilation: First pass collects all variable declarations
	// so that function/constant order doesn't matter
	for _, stmt := range program.Statements {
		if err := fc.collectSymbols(stmt); err != nil {
			return err
		}
	}

	// Reset arena depth before compilation pass
	fc.currentArena = 0

	// Function prologue - set up stack frame for main code
	// This allows stack-relative addressing via RBP
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")

	fc.pushDeferScope()

	// Second pass: Generate actual code with all symbols known
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Compiling %d statements\n", len(program.Statements))
		for i, stmt := range program.Statements {
			fmt.Fprintf(os.Stderr, "DEBUG:   Statement %d: %T\n", i, stmt)
		}
	}
	for i, stmt := range program.Statements {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: About to compile statement %d: %T\n", i, stmt)
		}
		fc.compileStatement(stmt)
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Finished compiling statement %d\n", i)
		}
	}

	fc.popDeferScope()

	// Automatically exit if no explicit exit() was called
	// Use libc's exit(0) to ensure proper cleanup (flushes printf buffers, etc.)
	if !fc.hasExplicitExit {
		fc.out.XorRegWithReg("rdi", "rdi") // exit code 0
		// Restore stack pointer to frame pointer (rsp % 16 == 8 for proper call alignment)
		// Don't pop rbp since exit() never returns
		fc.out.MovRegToReg("rsp", "rbp")
		fc.trackFunctionCall("exit")
		fc.eb.GenerateCallInstruction("exit")
	}

	// Generate lambda functions
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: About to generate lambda functions\n")
	}
	fc.generateLambdaFunctions()
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Finished generating lambda functions\n")
	}

	// Generate runtime helper functions
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: About to generate runtime helpers\n")
	}
	fc.generateRuntimeHelpers()
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Finished generating runtime helpers\n")
	}

	// Write ELF using existing infrastructure
	return fc.writeELF(program, outputPath)
}

func (fc *FlapCompiler) writeELF(program *Program, outputPath string) error {
	// Build pltFunctions list from all called functions
	// Start with essential functions that runtime helpers need
	pltFunctions := []string{"printf", "exit", "malloc"}

	// Add all functions from usedFunctions (includes call() dynamic calls)
	pltSet := make(map[string]bool)
	for _, f := range pltFunctions {
		pltSet[f] = true
	}
	for funcName := range fc.usedFunctions {
		if !pltSet[funcName] {
			pltFunctions = append(pltFunctions, funcName)
			pltSet[funcName] = true
		}
	}

	// Build mapping from actual calls to PLT indices
	callToPLT := make(map[string]int)
	for i, f := range pltFunctions {
		callToPLT[f] = i
	}

	// Set up dynamic sections
	ds := NewDynamicSections()

	// Only add NEEDED libraries if their functions are actually used
	// libc.so.6 is always needed for basic functionality
	ds.AddNeeded("libc.so.6")

	// Check if any libm functions are called (via call() FFI)
	// Note: builtin math functions like sqrt(), sin(), cos() use hardware instructions, not libm
	// But call("sqrt", ...) calls libm's sqrt
	libmFunctions := map[string]bool{
		"sqrt": true, "sin": true, "cos": true, "tan": true,
		"asin": true, "acos": true, "atan": true, "atan2": true,
		"sinh": true, "cosh": true, "tanh": true,
		"log": true, "log10": true, "exp": true, "pow": true,
		"fabs": true, "fmod": true, "ceil": true, "floor": true,
	}
	needsLibm := false
	for funcName := range fc.usedFunctions {
		if libmFunctions[funcName] {
			needsLibm = true
			break
		}
	}
	if needsLibm {
		ds.AddNeeded("libm.so.6")
	}

	// Add C library dependencies from imports
	for libName := range fc.cLibHandles {
		if libName != "linked" { // Skip our marker value
			// Skip "c" - standard C library functions are already in libc.so.6
			if libName == "c" {
				continue
			}

			// If library name already contains .so, it's a direct .so file - use it as-is
			libSoName := libName
			if strings.Contains(libSoName, ".so") {
				// Direct .so file (e.g., "libmanyargs.so" from import "/tmp/libmanyargs.so" as mylib)
				// Use it directly for DT_NEEDED
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Adding custom C library dependency: %s\n", libSoName)
				}
				ds.AddNeeded(libSoName)
				continue
			}

			// Add .so.X suffix if not present (standard library mapping)
			if !strings.Contains(libSoName, ".so") {
				// Try to get library name from pkg-config
				cmd := exec.Command("pkg-config", "--libs-only-l", libName)
				if output, err := cmd.Output(); err == nil {
					// Parse output like "-lSDL3" to get "SDL3"
					libs := strings.TrimSpace(string(output))
					if strings.HasPrefix(libs, "-l") {
						libSoName = "lib" + strings.TrimPrefix(libs, "-l") + ".so"
					} else {
						// Fallback to standard naming
						if !strings.HasPrefix(libSoName, "lib") {
							libSoName = "lib" + libSoName
						}
						libSoName += ".so"
					}
				} else {
					// pkg-config failed, try to find versioned .so using ldconfig
					if !strings.HasPrefix(libSoName, "lib") {
						libSoName = "lib" + libSoName
					}

					// Try to find the actual .so file with ldconfig
					ldconfigCmd := exec.Command("ldconfig", "-p")
					if ldOutput, ldErr := ldconfigCmd.Output(); ldErr == nil {
						// Search for libname.so in ldconfig output
						lines := strings.Split(string(ldOutput), "\n")
						for _, line := range lines {
							if strings.Contains(line, libSoName) && strings.Contains(line, "=>") {
								// Extract the path after =>
								parts := strings.Split(line, "=>")
								if len(parts) == 2 {
									actualPath := strings.TrimSpace(parts[1])
									// Extract just the filename from the path
									pathParts := strings.Split(actualPath, "/")
									if len(pathParts) > 0 {
										libSoName = pathParts[len(pathParts)-1]
									}
									break
								}
							}
						}
					}

					// If still no version, just add .so
					if !strings.Contains(libSoName, ".so") {
						libSoName += ".so"
					}
				}
			}
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Adding C library dependency: %s\n", libSoName)
			}
			ds.AddNeeded(libSoName)
		}
	}

	// Note: dlopen/dlsym/dlclose are part of libc.so.6 on modern glibc (2.34+)
	// No need to link libdl.so.2 separately

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

	// Clear rodata buffer before writing sorted symbols
	// (in case any data was written during code generation)
	fc.eb.rodata.Reset()

	estimatedRodataAddr := uint64(0x403000 + 0x100)
	currentAddr := estimatedRodataAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]

		// Align string literals to 8-byte boundaries for proper float64 access
		if strings.HasPrefix(symbol, "str_") {
			padding := (8 - (currentAddr % 8)) % 8
			if padding > 0 {
				fc.eb.WriteRodata(make([]byte, padding))
				currentAddr += padding
			}
		}

		fc.eb.WriteRodata([]byte(value))
		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
	}
	if fc.eb.rodata.Len() > 0 {
		previewLen := 32
		if fc.eb.rodata.Len() < previewLen {
			previewLen = fc.eb.rodata.Len()
		}
	}

	// Write complete dynamic ELF with unique PLT functions
	// Note: We pass pltFunctions (unique) for building PLT/GOT structure
	// We'll use fc.callOrder (with duplicates) later for patching actual call sites
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "\n=== First compilation callOrder: %v ===\n", fc.callOrder)
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "=== pltFunctions (unique): %v ===\n", pltFunctions)
		}
	}
	gotBase, rodataBaseAddr, textAddr, pltBase, err := fc.eb.WriteCompleteDynamicELF(ds, pltFunctions)
	if err != nil {
		return err
	}

	// Update rodata addresses using same sorted order
	currentAddr = rodataBaseAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]

		// Apply same alignment as when writing rodata
		if strings.HasPrefix(symbol, "str_") {
			padding := (8 - (currentAddr % 8)) % 8
			currentAddr += padding
		}

		fc.eb.DefineAddr(symbol, currentAddr)
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
	fc.out.SubImmFromReg("rsp", StackSlotSize) // Align stack to 16 bytes
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
	// NOTE: Use the original program parameter (which includes imports),
	// not a reparsed version from source which would lose imported statements

	// Reset compiler state for second pass
	fc.variables = make(map[string]int)
	fc.mutableVars = make(map[string]bool)
	fc.varTypes = make(map[string]string)
	fc.stackOffset = 0
	fc.lambdaFuncs = nil // Clear lambda list to avoid duplicates
	fc.lambdaCounter = 0

	// Collect symbols again (two-pass compilation for second regeneration)
	for _, stmt := range program.Statements {
		if err := fc.collectSymbols(stmt); err != nil {
			return err
		}
	}

	fc.pushDeferScope()

	// Generate code with symbols collected
	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

	fc.popDeferScope()

	// Automatically exit if no explicit exit() was called
	// Use libc's exit(0) to ensure proper cleanup (flushes printf buffers, etc.)
	if !fc.hasExplicitExit {
		fc.out.XorRegWithReg("rdi", "rdi") // exit code 0
		// Restore stack pointer to frame pointer (rsp % 16 == 8 for proper call alignment)
		// Don't pop rbp since exit() never returns
		fc.out.MovRegToReg("rsp", "rbp")
		fc.trackFunctionCall("exit")
		fc.eb.GenerateCallInstruction("exit")
	}

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
		sort.Strings(newSymbols)

		// Append new symbols to rodata and assign addresses
		for _, symbol := range newSymbols {
			value := rodataSymbols[symbol]
			fc.eb.WriteRodata([]byte(value))
			fc.eb.DefineAddr(symbol, currentAddr)
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
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "\n=== Second compilation callOrder: %v ===\n", fc.callOrder)
		}
	}
	fc.eb.patchPLTCalls(ds, textAddr, pltBase, fc.callOrder)

	// Patch PC-relative relocations
	rodataSize := fc.eb.rodata.Len()
	fc.eb.PatchPCRelocations(textAddr, rodataBaseAddr, rodataSize)

	// Patch function calls in regenerated code
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "\n=== Patching function calls (regenerated code) ===\n")
		}
	}
	fc.eb.PatchCallSites(textAddr)

	// Update ELF with regenerated code
	fc.eb.patchTextInELF()

	// Output the executable file
	elfBytes := fc.eb.Bytes()
	if err := os.WriteFile(outputPath, elfBytes, 0o755); err != nil {
		return err
	}

	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Final GOT base: 0x%x\n", gotBase)
		}
	}
	return nil
}

// collectSymbols performs the first pass: collect all variable declarations
// without generating any code. This allows forward references.
func (fc *FlapCompiler) collectSymbols(stmt Statement) error {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Check if variable already exists
		_, exists := fc.variables[s.Name]

		if s.IsUpdate {
			// Update operation (<-) requires existing mutable variable
			if !exists {
				return fmt.Errorf("cannot update undefined variable '%s'", s.Name)
			}
			if !fc.mutableVars[s.Name] {
				return fmt.Errorf("cannot update immutable variable '%s' (use <- only for mutable variables)", s.Name)
			}
		} else if s.Mutable {
			// := - Define mutable variable (error if already exists)
			if exists {
				return fmt.Errorf("variable '%s' already defined (use <- to update)", s.Name)
			}
			// Allocate stack space (16 bytes for alignment)
			fc.stackOffset += 16
			offset := fc.stackOffset
			fc.variables[s.Name] = offset
			fc.mutableVars[s.Name] = true
			if fc.debug {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "DEBUG collectSymbols: storing mutable variable '%s' at offset %d\n", s.Name, offset)
				}
			}

			// Track type if we can determine it from the expression
			exprType := fc.getExprType(s.Value)
			if exprType != "number" && exprType != "unknown" {
				fc.varTypes[s.Name] = exprType
			}
		} else {
			// = - Define immutable variable (can shadow existing immutable, but not mutable)
			if exists && fc.mutableVars[s.Name] {
				return fmt.Errorf("cannot shadow mutable variable '%s' with immutable variable", s.Name)
			}
			// Allocate stack space (16 bytes for alignment) - even if shadowing
			fc.stackOffset += 16
			offset := fc.stackOffset
			fc.variables[s.Name] = offset
			fc.mutableVars[s.Name] = false
			if fc.debug {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "DEBUG collectSymbols: storing immutable variable '%s' at offset %d\n", s.Name, offset)
				}
			}

			// Track type if we can determine it from the expression
			exprType := fc.getExprType(s.Value)
			if exprType != "number" && exprType != "unknown" {
				fc.varTypes[s.Name] = exprType
			}
		}
	case *LoopStmt:
		// Recursively collect symbols from loop body
		for _, bodyStmt := range s.Body {
			if err := fc.collectSymbols(bodyStmt); err != nil {
				return err
			}
		}
	case *ArenaStmt:
		// Track arena depth during symbol collection
		// This ensures alloc() calls are validated correctly
		previousArena := fc.currentArena
		fc.currentArena++

		// Recursively collect symbols from arena body
		// Note: Arena pointers are stored in static storage (_flap_arena_ptrs)
		for _, bodyStmt := range s.Body {
			if err := fc.collectSymbols(bodyStmt); err != nil {
				return err
			}
		}

		// Restore arena depth
		fc.currentArena = previousArena
	case *ExpressionStmt:
		// No symbols to collect from expression statements
	}
	return nil
}

func (fc *FlapCompiler) compileStatement(stmt Statement) {
	switch s := stmt.(type) {
	case *AssignStmt:
		// Variable already registered in collectSymbols pass
		offset := fc.variables[s.Name]

		if fc.debug {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG compileStatement: compiling assignment '%s' (type: %T)\n", s.Name, s.Value)
			}
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
		// Handle PostfixExpr as a statement (like Go)
		if postfix, ok := s.Expr.(*PostfixExpr); ok {
			// x++ and x-- are statements only, not expressions
			identExpr, ok := postfix.Operand.(*IdentExpr)
			if !ok {
				compilerError("postfix operator %s requires a variable operand", postfix.Operator)
			}

			// Get the variable's stack offset
			offset, exists := fc.variables[identExpr.Name]
			if !exists {
				compilerError("undefined variable '%s'", identExpr.Name)
			}

			// Check if variable is mutable
			if !fc.mutableVars[identExpr.Name] {
				compilerError("cannot modify immutable variable '%s'", identExpr.Name)
			}

			// Load current value into xmm0
			fc.out.MovMemToXmm("xmm0", "rbp", -offset)

			// Create 1.0 constant
			labelName := fmt.Sprintf("one_%d", fc.stringCounter)
			fc.stringCounter++

			one := 1.0
			bits := uint64(0)
			*(*float64)(unsafe.Pointer(&bits)) = one
			var floatData []byte
			for i := 0; i < 8; i++ {
				floatData = append(floatData, byte((bits>>(i*8))&ByteMask))
			}
			fc.eb.Define(labelName, string(floatData))

			// Load 1.0 into xmm1
			fc.out.LeaSymbolToReg("rax", labelName)
			fc.out.MovMemToXmm("xmm1", "rax", 0)

			// Apply the operation
			switch postfix.Operator {
			case "++":
				fc.out.AddsdXmm("xmm0", "xmm1") // xmm0 = xmm0 + 1.0
			case "--":
				fc.out.SubsdXmm("xmm0", "xmm1") // xmm0 = xmm0 - 1.0
			default:
				compilerError("unknown postfix operator '%s'", postfix.Operator)
			}

			// Store the modified value back to the variable
			fc.out.MovXmmToMem("xmm0", "rbp", -offset)
		} else {
			fc.compileExpression(s.Expr)
		}

	case *ArenaStmt:
		fc.compileArenaStmt(s)

	case *DeferStmt:
		if len(fc.deferredExprs) == 0 {
			compilerError("defer can only be used inside a function or block scope")
		}
		currentScope := len(fc.deferredExprs) - 1
		fc.deferredExprs[currentScope] = append(fc.deferredExprs[currentScope], s.Call)
	}
}

func (fc *FlapCompiler) pushDeferScope() {
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: pushDeferScope called, len before = %d\n", len(fc.deferredExprs))
	}
	fc.deferredExprs = append(fc.deferredExprs, []Expression{})
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: pushDeferScope called, len after = %d\n", len(fc.deferredExprs))
	}
}

func (fc *FlapCompiler) popDeferScope() {
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: popDeferScope called, len before = %d\n", len(fc.deferredExprs))
	}
	if len(fc.deferredExprs) == 0 {
		return
	}

	currentScope := len(fc.deferredExprs) - 1
	deferred := fc.deferredExprs[currentScope]

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: popDeferScope emitting %d deferred expressions\n", len(deferred))
	}

	for i := len(deferred) - 1; i >= 0; i-- {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG:   Emitting deferred expr %d: %T - %v\n", i, deferred[i], deferred[i])
		}
		fc.compileExpression(deferred[i])
	}

	fc.deferredExprs = fc.deferredExprs[:currentScope]
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: popDeferScope done, len after = %d\n", len(fc.deferredExprs))
	}
}

func (fc *FlapCompiler) compileArenaStmt(stmt *ArenaStmt) {
	// Mark that this program uses arenas
	fc.usesArenas = true

	// Save previous arena context and increment depth
	previousArena := fc.currentArena
	fc.currentArena++
	arenaDepth := fc.currentArena

	// Ensure meta-arena has enough capacity
	// Call _flap_arena_ensure_capacity(arenaDepth)
	fc.out.MovImmToReg("rdi", fmt.Sprintf("%d", arenaDepth))
	fc.out.CallSymbol("_flap_arena_ensure_capacity")

	// Load arena pointer from meta-arena: _flap_arena_meta[arenaDepth-1]
	// Each pointer is 8 bytes, so offset = (arenaDepth-1) * 8
	offset := (arenaDepth - 1) * 8
	fc.out.LeaSymbolToReg("rax", "_flap_arena_meta")
	fc.out.MovMemToReg("rax", "rax", 0)     // Load the meta-arena pointer
	fc.out.MovMemToReg("rax", "rax", offset) // Load the arena pointer from slot

	fc.pushDeferScope()

	// Compile statements in arena body
	for _, bodyStmt := range stmt.Body {
		fc.compileStatement(bodyStmt)
	}

	fc.popDeferScope()

	// Restore previous arena context
	fc.currentArena = previousArena

	// Load arena pointer from meta-arena and destroy
	fc.out.LeaSymbolToReg("rdi", "_flap_arena_meta")
	fc.out.MovMemToReg("rdi", "rdi", 0)     // Load the meta-arena pointer
	fc.out.MovMemToReg("rdi", "rdi", offset) // Load the arena pointer from slot
	fc.out.CallSymbol("flap_arena_destroy")
}

func (fc *FlapCompiler) compileLoopStatement(stmt *LoopStmt) {
	// Check if iterating over a range expression (0..<10, 0..=10)
	if rangeExpr, isRange := stmt.Iterable.(*RangeExpr); isRange {
		// Range loop (lazy iteration)
		fc.compileRangeLoop(stmt, rangeExpr)
	} else {
		// List iteration
		fc.compileListLoop(stmt)
	}
}

func (fc *FlapCompiler) compileRangeLoop(stmt *LoopStmt, rangeExpr *RangeExpr) {
	// Increment label counter for uniqueness
	fc.labelCounter++

	// Evaluate the range start and end
	// Start: evaluate and convert to integer
	fc.compileExpression(rangeExpr.Start)
	fc.out.Cvttsd2si("r8", "xmm0") // r8 = start value (integer)

	// End: evaluate and convert to integer
	fc.compileExpression(rangeExpr.End)
	fc.out.Cvttsd2si("r9", "xmm0") // r9 = end value (integer)

	// For exclusive ranges (..<), end is already correct
	// For inclusive ranges (..=), increment end by 1
	if rangeExpr.Inclusive {
		fc.out.AddImmToReg("r9", 1) // r9 = end + 1 (so loop is start <= i < end+1)
	}

	// Allocate stack space for loop limit (end value) (16 bytes for alignment)
	fc.stackOffset += 16
	limitOffset := fc.stackOffset
	fc.out.SubImmFromReg("rsp", 16)

	// Store limit: mov [rbp - limitOffset], r9
	fc.out.MovRegToMem("r9", "rbp", -limitOffset)

	// Allocate stack space for iterator variable (16 bytes for alignment)
	fc.stackOffset += 16
	iterOffset := fc.stackOffset
	fc.variables[stmt.Iterator] = iterOffset
	fc.mutableVars[stmt.Iterator] = true
	fc.out.SubImmFromReg("rsp", 16)

	// Initialize iterator to start value (from r8)
	// cvtsi2sd xmm0, r8 (convert start integer to float64)
	fc.out.Cvtsi2sd("xmm0", "r8")
	// movsd [rbp - iterOffset], xmm0
	fc.out.MovXmmToMem("xmm0", "rbp", -iterOffset)

	// Loop start label
	loopStartPos := fc.eb.text.Len()

	// Register this loop on the active loop stack
	// Label is determined by loop depth (1-indexed)
	loopLabel := len(fc.activeLoops) + 1
	loopInfo := LoopInfo{
		Label:            loopLabel,
		StartPos:         loopStartPos,
		EndPatches:       []int{},
		IteratorOffset:   iterOffset,
		UpperBoundOffset: limitOffset,
		IsRangeLoop:      true,
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

	// Mark continue position (increment step)
	continuePos := fc.eb.text.Len()
	fc.activeLoops[len(fc.activeLoops)-1].ContinuePos = continuePos

	// Patch all continue jumps to point here
	for _, patchPos := range fc.activeLoops[len(fc.activeLoops)-1].ContinuePatches {
		backOffset := int32(continuePos - (patchPos + 4)) // 4 bytes for 32-bit offset
		fc.patchJumpImmediate(patchPos, backOffset)
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
	backOffset := int32(loopStartPos - (loopBackJumpPos + UnconditionalJumpSize)) // 5 bytes for unconditional jump
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
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm1", "rsp", 0)
	fc.out.MovMemToReg("rax", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)

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
	// Label is determined by loop depth (1-indexed)
	loopLabel := len(fc.activeLoops) + 1
	loopInfo := LoopInfo{
		Label:            loopLabel,
		StartPos:         loopStartPos,
		EndPatches:       []int{},
		IteratorOffset:   iterOffset,
		IndexOffset:      indexOffset,
		UpperBoundOffset: lengthOffset,
		ListPtrOffset:    listPtrOffset,
		IsRangeLoop:      false,
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
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm1", "rsp", 0)
	fc.out.MovMemToReg("rbx", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)

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

	// Mark continue position (increment step)
	continuePos := fc.eb.text.Len()
	fc.activeLoops[len(fc.activeLoops)-1].ContinuePos = continuePos

	// Patch all continue jumps to point here
	for _, patchPos := range fc.activeLoops[len(fc.activeLoops)-1].ContinuePatches {
		backOffset := int32(continuePos - (patchPos + 4)) // 4 bytes for 32-bit offset
		fc.patchJumpImmediate(patchPos, backOffset)
	}

	// Increment index
	fc.out.MovMemToReg("rax", "rbp", -indexOffset)
	fc.out.IncReg("rax")
	fc.out.MovRegToMem("rax", "rbp", -indexOffset)

	// Jump back to loop start
	loopBackJumpPos := fc.eb.text.Len()
	backOffset := int32(loopStartPos - (loopBackJumpPos + UnconditionalJumpSize)) // 5 bytes for unconditional jump
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
	// New semantics with ret keyword:
	// ret (Label=0, IsBreak=true): return from function
	// ret @N (Label=N, IsBreak=true): exit loop N and all inner loops
	// @N (Label=N, IsBreak=false): continue loop N

	// Handle function return: ret with Label=0
	if stmt.Label == 0 && stmt.IsBreak {
		// Return from function
		if stmt.Value != nil {
			fc.compileExpression(stmt.Value)
			// xmm0 now contains return value
		}
		fc.out.MovRegToReg("rsp", "rbp")
		fc.out.PopReg("rbp")
		fc.out.Ret()
		return
	}

	// All other cases require being inside a loop
	if len(fc.activeLoops) == 0 {
		keyword := "@"
		if stmt.IsBreak {
			keyword = "ret"
		}
		compilerError("%s used outside of loop", keyword)
	}

	targetLoopIndex := -1

	if stmt.Label == 0 {
		// Label 0 with IsBreak=false means innermost loop continue
		targetLoopIndex = len(fc.activeLoops) - 1
	} else {
		// Find loop with specified label
		for i := 0; i < len(fc.activeLoops); i++ {
			if fc.activeLoops[i].Label == stmt.Label {
				targetLoopIndex = i
				break
			}
		}

		if targetLoopIndex == -1 {
			keyword := "@"
			if stmt.IsBreak {
				keyword = "ret"
			}
			compilerError("%s @%d references loop @%d which is not active",
				keyword, stmt.Label, stmt.Label)
		}
	}

	if stmt.IsBreak {
		// Break: jump to end of target loop
		jumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Placeholder
		fc.activeLoops[targetLoopIndex].EndPatches = append(
			fc.activeLoops[targetLoopIndex].EndPatches,
			jumpPos+1, // +1 to skip the opcode byte
		)
	} else {
		// Continue: jump to continue point of target loop
		jumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Placeholder
		fc.activeLoops[targetLoopIndex].ContinuePatches = append(
			fc.activeLoops[targetLoopIndex].ContinuePatches,
			jumpPos+1, // +1 to skip the opcode byte
		)
	}
}

func (fc *FlapCompiler) patchJumpImmediate(pos int, offset int32) {
	// Get the current bytes from buffer
	// This is safe because we're patching backwards into already-written code
	bytes := fc.eb.text.Bytes()

	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG PATCH: Before patching at pos %d: %02x %02x %02x %02x\n", pos, bytes[pos], bytes[pos+1], bytes[pos+2], bytes[pos+3])
		}
	}

	// Write 32-bit little-endian offset at position
	bytes[pos] = byte(offset)
	bytes[pos+1] = byte(offset >> 8)
	bytes[pos+2] = byte(offset >> 16)
	bytes[pos+3] = byte(offset >> 24)

	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG PATCH: After patching at pos %d: %02x %02x %02x %02x (offset=%d)\n", pos, bytes[pos], bytes[pos+1], bytes[pos+2], bytes[pos+3], offset)
		}
	}
}

// getExprType returns the type of an expression at compile time
// Returns: "string", "number", "list", "map", or "unknown"
func (fc *FlapCompiler) getExprType(expr Expression) string {
	switch e := expr.(type) {
	case *StringExpr:
		return "string"
	case *NumberExpr:
		return "number"
	case *RangeExpr:
		return "list" // Range expressions compile to lists
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
	case *NamespacedIdentExpr:
		// C constants are always numbers
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
	case *CallExpr:
		// Function calls - check if function returns a string
		stringFuncs := map[string]bool{
			"str": true, "read_file": true, "readln": true,
			"upper": true, "lower": true, "trim": true,
		}
		if stringFuncs[e.Function] {
			return "string"
		}
		// Other functions return numbers by default
		return "number"
	case *SliceExpr:
		// Slicing preserves the type of the list
		return fc.getExprType(e.List)
	case *FStringExpr:
		// F-strings are always strings
		return "string"
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
				floatData = append(floatData, byte((bits>>(i*8))&ByteMask))
			}
			fc.eb.Define(labelName, string(floatData))

			// Load from .rodata
			fc.out.LeaSymbolToReg("rax", labelName)
			fc.out.MovMemToXmm("xmm0", "rax", 0)
		}

	case *StringExpr:
		labelName := fmt.Sprintf("str_%d", fc.stringCounter)
		fc.stringCounter++

		if fc.cContext {
			// C context: compile as null-terminated C string
			// Format: just the raw bytes followed by null terminator
			cStringData := append([]byte(e.Value), 0) // Add null terminator
			fc.eb.Define(labelName, string(cStringData))

			// Load pointer to C string into rax (not xmm0)
			fc.out.LeaSymbolToReg("rax", labelName)
			// Note: In C context, we keep the pointer in rax, not convert to float64
			// The caller (compileCFunctionCall) will handle it appropriately
		} else {
			// Flap context: compile as map[uint64]float64 where keys are indices
			// and values are character codes
			// Map format: [count][key0][val0][key1][val1]...
			// Following Lisp philosophy: even empty strings are objects (count=0), not null

			// Build map data: count followed by key-value pairs
			var mapData []byte

			// Count (number of UTF-8 bytes) - can be 0 for empty strings
			// Note: len(e.Value) returns byte count in Go, not rune count
			count := float64(len(e.Value))
			countBits := uint64(0)
			*(*float64)(unsafe.Pointer(&countBits)) = count
			for i := 0; i < 8; i++ {
				mapData = append(mapData, byte((countBits>>(i*8))&ByteMask))
			}

			// Add each UTF-8 byte as a key-value pair (none for empty strings)
			// IMPORTANT: Iterate over bytes, not runes, to preserve UTF-8 encoding
			for idx := 0; idx < len(e.Value); idx++ {
				// Key: byte index as float64
				keyVal := float64(idx)
				keyBits := uint64(0)
				*(*float64)(unsafe.Pointer(&keyBits)) = keyVal
				for i := 0; i < 8; i++ {
					mapData = append(mapData, byte((keyBits>>(i*8))&ByteMask))
				}

				// Value: UTF-8 byte value as float64 (0-255)
				byteVal := float64(e.Value[idx])
				byteBits := uint64(0)
				*(*float64)(unsafe.Pointer(&byteBits)) = byteVal
				for i := 0; i < 8; i++ {
					mapData = append(mapData, byte((byteBits>>(i*8))&ByteMask))
				}
			}

			fc.eb.Define(labelName, string(mapData))
			fc.out.LeaSymbolToReg("rax", labelName)
			// Convert pointer to float64 (direct register move, no stack)
			fc.out.MovqRegToXmm("xmm0", "rax")
		}

	case *FStringExpr:
		// F-string: concatenate all parts
		if len(e.Parts) == 0 {
			// Empty f-string, return empty string
			fc.compileExpression(&StringExpr{Value: ""})
			return
		}

		// Compile first part
		// Check if it needs str() conversion using type checking
		firstPart := e.Parts[0]
		if fc.getExprType(firstPart) == "string" {
			// Already a string - compile directly
			fc.compileExpression(firstPart)
		} else {
			// Not a string - wrap with str() for conversion
			fc.compileExpression(&CallExpr{
				Function: "str",
				Args:     []Expression{firstPart},
			})
		}

		// Concatenate remaining parts
		for i := 1; i < len(e.Parts); i++ {
			// Save left pointer (current result) to stack
			fc.out.SubImmFromReg("rsp", 16)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)

			// Evaluate right string (next part)
			part := e.Parts[i]
			if fc.getExprType(part) == "string" {
				// Already a string - compile directly
				fc.compileExpression(part)
			} else {
				// Not a string - wrap with str() for conversion
				fc.compileExpression(&CallExpr{
					Function: "str",
					Args:     []Expression{part},
				})
			}

			// Save right pointer to stack
			fc.out.SubImmFromReg("rsp", 16)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)

			// Load arguments: rdi = left ptr, rsi = right ptr
			fc.out.MovMemToReg("rdi", "rsp", 16) // left ptr from [rsp+16]
			fc.out.MovMemToReg("rsi", "rsp", 0)  // right ptr from [rsp]
			fc.out.AddImmToReg("rsp", 32)        // clean up both stack slots

			// Align stack for call
			fc.out.SubImmFromReg("rsp", StackSlotSize)

			// Call _flap_string_concat(rdi, rsi) -> rax
			fc.out.CallSymbol("_flap_string_concat")

			// Restore stack alignment
			fc.out.AddImmToReg("rsp", StackSlotSize)

			// Convert result pointer from rax back to xmm0 (direct register move)
			fc.out.MovqRegToXmm("xmm0", "rax")
		}

	case *IdentExpr:
		// Load variable from stack into xmm0
		offset, exists := fc.variables[e.Name]
		if !exists {
			compilerError("undefined variable '%s' at line %d", e.Name, 0)
		}
		// movsd xmm0, [rbp - offset]
		fc.out.MovMemToXmm("xmm0", "rbp", -offset)

	case *NamespacedIdentExpr:
		// Handle namespaced identifiers like sdl.SDL_INIT_VIDEO or data.field
		// Check if this is a C constant
		if constants, ok := fc.cConstants[e.Namespace]; ok {
			if value, found := constants.Constants[e.Name]; found {
				// Found a C constant - load it as a number
				fc.out.MovImmToReg("rax", strconv.FormatInt(value, 10))
				fc.out.Cvtsi2sd("xmm0", "rax")
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Resolved C constant %s.%s = %d\n", e.Namespace, e.Name, value)
				}
			} else {
				compilerError("undefined constant '%s.%s'", e.Namespace, e.Name)
			}
		} else {
			// Not a C import - treat as field access (obj.field)
			// Convert to IndexExpr and compile it
			hashValue := hashStringKey(e.Name)
			indexExpr := &IndexExpr{
				List:  &IdentExpr{Name: e.Namespace},
				Index: &NumberExpr{Value: float64(hashValue)},
			}
			fc.compileExpression(indexExpr)
		}

	case *LoopStateExpr:
		// @first, @last, @counter, @i are special loop state variables
		if len(fc.activeLoops) == 0 {
			compilerError("@%s used outside of loop", e.Type)
		}

		currentLoop := fc.activeLoops[len(fc.activeLoops)-1]

		switch e.Type {
		case "first":
			// @first: check if counter == 0
			var counterOffset int
			if currentLoop.IsRangeLoop {
				counterOffset = currentLoop.IteratorOffset
				// Load iterator as float, convert to int
				fc.out.MovMemToXmm("xmm0", "rbp", -counterOffset)
				fc.out.Cvttsd2si("rax", "xmm0")
			} else {
				counterOffset = currentLoop.IndexOffset
				// Load index as integer
				fc.out.MovMemToReg("rax", "rbp", -counterOffset)
			}
			// Compare with 0
			fc.out.CmpRegToImm("rax", 0)
			// Set rax to 1 if equal, 0 if not
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			fc.out.Cmove("rax", "rcx") // rax = (counter == 0) ? 1 : 0
			// Convert to float64
			fc.out.Cvtsi2sd("xmm0", "rax")

		case "last":
			// @last: check if counter == upper_bound - 1
			var counterOffset int
			if currentLoop.IsRangeLoop {
				counterOffset = currentLoop.IteratorOffset
				// Load iterator as float, convert to int
				fc.out.MovMemToXmm("xmm0", "rbp", -counterOffset)
				fc.out.Cvttsd2si("rax", "xmm0")
			} else {
				counterOffset = currentLoop.IndexOffset
				// Load index as integer
				fc.out.MovMemToReg("rax", "rbp", -counterOffset)
			}
			// Load upper bound
			fc.out.MovMemToReg("rdi", "rbp", -currentLoop.UpperBoundOffset)
			// Subtract 1 from upper bound: rdi = upper_bound - 1
			fc.out.SubImmFromReg("rdi", 1)
			// Compare counter with upper_bound - 1
			fc.out.CmpRegToReg("rax", "rdi")
			// Set rax to 1 if equal, 0 if not
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			fc.out.Cmove("rax", "rcx") // rax = (counter == upper_bound - 1) ? 1 : 0
			// Convert to float64
			fc.out.Cvtsi2sd("xmm0", "rax")

		case "counter":
			// @counter: return the iteration counter (starting at 0)
			if currentLoop.IsRangeLoop {
				// For range loops, iterator is the counter
				fc.out.MovMemToXmm("xmm0", "rbp", -currentLoop.IteratorOffset)
			} else {
				// For list loops, index is the counter
				fc.out.MovMemToReg("rax", "rbp", -currentLoop.IndexOffset)
				fc.out.Cvtsi2sd("xmm0", "rax")
			}

		case "i":
			// @i (level 0): current loop iterator
			// @i1 (level 1): outermost loop iterator
			// @i2 (level 2): second loop iterator, etc.

			var targetLoop LoopInfo
			if e.LoopLevel == 0 {
				// @i means current loop
				targetLoop = currentLoop
			} else {
				// @iN means loop at level N (1-indexed from outermost)
				if e.LoopLevel > len(fc.activeLoops) {
					compilerError("@i%d refers to loop level %d, but only %d loops active",
						e.LoopLevel, e.LoopLevel, len(fc.activeLoops))
				}
				// activeLoops[0] is outermost (level 1), activeLoops[1] is level 2, etc.
				targetLoop = fc.activeLoops[e.LoopLevel-1]
			}

			// Return the iterator value from the target loop
			fc.out.MovMemToXmm("xmm0", "rbp", -targetLoop.IteratorOffset)

		default:
			compilerError("unknown loop state variable @%s", e.Type)
		}

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
				floatData = append(floatData, byte((bits>>(i*8))&ByteMask))
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
		case "~b":
			// Bitwise NOT: convert to int64, NOT, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.NotReg("rax")            // rax = ~rax
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case "^":
			// Head operator: return first element of list
			// xmm0 contains list pointer (as float64)
			// List format: [length (8 bytes)] [element0] [element1] ...
			// Convert pointer from xmm0 to rax
			fc.out.SubImmFromReg("rsp", StackSlotSize)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)
			fc.out.MovMemToReg("rax", "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)
			// Skip past length (8 bytes) to get to first element
			fc.out.AddImmToReg("rax", 8)
			// Load first element into xmm0
			fc.out.MovMemToXmm("xmm0", "rax", 0)
		case "&":
			// Tail operator: return list without first element
			// xmm0 contains list pointer (as float64)
			// We need to create a new list starting from element 1
			// For now, just adjust the pointer by 8 bytes (skip first element)
			// Note: This is a simplified implementation
			fc.out.SubImmFromReg("rsp", StackSlotSize)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)
			fc.out.MovMemToReg("rax", "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)
			// Load current length
			fc.out.MovMemToReg("rcx", "rax", 0)
			// Decrement length (convert to int, subtract 1, convert back)
			fc.out.SubImmFromReg("rsp", StackSlotSize)
			fc.out.MovRegToMem("rcx", "rsp", 0)
			fc.out.MovMemToXmm("xmm1", "rsp", 0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int(length)
			fc.out.SubImmFromReg("rcx", 1)  // rcx = length - 1
			fc.out.Cvtsi2sd("xmm1", "rcx")  // xmm1 = float(length - 1)
			fc.out.MovXmmToMem("xmm1", "rsp", 0)
			fc.out.MovMemToReg("rcx", "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)
			// Store new length
			fc.out.MovRegToMem("rcx", "rax", 0)
			// Skip first element (advance by 8 bytes)
			fc.out.AddImmToReg("rax", 8)
			// Return adjusted pointer in xmm0
			fc.out.SubImmFromReg("rsp", StackSlotSize)
			fc.out.MovRegToMem("rax", "rsp", 0)
			fc.out.MovMemToXmm("xmm0", "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)
		}

	case *PostfixExpr:
		// PostfixExpr (x++, x--) can only be used as statements, not expressions
		compilerError("%s can only be used as a statement, not in an expression (like Go)", e.Operator)

	case *BinaryExpr:
		// Handle or! error handling operator (special case: short-circuit)
		if e.Operator == "or!" {
			// Evaluate the condition expression (comparisons now return 0.0/1.0)
			fc.compileExpression(e.Left)
			// xmm0 contains the condition result

			// Compare with 0.0 to check if condition is false
			fc.out.XorpdXmm("xmm1", "xmm1") // xmm1 = 0.0
			fc.out.Ucomisd("xmm0", "xmm1")  // Compare xmm0 with 0

			// Jump to success label if condition is non-zero (true)
			// Save position for jump patching
			jumpPos := fc.eb.text.Len()
			fc.out.JumpConditional(JumpNotEqual, 0) // Placeholder, will patch later

			// Condition is false (zero): print error message and exit
			// Right side should be a string literal
			if strExpr, ok := e.Right.(*StringExpr); ok {
				// Write error message to stderr (fd=2)
				errorMsg := strExpr.Value + "\n"

				// Create label for error message
				errorLabel := fmt.Sprintf("error_msg_%d", fc.stringCounter)
				fc.stringCounter++
				fc.eb.Define(errorLabel, errorMsg)

				// syscall: write(2, msg, len)
				fc.out.MovImmToReg("rax", "1")                              // syscall number for write
				fc.out.MovImmToReg("rdi", "2")                              // fd = 2 (stderr)
				fc.out.LeaSymbolToReg("rsi", errorLabel)                    // msg = error string
				fc.out.MovImmToReg("rdx", fmt.Sprintf("%d", len(errorMsg))) // len
				fc.eb.Emit("syscall")

				// syscall: exit(1)
				fc.out.MovImmToReg("rax", "60") // syscall number for exit
				fc.out.MovImmToReg("rdi", "1")  // exit code = 1
				fc.eb.Emit("syscall")
			} else {
				// Right side is not a string literal, just exit with code 1
				fc.out.MovImmToReg("rax", "60") // syscall number for exit
				fc.out.MovImmToReg("rdi", "1")  // exit code = 1
				fc.eb.Emit("syscall")
			}

			// Success label: condition was true, patch jump to here
			successPos := fc.eb.text.Len()
			// Calculate offset from end of jump instruction to success position
			// JumpConditional emits 6 bytes on x86-64 (0x0f 0x85 + 4-byte offset)
			jumpEndPos := jumpPos + 6
			offset := int32(successPos - jumpEndPos)
			fc.patchJumpImmediate(jumpPos+2, offset) // +2 to skip opcode bytes, patch the 4-byte offset
			// xmm0 still contains the original condition value
			return
		}

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
						mapData = append(mapData, byte((countBits>>(i*8))&ByteMask))
					}

					for idx, ch := range result {
						// Key: index
						keyVal := float64(idx)
						keyBits := uint64(0)
						*(*float64)(unsafe.Pointer(&keyBits)) = keyVal
						for i := 0; i < 8; i++ {
							mapData = append(mapData, byte((keyBits>>(i*8))&ByteMask))
						}

						// Value: char code
						charVal := float64(ch)
						charBits := uint64(0)
						*(*float64)(unsafe.Pointer(&charBits)) = charVal
						for i := 0; i < 8; i++ {
							mapData = append(mapData, byte((charBits>>(i*8))&ByteMask))
						}
					}

					fc.eb.Define(labelName, string(mapData))
					fc.out.LeaSymbolToReg("rax", labelName)
					fc.out.SubImmFromReg("rsp", StackSlotSize)
					fc.out.MovRegToMem("rax", "rsp", 0)
					fc.out.MovMemToXmm("xmm0", "rsp", 0)
					fc.out.AddImmToReg("rsp", StackSlotSize)
					break
				}

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
				fc.out.SubImmFromReg("rsp", StackSlotSize)

				// Call the helper function (direct call, not through PLT)
				fc.out.CallSymbol("_flap_string_concat")

				// Restore stack alignment
				fc.out.AddImmToReg("rsp", StackSlotSize)

				// Result pointer is in rax, convert to xmm0
				fc.out.SubImmFromReg("rsp", StackSlotSize)
				fc.out.MovRegToMem("rax", "rsp", 0)
				fc.out.MovMemToXmm("xmm0", "rsp", 0)
				fc.out.AddImmToReg("rsp", StackSlotSize)
				break
			}

			if leftType == "list" && rightType == "list" {
				// List concatenation: [1, 2] + [3, 4] -> [1, 2, 3, 4]
				leftList, leftIsLiteral := e.Left.(*ListExpr)
				rightList, rightIsLiteral := e.Right.(*ListExpr)

				if leftIsLiteral && rightIsLiteral {
					// Compile-time concatenation
					labelName := fmt.Sprintf("list_%d", fc.stringCounter)
					fc.stringCounter++

					var listData []byte

					// Calculate total length
					totalLen := float64(len(leftList.Elements) + len(rightList.Elements))
					lengthBits := uint64(0)
					*(*float64)(unsafe.Pointer(&lengthBits)) = totalLen
					for i := 0; i < 8; i++ {
						listData = append(listData, byte((lengthBits>>(i*8))&ByteMask))
					}

					// Add all elements from left list
					for _, elem := range leftList.Elements {
						if numExpr, ok := elem.(*NumberExpr); ok {
							elemBits := uint64(0)
							*(*float64)(unsafe.Pointer(&elemBits)) = numExpr.Value
							for i := 0; i < 8; i++ {
								listData = append(listData, byte((elemBits>>(i*8))&ByteMask))
							}
						}
					}

					// Add all elements from right list
					for _, elem := range rightList.Elements {
						if numExpr, ok := elem.(*NumberExpr); ok {
							elemBits := uint64(0)
							*(*float64)(unsafe.Pointer(&elemBits)) = numExpr.Value
							for i := 0; i < 8; i++ {
								listData = append(listData, byte((elemBits>>(i*8))&ByteMask))
							}
						}
					}

					fc.eb.Define(labelName, string(listData))
					fc.out.LeaSymbolToReg("rax", labelName)
					fc.out.SubImmFromReg("rsp", StackSlotSize)
					fc.out.MovRegToMem("rax", "rsp", 0)
					fc.out.MovMemToXmm("xmm0", "rsp", 0)
					fc.out.AddImmToReg("rsp", StackSlotSize)
					return
				}

				// Runtime concatenation
				// Compile left list (result in xmm0)
				fc.compileExpression(e.Left)
				fc.out.SubImmFromReg("rsp", 16)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)

				// Compile right list (result in xmm0)
				fc.compileExpression(e.Right)
				fc.out.SubImmFromReg("rsp", 16)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)

				// Call _flap_list_concat(left_ptr, right_ptr)
				fc.out.MovMemToReg("rdi", "rsp", 16) // left ptr
				fc.out.MovMemToReg("rsi", "rsp", 0)  // right ptr
				fc.out.AddImmToReg("rsp", 32)

				// Align stack for call
				fc.out.SubImmFromReg("rsp", StackSlotSize)

				// Call the helper function
				// Note: Don't track internal function calls (see comment at _flap_string_eq call)
				fc.out.CallSymbol("_flap_list_concat")

				fc.out.AddImmToReg("rsp", StackSlotSize)

				// Result pointer is in rax, convert to xmm0
				fc.out.SubImmFromReg("rsp", StackSlotSize)
				fc.out.MovRegToMem("rax", "rsp", 0)
				fc.out.MovMemToXmm("xmm0", "rsp", 0)
				fc.out.AddImmToReg("rsp", StackSlotSize)

				// Return early - don't do numeric operation
				return
			}

			if leftType == "map" && rightType == "map" {
				// Map union - TODO
				compilerError("map union not yet implemented")
			}
		}

		// String comparison operators
		if e.Operator == "==" || e.Operator == "!=" {
			leftType := fc.getExprType(e.Left)
			rightType := fc.getExprType(e.Right)

			if leftType == "string" && rightType == "string" {
				// String comparison: compare character by character
				// Compile left string (result in xmm0)
				fc.compileExpression(e.Left)
				fc.out.SubImmFromReg("rsp", 16)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)

				// Compile right string (result in xmm0)
				fc.compileExpression(e.Right)
				fc.out.SubImmFromReg("rsp", 16)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)

				// Call _flap_string_eq(left_ptr, right_ptr)
				fc.out.MovMemToReg("rdi", "rsp", 16) // left ptr
				fc.out.MovMemToReg("rsi", "rsp", 0)  // right ptr
				fc.out.AddImmToReg("rsp", 32)

				// Align stack for call
				fc.out.SubImmFromReg("rsp", StackSlotSize)

				// Call the helper function
				// Note: Don't track internal function calls - they use CallSymbol (0x00000000 placeholder)
				// not GenerateCallInstruction (0x12345678 placeholder), so they shouldn't be in callOrder
				fc.out.CallSymbol("_flap_string_eq")

				// Restore stack alignment
				fc.out.AddImmToReg("rsp", StackSlotSize)

				// Result (1.0 or 0.0) is in xmm0
				if e.Operator == "!=" {
					// Invert the result: result = 1.0 - result
					labelName := fmt.Sprintf("float_const_%d", fc.stringCounter)
					fc.stringCounter++
					one := 1.0
					bits := uint64(0)
					*(*float64)(unsafe.Pointer(&bits)) = one
					var floatData []byte
					for i := 0; i < 8; i++ {
						floatData = append(floatData, byte((bits>>(i*8))&ByteMask))
					}
					fc.eb.Define(labelName, string(floatData))
					fc.out.LeaSymbolToReg("rax", labelName)
					fc.out.MovMemToXmm("xmm1", "rax", 0)
					fc.out.SubsdXmm("xmm1", "xmm0") // xmm1 = 1.0 - xmm0
					fc.out.MovRegToReg("xmm0", "xmm1")
				}
				return
			}
		}

		// Default: numeric binary operation
		// Compile left into xmm0
		fc.compileExpression(e.Left)
		// Save left in xmm2 (register-to-register, no stack needed)
		fc.out.MovRegToReg("xmm2", "xmm0")
		// Compile right into xmm0
		fc.compileExpression(e.Right)
		// Move right operand to xmm1
		fc.out.MovRegToReg("xmm1", "xmm0")
		// Move left operand from xmm2 to xmm0
		fc.out.MovRegToReg("xmm0", "xmm2")
		// Perform scalar floating-point operation
		switch e.Operator {
		case "+":
			fc.out.AddsdXmm("xmm0", "xmm1") // addsd xmm0, xmm1
		case "-":
			fc.out.SubsdXmm("xmm0", "xmm1") // subsd xmm0, xmm1
		case "*":
			fc.out.MulsdXmm("xmm0", "xmm1") // mulsd xmm0, xmm1
		case "*+":
			// FMA: a *+ b = a * a + b (square and add, using fused multiply-add)
			// Use VFMADD213SD xmm0, xmm0, xmm1 => xmm0 = xmm0 * xmm0 + xmm1
			// Encoding: C4 E2 F9 A9 C1 (VFMADD213SD xmm0, xmm0, xmm1)
			fc.out.Write(0xC4) // VEX 3-byte prefix
			fc.out.Write(0xE2) // VEX byte 1: R=1, X=1, B=1, m=00010 (0F38)
			fc.out.Write(0xF9) // VEX byte 2: W=1, vvvv=0000 (xmm0), L=0, pp=01 (66)
			fc.out.Write(0xA9) // Opcode: VFMADD213SD
			fc.out.Write(0xC1) // ModR/M: 11 000 001 (xmm0, xmm0, xmm1)
		case "/":
			// Check for division by zero (xmm1 == 0.0)
			fc.out.XorpdXmm("xmm2", "xmm2") // xmm2 = 0.0
			fc.out.Ucomisd("xmm1", "xmm2")  // Compare divisor with 0

			// Jump to division if not zero
			jumpPos := fc.eb.text.Len()
			fc.out.JumpConditional(JumpNotEqual, 0) // Placeholder, will patch later

			// Division by zero: print error and exit
			errorMsg := "Error: division by zero\n"
			errorLabel := fmt.Sprintf("div_zero_error_%d", fc.stringCounter)
			fc.stringCounter++
			fc.eb.Define(errorLabel, errorMsg)

			// syscall: write(2, msg, len)
			fc.out.MovImmToReg("rax", "1")                              // syscall number for write
			fc.out.MovImmToReg("rdi", "2")                              // fd = 2 (stderr)
			fc.out.LeaSymbolToReg("rsi", errorLabel)                    // msg = error string
			fc.out.MovImmToReg("rdx", fmt.Sprintf("%d", len(errorMsg))) // len
			fc.eb.Emit("syscall")

			// syscall: exit(1)
			fc.out.MovImmToReg("rax", "60") // syscall number for exit
			fc.out.MovImmToReg("rdi", "1")  // exit code = 1
			fc.eb.Emit("syscall")

			// Patch jump to here (safe division)
			safePos := fc.eb.text.Len()
			jumpEndPos := jumpPos + 6
			offset := int32(safePos - jumpEndPos)
			fc.patchJumpImmediate(jumpPos+2, offset)

			fc.out.DivsdXmm("xmm0", "xmm1") // divsd xmm0, xmm1
		case "mod", "%":
			// Modulo: a mod b = a - b * floor(a / b)
			// xmm0 = dividend (a), xmm1 = divisor (b)

			// Check for modulo by zero (xmm1 == 0.0)
			fc.out.XorpdXmm("xmm4", "xmm4") // xmm4 = 0.0
			fc.out.Ucomisd("xmm1", "xmm4")  // Compare divisor with 0

			// Jump to modulo if not zero
			jumpPos := fc.eb.text.Len()
			fc.out.JumpConditional(JumpNotEqual, 0) // Placeholder

			// Modulo by zero: print error and exit
			errorMsg := "Error: modulo by zero\n"
			errorLabel := fmt.Sprintf("mod_zero_error_%d", fc.stringCounter)
			fc.stringCounter++
			fc.eb.Define(errorLabel, errorMsg)

			// syscall: write(2, msg, len)
			fc.out.MovImmToReg("rax", "1")
			fc.out.MovImmToReg("rdi", "2")
			fc.out.LeaSymbolToReg("rsi", errorLabel)
			fc.out.MovImmToReg("rdx", fmt.Sprintf("%d", len(errorMsg)))
			fc.eb.Emit("syscall")

			// syscall: exit(1)
			fc.out.MovImmToReg("rax", "60")
			fc.out.MovImmToReg("rdi", "1")
			fc.eb.Emit("syscall")

			// Patch jump to here (safe modulo)
			safePos := fc.eb.text.Len()
			jumpEndPos := jumpPos + 6
			offset := int32(safePos - jumpEndPos)
			fc.patchJumpImmediate(jumpPos+2, offset)

			fc.out.MovXmmToXmm("xmm2", "xmm0") // Save dividend in xmm2
			fc.out.MovXmmToXmm("xmm3", "xmm1") // Save divisor in xmm3
			fc.out.DivsdXmm("xmm0", "xmm1")    // xmm0 = a / b
			// Floor: convert to int64 and back
			fc.out.Cvttsd2si("rax", "xmm0")    // rax = floor(a / b) as int
			fc.out.Cvtsi2sd("xmm0", "rax")     // xmm0 = floor(a / b) as float
			fc.out.MulsdXmm("xmm0", "xmm3")    // xmm0 = floor(a / b) * b
			fc.out.SubsdXmm("xmm2", "xmm0")    // xmm2 = a - floor(a / b) * b
			fc.out.MovXmmToXmm("xmm0", "xmm2") // Result in xmm0
		case "<", "<=", ">", ">=", "==", "!=":
			// Compare xmm0 with xmm1, sets flags
			fc.out.Ucomisd("xmm0", "xmm1")
			// Convert comparison result to boolean (0.0 or 1.0)
			fc.out.MovImmToReg("rax", "0")
			fc.out.MovImmToReg("rcx", "1")
			// Use conditional move based on comparison operator
			switch e.Operator {
			case "<":
				fc.out.Cmovb("rax", "rcx") // rax = (xmm0 < xmm1) ? 1 : 0
			case "<=":
				fc.out.Cmovbe("rax", "rcx") // rax = (xmm0 <= xmm1) ? 1 : 0
			case ">":
				fc.out.Cmova("rax", "rcx") // rax = (xmm0 > xmm1) ? 1 : 0
			case ">=":
				fc.out.Cmovae("rax", "rcx") // rax = (xmm0 >= xmm1) ? 1 : 0
			case "==":
				fc.out.Cmove("rax", "rcx") // rax = (xmm0 == xmm1) ? 1 : 0
			case "!=":
				fc.out.Cmovne("rax", "rcx") // rax = (xmm0 != xmm1) ? 1 : 0
			}
			// Convert integer result to float64
			fc.out.Cvtsi2sd("xmm0", "rax")
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
		case "|b":
			// Bitwise OR: convert to int64, OR, convert back
			fc.out.Cvttsd2si("rax", "xmm0")   // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1")   // rcx = int64(xmm1)
			fc.out.OrRegWithReg("rax", "rcx") // rax |= rcx
			fc.out.Cvtsi2sd("xmm0", "rax")    // xmm0 = float64(rax)
		case "&b":
			// Bitwise AND: convert to int64, AND, convert back
			fc.out.Cvttsd2si("rax", "xmm0")    // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1")    // rcx = int64(xmm1)
			fc.out.AndRegWithReg("rax", "rcx") // rax &= rcx
			fc.out.Cvtsi2sd("xmm0", "rax")     // xmm0 = float64(rax)
		case "^b":
			// Bitwise XOR: convert to int64, XOR, convert back
			fc.out.Cvttsd2si("rax", "xmm0")    // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1")    // rcx = int64(xmm1)
			fc.out.XorRegWithReg("rax", "rcx") // rax ^= rcx
			fc.out.Cvtsi2sd("xmm0", "rax")     // xmm0 = float64(rax)
		case "<b":
			// Shift left (same as shl): convert to int64, shift, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.ShlClReg("rax", "cl")    // rax <<= cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case ">b":
			// Shift right (same as shr): convert to int64, shift, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.ShrClReg("rax", "cl")    // rax >>= cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case "<<b":
			// Rotate left (same as rol): convert to int64, rotate, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.RolClReg("rax", "cl")    // rol rax, cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		case ">>b":
			// Rotate right (same as ror): convert to int64, rotate, convert back
			fc.out.Cvttsd2si("rax", "xmm0") // rax = int64(xmm0)
			fc.out.Cvttsd2si("rcx", "xmm1") // rcx = int64(xmm1)
			fc.out.RorClReg("rax", "cl")    // ror rax, cl
			fc.out.Cvtsi2sd("xmm0", "rax")  // xmm0 = float64(rax)
		}

	case *CallExpr:
		fc.compileCall(e)

	case *DirectCallExpr:
		fc.compileDirectCall(e)

	case *RangeExpr:
		// Compile range expression by expanding it to a list
		// 0..<10 becomes [0, 1, 2, ..., 9]
		// 0..=10 becomes [0, 1, 2, ..., 10]

		// Evaluate start and end expressions (must be compile-time constants for now)
		startNum, startOk := e.Start.(*NumberExpr)
		endNum, endOk := e.End.(*NumberExpr)

		if !startOk || !endOk {
			compilerError("range expressions currently only support number literals")
		}

		start := int64(startNum.Value)
		end := int64(endNum.Value)

		// Build list of elements
		var elements []Expression
		if e.Inclusive {
			// ..= includes end value
			for i := start; i <= end; i++ {
				elements = append(elements, &NumberExpr{Value: float64(i)})
			}
		} else {
			// ..< excludes end value
			for i := start; i < end; i++ {
				elements = append(elements, &NumberExpr{Value: float64(i)})
			}
		}

		// Compile as a list
		listExpr := &ListExpr{Elements: elements}
		fc.compileExpression(listExpr)

	case *ListExpr:
		// Following Lisp philosophy: even empty lists are objects (length=0), not null
		// Create list data in .rodata and return pointer
		// List format: [length (8 bytes)] [element1] [element2] ...

		// Allocate list data in .rodata
		labelName := fmt.Sprintf("list_%d", fc.stringCounter)
		fc.stringCounter++

		// Store list as: [length (8 bytes)] [element1] [element2] ...
		var listData []byte

		// First, add length as float64
		length := float64(len(e.Elements))
		lengthBits := uint64(0)
		*(*float64)(unsafe.Pointer(&lengthBits)) = length
		listData = append(listData, byte(lengthBits&ByteMask))
		listData = append(listData, byte((lengthBits>>8)&ByteMask))
		listData = append(listData, byte((lengthBits>>16)&ByteMask))
		listData = append(listData, byte((lengthBits>>24)&ByteMask))
		listData = append(listData, byte((lengthBits>>32)&ByteMask))
		listData = append(listData, byte((lengthBits>>40)&ByteMask))
		listData = append(listData, byte((lengthBits>>48)&ByteMask))
		listData = append(listData, byte((lengthBits>>56)&ByteMask))

		// Then add elements
		for _, elem := range e.Elements {
			// Evaluate element to get float64 value
			// For now, only support number literals
			if numExpr, ok := elem.(*NumberExpr); ok {
				val := numExpr.Value
				// Convert float64 to 8 bytes (little-endian)
				bits := uint64(0)
				*(*float64)(unsafe.Pointer(&bits)) = val
				listData = append(listData, byte(bits&ByteMask))
				listData = append(listData, byte((bits>>8)&ByteMask))
				listData = append(listData, byte((bits>>16)&ByteMask))
				listData = append(listData, byte((bits>>24)&ByteMask))
				listData = append(listData, byte((bits>>32)&ByteMask))
				listData = append(listData, byte((bits>>40)&ByteMask))
				listData = append(listData, byte((bits>>48)&ByteMask))
				listData = append(listData, byte((bits>>56)&ByteMask))
			} else {
				compilerError("list literal elements must be constant numbers")
			}
		}

		fc.eb.Define(labelName, string(listData))
		fc.out.LeaSymbolToReg("rax", labelName)
		// Convert pointer to float64: reinterpret rax as xmm0
		// Push rax to stack, then load as float64 into xmm0
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case *InExpr:
		// Membership testing: value in container
		// Returns 1.0 if found, 0.0 if not found

		// Compile value to search for
		fc.compileExpression(e.Value)
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Compile container
		fc.compileExpression(e.Container)
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)
		fc.out.MovMemToReg("rbx", "rsp", StackSlotSize) // rbx = container pointer

		// Load count from container (empty containers have count=0, not null)
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

		// Patch found jump to skip to end
		endPos := fc.eb.text.Len()
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
			mapData = append(mapData, byte((countBits>>(i*8))&ByteMask))
		}

		// Add key-value pairs (if any)
		for i := range e.Keys {
			if keyNum, ok := e.Keys[i].(*NumberExpr); ok {
				keyBits := uint64(0)
				*(*float64)(unsafe.Pointer(&keyBits)) = keyNum.Value
				for j := 0; j < 8; j++ {
					mapData = append(mapData, byte((keyBits>>(j*8))&ByteMask))
				}
			}
			if valNum, ok := e.Values[i].(*NumberExpr); ok {
				valBits := uint64(0)
				*(*float64)(unsafe.Pointer(&valBits)) = valNum.Value
				for j := 0; j < 8; j++ {
					mapData = append(mapData, byte((valBits>>(j*8))&ByteMask))
				}
			}
		}

		fc.eb.Define(labelName, string(mapData))
		fc.out.LeaSymbolToReg("rax", labelName)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
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
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)

		// Load container pointer from stack to rbx
		fc.out.MovMemToXmm("xmm1", "rsp", 0)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm1", "rsp", 0)
		fc.out.MovMemToReg("rbx", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

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
			fc.out.MovMemToXmm("xmm2", "rsp", StackSlotSize)

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
			fc.out.MovMemToXmm("xmm0", "rsp", StackSlotSize)
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

		if fc.debug {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG compileExpression: adding lambda '%s' with %d params\n", funcName, len(e.Params))
			}
		}

		// Store lambda for later code generation
		fc.lambdaFuncs = append(fc.lambdaFuncs, LambdaFunc{
			Name:   funcName,
			Params: e.Params,
			Body:   e.Body,
		})

		// Return function pointer as float64 in xmm0
		// Use LEA to get function address, then convert to float64
		fc.out.LeaSymbolToReg("rax", funcName)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case *LengthExpr:
		// Compile the operand (should be a list, returns pointer as float64 in xmm0)
		fc.compileExpression(e.Operand)

		// Convert pointer from float64 to integer in rax
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rax", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Load length from list (empty lists have length=0.0, not null)
		fc.out.MovMemToXmm("xmm0", "rax", 0)

		// Length is now in xmm0 as float64

	case *BlockExpr:
		// First, collect symbols from all statements in the block
		for _, stmt := range e.Statements {
			if err := fc.collectSymbols(stmt); err != nil {
				compilerError("%v at line 0", err)
			}
		}

		// Compile each statement in the block
		// The last statement should leave its value in xmm0
		for i, stmt := range e.Statements {
			fc.compileStatement(stmt)
			// If it's not the last statement and it's an expression statement,
			// the value is already in xmm0 but will be overwritten by the next statement
			if i == len(e.Statements)-1 {
				// Last statement - its value should already be in xmm0
				// If it's an expression statement, compileStatement already put it there
				// If it's an assignment, we need to load the assigned value
				if assignStmt, ok := stmt.(*AssignStmt); ok {
					fc.compileExpression(&IdentExpr{Name: assignStmt.Name})
				} else if exprStmt, ok := stmt.(*ExpressionStmt); ok {
					// Expression statement - result should already be in xmm0
					fc.compileExpression(exprStmt.Expr)
				}
			}
		}

	case *MatchExpr:
		fc.compileMatchExpr(e)

	case *ParallelExpr:
		fc.compileParallelExpr(e)

	case *PipeExpr:
		fc.compilePipeExpr(e)

	case *ConcurrentGatherExpr:
		fc.compileConcurrentGatherExpr(e)

	case *CastExpr:
		fc.compileCastExpr(e)

	case *UnsafeExpr:
		fc.compileUnsafeExpr(e)

	case *SliceExpr:
		fc.compileSliceExpr(e)

	case *VectorExpr:
		// Allocate stack space for vector components (8 bytes per float64)
		stackSize := int64(e.Size * 8)
		fc.out.SubImmFromReg("rsp", stackSize)

		// Compile and store each component
		for i, comp := range e.Components {
			fc.compileExpression(comp)
			// Result is in xmm0, store it to stack at offset i*8
			offset := i * 8
			fc.out.MovXmmToMem("xmm0", "rsp", offset)
		}

		// Load stack address into rax and convert to float64 for return
		fc.out.MovRegToReg("rax", "rsp")
		fc.out.Cvtsi2sd("xmm0", "rax")
	}
}

func (fc *FlapCompiler) compileMatchExpr(expr *MatchExpr) {
	fc.compileExpression(expr.Condition)

	fc.labelCounter++

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
		needsZeroCompare = true
	}

	var defaultJumpPos int
	if needsZeroCompare {
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm1", "rax")
		fc.out.Ucomisd("xmm0", "xmm1")
		defaultJumpPos = fc.eb.text.Len()
		fc.out.JumpConditional(JumpEqual, 0)
	} else {
		defaultJumpPos = fc.eb.text.Len()
		fc.out.JumpConditional(jumpCond, 0)
	}

	endJumpPositions := []int{}
	pendingGuardJumps := []int{}

	if len(expr.Clauses) == 0 {
		// Preserve the condition's value when the block only specifies a default
		jumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)
		endJumpPositions = append(endJumpPositions, jumpPos)
	} else {
		for _, clause := range expr.Clauses {
			// Patch any guards that should skip to this clause
			for _, pos := range pendingGuardJumps {
				offset := int32(fc.eb.text.Len() - (pos + 6))
				fc.patchJumpImmediate(pos+2, offset)
			}
			pendingGuardJumps = pendingGuardJumps[:0]

			if clause.Guard != nil {
				fc.compileExpression(clause.Guard)
				fc.out.XorRegWithReg("rax", "rax")
				fc.out.Cvtsi2sd("xmm1", "rax")
				fc.out.Ucomisd("xmm0", "xmm1")
				guardJump := fc.eb.text.Len()
				fc.out.JumpConditional(JumpEqual, 0)
				pendingGuardJumps = append(pendingGuardJumps, guardJump)
			}

			fc.compileMatchClauseResult(clause.Result, &endJumpPositions)
		}
	}

	defaultPos := fc.eb.text.Len()

	for _, pos := range pendingGuardJumps {
		offset := int32(defaultPos - (pos + 6))
		fc.patchJumpImmediate(pos+2, offset)
	}

	defaultOffset := int32(defaultPos - (defaultJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(defaultJumpPos+2, defaultOffset)

	fc.compileMatchDefault(expr.DefaultExpr)

	endPos := fc.eb.text.Len()
	for _, jumpPos := range endJumpPositions {
		endOffset := int32(endPos - (jumpPos + 5))
		fc.patchJumpImmediate(jumpPos+1, endOffset)
	}
}

func (fc *FlapCompiler) compileMatchClauseResult(result Expression, endJumps *[]int) {
	if jumpExpr, isJump := result.(*JumpExpr); isJump {
		fc.compileMatchJump(jumpExpr)
		return
	}

	fc.compileExpression(result)
	jumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	*endJumps = append(*endJumps, jumpPos)
}

func (fc *FlapCompiler) compileMatchDefault(result Expression) {
	if jumpExpr, isJump := result.(*JumpExpr); isJump {
		fc.compileMatchJump(jumpExpr)
		return
	}

	fc.compileExpression(result)
}

func (fc *FlapCompiler) compileMatchJump(jumpExpr *JumpExpr) {
	// Handle ret (Label=0, IsBreak=true) - return from function
	if jumpExpr.Label == 0 && jumpExpr.IsBreak {
		// Return from function
		if jumpExpr.Value != nil {
			fc.compileExpression(jumpExpr.Value)
			// xmm0 now contains return value
		}
		fc.out.MovRegToReg("rsp", "rbp")
		fc.out.PopReg("rbp")
		fc.out.Ret()
		return
	}

	// Handle ret @N or @N - loop control
	if len(fc.activeLoops) == 0 {
		keyword := "@"
		if jumpExpr.IsBreak {
			keyword = "ret"
		}
		compilerError("%s @%d used outside of loop in match expression", keyword, jumpExpr.Label)
	}

	// Find the loop with the specified label
	targetLoopIndex := -1
	for i := 0; i < len(fc.activeLoops); i++ {
		if fc.activeLoops[i].Label == jumpExpr.Label {
			targetLoopIndex = i
			break
		}
	}

	if targetLoopIndex == -1 {
		keyword := "@"
		if jumpExpr.IsBreak {
			keyword = "ret"
		}
		compilerError("%s @%d references loop @%d which is not active",
			keyword, jumpExpr.Label, jumpExpr.Label)
	}

	if jumpExpr.IsBreak {
		// ret @N - exit loop N and all inner loops
		jumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)
		fc.activeLoops[targetLoopIndex].EndPatches = append(
			fc.activeLoops[targetLoopIndex].EndPatches,
			jumpPos+1,
		)
	} else {
		// @N - continue loop N (jump to continue point)
		jumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)
		fc.activeLoops[targetLoopIndex].ContinuePatches = append(
			fc.activeLoops[targetLoopIndex].ContinuePatches,
			jumpPos+1,
		)
	}
}

func (fc *FlapCompiler) compileCastExpr(expr *CastExpr) {
	// Compile the expression being cast (result in xmm0)
	fc.compileExpression(expr.Expr)

	// Cast conversions for FFI:
	// - Integer types (i8-i64, u8-u64): truncate float64 to integer
	// - Float types (f32, f64): precision changes (f64 is no-op)
	// - cstr: convert Flap string map to C null-terminated string
	// - ptr: reinterpret bits (no conversion)
	// - number: no-op (already float64)

	switch expr.Type {
	case "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64":
		// Integer casts: convert float64 to integer and back
		// This truncates fractional parts for FFI with C integer types
		// cvttsd2si rax, xmm0  (convert with truncation)
		fc.out.Cvttsd2si("rax", "xmm0")
		// cvtsi2sd xmm0, rax (convert back to float64)
		fc.out.Cvtsi2sd("xmm0", "rax")
		// Note: Since Flap uses float64 internally, we don't mask bits
		// The truncation is sufficient for C FFI purposes

	case "f32":
		// f32 cast: for C float arguments
		// For now, keep as float64 (C will handle the conversion)
		// TODO: Add explicit cvtsd2ss/cvtss2sd if needed for precision

	case "f64":
		// Already float64, nothing to do
		// This is the native Flap type

	case "ptr":
		// Pointer cast: value is already in xmm0 as float64 (reinterpreted bits)
		// No conversion needed - bits pass through as-is
		// Used for NULL pointers and raw memory addresses

	case "number":
		// Convert C return value to Flap number (identity, already float64)
		// This is a no-op but explicit for FFI clarity

	case "cstr":
		// Convert Flap string to C null-terminated string
		// xmm0 contains pointer to Flap string map
		// Call runtime function: flap_string_to_cstr(xmm0) -> rax
		fc.out.CallSymbol("flap_string_to_cstr")
		// Convert C string pointer (rax) back to float64 in xmm0
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovRegToMem("rax", "rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "string":
		// Convert C char* to Flap string
		// xmm0 contains C string pointer as float64
		// TODO: implement flap_cstr_to_string runtime function
		if VerboseMode {
			fmt.Fprintln(os.Stderr, "Use 'as cstr' to convert Flap strings to C strings")
		}
		compilerError("'as string' conversion not yet implemented")

	case "list":
		// Convert C array to Flap list
		// TODO: implement when needed (requires length parameter)
		compilerError("'as list' conversion not yet implemented")

	default:
		compilerError("unknown cast type '%s'", expr.Type)
	}
}

func (fc *FlapCompiler) compileSliceExpr(expr *SliceExpr) {
	// Slice syntax: list[start:end:step] or string[start:end:step]
	// For now, implement simple case: string/list[start:end] (step=1, forward)

	// Compile the collection expression (result in xmm0 as pointer)
	fc.compileExpression(expr.List)

	// Save collection pointer on stack
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)

	// Compile step parameter first to know if we need special defaults
	if expr.Step != nil {
		fc.compileExpression(expr.Step)
		// step is now in xmm0
	} else {
		// Default step = 1
		fc.out.MovImmToReg("rax", "1")
		fc.out.Cvtsi2sd("xmm0", "rax")
	}
	// Save step on stack temporarily
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)

	// Compile start index (default depends on step sign)
	if expr.Start != nil {
		fc.compileExpression(expr.Start)
	} else {
		// Check if step is negative (convert to integer first)
		fc.out.MovMemToXmm("xmm0", "rsp", 0) // load step
		fc.out.Cvttsd2si("rax", "xmm0")      // convert to integer
		fc.out.XorRegWithReg("rbx", "rbx")
		fc.out.CmpRegToReg("rax", "rbx") // compare with 0

		negStepStartJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpLess, 0) // If step < 0, jump to negative step path

		// Positive step: default start = 0
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm0", "rax")

		negStepStartEndJumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Skip negative step path

		// Negative step: default start = length - 1
		negStepStartPos := fc.eb.text.Len()
		negStepStartOffset := int32(negStepStartPos - (negStepStartJumpPos + ConditionalJumpSize))
		fc.patchJumpImmediate(negStepStartJumpPos+2, negStepStartOffset)

		fc.out.MovMemToReg("rax", "rsp", StackSlotSize) // Load collection pointer
		fc.out.MovMemToXmm("xmm0", "rax", 0)            // Load length
		fc.out.MovImmToReg("rax", "1")
		fc.out.Cvtsi2sd("xmm1", "rax")
		fc.out.SubsdXmm("xmm0", "xmm1") // xmm0 = length - 1

		negStepStartEndPos := fc.eb.text.Len()
		negStepStartEndOffset := int32(negStepStartEndPos - (negStepStartEndJumpPos + UnconditionalJumpSize))
		fc.patchJumpImmediate(negStepStartEndJumpPos+1, negStepStartEndOffset)
	}
	// Save start on stack
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)

	// Compile end index (default depends on step sign)
	if expr.End != nil {
		fc.compileExpression(expr.End)
		// end is now in xmm0
	} else {
		// Check if step is negative (convert to integer first)
		fc.out.MovMemToXmm("xmm0", "rsp", StackSlotSize) // load step (now 8 bytes back from start)
		fc.out.Cvttsd2si("rax", "xmm0")                  // convert to integer
		fc.out.XorRegWithReg("rbx", "rbx")
		fc.out.CmpRegToReg("rax", "rbx") // compare with 0

		negStepEndJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpLess, 0) // If step < 0, jump to negative step path

		// Positive step: default end = length
		fc.out.MovMemToReg("rax", "rsp", 16) // Load collection pointer
		fc.out.MovMemToXmm("xmm0", "rax", 0) // Load length

		negStepEndEndJumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0) // Skip negative step path

		// Negative step: default end = -1
		negStepEndPos := fc.eb.text.Len()
		negStepEndOffset := int32(negStepEndPos - (negStepEndJumpPos + ConditionalJumpSize))
		fc.patchJumpImmediate(negStepEndJumpPos+2, negStepEndOffset)

		fc.out.XorRegWithReg("rax", "rax") // rax = 0
		fc.out.SubImmFromReg("rax", 1)     // rax = -1
		fc.out.Cvtsi2sd("xmm0", "rax")     // xmm0 = -1

		negStepEndEndPos := fc.eb.text.Len()
		negStepEndEndOffset := int32(negStepEndEndPos - (negStepEndEndJumpPos + UnconditionalJumpSize))
		fc.patchJumpImmediate(negStepEndEndJumpPos+1, negStepEndEndOffset)
	}
	// end is in xmm0
	// Save end on stack
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)

	// Stack layout: [collection_ptr][step][start][end] (rsp points to end)
	// Call runtime function: flap_slice_string(collection_ptr, start, end, step) -> new_collection_ptr

	// Load step into rcx (arg4)
	fc.out.MovMemToXmm("xmm0", "rsp", 16)
	fc.out.Cvttsd2si("rcx", "xmm0") // rcx = step (as integer)

	// Load end into rdx (arg3)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.Cvttsd2si("rdx", "xmm0") // rdx = end (as integer)

	// Load start into rsi (arg2)
	fc.out.MovMemToXmm("xmm0", "rsp", StackSlotSize)
	fc.out.Cvttsd2si("rsi", "xmm0") // rsi = start (as integer)

	// Load collection pointer into rdi (arg1)
	fc.out.MovMemToReg("rdi", "rsp", 24) // rdi = collection pointer

	// Clean up stack before call (4 values * 8 bytes = 32)
	fc.out.AddImmToReg("rsp", 32)

	// Call runtime function
	fc.out.CallSymbol("flap_slice_string")

	// Result (new string pointer) is in rax, convert to float64 in xmm0
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovRegToMem("rax", "rsp", 0)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)
}

func (fc *FlapCompiler) compileUnsafeExpr(expr *UnsafeExpr) {
	// Execute the appropriate architecture block based on target
	// For now, we only support x86_64
	arch := "x86_64" // TODO: Get from build target

	var block []Statement
	switch arch {
	case "x86_64":
		block = expr.X86_64Block
	case "arm64":
		block = expr.ARM64Block
	case "riscv64":
		block = expr.RISCV64Block
	default:
		compilerError("unsupported architecture: %s", arch)
	}

	// Compile each statement in the unsafe block
	for _, stmt := range block {
		switch s := stmt.(type) {
		case *RegisterAssignStmt:
			fc.compileRegisterAssignment(s)
		case *MemoryStore:
			fc.compileSizedMemoryStore(s)
		case *SyscallStmt:
			fc.compileSyscall()
		default:
			compilerError("unsupported statement type in unsafe block: %T", s)
		}
	}

	// The result of an unsafe block is the value in the accumulator register
	// x86_64: rax, arm64: x0, riscv64: a0
	// Convert the integer in rax to float64 in xmm0
	fc.out.Cvtsi2sd("xmm0", "rax")
}

func (fc *FlapCompiler) compileSyscall() {
	// Emit raw syscall instruction
	// Registers must be set up before calling syscall:
	// x86-64: rax=syscall#, rdi=arg1, rsi=arg2, rdx=arg3, r10=arg4, r8=arg5, r9=arg6
	// ARM64: x8=syscall#, x0-x6=args
	// RISC-V: a7=syscall#, a0-a6=args
	fc.out.Syscall()
}

func (fc *FlapCompiler) compileRegisterAssignment(stmt *RegisterAssignStmt) {
	// Handle memory stores: [rax] <- value
	if len(stmt.Register) > 2 && stmt.Register[0] == '[' && stmt.Register[len(stmt.Register)-1] == ']' {
		addr := stmt.Register[1 : len(stmt.Register)-1]
		fc.compileMemoryStore(addr, stmt.Value)
		return
	}

	// Handle various value types
	switch v := stmt.Value.(type) {
	case *NumberExpr:
		// Immediate value: register <- 42
		if stmt.Register == "stack" {
			compilerError("cannot assign immediate value to stack; use 'stack <- register' to push")
		}
		val := int64(v.Value)
		fc.out.MovImmToReg(stmt.Register, strconv.FormatInt(val, 10))

	case string:
		// Handle stack operations
		if stmt.Register == "stack" && v != "stack" {
			// Push: stack <- rax
			fc.out.PushReg(v)
		} else if stmt.Register != "stack" && v == "stack" {
			// Pop: rax <- stack
			fc.out.PopReg(stmt.Register)
		} else if stmt.Register == "stack" && v == "stack" {
			compilerError("cannot do 'stack <- stack'")
		} else {
			// Register-to-register move: rax <- rbx
			fc.out.MovRegToReg(stmt.Register, v)
		}

	case *RegisterOp:
		// Arithmetic or bitwise operation
		fc.compileRegisterOp(stmt.Register, v)

	case *MemoryLoad:
		// Memory load: rax <- [rbx] or rax <- u8 [rbx + 16]
		fc.compileMemoryLoad(stmt.Register, v)

	case *CastExpr:
		// Type cast: rax <- 42 as uint8, rax <- ptr as pointer
		fc.compileUnsafeCast(stmt.Register, v)

	default:
		compilerError("unsupported value type in register assignment: %T", v)
	}
}

func (fc *FlapCompiler) compileRegisterOp(dest string, op *RegisterOp) {
	// Unary operations
	if op.Left == "" {
		if op.Operator == "~b" {
			// NOT: dest <- ~b src
			srcReg := op.Right.(string)
			if dest != srcReg {
				fc.out.MovRegToReg(dest, srcReg)
			}
			fc.out.NotReg(dest)
			return
		}
		compilerError("unsupported unary operator in unsafe block: %s", op.Operator)
	}

	// Binary operations: dest <- left OP right
	// First, ensure left operand is in dest
	if dest != op.Left {
		fc.out.MovRegToReg(dest, op.Left)
	}

	// Now apply the operation
	switch op.Operator {
	case "+":
		switch r := op.Right.(type) {
		case string:
			fc.out.AddRegToReg(dest, r)
		case *NumberExpr:
			fc.out.AddImmToReg(dest, int64(r.Value))
		}
	case "-":
		switch r := op.Right.(type) {
		case string:
			fc.out.SubRegFromReg(dest, r)
		case *NumberExpr:
			fc.out.SubImmFromReg(dest, int64(r.Value))
		}
	case "&":
		switch r := op.Right.(type) {
		case string:
			fc.out.AndRegWithReg(dest, r)
		case *NumberExpr:
			fc.out.AndRegWithImm(dest, int32(r.Value))
		}
	case "|":
		switch r := op.Right.(type) {
		case string:
			fc.out.OrRegWithReg(dest, r)
		case *NumberExpr:
			fc.out.OrRegWithImm(dest, int32(r.Value))
		}
	case "^b":
		switch r := op.Right.(type) {
		case string:
			fc.out.XorRegWithReg(dest, r)
		case *NumberExpr:
			fc.out.XorRegWithImm(dest, int32(r.Value))
		}
	case "*":
		switch r := op.Right.(type) {
		case string:
			fc.out.ImulRegWithReg(dest, r)
		case *NumberExpr:
			fc.out.ImulImmToReg(dest, int64(r.Value))
		}
	case "/":
		switch r := op.Right.(type) {
		case string:
			fc.out.DivRegByReg(dest, r)
		case *NumberExpr:
			fc.out.DivRegByImm(dest, int64(r.Value))
		}
	case "<<":
		switch r := op.Right.(type) {
		case string:
			fc.out.ShlRegByReg(dest, r)
		case *NumberExpr:
			fc.out.ShlRegByImm(dest, int64(r.Value))
		}
	case ">>":
		switch r := op.Right.(type) {
		case string:
			fc.out.ShrRegByReg(dest, r)
		case *NumberExpr:
			fc.out.ShrRegByImm(dest, int64(r.Value))
		}
	default:
		compilerError("operator %s not yet implemented in v1.5.0", op.Operator)
	}
}

func (fc *FlapCompiler) compileMemoryLoad(dest string, load *MemoryLoad) {
	// Memory load: dest <- [addr + offset]
	// Support sized loads: uint8, int8, uint16, int16, uint32, int32, uint64, int64
	switch load.Size {
	case "", "uint64", "int64":
		// Default 64-bit load (unsigned and signed are the same for full width)
		fc.out.MovMemToReg(dest, load.Address, int(load.Offset))
	case "uint8":
		// Zero-extend byte to 64-bit
		fc.out.MovU8MemToReg(dest, load.Address, int(load.Offset))
	case "int8":
		// Sign-extend byte to 64-bit
		fc.out.MovI8MemToReg(dest, load.Address, int(load.Offset))
	case "uint16":
		// Zero-extend word to 64-bit
		fc.out.MovU16MemToReg(dest, load.Address, int(load.Offset))
	case "int16":
		// Sign-extend word to 64-bit
		fc.out.MovI16MemToReg(dest, load.Address, int(load.Offset))
	case "uint32":
		// Zero-extend dword to 64-bit (automatic on x86-64)
		fc.out.MovU32MemToReg(dest, load.Address, int(load.Offset))
	case "int32":
		// Sign-extend dword to 64-bit
		fc.out.MovI32MemToReg(dest, load.Address, int(load.Offset))
	default:
		compilerError("unsupported memory load size: %s (supported: uint8, int8, uint16, int16, uint32, int32, uint64, int64)", load.Size)
	}
}

func (fc *FlapCompiler) compileSizedMemoryStore(store *MemoryStore) {
	// Memory store: [addr + offset] <- value as size
	// Get the value into a register first
	var srcReg string

	switch v := store.Value.(type) {
	case string:
		// Value is already in a register
		srcReg = v
	case *NumberExpr:
		// Load immediate value into a temporary register (r11)
		srcReg = "r11"
		val := int64(v.Value)
		fc.out.MovImmToReg(srcReg, strconv.FormatInt(val, 10))
	default:
		compilerError("unsupported value type in memory store: %T", store.Value)
	}

	// Perform sized store based on Size field
	switch store.Size {
	case "", "uint64", "int64":
		// Default 64-bit store
		fc.out.MovRegToMem(srcReg, store.Address, int(store.Offset))
	case "uint8", "int8":
		// Byte store (signed and unsigned are the same for stores)
		fc.out.MovU8RegToMem(srcReg, store.Address, int(store.Offset))
	case "uint16", "int16":
		// Word store
		fc.out.MovU16RegToMem(srcReg, store.Address, int(store.Offset))
	case "uint32", "int32":
		// Dword store
		fc.out.MovU32RegToMem(srcReg, store.Address, int(store.Offset))
	default:
		compilerError("unsupported memory store size: %s (supported: uint8, int8, uint16, int16, uint32, int32, uint64, int64)", store.Size)
	}
}

func (fc *FlapCompiler) compileMemoryStore(addr string, value interface{}) {
	// Memory store: [addr] <- value
	switch v := value.(type) {
	case string:
		// Store register: [rax] <- rbx
		fc.out.MovRegToMem(v, addr, 0)
	case *NumberExpr:
		// Store immediate: [rax] <- 42
		fc.out.MovImmToMem(int64(v.Value), addr, 0)
	default:
		compilerError("unsupported memory store value type: %T", value)
	}
}

func (fc *FlapCompiler) compileUnsafeCast(dest string, cast *CastExpr) {
	// Handle type casts in unsafe blocks
	// Examples: rax <- 42 as uint8, rax <- buffer as pointer, rax <- msg as cstr

	switch expr := cast.Expr.(type) {
	case *NumberExpr:
		// Immediate cast: rax <- 42 as uint8
		val := int64(expr.Value)
		// For integer types, just load the value (truncation happens naturally)
		fc.out.MovImmToReg(dest, strconv.FormatInt(val, 10))

	case *IdentExpr:
		// Variable cast: rax <- buffer as pointer, rax <- msg as cstr
		// Load the variable value
		if offset, ok := fc.variables[expr.Name]; ok {
			// Stack variable - load as float64 in xmm0
			fc.out.MovMemToXmm("xmm0", "rbp", -offset)

			// Handle specific cast types
			if cast.Type == "cstr" || cast.Type == "cstring" {
				// Convert Flap string to C null-terminated string
				// xmm0 contains pointer to Flap string map
				fc.trackFunctionCall("flap_string_to_cstr")
				fc.out.CallSymbol("flap_string_to_cstr")
				// Result is C string pointer in rax
				if dest != "rax" {
					fc.out.MovRegToReg(dest, "rax")
				}
			} else {
				// For other types (pointer, integer types), convert to int
				fc.out.Cvttsd2si(dest, "xmm0")
			}
		} else {
			compilerError("undefined variable in unsafe cast: %s", expr.Name)
		}

	default:
		compilerError("unsupported cast expression type in unsafe block: %T", expr)
	}
}

func (fc *FlapCompiler) compileParallelExpr(expr *ParallelExpr) {
	// For now, only support: list || lambda
	lambda, ok := expr.Operation.(*LambdaExpr)
	if !ok {
		compilerError("parallel operator (||) currently only supports lambda expressions")
	}

	if len(lambda.Params) != 1 {
		compilerError("parallel operator lambda must have exactly one parameter")
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
	fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize) // Store at rsp+8
	fc.out.MovMemToReg("r11", "rsp", StackSlotSize)  // Reinterpret float64 bits as pointer
	fc.out.MovRegToMem("r11", "rsp", StackSlotSize)  // Keep integer pointer for later loads

	// Compile the input list expression (returns pointer as float64 in xmm0)
	fc.compileExpression(expr.List)

	// Save list pointer to stack (reuse reserved slot) and load as integer pointer
	fc.out.MovXmmToMem("xmm0", "rsp", 0) // Store at rsp+0
	fc.out.MovMemToReg("r13", "rsp", 0)

	// Load list length from [r13] into r14 (empty lists have length=0, not null)
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
	backOffset := int32(loopStart - (loopBackJumpPos + UnconditionalJumpSize))
	fc.out.JumpUnconditional(backOffset)

	// Loop end
	loopEndPos := fc.eb.text.Len()

	// Patch conditional jump
	endOffset := int32(loopEndPos - (loopEndJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(loopEndJumpPos+2, endOffset)

	// Don't clean up the lambda/list spill area yet - it's part of our memory layout
	// The result buffer includes this space in its allocation

	// Return result list pointer as float64 in xmm0
	// r12 points to the result buffer on stack
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovRegToMem("r12", "rsp", 0)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)

	// Adjust stack pointer to account for result buffer AND spill area still being there
	// The calling code must use the result before further stack operations
	fc.out.AddImmToReg("rsp", parallelResultAlloc+16)

	// End of parallel operator - xmm0 contains result pointer as float64
}

func (fc *FlapCompiler) generateLambdaFunctions() {
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG generateLambdaFunctions: generating %d lambdas\n", len(fc.lambdaFuncs))
		}
	}
	for _, lambda := range fc.lambdaFuncs {
		if fc.debug {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG generateLambdaFunctions: generating lambda '%s'\n", lambda.Name)
			}
		}
		// Record the offset of this lambda function in .text
		fc.lambdaOffsets[lambda.Name] = fc.eb.text.Len()

		// Mark the start of the lambda function with a label
		fc.eb.MarkLabel(lambda.Name)

		// Function prologue
		fc.out.PushReg("rbp")
		fc.out.MovRegToReg("rbp", "rsp")
		fc.out.SubImmFromReg("rsp", StackSlotSize) // Align stack

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
				compilerError("lambda has too many parameters (max 6)")
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

		fc.pushDeferScope()

		// Compile lambda body (result in xmm0)
		fc.compileExpression(lambda.Body)

		fc.popDeferScope()

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

func (fc *FlapCompiler) generateCacheLookup() {
	fc.eb.MarkLabel("flap_cache_lookup")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")

	fc.out.MovRegToReg("r12", "rdi")
	fc.out.MovRegToReg("r13", "rsi")

	fc.out.MovMemToReg("rax", "r12", 0)
	fc.out.CmpRegToImm("rax", 0)
	notInitJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)

	fc.out.MovMemToReg("rdi", "r12", 0)
	fc.out.MovMemToReg("rsi", "r12", 8)

	fc.out.MovRegToReg("rax", "r13")
	fc.out.AndRegWithImm("rax", 31)

	fc.out.Emit([]byte{0x48, 0xc1, 0xe0, 0x04})
	fc.out.AddRegToReg("rax", "rdi")
	fc.out.MovRegToReg("rbx", "rax")

	fc.out.XorRegWithReg("rcx", "rcx")

	loopStart := fc.eb.text.Len()
	fc.out.CmpRegToImm("rcx", 32)
	loopEndJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	fc.out.MovMemToReg("rax", "rbx", 0)
	fc.out.CmpRegToReg("rax", "r13")
	foundJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)

	fc.out.AddImmToReg("rbx", 16)
	fc.out.AddImmToReg("rcx", 1)
	backJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(int32(loopStart - (backJump + 5)))

	foundLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(foundJump+2, int32(foundLabel-(foundJump+6)))
	fc.out.LeaMemToReg("rax", "rbx", 8)

	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	notInitLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(notInitJump+2, int32(notInitLabel-(notInitJump+6)))

	loopEndLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(loopEndJump+2, int32(loopEndLabel-(loopEndJump+6)))
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()
}

func (fc *FlapCompiler) generateCacheInsert() {
	fc.eb.MarkLabel("flap_cache_insert")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")
	fc.out.SubImmFromReg("rsp", 8)

	fc.out.MovRegToReg("r12", "rdi")
	fc.out.MovRegToReg("r13", "rsi")
	fc.out.MovRegToReg("r14", "rdx")

	fc.out.MovMemToReg("rax", "r12", 0)
	fc.out.CmpRegToImm("rax", 0)
	alreadyInitJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)

	fc.out.MovImmToReg("rdi", "512")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToMem("rax", "r12", 0)
	fc.out.MovImmToReg("rax", "32")
	fc.out.MovRegToMem("rax", "r12", 8)

	alreadyInitLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(alreadyInitJump+2, int32(alreadyInitLabel-(alreadyInitJump+6)))

	fc.out.MovMemToReg("rdi", "r12", 0)

	fc.out.MovRegToReg("rax", "r13")
	fc.out.AndRegWithImm("rax", 31)

	fc.out.Emit([]byte{0x48, 0xc1, 0xe0, 0x04})
	fc.out.AddRegToReg("rax", "rdi")
	fc.out.MovRegToMem("r13", "rax", 0)
	fc.out.MovRegToMem("r14", "rax", 8)

	fc.out.AddImmToReg("rsp", 8)
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()
}

func (fc *FlapCompiler) generateRuntimeHelpers() {
	fc.eb.EmitArenaRuntimeCode()

	if fc.usesArenas {
		fc.eb.Define("_flap_arena_meta", "\x00\x00\x00\x00\x00\x00\x00\x00")
		fc.eb.Define("_flap_arena_meta_cap", "\x00\x00\x00\x00\x00\x00\x00\x00")
		fc.eb.Define("_flap_arena_meta_len", "\x00\x00\x00\x00\x00\x00\x00\x00")
	}

	fc.generateCacheLookup()
	fc.generateCacheInsert()

	for lambdaName := range fc.cacheEnabledLambdas {
		cacheName := lambdaName + "_cache"
		fc.eb.Define(cacheName, "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00")
	}

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
	fc.out.SubImmFromReg("rsp", StackSlotSize)

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
	fc.out.AddImmToReg("rsp", StackSlotSize)

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
	fc.out.PushReg("r14") // Save r14 for count (callee-saved, won't be clobbered by malloc)

	// Still need alignment: 5 pushes + rbp = 6 pushes = 48 bytes
	// Need to align to 16 bytes before calls, so add 8 bytes
	fc.out.SubImmFromReg("rsp", StackSlotSize)

	// Convert float64 pointer to integer pointer in r12
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)
	fc.out.MovMemToReg("r12", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)

	// Get string length from map: count = [r12+0]
	fc.out.MovMemToXmm("xmm0", "r12", 0)
	fc.out.Emit([]byte{0xf2, 0x4c, 0x0f, 0x2c, 0xf0}) // cvttsd2si r14, xmm0 (r14 = count, callee-saved)

	// Allocate memory: malloc(count + 1) for null terminator
	fc.out.MovRegToReg("rdi", "r14")
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x01}) // add rdi, 1
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax") // r13 = C string buffer

	// Initialize: rbx = current index, r12 = map ptr, r13 = cstr ptr, r14 = count
	fc.out.XorRegWithReg("rbx", "rbx") // rbx = 0 (current index)

	// Loop through map entries to extract characters
	fc.eb.MarkLabel("_cstr_convert_loop")
	fc.out.Emit([]byte{0x4c, 0x39, 0xf3}) // cmp rbx, r14
	fc.out.Emit([]byte{0x74, 0x27})       // je +39 bytes (adjusted for r14: skip loop body)

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
	// r13 requires special handling - use mod=01 with disp8=0
	fc.out.Emit([]byte{0x41, 0x88, 0x44, 0x1d, 0x00}) // mov [r13 + rbx + 0], al

	// Increment index
	fc.out.Emit([]byte{0x48, 0xff, 0xc3}) // inc rbx
	fc.out.Emit([]byte{0xeb, 0xd4})       // jmp _cstr_convert_loop (-44 bytes, adjusted for r14)

	// Add null terminator: [r13 + r14] = 0
	// r13 requires mod=01 with disp8=0, r14 as index register
	fc.out.Emit([]byte{0x43, 0xc6, 0x44, 0x35, 0x00, 0x00}) // mov byte [r13 + r14 + 0], 0

	// Return C string pointer in rax
	fc.out.MovRegToReg("rax", "r13")

	// Restore stack alignment
	fc.out.AddImmToReg("rsp", StackSlotSize)

	// Restore callee-saved registers
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")

	// Function epilogue
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate cstr_to_flap_string(cstr_ptr) -> flap_string_ptr
	// Converts a null-terminated C string to a Flap string (map format)
	// Argument: rdi = C string pointer
	// Returns: xmm0 = Flap string pointer (as float64)
	fc.eb.MarkLabel("cstr_to_flap_string")

	// Function prologue
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")

	// Save callee-saved registers
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")

	// Stack is now 16-byte aligned (call pushed 8, then 5 pushes = 48 bytes total)
	// No additional alignment needed

	// Save C string pointer
	fc.out.MovRegToReg("r12", "rdi") // r12 = C string pointer

	// Calculate string length using strlen(r12)
	fc.out.MovRegToReg("rdi", "r12") // Set argument for strlen
	fc.trackFunctionCall("strlen")
	fc.eb.GenerateCallInstruction("strlen")
	fc.out.MovRegToReg("r14", "rax") // r14 = string length

	// Allocate Flap string map: 8 + (length * 16) bytes
	// count (8 bytes) + (key, value) pairs (16 bytes each)
	fc.out.MovRegToReg("rdi", "r14")
	fc.out.Emit([]byte{0x48, 0xc1, 0xe7, 0x04}) // shl rdi, 4 (multiply by 16)
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x08}) // add rdi, 8
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax") // r13 = Flap string map pointer

	// Store count in map[0]
	fc.out.MovRegToReg("rax", "r14")
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r13", 0)

	// Fill map with character data
	fc.out.XorRegWithReg("rbx", "rbx") // rbx = index

	// Loop: for each character
	cstrLoopStart := fc.eb.text.Len()
	fc.eb.MarkLabel("_cstr_to_flap_loop")

	// Compare index with length
	fc.out.Emit([]byte{0x4c, 0x39, 0xf3}) // cmp rbx, r14
	cstrExitJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // je to exit (will patch later)

	// Load character from C string: al = [r12 + rbx]
	fc.out.Emit([]byte{0x41, 0x8a, 0x04, 0x1c}) // mov al, [r12 + rbx]

	// Convert character to float64
	fc.out.Emit([]byte{0x48, 0x0f, 0xb6, 0xc0}) // movzx rax, al
	fc.out.Cvtsi2sd("xmm0", "rax")

	// Convert index to float64 for key
	fc.out.MovRegToReg("rdx", "rbx")
	fc.out.Cvtsi2sd("xmm1", "rdx")

	// Calculate offset for entry: 8 + (rbx * 16)
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.Emit([]byte{0x48, 0xc1, 0xe0, 0x04}) // shl rax, 4
	fc.out.Emit([]byte{0x48, 0x83, 0xc0, 0x08}) // add rax, 8

	// Add offset to base pointer: rax = r13 + rax
	fc.out.Emit([]byte{0x4c, 0x01, 0xe8}) // add rax, r13

	// Store key (index): [rax] = xmm1
	fc.out.Emit([]byte{0xf2, 0x0f, 0x11, 0x08}) // movsd [rax], xmm1

	// Store value (character): [rax + 8] = xmm0
	fc.out.Emit([]byte{0xf2, 0x0f, 0x11, 0x40, 0x08}) // movsd [rax + 8], xmm0

	// Increment index
	fc.out.Emit([]byte{0x48, 0xff, 0xc3}) // inc rbx

	// Jump back to loop start
	cstrLoopEnd := fc.eb.text.Len()
	cstrOffset := int32(cstrLoopStart - (cstrLoopEnd + 2))
	fc.out.Emit([]byte{0xeb, byte(cstrOffset)}) // jmp rel8

	// Patch the exit jump
	cstrExitPos := fc.eb.text.Len()
	fc.patchJumpImmediate(cstrExitJumpPos+2, int32(cstrExitPos-(cstrExitJumpPos+6)))

	// Return Flap string pointer in xmm0
	fc.out.MovRegToXmm("xmm0", "r13")

	// Restore callee-saved registers (no stack adjustment needed)
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")

	// Function epilogue
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate flap_slice_string(str_ptr, start, end, step) -> new_str_ptr
	// Arguments: rdi = string_ptr, rsi = start_index (int64), rdx = end_index (int64), rcx = step (int64)
	// Returns: rax = pointer to new sliced string
	// String format (map): [count (float64)][key0 (float64)][val0 (float64)]...
	// Note: Currently only step == 1 is fully supported

	fc.eb.MarkLabel("flap_slice_string")

	// Function prologue
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")

	// Save callee-saved registers
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")

	// Save arguments
	fc.out.MovRegToReg("r12", "rdi") // r12 = original string pointer
	fc.out.MovRegToReg("r13", "rsi") // r13 = start index
	fc.out.MovRegToReg("r14", "rdx") // r14 = end index
	fc.out.MovRegToReg("r8", "rcx")  // r8 = step

	// Calculate result length based on step
	// For step == 1: length = end - start
	// For step > 1: length = ((end - start + step - 1) / step)
	// For step < 0: length = ((start - end - step - 1) / (-step))

	fc.out.XorRegWithReg("rax", "rax")
	fc.out.CmpRegToReg("r8", "rax")
	stepNegativeJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLess, 0) // If step < 0, jump to negative path

	// Positive step path
	fc.out.MovImmToReg("rax", "1")
	fc.out.CmpRegToReg("r8", "rax")
	stepOneJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // If step == 1, use simple path

	// Step > 1 path: length = ((end - start + step - 1) / step)
	fc.out.MovRegToReg("r15", "r14")
	fc.out.SubRegFromReg("r15", "r13") // r15 = end - start
	fc.out.AddRegToReg("r15", "r8")    // r15 = end - start + step
	fc.out.SubImmFromReg("r15", 1)     // r15 = end - start + step - 1
	fc.out.MovRegToReg("rax", "r15")
	fc.out.XorRegWithReg("rdx", "rdx")    // Clear rdx for division
	fc.out.Emit([]byte{0x49, 0xF7, 0xF8}) // idiv r8
	fc.out.MovRegToReg("r15", "rax")      // r15 = result length

	stepEndJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // Jump to end

	// Patch step == 1 jump to here
	stepOnePos := fc.eb.text.Len()
	stepOneOffset := int32(stepOnePos - (stepOneJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(stepOneJumpPos+2, stepOneOffset)

	// Step == 1 simple path: length = end - start
	fc.out.MovRegToReg("r15", "r14")
	fc.out.SubRegFromReg("r15", "r13") // r15 = length

	stepPosEndJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // Jump to end

	// Patch negative step jump to here
	stepNegativePos := fc.eb.text.Len()
	stepNegativeOffset := int32(stepNegativePos - (stepNegativeJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(stepNegativeJumpPos+2, stepNegativeOffset)

	// Negative step path: length = ((start - end - step - 1) / (-step))
	fc.out.MovRegToReg("r15", "r13")   // r15 = start
	fc.out.SubRegFromReg("r15", "r14") // r15 = start - end
	fc.out.SubRegFromReg("r15", "r8")  // r15 = start - end - step
	fc.out.SubImmFromReg("r15", 1)     // r15 = start - end - step - 1
	// Divide by -step, so negate r8, divide, then restore r8
	fc.out.MovRegToReg("r10", "r8")       // Save r8
	fc.out.Emit([]byte{0x49, 0xF7, 0xD8}) // neg r8 (r8 = -r8)
	fc.out.MovRegToReg("rax", "r15")
	fc.out.XorRegWithReg("rdx", "rdx")    // Clear rdx for division
	fc.out.Emit([]byte{0x49, 0xF7, 0xF8}) // idiv r8
	fc.out.MovRegToReg("r15", "rax")      // r15 = result length
	fc.out.MovRegToReg("r8", "r10")       // Restore r8

	// Patch end jumps
	stepEndPos := fc.eb.text.Len()
	stepEndOffset := int32(stepEndPos - (stepEndJumpPos + UnconditionalJumpSize))
	fc.patchJumpImmediate(stepEndJumpPos+1, stepEndOffset)

	stepPosEndOffset := int32(stepEndPos - (stepPosEndJumpPos + UnconditionalJumpSize))
	fc.patchJumpImmediate(stepPosEndJumpPos+1, stepPosEndOffset)

	// Allocate memory for new string: 8 + (length * 16) bytes
	fc.out.MovRegToReg("rax", "r15")
	fc.out.ShlRegImm("rax", "4") // shl rax, 4 (multiply by 16)
	fc.out.AddImmToReg("rax", 8) // add rax, 8
	fc.out.MovRegToReg("rdi", "rax")
	// Save r8 (step) before malloc since it's caller-saved
	fc.out.PushReg("r8")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("rbx", "rax") // rbx = new string pointer
	// Restore r8 (step)
	fc.out.PopReg("r8")

	// Store count (length) as float64 in first 8 bytes
	fc.out.Cvtsi2sd("xmm0", "r15") // xmm0 = length as float64
	fc.out.MovXmmToMem("xmm0", "rbx", 0)

	// Copy characters from original string
	// Initialize loop counter (output index): rcx = 0
	fc.out.XorRegWithReg("rcx", "rcx")
	// Initialize source index: r9 = start
	fc.out.MovRegToReg("r9", "r13")

	fc.eb.MarkLabel("_slice_copy_loop")
	sliceLoopStart := fc.eb.text.Len() // Track actual loop start position

	// Check if rcx >= length (exit loop if true)
	fc.out.CmpRegToReg("rcx", "r15")
	loopExitJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpAboveOrEqual, 0) // Placeholder, will patch later

	// Use source index from r9
	fc.out.MovRegToReg("rax", "r9")

	// Calculate source address: r11 = r12 + 8 + (source_idx * 16)
	fc.out.ShlRegImm("rax", "4") // rax = source_idx * 16
	fc.out.AddImmToReg("rax", 8) // rax = source_idx * 16 + 8
	fc.out.MovRegToReg("r11", "r12")
	fc.out.AddRegToReg("r11", "rax") // r11 = r12 + rax

	// Load key and value from source string
	fc.out.MovMemToXmm("xmm0", "r11", 0) // xmm0 = [r11] (key)
	fc.out.MovMemToXmm("xmm1", "r11", 8) // xmm1 = [r11 + 8] (value)

	// Calculate destination address: rdx = 8 + (rcx * 16)
	fc.out.MovRegToReg("rdx", "rcx")
	fc.out.ShlRegImm("rdx", "4") // rdx = rcx * 16
	fc.out.AddImmToReg("rdx", 8) // rdx = rcx * 16 + 8

	// Calculate full destination address: r11 = rbx + rdx
	fc.out.MovRegToReg("r11", "rbx")
	fc.out.AddRegToReg("r11", "rdx") // r11 = rbx + rdx

	// Store key as rcx (new index), and value
	fc.out.Cvtsi2sd("xmm0", "rcx")       // xmm0 = rcx as float64 (new key)
	fc.out.MovXmmToMem("xmm0", "r11", 0) // [r11] = xmm0 (key)
	fc.out.MovXmmToMem("xmm1", "r11", 8) // [r11 + 8] = xmm1 (value)

	// Increment loop counter
	fc.out.IncReg("rcx")

	// Increment source index by step
	fc.out.AddRegToReg("r9", "r8") // r9 = r9 + step

	// Jump back to loop start
	loopBackJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // Placeholder, will patch later

	// Patch loop jumps
	loopExitPos := fc.eb.text.Len()

	// Patch exit jump: JumpConditional emits 6 bytes (0x0f 0x83 + 4-byte offset)
	// Offset is from end of jump instruction to loop exit
	loopExitOffset := int32(loopExitPos - (loopExitJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(loopExitJumpPos+2, loopExitOffset) // +2 to skip 0x0f 0x83 opcode bytes

	// Patch back jump: JumpUnconditional emits 5 bytes (0xe9 + 4-byte offset)
	// Offset is from end of jump instruction back to loop start
	loopBackOffset := int32(sliceLoopStart - (loopBackJumpPos + UnconditionalJumpSize))
	fc.patchJumpImmediate(loopBackJumpPos+1, loopBackOffset) // +1 to skip 0xe9 opcode byte

	// Return new string pointer in rax
	fc.out.MovRegToReg("rax", "rbx")

	// Restore callee-saved registers
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")

	// Function epilogue
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate _flap_list_concat(left_ptr, right_ptr) -> new_ptr
	// Arguments: rdi = left_ptr, rsi = right_ptr
	// Returns: rax = pointer to new concatenated list
	// List format: [length (8 bytes)][elem0 (8 bytes)][elem1 (8 bytes)]...

	fc.eb.MarkLabel("_flap_list_concat")

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
	fc.out.SubImmFromReg("rsp", StackSlotSize)

	// Save arguments
	fc.out.MovRegToReg("r12", "rdi") // r12 = left_ptr
	fc.out.MovRegToReg("r13", "rsi") // r13 = right_ptr

	// Get left list length
	fc.out.MovMemToXmm("xmm0", "r12", 0)              // load length as float64
	fc.out.Emit([]byte{0xf2, 0x4c, 0x0f, 0x2c, 0xf0}) // cvttsd2si r14, xmm0

	// Get right list length
	fc.out.MovMemToXmm("xmm0", "r13", 0)              // load length as float64
	fc.out.Emit([]byte{0xf2, 0x4c, 0x0f, 0x2c, 0xf8}) // cvttsd2si r15, xmm0

	// Calculate total length: rbx = r14 + r15
	fc.out.MovRegToReg("rbx", "r14")
	fc.out.Emit([]byte{0x4c, 0x01, 0xfb}) // add rbx, r15

	// Calculate allocation size: rax = 8 + rbx * 8
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.Emit([]byte{0x48, 0xc1, 0xe0, 0x03}) // shl rax, 3 (multiply by 8)
	fc.out.Emit([]byte{0x48, 0x83, 0xc0, 0x08}) // add rax, 8

	// Align to 16 bytes for safety
	fc.out.Emit([]byte{0x48, 0x83, 0xc0, 0x0f}) // add rax, 15
	fc.out.Emit([]byte{0x48, 0x83, 0xe0, 0xf0}) // and rax, ~15

	// Call malloc(rax)
	fc.out.MovRegToReg("rdi", "rax")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r10", "rax") // r10 = result pointer

	// Write total length to result
	fc.out.Emit([]byte{0xf2, 0x48, 0x0f, 0x2a, 0xc3}) // cvtsi2sd xmm0, rbx
	fc.out.MovXmmToMem("xmm0", "r10", 0)

	// Copy left list elements
	// memcpy(r10 + 8, r12 + 8, r14 * 8)
	fc.out.Emit([]byte{0x4d, 0x89, 0xf1})             // mov r9, r14 (counter)
	fc.out.Emit([]byte{0x49, 0x8d, 0x74, 0x24, 0x08}) // lea rsi, [r12 + 8]
	fc.out.Emit([]byte{0x49, 0x8d, 0x7a, 0x08})       // lea rdi, [r10 + 8]

	// Loop to copy left elements
	fc.eb.MarkLabel("_list_concat_copy_left_loop")
	fc.out.Emit([]byte{0x4d, 0x85, 0xc9}) // test r9, r9
	fc.out.Emit([]byte{0x74, 0x17})       // jz +23 bytes (skip loop body)

	fc.out.MovMemToXmm("xmm0", "rsi", 0)        // load element (4 bytes)
	fc.out.MovXmmToMem("xmm0", "rdi", 0)        // store element (4 bytes)
	fc.out.Emit([]byte{0x48, 0x83, 0xc6, 0x08}) // add rsi, 8 (4 bytes)
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x08}) // add rdi, 8 (4 bytes)
	fc.out.Emit([]byte{0x49, 0xff, 0xc9})       // dec r9 (3 bytes)
	fc.out.Emit([]byte{0xeb, 0xe4})             // jmp back -28 bytes (2 bytes)

	// Copy right list elements
	// memcpy(r10 + 8 + r14*8, r13 + 8, r15 * 8)
	fc.out.Emit([]byte{0x49, 0x8d, 0x75, 0x08}) // lea rsi, [r13 + 8]
	// rdi already points to correct position

	fc.eb.MarkLabel("_list_concat_copy_right_loop")
	fc.out.Emit([]byte{0x4d, 0x85, 0xff}) // test r15, r15
	fc.out.Emit([]byte{0x74, 0x17})       // jz +23 bytes (skip loop body)

	fc.out.MovMemToXmm("xmm0", "rsi", 0)        // load element (4 bytes)
	fc.out.MovXmmToMem("xmm0", "rdi", 0)        // store element (4 bytes)
	fc.out.Emit([]byte{0x48, 0x83, 0xc6, 0x08}) // add rsi, 8 (4 bytes)
	fc.out.Emit([]byte{0x48, 0x83, 0xc7, 0x08}) // add rdi, 8 (4 bytes)
	fc.out.Emit([]byte{0x49, 0xff, 0xcf})       // dec r15 (3 bytes)
	fc.out.Emit([]byte{0xeb, 0xe4})             // jmp back -28 bytes (2 bytes)

	// Return result pointer in rax
	fc.out.MovRegToReg("rax", "r10")

	// Restore stack alignment
	fc.out.AddImmToReg("rsp", StackSlotSize)

	// Restore callee-saved registers
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")

	// Function epilogue
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate _flap_string_eq(left_ptr, right_ptr) -> 1.0 or 0.0
	// Arguments: rdi = left_ptr, rsi = right_ptr
	// Returns: xmm0 = 1.0 if equal, 0.0 if not
	// String format: [count (8 bytes)][key0 (8)][val0 (8)][key1 (8)][val1 (8)]...

	fc.eb.MarkLabel("_flap_string_eq")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")

	// rdi = left_ptr, rsi = right_ptr
	// Check if both are null (empty strings)
	fc.out.MovRegToReg("rax", "rdi")
	fc.out.OrRegToReg("rax", "rsi")
	fc.out.TestRegReg("rax", "rax")
	eqNullJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // If both null, they're equal

	// Check if only one is null
	fc.out.TestRegReg("rdi", "rdi")
	neqJumpPos1 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // left is null but right isn't

	fc.out.TestRegReg("rsi", "rsi")
	neqJumpPos2 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // right is null but left isn't

	// Both non-null, load counts
	fc.out.MovMemToXmm("xmm0", "rdi", 0) // left count
	fc.out.MovMemToXmm("xmm1", "rsi", 0) // right count

	// Convert counts to integers for comparison
	fc.out.Cvttsd2si("r12", "xmm0") // left count in r12
	fc.out.Cvttsd2si("r13", "xmm1") // right count in r13

	// Compare counts
	fc.out.CmpRegToReg("r12", "r13")
	neqJumpPos3 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0) // If counts differ, not equal

	// Counts are equal, compare each character
	// rbx = index counter
	fc.out.XorRegWithReg("rbx", "rbx")

	loopStart := fc.eb.text.Len()

	// Check if we've compared all characters
	fc.out.CmpRegToReg("rbx", "r12")
	endLoopJumpPos := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	// Calculate offset: 8 + rbx * 16 (count is 8 bytes, each key-value pair is 16 bytes)
	// Actually, format is [count][key0][val0][key1][val1]...
	// So to get value at index i: offset = 8 + i*16 + 8 = 16 + i*16
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.ShlRegImm("rax", "4")  // multiply by 16
	fc.out.AddImmToReg("rax", 16) // skip count (8) and key (8)

	// Load characters
	fc.out.Comment("Load left[rbx] and right[rbx]")
	fc.out.MovRegToReg("r8", "rdi")
	fc.out.AddRegToReg("r8", "rax")
	fc.out.MovMemToXmm("xmm2", "r8", 0)

	fc.out.MovRegToReg("r9", "rsi")
	fc.out.AddRegToReg("r9", "rax")
	fc.out.MovMemToXmm("xmm3", "r9", 0)

	// Compare characters
	fc.out.Ucomisd("xmm2", "xmm3")
	neqJumpPos4 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)

	// Increment index and continue
	fc.out.AddImmToReg("rbx", 1)
	loopJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // jump back to loop start

	// Patch loop jump
	offset := int32(loopStart - (loopJumpPos + UnconditionalJumpSize))
	fc.patchJumpImmediate(loopJumpPos+1, offset)

	// All characters matched - return 1.0
	endLoopLabel := fc.eb.text.Len()
	eqNullLabel := fc.eb.text.Len() // Same position as endLoopLabel
	fc.out.MovImmToReg("rax", "1")
	fc.out.Cvtsi2sd("xmm0", "rax")
	doneJumpPos := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)

	// Not equal - return 0.0
	neqLabel := fc.eb.text.Len()
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.Cvtsi2sd("xmm0", "rax")

	// Done label
	doneLabel := fc.eb.text.Len()

	// Patch all jumps
	// Patch eqNull jump to eqNullLabel
	offset = int32(eqNullLabel - (eqNullJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(eqNullJumpPos+2, offset)

	// Patch neq jumps to neqLabel
	offset = int32(neqLabel - (neqJumpPos1 + 6))
	fc.patchJumpImmediate(neqJumpPos1+2, offset)

	offset = int32(neqLabel - (neqJumpPos2 + 6))
	fc.patchJumpImmediate(neqJumpPos2+2, offset)

	offset = int32(neqLabel - (neqJumpPos3 + 6))
	fc.patchJumpImmediate(neqJumpPos3+2, offset)

	offset = int32(neqLabel - (neqJumpPos4 + 6))
	fc.patchJumpImmediate(neqJumpPos4+2, offset)

	// Patch endLoop jump to endLoopLabel
	offset = int32(endLoopLabel - (endLoopJumpPos + ConditionalJumpSize))
	fc.patchJumpImmediate(endLoopJumpPos+2, offset)

	// Patch done jump to doneLabel
	offset = int32(doneLabel - (doneJumpPos + UnconditionalJumpSize))
	fc.patchJumpImmediate(doneJumpPos+1, offset)

	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate upper_string(flap_string_ptr) -> uppercase_flap_string_ptr
	// Converts a Flap string to uppercase
	// Argument: rdi = Flap string pointer (as integer)
	// Returns: xmm0 = uppercase Flap string pointer (as float64)
	fc.eb.MarkLabel("upper_string")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")

	fc.out.MovRegToReg("r12", "rdi") // r12 = input string

	// Get string length
	fc.out.MovMemToXmm("xmm0", "r12", 0)
	fc.out.Cvttsd2si("r14", "xmm0") // r14 = count

	// Allocate new string map
	fc.out.MovRegToReg("rax", "r14")
	fc.out.ShlRegImm("rax", "4") // rax = count * 16
	fc.out.AddImmToReg("rax", 8)
	fc.out.MovRegToReg("rdi", "rax")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax") // r13 = output string

	// Copy count
	fc.out.MovRegToReg("rax", "r14")
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r13", 0)

	// Loop through characters
	fc.out.XorRegWithReg("rbx", "rbx") // rbx = loop counter
	upperLoopStart := fc.eb.text.Len()
	fc.out.CmpRegToReg("rbx", "r14")
	upperLoopEnd := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	// Calculate offset: rax = 8 + rbx*16
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.ShlRegImm("rax", "4") // rax = rbx * 16
	fc.out.AddImmToReg("rax", 8) // rax = 8 + rbx * 16

	// Calculate source address: r15 = r12 + rax
	fc.out.MovRegToReg("r15", "r12")
	fc.out.AddRegToReg("r15", "rax")

	// Calculate dest address: r10 = r13 + rax
	fc.out.MovRegToReg("r10", "r13")
	fc.out.AddRegToReg("r10", "rax")

	// Copy key (index)
	fc.out.MovMemToXmm("xmm0", "r15", 0)
	fc.out.MovXmmToMem("xmm0", "r10", 0)

	// Load character value and convert
	fc.out.MovMemToXmm("xmm0", "r15", 8)
	fc.out.Cvttsd2si("rax", "xmm0") // Use rax for the character value

	// Convert to uppercase: if (c >= 'a' && c <= 'z') c -= 32
	fc.out.CmpRegToImm("rax", int64('a'))
	notLowerJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLess, 0)
	fc.out.CmpRegToImm("rax", int64('z'))
	notLowerJump2 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreater, 0)
	fc.out.SubImmFromReg("rax", 32)

	// Store uppercase character
	notLowerPos := fc.eb.text.Len()
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r10", 8)

	fc.out.IncReg("rbx")
	jumpBack := int32(upperLoopStart - (fc.eb.text.Len() + 5))
	fc.out.JumpUnconditional(jumpBack)

	upperDone := fc.eb.text.Len()
	fc.patchJumpImmediate(upperLoopEnd+2, int32(upperDone-(upperLoopEnd+6)))
	fc.patchJumpImmediate(notLowerJump+2, int32(notLowerPos-(notLowerJump+6)))
	fc.patchJumpImmediate(notLowerJump2+2, int32(notLowerPos-(notLowerJump2+6)))

	fc.out.MovRegToXmm("xmm0", "r13")
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate lower_string(flap_string_ptr) -> lowercase_flap_string_ptr
	// Converts a Flap string to lowercase
	// Argument: rdi = Flap string pointer (as integer)
	// Returns: xmm0 = lowercase Flap string pointer (as float64)
	fc.eb.MarkLabel("lower_string")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")

	fc.out.MovRegToReg("r12", "rdi") // r12 = input string

	// Get string length
	fc.out.MovMemToXmm("xmm0", "r12", 0)
	fc.out.Cvttsd2si("r14", "xmm0") // r14 = count

	// Allocate new string map
	fc.out.MovRegToReg("rax", "r14")
	fc.out.ShlRegImm("rax", "4") // rax = count * 16
	fc.out.AddImmToReg("rax", 8)
	fc.out.MovRegToReg("rdi", "rax")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax") // r13 = output string

	// Copy count
	fc.out.MovRegToReg("rax", "r14")
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r13", 0)

	// Loop through characters
	fc.out.XorRegWithReg("rbx", "rbx") // rbx = loop counter
	lowerLoopStart := fc.eb.text.Len()
	fc.out.CmpRegToReg("rbx", "r14")
	lowerLoopEnd := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	// Calculate offset: rax = 8 + rbx*16
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.ShlRegImm("rax", "4") // rax = rbx * 16
	fc.out.AddImmToReg("rax", 8) // rax = 8 + rbx * 16

	// Calculate source address: r15 = r12 + rax
	fc.out.MovRegToReg("r15", "r12")
	fc.out.AddRegToReg("r15", "rax")

	// Calculate dest address: r10 = r13 + rax
	fc.out.MovRegToReg("r10", "r13")
	fc.out.AddRegToReg("r10", "rax")

	// Copy key (index)
	fc.out.MovMemToXmm("xmm0", "r15", 0)
	fc.out.MovXmmToMem("xmm0", "r10", 0)

	// Load character value and convert
	fc.out.MovMemToXmm("xmm0", "r15", 8)
	fc.out.Cvttsd2si("rax", "xmm0") // Use rax for the character value

	// Convert to lowercase: if (c >= 'A' && c <= 'Z') c += 32
	fc.out.CmpRegToImm("rax", int64('A'))
	notUpperJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLess, 0)
	fc.out.CmpRegToImm("rax", int64('Z'))
	notUpperJump2 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreater, 0)
	fc.out.AddImmToReg("rax", 32)

	// Store lowercase character
	notUpperPos := fc.eb.text.Len()
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r10", 8)

	fc.out.IncReg("rbx")
	jumpBack = int32(lowerLoopStart - (fc.eb.text.Len() + 5))
	fc.out.JumpUnconditional(jumpBack)

	lowerDone := fc.eb.text.Len()
	fc.patchJumpImmediate(lowerLoopEnd+2, int32(lowerDone-(lowerLoopEnd+6)))
	fc.patchJumpImmediate(notUpperJump+2, int32(notUpperPos-(notUpperJump+6)))
	fc.patchJumpImmediate(notUpperJump2+2, int32(notUpperPos-(notUpperJump2+6)))

	fc.out.MovRegToXmm("xmm0", "r13")
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate trim_string(flap_string_ptr) -> trimmed_flap_string_ptr
	// Removes leading and trailing whitespace
	// Argument: rdi = Flap string pointer (as integer)
	// Returns: xmm0 = trimmed Flap string pointer (as float64)
	fc.eb.MarkLabel("trim_string")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")

	fc.out.MovRegToReg("r12", "rdi") // r12 = input string

	// Get string length
	fc.out.MovMemToXmm("xmm0", "r12", 0)
	fc.out.Cvttsd2si("r14", "xmm0") // r14 = original count

	// Find start (skip leading whitespace)
	fc.out.XorRegWithReg("rbx", "rbx") // rbx = start index
	trimStartLoop := fc.eb.text.Len()
	fc.out.Emit([]byte{0x4c, 0x39, 0xf3}) // cmp rbx, r14
	trimStartDone := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	// Load character at rbx
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.ShlRegImm("rax", "4") // rax = rbx * 16
	fc.out.AddImmToReg("rax", 8) // rax = 8 + rbx * 16
	fc.out.MovRegToReg("r8", "r12")
	fc.out.AddRegToReg("r8", "rax")     // r8 = r12 + offset
	fc.out.MovMemToXmm("xmm0", "r8", 8) // Load value
	fc.out.Cvttsd2si("r10", "xmm0")

	// Check if whitespace (space=32, tab=9, newline=10, cr=13)
	fc.out.CmpRegToImm("r10", 32)
	notWhitespace1 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	fc.out.IncReg("rbx")
	jumpStartLoop := int32(trimStartLoop - (fc.eb.text.Len() + 2))
	fc.out.Emit([]byte{0xeb, byte(jumpStartLoop)})

	notWS1Pos := fc.eb.text.Len()
	fc.out.CmpRegToImm("r10", 9)
	notWhitespace2 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	fc.out.IncReg("rbx")
	jumpStartLoop2 := int32(trimStartLoop - (fc.eb.text.Len() + 2))
	fc.out.Emit([]byte{0xeb, byte(jumpStartLoop2)})

	notWS2Pos := fc.eb.text.Len()
	fc.out.CmpRegToImm("r10", 10)
	notWhitespace3 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	fc.out.IncReg("rbx")
	jumpStartLoop3 := int32(trimStartLoop - (fc.eb.text.Len() + 2))
	fc.out.Emit([]byte{0xeb, byte(jumpStartLoop3)})

	notWS3Pos := fc.eb.text.Len()
	fc.out.CmpRegToImm("r10", 13)
	trimStartFound := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)
	fc.out.IncReg("rbx")
	jumpStartLoop4 := int32(trimStartLoop - (fc.eb.text.Len() + 2))
	fc.out.Emit([]byte{0xeb, byte(jumpStartLoop4)})

	// Start found - rbx = start index
	trimFoundStart := fc.eb.text.Len()
	fc.patchJumpImmediate(trimStartDone+2, int32(trimFoundStart-(trimStartDone+6)))
	fc.patchJumpImmediate(notWhitespace1+2, int32(notWS1Pos-(notWhitespace1+6)))
	fc.patchJumpImmediate(notWhitespace2+2, int32(notWS2Pos-(notWhitespace2+6)))
	fc.patchJumpImmediate(notWhitespace3+2, int32(notWS3Pos-(notWhitespace3+6)))
	fc.patchJumpImmediate(trimStartFound+2, int32(trimFoundStart-(trimStartFound+6)))

	// Find end (skip trailing whitespace) - work backwards from r14-1
	fc.out.MovRegToReg("r15", "r14") // r15 = end index (exclusive)
	// Handle empty or all-whitespace case
	fc.out.Emit([]byte{0x4c, 0x39, 0xfb}) // cmp rbx, r15
	emptyResult := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	trimEndLoop := fc.eb.text.Len()
	fc.out.Emit([]byte{0x49, 0x83, 0xff, 0x00}) // cmp r15, 0
	trimEndDone := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLessOrEqual, 0)
	fc.out.Emit([]byte{0x49, 0x83, 0xef, 0x01}) // dec r15

	// Load character at r15
	fc.out.MovRegToReg("rax", "r15")
	fc.out.ShlRegImm("rax", "4") // rax = r15 * 16
	fc.out.AddImmToReg("rax", 8) // rax = 8 + r15 * 16
	fc.out.MovRegToReg("r8", "r12")
	fc.out.AddRegToReg("r8", "rax")     // r8 = r12 + offset
	fc.out.MovMemToXmm("xmm0", "r8", 8) // Load value
	fc.out.Cvttsd2si("r10", "xmm0")

	// Check if whitespace
	fc.out.CmpRegToImm("r10", 32)
	trimWSJump1 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	fc.out.CmpRegToImm("r10", 9)
	trimWSJump2 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	fc.out.CmpRegToImm("r10", 10)
	trimWSJump3 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)
	fc.out.CmpRegToImm("r10", 13)
	trimWSJump4 := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0)

	// Not whitespace - found end
	fc.out.IncReg("r15") // Make exclusive
	trimFoundEnd := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)

	// Was whitespace - continue loop
	trimWSTarget := fc.eb.text.Len()
	jumpEndLoop := int32(trimEndLoop - (fc.eb.text.Len() + 2))
	fc.out.Emit([]byte{0xeb, byte(jumpEndLoop)})

	// Patch jumps
	trimRealEnd := fc.eb.text.Len()
	fc.patchJumpImmediate(trimEndDone+2, int32(trimRealEnd-(trimEndDone+6)))
	fc.patchJumpImmediate(trimWSJump1+2, int32(trimWSTarget-(trimWSJump1+6)))
	fc.patchJumpImmediate(trimWSJump2+2, int32(trimWSTarget-(trimWSJump2+6)))
	fc.patchJumpImmediate(trimWSJump3+2, int32(trimWSTarget-(trimWSJump3+6)))
	fc.patchJumpImmediate(trimWSJump4+2, int32(trimWSTarget-(trimWSJump4+6)))
	fc.patchJumpImmediate(trimFoundEnd+1, int32(trimRealEnd-(trimFoundEnd+5)))

	// Now rbx = start, r15 = end (exclusive), create substring
	// new_len = r15 - rbx
	fc.out.MovRegToReg("r14", "r15")
	fc.out.SubRegFromReg("r14", "rbx")

	// Allocate new string
	fc.out.MovRegToReg("rdi", "r14")
	fc.out.ShlRegImm("rdi", "4") // rdi = r14 * 16
	fc.out.AddImmToReg("rdi", 8) // rdi = r14 * 16 + 8
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax")

	// Copy count
	fc.out.MovRegToReg("rax", "r14")
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r13", 0)

	// Copy characters from rbx to r15
	fc.out.XorRegWithReg("rcx", "rcx") // rcx = output index
	trimCopyLoop := fc.eb.text.Len()
	fc.out.Emit([]byte{0x4c, 0x39, 0xf1}) // cmp rcx, r14
	trimCopyDone := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0)

	// Calculate source offset (rbx + rcx)
	fc.out.MovRegToReg("rax", "rbx")
	fc.out.AddRegToReg("rax", "rcx") // rax = rbx + rcx (source index)
	fc.out.ShlRegImm("rax", "4")     // rax = (rbx + rcx) * 16
	fc.out.AddImmToReg("rax", 8)     // rax = (rbx + rcx) * 16 + 8

	// Calculate dest offset (rcx)
	fc.out.MovRegToReg("rdx", "rcx")
	fc.out.ShlRegImm("rdx", "4") // rdx = rcx * 16
	fc.out.AddImmToReg("rdx", 8) // rdx = rcx * 16 + 8

	// Calculate source and dest addresses
	fc.out.MovRegToReg("r8", "r12")
	fc.out.AddRegToReg("r8", "rax") // r8 = source base + offset
	fc.out.MovRegToReg("r9", "r13")
	fc.out.AddRegToReg("r9", "rdx") // r9 = dest base + offset

	// Copy key
	fc.out.Cvtsi2sd("xmm0", "rcx")
	fc.out.MovXmmToMem("xmm0", "r9", 0)

	// Copy value
	fc.out.MovMemToXmm("xmm0", "r8", 8)
	fc.out.MovXmmToMem("xmm0", "r9", 8)

	fc.out.IncReg("rcx")
	jumpCopyLoop := int32(trimCopyLoop - (fc.eb.text.Len() + 2))
	fc.out.Emit([]byte{0xeb, byte(jumpCopyLoop)})

	// Handle empty result case
	emptyPos := fc.eb.text.Len()
	fc.patchJumpImmediate(emptyResult+2, int32(emptyPos-(emptyResult+6)))

	// Allocate empty string
	fc.out.MovImmToReg("rdi", "8")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("r13", "rax")
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.Cvtsi2sd("xmm0", "rax")
	fc.out.MovXmmToMem("xmm0", "r13", 0)

	// Done
	trimAllDone := fc.eb.text.Len()
	fc.patchJumpImmediate(trimCopyDone+2, int32(trimAllDone-(trimCopyDone+6)))

	fc.out.MovRegToXmm("xmm0", "r13")
	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate flap_arena_create(capacity) -> arena_ptr
	// Creates a new arena with the specified capacity
	// Argument: rdi = capacity (int64)
	// Returns: rax = arena pointer
	// Arena structure: [buffer_ptr (8)][capacity (8)][offset (8)][alignment (8)] = 32 bytes header
	fc.eb.MarkLabel("flap_arena_create")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")

	// Save capacity argument
	fc.out.MovRegToReg("r12", "rdi") // r12 = capacity

	// Allocate arena structure (32 bytes)
	fc.out.MovImmToReg("rdi", "32")
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")
	fc.out.MovRegToReg("rbx", "rax") // rbx = arena struct pointer

	// Allocate arena buffer
	fc.out.MovRegToReg("rdi", "r12") // rdi = capacity
	fc.trackFunctionCall("malloc")
	fc.eb.GenerateCallInstruction("malloc")

	// Fill arena structure
	fc.out.MovRegToMem("rax", "rbx", 0)      // [rbx+0] = buffer_ptr
	fc.out.MovRegToMem("r12", "rbx", 8)      // [rbx+8] = capacity
	fc.out.MovImmToMem(0, "rbx", 16)         // [rbx+16] = offset (0)
	fc.out.MovImmToMem(8, "rbx", 24)         // [rbx+24] = alignment (8)

	// Return arena pointer in rax
	fc.out.MovRegToReg("rax", "rbx")

	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate flap_arena_alloc(arena_ptr, size) -> allocation_ptr
	// Allocates memory from the arena using bump allocation with auto-growing
	// If arena is full, reallocs buffer to 2x size
	// Arguments: rdi = arena_ptr, rsi = size (int64)
	// Returns: rax = allocated memory pointer
	fc.eb.MarkLabel("flap_arena_alloc")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14") // Extra push for 16-byte stack alignment (5 total pushes = 40 bytes)

	fc.out.MovRegToReg("rbx", "rdi") // rbx = arena_ptr (preserve across calls)
	fc.out.MovRegToReg("r12", "rsi") // r12 = size (preserve across calls)

	// Load arena fields
	fc.out.MovMemToReg("r8", "rbx", 0)   // r8 = buffer_ptr
	fc.out.MovMemToReg("r9", "rbx", 8)   // r9 = capacity
	fc.out.MovMemToReg("r10", "rbx", 16) // r10 = current offset
	fc.out.MovMemToReg("r11", "rbx", 24) // r11 = alignment

	// Align offset: aligned_offset = (offset + alignment - 1) & ~(alignment - 1)
	fc.out.MovRegToReg("rax", "r10")      // rax = offset
	fc.out.AddRegToReg("rax", "r11")      // rax = offset + alignment
	fc.out.SubImmFromReg("rax", 1)        // rax = offset + alignment - 1
	fc.out.MovRegToReg("rcx", "r11")      // rcx = alignment
	fc.out.SubImmFromReg("rcx", 1)        // rcx = alignment - 1
	fc.out.Emit([]byte{0x48, 0xf7, 0xd1}) // not rcx
	fc.out.Emit([]byte{0x48, 0x21, 0xc8}) // and rax, rcx (aligned_offset in rax)
	fc.out.MovRegToReg("r13", "rax")      // r13 = aligned_offset

	// Check if we have enough space: if (aligned_offset + size > capacity) grow
	fc.out.MovRegToReg("rdx", "r13")  // rdx = aligned_offset
	fc.out.AddRegToReg("rdx", "r12")  // rdx = aligned_offset + size
	fc.out.CmpRegToReg("rdx", "r9")   // compare with capacity
	arenaGrowJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreater, 0) // jg to grow path

	// Fast path: enough space, no need to grow
	fc.eb.MarkLabel("_arena_alloc_fast")
	fc.out.MovRegToReg("rax", "r8")    // rax = buffer_ptr
	fc.out.AddRegToReg("rax", "r13")   // rax = buffer_ptr + aligned_offset

	// Update arena offset
	fc.out.MovRegToReg("rdx", "r13")   // rdx = aligned_offset
	fc.out.AddRegToReg("rdx", "r12")   // rdx = aligned_offset + size
	fc.out.MovRegToMem("rdx", "rbx", 16) // [arena_ptr+16] = new offset

	arenaDoneJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // jmp to done

	// Grow path: realloc buffer to 2x size
	arenaGrowLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(arenaGrowJump+2, int32(arenaGrowLabel-(arenaGrowJump+6)))
	fc.eb.MarkLabel("_arena_alloc_grow")

	// Calculate new capacity: max(capacity * 2, aligned_offset + size)
	fc.out.MovRegToReg("rdi", "r9")    // rdi = capacity
	fc.out.AddRegToReg("rdi", "r9")    // rdi = capacity * 2
	fc.out.MovRegToReg("rsi", "r13")   // rsi = aligned_offset
	fc.out.AddRegToReg("rsi", "r12")   // rsi = aligned_offset + size
	fc.out.CmpRegToReg("rdi", "rsi")   // compare 2*capacity with needed
	skipMaxJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreaterOrEqual, 0) // jge skip_max
	fc.out.MovRegToReg("rdi", "rsi")   // rdi = max(2*capacity, needed)
	skipMaxLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(skipMaxJump+2, int32(skipMaxLabel-(skipMaxJump+6)))

	// rdi now contains new_capacity
	fc.out.MovRegToReg("r9", "rdi")    // r9 = new_capacity (update)

	// Call realloc(buffer_ptr, new_capacity)
	fc.out.MovRegToReg("rdi", "r8")    // rdi = old buffer_ptr
	fc.out.MovRegToReg("rsi", "r9")    // rsi = new_capacity
	fc.trackFunctionCall("realloc")
	fc.eb.GenerateCallInstruction("realloc")

	// Check if realloc failed (returns NULL)
	fc.out.TestRegReg("rax", "rax")
	arenaErrorJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // je to error (realloc failed - rax==0)

	// Realloc succeeded: update arena structure
	fc.out.MovRegToMem("rax", "rbx", 0) // [arena_ptr+0] = new buffer_ptr
	fc.out.MovRegToMem("r9", "rbx", 8)  // [arena_ptr+8] = new capacity
	fc.out.MovRegToReg("r8", "rax")     // r8 = new buffer_ptr

	// Now allocate from the grown arena
	fc.out.MovRegToReg("rax", "r8")     // rax = buffer_ptr
	fc.out.AddRegToReg("rax", "r13")    // rax = buffer_ptr + aligned_offset
	fc.out.MovRegToReg("rdx", "r13")    // rdx = aligned_offset
	fc.out.AddRegToReg("rdx", "r12")    // rdx = aligned_offset + size
	fc.out.MovRegToMem("rdx", "rbx", 16) // [arena_ptr+16] = new offset

	arenaDoneJump2 := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // jmp to done

	// Error path: realloc failed
	arenaErrorLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(arenaErrorJump+2, int32(arenaErrorLabel-(arenaErrorJump+6)))
	fc.eb.MarkLabel("_arena_alloc_error")

	// Print error message and exit(1)
	// TODO: Integrate with or! error handling
	fc.trackFunctionCall("exit")
	fc.out.MovImmToReg("rdi", "1")
	fc.eb.GenerateCallInstruction("exit")

	// Done label
	arenaDoneLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(arenaDoneJump+1, int32(arenaDoneLabel-(arenaDoneJump+5)))
	fc.patchJumpImmediate(arenaDoneJump2+1, int32(arenaDoneLabel-(arenaDoneJump2+5)))
	fc.eb.MarkLabel("_arena_alloc_done")

	fc.out.PopReg("r14") // Pop extra register for stack alignment
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate flap_arena_destroy(arena_ptr)
	// Frees all memory associated with the arena
	// Argument: rdi = arena_ptr
	fc.eb.MarkLabel("flap_arena_destroy")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")

	fc.out.MovRegToReg("rbx", "rdi") // rbx = arena_ptr

	// Free buffer
	fc.out.MovMemToReg("rdi", "rbx", 0) // rdi = buffer_ptr
	fc.trackFunctionCall("free")
	fc.eb.GenerateCallInstruction("free")

	// Free arena structure
	fc.out.MovRegToReg("rdi", "rbx") // rdi = arena_ptr
	fc.trackFunctionCall("free")
	fc.eb.GenerateCallInstruction("free")

	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate flap_arena_reset(arena_ptr)
	// Resets the arena offset to 0, effectively freeing all allocations
	// Argument: rdi = arena_ptr
	fc.eb.MarkLabel("flap_arena_reset")

	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")

	// Reset offset to 0
	fc.out.MovImmToMem(0, "rdi", 16) // [arena_ptr+16] = 0

	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Generate _flap_arena_ensure_capacity if arenas are used
	if fc.usesArenas {
		fc.generateArenaEnsureCapacity()
	}
}

// generateArenaEnsureCapacity generates the _flap_arena_ensure_capacity function
// This function ensures the meta-arena has enough capacity for the requested depth
// Argument: rdi = required_depth
func (fc *FlapCompiler) generateArenaEnsureCapacity() {
	fc.eb.MarkLabel("_flap_arena_ensure_capacity")

	// Function prologue
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.PushReg("rbx")
	fc.out.PushReg("r12")
	fc.out.PushReg("r13")
	fc.out.PushReg("r14")
	fc.out.PushReg("r15")

	// r12 = required_depth
	fc.out.MovRegToReg("r12", "rdi")

	// Load current capacity
	fc.out.LeaSymbolToReg("rbx", "_flap_arena_meta_cap")
	fc.out.MovMemToReg("r13", "rbx", 0) // r13 = current capacity

	// Check if this is first allocation (capacity == 0)
	fc.out.TestRegReg("r13", "r13")
	firstAllocJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // je to first_alloc

	// Not first time - load len
	fc.out.LeaSymbolToReg("rbx", "_flap_arena_meta_len")
	fc.out.MovMemToReg("r14", "rbx", 0) // r14 = current len

	// Check if we already have enough arenas (required <= len)
	fc.out.CmpRegToReg("r12", "r14")
	noGrowthNeededJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLessOrEqual, 0) // jle to return (required <= len)

	// Need more arenas - check if we have capacity for them
	fc.out.CmpRegToReg("r12", "r13")
	needCapacityGrowthJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpGreater, 0) // jg to capacity growth (required > capacity)

	// Have capacity, just need to initialize more arenas
	// r12 = required, r14 = current len
	// Load meta-arena pointer into r15
	fc.out.LeaSymbolToReg("rbx", "_flap_arena_meta")
	fc.out.MovMemToReg("r15", "rbx", 0) // r15 = meta-arena pointer
	fc.out.MovRegToReg("r13", "r14")    // r13 = current len (start index for init loop)
	fc.generateArenaInitLoop()
	// Jump to return
	lenOnlyGrowthJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)

	// Capacity growth needed - realloc and initialize new slots
	growLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(needCapacityGrowthJump+2, int32(growLabel-(needCapacityGrowthJump+6)))

	// Use helper function to grow meta-arena
	fc.generateMetaArenaGrowth()

	// Use helper function to initialize new arena structures
	fc.generateArenaInitLoop()

	// Update capacity
	fc.out.LeaSymbolToReg("rbx", "_flap_arena_meta_cap")
	fc.out.MovRegToMem("r14", "rbx", 0)

	// Jump to return
	returnJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0) // Will patch this later

	// First allocation path
	firstAllocLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(firstAllocJump+2, int32(firstAllocLabel-(firstAllocJump+6)))

	// Use helper function for first meta-arena allocation
	fc.generateFirstMetaArenaAlloc()

	// Check if we need to grow further (required > 8)
	fc.out.CmpRegToImm("r12", 8)
	firstGrowCheckJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpLessOrEqual, 0) // jle to return (no growth needed)

	// Need to grow: load capacity into r13 for growth path
	fc.out.LeaSymbolToReg("rbx", "_flap_arena_meta_cap")
	fc.out.MovMemToReg("r13", "rbx", 0) // r13 = capacity (8)

	// Jump to growth path
	growthJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)
	fc.patchJumpImmediate(growthJump+1, int32(growLabel-(growthJump+5)))

	// Return (no growth needed)
	returnLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(noGrowthNeededJump+2, int32(returnLabel-(noGrowthNeededJump+6)))
	fc.patchJumpImmediate(returnJump+1, int32(returnLabel-(returnJump+5)))
	fc.patchJumpImmediate(lenOnlyGrowthJump+1, int32(returnLabel-(lenOnlyGrowthJump+5)))
	fc.patchJumpImmediate(firstGrowCheckJump+2, int32(returnLabel-(firstGrowCheckJump+6)))

	fc.out.PopReg("r15")
	fc.out.PopReg("r14")
	fc.out.PopReg("r13")
	fc.out.PopReg("r12")
	fc.out.PopReg("rbx")
	fc.out.PopReg("rbp")
	fc.out.Ret()

	// Error path: malloc/realloc failed
	errorLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(fc.metaArenaGrowthErrorJump+2, int32(errorLabel-(fc.metaArenaGrowthErrorJump+6)))
	fc.patchJumpImmediate(fc.firstMetaArenaMallocErrorJump+2, int32(errorLabel-(fc.firstMetaArenaMallocErrorJump+6)))

	fc.out.MovImmToReg("rdi", "1")
	fc.trackFunctionCall("exit")
	fc.eb.GenerateCallInstruction("exit")
}

func (fc *FlapCompiler) compileStoredFunctionCall(call *CallExpr) {
	// Load function pointer from variable
	offset, _ := fc.variables[call.Function]
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG compileStoredFunctionCall: calling '%s' at offset %d\n", call.Function, offset)
		}
	}
	fc.out.MovMemToXmm("xmm0", "rbp", -offset)

	// Convert function pointer from float64 to integer in rax
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)
	fc.out.MovMemToReg("rax", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)

	// Compile arguments and put them in xmm registers
	xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5"}
	if len(call.Args) > len(xmmRegs) {
		compilerError("too many arguments to stored function (max 6)")
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
	fc.out.SubImmFromReg("rsp", StackSlotSize)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)
	fc.out.MovMemToReg("rax", "rsp", 0)
	fc.out.AddImmToReg("rsp", StackSlotSize)

	// Compile arguments and put them in xmm registers
	xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5"}
	if len(call.Args) > len(xmmRegs) {
		compilerError("too many arguments to direct call (max 6)")
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

	// Load count from map[0] (empty strings have count=0, not null)
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

	// Note: Stack not cleaned up here - caller must handle
}

// compilePrintMapAsString converts a string map to bytes for printing via syscall
// Input: mapPtr (register) = pointer to string map, bufPtr (register) = buffer start
// Output: rsi = pointer to string data, rdx = length (including newline)
func (fc *FlapCompiler) compilePrintMapAsString(mapPtr, bufPtr string) {
	// Load count from map[0] (empty strings have count=0, not null)
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

	// Check if fractional part is zero
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.Cvtsi2sd("xmm2", "rax") // xmm2 = 0.0
	fc.out.Ucomisd("xmm0", "xmm2")
	fracZeroJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpEqual, 0) // Jump if frac == 0
	fracZeroEnd := fc.eb.text.Len()

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

	// Strip trailing zeros by walking backwards
	// rsi points one past the last digit
	stripLoopStart := fc.eb.text.Len()
	// Go back one byte
	fc.out.SubImmFromReg("rsi", 1)
	// Load the byte
	fc.out.MovMemToReg("r10", "rsi", 0)
	// Mask to low byte: AND r10, 0xFF
	fc.out.Write(0x49) // REX.W for r10
	fc.out.Write(0x81) // AND r/m64, imm32
	fc.out.Write(0xE2) // ModR/M for r10
	fc.out.Write(0xFF) // imm = 0xFF
	fc.out.Write(0x00)
	fc.out.Write(0x00)
	fc.out.Write(0x00)
	// Compare with '0' (48)
	fc.out.CmpRegToImm("r10", 48)
	// If equal to '0', continue stripping
	fc.out.JumpConditional(JumpEqual, int32(stripLoopStart-(fc.eb.text.Len()+6)))
	// Not a '0', so advance back to position after this character
	fc.out.AddImmToReg("rsi", 1)

	// Fractional part was zero - remove the decimal point we added
	fracZeroPos := fc.eb.text.Len()
	fc.patchJumpImmediate(fracZeroJump+2, int32(fracZeroPos-fracZeroEnd))
	fc.out.SubImmFromReg("rsi", 1) // Remove the '.' we added

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
	startPosReg := "r14"
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
	fc.out.Emit([]byte{0x48, 0xF7, 0xD8}) // neg rax

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
		compilerError("tail call to 'me' has %d args but function has %d params",
			len(call.Args), len(fc.currentLambda.Params))
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

func (fc *FlapCompiler) compileCachedCall(call *CallExpr) {
	if fc.currentLambda == nil {
		compilerError("cme can only be used inside a lambda function")
	}

	numArgs := len(call.Args)
	if numArgs < 1 || numArgs > 3 {
		compilerError("cme requires 1-3 arguments: cme(arg) or cme(arg, max_size) or cme(arg, max_size, cleanup_fn)")
	}

	fc.cacheEnabledLambdas[fc.currentLambda.Name] = true
	cacheName := fc.currentLambda.Name + "_cache"

	fc.compileExpression(call.Args[0])

	fc.out.SubImmFromReg("rsp", 32)
	fc.out.MovXmmToMem("xmm0", "rsp", 0)

	fc.out.LeaSymbolToReg("rdi", cacheName)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.MovqXmmToReg("rsi", "xmm0")

	fc.trackFunctionCall("flap_cache_lookup")
	fc.out.CallSymbol("flap_cache_lookup")

	fc.out.CmpRegToImm("rax", 0)
	cacheHitJump := fc.eb.text.Len()
	fc.out.JumpConditional(JumpNotEqual, 0)

	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.SubImmFromReg("rsp", 8)

	callPos := fc.eb.text.Len()
	fc.eb.callPatches = append(fc.eb.callPatches, CallPatch{
		position:   callPos + 1,
		targetName: fc.currentLambda.Name,
	})
	fc.out.Emit([]byte{0xE8, 0x00, 0x00, 0x00, 0x00})

	fc.out.AddImmToReg("rsp", 8)
	fc.out.MovXmmToMem("xmm0", "rsp", 8)

	fc.out.LeaSymbolToReg("rdi", cacheName)
	fc.out.MovMemToXmm("xmm0", "rsp", 0)
	fc.out.MovqXmmToReg("rsi", "xmm0")
	fc.out.MovMemToXmm("xmm0", "rsp", 8)
	fc.out.MovqXmmToReg("rdx", "xmm0")

	fc.trackFunctionCall("flap_cache_insert")
	fc.out.CallSymbol("flap_cache_insert")

	fc.out.MovMemToXmm("xmm0", "rsp", 8)

	skipInsertJump := fc.eb.text.Len()
	fc.out.JumpUnconditional(0)

	cacheHitLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(cacheHitJump+2, int32(cacheHitLabel-(cacheHitJump+6)))

	fc.out.MovMemToXmm("xmm0", "rax", 0)

	skipInsertLabel := fc.eb.text.Len()
	fc.patchJumpImmediate(skipInsertJump+1, int32(skipInsertLabel-(skipInsertJump+5)))

	fc.out.AddImmToReg("rsp", 32)
}

func (fc *FlapCompiler) compileCFunctionCall(libName string, funcName string, args []Expression) {
	// Generate C FFI call
	// Strategy for v1.1.0:
	// 1. Marshal arguments according to System V AMD64 ABI
	// 2. Call function using PLT (dynamic linking)
	// 3. Convert result to float64 in xmm0
	//
	// Note: Library is linked dynamically via DT_NEEDED in ELF

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Generating C FFI call: %s.%s with %d args\n", libName, funcName, len(args))
	}

	// Track library dependency for ELF generation
	fc.cLibHandles[libName] = "linked" // Mark as needing dynamic linking

	// Track function usage for PLT generation and call order patching
	fc.trackFunctionCall(funcName)

	// Marshal arguments according to System V AMD64 ABI:
	// Integer/pointer args: rdi, rsi, rdx, rcx, r8, r9, then stack
	// Float args: xmm0-xmm7, then stack

	// System V AMD64 ABI register sequences
	intArgRegs := []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
	floatArgRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7"}

	// Look up function signature from DWARF if available
	// Need to find the alias for this library name (reverse lookup)
	var libAlias string
	for alias, lib := range fc.cImports {
		if lib == libName {
			libAlias = alias
			break
		}
	}

	var funcSig *CFunctionSignature
	if libAlias != "" {
		if constants, ok := fc.cConstants[libAlias]; ok {
			if sig, found := constants.Functions[funcName]; found {
				funcSig = sig
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Found signature for %s: %d params, return=%s\n",
						funcName, len(sig.Params), sig.ReturnType)
				}
			}
		}
	}

	// Allocate stack space to save arguments temporarily
	if len(args) > 0 {
		argStackOffset := len(args) * 8
		fc.out.SubImmFromReg("rsp", int64(argStackOffset))

		// Compile each argument and store on stack
		for i, arg := range args {
			// Determine the parameter type from signature or cast
			var paramType string
			var castType string
			var innerExpr Expression = arg

			if castExpr, ok := arg.(*CastExpr); ok {
				// Explicit cast provided
				castType = castExpr.Type
				innerExpr = castExpr.Expr
			}

			// Determine actual parameter type from signature
			if funcSig != nil && i < len(funcSig.Params) {
				paramType = funcSig.Params[i].Type
			}

			// Decide whether this parameter should be treated as float or int
			isFloatParam := false
			if paramType == "float" || paramType == "double" {
				isFloatParam = true
			}

			// If no signature, fall back to cast type or defaults
			if castType == "" {
				exprType := fc.getExprType(arg)
				if exprType == "string" {
					castType = "cstr"
				} else if isFloatParam {
					castType = "float"
				} else if paramType != "" {
					if isPointerType(paramType) {
						castType = "pointer"
					} else if strings.Contains(paramType, "char") && strings.Contains(paramType, "*") {
						castType = "cstr"
					} else {
						castType = "int"
					}
				} else {
					castType = "int" // Default to int if no info available
				}
			}

			// Set C context for string literals
			isStringLiteral := false
			if _, ok := innerExpr.(*StringExpr); ok {
				isStringLiteral = true
				fc.cContext = true
			}

			// Compile the inner expression (result in xmm0 for Flap values, rax for C strings)
			fc.compileExpression(innerExpr)

			// Reset C context after compilation
			if isStringLiteral {
				fc.cContext = false
			}

			// Store argument on stack based on its type
			if isFloatParam || castType == "float" || castType == "double" {
				// Keep as float64 in xmm0, store directly
				fc.out.MovXmmToMem("xmm0", "rsp", i*8)
			} else {
				// Convert to integer or pointer
				switch castType {
				case "cstr", "cstring":
					if isStringLiteral {
						// String literal was compiled as C string - rax already contains the pointer
						// No conversion needed, just store it
					} else {
						// Runtime string (Flap map format) - need to convert to C string
						fc.out.SubImmFromReg("rsp", StackSlotSize)
						fc.out.MovXmmToMem("xmm0", "rsp", 0)
						fc.out.MovMemToReg("rax", "rsp", 0)
						fc.out.AddImmToReg("rsp", StackSlotSize)

						// Call flap_string_to_cstr(map_ptr) -> char*
						fc.out.SubImmFromReg("rsp", StackSlotSize)
						fc.out.MovRegToMem("rax", "rsp", 0)
						fc.out.MovMemToReg("rdi", "rsp", 0)
						fc.out.AddImmToReg("rsp", StackSlotSize)
						fc.trackFunctionCall("flap_string_to_cstr")
						fc.out.CallSymbol("flap_string_to_cstr")
						// Result in rax (C string pointer)
					}

				case "ptr", "pointer":
					// Pointer type - convert float64 to integer pointer
					fc.out.Cvttsd2si("rax", "xmm0")

				case "int", "i32", "int32":
					// Signed 32-bit integer
					fc.out.Cvttsd2si("rax", "xmm0")

				case "uint32", "u32":
					// Unsigned 32-bit integer
					fc.out.Cvttsd2si("rax", "xmm0")

				default:
					// Default: convert float64 to integer
					fc.out.Cvttsd2si("rax", "xmm0")
				}

				// Store on stack at offset i*8
				fc.out.MovRegToMem("rax", "rsp", i*8)
			}
		}

		// Load arguments from stack into ABI registers
		// Track int and float register usage separately
		intRegIdx := 0
		floatRegIdx := 0
		stackArgCount := 0

		// Build a list of stack arguments that overflow registers
		type stackArg struct {
			offset int
			value  int
		}
		var stackArgs []stackArg

		// First pass: determine which arguments go in registers vs stack
		for i := 0; i < len(args); i++ {
			var paramType string
			if funcSig != nil && i < len(funcSig.Params) {
				paramType = funcSig.Params[i].Type
			}

			isFloatParam := (paramType == "float" || paramType == "double")

			if isFloatParam {
				if floatRegIdx < len(floatArgRegs) {
					// Load into float register
					fc.out.MovMemToXmm(floatArgRegs[floatRegIdx], "rsp", i*8)
					floatRegIdx++
				} else {
					// Goes on stack
					stackArgs = append(stackArgs, stackArg{offset: i * 8, value: stackArgCount})
					stackArgCount++
				}
			} else {
				if intRegIdx < len(intArgRegs) {
					// Load into int register
					fc.out.MovMemToReg(intArgRegs[intRegIdx], "rsp", i*8)
					intRegIdx++
				} else {
					// Goes on stack
					stackArgs = append(stackArgs, stackArg{offset: i * 8, value: stackArgCount})
					stackArgCount++
				}
			}
		}

		// Clean up temp stack space, but preserve stack arguments
		if stackArgCount > 0 {
			// Move stack args to the bottom of the stack
			for i, arg := range stackArgs {
				fc.out.MovMemToReg("r11", "rsp", arg.offset)
				fc.out.MovRegToMem("r11", "rsp", i*8)
			}
			// Adjust RSP to remove register arg space, keeping stack args
			fc.out.AddImmToReg("rsp", int64(argStackOffset-stackArgCount*8))
		} else {
			// No stack args - clean up all temp space
			fc.out.AddImmToReg("rsp", int64(argStackOffset))
		}

		// Generate PLT call
		fc.eb.GenerateCallInstruction(funcName)

		// Clean up stack arguments after call
		if stackArgCount > 0 {
			fc.out.AddImmToReg("rsp", int64(stackArgCount*8))
		}

		// Handle return value based on signature
		var returnType string
		if funcSig != nil {
			returnType = funcSig.ReturnType
		}

		if returnType == "float" || returnType == "double" {
			// Result is already in xmm0 as double - no conversion needed
		} else {
			// Convert integer result in rax to float64 for Flap
			fc.out.Cvtsi2sd("xmm0", "rax")
		}
	} else {
		// No arguments - just call the function
		// No stack adjustment needed - RSP is already at (16n - 8) from main() prologue
		fc.eb.GenerateCallInstruction(funcName)

		// Handle return value based on signature
		var returnType string
		if funcSig != nil {
			returnType = funcSig.ReturnType
		}

		if returnType == "float" || returnType == "double" {
			// Result is already in xmm0 as double - no conversion needed
		} else {
			// Convert integer result in rax to float64 for Flap
			fc.out.Cvtsi2sd("xmm0", "rax")
		}
	}
}

func (fc *FlapCompiler) compileCall(call *CallExpr) {
	// Check for "me" self-reference (tail recursion candidate)
	if call.Function == "me" && fc.currentLambda != nil {
		fc.compileTailCall(call)
		return
	}

	// Check for "cme" cached/memoized self-reference
	if call.Function == "cme" && fc.currentLambda != nil {
		fc.compileCachedCall(call)
		return
	}

	// Check if this is a C library function call (namespace.function)
	if strings.Contains(call.Function, ".") {
		parts := strings.Split(call.Function, ".")
		if len(parts) == 2 {
			namespace := parts[0]
			funcName := parts[1]

			// Check if namespace is a registered C import
			if libName, ok := fc.cImports[namespace]; ok {
				fc.compileCFunctionCall(libName, funcName, call.Args)
				return
			}
		}
	}

	// Check if this is a stored function (variable containing function pointer)
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG compileCall: looking up function '%s'\n", call.Function)
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG compileCall: variables map has: %v\n", fc.variables)
		}
	}
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
			fc.out.SubImmFromReg("rsp", StackSlotSize)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)
			fc.out.MovMemToReg("rax", "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)

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
			compilerError("printf() requires at least a format string")
		}

		// First argument must be a string (format string)
		formatArg := call.Args[0]
		strExpr, ok := formatArg.(*StringExpr)
		if !ok {
			compilerError("printf() first argument must be a string literal (got %T)", formatArg)
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
			compilerError("printf() supports max 8 arguments (got %d)", numArgs)
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

		// Load float arguments from stack into xmm registers
		// We pushed args in forward order, so we need to pop in reverse to get correct order
		// Stack layout after pushing arg1, arg2, arg3: [arg3][arg2][arg1] <- rsp
		// But we want: arg1→xmm0, arg2→xmm1, arg3→xmm2
		// So we need to load from stack in reverse: pop arg1 (deepest), then arg2, then arg3

		// Count how many float args we have
		numFloatArgs := 0
		for i := 0; i < numArgs; i++ {
			if !boolPositions[i] && !stringPositions[i] {
				numFloatArgs++
			}
		}

		// Now load them in the correct order
		// Start from the deepest item in stack and work backwards
		for i := 0; i < numArgs; i++ {
			if !boolPositions[i] && !stringPositions[i] {
				// Calculate offset from rsp: (numFloatArgs - xmmArgCount - 1) * 16
				offset := (numFloatArgs - xmmArgCount - 1) * 16
				fc.out.MovMemToXmm(xmmRegs[xmmArgCount], "rsp", offset)
				xmmArgCount++
			}
		}
		// Clean up stack
		if numFloatArgs > 0 {
			fc.out.AddImmToReg("rsp", int64(numFloatArgs*16))
		}

		// Load format string to rdi
		fc.out.LeaSymbolToReg("rdi", labelName)

		// Set rax = number of vector registers used
		fc.out.MovImmToReg("rax", fmt.Sprintf("%d", xmmArgCount))

		fc.trackFunctionCall("printf")
		fc.eb.GenerateCallInstruction("printf")

	case "exit":
		fc.hasExplicitExit = true // Mark that program has explicit exit
		if len(call.Args) > 0 {
			fc.compileExpression(call.Args[0])
			// Convert float64 in xmm0 to int64 in rdi
			fc.out.Cvttsd2si("rdi", "xmm0") // truncate float to int
		} else {
			fc.out.XorRegWithReg("rdi", "rdi")
		}
		// Restore stack pointer to frame pointer (rsp % 16 == 8 for proper call alignment)
		// Don't pop rbp since exit() never returns
		fc.out.MovRegToReg("rsp", "rbp")
		fc.trackFunctionCall("exit")
		fc.eb.GenerateCallInstruction("exit")

	case "arena_create":
		// arena_create(capacity) -> arena_ptr
		// Create a new arena with the given capacity
		if len(call.Args) != 1 {
			compilerError("arena_create() requires exactly 1 argument (capacity)")
		}
		fc.compileExpression(call.Args[0])
		// Convert float64 capacity to int64
		fc.out.Cvttsd2si("rdi", "xmm0")
		fc.out.CallSymbol("flap_arena_create")
		// Result in rax, convert to float64
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "arena_alloc":
		// arena_alloc(arena_ptr, size) -> allocation_ptr
		// Allocate memory from the arena
		if len(call.Args) != 2 {
			compilerError("arena_alloc() requires exactly 2 arguments (arena_ptr, size)")
		}
		// First arg: arena_ptr
		fc.compileExpression(call.Args[0])
		fc.out.Cvttsd2si("rdi", "xmm0")
		// Second arg: size
		fc.compileExpression(call.Args[1])
		fc.out.Cvttsd2si("rsi", "xmm0")
		fc.out.CallSymbol("flap_arena_alloc")
		// Result in rax, convert to float64
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "arena_destroy":
		// arena_destroy(arena_ptr)
		// Destroy the arena and free all memory
		if len(call.Args) != 1 {
			compilerError("arena_destroy() requires exactly 1 argument (arena_ptr)")
		}
		fc.compileExpression(call.Args[0])
		fc.out.Cvttsd2si("rdi", "xmm0")
		fc.out.CallSymbol("flap_arena_destroy")
		// No return value, set xmm0 to 0
		fc.out.XorpdXmm("xmm0", "xmm0")

	case "arena_reset":
		// arena_reset(arena_ptr)
		// Reset the arena offset to 0, freeing all allocations
		if len(call.Args) != 1 {
			compilerError("arena_reset() requires exactly 1 argument (arena_ptr)")
		}
		fc.compileExpression(call.Args[0])
		fc.out.Cvttsd2si("rdi", "xmm0")
		fc.out.CallSymbol("flap_arena_reset")
		// No return value, set xmm0 to 0
		fc.out.XorpdXmm("xmm0", "xmm0")

	case "syscall":
		// Raw Linux syscall: syscall(number, arg1, arg2, arg3, arg4, arg5, arg6)
		// x86-64 syscall convention: rax=number, rdi, rsi, rdx, r10, r8, r9
		if len(call.Args) < 1 || len(call.Args) > 7 {
			compilerError("syscall() requires 1-7 arguments (syscall number + up to 6 args)")
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
			compilerError("getpid() takes no arguments")
		}
		fc.trackFunctionCall("getpid")
		fc.eb.GenerateCallInstruction("getpid")
		// Convert result from rax to xmm0
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "sqrt":
		if len(call.Args) != 1 {
			compilerError("sqrt() requires exactly 1 argument")
		}
		// Compile argument (result in xmm0)
		fc.compileExpression(call.Args[0])
		// Use x86-64 SQRTSD instruction (hardware sqrt)
		// sqrtsd xmm0, xmm0 - sqrt of xmm0, result in xmm0
		fc.out.Sqrtsd("xmm0", "xmm0")

	case "sin":
		if len(call.Args) != 1 {
			compilerError("sin() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FSIN instruction
		// xmm0 -> memory -> ST(0) -> FSIN -> memory -> xmm0
		fc.out.SubImmFromReg("rsp", StackSlotSize) // Allocate 8 bytes on stack
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0)
		fc.out.Fsin()
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize) // Restore stack

	case "cos":
		if len(call.Args) != 1 {
			compilerError("cos() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FCOS instruction
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0)
		fc.out.Fcos()
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "tan":
		if len(call.Args) != 1 {
			compilerError("tan() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FPTAN instruction
		// FPTAN computes tan and pushes 1.0, so we need to pop the 1.0
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0)
		fc.out.Fptan()
		fc.out.Fpop() // Pop the 1.0 that FPTAN pushes
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "atan":
		if len(call.Args) != 1 {
			compilerError("atan() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// Use x87 FPU FPATAN: atan(x) = atan2(x, 1.0)
		// FPATAN expects ST(1)=y, ST(0)=x, computes atan2(y,x)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.Fld1()           // ST(0) = 1.0, ST(1) = x
		fc.out.Fpatan()         // ST(0) = atan2(x, 1.0) = atan(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "asin":
		if len(call.Args) != 1 {
			compilerError("asin() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// asin(x) = atan2(x, sqrt(1 - x²))
		// FPATAN needs ST(1)=x, ST(0)=sqrt(1-x²)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
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
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.FstpMem("rsp", 0)            // Store sqrt to [rsp+0]
		fc.out.FldMem("rsp", StackSlotSize) // Load x: ST(0) = x
		fc.out.FldMem("rsp", 0)             // Load sqrt: ST(0) = sqrt, ST(1) = x
		fc.out.Fpatan()                     // ST(0) = atan2(x, sqrt(1-x²)) = asin(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", 16) // Restore both allocations

	case "acos":
		if len(call.Args) != 1 {
			compilerError("acos() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// acos(x) = atan2(sqrt(1-x²), x)
		// FPATAN needs ST(1)=sqrt(1-x²), ST(0)=x
		fc.out.SubImmFromReg("rsp", StackSlotSize)
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
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "abs":
		if len(call.Args) != 1 {
			compilerError("abs() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// abs(x) using FABS
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.FldMem("rsp", 0) // ST(0) = x
		fc.out.Fabs()           // ST(0) = |x|
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "floor":
		if len(call.Args) != 1 {
			compilerError("floor() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// floor(x): round toward -∞
		// FPU control word: set rounding mode to 01 (round down)
		fc.out.SubImmFromReg("rsp", 16) // Need space for control word + value
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)

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
		fc.out.FldMem("rsp", StackSlotSize)
		fc.out.Frndint()
		fc.out.FstpMem("rsp", StackSlotSize)

		// Restore original control word
		fc.out.FldcwMem("rsp", 0)

		fc.out.MovMemToXmm("xmm0", "rsp", StackSlotSize)
		fc.out.AddImmToReg("rsp", 16)

	case "ceil":
		if len(call.Args) != 1 {
			compilerError("ceil() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// ceil(x): round toward +∞
		// FPU control word: set rounding mode to 10 (round up)
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)

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
		fc.out.FldMem("rsp", StackSlotSize)
		fc.out.Frndint()
		fc.out.FstpMem("rsp", StackSlotSize)
		fc.out.FldcwMem("rsp", 0) // Restore

		fc.out.MovMemToXmm("xmm0", "rsp", StackSlotSize)
		fc.out.AddImmToReg("rsp", 16)

	case "round":
		if len(call.Args) != 1 {
			compilerError("round() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// round(x): round to nearest (even)
		// FPU control word: set rounding mode to 00 (round to nearest)
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)

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
		fc.out.FldMem("rsp", StackSlotSize)
		fc.out.Frndint()
		fc.out.FstpMem("rsp", StackSlotSize)
		fc.out.FldcwMem("rsp", 0) // Restore

		fc.out.MovMemToXmm("xmm0", "rsp", StackSlotSize)
		fc.out.AddImmToReg("rsp", 16)

	case "log":
		if len(call.Args) != 1 {
			compilerError("log() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// log(x) = ln(x) = log2(x) / log2(e) = log2(x) * ln(2) / (ln(2) / ln(e))
		// FYL2X computes ST(1) * log2(ST(0))
		// So: log(x) = ln(2) * log2(x) = FYL2X with ST(1)=ln(2), ST(0)=x
		// But we want ln(x), not log2(x)
		// ln(x) = log2(x) * ln(2)
		// Actually: FYL2X gives us: ST(1) * log2(ST(0))
		// So if ST(1) = ln(2) and ST(0) = x, we get: ln(2) * log2(x) = ln(x) ✓
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.Fldln2()         // ST(0) = ln(2)
		fc.out.FldMem("rsp", 0) // ST(0) = x, ST(1) = ln(2)
		fc.out.Fyl2x()          // ST(0) = ln(2) * log2(x) = ln(x)
		fc.out.FstpMem("rsp", 0)
		fc.out.MovMemToXmm("xmm0", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "exp":
		if len(call.Args) != 1 {
			compilerError("exp() requires exactly 1 argument")
		}
		fc.compileExpression(call.Args[0])
		// exp(x) = e^x = 2^(x * log2(e))
		// Steps:
		// 1. Multiply x by log2(e): x' = x * log2(e)
		// 2. Split x' = n + f where n is integer, -1 <= f <= 1
		// 3. Compute 2^f using F2XM1: 2^f = 1 + F2XM1(f)
		// 4. Scale by 2^n using FSCALE
		fc.out.SubImmFromReg("rsp", StackSlotSize)
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
		fc.out.AddImmToReg("rsp", StackSlotSize)

	case "pow":
		if len(call.Args) != 2 {
			compilerError("pow() requires exactly 2 arguments")
		}
		fc.compileExpression(call.Args[0]) // x in xmm0
		fc.out.SubImmFromReg("rsp", 16)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.compileExpression(call.Args[1]) // y in xmm0
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)

		// pow(x, y) = x^y = 2^(y * log2(x))
		// Steps:
		// 1. Compute log2(x) using FYL2X
		// 2. Multiply by y
		// 3. Split into integer and fractional parts
		// 4. Use F2XM1 and FSCALE like in exp

		fc.out.Fld1()                       // ST(0) = 1.0
		fc.out.FldMem("rsp", 0)             // ST(0) = x, ST(1) = 1.0
		fc.out.Fyl2x()                      // ST(0) = 1 * log2(x) = log2(x)
		fc.out.FldMem("rsp", StackSlotSize) // ST(0) = y, ST(1) = log2(x)
		fc.out.Fmulp()                      // ST(0) = y * log2(x)

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

	case "str":
		// Convert number to string
		// str(x) converts a number to a Flap string (map[uint64]float64)
		if len(call.Args) != 1 {
			compilerError("str() requires exactly 1 argument")
		}

		// Compile argument (result in xmm0)
		fc.compileExpression(call.Args[0])

		// Allocate 32 bytes for ASCII conversion buffer
		fc.out.SubImmFromReg("rsp", 32)
		// Save buffer address before compileFloatToString changes rsp
		fc.out.MovRegToReg("r15", "rsp")

		// Convert float64 in xmm0 to ASCII string at r15
		// Result: rsi = string start, rdx = length
		fc.compileFloatToString("xmm0", "r15")

		// Check if last char is newline and adjust length
		// rax = rdx - 1
		fc.out.MovRegToReg("rax", "rdx")
		fc.out.SubImmFromReg("rax", 1)
		// r10 = rsi + rax (pointer to last char)
		fc.out.MovRegToReg("r10", "rsi")
		fc.out.AddRegToReg("r10", "rax")
		// Load byte at r10
		fc.out.Emit([]byte{0x45, 0x0f, 0xb6, 0x12}) // movzx r10d, byte [r10]
		// Compare r10 with 10 (newline)
		fc.out.Emit([]byte{0x49, 0x83, 0xfa, 0x0a}) // cmp r10, 10
		skipNewlineLabel := fc.eb.text.Len()
		fc.out.JumpConditional(JumpNotEqual, 0)
		skipNewlineEnd := fc.eb.text.Len()

		// Has newline - decrement length
		fc.out.SubImmFromReg("rdx", 1)

		// Skip target
		skipNewline := fc.eb.text.Len()
		fc.patchJumpImmediate(skipNewlineLabel+2, int32(skipNewline-skipNewlineEnd))

		// Calculate map size: 8 + length * 16
		// rdi = rdx * 16
		fc.out.MovRegToReg("rdi", "rdx")
		fc.out.Emit([]byte{0x48, 0xc1, 0xe7, 0x04}) // shl rdi, 4
		fc.out.AddImmToReg("rdi", 8)

		// Save rsi and rdx before malloc
		fc.out.PushReg("rsi")
		fc.out.PushReg("rdx")

		// Call malloc
		fc.trackFunctionCall("malloc")
		fc.eb.GenerateCallInstruction("malloc")

		// Restore
		fc.out.PopReg("rdx")
		fc.out.PopReg("rsi")

		// Write count
		fc.out.Cvtsi2sd("xmm1", "rdx")
		fc.out.MovXmmToMem("xmm1", "rax", 0)

		// Save map pointer
		fc.out.MovRegToReg("r11", "rax")

		// Loop to build map
		fc.out.XorRegWithReg("rcx", "rcx")
		fc.out.MovRegToReg("rdi", "rax")
		fc.out.AddImmToReg("rdi", 8)

		loopStart := fc.eb.text.Len()

		// cmp rcx, rdx
		fc.out.Emit([]byte{0x48, 0x39, 0xd1}) // cmp rcx, rdx
		loopEndJump := fc.eb.text.Len()
		fc.out.JumpConditional(JumpGreaterOrEqual, 0)
		loopEndJumpEnd := fc.eb.text.Len()

		// Write key
		fc.out.Cvtsi2sd("xmm1", "rcx")
		fc.out.MovXmmToMem("xmm1", "rdi", 0)
		fc.out.AddImmToReg("rdi", 8)

		// Load char and write value
		fc.out.Emit([]byte{0x4c, 0x0f, 0xb6, 0x16}) // movzx r10, byte [rsi]
		fc.out.Cvtsi2sd("xmm1", "r10")
		fc.out.MovXmmToMem("xmm1", "rdi", 0)
		fc.out.AddImmToReg("rdi", 8)

		// Increment
		fc.out.AddImmToReg("rcx", 1)
		fc.out.AddImmToReg("rsi", 1)

		// Jump back
		loopEnd := fc.eb.text.Len()
		offset := loopStart - (loopEnd + 2)
		fc.out.Emit([]byte{0xeb, byte(offset)})

		// Loop done
		loopDone := fc.eb.text.Len()
		fc.patchJumpImmediate(loopEndJump+2, int32(loopDone-loopEndJumpEnd))

		// Return map pointer as float64 (move bits directly, don't convert)
		// Use movq xmm0, r11 to transfer pointer bits without conversion
		// movq xmm0, r11 = 66 49 0f 6e c3
		fc.out.Emit([]byte{0x66, 0x49, 0x0f, 0x6e, 0xc3})

		// Clean up
		fc.out.AddImmToReg("rsp", 32)

	case "num":
		// Parse string to number
		// num(string) converts a Flap string to a number
		if len(call.Args) != 1 {
			compilerError("num() requires exactly 1 argument")
		}

		// Compile argument (Flap string pointer in xmm0)
		fc.compileExpression(call.Args[0])

		// Convert Flap string to C string
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("flap_string_to_cstr")

		// Call strtod(str, NULL) to parse the string
		// rdi = C string (already in rax from flap_string_to_cstr)
		fc.out.MovRegToReg("rdi", "rax")
		fc.out.XorRegWithReg("rsi", "rsi") // endptr = NULL
		fc.trackFunctionCall("strtod")
		fc.eb.GenerateCallInstruction("strtod")
		// Result in xmm0

	case "upper":
		// Convert string to uppercase
		// upper(string) returns a new uppercase string
		if len(call.Args) != 1 {
			compilerError("upper() requires exactly 1 argument")
		}

		// Compile argument (Flap string pointer in xmm0)
		fc.compileExpression(call.Args[0])

		// Call runtime helper upper_string
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("upper_string")
		// Result in xmm0

	case "lower":
		// Convert string to lowercase
		// lower(string) returns a new lowercase string
		if len(call.Args) != 1 {
			compilerError("lower() requires exactly 1 argument")
		}

		// Compile argument (Flap string pointer in xmm0)
		fc.compileExpression(call.Args[0])

		// Call runtime helper lower_string
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("lower_string")
		// Result in xmm0

	case "trim":
		// Remove leading/trailing whitespace
		// trim(string) returns a new trimmed string
		if len(call.Args) != 1 {
			compilerError("trim() requires exactly 1 argument")
		}

		// Compile argument (Flap string pointer in xmm0)
		fc.compileExpression(call.Args[0])

		// Call runtime helper trim_string
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("trim_string")
		// Result in xmm0

	case "write_i8", "write_i16", "write_i32", "write_i64",
		"write_u8", "write_u16", "write_u32", "write_u64", "write_f64":
		// FFI memory write: write_TYPE(ptr, index, value)
		if len(call.Args) != 3 {
			compilerError("%s() requires exactly 3 arguments (ptr, index, value)", call.Function)
		}

		// Determine type size
		var typeSize int
		switch call.Function {
		case "write_i8", "write_u8":
			typeSize = 1
		case "write_i16", "write_u16":
			typeSize = 2
		case "write_i32", "write_u32":
			typeSize = 4
		case "write_i64", "write_u64", "write_f64":
			typeSize = 8
		}

		// Compile pointer (arg 0) - result in xmm0
		fc.compileExpression(call.Args[0])
		// Convert pointer from float64 to integer in r10
		fc.out.Cvttsd2si("r10", "xmm0")
		// Save pointer on stack (push r10)
		fc.out.Emit([]byte{0x41, 0x52}) // push r10

		// Compile index (arg 1) - result in xmm0
		fc.compileExpression(call.Args[1])
		// Convert index to integer in r11
		fc.out.Cvttsd2si("r11", "xmm0")
		// Save index on stack (push r11)
		fc.out.Emit([]byte{0x41, 0x53}) // push r11

		// Compile value (arg 2) - result in xmm0
		fc.compileExpression(call.Args[2])
		// Save value in xmm1
		fc.out.MovXmmToXmm("xmm1", "xmm0")

		// Restore index and pointer (pop r11, pop r10)
		fc.out.Emit([]byte{0x41, 0x5b}) // pop r11
		fc.out.Emit([]byte{0x41, 0x5a}) // pop r10

		// Calculate address: r10 + (r11 * typeSize)
		if typeSize > 1 {
			// Multiply index by type size: rax = r11 * typeSize
			fc.out.MovImmToReg("rax", fmt.Sprintf("%d", typeSize))
			fc.out.Emit([]byte{0x49, 0x0f, 0xaf, 0xc3}) // imul rax, r11 (rax = rax * r11)
			// Add to base pointer: r10 = r10 + rax
			fc.out.Emit([]byte{0x49, 0x01, 0xc2}) // add r10, rax
		} else {
			// If typeSize == 1, r10 = r10 + r11 directly
			fc.out.Emit([]byte{0x4d, 0x01, 0xda}) // add r10, r11
		}

		// Restore value from xmm1 to xmm0
		fc.out.MovXmmToXmm("xmm0", "xmm1")

		// Write value to memory
		if call.Function == "write_f64" {
			// Write float64 directly
			fc.out.MovXmmToMem("xmm0", "r10", 0)
		} else {
			// Convert to integer and write
			fc.out.Cvttsd2si("rax", "xmm0")
			switch typeSize {
			case 1:
				fc.out.MovByteRegToMem("rax", "r10", 0)
			case 2:
				// mov word [r10], ax
				fc.out.Write(0x66) // 16-bit operand prefix
				fc.out.Write(0x41) // REX prefix for r10
				fc.out.Write(0x89) // mov r/m16, r16
				fc.out.Write(0x02) // ModR/M: [r10]
			case 4:
				// mov dword [r10], eax
				fc.out.Write(0x41) // REX prefix for r10
				fc.out.Write(0x89) // mov r/m32, r32
				fc.out.Write(0x02) // ModR/M: [r10]
			case 8:
				// mov qword [r10], rax
				fc.out.MovRegToMem("rax", "r10", 0)
			}
		}

		// Return 0 (these functions don't return values)
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "read_i8", "read_i16", "read_i32", "read_i64",
		"read_u8", "read_u16", "read_u32", "read_u64", "read_f64":
		// FFI memory read: read_TYPE(ptr, index) -> value
		if len(call.Args) != 2 {
			compilerError("%s() requires exactly 2 arguments (ptr, index)", call.Function)
		}

		// Determine type size and signed/unsigned
		var typeSize int
		isSigned := strings.HasPrefix(call.Function, "read_i")
		isFloat := call.Function == "read_f64"

		switch call.Function {
		case "read_i8", "read_u8":
			typeSize = 1
		case "read_i16", "read_u16":
			typeSize = 2
		case "read_i32", "read_u32":
			typeSize = 4
		case "read_i64", "read_u64", "read_f64":
			typeSize = 8
		}

		// Compile pointer (arg 0) - result in xmm0
		fc.compileExpression(call.Args[0])
		// Convert pointer from float64 to integer in r10
		fc.out.Cvttsd2si("r10", "xmm0")
		// Save pointer on stack (push r10)
		fc.out.Emit([]byte{0x41, 0x52}) // push r10

		// Compile index (arg 1) - result in xmm0
		fc.compileExpression(call.Args[1])
		// Convert index to integer in r11
		fc.out.Cvttsd2si("r11", "xmm0")

		// Restore pointer (pop r10)
		fc.out.Emit([]byte{0x41, 0x5a}) // pop r10

		// Calculate address: r10 + (r11 * typeSize)
		if typeSize > 1 {
			// Multiply index by type size: rax = r11 * typeSize
			fc.out.MovImmToReg("rax", fmt.Sprintf("%d", typeSize))
			fc.out.Emit([]byte{0x49, 0x0f, 0xaf, 0xc3}) // imul rax, r11 (rax = rax * r11)
			// Add to base pointer: r10 = r10 + rax
			fc.out.Emit([]byte{0x49, 0x01, 0xc2}) // add r10, rax
		} else {
			// If typeSize == 1, r10 = r10 + r11 directly
			fc.out.Emit([]byte{0x4d, 0x01, 0xda}) // add r10, r11
		}

		// Read value from memory
		if isFloat {
			// Read float64 directly
			fc.out.MovMemToXmm("xmm0", "r10", 0)
		} else {
			// Read integer and convert
			switch typeSize {
			case 1:
				if isSigned {
					// movsx rax, byte [r10]
					fc.out.Write(0x49) // REX.W + REX.B
					fc.out.Write(0x0f) // Two-byte opcode
					fc.out.Write(0xbe) // movsx
					fc.out.Write(0x02) // ModR/M: [r10]
				} else {
					// movzx rax, byte [r10]
					fc.out.Write(0x49) // REX.W + REX.B
					fc.out.Write(0x0f) // Two-byte opcode
					fc.out.Write(0xb6) // movzx
					fc.out.Write(0x02) // ModR/M: [r10]
				}
			case 2:
				if isSigned {
					// movsx rax, word [r10]
					fc.out.Write(0x49) // REX.W + REX.B
					fc.out.Write(0x0f) // Two-byte opcode
					fc.out.Write(0xbf) // movsx
					fc.out.Write(0x02) // ModR/M: [r10]
				} else {
					// movzx rax, word [r10]
					fc.out.Write(0x49) // REX.W + REX.B
					fc.out.Write(0x0f) // Two-byte opcode
					fc.out.Write(0xb7) // movzx
					fc.out.Write(0x02) // ModR/M: [r10]
				}
			case 4:
				if isSigned {
					// movsxd rax, dword [r10]
					fc.out.Write(0x49) // REX.W + REX.B
					fc.out.Write(0x63) // movsxd
					fc.out.Write(0x02) // ModR/M: [r10]
				} else {
					// mov eax, dword [r10] (zero extends to rax)
					fc.out.Write(0x41) // REX.B for r10
					fc.out.Write(0x8b) // mov
					fc.out.Write(0x02) // ModR/M: [r10]
				}
			case 8:
				// mov rax, qword [r10]
				fc.out.MovMemToReg("rax", "r10", 0)
			}
			// Convert integer to float64
			if isSigned {
				fc.out.Cvtsi2sd("xmm0", "rax")
			} else {
				// For unsigned, need special handling for large values
				// For simplicity, just use signed conversion (works for values < 2^63)
				fc.out.Cvtsi2sd("xmm0", "rax")
			}
		}

	case "call":
		// FFI: call(function_name, args...)
		// First argument must be a string literal (function name)
		if len(call.Args) < 1 {
			compilerError("call() requires at least a function name")
		}

		fnNameExpr, ok := call.Args[0].(*StringExpr)
		if !ok {
			compilerError("call() first argument must be a string literal (function name)")
		}
		fnName := fnNameExpr.Value

		// x86-64 calling convention:
		// Integer/pointer args: rdi, rsi, rdx, rcx, r8, r9
		// Float args: xmm0-xmm7
		intRegs := []string{"rdi", "rsi", "rdx", "rcx", "r8", "r9"}
		xmmRegs := []string{"xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6", "xmm7"}

		intArgCount := 0
		xmmArgCount := 0
		numArgs := len(call.Args) - 1 // Exclude function name

		if numArgs > 8 {
			compilerError("call() supports max 8 arguments (got %d)", numArgs)
		}

		// Determine argument types by checking for cast expressions
		argTypes := make([]string, numArgs)
		for i := 0; i < numArgs; i++ {
			arg := call.Args[i+1]
			if castExpr, ok := arg.(*CastExpr); ok {
				argTypes[i] = castExpr.Type
			} else {
				// No cast - assume float64
				argTypes[i] = "f64"
			}
		}

		// Evaluate all arguments and save to stack
		for i := 0; i < numArgs; i++ {
			fc.compileExpression(call.Args[i+1])
			fc.out.SubImmFromReg("rsp", StackSlotSize)
			fc.out.MovXmmToMem("xmm0", "rsp", 0)
		}

		// Load arguments into registers (in reverse order from stack)
		for i := numArgs - 1; i >= 0; i-- {
			argType := argTypes[i]

			// Determine if this is an integer/pointer argument or float argument
			isIntArg := false
			switch argType {
			case "i8", "i16", "i32", "i64", "u8", "u16", "u32", "u64", "ptr", "cstr":
				isIntArg = true
			case "f32", "f64":
				isIntArg = false
			default:
				// Unknown type - assume float
				isIntArg = false
			}

			fc.out.MovMemToXmm("xmm0", "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)

			if isIntArg {
				// Integer/pointer argument
				if intArgCount < len(intRegs) {
					// For cstr, the value is already a pointer in xmm0
					// For integers, convert from float64 to integer
					if argType == "cstr" {
						// cstr is already a pointer - just transfer bits
						fc.out.SubImmFromReg("rsp", StackSlotSize)
						fc.out.MovXmmToMem("xmm0", "rsp", 0)
						fc.out.MovMemToReg(intRegs[intArgCount], "rsp", 0)
						fc.out.AddImmToReg("rsp", StackSlotSize)
					} else {
						// Convert float64 to integer
						fc.out.Cvttsd2si(intRegs[intArgCount], "xmm0")
					}
					intArgCount++
				} else {
					compilerError("call() supports max 6 integer/pointer arguments")
				}
			} else {
				// Float argument
				if xmmArgCount < len(xmmRegs) {
					if xmmArgCount != 0 {
						// Move to appropriate xmm register
						fc.out.SubImmFromReg("rsp", StackSlotSize)
						fc.out.MovXmmToMem("xmm0", "rsp", 0)
						fc.out.MovMemToXmm(xmmRegs[xmmArgCount], "rsp", 0)
						fc.out.AddImmToReg("rsp", StackSlotSize)
					}
					// else: already in xmm0
					xmmArgCount++
				} else {
					compilerError("call() supports max 8 float arguments")
				}
			}
		}

		// Set rax = number of vector registers used (required by x86-64 ABI for varargs)
		fc.out.MovImmToReg("rax", fmt.Sprintf("%d", xmmArgCount))

		// Call the C function
		fc.trackFunctionCall(fnName)
		fc.eb.GenerateCallInstruction(fnName)

		// Result is in rax (for integer/pointer returns) or xmm0 (for float returns)
		// Check if this is a known floating-point function
		floatFunctions := map[string]bool{
			"sqrt": true, "sin": true, "cos": true, "tan": true,
			"asin": true, "acos": true, "atan": true, "atan2": true,
			"log": true, "log10": true, "exp": true, "pow": true,
			"fabs": true, "fmod": true, "ceil": true, "floor": true,
		}

		if floatFunctions[fnName] {
			// Float return - result already in xmm0
			// Nothing to do
		} else {
			// Integer/pointer return - result in rax
			// For most functions, we want to preserve the value semantics (convert int to float)
			// For pointer returns (getenv, malloc, etc), "as number" will be used to get the pointer bits
			fc.out.Cvtsi2sd("xmm0", "rax")
		}

	case "alloc":
		// alloc(size) - Context-aware memory allocation
		// Inside arena { }: allocates from arena with auto-growing
		// Outside arena: error (use malloc via C FFI if needed)
		if len(call.Args) != 1 {
			compilerError("alloc() requires 1 argument (size)")
		}

		// Check if we're in an arena context
		if fc.currentArena == 0 {
			compilerError("alloc() can only be used inside an arena { ... } block. Use malloc() via C FFI if you need manual memory management.")
		}

		// Load arena pointer from meta-arena: _flap_arena_meta[currentArena-1]
		offset := (fc.currentArena - 1) * 8
		fc.out.LeaSymbolToReg("rdi", "_flap_arena_meta")
		fc.out.MovMemToReg("rdi", "rdi", 0)     // Load the meta-arena pointer
		fc.out.MovMemToReg("rdi", "rdi", offset) // Load the arena pointer from slot

		// Compile size argument
		fc.compileExpression(call.Args[0])
		fc.out.Cvttsd2si("rsi", "xmm0") // size in rsi

		// Call arena_alloc (with auto-growing via realloc)
		fc.out.CallSymbol("flap_arena_alloc")
		// Result in rax, convert to float64
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "dlopen":
		// dlopen(path, flags) - Open a dynamic library
		// path: string (Flap string), flags: number (RTLD_LAZY=1, RTLD_NOW=2)
		// Returns: library handle as float64
		if len(call.Args) != 2 {
			compilerError("dlopen() requires 2 arguments (path, flags)")
		}

		// Evaluate flags argument first (will be in rdi later)
		fc.compileExpression(call.Args[1])
		// Convert flags to integer
		fc.out.Cvttsd2si("r8", "xmm0")
		// Save flags to stack
		fc.out.Emit([]byte{0x41, 0x50}) // push r8

		// Evaluate path argument (Flap string)
		fc.compileExpression(call.Args[0])
		// Convert Flap string to C string (xmm0 has map pointer)
		// Save xmm0 to stack, call conversion function
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0) // C string pointer will be in rax after call
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Call flap_string_to_cstr (result in rax)
		fc.out.CallSymbol("flap_string_to_cstr")

		// Now rax = C string pointer
		// Pop flags from stack to rsi
		fc.out.Emit([]byte{0x41, 0x58})  // pop r8
		fc.out.MovRegToReg("rdi", "rax") // path in rdi
		fc.out.MovRegToReg("rsi", "r8")  // flags in rsi

		// Align stack for C call
		fc.out.SubImmFromReg("rsp", StackSlotSize)

		// Call dlopen
		fc.out.CallSymbol("dlopen")

		// Restore stack
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// rax = library handle (pointer)
		// Convert to float64
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "dlsym":
		// dlsym(handle, symbol) - Get symbol address from library
		// handle: number (library handle from dlopen), symbol: string
		// Returns: symbol address as float64
		if len(call.Args) != 2 {
			compilerError("dlsym() requires 2 arguments (handle, symbol)")
		}

		// Evaluate handle first
		fc.compileExpression(call.Args[0])
		fc.out.Cvttsd2si("r8", "xmm0")
		fc.out.Emit([]byte{0x41, 0x50}) // push r8

		// Evaluate symbol (Flap string)
		fc.compileExpression(call.Args[1])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Convert to C string
		fc.out.CallSymbol("flap_string_to_cstr")

		// Pop handle to rdi
		fc.out.Emit([]byte{0x41, 0x58})  // pop r8
		fc.out.MovRegToReg("rsi", "rax") // symbol in rsi
		fc.out.MovRegToReg("rdi", "r8")  // handle in rdi

		// Align stack
		fc.out.SubImmFromReg("rsp", StackSlotSize)

		// Call dlsym
		fc.out.CallSymbol("dlsym")

		// Restore stack
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// rax = symbol address
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "dlclose":
		// dlclose(handle) - Close a dynamic library
		// handle: number (library handle from dlopen)
		// Returns: 0.0 on success, non-zero on error
		if len(call.Args) != 1 {
			compilerError("dlclose() requires 1 argument (handle)")
		}

		// Evaluate handle
		fc.compileExpression(call.Args[0])
		fc.out.Cvttsd2si("rdi", "xmm0")

		// Align stack
		fc.out.SubImmFromReg("rsp", StackSlotSize)

		// Call dlclose
		fc.out.CallSymbol("dlclose")

		// Restore stack
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// rax = return code (0 on success)
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "readln":
		// readln() - Read a line from stdin, return as Flap string
		if len(call.Args) != 0 {
			compilerError("readln() takes no arguments")
		}

		// Allocate space on stack for getline parameters
		// getline(&lineptr, &n, stdin)
		// lineptr will be allocated by getline
		fc.out.SubImmFromReg("rsp", 16) // 8 bytes for lineptr, 8 for n

		// Initialize lineptr = NULL, n = 0
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.MovRegToMem("rax", "rsp", 0) // lineptr = NULL
		fc.out.MovRegToMem("rax", "rsp", 8) // n = 0

		// Load stdin from libc
		// stdin is at stdin@@GLIBC_2.2.5
		fc.out.LeaSymbolToReg("rdx", "stdin")
		fc.out.MovMemToReg("rdx", "rdx", 0) // dereference stdin pointer

		// Set up getline arguments
		fc.out.MovRegToReg("rdi", "rsp")    // &lineptr
		fc.out.LeaMemToReg("rsi", "rsp", 8) // &n
		// rdx already has stdin

		// Call getline
		fc.trackFunctionCall("getline")
		fc.trackFunctionCall("stdin")
		fc.eb.GenerateCallInstruction("getline")

		// getline returns number of characters read (or -1 on error)
		// lineptr now points to allocated buffer with the line

		// Load lineptr from stack
		fc.out.MovMemToReg("rdi", "rsp", 0)

		// Check if lineptr is NULL (error case)
		fc.out.TestRegReg("rdi", "rdi")
		errorJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpEqual, 0) // Jump if NULL

		// Strip newline if present (getline includes \n)
		// Check if rax > 0 (characters read)
		fc.out.TestRegReg("rax", "rax")
		emptyJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpLessOrEqual, 0) // Jump if no characters

		// Check if last character is newline: byte [rdi + rax - 1] == '\n'
		fc.out.Emit([]byte{0x80, 0x7c, 0x07, 0xff, 0x0a}) // cmp byte [rdi + rax - 1], '\n'
		noNewlineJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpNotEqual, 0) // Jump if not newline

		// Replace newline with null terminator
		fc.out.Emit([]byte{0xc6, 0x44, 0x07, 0xff, 0x00}) // mov byte [rdi + rax - 1], 0

		// Patch no-newline jump to here
		noNewlinePos := fc.eb.text.Len()
		fc.patchJumpImmediate(noNewlineJumpPos+2, int32(noNewlinePos-(noNewlineJumpPos+6)))

		// Patch empty jump to here
		emptyPos := fc.eb.text.Len()
		fc.patchJumpImmediate(emptyJumpPos+2, int32(emptyPos-(emptyJumpPos+6)))

		// Convert C string to Flap string
		// rdi already has lineptr
		fc.out.CallSymbol("cstr_to_flap_string")
		// Result in xmm0

		// Save result
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 16) // Save above the getline locals

		// Free the lineptr buffer
		fc.out.MovMemToReg("rdi", "rsp", StackSlotSize) // Load lineptr from original position
		fc.trackFunctionCall("free")
		fc.eb.GenerateCallInstruction("free")

		// Restore result
		fc.out.MovMemToXmm("xmm0", "rsp", 16)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Clean up stack
		fc.out.AddImmToReg("rsp", 16)

		// Jump to end
		endJumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)

		// Error case: return empty string
		errorPos := fc.eb.text.Len()
		fc.patchJumpImmediate(errorJumpPos+2, int32(errorPos-(errorJumpPos+6)))

		// Clean up stack
		fc.out.AddImmToReg("rsp", 16)

		// Create empty Flap string (count = 0)
		fc.out.MovImmToReg("rdi", "8") // Allocate 8 bytes for count
		fc.trackFunctionCall("malloc")
		fc.eb.GenerateCallInstruction("malloc")
		fc.out.XorRegWithReg("rdx", "rdx")
		fc.out.Cvtsi2sd("xmm0", "rdx")       // xmm0 = 0.0
		fc.out.MovXmmToMem("xmm0", "rax", 0) // [map] = 0.0
		fc.out.MovRegToXmm("xmm0", "rax")    // Return map pointer

		// Patch end jump
		endPos := fc.eb.text.Len()
		fc.patchJumpImmediate(endJumpPos+1, int32(endPos-(endJumpPos+5)))

	case "read_file":
		// read_file(path) - Read entire file, return as Flap string
		// Uses Linux syscalls (open/lseek/read/close) instead of libc for simplicity
		if len(call.Args) != 1 {
			compilerError("read_file() requires 1 argument (path)")
		}

		// Evaluate path argument (Flap string)
		fc.compileExpression(call.Args[0])

		// Convert Flap string to C string (null-terminated)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("flap_string_to_cstr")

		// Allocate stack frame: 32 bytes (fd, size, buffer, result)
		fc.out.SubImmFromReg("rsp", 32)

		// syscall open(path, O_RDONLY=0, mode=0)
		// rax=2 (sys_open), rdi=path, rsi=flags, rdx=mode
		fc.out.MovRegToReg("rdi", "rax")   // path from flap_string_to_cstr
		fc.out.XorRegWithReg("rsi", "rsi") // O_RDONLY = 0
		fc.out.XorRegWithReg("rdx", "rdx") // mode = 0
		fc.out.MovImmToReg("rax", "2")     // sys_open = 2
		fc.out.Emit([]byte{0x0f, 0x05})    // syscall

		// Check if open failed (fd < 0)
		fc.out.TestRegReg("rax", "rax")
		errorJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpLess, 0) // Jump if negative

		// Save fd at [rsp+0]
		fc.out.MovRegToMem("rax", "rsp", 0)

		// syscall lseek(fd, 0, SEEK_END=2) to get file size
		// rax=8 (sys_lseek), rdi=fd, rsi=offset, rdx=whence
		fc.out.MovRegToReg("rdi", "rax")   // fd
		fc.out.XorRegWithReg("rsi", "rsi") // offset = 0
		fc.out.MovImmToReg("rdx", "2")     // SEEK_END = 2
		fc.out.MovImmToReg("rax", "8")     // sys_lseek = 8
		fc.out.Emit([]byte{0x0f, 0x05})    // syscall

		// Save size at [rsp+8]
		fc.out.MovRegToMem("rax", "rsp", 8)

		// syscall lseek(fd, 0, SEEK_SET=0) to rewind
		fc.out.MovMemToReg("rdi", "rsp", 0) // fd from [rsp+0]
		fc.out.XorRegWithReg("rsi", "rsi")  // offset = 0
		fc.out.XorRegWithReg("rdx", "rdx")  // SEEK_SET = 0
		fc.out.MovImmToReg("rax", "8")      // sys_lseek = 8
		fc.out.Emit([]byte{0x0f, 0x05})     // syscall

		// Allocate buffer: malloc(size + 1) for null terminator
		fc.out.MovMemToReg("rdi", "rsp", 8) // size from [rsp+8]
		fc.out.AddImmToReg("rdi", 1)        // +1 for null terminator
		fc.trackFunctionCall("malloc")
		fc.eb.GenerateCallInstruction("malloc")

		// Save buffer at [rsp+16]
		fc.out.MovRegToMem("rax", "rsp", 16)

		// syscall read(fd, buffer, size)
		// rax=0 (sys_read), rdi=fd, rsi=buffer, rdx=count
		fc.out.MovMemToReg("rdi", "rsp", 0)  // fd from [rsp+0]
		fc.out.MovMemToReg("rsi", "rsp", 16) // buffer from [rsp+16]
		fc.out.MovMemToReg("rdx", "rsp", 8)  // size from [rsp+8]
		fc.out.XorRegWithReg("rax", "rax")   // sys_read = 0
		fc.out.Emit([]byte{0x0f, 0x05})      // syscall

		// Add null terminator: buffer[size] = 0
		fc.out.MovMemToReg("rdi", "rsp", 16)        // buffer from [rsp+16]
		fc.out.MovMemToReg("rdx", "rsp", 8)         // size from [rsp+8]
		fc.out.Emit([]byte{0xc6, 0x04, 0x17, 0x00}) // mov byte [rdi + rdx], 0

		// syscall close(fd)
		// rax=3 (sys_close), rdi=fd
		fc.out.MovMemToReg("rdi", "rsp", 0) // fd from [rsp+0]
		fc.out.MovImmToReg("rax", "3")      // sys_close = 3
		fc.out.Emit([]byte{0x0f, 0x05})     // syscall

		// Convert buffer to Flap string
		fc.out.MovMemToReg("rdi", "rsp", 16) // buffer from [rsp+16]
		fc.out.CallSymbol("cstr_to_flap_string")
		// Result in xmm0

		// Save result at [rsp+24]
		fc.out.MovXmmToMem("xmm0", "rsp", 24)

		// Free buffer
		fc.out.MovMemToReg("rdi", "rsp", 16) // buffer from [rsp+16]
		fc.trackFunctionCall("free")
		fc.eb.GenerateCallInstruction("free")

		// Restore result
		fc.out.MovMemToXmm("xmm0", "rsp", 24)

		// Clean up stack frame
		fc.out.AddImmToReg("rsp", 32)

		// Jump to end
		endJumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)

		// Error case: return empty string
		errorPos := fc.eb.text.Len()
		fc.patchJumpImmediate(errorJumpPos+2, int32(errorPos-(errorJumpPos+6)))

		// Clean up stack
		fc.out.AddImmToReg("rsp", 32)

		// Create empty Flap string (count = 0)
		fc.out.MovImmToReg("rdi", "8")
		fc.trackFunctionCall("malloc")
		fc.eb.GenerateCallInstruction("malloc")
		fc.out.XorRegWithReg("rdx", "rdx")
		fc.out.Cvtsi2sd("xmm0", "rdx")
		fc.out.MovXmmToMem("xmm0", "rax", 0)
		fc.out.MovRegToXmm("xmm0", "rax")

		// Patch end jump
		endPos := fc.eb.text.Len()
		fc.patchJumpImmediate(endJumpPos+1, int32(endPos-(endJumpPos+5)))

	case "write_file":
		// write_file(path, content) - Write string to file
		if len(call.Args) != 2 {
			compilerError("write_file() requires 2 arguments (path, content)")
		}

		// Evaluate and convert content first
		fc.compileExpression(call.Args[1])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("flap_string_to_cstr")
		fc.out.PushReg("rax") // Save content C string

		// Evaluate and convert path
		fc.compileExpression(call.Args[0])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rdi", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)
		fc.out.CallSymbol("flap_string_to_cstr")

		// Open file: fopen(path, "w")
		fc.out.MovRegToReg("rdi", "rax") // path
		labelName := fmt.Sprintf("write_mode_%d", fc.stringCounter)
		fc.stringCounter++
		fc.eb.Define(labelName, "w\x00")
		fc.out.LeaSymbolToReg("rsi", labelName) // mode = "w"
		fc.trackFunctionCall("fopen")
		fc.eb.GenerateCallInstruction("fopen")

		// Check if fopen succeeded
		fc.out.TestRegReg("rax", "rax")
		errorJumpPos := fc.eb.text.Len()
		fc.out.JumpConditional(JumpEqual, 0) // Jump if NULL

		// Save FILE* pointer
		fc.out.PushReg("rax")

		// Get content length using strlen
		fc.out.MovMemToReg("rdi", "rsp", StackSlotSize) // content
		fc.trackFunctionCall("strlen")
		fc.eb.GenerateCallInstruction("strlen")
		fc.out.PushReg("rax") // Save length

		// Write file: fwrite(content, 1, length, file)
		fc.out.MovMemToReg("rdi", "rsp", StackSlotSize*2) // content
		fc.out.MovImmToReg("rsi", "1")                    // element size = 1
		fc.out.MovMemToReg("rdx", "rsp", 0)               // length
		fc.out.MovMemToReg("rcx", "rsp", StackSlotSize)   // FILE*
		fc.trackFunctionCall("fwrite")
		fc.eb.GenerateCallInstruction("fwrite")

		// Close file: fclose(file)
		fc.out.MovMemToReg("rdi", "rsp", StackSlotSize) // FILE*
		fc.trackFunctionCall("fclose")
		fc.eb.GenerateCallInstruction("fclose")

		// Clean up stack (length + FILE* + content)
		fc.out.AddImmToReg("rsp", StackSlotSize*3)

		// Return 0 (success)
		fc.out.XorRegWithReg("rax", "rax")
		fc.out.Cvtsi2sd("xmm0", "rax")

		// Jump to end
		endJumpPos := fc.eb.text.Len()
		fc.out.JumpUnconditional(0)

		// Error case: clean up and return -1
		errorPos := fc.eb.text.Len()
		fc.patchJumpImmediate(errorJumpPos+2, int32(errorPos-(errorJumpPos+6)))

		// Clean up content from stack
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Return -1 (error)
		fc.out.MovImmToReg("rax", "-1")
		fc.out.Cvtsi2sd("xmm0", "rax")

		// Patch end jump
		endPos := fc.eb.text.Len()
		fc.patchJumpImmediate(endJumpPos+1, int32(endPos-(endJumpPos+5)))

	case "sizeof_i8", "sizeof_u8":
		// sizeof_i8() / sizeof_u8() - Return size of 8-bit integer (1 byte)
		if len(call.Args) != 0 {
			compilerError("%s() takes no arguments", call.Function)
		}
		// Load 1.0 into xmm0
		fc.out.MovImmToReg("rax", "1")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "sizeof_i16", "sizeof_u16":
		// sizeof_i16() / sizeof_u16() - Return size of 16-bit integer (2 bytes)
		if len(call.Args) != 0 {
			compilerError("%s() takes no arguments", call.Function)
		}
		fc.out.MovImmToReg("rax", "2")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "sizeof_i32", "sizeof_u32", "sizeof_f32":
		// sizeof_i32() / sizeof_u32() / sizeof_f32() - Return size (4 bytes)
		if len(call.Args) != 0 {
			compilerError("%s() takes no arguments", call.Function)
		}
		fc.out.MovImmToReg("rax", "4")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "sizeof_i64", "sizeof_u64", "sizeof_f64", "sizeof_ptr":
		// sizeof_i64() / sizeof_u64() / sizeof_f64() / sizeof_ptr() - Return size (8 bytes)
		if len(call.Args) != 0 {
			compilerError("%s() takes no arguments", call.Function)
		}
		fc.out.MovImmToReg("rax", "8")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "vadd":
		// vadd(v1, v2) - Vector addition using SIMD
		if len(call.Args) != 2 {
			compilerError("vadd() requires exactly 2 arguments")
		}

		// Compile first vector argument -> pointer in xmm0
		fc.compileExpression(call.Args[0])
		// Push pointer to stack (save for later)
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)

		// Compile second vector argument -> pointer in xmm0
		fc.compileExpression(call.Args[1])
		// Convert second vector pointer to rbx
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("rbx", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Get first vector pointer from stack to rax
		fc.out.MovMemToReg("rax", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// For now, just return the first vector pointer to test if the logic works
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "vsub":
		// vsub(v1, v2) - Vector subtraction using SIMD
		if len(call.Args) != 2 {
			compilerError("vsub() requires exactly 2 arguments")
		}

		// Compile first vector argument -> pointer in xmm0
		fc.compileExpression(call.Args[0])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("r12", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Compile second vector argument -> pointer in xmm0
		fc.compileExpression(call.Args[1])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("r13", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Allocate stack space for result
		fc.out.SubImmFromReg("rsp", 16)

		// Load and subtract vectors using SIMD
		fc.out.MovupdMemToXmm("xmm0", "r12", 0)
		fc.out.MovupdMemToXmm("xmm1", "r13", 0)
		fc.out.SubpdXmm("xmm0", "xmm1")

		// Store result and return pointer
		fc.out.MovupdXmmToMem("xmm0", "rsp", 0)
		fc.out.MovRegToReg("rax", "rsp")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "vmul":
		// vmul(v1, v2) - Vector element-wise multiplication using SIMD
		if len(call.Args) != 2 {
			compilerError("vmul() requires exactly 2 arguments")
		}

		// Compile first vector argument
		fc.compileExpression(call.Args[0])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("r12", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Compile second vector argument
		fc.compileExpression(call.Args[1])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("r13", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Allocate stack space for result
		fc.out.SubImmFromReg("rsp", 16)

		// Load and multiply vectors using SIMD
		fc.out.MovupdMemToXmm("xmm0", "r12", 0)
		fc.out.MovupdMemToXmm("xmm1", "r13", 0)
		fc.out.MulpdXmm("xmm0", "xmm1")

		// Store result and return pointer
		fc.out.MovupdXmmToMem("xmm0", "rsp", 0)
		fc.out.MovRegToReg("rax", "rsp")
		fc.out.Cvtsi2sd("xmm0", "rax")

	case "vdiv":
		// vdiv(v1, v2) - Vector element-wise division using SIMD
		if len(call.Args) != 2 {
			compilerError("vdiv() requires exactly 2 arguments")
		}

		// Compile first vector argument
		fc.compileExpression(call.Args[0])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("r12", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Compile second vector argument
		fc.compileExpression(call.Args[1])
		fc.out.SubImmFromReg("rsp", StackSlotSize)
		fc.out.MovXmmToMem("xmm0", "rsp", 0)
		fc.out.MovMemToReg("r13", "rsp", 0)
		fc.out.AddImmToReg("rsp", StackSlotSize)

		// Allocate stack space for result
		fc.out.SubImmFromReg("rsp", 16)

		// Load and divide vectors using SIMD
		fc.out.MovupdMemToXmm("xmm0", "r12", 0)
		fc.out.MovupdMemToXmm("xmm1", "r13", 0)
		fc.out.DivpdXmm("xmm0", "xmm1")

		// Store result and return pointer
		fc.out.MovupdXmmToMem("xmm0", "rsp", 0)
		fc.out.MovRegToReg("rax", "rsp")
		fc.out.Cvtsi2sd("xmm0", "rax")

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
				fc.out.SubImmFromReg("rsp", StackSlotSize)
				fc.out.MovXmmToMem("xmm0", "rsp", 0)
			}
		}

		// Restore arguments from stack in reverse order to registers
		// Last arg is already in xmm0
		for i := len(call.Args) - 2; i >= 0; i-- {
			regName := fmt.Sprintf("xmm%d", i)
			fc.out.MovMemToXmm(regName, "rsp", 0)
			fc.out.AddImmToReg("rsp", StackSlotSize)
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
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)
		fc.out.MovMemToReg("r11", "rsp", StackSlotSize)

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
		fc.out.MovXmmToMem("xmm0", "rsp", StackSlotSize)
		fc.out.MovMemToReg("r11", "rsp", StackSlotSize)

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
	if VerboseMode {
		fmt.Fprintln(os.Stderr, "This feature requires runtime support for concurrency")
	}
	compilerError("concurrent gather operator ||| is not yet implemented")
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
		for _, clause := range e.Clauses {
			if clause.Guard != nil {
				collectFunctionCalls(clause.Guard, calls)
			}
			collectFunctionCalls(clause.Result, calls)
		}
		if e.DefaultExpr != nil {
			collectFunctionCalls(e.DefaultExpr, calls)
		}
	case *LambdaExpr:
		collectFunctionCalls(e.Body, calls)
	case *RangeExpr:
		collectFunctionCalls(e.Start, calls)
		collectFunctionCalls(e.End, calls)
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
		"getpid": true, "me": true,
		"println": true, // println is a builtin optimization, not a dependency
		// Math functions (hardware instructions)
		"sqrt": true, "sin": true, "cos": true, "tan": true,
		"asin": true, "acos": true, "atan": true, "atan2": true,
		"exp": true, "log": true, "pow": true,
		"floor": true, "ceil": true, "round": true,
		"abs": true,
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

// filterPrivateFunctions removes all function definitions with names starting with _
// Private functions (starting with _) are not exported when importing modules
func filterPrivateFunctions(program *Program) {
	var publicStmts []Statement
	for _, stmt := range program.Statements {
		// Check if this is an assignment statement
		if assign, ok := stmt.(*AssignStmt); ok {
			// Check if the name starts with _
			if len(assign.Name) > 0 && assign.Name[0] == '_' {
				// Skip private functions - don't export them
				continue
			}
		}
		// Keep all non-private statements
		publicStmts = append(publicStmts, stmt)
	}
	program.Statements = publicStmts
}

func processImports(program *Program) error {
	// Find all import statements (both Git and C imports)
	var imports []*ImportStmt
	var cImports []*CImportStmt
	for _, stmt := range program.Statements {
		if importStmt, ok := stmt.(*ImportStmt); ok {
			imports = append(imports, importStmt)
		}
		if cImportStmt, ok := stmt.(*CImportStmt); ok {
			cImports = append(cImports, cImportStmt)
		}
	}

	// Process C imports first (simpler, no dependency resolution)
	if len(cImports) > 0 {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Processing %d C import(s)\n", len(cImports))
		}
		for _, cImp := range cImports {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Importing C library %s as %s\n", cImp.Library, cImp.Alias)
			}
			// C imports are handled during compilation, not here
			// They just need to be tracked for namespace resolution
		}
	}

	if len(imports) == 0 {
		return nil // No Git imports to process
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Processing %d Git import(s)\n", len(imports))
	}

	// Process each import
	for _, imp := range imports {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Importing %s", imp.URL)
		}
		if imp.Version != "" {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "@%s", imp.Version)
			}
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, " as %s\n", imp.Alias)
		}

		// Clone/cache the repository
		repoPath, err := EnsureRepoClonedWithVersion(imp.URL, imp.Version, UpdateDepsFlag)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %v", imp.URL, err)
		}

		// Find all .flap files in the repository
		flapFiles, err := FindFlapFiles(repoPath)
		if err != nil {
			return fmt.Errorf("failed to find .flap files in %s: %v", repoPath, err)
		}

		// Parse and merge each .flap file with namespace handling
		for _, flapFile := range flapFiles {
			depContent, err := os.ReadFile(flapFile)
			if err != nil {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", flapFile, err)
				}
				continue
			}

			depParser := NewParserWithFilename(string(depContent), flapFile)
			depProgram := depParser.ParseProgram()

			// Filter out private functions (names starting with _)
			filterPrivateFunctions(depProgram)

			// If alias is "*", import into same namespace
			// Otherwise, prefix all function names with namespace
			if imp.Alias != "*" {
				addNamespaceToFunctions(depProgram, imp.Alias)
			}

			// Prepend dependency program to main program
			program.Statements = append(depProgram.Statements, program.Statements...)
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Loaded %s from %s\n", flapFile, imp.URL)
			}
		}
	}

	// Remove import statements from program (they've been processed)
	var filteredStmts []Statement
	for _, stmt := range program.Statements {
		if _, ok := stmt.(*ImportStmt); !ok {
			filteredStmts = append(filteredStmts, stmt)
		}
	}
	program.Statements = filteredStmts

	// Debug: print final program
	if os.Getenv("DEBUG_FLAP") != "" {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG processImports: final program after import processing:\n")
		}
		for i, stmt := range program.Statements {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "  [%d] %s\n", i, stmt.String())
			}
		}
	}

	return nil
}

func addNamespaceToFunctions(program *Program, namespace string) {
	for _, stmt := range program.Statements {
		if assign, ok := stmt.(*AssignStmt); ok {
			// Add namespace prefix to function name
			assign.Name = namespace + "." + assign.Name
		}
	}
}

func CompileFlap(inputPath string, outputPath string, platform Platform) (err error) {
	// Recover from parser panics and convert to errors
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = fmt.Errorf("panic during compilation: %v", r)
			}
		}
	}()

	// Read input file
	content, readErr := os.ReadFile(inputPath)
	if readErr != nil {
		return fmt.Errorf("failed to read %s: %v", inputPath, readErr)
	}

	// Parse main file
	parser := NewParserWithFilename(string(content), inputPath)
	program := parser.ParseProgram()

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Parsed program:\n%s\n", program.String())
	}

	// Process explicit import statements
	err = processImports(program)
	if err != nil {
		return fmt.Errorf("failed to process imports: %v", err)
	}

	// Check for unknown functions and resolve dependencies
	// Build combined source code (dependencies + main)
	var combinedSource string
	unknownFuncs := getUnknownFunctions(program)
	if len(unknownFuncs) > 0 {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Resolving dependencies for: %v\n", unknownFuncs)
		}

		// Resolve dependencies
		repos := ResolveDependencies(unknownFuncs)
		if len(repos) > 0 {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Loading dependencies from %d repositories\n", len(repos))
			}

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
						if VerboseMode {
							fmt.Fprintf(os.Stderr, "Warning: failed to read %s: %v\n", flapFile, err)
						}
						continue
					}

					depParser := NewParserWithFilename(string(depContent), flapFile)
					depProgram := depParser.ParseProgram()

					// Prepend dependency program to main program (dependencies must be defined before use)
					program.Statements = append(depProgram.Statements, program.Statements...)
					// Prepend dependency source to combined source
					combinedSource = string(depContent) + "\n" + combinedSource
					if VerboseMode {
						fmt.Fprintf(os.Stderr, "Loaded %s from %s\n", flapFile, repoURL)
					}
				}
			}
		}
	}
	// Append main file source
	combinedSource = combinedSource + string(content)

	// Compile
	compiler, err := NewFlapCompiler(platform)
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

// compileARM64 compiles a program for ARM64 architecture (macOS)
func (fc *FlapCompiler) compileARM64(program *Program, outputPath string) error {
	// Create ARM64 code generator
	acg := NewARM64CodeGen(fc.eb)

	// Generate code
	if err := acg.CompileProgram(program); err != nil {
		return err
	}

	// Write Mach-O file
	return fc.writeMachOARM64(outputPath)
}

// compileRiscv64 compiles a program for RISC-V64 architecture
func (fc *FlapCompiler) compileRiscv64(program *Program, outputPath string) error {
	// Create RISC-V64 code generator
	rcg := NewRiscvCodeGen(fc.eb)

	// Generate code
	if err := rcg.CompileProgram(program); err != nil {
		return err
	}

	// Write ELF file
	return fc.writeELFRiscv64(outputPath)
}

// writeMachOARM64 writes an ARM64 Mach-O executable for macOS
func (fc *FlapCompiler) writeMachOARM64(outputPath string) error {
	// First, write all rodata symbols to the rodata buffer and assign addresses
	pageSize := uint64(0x4000) // 16KB page size for ARM64
	textSize := uint64(fc.eb.text.Len())
	textSizeAligned := (textSize + pageSize - 1) &^ (pageSize - 1)

	// Calculate rodata address (comes after __TEXT segment)
	rodataAddr := pageSize + textSizeAligned

	if VerboseMode {
		fmt.Fprintln(os.Stderr, "-> Writing rodata symbols")
	}

	// Get all rodata symbols and write them
	rodataSymbols := fc.eb.RodataSection()
	currentAddr := rodataAddr
	for symbol, value := range rodataSymbols {
		fc.eb.WriteRodata([]byte(value))
		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "   %s at 0x%x (%d bytes)\n", symbol, fc.eb.consts[symbol].addr, len(value))
		}
	}

	rodataSize := fc.eb.rodata.Len()

	// Now patch PC-relative relocations in the text
	if VerboseMode && len(fc.eb.pcRelocations) > 0 {
		fmt.Fprintf(os.Stderr, "-> Patching %d PC-relative relocations\n", len(fc.eb.pcRelocations))
	}
	textAddr := pageSize
	fc.eb.PatchPCRelocations(textAddr, rodataAddr, rodataSize)

	// Use the existing Mach-O writer infrastructure
	if err := fc.eb.WriteMachO(); err != nil {
		return fmt.Errorf("failed to write Mach-O: %v", err)
	}

	// Write the executable
	machoBytes := fc.eb.elf.Bytes() // Note: elf buffer is reused for Mach-O

	// Debug: check flags in buffer before writing
	if len(machoBytes) >= 28 {
		flagsInBytes := binary.LittleEndian.Uint32(machoBytes[24:28])
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: machoBytes flags before writing = 0x%08x\n", flagsInBytes)
		}
	}

	if err := os.WriteFile(outputPath, machoBytes, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %v", err)
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Wrote ARM64 Mach-O executable: %s\n", outputPath)
		fmt.Fprintf(os.Stderr, "   Text size: %d bytes\n", fc.eb.text.Len())
		fmt.Fprintf(os.Stderr, "   Rodata size: %d bytes\n", rodataSize)
	}

	return nil
}

// writeELFRiscv64 writes a RISC-V64 ELF executable
func (fc *FlapCompiler) writeELFRiscv64(outputPath string) error {
	// For now, create a static ELF (no dynamic linking)
	// This is simpler and works with Spike

	textBytes := fc.eb.text.Bytes()
	rodataBytes := fc.eb.rodata.Bytes()

	// Generate basic ELF header and program headers for RISC-V64
	fc.eb.WriteELFHeader()

	// Write the executable
	elfBytes := fc.eb.Bytes()
	if err := os.WriteFile(outputPath, elfBytes, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %v", err)
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Wrote RISC-V64 executable: %s\n", outputPath)
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "   Text size: %d bytes\n", len(textBytes))
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "   Rodata size: %d bytes\n", len(rodataBytes))
		}
	}

	return nil
}
