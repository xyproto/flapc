# TODO - Implementation Tasks

**Priority Levels:**
- 🔴 **CRITICAL** - Blocking functionality
- 🟡 **HIGH** - Important for language completeness
- 🟢 **MEDIUM** - Nice to have
- 🔵 **LOW** - Future enhancements

---

## 🟡 HIGH Priority

### Result Type - or! operator
**Reference:** LANGUAGE.md Section "Result Type Operations"

- [ ] Edge case: Complex `or!` expressions need debugging

### Reduce Pipe Operator `|||`
**Reference:** LANGUAGE.md Section "Reduce Pipe"

- [ ] Add TOKEN_PIPEPIPEPIPE to lexer
- [ ] Add ReduceExpr to AST
- [ ] Implement parser support
- [ ] Generate reduce/fold codegen with accumulator
- [ ] Test: `[1, 2, 3, 4, 5] ||| (acc, x) => acc + x` returns `15.0`

### Result Type - .error property
**Reference:** LANGUAGE.md Section "Result Type Operations"

- [ ] Add property access for Result types
- [ ] Extract 4-letter error code from NaN/Inf bits
- [ ] Return as Flap string
- [ ] Test: `(10/0).error` returns `"dv0 "`

---

## 🟢 MEDIUM Priority

### Random Operator `???`
**Reference:** LANGUAGE.md Section "Random Operator"

- [ ] Add TOKEN_QUESTIONQUESTIONQUESTION to lexer
- [ ] Add random expression to AST
- [ ] Implement xoshiro256** PRNG in runtime
- [ ] Add getrandom() syscall wrapper
- [ ] Initialize from SEED env var or system entropy
- [ ] Generate code to call _flap_random()
- [ ] Make thread-safe for parallel code
- [ ] Test: `???` returns values in [0.0, 1.0)

---

## 🔵 LOW Priority

### CStruct Ergonomics
**Reference:** LANGUAGE.md Section "CStruct"

- [ ] Track field types in CStruct

---

## Implementation Process

1. Write RED test in unimplemented_test.go (skip it)
2. Implement lexer changes
3. Update AST
4. Implement parser
5. Implement codegen
6. Un-skip test - it should pass (GREEN)
7. Refactor if needed
8. Update TODO.md
9. Commit changes
