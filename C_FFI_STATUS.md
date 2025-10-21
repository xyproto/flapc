# C FFI Status Report

## Summary
The Flap compiler has a robust C FFI foundation with pkg-config support, header parsing, and dynamic linking. Math functions work perfectly. String arguments need fixing.

## What Works ✓

### 1. Dynamic Linking
- **Status**: ✓ WORKING
- PLT/GOT dynamic linking fully functional
- Libraries correctly added to ELF DT_NEEDED
- Tested with: libm.so (math library), libncursesw.so

### 2. pkg-config Integration  
- **Status**: ✓ WORKING
- Automatically finds include paths via `pkg-config --cflags`
- Tested with ncurses: successfully found `/usr/include/ncurses.h`
- Handles library name variants (e.g., "ncurses", "ncursesw")

### 3. C Header Parsing
- **Status**: ✓ WORKING
- Parses `#define` constants (integers, hex, binary, bitwise expressions)
- Extracts function signatures
- Handles `#include` directives recursively
- Tested: extracted 227 constants from ncurses.h
- Supports expressions like `(1 << 5)` and `FLAG_A | FLAG_B`

### 4. Math Library Functions
- **Status**: ✓ WORKING PERFECTLY
- Float arguments passed in xmm0-xmm5 registers (correct ABI)
- Float return values in xmm0
- Tested functions: `sqrt()`, `pow()`
```flap
import m as math
x := 16.0
result := math.sqrt(x)  // Works: 4.00
power := math.pow(2.0, 10.0)  // Works: 1024
```

### 5. C Constants
- **Status**: ✓ WORKING
- Constants accessible via `namespace.CONSTANT` syntax
- Resolved at compile time
- Example: `sdl.SDL_INIT_VIDEO` resolves to `0x00000020`

## What Needs Fixing ✗

### 1. String Arguments to C Functions
- **Status**: ✗ BROKEN
- **Problem**: String literals compiled as Flap map format, not C strings
  - Flap strings: `map[index]char_code` in .rodata
  - C expects: null-terminated byte array
- **Impact**: ncurses.printw(), SDL_CreateWindow(), etc. crash
- **Root Cause**: 
  ```go
  // parser.go:3539 - compiles strings as maps
  case *StringExpr:
      // Builds map data: [count][key0][val0][key1][val1]...
  ```
- **Fix Needed**: Detect C FFI context and compile strings as C format
- **Workaround**: None currently

### 2. Pointer Return Values
- **Status**: ⚠️ UNTESTED
- C functions returning pointers (void*, WINDOW*, etc.)
- Current code converts rax to float64, may lose pointer value
- Needs testing with functions like `initscr()` → WINDOW*

### 3. Struct Arguments/Returns
- **Status**: ✗ NOT IMPLEMENTED
- No support for passing/returning C structs
- Documented limitation

## Testing Results

### Math Library (libm)
```bash
$ ./test_c_ffi_math
sqrt(16) = 4.00
pow(2, 10) = 1024
✓ SUCCESS
```

### ncurses (libncursesw)
```bash
$ ./flapc test_ncurses_simple.flap
Parsing C header: /usr/include/ncurses.h
Extracted 227 constants from ncurses
✓ COMPILES

$ ./test_ncurses_simple  
Segmentation fault (core dumped)
✗ CRASHES - string argument issue
```

## Bottom-Up Analysis

### Layer 1: ELF Generation ✓
- Dynamic section with DT_NEEDED entries
- PLT/GOT tables correctly generated
- Relocations properly configured

### Layer 2: ABI Compliance ✓
- System V AMD64 calling convention
- Integer args: rdi, rsi, rdx, rcx, r8, r9
- Float args: xmm0-xmm7
- Return: rax (int/ptr), xmm0 (float)

### Layer 3: Type Marshaling ⚠️
- Numbers → integers: ✓ WORKING
- Floats → xmm registers: ✓ WORKING  
- Strings → char*: ✗ BROKEN (compiles as Flap maps)
- Pointers: ⚠️ UNTESTED

### Layer 4: Header Processing ✓
- pkg-config path discovery
- Recursive #include parsing
- Constant extraction with expression evaluation
- Function signature parsing

## Recommendations

### Immediate Priority
1. **Fix string literal compilation for C FFI**
   - Add context flag to `compileExpression()` 
   - When in C FFI context, compile `StringExpr` as C string:
     ```
     labelName: .asciz "Hello, World\!"
     ```
   - Skip `flap_string_to_cstr` call for string literals

### Medium Priority
2. **Test pointer return values**
   - Create test with `malloc()`, `initscr()`
   - Verify pointer preservation through float64 conversion

3. **Add /lib path discovery**
   - Check `/usr/lib`, `/usr/local/lib`, `/lib`
   - Verify .so files exist before linking

### Future Enhancements
4. **Struct support** (complex, requires layout analysis)
5. **>6 arguments** (stack-based argument passing)
6. **Variadic functions** (requires special handling)

## Code Locations

- **C FFI compilation**: `parser.go:7782` (`compileCFunctionCall`)
- **String compilation**: `parser.go:3539` (case `*StringExpr`)
- **Header parsing**: `cffi.go:44` (`ExtractConstantsFromLibrary`)
- **Dynamic linking**: `elf_dynamic.go`, `dynlink.go`
- **pkg-config**: `cffi.go:86` (`getPkgConfigIncludes`)

## Test Files Created

1. `/tmp/test_c_ffi_math.flap` - ✓ Working math library test
2. `/tmp/test_ncurses_simple.flap` - ✗ Crashes on string args
3. `/tmp/test_ncurses_nostring.flap` - Created but hangs on getch()

