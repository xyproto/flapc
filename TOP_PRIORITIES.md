# Top 3 Remaining Tasks (Most Fundamental First)

## 1. Fix ARM64 Multi-Digit Number Printing (Most Fundamental)

**Priority:** CRITICAL
**Effort:** Small (1-2 hours)
**Impact:** Completes ARM64 basic functionality

### Current State
- Single digits work perfectly: `println(5)` ✅
- Multi-digits corrupted: `println(10)` outputs "1:" instead of "10"
- Pattern: Second+ digits have +10 offset error

### Root Cause Options
1. **Buffer addressing bug** - PC relocation or offset calculation wrong
2. **Register clobbering** - x6 or x8 getting corrupted between iterations
3. **Instruction encoding** - Store instruction malformed

### Why This Is Most Fundamental
- Blocks basic I/O on ARM64
- Required for debugging and testing
- Single-digit workaround proves infrastructure is sound
- Once fixed, ARM64 is 100% functional for basic programs

### Recommended Fix Path
**Option A: Debug current implementation** (1-2 hours)
- Add debug output to see actual buffer contents
- Verify PC relocation gives correct buffer address
- Check register values between iterations
- Fix the specific arithmetic/addressing bug

**Option B: Use libc sprintf** (30 minutes)
```c
// Call sprintf(_itoa_buffer, "%lld", number)
// Much simpler, proven correct
```

**Option C: Rewrite with simpler algorithm** (1 hour)
- Use repeated division without modulo
- Or use lookup table for small numbers
- Avoid complex addressing

---

## 2. Complete Runtime Helpers for Lists/Maps

**Priority:** HIGH
**Effort:** Medium (4-8 hours)
**Impact:** Enables advanced Flap features

### Current State
- Basic list operations: Partial (some helpers exist)
- List concatenation: Implemented but needs testing
- Map operations: Minimal implementation
- Lambda support: Basic framework exists

### What's Needed
For full Flap language support:

1. **List Operations** (2-3 hours)
   - `_flap_list_get(list, index)` - Get element at index
   - `_flap_list_set(list, index, value)` - Update element
   - `_flap_list_append(list, value)` - Add to end
   - `_flap_list_len(list)` - Get length
   - Testing on all architectures

2. **Map Operations** (2-3 hours)
   - `_flap_map_get(map, key)` - Lookup value
   - `_flap_map_set(map, key, value)` - Insert/update
   - `_flap_map_has(map, key)` - Check existence
   - `_flap_map_delete(map, key)` - Remove entry
   - Hash function for keys

3. **String Operations** (1-2 hours)
   - String concatenation ✅ (done)
   - String slicing
   - String comparison
   - String formatting (f-strings)

### Why This Is Second Priority
- Language completeness requires these
- Many Flap programs use lists/maps
- Not blocking basic programs
- Infrastructure is ready, just needs implementation

### Current Architecture
```
x86_64: Uses arena allocator (partially done)
ARM64:  Uses malloc directly (simpler, working)
Both:   Need same runtime helper interface
```

---

## 3. Fix Printf Calling Convention on ARM64

**Priority:** MEDIUM  
**Effort:** Medium (3-5 hours)
**Impact:** Enables formatted output (but println covers basics)

### Current State
- Strings via println: ✅ Working perfectly
- Numbers via println: 🟡 Single-digit works, multi-digit has bug
- Printf via libc: ❌ Not working on ARM64
- Printf works on x86_64: ✅

### The Issue
ARM64 AAPCS (calling convention) for variadic functions:
- Integer args in x0-x7
- Float args in d0-d7 (NOT in x registers!)
- Variadic args follow special rules
- Our current calling convention doesn't handle this

### Why This Is Third Priority
- Not blocking basic functionality
- Println can handle most use cases
- More complex to fix than other issues
- Less critical for initial ARM64 support

### What's Needed
1. Detect variadic vs non-variadic functions
2. Implement ARM64-specific argument marshaling
3. Handle float args in SIMD registers
4. Test with various printf format strings

### Workaround
For now, use `println()` for output:
```flap
println("x = ")
println(42)
// Instead of: printf("x = %d\n", 42)
```

---

## Summary

**Critical Path to ARM64 100%:**
1. Fix multi-digit printing (1-2 hours) → ARM64 fully functional
2. Add runtime helpers (4-8 hours) → Language feature complete
3. Fix printf (3-5 hours) → Full libc integration

**Recommended Order:**
1. **Fix #1 (multi-digit)** - Unblocks everything, small effort
2. **Add #2 (runtime helpers)** - Language completeness
3. **Fix #3 (printf)** - Nice-to-have, has workaround

**Total Effort to 100%:** ~10-15 hours across all three

The compiler is in excellent shape! These are refinements, not fundamental issues.
