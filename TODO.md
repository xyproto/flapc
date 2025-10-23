# Flapc TODO

## Bugs

- [ ] Fix UTF-8 rune rendering issues.
- [ ] Fix the TODO issues mentioned in examples/*.flap.
- [ ] Do not use external tools or packages or cgo.
- [ ] Use Go code instead of the "file" utility.

## Features

- [ ] Add an unsafe block that combines all platforms, and uses "a" for the first register, "b" for the next register etc.
      Inspired by the Battlestar programming language.
- [ ] Add an "alias" keyword, for creating aliases for keywords, such as "alias for=@" etc.
- [ ] Add aliases for: for, break, continue, return, mod and in. (Use ":" instead of "in" in the language).

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

## Mnemonics

Add support for emitting these mnemonics, one .go file per mnemonic:

aad
aam
adc
add
cbw
cwd
imul
in
int
jnp
jns
jnz
jp
jz
mov
or
pop
stosw
xadd
xchg
xor
