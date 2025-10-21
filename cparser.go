package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// CParser is a simple C header file parser for extracting constants, macros, and function signatures
type CParser struct {
	tokens       []CToken
	pos          int
	results      *CHeaderConstants
	maxTokens    int // Maximum number of tokens to prevent memory exhaustion
	maxParseOps  int // Maximum number of parse operations to prevent infinite loops
	parseOpCount int // Current parse operation count
}

const (
	// Safety limits
	MaxTokensPerFile   = 1000000  // 1M tokens max per file
	MaxParseOperations = 10000000 // 10M parse ops max
)

// CTokenType represents the type of a C token
type CTokenType int

const (
	CTokEOF CTokenType = iota
	CTokIdentifier
	CTokNumber
	CTokString
	CTokPunctuation
	CTokPreprocessor
	CTokNewline
)

// CToken represents a single token from the C source
type CToken struct {
	Type  CTokenType
	Value string
	Line  int
}

// NewCParser creates a new C header parser
func NewCParser() *CParser {
	return &CParser{
		results:      NewCHeaderConstants(),
		maxTokens:    MaxTokensPerFile,
		maxParseOps:  MaxParseOperations,
		parseOpCount: 0,
	}
}

// ParseFile parses a C header file and extracts constants, macros, and function signatures
func (p *CParser) ParseFile(filepath string) (*CHeaderConstants, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	// Tokenize the input
	p.tokens = p.tokenize(string(content))
	p.pos = 0

	// Parse top-level declarations with safety limit
	for !p.isAtEnd() {
		p.parseOpCount++
		if p.parseOpCount > p.maxParseOps {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Warning: reached max parse operations limit (%d), stopping parse\n", p.maxParseOps)
			}
			break
		}
		p.parseTopLevel()
	}

	return p.results, nil
}

// tokenize converts C source code into tokens
func (p *CParser) tokenize(source string) []CToken {
	var tokens []CToken
	lines := strings.Split(source, "\n")

	for lineNum, line := range lines {
		i := 0
		for i < len(line) {
			// Safety check: prevent token explosion
			if len(tokens) >= p.maxTokens {
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Warning: reached max tokens limit (%d), stopping tokenization\n", p.maxTokens)
				}
				return tokens
			}
			// Skip whitespace (except newlines)
			if unicode.IsSpace(rune(line[i])) && line[i] != '\n' {
				i++
				continue
			}

			// Preprocessor directive
			if i == 0 && line[i] == '#' {
				// Find the directive name
				start := i
				i++
				for i < len(line) && !unicode.IsSpace(rune(line[i])) {
					i++
				}
				directive := line[start:i]

				// Get the rest of the line (the directive content)
				for i < len(line) && unicode.IsSpace(rune(line[i])) {
					i++
				}
				rest := strings.TrimSpace(line[i:])

				tokens = append(tokens, CToken{
					Type:  CTokPreprocessor,
					Value: directive + " " + rest,
					Line:  lineNum + 1,
				})
				break // Done with this line
			}

			// Single-line comment
			if i < len(line)-1 && line[i] == '/' && line[i+1] == '/' {
				break // Skip rest of line
			}

			// Multi-line comment (simplified - assumes it ends on same line)
			if i < len(line)-1 && line[i] == '/' && line[i+1] == '*' {
				i += 2
				for i < len(line)-1 {
					if line[i] == '*' && line[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}

			// String literal
			if line[i] == '"' {
				start := i
				i++
				for i < len(line) && line[i] != '"' {
					if line[i] == '\\' {
						i++ // Skip escaped character
					}
					i++
				}
				if i < len(line) {
					i++ // Skip closing quote
				}
				tokens = append(tokens, CToken{
					Type:  CTokString,
					Value: line[start:i],
					Line:  lineNum + 1,
				})
				continue
			}

			// Number (hex, decimal, binary)
			if unicode.IsDigit(rune(line[i])) || (line[i] == '0' && i+1 < len(line) && (line[i+1] == 'x' || line[i+1] == 'X' || line[i+1] == 'b' || line[i+1] == 'B')) {
				start := i
				if line[i] == '0' && i+1 < len(line) && (line[i+1] == 'x' || line[i+1] == 'X') {
					i += 2 // Skip 0x
					for i < len(line) && (unicode.IsDigit(rune(line[i])) || (line[i] >= 'a' && line[i] <= 'f') || (line[i] >= 'A' && line[i] <= 'F')) {
						i++
					}
				} else if line[i] == '0' && i+1 < len(line) && (line[i+1] == 'b' || line[i+1] == 'B') {
					i += 2 // Skip 0b
					for i < len(line) && (line[i] == '0' || line[i] == '1') {
						i++
					}
				} else {
					for i < len(line) && unicode.IsDigit(rune(line[i])) {
						i++
					}
				}
				// Skip type suffixes (u, l, ul, ll, ull, etc.)
				for i < len(line) && (line[i] == 'u' || line[i] == 'U' || line[i] == 'l' || line[i] == 'L') {
					i++
				}
				tokens = append(tokens, CToken{
					Type:  CTokNumber,
					Value: line[start:i],
					Line:  lineNum + 1,
				})
				continue
			}

			// Identifier or keyword
			if unicode.IsLetter(rune(line[i])) || line[i] == '_' {
				start := i
				for i < len(line) && (unicode.IsLetter(rune(line[i])) || unicode.IsDigit(rune(line[i])) || line[i] == '_') {
					i++
				}
				tokens = append(tokens, CToken{
					Type:  CTokIdentifier,
					Value: line[start:i],
					Line:  lineNum + 1,
				})
				continue
			}

			// Punctuation (operators, braces, etc.)
			// Handle multi-character operators
			if i < len(line)-1 {
				twoChar := line[i : i+2]
				if twoChar == "<<" || twoChar == ">>" || twoChar == "##" || twoChar == "::" {
					tokens = append(tokens, CToken{
						Type:  CTokPunctuation,
						Value: twoChar,
						Line:  lineNum + 1,
					})
					i += 2
					continue
				}
			}

			// Single character punctuation
			tokens = append(tokens, CToken{
				Type:  CTokPunctuation,
				Value: string(line[i]),
				Line:  lineNum + 1,
			})
			i++
		}
	}

	return tokens
}

// parseTopLevel parses a top-level declaration
func (p *CParser) parseTopLevel() {
	if p.isAtEnd() {
		return
	}

	tok := p.peek()

	// Preprocessor directive
	if tok.Type == CTokPreprocessor {
		p.parsePreprocessor()
		return
	}

	// Function declaration
	if tok.Type == CTokIdentifier {
		// Try to parse as function declaration
		// Look for pattern: [extern] [MACRO]* type [MACRO]* identifier ( params ) ;
		if p.tryParseFunctionDecl() {
			return
		}
	}

	// Skip unrecognized tokens
	p.advance()
}

// parsePreprocessor handles preprocessor directives
func (p *CParser) parsePreprocessor() {
	if p.isAtEnd() {
		return
	}

	tok := p.advance()
	parts := strings.SplitN(tok.Value, " ", 2)
	if len(parts) < 2 {
		return
	}

	directive := parts[0]
	rest := strings.TrimSpace(parts[1])

	switch directive {
	case "#define":
		p.parseDefine(rest)
	case "#include":
		// Could handle includes for recursive parsing, but skip for now
	}
}

// parseDefine parses a #define directive
func (p *CParser) parseDefine(content string) {
	// Check for function-like macro: NAME(params) body
	if idx := strings.Index(content, "("); idx != -1 {
		nameEnd := idx
		name := strings.TrimSpace(content[:nameEnd])

		// Find closing paren
		closeIdx := strings.Index(content, ")")
		if closeIdx == -1 {
			return
		}

		// Get the body after the closing paren
		body := strings.TrimSpace(content[closeIdx+1:])

		// Store the macro
		p.results.Macros[name] = body

		if VerboseMode {
			fmt.Fprintf(os.Stderr, "  Macro: %s = %s\n", name, body)
		}
		return
	}

	// Simple constant: NAME value
	parts := strings.Fields(content)
	if len(parts) < 2 {
		return
	}

	name := parts[0]
	valueStr := strings.Join(parts[1:], " ")

	// Remove inline comments
	if idx := strings.Index(valueStr, "//"); idx != -1 {
		valueStr = strings.TrimSpace(valueStr[:idx])
	}
	if idx := strings.Index(valueStr, "/*"); idx != -1 {
		valueStr = strings.TrimSpace(valueStr[:idx])
	}

	// Try to evaluate the constant value
	value, ok := p.evalConstant(valueStr)
	if ok {
		p.results.Constants[name] = value
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "  Constant: %s = %d (0x%x)\n", name, value, value)
		}
	}
}

// evalConstant evaluates a constant expression
func (p *CParser) evalConstant(expr string) (int64, bool) {
	expr = strings.TrimSpace(expr)

	// Remove type suffixes
	expr = strings.TrimSuffix(expr, "u")
	expr = strings.TrimSuffix(expr, "U")
	expr = strings.TrimSuffix(expr, "l")
	expr = strings.TrimSuffix(expr, "L")
	expr = strings.TrimSuffix(expr, "ul")
	expr = strings.TrimSuffix(expr, "UL")
	expr = strings.TrimSuffix(expr, "ll")
	expr = strings.TrimSuffix(expr, "LL")
	expr = strings.TrimSuffix(expr, "ull")
	expr = strings.TrimSuffix(expr, "ULL")

	// Try hex number
	if strings.HasPrefix(expr, "0x") || strings.HasPrefix(expr, "0X") {
		if val, err := strconv.ParseInt(expr[2:], 16, 64); err == nil {
			return val, true
		}
	}

	// Try binary number
	if strings.HasPrefix(expr, "0b") || strings.HasPrefix(expr, "0B") {
		if val, err := strconv.ParseInt(expr[2:], 2, 64); err == nil {
			return val, true
		}
	}

	// Try decimal number
	if val, err := strconv.ParseInt(expr, 10, 64); err == nil {
		return val, true
	}

	// Try to resolve reference to another constant
	if val, ok := p.results.Constants[expr]; ok {
		return val, true
	}

	// Try simple expressions (parentheses)
	if strings.HasPrefix(expr, "(") && strings.HasSuffix(expr, ")") {
		return p.evalConstant(expr[1 : len(expr)-1])
	}

	// Try bitwise shift: value << shift
	if idx := strings.Index(expr, "<<"); idx != -1 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+2:])
		leftVal, leftOk := p.evalConstant(left)
		rightVal, rightOk := p.evalConstant(right)
		if leftOk && rightOk {
			return leftVal << uint(rightVal), true
		}
	}

	// Try bitwise shift: value >> shift
	if idx := strings.Index(expr, ">>"); idx != -1 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+2:])
		leftVal, leftOk := p.evalConstant(left)
		rightVal, rightOk := p.evalConstant(right)
		if leftOk && rightOk {
			return leftVal >> uint(rightVal), true
		}
	}

	// Try bitwise OR: value | value
	if idx := strings.Index(expr, "|"); idx != -1 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+1:])
		leftVal, leftOk := p.evalConstant(left)
		rightVal, rightOk := p.evalConstant(right)
		if leftOk && rightOk {
			return leftVal | rightVal, true
		}
	}

	// Try bitwise AND: value & value
	if idx := strings.Index(expr, "&"); idx != -1 {
		left := strings.TrimSpace(expr[:idx])
		right := strings.TrimSpace(expr[idx+1:])
		leftVal, leftOk := p.evalConstant(left)
		rightVal, rightOk := p.evalConstant(right)
		if leftOk && rightOk {
			return leftVal & rightVal, true
		}
	}

	return 0, false
}

// tryParseFunctionDecl attempts to parse a function declaration
func (p *CParser) tryParseFunctionDecl() bool {
	saved := p.pos

	// Skip 'extern' if present
	if p.match("extern") {
		p.advance()
	}

	// Collect return type tokens (may include macros like SDL_DECLSPEC)
	var returnTypeParts []string
	foundOpenParen := false

	maxReturnTypeParts := 30 // Reasonable limit for return type components
	for !p.isAtEnd() && len(returnTypeParts) < maxReturnTypeParts {
		tok := p.peek()

		if tok.Type == CTokPunctuation && tok.Value == "(" {
			foundOpenParen = true
			break
		}

		if tok.Type == CTokPunctuation && tok.Value == ";" {
			// Not a function
			p.pos = saved
			return false
		}

		returnTypeParts = append(returnTypeParts, tok.Value)
		p.advance()
	}

	if !foundOpenParen || len(returnTypeParts) == 0 {
		p.pos = saved
		return false
	}

	// The last identifier before '(' is the function name
	// Everything else is the return type (possibly with macros)
	funcName := returnTypeParts[len(returnTypeParts)-1]
	returnTypeParts = returnTypeParts[:len(returnTypeParts)-1]

	// Filter out common C macros to get the actual return type
	var actualReturnType []string
	for _, part := range returnTypeParts {
		// Skip common SDL/library macros
		if part == "SDL_DECLSPEC" || part == "SDLCALL" || part == "RAYLIB_API" ||
			part == "SDL_MALLOC" || part == "SDL_FORCE_INLINE" || part == "static" ||
			part == "inline" || strings.HasPrefix(part, "__attribute__") {
			continue
		}
		actualReturnType = append(actualReturnType, part)
	}

	returnType := strings.Join(actualReturnType, " ")
	if returnType == "" {
		returnType = "void"
	}

	// Parse parameter list
	p.advance() // skip '('
	params := p.parseParameters()

	// Look for closing ';'
	if !p.match(";") {
		p.pos = saved
		return false
	}
	p.advance() // skip ';'

	// Store the function signature
	p.results.Functions[funcName] = &CFunctionSignature{
		ReturnType: returnType,
		Params:     params,
	}

	if VerboseMode {
		paramStrs := make([]string, len(params))
		for i, param := range params {
			if param.Name != "" {
				paramStrs[i] = param.Type + " " + param.Name
			} else {
				paramStrs[i] = param.Type
			}
		}
		fmt.Fprintf(os.Stderr, "  Function: %s %s(%s)\n", returnType, funcName, strings.Join(paramStrs, ", "))
	}

	return true
}

// parseParameters parses function parameters
func (p *CParser) parseParameters() []CFunctionParam {
	var params []CFunctionParam

	// Handle empty parameter list or (void)
	if p.match(")") {
		return params
	}

	if p.match("void") {
		p.advance()
		if p.match(")") {
			return params
		}
	}

	maxParams := 100 // Reasonable limit for function parameters
	for !p.isAtEnd() && len(params) < maxParams {
		// Parse one parameter: type [name]
		var paramTypeParts []string
		var paramName string

		maxTypeParts := 20 // Reasonable limit for type components
		for !p.isAtEnd() && len(paramTypeParts) < maxTypeParts {
			tok := p.peek()

			// End of parameter
			if tok.Type == CTokPunctuation && (tok.Value == "," || tok.Value == ")") {
				break
			}

			paramTypeParts = append(paramTypeParts, tok.Value)
			p.advance()
		}

		if len(paramTypeParts) == 0 {
			break
		}

		// Last identifier might be the parameter name
		// If it doesn't look like a type component, it's the name
		lastPart := paramTypeParts[len(paramTypeParts)-1]
		if len(paramTypeParts) > 1 && !strings.Contains(lastPart, "*") &&
			lastPart != "const" && lastPart != "struct" && lastPart != "enum" &&
			!strings.HasPrefix(lastPart, "SDL_") && !strings.HasPrefix(lastPart, "RL_") {
			paramName = lastPart
			paramTypeParts = paramTypeParts[:len(paramTypeParts)-1]
		}

		// Filter out macros from parameter type
		var actualParamType []string
		for _, part := range paramTypeParts {
			if part == "SDL_DECLSPEC" || part == "SDLCALL" || part == "const" {
				continue
			}
			actualParamType = append(actualParamType, part)
		}

		paramType := strings.Join(actualParamType, " ")
		if paramType != "" {
			params = append(params, CFunctionParam{
				Type: paramType,
				Name: paramName,
			})
		}

		// Check for comma (more parameters) or closing paren
		if p.match(",") {
			p.advance()
			continue
		}

		if p.match(")") {
			break
		}
	}

	return params
}

// Token navigation helpers
func (p *CParser) peek() CToken {
	if p.isAtEnd() {
		return CToken{Type: CTokEOF}
	}
	return p.tokens[p.pos]
}

func (p *CParser) advance() CToken {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.tokens[p.pos-1]
}

func (p *CParser) isAtEnd() bool {
	return p.pos >= len(p.tokens)
}

func (p *CParser) match(value string) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Value == value
}

// ParseCHeaderFile is a convenience function that parses a C header file
func ParseCHeaderFile(filepath string) (*CHeaderConstants, error) {
	parser := NewCParser()
	return parser.ParseFile(filepath)
}
