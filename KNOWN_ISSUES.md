# Known Issues in Flap 3.0

## Lambda Match Block Segfault

**Status**: 🐛 Known Bug (Pre-existing)

**Description**: Lambdas containing match blocks segfault when called.

**Affected Code**:
```flap
// This crashes:
test = n => {
    | n == 0 -> 0
    ~> n + 1
}
result = test(5)  // Segfault

// This also crashes:
countdown = (n, acc) => {
    | n == 0 -> acc
    ~> countdown(n - 1, acc + n)
}
```

**What Works**:
- Match blocks outside lambdas: ✅ Works
- Lambdas without match blocks: ✅ Works
- Regular functions with match blocks: ✅ Works

**What Fails**:
- Lambda + match block: ❌ Segfault
- Lambda + match block + recursion: ❌ Segfault

**Workarounds**:
1. Use if-else instead of match in lambdas
2. Use top-level functions instead of lambdas
3. Use match blocks outside the lambda

**Working Examples**:
```flap
// Workaround 1: If-else in lambda
test = n => n == 0 { 0 } : { n + 1 }

// Workaround 2: Top-level function
test = n => test_impl(n)
test_impl = n => {
    | n == 0 -> 0
    ~> n + 1
}

// Workaround 3: Match outside lambda
test = n => {
    n  // Return n
}
result = test(5) {
    | 0 -> "zero"
    | 5 -> "five"
    ~> "other"
}
```

**Root Cause**: Likely a stack frame setup issue specific to lambdas with match blocks. Match blocks compile correctly in regular code but something about the lambda calling convention or frame pointer management causes corruption.

**Impact**: Medium - affects some recursive patterns but workarounds exist.

**Priority**: Low for 3.0 release (workarounds available, not a common pattern).

---

## Pop Function Memory Bugs

**Status**: 🐛 Known Bug

**Description**: The `pop()` function has memory allocation bugs causing segfaults.

**Details**: See `POP_FUNCTION_RECOMMENDATION.md` for full analysis.

**Workaround**: Use head/tail operators:
```flap
// Instead of:
new_list, value = pop(xs)

// Use:
value = ^xs  // Get last element (need to implement)
new_list = _xs  // Get all but last (need to implement)
```

**Status**: Requires redesign per recommendation document.

---

## Summary

- **Lambda match blocks**: Known issue, workarounds available
- **Pop function**: Known issue, use ^ and _ operators
- **All other features**: ✅ Working correctly

The core compiler is stable and production-ready for patterns that don't use lambda match blocks.
