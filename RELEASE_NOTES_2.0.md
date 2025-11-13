# Flap 2.0.0 Release Notes

## Overview

Flap 2.0.0 represents a major cleanup and stabilization release, removing deprecated syntax and finalizing the language specification for production use.

## Breaking Changes

### Bitwise Operator Syntax (Final)

**Removed:**
- `shl`, `shr`, `rol`, `ror` keywords
- `<b`, `>b` operators

**New Final Syntax:**
- `<<b` - Shift left
- `>>b` - Shift right  
- `<<<b` - Rotate left (NEW)
- `>>>b` - Rotate right (NEW)

**Migration:**
```flap
// Old (no longer works)
x shl 1
x >>b 1   // Was rotate

// New (2.0 syntax)
x <<b 1   // Shift left
x >>>b 1  // Rotate right
```

### Lambda Syntax (Unified)

**Removed:**
- `==>` as separate arrow (now treated same as `=>`)

**Unified Syntax:**
```flap
// All functions use => 
add = (a, b) => a + b

// Block semantics inferred from content
classify = x => x {
    0 -> "zero"          // Has arrows = match function
    ~> "other"
}

process = x => {
    temp := x * 2        // No arrows = regular function
    ret temp
}
```

## New Features

### Triple-Angle Rotate Operators
- `<<<b` for rotate left
- `>>>b` for rotate right
- Distinct from shift operations

### Cleaner Grammar
- Removed 4 deprecated keywords
- Removed 2 deprecated tokens
- More consistent operator families

## Improvements

### Documentation
- **README.md**: Completely rewritten with accurate, tested examples
- **LANGUAGE.md**: Updated grammar and operator sections
- Removed all deprecated/legacy references

### Code Quality
- Cleaner lexer without keyword cruft
- Simplified parser logic
- Consistent operator naming

## Test Status

- **Pass Rate**: 96.5% (253/262 tests)
- **No Regressions**: All previously passing tests still pass
- **Known Issues**: List update and cons operations (pre-existing)

## Performance Analysis

### Missing High-Impact Instructions

We analyzed x86-64 instructions not currently accessible through Flap syntax and identified these priorities for future versions:

**High Priority** (game performance):
1. `prefetch` - Cache hints (2-10x speedup for memory-bound code)
2. `popcnt` - Bit counting (10-100x faster than loops)
3. `bsf`/`bsr` - Bit scanning (critical for collision detection)
4. `rdtsc` - Precise timing (nanosecond resolution)

**Medium Priority**:
5. `crc32` - Hardware checksums (10x faster)
6. `pause` - Spinlock optimization
7. `cpuid` - Runtime feature detection

These will be considered for addition as builtin functions in future releases.

## Upgrade Guide

### For Existing Code

1. **Replace shift/rotate keywords:**
   ```bash
   sed -i 's/ shl / <<b /g' *.flap
   sed -i 's/ shr / >>b /g' *.flap
   sed -i 's/ rol / <<<b /g' *.flap
   sed -i 's/ ror / >>>b /g' *.flap
   ```

2. **Update lambdas** (optional, `==>` still parses):
   ```bash
   sed -i 's/==>/=>/g' *.flap
   ```

3. **Rebuild:**
   ```bash
   flapc yourprogram.flap
   ```

### Testing Your Code

```bash
# Ensure clean build
go build

# Run your tests
flapc test.flap && ./test

# Or use the test suite as reference
go test -short
```

## What's Next

Planned for 2.1.0:
- List update operations fix
- List cons operator implementation
- Additional builtin functions (prefetch, popcount, etc.)
- ARM64 backend completion
- Register allocator improvements

## Installation

```bash
git clone https://github.com/xyproto/flapc
cd flapc
go build
sudo install -Dm755 flapc /usr/bin/flapc
```

## Links

- Repository: https://github.com/xyproto/flapc
- Documentation: [LANGUAGE.md](LANGUAGE.md)
- Examples: [testprograms/](testprograms/)

---

**Flap 2.0.0** - Clean, consistent, production-ready.
