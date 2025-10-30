# Flapc v1.3.0 Release Notes - Polish & Robustness

## Completed Improvements

### Core Stability & Error Handling ✅
- **Fixed critical segfault**: Programs now exit cleanly using direct syscall instead of libc exit()
  - Resolves issues with SDL3 programs crashing after successful execution
- **Improved error messages**: Replaced all log.Fatal calls with user-friendly error messages
- **Better error reporting**: Clear, actionable error messages throughout the compiler

### Code Quality ✅
- **Removed log package dependency**: Simplified error handling
- **Consistent error formatting**: All errors now follow the same format
- **Cleaner TODO.md**: Reorganized into actionable items for v1.3.0

### Parser Enhancements ✅
- **Added LoopExpr case**: Loop expressions are now recognized (though not fully implemented)
- **Better error detection**: Clear messages for unsupported features

## Partially Implemented

### Parallel Loop Reducers 🚧
- **Syntax fully parsed**: `@@ i in 0..<N { expr } | a,b | { reducer }`
- **Error messages added**: Clear indication when reducers aren't supported
- **Foundation laid**: Structure in place for future implementation

## Not Yet Implemented

### Features Postponed to v1.4.0
- Full parallel loop reducer implementation
- Atomic operations (atomic_add, atomic_cas, atomic_load, atomic_store)
- Hot reload infrastructure
- Thread-local storage for parallel computations
- Network message parsing improvements

## Breaking Changes
None - all changes are backward compatible.

## Migration Guide
No migration needed - existing code will continue to work with improved stability.

## Known Limitations
1. Parallel loop expressions with reducers parse but don't compile
2. Loop expressions (that return values) are not fully implemented
3. Atomic operations are not yet available

## Next Steps (v1.4.0)
- Complete parallel reducer implementation
- Add atomic operations support
- Implement hot reload for live code updates
- Enhance network programming capabilities

## Testing
All existing tests pass. New test files created:
- `testprograms/parallel_sum.flap` - Demonstrates reducer syntax (parse-only)

## Contributors
This release focused on stability and robustness improvements to make Flapc more production-ready.