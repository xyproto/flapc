# Flapc Compiler Improvements - 2025-11-12

## Overview
Fixed critical issues in the Flapc compiler affecting error handling, loops, and C FFI integration.
Test pass rate improved from ~90% to 96.5% with 9 test fixes.

## Issues Fixed

### 1. Result Type Error Handling
- **`or!` operator**: Removed incorrect assertion implementation, fixed NaN handling
- **`.error` property**: Fixed parser to correctly transform `x.error` into builtin call
- **Impact**: Complete Result type functionality now works correctly

### 2. Loop Functionality  
- **List iteration**: Added support for `@ item in list_variable` without requiring `max`
- **Loop break**: Fixed `ret @` syntax to properly exit current loop
- **Nested loops**: Fixed critical stack alignment bug causing segfaults with printf

### 3. C FFI Integration
- **Stack alignment**: Callee-saved register preservation now maintains 16-byte alignment
- **Buffering**: println now calls fflush when printf has been used
- **Impact**: Nested loops can now safely call C functions like printf

## Technical Details

### Stack Alignment Fix
The root cause of nested loop crashes was that `push` instructions for callee-saved 
registers broke the required 16-byte stack alignment for C function calls. Fixed by 
adding 8-byte padding when saving registers.

### Parser Fix for `.error`
The parser checks for special properties like `.error` were outside the identifier 
handling block, causing `x.error` to be parsed as NamespacedIdentExpr instead of 
transforming into `_error_code_extract(x)`.

## Test Results
- **Before**: ~243/262 tests passing (92.7%)
- **After**: ~253/262 tests passing (96.5%)
- **Fixed**: 9 specific test cases
- **Commits**: 3 focused commits with clear descriptions

## Remaining Work
- List update operations (runtime helper bug)
- List cons operator codegen
- Minor test failures (lambda, ENet dependencies)

## Files Modified
- `codegen.go`: or! fix, loop alignment, fflush call, label -1 handling
- `parser.go`: .error property fix, list iteration support

All changes maintain backward compatibility and follow the Flap spec in LANGUAGE.md.
