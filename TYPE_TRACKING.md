# Flapc Type Tracking System

## Overview

While Flap has only one runtime type (`map[uint64]float64`), the **Flapc compiler** must track semantic types during compilation for optimization, validation, and correct code generation.

## Philosophy

**At runtime:** Everything IS `map[uint64]float64` (no exceptions)

**At compile-time:** Track semantic meaning for:
- Register allocation optimization
- Efficient machine code generation
- Type-aware error checking
- C FFI interop
- Result type handling

## Semantic Type System

### FlapType Enum

```go
type FlapType int

const (
    // Flap native types (all are map[uint64]float64 at runtime)
    TypeUnknown    FlapType = iota  // Not yet inferred
    TypeNumber                       // Map with single entry {0: value}
    TypeString                       // Map {0: char0, 1: char1, ...}
    TypeList                         // Map {0: elem0, 1: elem1, ...}
    TypeMap                          // Map {key: value, ...}
    TypeAddress                      // Map {0: address_value} (reference to map)
    TypeLambda                       // Map {0: function_ptr}
    TypeResult                       // Map with potential error encoding
    
    // C FFI types (still map[uint64]float64, but with C semantics)
    TypeCInt8                        // C int8_t
    TypeCInt16                       // C int16_t
    TypeCInt32                       // C int32_t
    TypeCInt64                       // C int64_t
    TypeCUInt8                       // C uint8_t
    TypeCUInt16                      // C uint16_t
    TypeCUInt32                      // C uint32_t
    TypeCUInt64                      // C uint64_t
    TypeCFloat32                     // C float
    TypeCFloat64                     // C double
    TypeCString                      // C char* (null-terminated)
    TypeCPointer                     // C void*
)
```

### Type Properties

```go
type TypeInfo struct {
    Type        FlapType
    IsError     bool        // For Result type: contains error code
    CStructType string      // For C structs: struct name
    ElementType *TypeInfo   // For lists: element type hint
}
```

## Result Type Semantics

The Result type is special:

### Encoding

A Result is `map[uint64]float64` where:
- **Success:** Contains actual value (number, string, list, map)
- **Error:** Contains error code string in special encoding

### Error Detection

```go
// At runtime, check if value is error:
// 1. Try to interpret as pointer (address > 0x1000)
//    - If valid pointer: SUCCESS (contains actual value)
// 2. If not pointer, interpret as error code string:
//    - Extract 4-character error code from the map
//    - Examples: "dv0 " (division by zero), "idx " (index out of bounds)
```

### Error Codes (4 chars, space-padded)

```
"dv0 " - Division by zero
"idx " - Index out of bounds  
"key " - Key not found
"typ " - Type mismatch
"nil " - Null pointer
"mem " - Out of memory
"arg " - Invalid argument
"io  " - I/O error
"net " - Network error
"prs " - Parse error
"   " - Empty/no error (success)
```

### .error Property

When `.error` is accessed:

1. Check if value contains error encoding
2. If error: extract 4-char code, strip trailing spaces, return as string
3. If success: return empty string `""`

```flap
result := risky_operation()

// Check error
result.error {
    "" -> println("Success!")
    ~> println("Error:", result.error)  // Prints error code
}
```

### or! Operator

The `or!` operator checks for errors:

```flap
x := 10 / 0        // Returns error-encoded Result
safe := x or! 99   // Evaluates to 99 (error case)

y := 10 / 2        // Returns success Result with value 5
safe2 := y or! 99  // Evaluates to 5 (success case)
```

Compilation:
1. Evaluate left side
2. Check if result is error (see Error Detection above)
3. If error: evaluate and return right side
4. If success: return left side value

## Type Inference Rules

### Assignment
```flap
x = 42           // TypeNumber
y = "hello"      // TypeString
z = [1, 2, 3]    // TypeList
m = {0: "a"}     // TypeMap
```

### Operations
```flap
a + b            // If both TypeNumber -> TypeNumber, else TypeResult
s + t            // If both TypeString -> TypeString (concat)
x / 0            // TypeResult (may contain error)
```

### C FFI
```flap
c:func() as i32  // TypeCInt32
c:ptr() as ptr   // TypeCPointer
```

### Function Returns
```flap
f = x => x + 1   // Returns TypeNumber (if x is TypeNumber)
g = x => x / y   // Returns TypeResult (division may fail)
```

## Implementation Strategy

### Phase 1: Add TypeInfo to AST

```go
// In ast.go
type Expression interface {
    Node
    expressionNode()
    TypeHint() *TypeInfo  // NEW: Type inference hint
}

// Add to all expression types
type NumberExpr struct {
    Value    float64
    typeInfo *TypeInfo  // NEW
}

func (n *NumberExpr) TypeHint() *TypeInfo {
    if n.typeInfo == nil {
        n.typeInfo = &TypeInfo{Type: TypeNumber}
    }
    return n.typeInfo
}
```

### Phase 2: Type Inference Pass

```go
// In parser.go or new type_checker.go
func InferTypes(prog *Program) error {
    // Walk AST and infer types
    // Propagate type information
    // Mark Result types where operations can fail
    return nil
}
```

### Phase 3: Code Generation

```go
// In x86_64_codegen.go
func (b *X86_64Backend) compileExpression(expr Expression) error {
    typeInfo := expr.TypeHint()
    
    switch typeInfo.Type {
    case TypeNumber:
        // Optimize: keep in register if possible
        // Generate scalar arithmetic
    case TypeResult:
        // Generate error-checking code
        // Emit conditional branches for or!
    case TypeCInt32:
        // Generate C-compatible int32 code
        // Use 32-bit registers (eax, edi, etc.)
    }
}
```

### Phase 4: Result Type Runtime

```go
// In flap_runtime.go
func IsError(value MapValue) bool {
    // Check if value is error-encoded
    // Returns true if contains error code
}

func ExtractError(value MapValue) string {
    // Extract 4-char error code
    // Strip trailing spaces
    // Return as string
}

func EncodeError(code string) MapValue {
    // Encode 4-char error code into map
    // Ensure it's detectable as error
}
```

## Memory Layout Examples

### Number
```
Runtime: map[uint64]float64{0: 42.0}
Compiler tracks: TypeNumber
Register: XMM0 or RAX (optimized)
```

### String "hi"
```
Runtime: map[uint64]float64{0: 104.0, 1: 105.0}  // 'h'=104, 'i'=105
Compiler tracks: TypeString
Memory: Pointer to map structure
```

### Result (Success)
```
Runtime: map[uint64]float64{0: 5.0}  // Actual value
Compiler tracks: TypeResult, IsError=false
Check: value treated as pointer → valid → success
```

### Result (Error "dv0 ")
```
Runtime: map[uint64]float64{0: 100.0, 1: 118.0, 2: 48.0, 3: 32.0}  // "dv0 "
Compiler tracks: TypeResult, IsError=true
Check: value not valid pointer → extract chars → "dv0"
```

## Benefits

1. **Optimization:** Scalar numbers stay in registers
2. **Safety:** Detect type mismatches at compile time
3. **Clarity:** Generated code matches semantic intent
4. **C FFI:** Proper marshalling between Flap and C
5. **Debugging:** Better error messages with type info
6. **Result handling:** Efficient error checking without exceptions

## Note

This is **compile-time tracking only**. At runtime, everything remains `map[uint64]float64`. The type system exists solely to generate better machine code while respecting Flap's radical simplification philosophy.
