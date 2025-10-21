# Flapc TODO - x86_64 Focus

## Current Sprint (x86_64 only)
- [ ] **SIMD intrinsics** - Vector operations for audio DSP, graphics effects, particle systems
  - [x] vec2() and vec4() constructors
  - [x] VectorExpr AST node and parser support
  - [x] SIMD instruction wrappers (movupd, addpd, subpd, mulpd, divpd)
  - [ ] Vector arithmetic operations (vadd, vsub, vmul, vdiv) - needs debugging
  - [ ] Vector component access (v.x, v.y, v.z, v.w or v[0], v[1], etc.)
  - [ ] Dot product, magnitude, normalize operations
- [ ] **Register allocation improvements** - Better register usage for performance
  - [x] Binary operation optimization (use xmm2 instead of stack spills)
  - [x] Direct register-to-register moves (movq xmm, rax)
  - [x] Keep loop counters in registers (r12/r13 for range loops)
  - [ ] Register allocation for frequently-used variables
  - [ ] Full register allocator with liveness analysis
- [x] **Dead code elimination** - Remove unused variables and unreachable code
- [x] **Constant propagation** - Substitute immutable variables with their constant values
- [x] **Function inlining** - Auto-inline small functions for performance
- [ ] **Pure function memoization** - Cache results of pure functions (future)

## Compiler Bugs (High Priority)

These bugs are preventing proper testing and limit language usability:

- [ ] **Nested loops bug** - Outer loop terminates after first iteration
  - Affects: programs/nested_loop.flap, programs/ascii_art.flap, programs/test_for_break.flap
  - Workaround: Use explicit output or flatten loops
  - Impact: Major - prevents many real-world use cases
- [ ] **Match on numeric literals** - Doesn't work correctly
  - Affects: programs/match_unicode.flap
  - Workaround: Use nested if-else chains
  - Impact: Medium - limits pattern matching utility
- [ ] **Single-parameter lambdas with match** - Return wrong values
  - Affects: programs/factorial.flap
  - Workaround: Use two-parameter accumulator pattern
  - Impact: Medium - requires workarounds for common patterns
- [ ] **Lambda-returning-lambda (closures)** - Cause segfaults
  - Affects: programs/lambda_calculator.flap
  - Workaround: Manual composition instead of higher-order functions
  - Impact: High - prevents functional programming patterns
- [ ] **`ret @2` in nested loops** - Causes infinite loops
  - Affects: programs/nested_break_test.flap
  - Workaround: Use explicit control flow
  - Impact: Low - workarounds available
- [ ] **Assignment in match clause results** - Parser rejects
  - Affects: programs/prime_sieve.flap
  - Workaround: Restructure code or use explicit output
  - Impact: Medium - limits expressiveness
- [ ] **Mutable variable updates in loops** - `=` doesn't work, must use `<-`
  - Affects: programs/match_unicode.flap
  - Workaround: Use `<-` operator inside loops
  - Impact: Low - documented workaround exists

## Test Coverage Improvements

Current coverage: 210/221 programs (95.0%) with result files

### High Priority
- [ ] **Match expression tests** - Only 2 tests for major language feature
  - Add tests for: multiple conditions, nested matches, match with lists/maps
- [ ] **Mutable variable tests** - Only 2 tests, critical for state management
  - Add tests for: shadowing, scope rules, concurrent updates
- [ ] **Compound assignment operators** - Only 1 test for `+=`, `-=`, `*=`, `/=`, `%=`
  - Add comprehensive test for all operators with edge cases

### Medium Priority
- [ ] **Edge case tests** - Missing boundary condition testing
  - Empty collections with all operations
  - Min/max values for numeric types
  - Error handling paths
- [ ] **Unicode/internationalization** - Only match_unicode test currently
  - Add tests for: non-ASCII identifiers, emoji in strings, UTF-8 edge cases
- [ ] **Tail recursion edge cases** - More stack overflow prevention tests
  - Deep recursion limits, mutual recursion

### Low Priority
- [ ] **Performance benchmarks** - No current performance tests
  - Add benchmarks for: SIMD ops, hash maps, string operations
- [ ] **Memory leak tests** - Arena allocator validation
  - Add tests for: arena cleanup, large allocations, fragmentation

## Future stdlib (architecture-agnostic)

- [ ] **Collections** - Hash map, tree, queue, stack
- [ ] **String manipulation** - split, join, replace, regex
- [ ] **File I/O library** - High-level wrappers for file operations
- [ ] **Network programming** - Sockets, HTTP
- [ ] **JSON parsing and serialization** - Configuration and data exchange
- [ ] **Date/time library** - Timing and scheduling utilities
