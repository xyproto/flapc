package main

import (
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
	TOKEN_FSTRING // f"..." interpolated string
	TOKEN_PLUS
	TOKEN_MINUS
	TOKEN_STAR
	TOKEN_SLASH
	TOKEN_MOD
	TOKEN_EQUALS
	TOKEN_COLON_EQUALS
	TOKEN_PLUS_EQUALS  // +=
	TOKEN_MINUS_EQUALS // -=
	TOKEN_STAR_EQUALS  // *=
	TOKEN_SLASH_EQUALS // /=
	TOKEN_MOD_EQUALS   // %=
	TOKEN_LPAREN
	TOKEN_RPAREN
	TOKEN_COMMA
	TOKEN_COLON
	TOKEN_SEMICOLON
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
	TOKEN_AT_MINUS      // @-
	TOKEN_AT_EQUALS     // @=
	TOKEN_IN            // in keyword
	TOKEN_LBRACE        // {
	TOKEN_RBRACE        // }
	TOKEN_LBRACKET      // [
	TOKEN_RBRACKET      // ]
	TOKEN_ARROW         // ->
	TOKEN_FAT_ARROW     // =>
	TOKEN_LEFT_ARROW    // <-
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
	TOKEN_INCREMENT     // ++
	TOKEN_DECREMENT     // --
	TOKEN_FMA           // *+ (fused multiply-add)
	TOKEN_OR_BANG       // or! (error handling)
	TOKEN_ME            // me (self-reference)
	TOKEN_RET           // ret keyword (return/break)
	TOKEN_AT_FIRST      // @first (first iteration)
	TOKEN_AT_LAST       // @last (last iteration)
	TOKEN_AT_COUNTER    // @counter (iteration counter)
	TOKEN_AT_KEY        // @key (current element/key)
	TOKEN_PIPE_B        // |b (bitwise OR)
	TOKEN_AMP_B         // &b (bitwise AND)
	TOKEN_CARET_B       // ^b (bitwise XOR)
	TOKEN_TILDE_B       // ~b (bitwise NOT)
	TOKEN_CARET         // ^ (head of list)
	TOKEN_AMP           // & (tail of list)
	TOKEN_LT_B          // <b (shift left)
	TOKEN_GT_B          // >b (shift right)
	TOKEN_LTLT_B        // <<b (rotate left)
	TOKEN_GTGT_B        // >>b (rotate right)
	TOKEN_AS            // as (type casting)
	// C type keywords
	TOKEN_I8   // i8
	TOKEN_I16  // i16
	TOKEN_I32  // i32
	TOKEN_I64  // i64
	TOKEN_U8   // u8
	TOKEN_U16  // u16
	TOKEN_U32  // u32
	TOKEN_U64  // u64
	TOKEN_F32  // f32
	TOKEN_F64  // f64
	TOKEN_CSTR // cstr
	TOKEN_PTR  // ptr
	// Flap type keywords
	TOKEN_NUMBER_TYPE // number
	TOKEN_STRING_TYPE // string (type)
	TOKEN_LIST_TYPE   // list (type)
	TOKEN_USE         // use (import)
	TOKEN_IMPORT      // import (with git URL)
	TOKEN_FROM        // from (C library imports)
	TOKEN_DOT         // . (for namespaced calls)
	TOKEN_DOTDOTEQ    // ..= (inclusive range operator)
	TOKEN_DOTDOTLT    // ..< (exclusive range operator)
	TOKEN_UNSAFE      // unsafe (architecture-specific code blocks)
	TOKEN_SYSCALL     // syscall (system call in unsafe blocks)
	TOKEN_ARENA       // arena (arena memory blocks)
	TOKEN_DEFER       // defer (deferred execution)
)

// Code generation constants
const (
	// Jump instruction sizes on x86-64
	UnconditionalJumpSize = 5 // Size of JumpUnconditional (0xe9 + 4-byte offset)
	ConditionalJumpSize   = 6 // Size of JumpConditional (0x0f 0x8X + 4-byte offset)

	// Stack layout
	StackSlotSize = 8 // Size of a stack slot (8 bytes for float64/pointer)

	// Byte manipulation
	ByteMask = 0xFF // Mask for extracting a single byte
)

type Token struct {
	Type  TokenType
	Value string
	Line  int
}

// isHexDigit checks if a byte is a valid hexadecimal digit
func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
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
		// Recursively get the next token after the comment
		return l.NextToken()
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
			// Skip escaped characters (including escaped quotes)
			if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
				l.pos += 2 // Skip backslash and next character
			} else {
				l.pos++
			}
		}
		value := l.input[start:l.pos]
		l.pos++ // skip closing "
		// Process escape sequences like \n, \t, etc.
		value = processEscapeSequences(value)
		return Token{Type: TOKEN_STRING, Value: value, Line: l.line}
	}

	// Number (including hex 0x... and binary 0b...)
	if unicode.IsDigit(rune(ch)) {
		start := l.pos

		// Check for hex or binary prefix
		if ch == '0' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			if next == 'x' || next == 'X' {
				// Hexadecimal: 0x[0-9a-fA-F]+
				l.pos += 2 // skip '0x'
				if l.pos >= len(l.input) || !isHexDigit(l.input[l.pos]) {
					// Invalid hex literal
					return Token{Type: TOKEN_NUMBER, Value: "0", Line: l.line}
				}
				for l.pos < len(l.input) && isHexDigit(l.input[l.pos]) {
					l.pos++
				}
				return Token{Type: TOKEN_NUMBER, Value: l.input[start:l.pos], Line: l.line}
			} else if next == 'b' || next == 'B' {
				// Binary: 0b[01]+
				l.pos += 2 // skip '0b'
				if l.pos >= len(l.input) || (l.input[l.pos] != '0' && l.input[l.pos] != '1') {
					// Invalid binary literal
					return Token{Type: TOKEN_NUMBER, Value: "0", Line: l.line}
				}
				for l.pos < len(l.input) && (l.input[l.pos] == '0' || l.input[l.pos] == '1') {
					l.pos++
				}
				return Token{Type: TOKEN_NUMBER, Value: l.input[start:l.pos], Line: l.line}
			}
		}

		// Regular decimal number
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

		// Check for f-string: f"..."
		if value == "f" && l.pos < len(l.input) && l.input[l.pos] == '"' {
			l.pos++ // skip opening "
			fstringStart := l.pos
			for l.pos < len(l.input) && l.input[l.pos] != '"' {
				// Skip escaped characters (including escaped quotes)
				if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
					l.pos += 2
				} else {
					l.pos++
				}
			}
			fstringValue := l.input[fstringStart:l.pos]
			l.pos++ // skip closing "
			return Token{Type: TOKEN_FSTRING, Value: fstringValue, Line: l.line}
		}

		// Check for keywords
		switch value {
		case "in":
			return Token{Type: TOKEN_IN, Value: value, Line: l.line}
		case "and":
			return Token{Type: TOKEN_AND, Value: value, Line: l.line}
		case "or":
			// Check for or!
			if l.pos < len(l.input) && l.input[l.pos] == '!' {
				l.pos++ // consume the !
				return Token{Type: TOKEN_OR_BANG, Value: "or!", Line: l.line}
			}
			return Token{Type: TOKEN_OR, Value: value, Line: l.line}
		case "not":
			return Token{Type: TOKEN_NOT, Value: value, Line: l.line}
		case "me":
			return Token{Type: TOKEN_ME, Value: value, Line: l.line}
		case "ret":
			return Token{Type: TOKEN_RET, Value: value, Line: l.line}
		case "use":
			return Token{Type: TOKEN_USE, Value: value, Line: l.line}
		case "import":
			return Token{Type: TOKEN_IMPORT, Value: value, Line: l.line}
		case "from":
			return Token{Type: TOKEN_FROM, Value: value, Line: l.line}
		case "as":
			return Token{Type: TOKEN_AS, Value: value, Line: l.line}
		case "unsafe":
			return Token{Type: TOKEN_UNSAFE, Value: value, Line: l.line}
		case "syscall":
			return Token{Type: TOKEN_SYSCALL, Value: value, Line: l.line}
		case "arena":
			return Token{Type: TOKEN_ARENA, Value: value, Line: l.line}
		case "defer":
			return Token{Type: TOKEN_DEFER, Value: value, Line: l.line}
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
			// Note: All type keywords (i8, i16, i32, i64, u8, u16, u32, u64, f32, f64,
			// cstr, ptr, number, string, list) are contextual keywords.
			// They are only treated as type keywords after "as" in cast expressions.
			// Otherwise they can be used as identifiers.
		}

		return Token{Type: TOKEN_IDENT, Value: value, Line: l.line}
	}

	// Operators and punctuation
	switch ch {
	case '+':
		l.pos++
		// Check for ++
		if l.pos < len(l.input) && l.input[l.pos] == '+' {
			l.pos++
			return Token{Type: TOKEN_INCREMENT, Value: "++", Line: l.line}
		}
		// Check for +=
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos++
			return Token{Type: TOKEN_PLUS_EQUALS, Value: "+=", Line: l.line}
		}
		return Token{Type: TOKEN_PLUS, Value: "+", Line: l.line}
	case '-':
		// Check for ->
		if l.peek() == '>' {
			l.pos += 2
			return Token{Type: TOKEN_ARROW, Value: "->", Line: l.line}
		}
		// Check for --
		if l.peek() == '-' {
			l.pos += 2
			return Token{Type: TOKEN_DECREMENT, Value: "--", Line: l.line}
		}
		// Check for -=
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_MINUS_EQUALS, Value: "-=", Line: l.line}
		}
		// Check for negative number literal
		if l.pos+1 < len(l.input) && unicode.IsDigit(rune(l.peek())) {
			start := l.pos
			l.pos++ // skip '-'
			for l.pos < len(l.input) && (unicode.IsDigit(rune(l.input[l.pos])) || l.input[l.pos] == '.') {
				l.pos++
			}
			return Token{Type: TOKEN_NUMBER, Value: l.input[start:l.pos], Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_MINUS, Value: "-", Line: l.line}
	case '*':
		l.pos++
		// Check for *+ (fused multiply-add)
		if l.pos < len(l.input) && l.input[l.pos] == '+' {
			l.pos++
			return Token{Type: TOKEN_FMA, Value: "*+", Line: l.line}
		}
		// Check for *=
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos++
			return Token{Type: TOKEN_STAR_EQUALS, Value: "*=", Line: l.line}
		}
		return Token{Type: TOKEN_STAR, Value: "*", Line: l.line}
	case '/':
		l.pos++
		// Check for /=
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos++
			return Token{Type: TOKEN_SLASH_EQUALS, Value: "/=", Line: l.line}
		}
		return Token{Type: TOKEN_SLASH, Value: "/", Line: l.line}
	case '%':
		l.pos++
		// Check for %=
		if l.pos < len(l.input) && l.input[l.pos] == '=' {
			l.pos++
			return Token{Type: TOKEN_MOD_EQUALS, Value: "%=", Line: l.line}
		}
		return Token{Type: TOKEN_MOD, Value: "%", Line: l.line}
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
		// Check for =>
		if l.peek() == '>' {
			l.pos += 2
			return Token{Type: TOKEN_FAT_ARROW, Value: "=>", Line: l.line}
		}
		// Check for ==
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_EQ, Value: "==", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_EQUALS, Value: "=", Line: l.line}
	case '<':
		// Check for <-, then <<b (rotate left), then <b (shift left), then <=, then <
		if l.peek() == '-' {
			l.pos += 2
			return Token{Type: TOKEN_LEFT_ARROW, Value: "<-", Line: l.line}
		}
		if l.peek() == '<' && l.pos+2 < len(l.input) && l.input[l.pos+2] == 'b' {
			l.pos += 3
			return Token{Type: TOKEN_LTLT_B, Value: "<<b", Line: l.line}
		}
		if l.peek() == 'b' {
			l.pos += 2
			return Token{Type: TOKEN_LT_B, Value: "<b", Line: l.line}
		}
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_LE, Value: "<=", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_LT, Value: "<", Line: l.line}
	case '>':
		// Check for >>b (rotate right), then >b (shift right), then >=, then >
		if l.peek() == '>' && l.pos+2 < len(l.input) && l.input[l.pos+2] == 'b' {
			l.pos += 3
			return Token{Type: TOKEN_GTGT_B, Value: ">>b", Line: l.line}
		}
		if l.peek() == 'b' {
			l.pos += 2
			return Token{Type: TOKEN_GT_B, Value: ">b", Line: l.line}
		}
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
		// Check for ~> first, then ~b
		if l.peek() == '>' {
			l.pos += 2
			return Token{Type: TOKEN_DEFAULT_ARROW, Value: "~>", Line: l.line}
		}
		if l.peek() == 'b' {
			l.pos += 2
			return Token{Type: TOKEN_TILDE_B, Value: "~b", Line: l.line}
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
	case ';':
		l.pos++
		return Token{Type: TOKEN_SEMICOLON, Value: ";", Line: l.line}
	case '.':
		// Check for ..< or ..=
		if l.pos+1 < len(l.input) && l.input[l.pos+1] == '.' {
			if l.pos+2 < len(l.input) {
				if l.input[l.pos+2] == '<' {
					// ..<
					l.pos += 3
					return Token{Type: TOKEN_DOTDOTLT, Value: "..<", Line: l.line}
				} else if l.input[l.pos+2] == '=' {
					// ..=
					l.pos += 3
					return Token{Type: TOKEN_DOTDOTEQ, Value: "..=", Line: l.line}
				}
			}
			// Just .. is an error - must be ..< or ..=
			// For now, treat as single .
		}
		// Single .
		l.pos++
		return Token{Type: TOKEN_DOT, Value: ".", Line: l.line}
	case '@':
		// Check for @first, @last, @-, @=
		if l.peek() >= 'a' && l.peek() <= 'z' {
			start := l.pos
			l.pos++ // skip @
			for l.pos < len(l.input) && ((l.input[l.pos] >= 'a' && l.input[l.pos] <= 'z') || (l.input[l.pos] >= 'A' && l.input[l.pos] <= 'Z')) {
				l.pos++
			}
			value := l.input[start:l.pos]
			if value == "@first" {
				return Token{Type: TOKEN_AT_FIRST, Value: value, Line: l.line}
			}
			if value == "@last" {
				return Token{Type: TOKEN_AT_LAST, Value: value, Line: l.line}
			}
			if value == "@counter" {
				return Token{Type: TOKEN_AT_COUNTER, Value: value, Line: l.line}
			}
			if value == "@key" {
				return Token{Type: TOKEN_AT_KEY, Value: value, Line: l.line}
			}
			// Unknown @identifier, treat as error or identifier
			l.pos = start + 1
			return Token{Type: TOKEN_AT, Value: "@", Line: l.line}
		}
		if l.peek() == '-' {
			l.pos += 2
			return Token{Type: TOKEN_AT_MINUS, Value: "@-", Line: l.line}
		}
		if l.peek() == '=' {
			l.pos += 2
			return Token{Type: TOKEN_AT_EQUALS, Value: "@=", Line: l.line}
		}
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
		// Check for ||| first, then ||, then |b, then |
		if l.peek() == '|' {
			if l.pos+2 < len(l.input) && l.input[l.pos+2] == '|' {
				l.pos += 3
				return Token{Type: TOKEN_PIPEPIPEPIPE, Value: "|||", Line: l.line}
			}
			l.pos += 2
			return Token{Type: TOKEN_PIPEPIPE, Value: "||", Line: l.line}
		}
		if l.peek() == 'b' {
			l.pos += 2
			return Token{Type: TOKEN_PIPE_B, Value: "|b", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_PIPE, Value: "|", Line: l.line}
	case '&':
		// Check for &b
		if l.peek() == 'b' {
			l.pos += 2
			return Token{Type: TOKEN_AMP_B, Value: "&b", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_AMP, Value: "&", Line: l.line}
	case '^':
		// Check for ^b
		if l.peek() == 'b' {
			l.pos += 2
			return Token{Type: TOKEN_CARET_B, Value: "^b", Line: l.line}
		}
		l.pos++
		return Token{Type: TOKEN_CARET, Value: "^", Line: l.line}
	case '#':
		l.pos++
		return Token{Type: TOKEN_HASH, Value: "#", Line: l.line}
	}

	return Token{Type: TOKEN_EOF, Line: l.line}
}
