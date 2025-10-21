# Flapc TODO

## Current Work

- [ ] **SIMD intrinsics** - Vector operations
  - [ ] Vector arithmetic operations (vadd, vsub, vmul, vdiv) - needs debugging
  - [ ] Vector component access (v.x, v.y, v.z, v.w or v[0], v[1], etc.)
  - [ ] Dot product, magnitude, normalize operations
- [ ] **Register allocation improvements**
  - [ ] Register allocation for frequently-used variables
  - [ ] Full register allocator with liveness analysis

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
  - Impact: High - prevents functional programming patterns
- [ ] **Assignment in match clause results** - Parser rejects
  - Affects: programs/prime_sieve.flap
  - Impact: Medium - limits expressiveness

## Test Coverage Improvements

- [ ] **Match expression tests** - Only 2 tests for major language feature
- [ ] **Mutable variable tests** - Only 2 tests, critical for state management
- [ ] **Edge case tests** - Empty collections, min/max values, error paths
