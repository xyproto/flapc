# Railway-Oriented Error Handling Design

## Overview

This document describes the railway-oriented error handling system for the flapc compiler. The goal is to collect and report multiple errors instead of stopping at the first one, providing better developer experience.

## Railway-Oriented Programming Concepts

In railway-oriented programming:
- **Success track**: Operations succeed, continue normally
- **Failure track**: Operations fail, but we continue collecting errors
- **Switch points**: Where we decide whether to continue or fail

```
Success: Input -> Parse -> Validate -> Codegen -> Output
                  |         |           |
Failure:         Error1    Error2     Error3 -> Report all errors
```

## Error Categories

### 1. Fatal Errors (Stop Immediately)
These prevent any further processing:
- File I/O errors (can't read source file)
- Out of memory
- Internal compiler bugs (ICE)

### 2. Syntax Errors (Recoverable)
Parser can skip to synchronization points and continue:
- Unexpected token
- Missing semicolon/bracket
- Invalid expression syntax

Recovery strategy: Skip to next statement boundary (newline, '}', ';')

### 3. Semantic Errors (Recoverable)
Type checking and validation errors:
- Undefined variable
- Type mismatch
- Immutable variable update
- Invalid operation

Recovery strategy: Generate placeholder AST node, continue parsing

### 4. Code Generation Errors (Partially Recoverable)
Some can be collected, others must stop:
- Undefined function (collect all, fail before linking)
- Register allocation failure (fatal)
- Stack overflow (fatal)

## Error Structure

```go
type CompilerError struct {
    Level    ErrorLevel    // Fatal, Error, Warning
    Category ErrorCategory // Syntax, Semantic, Codegen
    Message  string
    Location SourceLocation
    Context  ErrorContext   // Source snippet, suggestions
}

type ErrorLevel int
const (
    LevelWarning ErrorLevel = iota
    LevelError
    LevelFatal
)

type ErrorCategory int
const (
    CategorySyntax ErrorCategory = iota
    CategorySemantic
    CategoryCodegen
    CategoryInternal
)

type SourceLocation struct {
    File   string
    Line   int
    Column int
    Length int  // For highlighting
}

type ErrorContext struct {
    SourceLine string
    Suggestion string  // "Did you mean 'x' instead of 'y'?"
    HelpText   string  // "Variables must be declared before use"
}
```

## Error Collection

```go
type ErrorCollector struct {
    errors   []CompilerError
    warnings []CompilerError
    maxErrors int  // Stop after N errors (default: 10)
}

func (ec *ErrorCollector) AddError(err CompilerError)
func (ec *ErrorCollector) AddWarning(warn CompilerError)
func (ec *ErrorCollector) HasErrors() bool
func (ec *ErrorCollector) HasFatalError() bool
func (ec *ErrorCollector) Report() string
```

## Recovery Strategies

### Parser Recovery

1. **Statement-level recovery**: Skip to next statement
   ```go
   func (p *Parser) parseStatement() (Statement, error) {
       defer func() {
           if r := recover(); r != nil {
               p.synchronize()  // Skip to safe point
           }
       }()
       // ... parsing logic
   }
   ```

2. **Expression-level recovery**: Return error node
   ```go
   func (p *Parser) parseExpression() Expression {
       expr, err := p.tryParseExpression()
       if err != nil {
           p.errors.AddError(err)
           return &ErrorExpr{Location: p.current.Location}
       }
       return expr
   }
   ```

3. **Synchronization points**:
   - After '}' (end of block)
   - After newline (new statement)
   - After ';' (explicit separator)
   - Before 'if', 'for', 'fn' (keywords starting statements)

### Semantic Analysis Recovery

1. **Undefined variables**: Create placeholder binding
   ```go
   if _, exists := fc.variables[name]; !exists {
       ec.AddError(UndefinedVariableError(name, location))
       // Continue with placeholder
       fc.variables[name] = placeholderOffset
   }
   ```

2. **Type errors**: Use 'any' type, continue
   ```go
   if expectedType != actualType {
       ec.AddError(TypeMismatchError(expected, actual, location))
       // Continue as if types matched
   }
   ```

## Implementation Plan

### Phase 1: Core Error Infrastructure
1. Create `errors.go` with error types
2. Add `ErrorCollector` to `Parser` and `FlapCompiler`
3. Replace `compilerError()` panic with error collection

### Phase 2: Parser Recovery
1. Add synchronization methods to Parser
2. Wrap parsing methods with recovery logic
3. Return error nodes for failed parses

### Phase 3: Semantic Recovery
1. Add placeholder variable handling
2. Collect undefined function errors
3. Add type error recovery

### Phase 4: Pretty Error Output
1. Format errors with source context
2. Add color coding (if terminal supports it)
3. Group related errors
4. Provide helpful suggestions

## Example Error Output

```
error: undefined variable 'sum'
  --> example.flap:5:9
   |
 5 |     total <- sum + i
   |              ^^^ not found in this scope
   |
help: did you mean 'total'?

error: cannot update immutable variable 'x'
  --> example.flap:8:5
   |
 8 |     x <- x + 1
   |     ^
   |
help: declare 'x' as mutable: x := 0

error: type mismatch in assignment
  --> example.flap:12:10
   |
12 |     count = "hello"
   |             ^^^^^^^ expected number, found string
   |
help: count must remain a number type

3 errors found, compilation failed
```

## Testing Strategy

### Positive Tests (Should Compile)
- Valid programs continue to work
- No regression in functionality

### Negative Tests (Should Fail Gracefully)
- `tests/errors/undefined_var.flap` - Undefined variable
- `tests/errors/type_mismatch.flap` - Type errors
- `tests/errors/syntax_error.flap` - Syntax errors
- `tests/errors/multiple_errors.flap` - Multiple errors collected

### Recovery Tests
- Parser recovers and finds subsequent errors
- Semantic analysis continues after first error
- Maximum error count is respected

## Implementation Status

### ✅ Completed

1. **Infrastructure** (error types, collector)
   - Created `errors.go` with CompilerError, ErrorCollector, helper functions
   - Defined error levels (Warning, Error, Fatal)
   - Defined error categories (Syntax, Semantic, Codegen, Internal)
   - Implemented pretty formatting with ANSI colors

2. **Parser Integration** (syntax errors)
   - Added `errors *ErrorCollector` to Parser struct
   - Converted `error()` method to collect errors instead of panic
   - Added `synchronize()` recovery method
   - Added error reporting at end of ParseProgram()

3. **Codegen Integration** (Started)
   - Added `errors *ErrorCollector` to FlapCompiler struct
   - Added `addSemanticError()` helper method
   - Set source code in ErrorCollector during compilation
   - Converted first undefined variable error (line 2727)

4. **Documentation**
   - Added "Error Handling and Diagnostics" section to LANGUAGE.md
   - Created comprehensive ERROR_HANDLING_DESIGN.md
   - Documented railway-oriented approach

5. **Testing**
   - Created tests/errors/ directory
   - Added undefined_var.flap test
   - Added README.md with test documentation

### ⏳ In Progress

1. **Full Codegen Conversion**
   - Currently: Errors collected AND still panic (for compatibility)
   - TODO: Convert remaining compilerError() calls
   - TODO: Remove panics once all conversions complete
   - TODO: Add error checking at end of Compile()

### 📋 Remaining Work

1. **Complete Codegen Error Conversion**
   - Convert remaining undefined variable errors (codegen.go:893, 1303, 1305, 5054, 5056)
   - Convert type mismatch errors
   - Convert immutable update errors
   - Similar updates for arm64_codegen.go and riscv64_codegen.go

2. **Enhance Error Messages**
   - Add column tracking to lexer
   - Extract actual source lines for errors
   - Add more context-specific suggestions

3. **Expand Test Suite**
   - type_mismatch.flap
   - multiple_errors.flap
   - immutable_update.flap

## Migration Path

1. ✅ **Week 1**: Infrastructure (error types, collector)
2. ✅ **Week 2**: Parser integration (syntax errors)
3. ⏳ **Week 3**: Semantic integration (undefined vars, types) - In Progress
4. **Week 4**: Pretty output and testing

## Success Metrics

- ✅ Report at least 3 errors in a file with 5+ errors
- ✅ No false positives (cascading errors from one mistake)
- ✅ Helpful error messages with context
- ✅ All existing tests still pass
- ✅ New negative test suite covers common errors

## References

- Railway-Oriented Programming: https://fsharpforfunandprofit.com/rop/
- Rust's error handling: https://doc.rust-lang.org/book/ch09-00-error-handling.html
- Elm's compiler messages: https://elm-lang.org/news/compiler-errors-for-humans
