# Flapc TODO

## Essential - Compiler Bugs (Highest Priority)

These bugs prevent proper testing and limit language usability:

- [~] **Loop-local variables and stack management** - PARTIALLY FIXED
  - **Fixed**: Simple loops with loop-local variables (fibonacci, basic use cases)
  - **Fixed**: Proper runtime stack tracking to prevent unbounded stack growth
  - **Fixed**: Loop state offset calculation using pre-body stack position
  - **Remaining issue**: Nested loops with loop-local variables have incorrect behavior
  - **Implementation**: Added runtime stack tracking (`fc.runtimeStack`) separate from logical offsets
  - **Implementation**: Added `loopBaseOffsets` map to track stack position before each loop body
  - **Impact**: Critical improvement - fixes infinite loops and stack corruption in simple cases
  - **Next step**: Debug nested loop interaction with loop-local variables

- [x] **Nested loops bug** - FIXED ✓
  - Fixed using pure stack-based storage (following C/Go approach)
  - All loop counters and limits stored on stack, avoiding register conflicts
  - Working tests: test_nested_var_bounds, test_nested_simple, ascii_art, test_nested_loops_i
  - Works correctly for all nesting depths with simple variable access
  - Known limitation: 3+ level nesting with complex arithmetic expressions (i*100+j*10+k) shows incorrect output
    - Simple arithmetic (i+j+k) works fine
    - Individual variable access works fine
    - Issue appears to be with expression evaluation, not loop structure
  - Impact: Major improvement - covers 99% of real-world nested loop use cases

- [ ] **Lambda-returning-lambda (closures)** - Compilation error "undefined variable"
  - Root cause: Inner lambdas can't access outer lambda parameters (no closure capture)
  - Affects: Any lambda that returns another lambda
  - Impact: High - prevents functional programming patterns
  - Status: Requires implementing closure capture mechanism

- [x] **String variables with printf %s** - FIXED ✓
  - Root cause: Stack misalignment before malloc call in `flap_string_to_cstr`
  - Fix: Removed premature stack restoration (line 7520) to keep 16-byte alignment for malloc
  - Working tests: test_string_print_var.flap, test_match_string_test.flap, programs/match_unicode.flap
  - Impact: Fixed string variables in printf, match expressions with strings now work

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
