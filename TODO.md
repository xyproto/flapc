# Flapc TODO

## Essential - Compiler Bugs (Highest Priority)

These bugs prevent proper testing and limit language usability:

- [x] **Nested loops bug** - FIXED ✓
  - Fixed by moving from register-based to stack-based storage for loop counters
  - All nested loop tests now pass: test_nested_var_bounds, test_simple_nested, test_nested_trace, etc.
  - Impact: Major bug resolved - nested loops now work correctly

- [ ] **Lambda-returning-lambda (closures)** - Compilation error "undefined variable"
  - Root cause: Inner lambdas can't access outer lambda parameters (no closure capture)
  - Affects: Any lambda that returns another lambda
  - Impact: High - prevents functional programming patterns
  - Status: Requires implementing closure capture mechanism

- [ ] **Match expressions with string results** - Runtime crash ("Error")
  - Root cause: String results in match clauses cause segfault at runtime
  - Affects: Any match expression returning strings (programs/match_unicode.flap)
  - Workaround: Use numeric codes or if-else chains
  - Impact: High - severely limits match expression utility
  - Status: Bug identified, needs investigation of match clause compilation

- [ ] **Recursive lambdas** - Multiple related issues:
  - Issue 1: Can't call lambda by its own name (symbol lookup error)
    - Workaround: Use `me` keyword for self-reference
  - Issue 2: Non-tail recursion with `me` returns wrong values
    - Root cause: Tail call optimization applied to non-tail-recursive calls
    - Example: `me(n-1) * n` treats recursive call as tail call
    - Workaround: Use accumulator pattern for tail recursion
  - Affects: programs/factorial.flap (non-tail-recursive version)
  - Impact: Medium - requires specific recursion patterns

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
