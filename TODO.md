# Flap Compiler TODO

## 1.0.0 Release Criteria

Release when:
- All blockers complete
- Tests passing on x86-64 and ARM64
- README.md, LANGUAGE.md, TODO.md accurate

---

## Blockers for 1.0.0

### String Operations
- [ ] Test: Runtime string concatenation `s1 + s2` where s1, s2 are variables
- [ ] Test: String comparison `s1 == s2`, `s1 != s2`
- [ ] Test: String comparison `s1 < s2`, `s1 > s2` (lexicographic)
- [ ] Test: String slicing `s[1:3]` returns substring
- [ ] Test: String length `#s` returns character count
- [ ] Optimize: CString conversion from O(n²) to O(n)
- [ ] Test: Strings > 255 characters (multi-byte length encoding)

### Polymorphic Operators
- [ ] Test: `[1, 2] + [3, 4]` returns `[1, 2, 3, 4]`
- [ ] Test: `{1: 10} + {2: 20}` returns `{1: 10, 2: 20}`
- [ ] Test: `list ++ 42` appends single value
- [ ] Test: `map ++ {key: value}` adds single entry
- [ ] Test: `x++` increments number by 1.0
- [ ] Test: `list--` removes last element
- [ ] Test: `map--` removes last entry
- [ ] Test: `x--` decrements number by 1.0
- [ ] Test: `s1 - s2` removes characters (string difference)
- [ ] Test: `list1 - list2` removes elements (set difference)
- [ ] Test: `map1 - map2` removes keys (set difference)

### Control Flow
- [ ] Test: `return expr` exits function early
- [ ] Test: Nested function with `return`
- [ ] Test: Lambda with `return`

### I/O Functions
- [ ] Test: `readln()` reads line from stdin
- [ ] Test: `read_file("path")` returns string
- [ ] Test: `write_file("path", content)` writes string

### Collection Functions
- [ ] Test: `map(f, [1, 2, 3])` applies function to each element
- [ ] Test: `filter(f, [1, 2, 3])` filters by predicate
- [ ] Test: `reduce(f, [1, 2, 3], 0)` folds collection
- [ ] Test: `keys({1: 10, 2: 20})` returns `[1, 2]`
- [ ] Test: `values({1: 10, 2: 20})` returns `[10, 20]`
- [ ] Test: `sort([3, 1, 2])` returns `[1, 2, 3]`

### String Functions
- [ ] Test: `str(42.0)` returns `"42"`
- [ ] Test: `str(3.14)` returns `"3.14"`
- [ ] Test: `num("42")` returns `42.0`
- [ ] Test: `num("3.14")` returns `3.14`
- [ ] Test: `split("a,b,c", ",")` returns `["a", "b", "c"]`
- [ ] Test: `join(["a", "b"], ",")` returns `"a,b"`
- [ ] Test: `upper("hello")` returns `"HELLO"`
- [ ] Test: `lower("HELLO")` returns `"hello"`
- [ ] Test: `trim("  hello  ")` returns `"hello"`

### Math Functions
- [x] Test: `sqrt(4.0)` returns `2.0` (SQRTSD instruction)
- [x] Test: `pow(2.0, 3.0)` returns `8.0` ✓
- [x] Test: `abs(-5.0)` returns `5.0` ✓
- [x] Test: `floor(3.7)` returns `3.0` ✓
- [x] Test: `ceil(3.2)` returns `4.0` ✓
- [x] Test: `round(3.5)` returns `4.0` ✓
- [x] Test: `sin(0.0)` returns `0.0` (FSIN instruction)
- [x] Test: `cos(0.0)` returns `1.0` (FCOS instruction)
- [x] Test: `tan(0.0)` returns `0.0` (FPTAN instruction)
- [x] Test: `asin(0.0)` returns `0.0` (FPATAN + x87 math)
- [x] Test: `acos(1.0)` returns `0.0` (FPATAN + x87 math)
- [x] Test: `atan(0.0)` returns `0.0` (FPATAN instruction)
- [x] Test: `log(2.718281828)` returns `~1.0` ✓
- [x] Test: `exp(1.0)` returns `~2.718281828` ✓

### Vector Math Functions
- [ ] Test: `dot([1, 2, 3], [4, 5, 6])` returns `32.0` (1*4 + 2*5 + 3*6)
- [ ] Test: `cross([1, 0, 0], [0, 1, 0])` returns `[0, 0, 1]`
- [ ] Test: `magnitude([3, 4])` returns `5.0`
- [ ] Test: `normalize([3, 4])` returns `[0.6, 0.8]`

### ARM64 Support
- [ ] Test: Hello world compiles and runs on ARM64
- [ ] Test: All arithmetic operations work on ARM64
- [ ] Test: All string operations work on ARM64
- [ ] Test: All map operations work on ARM64
- [ ] Test: NEON SIMD map lookup (2 keys/iteration)
- [ ] Test: NEON SIMD map lookup (4 keys/iteration)
- [ ] Test: All 200+ x86-64 tests pass on ARM64

### Error Messages
- [ ] Test: Syntax error shows line number
- [ ] Test: Type error shows line number and types involved
- [ ] Test: Undefined variable shows line number and name
- [ ] Test: Wrong argument count shows expected vs actual

---

## Nice to Have (Post-1.0.0)

### Performance
- [ ] Test: AVX2 SIMD (4 keys/iteration) faster than SSE2
- [ ] Benchmark: Perfect hashing for compile-time constant maps
- [ ] Benchmark: Binary search for 32+ sorted keys vs linear SIMD

### RISC-V Support
- [ ] Test: Hello world compiles and runs on RISC-V
- [ ] Test: RVV vector map lookup on hardware with RVV

### Advanced Features (2.0.0)
- [ ] Test: `text =~ /[0-9]+/` matches regex
- [ ] Test: `x or! "error"` exits with error message
- [x] Implementation: `me()` self-reference with tail recursion optimization ✓
  - Compiler detects `me()` calls in lambda bodies
  - Tail calls optimized to jumps instead of function calls
  - Match expressions in lambda bodies supported via parseLambdaBody()
  - Tests: factorial, sum, fibonacci, countdown all passing
- [ ] Test: Pattern matching in match expressions
- [ ] Test: `@{x: 1, y: 2}` object definition
- [ ] Test: `@simd { }` explicit SIMD block
- [ ] Test: `data || map(f)` parallel operator
- [ ] Test: `data@[indices]` gather operation
- [ ] Test: `values ||> sum` reduction
- [ ] Test: `a *+ b + c` fused multiply-add

---

## Recently Completed (2025-10-09)

### Math Functions (Today)
- [x] `sqrt(x)` using SQRTSD (SSE2) ✓
- [x] `sin(x)`, `cos(x)`, `tan(x)` using FSIN, FCOS, FPTAN (x87 FPU) ✓
- [x] `atan(x)` using FPATAN (x87 FPU) ✓
- [x] `asin(x)`, `acos(x)` using FPATAN + x87 arithmetic (no libc!) ✓

### Earlier (2025-10-09)
- [x] Strings as map[uint64]float64 ✓
- [x] String indexing `s[1]` ✓
- [x] Compile-time string concatenation ✓
- [x] `println(string_var)` with CString conversion ✓
- [x] SIMD map indexing (SSE2, AVX-512) ✓
- [x] Runtime CPU detection ✓
- [x] Match expressions `->` and `~>` ✓
- [x] Loops `@ identifier in collection` ✓
- [x] Lambdas up to 6 parameters ✓
- [x] `println(number)` and `println("literal")` ✓
- [x] `printf(format, ...)` ✓
- [x] `range(n)` ✓
- [x] `#collection` length operator ✓
- [x] List literals `[1, 2, 3]` ✓
- [x] Map literals `{1: 10, 2: 20}` ✓

---

## Current Status

**Version**: 0.1.x (pre-alpha)
**Platform**: x86-64 Linux only
**Tests Passing**: 90+

**Next Actions**:
1. Pick a blocker item
2. Write failing test (red)
3. Implement feature (green)
4. Clean up code (refactor)
5. Update this TODO
6. Repeat until 1.0.0
