# ARM64 + Linux Support Status

## Completed ✓
- Basic arithmetic operations (add, subtract, multiply, divide, modulo, power)
- Comparison operations (==, !=, <, >, <=, >=)  
- Logical operations (and, or, not)
- Bitwise operations (~b, &b, |b, ^b, <<, >>)
- Variables and assignments
- Lambda functions (basic)
- Function calls
- Ret keyword (return from functions)
- Number literals (integers and floats)
- String literals (as maps)
- Map literals
- List literals (basic)
- Index expressions
- Length expressions (#)
- Unary operators (-, not)
- Postfix operators (++, --)
- ELF dynamic linking
- C FFI (basic function calls via PLT/GOT)
- Printf and print functions
- Exit and getpid syscalls
- Memory operations (write_i8, write_i32, write_f64, etc.)
- Alloc() for memory allocation
- Register assignments (in unsafe blocks)
- System calls (svc #0)
- Defer statements
- Arena blocks
- Match expressions (basic pattern matching)

## Known Limitations
- Calling non-lambda values from within lambdas requires closure/capture support
- Some advanced lambda features (closures with captured variables) not fully implemented
- Loop break/continue statements not yet implemented
- Parallel expressions not yet implemented  
- Some complex C struct handling may have issues

## Test Results
- All Go test suites pass (arithmetic, comparison, logical, basic_programs, etc.)
- Simple Flap programs compile and run correctly
- Lambda functions work for basic cases
- Exit codes are correctly returned

## Next Steps
1. Implement full closure support for captured variables
2. Add loop break/continue (@label syntax)
3. Test and fix more complex lambda scenarios
4. Add parallel expression support  
5. Improve C FFI for complex types
6. Test with larger real-world programs
