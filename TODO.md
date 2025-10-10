# Flap Compiler TODO

## 1.0.0 Release Criteria

Release when:
- All blockers complete
- Tests passing on x86-64 and ARM64
- README.md, LANGUAGE.md, TODO.md accurate

---

## Blockers for 1.0.0

### Language Features
- [ ] Implement: Multiple-lambda dispatch syntax `f = (x) -> x, (y) -> y + 1`
- [ ] Test: Dispatch selects correct lambda based on argument type/pattern
- [ ] Test: Forward references work (function called before definition)
- [ ] Implement: Two-pass compilation (symbols collected, then code generated)

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
- [x] Test: `break` exits loop early ✓
- [x] Test: `continue` skips to next iteration ✓
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

## Recently Completed

### I/O Functions (2025-10-10)
- [x] `println()` syscall-based implementation ✓
  - Direct `write(1, buf, len)` syscall instead of printf
  - String literals: Direct write with newline
  - Whole numbers: Integer-to-ASCII conversion via assembly
  - String variables: Map-to-bytes conversion then syscall
  - No external dependencies or PLT/GOT complexity
  - Tests passing: hello, test_simple, test_println_only
- [x] Helper functions for number formatting ✓
  - `compileWholeNumberToString`: Handles 0, positive, negative
  - `compilePrintMapAsString`: Converts string maps to bytes
  - `compileFloatToString`: Framework for float printing

### Control Flow (2025-10-10)
- [x] `break` statement exits loop early ✓
- [x] `continue` statement skips to next iteration ✓
- [x] Tests: for loops with break/continue working correctly ✓


---

## Current Status

**Version**: 0.1.x (pre-alpha)
**Platform**: x86-64 Linux only
**Tests Passing**: 23/138 (17%)

**Next Actions**:
1. Pick a blocker item
2. Write failing test (red)
3. Implement feature (green)
4. Clean up code (refactor)
5. Update this TODO
6. Repeat until 1.0.0

---

## Known Issues

### println() Implementation (2025-10-10)
**Status**: ✅ RESOLVED - Implemented with syscalls

The `println()` builtin is now implemented using direct `write` syscalls instead of printf:
- ✅ String literals: Direct syscall write
- ✅ Whole numbers: ASCII conversion via assembly (0, 42, -5, 100, etc.)
- ✅ String variables: Map-to-bytes conversion then syscall
- ⚠️ Fractional floats: Currently truncate to integers (3.14 → 3)
- ✅ No external dependencies (auto-dependency commented out)
- ✅ Tests passing: hello, test_simple, test_println_only

**Future Enhancement**: Proper float-to-string conversion for fractional numbers (e.g., 3.14159 → "3.141590")

### Assignment Operator Issue (:=)
**Status**: Blocking many tests

The `:=` operator is being treated as immutable assignment instead of mutable:
- Test files use `result := abs(-5.0)` expecting mutable assignment
- Compiler error: "cannot reassign immutable variable"
- Many test files follow pattern: `x := value` then try to use/reassign
- Affects: Most tests that use `:=` for initial assignment

**Root cause**: Parser or semantic analysis treats `:=` as immutable declaration

### External Dependencies
**Status**: ✅ Fixed (flap_math)

Fixed syntax errors in flap_math repository:
- abs.flap: Changed `=>` to `->`, changed `-x` to `(0 - x)`
- Committed and pushed to github.com/xyproto/flap_math
- Repository now loads correctly

---

## Automatic Dependency Resolution Implementation

### Phase 1: Basic Infrastructure ✓ (Planned)
- [ ] Create `dependencies.go` with function→repo map
- [ ] Add initial mappings:
  - `abs`, `sin`, `cos`, `tan`, `sqrt`, `pow` → `github.com/xyproto/flap_math`
  - `println` → `github.com/xyproto/flap_stdlib`
- [ ] Implement `~/.cache/flapc/` directory creation
- [ ] Add `--update-deps` flag to main.go

### Phase 2: Git Integration
- [ ] Implement `gitClone(repoURL, destPath string) error`
- [ ] Implement `gitPull(repoPath string) error`  
- [ ] Handle HTTPS vs SSH URLs
- [ ] Cache validation (check if repo exists, is up-to-date)
- [ ] Error handling for network failures

### Phase 3: Multi-File Compilation
- [ ] Modify parser to track unknown function calls
- [ ] Collect all unknown functions before compilation
- [ ] Resolve repositories needed (deduplicate)
- [ ] Load all `.flap` files from dependency repos
- [ ] Parse and merge ASTs from multiple files
- [ ] Resolve function definitions across files

### Phase 4: Advanced Features
- [ ] User-local config: `~/.config/flapc/deps.toml`
- [ ] Per-project config: `flap.toml` in project root
- [ ] Version pinning: `abs@v1.2.3 -> github.com/xyproto/flap_math`
- [ ] Git tag/commit support
- [ ] Dependency conflict resolution
- [ ] Circular dependency detection

### Implementation Notes

**Key Design Decisions:**
1. Clone entire repo, not individual files (simpler, supports library organization)
2. Cache by full repo URL (not by function name)
3. Include ALL `.flap` files from repo (explicit export list not needed)
4. No dependency declaration in Flap code (zero boilerplate)

**Cache Structure:**
```
~/.cache/flapc/
├── github.com/
│   └── xyproto/
│       ├── flap_math/
│       │   ├── abs.flap
│       │   ├── trig.flap
│       │   └── pow.flap
│       └── flap_raylib/
│           └── window.flap
└── gitlab.com/
    └── user/
        └── project/
```

**Compilation Flow:**
1. Parse main file → collect unknown functions
2. Look up each function in hardcoded map
3. For each unique repo: clone/update if needed
4. Find all `.flap` files in cached repos
5. Parse all files into single combined AST
6. Compile as one unit (existing code handles this)

**Error Handling:**
- Unknown function with no mapping → clear error message
- Git clone failure → suggest network check or manual clone
- Conflicting function definitions → show file locations
- Circular dependencies → detect and report cycle

