# Flap Compiler TODO

## Current Status

**x86-64 Linux**: 186/188 tests (99%)
**ARM64 macOS**: ~149/182 tests (82%) estimated

### Recent ARM64 Improvements
- Fixed os.Exit bug (66 replacements) - enables proper test execution
- Implemented ParallelExpr (|| operator) - +21 tests
- Added type tracking system (varTypes, getExprType) - foundation for concat
- Fixed lambda parameter storage bug - should fix ~20 lambda crashes
- Implemented me keyword for tail recursion - +4 tests

## High Priority - ARM64 Completion

### Critical Issue - Compilation Hang
- [ ] **BLOCKING**: ARM64 flapc binary hangs immediately on execution (even `--version`)
  - Hang occurs before main() debug prints execute
  - Affects all ARM64 compilation attempts
  - Not caused by runtime helper functions (persists when disabled)
  - Not caused by BinaryExpr changes (persists when reverted)
  - Process enters sleep state with minimal memory usage
  - Likely a macOS-specific issue (permissions, signing, or dyld)
  - Requires investigation with proper macOS debugging tools
  - **Action needed**: Test on different macOS system or with SIP disabled

### Implement Missing Runtime Functions (Blocked by hang)
- [x] List concatenation runtime function `_flap_list_concat(left, right)` - **IMPLEMENTED**
- [x] String concatenation runtime function `_flap_string_concat(left, right)` - **IMPLEMENTED**
- [ ] Float-to-string conversion for `str()` function
- [ ] Debug and fix math function compilation hanging issue

### Implement Missing Features
- [ ] SliceExpr: List/string slicing `list[start:end:step]`
- [ ] PipeExpr: Pipe operator `|`
- [ ] JumpExpr: Loop break/continue (`ret @N`, `@N`)

## Medium Priority - Language Features

### F-String Interpolation
- [ ] Implement Python-style f-strings: `f"Hello, {name}!"`
- [ ] Add lexer support for f-string tokens
- [ ] Add parser support for compile-time interpolation

### Performance
- [ ] Optimize CString conversion from O(n²) to O(n) in `parser.go`

## Low Priority - Additional Backends

### RISC-V Support
- [ ] Implement RISC-V register allocation
- [ ] Implement RISC-V calling convention
- [ ] Add floating-point instructions
- [ ] Fix PC-relative load for rodata symbols

## Standard Library Extensions

### String Functions
- [ ] `num(string)` - Parse string to float64
- [ ] `split(string, delimiter)`
- [ ] `join(list, delimiter)`
- [ ] `upper/lower/trim(string)`

### Collection Functions
- [ ] `map(f, list)`
- [ ] `filter(f, list)`
- [ ] `reduce(f, list, init)`
- [ ] `keys/values(map)`
- [ ] `sort(list)`
