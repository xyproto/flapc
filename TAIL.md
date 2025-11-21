# Tail Operator Implementation Notes

## Current State

The tail operator `_` is designed to work as both prefix and postfix:
- `_xs` - tail as prefix
- `xs_` - tail as postfix

Both should return all elements except the first one.

### For Lists
When taking the tail of a list, all elements except the first one should be copied somewhere, then all the indexes should be reduced by 1.

### For Maps
When taking the tail of a map, all elements except the first one should be copied somewhere, skipping the first element. If the indexes are increasing numbers starting with 0, it's a list; otherwise it's a map, for the purpose of using tail.

## Known Issues

The tail operator has proven problematic to implement correctly. Current implementation may not handle all edge cases properly, especially:
- Nested data structures
- Mixed list/map types
- Memory management during copying

## Deferred

This feature has been temporarily deferred to focus on more critical tasks like SDL3 + Windows support.

---

# SDL3 + Windows Support Issues

## Current State (2025-11-21)

The Flapc compiler can now generate PE executables that run under Wine. Basic programs like "Hello World" work correctly with proper Microsoft x64 calling convention.

However, there are two main issues:

### Issue 1: exitf() Implementation

The exitf() function needs to print formatted output to stderr and then exit. The implementation attempted to use fprintf() with stderr accessed via `__acrt_iob_func(2)`, but Wine's msvcrt.dll does not implement `__acrt_iob_func` (it's a newer Windows 10+ CRT function).

**Current Workaround**: exitf() should be simplified to either:
- Use write syscall directly to fd 2 (stderr) like eprintf does
- Or use a Wine-compatible way to access stderr (like the global `_iob` array)

This is a low-priority issue since:
- Regular printf() works fine
- exitf() is mainly for debugging/error reporting
- Linux target works correctly

### Issue 2: SDL3 Window Creation

SDL3 programs fail when run under Wine with errors like "Failed to create window".

**Root Causes**:
- Wine's DirectX/DXGI support is limited
- SDL3 prefers DirectX on Windows, which Wine doesn't fully support
- The formatted error messages from exitf() don't display properly (see Issue 1)

**Potential Solutions**:
1. Test on actual Windows instead of Wine
2. Use SDL2 instead (better Wine compatibility)
3. Force SDL3 to use software/OpenGL rendering
4. Use Xvfb with Wine for headless testing
5. Focus on Linux/x86_64 target first, Windows later

## Decision

SDL3 + Windows support is deferred. Priority should be:
1. Complete remaining TODO items for Linux target
2. Ensure all tests pass on Linux
3. Return to Windows support after core features are stable

The compiler successfully generates PE executables that work with Wine for basic programs (printf, file I/O, etc.), which is sufficient progress for now.
