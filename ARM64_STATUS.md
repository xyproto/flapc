# ARM64 Support Status

## Working ✓
- Simple value returns (e.g., `main = 42`)
- ELF binary generation for ARM64+Linux
- Target flag parsing (`--target arm64-linux`)
- Basic register operations (mov, add, sub, cmp, jump)
- Syscalls (exit)

## In Progress / Not Working ✗
- Function calls (including println, print)
- Block expressions with multiple statements
- C FFI integration
- Complex expressions
- Loops
- Match expressions
- Lambda functions
- Parallel execution

## Known Issues
1. **Flag ordering**: Flags must come BEFORE positional arguments
   - Works: `./flapc --target arm64-linux file.flap`
   - Fails: `./flapc file.flap --target arm64-linux`
   - This is a Go flag package limitation

2. **Exit codes**: Block expressions may return incorrect exit codes (INT_MAX seen)

## Testing
Simple programs work on ARM64 hardware:
```bash
# On x86_64 host:
./flapc --target arm64-linux exitcode.flap -o exitcode_arm64

# On ARM64 host:
./exitcode_arm64
echo $?  # Should print 42
```

## Architecture
- `arm64_backend.go`: Backend interface implementation
- `arm64_codegen.go`: Main code generation (4693 lines)
- `arm64_instructions.go`: Instruction encoding
- `codegen_arm64_writer.go`: ELF writer for ARM64
- `pltgot_aarch64.go`: PLT/GOT support

## Next Steps
1. Complete function call implementation
2. Fix block expression compilation
3. Add C FFI support for ARM64
4. Port runtime helpers (arena, parallel, etc.)
5. Comprehensive test suite
