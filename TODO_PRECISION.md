# Remaining Work: Printf Precision Support

## Current State
- Float decimals work perfectly with 6 digits
- Printf IGNORES precision specifiers like %.2f
- Output: "3.333333" instead of "3.33"

## What Needs to Be Done

### 1. Parse Precision (DONE in attempt)
Already added code to extract precision from format string:
- %.2f → precision = 2
- %.6f → precision = 6
- %f → precision = 6 (default)

### 2. Update Function Signature
Change:
```go
func (fc *FlapCompiler) emitSyscallPrintFloatPrecise()
```
To:
```go
func (fc *FlapCompiler) emitSyscallPrintFloatPrecise(precision int)
```

### 3. Dynamic Multiplier
Instead of hardcoded 1000000:
```go
multiplier := 1
for i := 0; i < precision; i++ {
    multiplier *= 10
}
fc.out.MovImmToReg("rax", fmt.Sprintf("%d", multiplier))
```

### 4. Dynamic Digit Extraction
Instead of 6 hardcoded divisions:
```go
for i := precision - 1; i >= 0; i-- {
    fc.out.XorRegWithReg("rdx", "rdx")
    fc.out.Emit([]byte{0x48, 0xf7, 0xf1}) // div rcx
    fc.out.AddImmToReg("rdx", 48)
    fc.out.MovByteRegToMem("dl", "rsp", 64+i)
}
```

### 5. Dynamic Print Length
```go
fc.out.MovImmToReg("rdx", fmt.Sprintf("%d", precision))
```

## Test Cases That Will Pass
1. TestArithmeticOperations/float_division: %.2f → "3.33\n" ✅
2. TestPrintfFormatting/printf_float: %.2f → "3.14\n" ✅  
3. TestForeignTypeAnnotations/cdouble: %.6f → "3.141590\n" ✅

## Implementation Notes
- Keep all the inline assembly approach
- Just make the digit count dynamic
- Precision range: 0-15 (float64 limit)
- Default precision: 6

## Why This Failed
File editing with tabs vs spaces was causing issues.
Need to rebuild the entire function cleanly.
