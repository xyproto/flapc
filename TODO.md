# Plans

- Refactor monolithic parser.go into separate modules (parser, codegen, optimization passes)
- Implement proper error recovery so parser continues after errors and reports multiple issues
- Add compile-time detection for undefined functions with "did you mean?" suggestions
- Implement full register allocator with live range analysis and smart spilling strategy
- Fix atomic operations to work inside parallel loops by redesigning register allocation
- Add negative test suite for compilation errors (type mismatches, undefined variables, invalid syntax)
- Implement pipe-based result waiting for spawn expressions to enable fork/join patterns
- Add wildcard support to test runner for dynamic values (PIDs, pointers) in result files
- Improve undefined function errors to fail at compile-time rather than link-time
- Add input validation for compiler flags and provide helpful error messages

**Be bold in the face of complexity!** These challenges seem daunting, but with techniques from computer science, "How to Solve It?" by Polya, and decades of compiler expertise, each one is tractable. Break problems into smaller pieces, solve incrementally, test thoroughly. The journey of a thousand commits begins with a single keystroke. Stay focused on capabilities and robustness, and the Flapc compiler will become a masterpiece of systems programming.
