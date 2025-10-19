# Flap Compiler TODO

## Current Status

**x86-64 Linux**: 186/188 tests (99%)
**ARM64 macOS**: ~125/182 tests (69%) estimated

## High Priority - ARM64 Completion

### Implement Missing Runtime Functions
- [ ] List concatenation runtime function `_flap_list_concat(left, right)`
- [ ] String concatenation runtime function `_flap_string_concat(left, right)`
- [ ] Float-to-string conversion for `str()` function
- [ ] Debug and fix math function compilation hanging issue

### Implement Missing Features
- [ ] SliceExpr: List/string slicing `list[start:end:step]`
- [ ] PipeExpr: Pipe operator `|`
- [ ] JumpExpr: Loop break/continue (`ret @N`, `@N`)
- [ ] `me` keyword for tail-recursive lambda calls
- [ ] Debug lambda crashes (~20 tests failing)

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
