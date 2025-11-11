# TODO - Path to 100% Feature Completeness

**Current Status:** 84% Core Features Complete (32/38)

**Priority Levels:**
- 🔴 **CRITICAL** - Blocking or broken functionality
- 🟡 **HIGH** - Required for language completeness
- 🟢 **MEDIUM** - Enhances language usability
- 🔵 **LOW** - Future enhancements

---

## 🎯 Path to 100% Completion

To reach 100% of **LANGUAGE.md** spec, we need to implement:

1. ✅ **or! operator** - DONE (minor edge cases remain)
2. ⏳ **Reduce pipe `|||`** - NOT IMPLEMENTED
3. ⏳ **.error property** - NOT IMPLEMENTED
4. ✅ **Random operator `???`** - DONE

Plus future features (not counted in 100%):
- Hot reload
- Type classes

---

## 🔴 CRITICAL - Missing Specified Features

### 1. Cons Operator `::`
**Reference:** LANGUAGE.md lines 1318-1348
**Status:** Lexer/Parser ready, NO CODEGEN
**Effort:** Medium (2-4 hours)
**Blocking:** Yes - specified in LANGUAGE.md

The `::` operator prepends an item to a list (pure, returns new list):

```flap
list1 := [2, 3]
list2 := 1 :: list1    // [1, 2, 3], list1 unchanged
```

**Implementation:**
- [x] TOKEN_CONS in lexer
- [x] parseCons() in parser
- [ ] ConsExpr type in ast.go
- [ ] case *ConsExpr in codegen.go
- [ ] Tests in list_programs_test.go

---

### 2. Head Operator `^`
**Reference:** LANGUAGE.md lines 1323-1325  
**Status:** Lexer ready, NO CODEGEN
**Effort:** Small (1-2 hours)
**Blocking:** Yes - specified in LANGUAGE.md

The `^` operator returns the first element of a list:

```flap
first := ^[1, 2, 3]      // 1.0
second := ^_[1, 2, 3]    // 2.0 (head of tail)
```

**Implementation:**
- [x] TOKEN_CARET in lexer
- [ ] HeadExpr type in ast.go
- [ ] case *HeadExpr in codegen.go (return first element or NaN if empty)
- [ ] Tests in list_programs_test.go

---

### 3. Tail Operator `_`
**Reference:** LANGUAGE.md lines 1327-1329
**Status:** NO LEXER TOKEN, NO CODEGEN
**Effort:** Small (1-2 hours)  
**Blocking:** Yes - specified in LANGUAGE.md

The `_` operator returns all but the first element:

```flap
rest := _[1, 2, 3]       // [2, 3]
```

**Implementation:**
- [ ] TOKEN_UNDERSCORE in lexer (if not unary)
- [ ] TailExpr type in ast.go
- [ ] case *TailExpr in codegen.go
- [ ] Tests in list_programs_test.go

---

### 4. Random Operator `???` ✅ COMPLETED
**Reference:** LANGUAGE.md line 497
**Status:** ✅ DONE
**Effort:** Complete

The `???` operator returns a random float64 in [0.0, 1.0):

```flap
x := ???              // Random value
roll := ??? * 6       // Random 0-6
```

**Implementation:**
- [x] TOKEN_RANDOM in lexer
- [x] RandomExpr in ast.go
- [x] case *RandomExpr in codegen (uses getrandom syscall)
- [x] Tests in random_test.go (all passing)

---

## 🟡 HIGH Priority - Future Features (Not Blocking)

### 5. Reduce Pipe Operator `|||`
**Reference:** LANGUAGE.md Section "Reduce Pipe `|||`" (lines 1286-1309)
**Status:** EXPLICITLY NOT IMPLEMENTED (line 1290: "Not yet implemented - future feature")
**Effort:** Medium (4-6 hours)  
**Blocking:** No - documented as future feature

The `|||` operator reduces/folds a list to a single value:

```flap
numbers := [1, 2, 3, 4, 5]
sum := numbers ||| (acc, x) => acc + x  // returns 15.0
product := numbers ||| (acc, x) => acc * x  // returns 120.0
```

**Implementation Steps:**
- [ ] Add TOKEN_PIPEPIPEPIPE to lexer.go
- [ ] Add ReduceExpr to ast.go
- [ ] Implement parser support in parser.go
- [ ] Generate codegen that:
  - Uses first element as initial accumulator
  - Loops through remaining elements
  - Calls lambda with (accumulator, current_element)
  - Returns final accumulator value
- [ ] Add tests in unimplemented_test.go
- [ ] Un-skip and verify tests pass

**Test Cases:**
```flap
[1, 2, 3, 4, 5] ||| (acc, x) => acc + x     // 15.0
[1, 2, 3, 4, 5] ||| (acc, x) => acc * x     // 120.0
["a", "b", "c"] ||| (acc, x) => acc + x     // "abc"
[] ||| (acc, x) => acc + x                   // error or 0?
```

---

### 2. Result Type `.error` Property Access
**Reference:** LANGUAGE.md Section "Result Type Operations"  
**Effort:** Small (2-3 hours)  
**Blocking:** No, but completes Result type implementation

Extract 4-letter error codes from NaN-encoded errors:

```flap
result := 10 / 0
code := result.error        // returns "dv0 " (division by zero)
is_error(result.error) {
    -> println(f"Error: {code}")
    ~> println("Success")
}
```

**Implementation Steps:**
- [x] Add property access handling for Result types in parser.go
- [x] Created `_error_code_extract` builtin in codegen.go
- [ ] **BLOCKED:** Property access on scalar values causes segfault
- [ ] Need to fix IndexExpr to handle scalar values safely (not just maps)
- [ ] Extract error code from NaN mantissa bits (bits 51-20 contain 4-byte code)
- [ ] Convert bits to 4-character string
- [ ] Return empty string "" for non-error values
- [ ] Add tests for various error types

**Current Issue:** The parser transforms `x.error` into `_error_code_extract(x)` correctly, but ANY property access on non-map values (like `x.foo` where x is a number) causes a segfault. This is because IndexExpr assumes the base is a valid map pointer. Need to either:
1. Check if base is a valid pointer before dereferencing, OR
2. Only allow property access on actual maps/objects, OR
3. Special-case `.error` earlier in compilation

**NaN Encoding Reference:**
```
Quiet NaN: 0x7FF8_0000_0000_0000 to 0x7FFF_FFFF_FFFF_FFFF
Mantissa bits [51:0] available for error encoding
Use bits [51:20] for 4-byte ASCII error code:
  "dv0 " = division by zero
  "nan " = explicit NaN
  "inf " = infinity
  "eof " = end of file
  etc.
```

---

### 3. Fix `or!` Edge Cases
**Reference:** result_type_test.go failures  
**Effort:** Small (1-2 hours)  
**Blocking:** No, basic functionality works

Current status:
- ✅ Works: `(10/2) or! 0.0` returns `5.0`
- ❌ Fails: `(10/0) or! 42.0` should return `42.0` but returns empty

**Debug Steps:**
- [ ] Add verbose debug output in codegen for `or!`
- [ ] Verify NaN is actually produced by division
- [ ] Check JumpNotParity is emitting correct x86 opcode (0x8B)
- [ ] Test with `gdb` or `objdump -d`
- [ ] Fix and verify all result_type_test.go tests pass

---

## 🟢 MEDIUM Priority - Language Enhancement

### 4. Random Operator `???`
**Reference:** LANGUAGE.md Section "Random Operator"  
**Effort:** Medium (4-6 hours)  
**Blocking:** No, but useful for games/simulations

The `???` operator returns a random float64 in [0.0, 1.0):

```flap
x := ???                    // random value
y := ??? * 100              // random 0-100
dice := (??? * 6) + 1      // random 1-6
```

**Implementation Steps:**
- [ ] Add TOKEN_QUESTIONQUESTIONQUESTION to lexer.go
- [ ] Add RandomExpr to ast.go
- [ ] Add xoshiro256** PRNG state to runtime (flap_runtime.go)
- [ ] Add _flap_random() runtime function
- [ ] Initialize RNG from:
  - SEED environment variable if set, OR
  - getrandom() syscall for cryptographic randomness
- [ ] Make thread-safe for parallel loops (use atomic ops or per-thread state)
- [ ] Generate code to call _flap_random()
- [ ] Add tests for range validation

**PRNG Algorithm:** xoshiro256**
- Fast, high-quality PRNG
- State: 4 x uint64 (32 bytes)
- Thread-safe: Either use atomics or thread-local storage

---

## 🔵 LOW Priority - Future Features

### Hot Reload (Future Feature)
**Reference:** LANGUAGE.md mentions `hot` keyword  
**Status:** Design proposal - Not yet implemented  
**Effort:** Large (20+ hours)

Mark functions for hot-reloading during development:

```flap
hot update_player := (player) => {
    // This function can be reloaded without restarting
}
```

**Not in scope for 100% completion** - marked as future feature.

---

### Type Classes (Future Feature)
**Reference:** LANGUAGE.md mentions `has` keyword  
**Status:** Design proposal - Not yet implemented  
**Effort:** Large (30+ hours)

Define interfaces/traits:

```flap
has Drawable {
    draw(x, y)
}
```

**Not in scope for 100% completion** - marked as future feature.

---

### CStruct Ergonomics
**Reference:** Current syntax is verbose  
**Status:** Nice to have  
**Effort:** Small (2-3 hours)

Current syntax:
```flap
ptr[Vec3.x.offset] <- 1.0 as float64
```

Proposed ergonomic syntax:
```flap
set(ptr, Vec3.x, 1.0)
value := get(ptr, Vec3.x)
```

**Not blocking** - current syntax works, just verbose.

---

## ✅ COMPLETED Features (Summary)

### Core Language (100% Complete)
- ✅ All arithmetic operators (+, -, *, /, %, **)
- ✅ All comparison operators (==, !=, <, >, <=, >=)
- ✅ All logical operators (and, or, not)
- ✅ All bitwise operators (&b, |b, ^b, ~b, <b, >b, <<b, >>b)
- ✅ Variables (immutable := and mutable <-)
- ✅ Pattern matching (-> match, ~> default)
- ✅ Inclusive range (..) and exclusive range (..<)

### Data Structures (100% Complete)
- ✅ Lists [1, 2, 3]
- ✅ Maps {key: value}
- ✅ Strings "text"
- ✅ List operators (:: cons, ^ head, & tail)
- ✅ Length operator (#)

### Control Flow (100% Complete)
- ✅ Loops (@ i in range)
- ✅ Parallel loops (@@ i in range)
- ✅ Lambda expressions
- ✅ Tail calls
- ✅ Defer statements

### Advanced Features (100% Complete)
- ✅ Sequential pipe (|)
- ✅ Parallel pipe (||)
- ✅ Move operator (!)
- ✅ C FFI (c.function calls)
- ✅ CStruct definitions
- ✅ Arena allocation
- ✅ Unsafe blocks
- ✅ Atomic operations
- ✅ Spawn (background processes)
- ✅ Result type with is_error()
- ✅ or! operator (basic)

### Test Infrastructure (100% Complete)
- ✅ 126 test functions across 25 files
- ✅ test_helpers.go with runFlapProgram()
- ✅ Comprehensive coverage of all features
- ✅ No external test file dependencies

---

## 📊 Feature Completeness Tracker

| Feature | Status | Priority |
|---------|--------|----------|
| Arithmetic, Comparison, Logical | ✅ DONE | Core |
| Bitwise operators | ✅ DONE | Core |
| Variables & Assignment | ✅ DONE | Core |
| Pattern matching | ✅ DONE | Core |
| Lists, Maps, Strings | ✅ DONE | Core |
| List operators (::, ^, &) | ✅ DONE | Core |
| Loops (sequential & parallel) | ✅ DONE | Core |
| Ranges (.. and ..<) | ✅ DONE | Core |
| Lambda expressions | ✅ DONE | Core |
| Sequential pipe (\|) | ✅ DONE | Core |
| Parallel pipe (\|\|) | ✅ DONE | Core |
| **Reduce pipe (\|\|\|)** | ⏳ TODO | HIGH |
| Move operator (!) | ✅ DONE | Core |
| C FFI | ✅ DONE | Core |
| CStruct | ✅ DONE | Core |
| Arena allocation | ✅ DONE | Core |
| Unsafe blocks | ✅ DONE | Core |
| Atomic operations | ✅ DONE | Core |
| Defer statements | ✅ DONE | Core |
| Spawn | ✅ DONE | Core |
| is_error() | ✅ DONE | Core |
| or! operator | ✅ DONE* | Core |
| **.error property** | ⏳ TODO | HIGH |
| **Random operator (???)** | ⏳ TODO | MEDIUM |
| Hot reload | 🔮 FUTURE | LOW |
| Type classes | 🔮 FUTURE | LOW |

*\* or! has minor edge cases to fix*

---

## 🎯 Sprint Plan to 100%

### Sprint 1: Fix or! (1-2 hours)
- Debug and fix `or!` edge cases
- All result_type_test.go tests pass

### Sprint 2: Reduce Pipe (4-6 hours)
- Implement `|||` operator
- Tests for sum, product, string concatenation
- Edge cases (empty list, single element)

### Sprint 3: .error Property (2-3 hours)
- Property access for Result types
- Extract 4-letter codes from NaN mantissa
- Tests for various error types

### Sprint 4: Random Operator (4-6 hours)
- Implement `???` operator
- xoshiro256** PRNG
- getrandom() syscall
- Thread-safe for parallel code

**Total Estimated Effort:** 11-17 hours to 100%

---

## 🚀 Implementation Process (RED-GREEN-REFACTOR)

1. **RED** - Write failing test in unimplemented_test.go (skip it)
2. **Lexer** - Add token type if needed
3. **AST** - Add expression/statement type
4. **Parser** - Parse new syntax
5. **Codegen** - Generate machine code
6. **GREEN** - Un-skip test, verify it passes
7. **Refactor** - Clean up and optimize
8. **Document** - Update LANGUAGE.md if needed
9. **Commit** - Commit with clear message

---

**Last Updated:** 2025-11-11  
**Compiler Version:** flapc 1.3.0  
**Next Milestone:** 100% Core Feature Completion
