# Flapc TODO

## Essential - Compiler Bugs (Highest Priority)

These bugs prevent proper testing and limit language usability:

- [ ] **Nested loops bug** - Outer loop terminates after first iteration
  - Affects: programs/nested_loop.flap, programs/ascii_art.flap, programs/test_for_break.flap
  - Current issue: Register save/restore and stack cleanup interaction
  - Impact: Major - prevents many real-world use cases

- [ ] **Lambda-returning-lambda (closures)** - Cause segfaults
  - Affects: programs/lambda_calculator.flap
  - Impact: High - prevents functional programming patterns

- [ ] **Match on numeric literals** - Doesn't work correctly
  - Affects: programs/match_unicode.flap
  - Workaround: Use nested if-else chains
  - Impact: Medium - limits pattern matching utility

- [ ] **Single-parameter lambdas with match** - Return wrong values
  - Affects: programs/factorial.flap
  - Workaround: Use two-parameter accumulator pattern
  - Impact: Medium - requires workarounds for common patterns

## Essential - Core Language Features

- [ ] Add an "alias" keyword for creating keyword aliases
  - Examples: `alias for=@`, `alias break=@-`, `alias continue=@=`
  - Enables language packs (python.flap, gdscript.flap, etc.)

- [ ] Add optional use of `:` followed by indentation (like Python) instead of `{}` blocks
  - Enables Python/GDScript-style syntax in language packs
  - Should be opt-in, not replacing existing brace syntax

## Important - Bug Fixes

- [ ] Fix the TODO issues mentioned in examples/*.flap

## Nice to Have - Advanced Features

- [ ] Add unsafe block with platform-agnostic register names
  - Use "a" for first register, "b" for next, etc.
  - Inspired by the Battlestar programming language
  - Combines all platforms into one assembly syntax

- [ ] Add approximate equality operator for float matching
  - Syntax: `0.3 =0.1= 0.2` (checks if 0.3 is within 0.2±0.1)
  - Useful for floating-point comparisons

- [ ] Add macro system for complex syntax transformations
  - Enables advanced language pack features
  - Pattern-based code transformation

- [ ] Add custom infix operator definitions
  - For language packs with different operator precedence
  - Example: Python's `**` for exponentiation

## Nice to Have - New Operators (After All Tests Pass)

These operators work in both safe and unsafe blocks:

- [ ] `++` (add with carry) - Multi-precision arithmetic
  - Safe mode: Automatic carry propagation between operations
  - Unsafe mode: Uses `adc` instruction (x86-64) for explicit carry handling
  - Syntax: `result <- high1 ++ high2`

- [ ] `<->` (swap/exchange) - Exchange two values
  - Safe mode: Swap variables or memory locations
  - Unsafe mode: Uses `xchg` instruction (x86-64) for atomic operations
  - Syntax: `a <-> b` or `rax <-> rbx` (unsafe)

**Not implementing** (obsolete, privileged, or redundant):
- `aad`, `aam` - Obsolete BCD instructions
- `cbw`, `cwd` - Sign extension (can use movsx instead)
- `in` - Port I/O (privileged, not useful in userspace)
- `int` - Software interrupt (already have syscall support)
- `stosw` - String operations (can use mov in loops)
- `xadd` - Atomic exchange-and-add (niche use case)

**Already implemented:**
- `add`, `imul`, `mov`, `or`, `pop`, `xor` - All have dedicated .go files
- Jump instructions (`jnp`, `jns`, `jnz`, `jp`, `jz`) - Handled in jmp.go
