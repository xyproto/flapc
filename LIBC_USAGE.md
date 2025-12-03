# Flapc - libc Usage Documentation

## Summary

**Flapc uses libc ONLY when you use C FFI.** For pure Flap programs on Linux, no libc is needed!

## Behavior Matrix

### Linux x86_64

| Program Type | libc Required? | How it Works |
|-------------|---------------|--------------|
| Hello World | ❌ NO | Direct syscalls (sys_write, sys_exit) |
| Pure Flap code | ❌ NO | All I/O via Linux syscalls |
| printf with floats | ❌ NO | Custom inline float formatting + syscalls |
| C FFI (c.malloc, etc) | ✅ YES | Dynamic linking to libc.so.6 |
| Import "sdl3" | ✅ YES | Dynamic linking to library |

### Windows x86_64  

| Program Type | libc Required? | How it Works |
|-------------|---------------|--------------|
| All programs | ✅ YES | Windows API calls via kernel32.dll/msvcrt.dll |

## Examples

### No libc - Pure Syscalls (Linux)

```flap
println("hello world")
x := 10 / 3
printf("result: %.2f\n", x)
```

**Binary dependencies:**
```bash
$ readelf -d hello | grep NEEDED
(no output - no dependencies!)
```

**How it works:**
- `println()` → `sys_write` syscall
- `printf()` → custom float formatting + `sys_write` syscall
- Float decimals extracted using SSE instructions
- Exit via `sys_exit` syscall

### With libc - C FFI (Linux)

```flap
ptr = c.malloc(1024)
c.memset(ptr, 0, 1024)
c.free(ptr)
println("done")
```

**Binary dependencies:**
```bash
$ readelf -d program | grep NEEDED
 0x0000000000000001 (NEEDED)             Shared library: [libc.so.6]
```

**How it works:**
- `c.malloc()` → PLT entry → libc malloc
- `c.free()` → PLT entry → libc free
- Dynamic linker loads libc.so.6 at runtime

## Technical Details

### Linux Syscall Implementation

For pure Flap programs on Linux, the compiler generates direct syscalls:

```assembly
; println("hello")
mov rax, 1          ; sys_write
mov rdi, 1          ; stdout
lea rsi, [string]   ; buffer
mov rdx, 5          ; length
syscall

; Exit
mov rax, 60         ; sys_exit
xor rdi, rdi        ; status 0
syscall
```

### Float Decimal Formatting

Printf with floats uses a fully inline implementation:
1. Extract integer part using `cvttsd2si`
2. Extract fractional part by subtraction
3. Multiply by 1,000,000 using SSE
4. Extract 6 decimal digits using division
5. Print via `sys_write` syscall

**No libc needed!** All floating-point operations use SSE instructions.

### C FFI Dynamic Linking

When you use `c.function()`, the compiler:
1. Generates PLT (Procedure Linkage Table) entries
2. Creates GOT (Global Offset Table) entries
3. Adds NEEDED entries in ELF dynamic section
4. Emits CALL instructions to PLT stubs

At runtime, the dynamic linker:
1. Loads required shared libraries
2. Resolves function addresses
3. Patches GOT entries
4. Subsequent calls go directly to libc

## Compiler Warnings

You may see warnings like:
```
Warning: No PLT entry or label for printf
```

These are **harmless**. They occur because the compiler tracks potential C function calls (for fallback scenarios) but doesn't actually generate PLT entries when using syscalls. The warnings can be safely ignored.

## Verifying Binary Dependencies

### Linux

```bash
# Check for dynamic dependencies
readelf -d binary | grep NEEDED

# Check if truly static
ldd binary
# Output: "statically linked" = no libc
# Output: "libc.so.6 => ..." = uses libc
```

### Windows

```bash
# All Windows binaries link to system DLLs
dumpbin /DEPENDENTS binary.exe
```

## Benefits

### No libc Required
- **Smaller binaries**: No libc overhead
- **Faster startup**: No dynamic linking
- **Simpler deployment**: Single self-contained binary
- **Better security**: Smaller attack surface

### With libc (C FFI)
- **Full C ecosystem**: Call any C library
- **Compatibility**: Use standard libraries
- **Flexibility**: Mix Flap and C code

## Conclusion

Flapc gives you the best of both worlds:
- **Pure Flap** → No dependencies, pure syscalls
- **C FFI** → Full access to C libraries when needed

You choose which approach fits your use case!
