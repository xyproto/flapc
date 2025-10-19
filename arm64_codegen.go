package main

import (
	"fmt"
	"os"
	"unsafe"
)

// ARM64CodeGen handles ARM64 code generation for macOS
type ARM64CodeGen struct {
	out            *ARM64Out
	eb             *ExecutableBuilder
	stackVars      map[string]int    // variable name -> stack offset from fp
	mutableVars    map[string]bool   // variable name -> is mutable
	stackSize      int               // current stack size
	stackFrameSize uint64            // total stack frame size allocated in prologue
	stringCounter  int               // counter for string labels
	stringInterns  map[string]string // string value -> label (for string interning)
	labelCounter   int               // counter for jump labels
	activeLoops    []ARM64LoopInfo   // stack of active loops for break/continue
	lambdaFuncs    []ARM64LambdaFunc // list of lambda functions to generate
	lambdaCounter  int               // counter for lambda names
	currentLambda  *ARM64LambdaFunc  // current lambda being compiled (for recursion)
}

// ARM64LambdaFunc represents a lambda function for ARM64
type ARM64LambdaFunc struct {
	Name   string
	Params []string
	Body   Expression
}

// ARM64LoopInfo tracks information about an active loop
type ARM64LoopInfo struct {
	Label            int   // Loop label (@1, @2, @3, etc.)
	StartPos         int   // Code position of loop start (condition check)
	ContinuePos      int   // Code position for continue (increment step)
	EndPatches       []int // Positions that need to be patched to jump to loop end
	ContinuePatches  []int // Positions that need to be patched to jump to continue position
	IteratorOffset   int   // Stack offset for iterator variable
	IndexOffset      int   // Stack offset for index counter (list loops only)
	UpperBoundOffset int   // Stack offset for limit (range) or length (list)
	ListPtrOffset    int   // Stack offset for list pointer (list loops only)
	IsRangeLoop      bool  // True for range loops, false for list loops
}

// NewARM64CodeGen creates a new ARM64 code generator
func NewARM64CodeGen(eb *ExecutableBuilder) *ARM64CodeGen {
	return &ARM64CodeGen{
		out:           &ARM64Out{out: &Out{machine: Platform{Arch: ArchARM64, OS: OSDarwin}, writer: eb.TextWriter(), eb: eb}},
		eb:            eb,
		stackVars:     make(map[string]int),
		mutableVars:   make(map[string]bool),
		stackSize:     0,
		stringCounter: 0,
		stringInterns: make(map[string]string),
		labelCounter:  0,
	}
}

// CompileProgram compiles a Flap program to ARM64
func (acg *ARM64CodeGen) CompileProgram(program *Program) error {
	// Function prologue - follow ARM64 ABI
	// Allocate stack frame: 16 bytes for saved registers + 256 bytes for local variables
	// Total: 272 bytes, rounded to 16-byte alignment = 272 bytes
	stackFrameSize := uint64(272)
	if err := acg.out.SubImm64("sp", "sp", uint32(stackFrameSize)); err != nil {
		return err
	}
	// Save frame pointer and link register
	if err := acg.out.StrImm64("x29", "sp", 0); err != nil {
		return err
	}
	if err := acg.out.StrImm64("x30", "sp", 8); err != nil {
		return err
	}
	// Set frame pointer
	if err := acg.out.AddImm64("x29", "sp", 0); err != nil {
		return err
	}

	// Store stack frame size for epilogue
	acg.stackFrameSize = stackFrameSize

	// Compile each statement
	for _, stmt := range program.Statements {
		if err := acg.compileStatement(stmt); err != nil {
			return err
		}
	}

	// Function epilogue (if no explicit exit)
	if err := acg.out.MovImm64("x0", 0); err != nil {
		return err
	}
	if err := acg.out.LdrImm64("x30", "sp", 8); err != nil {
		return err
	}
	if err := acg.out.LdrImm64("x29", "sp", 0); err != nil {
		return err
	}
	if err := acg.out.AddImm64("sp", "sp", uint32(acg.stackFrameSize)); err != nil {
		return err
	}
	if err := acg.out.Return("x30"); err != nil {
		return err
	}

	// Generate lambda functions after main program
	if err := acg.generateLambdaFunctions(); err != nil {
		return err
	}

	return nil
}

// compileStatement compiles a single statement
func (acg *ARM64CodeGen) compileStatement(stmt Statement) error {
	switch s := stmt.(type) {
	case *ExpressionStmt:
		return acg.compileExpression(s.Expr)
	case *AssignStmt:
		return acg.compileAssignment(s)
	case *LoopStmt:
		return acg.compileLoopStatement(s)
	default:
		return fmt.Errorf("unsupported statement type for ARM64: %T", stmt)
	}
}

// compileExpression compiles an expression and leaves result in d0 (float64 register)
func (acg *ARM64CodeGen) compileExpression(expr Expression) error {
	switch e := expr.(type) {
	case *NumberExpr:
		// Flap uses float64 for all numbers
		// For whole numbers, convert via integer; for decimals, load from .rodata
		if e.Value == float64(int64(e.Value)) {
			// Whole number - convert to int64, then to float64
			val := int64(e.Value)
			// Load integer into x0
			if err := acg.out.MovImm64("x0", uint64(val)); err != nil {
				return err
			}
			// Convert x0 (int64) to d0 (float64)
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e}) // scvtf d0, x0
		} else {
			// Decimal number - store in .rodata and load
			labelName := fmt.Sprintf("float_%d", acg.stringCounter)
			acg.stringCounter++

			// Convert float64 to 8 bytes (little-endian)
			bits := uint64(0)
			*(*float64)(unsafe.Pointer(&bits)) = e.Value
			var floatData []byte
			for i := 0; i < 8; i++ {
				floatData = append(floatData, byte((bits>>(i*8))&0xFF))
			}
			acg.eb.Define(labelName, string(floatData))

			// Load address of float into x0 using PC-relative addressing
			offset := uint64(acg.eb.text.Len())
			acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
				offset:     offset,
				symbolName: labelName,
			})
			// ADRP x0, label@PAGE
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x90})
			// ADD x0, x0, label@PAGEOFF
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x91})

			// Load float64 from [x0] into d0
			// ldr d0, [x0]
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x40, 0xfd})
		}

	case *StringExpr:
		// Strings are represented as map[uint64]float64
		// Map format: [count][key0][val0][key1][val1]...

		// Check if we've already interned this string
		var labelName string
		if existingLabel, exists := acg.stringInterns[e.Value]; exists {
			// Reuse existing label for this string
			labelName = existingLabel
		} else {
			// Create new label and intern it
			labelName = fmt.Sprintf("str_%d", acg.stringCounter)
			acg.stringCounter++
			acg.stringInterns[e.Value] = labelName

			// Build map data: count followed by key-value pairs
			var mapData []byte

			// Count (number of characters)
			count := float64(len(e.Value))
			countBits := uint64(0)
			*(*float64)(unsafe.Pointer(&countBits)) = count
			for i := 0; i < 8; i++ {
				mapData = append(mapData, byte((countBits>>(i*8))&0xFF))
			}

			// Add each character as a key-value pair
			for idx, ch := range e.Value {
				// Key: character index as float64
				keyVal := float64(idx)
				keyBits := uint64(0)
				*(*float64)(unsafe.Pointer(&keyBits)) = keyVal
				for i := 0; i < 8; i++ {
					mapData = append(mapData, byte((keyBits>>(i*8))&0xFF))
				}

				// Value: character code as float64
				charVal := float64(ch)
				charBits := uint64(0)
				*(*float64)(unsafe.Pointer(&charBits)) = charVal
				for i := 0; i < 8; i++ {
					mapData = append(mapData, byte((charBits>>(i*8))&0xFF))
				}
			}

			acg.eb.Define(labelName, string(mapData))
		}

		// Load address into x0
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: labelName,
		})
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x90}) // ADRP x0, label@PAGE
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x91}) // ADD x0, x0, label@PAGEOFF

		// Convert pointer to float64: scvtf d0, x0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})

	case *IdentExpr:
		// Load variable from stack into d0
		stackOffset, exists := acg.stackVars[e.Name]
		if !exists {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Error: undefined variable '%s'\n", e.Name)
			}
			return fmt.Errorf("undefined variable: %s", e.Name)
		}
		// ldr d0, [x29, #offset]
		// x29 points to saved fp location, variables start at offset 16
		offset := int32(16 + stackOffset - 8)
		if err := acg.out.LdrImm64Double("d0", "x29", offset); err != nil {
			return err
		}

	case *BinaryExpr:
		// Compile left operand (result in d0)
		if err := acg.compileExpression(e.Left); err != nil {
			return err
		}

		// Push d0 onto stack to save left operand (maintain 16-byte alignment)
		acg.out.SubImm64("sp", "sp", 16)
		// str d0, [sp]
		acg.out.out.writer.WriteBytes([]byte{0xe0, 0x03, 0x00, 0xfd}) // str d0, [sp]

		// Compile right operand (result in d0)
		if err := acg.compileExpression(e.Right); err != nil {
			return err
		}

		// Move right operand to d1
		// fmov d1, d0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x40, 0x60, 0x1e})

		// Pop left operand into d0
		acg.out.out.writer.WriteBytes([]byte{0xe0, 0x03, 0x40, 0xfd}) // ldr d0, [sp]
		acg.out.AddImm64("sp", "sp", 16)

		// Perform operation: d0 = d0 op d1
		switch e.Operator {
		case "+":
			// fadd d0, d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x28, 0x61, 0x1e})
		case "-":
			// fsub d0, d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x38, 0x61, 0x1e})
		case "*":
			// fmul d0, d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x08, 0x61, 0x1e})
		case "/":
			// fdiv d0, d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x18, 0x61, 0x1e})
		case "==":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, eq (x0 = 1 if equal, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x17, 0x9f, 0x9a})
			// scvtf d0, x0 (convert to float)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "!=":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, ne
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x07, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "<":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, lt
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0xa7, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "<=":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, le
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0xc7, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case ">":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, gt
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0xd7, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case ">=":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, ge
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0xb7, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "mod", "%":
			// Modulo: a % b = a - b * floor(a / b)
			// d0 = dividend (a), d1 = divisor (b)
			// fmov d2, d0 (save dividend in d2)
			acg.out.out.writer.WriteBytes([]byte{0x02, 0x40, 0x60, 0x1e})
			// fmov d3, d1 (save divisor in d3)
			acg.out.out.writer.WriteBytes([]byte{0x23, 0x40, 0x60, 0x1e})
			// fdiv d0, d0, d1 (d0 = a / b)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x18, 0x61, 0x1e})
			// fcvtzs x0, d0 (x0 = floor(a / b) as int)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// scvtf d0, x0 (d0 = floor(a / b) as float)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
			// fmul d0, d0, d3 (d0 = floor(a / b) * b)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x08, 0x63, 0x1e})
			// fsub d0, d2, d0 (d0 = a - floor(a / b) * b)
			acg.out.out.writer.WriteBytes([]byte{0x40, 0x38, 0x60, 0x1e})
		case "and":
			// Logical AND: returns 1.0 if both non-zero, else 0.0
			// Compare d0 with 0.0
			// fmov d2, xzr (d2 = 0.0)
			acg.out.out.writer.WriteBytes([]byte{0xe2, 0x03, 0x67, 0x9e})
			// fcmp d0, d2
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x62, 0x1e})
			// cset x0, ne (x0 = 1 if d0 != 0, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x07, 0x9f, 0x9a})
			// Compare d1 with 0.0
			// fcmp d1, d2
			acg.out.out.writer.WriteBytes([]byte{0x20, 0x20, 0x62, 0x1e})
			// cset x1, ne (x1 = 1 if d1 != 0, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe1, 0x07, 0x9f, 0x9a})
			// and x0, x0, x1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x01, 0x8a})
			// scvtf d0, x0 (convert result to float)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "or":
			// Logical OR: returns 1.0 if either non-zero, else 0.0
			// Compare d0 with 0.0
			// fmov d2, xzr (d2 = 0.0)
			acg.out.out.writer.WriteBytes([]byte{0xe2, 0x03, 0x67, 0x9e})
			// fcmp d0, d2
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x62, 0x1e})
			// cset x0, ne (x0 = 1 if d0 != 0, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x07, 0x9f, 0x9a})
			// Compare d1 with 0.0
			// fcmp d1, d2
			acg.out.out.writer.WriteBytes([]byte{0x20, 0x20, 0x62, 0x1e})
			// cset x1, ne (x1 = 1 if d1 != 0, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe1, 0x07, 0x9f, 0x9a})
			// orr x0, x0, x1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x01, 0xaa})
			// scvtf d0, x0 (convert result to float)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "xor":
			// Logical XOR: returns 1.0 if exactly one non-zero, else 0.0
			// Compare d0 with 0.0
			// fmov d2, xzr (d2 = 0.0)
			acg.out.out.writer.WriteBytes([]byte{0xe2, 0x03, 0x67, 0x9e})
			// fcmp d0, d2
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x62, 0x1e})
			// cset x0, ne (x0 = 1 if d0 != 0, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x07, 0x9f, 0x9a})
			// Compare d1 with 0.0
			// fcmp d1, d2
			acg.out.out.writer.WriteBytes([]byte{0x20, 0x20, 0x62, 0x1e})
			// cset x1, ne (x1 = 1 if d1 != 0, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe1, 0x07, 0x9f, 0x9a})
			// eor x0, x0, x1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x01, 0xca})
			// scvtf d0, x0 (convert result to float)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "shl":
			// Shift left: convert to int64, shift, convert back
			// fcvtzs x0, d0 (x0 = int64(d0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// fcvtzs x1, d1 (x1 = int64(d1))
			acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x78, 0x9e})
			// lsl x0, x0, x1 (x0 <<= x1)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0xc1, 0x9a})
			// scvtf d0, x0 (d0 = float64(x0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "shr":
			// Shift right: convert to int64, shift, convert back
			// fcvtzs x0, d0 (x0 = int64(d0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// fcvtzs x1, d1 (x1 = int64(d1))
			acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x78, 0x9e})
			// lsr x0, x0, x1 (x0 >>= x1)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x24, 0xc1, 0x9a})
			// scvtf d0, x0 (d0 = float64(x0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "rol":
			// Rotate left: convert to int64, rotate, convert back
			// fcvtzs x0, d0 (x0 = int64(d0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// fcvtzs x1, d1 (x1 = int64(d1))
			acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x78, 0x9e})
			// neg x2, x1 (x2 = -x1 for rotate)
			acg.out.out.writer.WriteBytes([]byte{0xe2, 0x03, 0x01, 0xcb})
			// ror x0, x0, x2 (rotate left by negating right rotate)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x2c, 0xc2, 0x9a})
			// scvtf d0, x0 (d0 = float64(x0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "ror":
			// Rotate right: convert to int64, rotate, convert back
			// fcvtzs x0, d0 (x0 = int64(d0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// fcvtzs x1, d1 (x1 = int64(d1))
			acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x78, 0x9e})
			// ror x0, x0, x1 (x0 rotate right by x1)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x2c, 0xc1, 0x9a})
			// scvtf d0, x0 (d0 = float64(x0))
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		default:
			return fmt.Errorf("unsupported binary operator for ARM64: %s", e.Operator)
		}

	case *ListExpr:
		// Lists are stored as: [count][elem0][elem1]...
		// For now, store list data in rodata
		labelName := fmt.Sprintf("list_%d", acg.stringCounter)
		acg.stringCounter++

		var listData []byte

		// Count
		count := float64(len(e.Elements))
		countBits := uint64(0)
		*(*float64)(unsafe.Pointer(&countBits)) = count
		for i := 0; i < 8; i++ {
			listData = append(listData, byte((countBits>>(i*8))&0xFF))
		}

		// Elements (for now, only support number literals)
		for _, elem := range e.Elements {
			if numExpr, ok := elem.(*NumberExpr); ok {
				elemBits := uint64(0)
				*(*float64)(unsafe.Pointer(&elemBits)) = numExpr.Value
				for i := 0; i < 8; i++ {
					listData = append(listData, byte((elemBits>>(i*8))&0xFF))
				}
			} else {
				return fmt.Errorf("unsupported list element type for ARM64: %T", elem)
			}
		}

		acg.eb.Define(labelName, string(listData))

		// Load address into x0
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: labelName,
		})
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x90}) // ADRP x0, label@PAGE
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x91}) // ADD x0, x0, label@PAGEOFF

		// Convert pointer to float64
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e}) // scvtf d0, x0

	case *IndexExpr:
		// Compile the list/map expression
		if err := acg.compileExpression(e.List); err != nil {
			return err
		}

		// d0 now contains pointer to list (as float64)
		// Convert to integer pointer: fcvtzs x0, d0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

		// Save list pointer (maintain 16-byte stack alignment)
		acg.out.SubImm64("sp", "sp", 16)
		acg.out.StrImm64("x0", "sp", 0)

		// Compile index expression
		if err := acg.compileExpression(e.Index); err != nil {
			return err
		}

		// Convert index from float64 to int64: fcvtzs x1, d0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x78, 0x9e})

		// Restore list pointer
		acg.out.LdrImm64("x0", "sp", 0)
		acg.out.AddImm64("sp", "sp", 16)

		// x0 = list pointer, x1 = index
		// Skip past count (8 bytes) and index by (index * 8)
		acg.out.AddImm64("x0", "x0", 8)
		// x1 = x1 << 3 (multiply by 8)
		acg.out.out.writer.WriteBytes([]byte{0x21, 0xf0, 0x7d, 0xd3}) // lsl x1, x1, #3
		// x0 = x0 + x1
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x01, 0x8b}) // add x0, x0, x1

		// Load element into d0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x40, 0xfd}) // ldr d0, [x0]

	case *CallExpr:
		return acg.compileCall(e)

	case *DirectCallExpr:
		return acg.compileDirectCall(e)

	case *MatchExpr:
		return acg.compileMatchExpr(e)

	case *LambdaExpr:
		// Generate a unique function name for this lambda
		acg.lambdaCounter++
		funcName := fmt.Sprintf("lambda_%d", acg.lambdaCounter)

		// Store lambda for later code generation
		acg.lambdaFuncs = append(acg.lambdaFuncs, ARM64LambdaFunc{
			Name:   funcName,
			Params: e.Params,
			Body:   e.Body,
		})

		// Return function pointer as float64 in d0
		// Load function address into x0
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: funcName,
		})
		// ADRP x0, funcName@PAGE
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x90})
		// ADD x0, x0, funcName@PAGEOFF
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x91})

		// Convert pointer to float64: scvtf d0, x0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})

	case *UnaryExpr:
		// Compile the operand first (result in d0)
		if err := acg.compileExpression(e.Operand); err != nil {
			return err
		}

		switch e.Operator {
		case "-":
			// Unary minus: negate the value
			// Use fneg d0, d0 instruction
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x40, 0x60, 0x1e}) // fneg d0, d0

		case "not":
			// Logical NOT: returns 1.0 if operand is 0.0, else 0.0
			// Compare d0 with 0.0
			// fmov d1, xzr (d1 = 0.0)
			acg.out.out.writer.WriteBytes([]byte{0xe1, 0x03, 0x67, 0x9e})
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, eq (x0 = 1 if equal, else 0)
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x17, 0x9f, 0x9a})
			// scvtf d0, x0 (convert to float64)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})

		case "~b":
			// Bitwise NOT: convert to int64, NOT, convert back
			// fcvtzs x0, d0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// mvn x0, x0 (bitwise NOT)
			acg.out.out.writer.WriteBytes([]byte{0xe0, 0x03, 0x00, 0xaa})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})

		default:
			return fmt.Errorf("unsupported unary operator for ARM64: %s", e.Operator)
		}

	case *LengthExpr:
		// Compile the operand (should be a list/map, returns pointer as float64 in d0)
		if err := acg.compileExpression(e.Operand); err != nil {
			return err
		}

		// Convert pointer from float64 to integer in x0
		// fcvtzs x0, d0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

		// Load length from list/map (first 8 bytes)
		// ldr d0, [x0]
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x40, 0xfd})

		// Length is now in d0 as float64

	case *MapExpr:
		// Map literal stored as: [count (float64)] [key1] [value1] [key2] [value2] ...
		labelName := fmt.Sprintf("map_%d", acg.stringCounter)
		acg.stringCounter++

		var mapData []byte

		// Add count
		count := float64(len(e.Keys))
		countBits := uint64(0)
		*(*float64)(unsafe.Pointer(&countBits)) = count
		for i := 0; i < 8; i++ {
			mapData = append(mapData, byte((countBits>>(i*8))&0xFF))
		}

		// Add key-value pairs (only number literals supported for now)
		for i := range e.Keys {
			if keyNum, ok := e.Keys[i].(*NumberExpr); ok {
				keyBits := uint64(0)
				*(*float64)(unsafe.Pointer(&keyBits)) = keyNum.Value
				for j := 0; j < 8; j++ {
					mapData = append(mapData, byte((keyBits>>(j*8))&0xFF))
				}
			} else {
				return fmt.Errorf("unsupported map key type for ARM64: %T", e.Keys[i])
			}

			if valNum, ok := e.Values[i].(*NumberExpr); ok {
				valBits := uint64(0)
				*(*float64)(unsafe.Pointer(&valBits)) = valNum.Value
				for j := 0; j < 8; j++ {
					mapData = append(mapData, byte((valBits>>(j*8))&0xFF))
				}
			} else {
				return fmt.Errorf("unsupported map value type for ARM64: %T", e.Values[i])
			}
		}

		acg.eb.Define(labelName, string(mapData))

		// Load address into x0
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: labelName,
		})
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x90}) // ADRP x0, label@PAGE
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x91}) // ADD x0, x0, label@PAGEOFF

		// Convert pointer to float64
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e}) // scvtf d0, x0

	case *InExpr:
		// Compile value to search for (result in d0)
		if err := acg.compileExpression(e.Value); err != nil {
			return err
		}

		// Save search value to stack
		acg.out.SubImm64("sp", "sp", 16)
		acg.out.out.writer.WriteBytes([]byte{0xe0, 0x03, 0x00, 0xfd}) // str d0, [sp]

		// Compile container expression (result in d0 as float64 pointer)
		if err := acg.compileExpression(e.Container); err != nil {
			return err
		}

		// Save container pointer
		acg.out.out.writer.WriteBytes([]byte{0xe0, 0x07, 0x00, 0xfd}) // str d0, [sp, #8]

		// Convert container pointer from float64 to integer in x0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e}) // fcvtzs x0, d0

		// Load count from container (first 8 bytes)
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x40, 0xfd}) // ldr d1, [x0]

		// Convert count to integer in x1
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x78, 0x9e}) // fcvtzs x1, d1

		// x0 = container pointer, x1 = count
		// x2 = loop index (start at 0)
		if err := acg.out.MovImm64("x2", 0); err != nil {
			return err
		}

		// Load search value into d2
		acg.out.out.writer.WriteBytes([]byte{0xe2, 0x03, 0x40, 0xfd}) // ldr d2, [sp]

		// Loop start
		loopStartPos := acg.eb.text.Len()

		// Compare index with count: cmp x2, x1
		acg.out.out.writer.WriteBytes([]byte{0x5f, 0x00, 0x01, 0xeb}) // cmp x2, x1

		// If index >= count, jump to not_found
		notFoundJumpPos := acg.eb.text.Len()
		acg.out.BranchCond("ge", 0) // Placeholder

		// Calculate element address: x0 + 8 + (x2 * 8)
		// x3 = x2 * 8
		acg.out.out.writer.WriteBytes([]byte{0x43, 0xf0, 0x7d, 0xd3}) // lsl x3, x2, #3
		// x3 = x0 + x3
		acg.out.out.writer.WriteBytes([]byte{0x03, 0x00, 0x00, 0x8b}) // add x3, x0, x3
		// x3 = x3 + 8 (skip count)
		if err := acg.out.AddImm64("x3", "x3", 8); err != nil {
			return err
		}

		// Load element into d3
		acg.out.out.writer.WriteBytes([]byte{0x63, 0x00, 0x40, 0xfd}) // ldr d3, [x3]

		// Compare element with search value: fcmp d2, d3
		acg.out.out.writer.WriteBytes([]byte{0x40, 0x20, 0x63, 0x1e})

		// If equal, jump to found
		foundJumpPos := acg.eb.text.Len()
		acg.out.BranchCond("eq", 0) // Placeholder

		// Increment index: x2++
		if err := acg.out.AddImm64("x2", "x2", 1); err != nil {
			return err
		}

		// Jump back to loop start
		loopBackOffset := int32(loopStartPos - (acg.eb.text.Len() + 4))
		acg.out.Branch(loopBackOffset)

		// Not found: return 0.0
		notFoundPos := acg.eb.text.Len()
		acg.out.out.writer.WriteBytes([]byte{0xe0, 0x03, 0x67, 0x9e}) // fmov d0, xzr

		// Jump to end
		endJumpPos := acg.eb.text.Len()
		acg.out.Branch(0) // Placeholder

		// Found: return 1.0
		foundPos := acg.eb.text.Len()
		if err := acg.out.MovImm64("x0", 1); err != nil {
			return err
		}
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e}) // scvtf d0, x0

		// End
		endPos := acg.eb.text.Len()

		// Clean up stack
		acg.out.AddImm64("sp", "sp", 16)

		// Patch jumps
		acg.patchJumpOffset(notFoundJumpPos, int32(notFoundPos-notFoundJumpPos))
		acg.patchJumpOffset(foundJumpPos, int32(foundPos-foundJumpPos))
		acg.patchJumpOffset(endJumpPos, int32(endPos-endJumpPos))

	default:
		return fmt.Errorf("unsupported expression type for ARM64: %T", expr)
	}

	return nil
}

// compileAssignment compiles an assignment statement
func (acg *ARM64CodeGen) compileAssignment(assign *AssignStmt) error {
	// Validate assignment semantics
	_, exists := acg.stackVars[assign.Name]
	isMutable := acg.mutableVars[assign.Name]

	if assign.IsUpdate {
		// <- Update existing mutable variable
		if !exists {
			return fmt.Errorf("cannot update undefined variable '%s'", assign.Name)
		}
		if !isMutable {
			return fmt.Errorf("cannot update immutable variable '%s' (use <- only for mutable variables)", assign.Name)
		}
	} else if assign.Mutable {
		// := Define mutable variable
		if exists {
			return fmt.Errorf("variable '%s' already defined (use <- to update)", assign.Name)
		}
	} else {
		// = Define immutable variable (can shadow existing immutable, but not mutable)
		if exists && isMutable {
			return fmt.Errorf("cannot shadow mutable variable '%s' with immutable variable", assign.Name)
		}
	}

	// Compile the value
	if err := acg.compileExpression(assign.Value); err != nil {
		return err
	}

	var offset int32
	if assign.IsUpdate {
		// <- Update existing mutable variable - look up its offset
		stackOffset := acg.stackVars[assign.Name]
		offset = int32(16 + stackOffset - 8)
	} else {
		// = or := - Allocate stack space for new variable (8-byte aligned)
		// This includes shadowing for immutable variables
		// Variables are stored at positive offsets from frame pointer
		acg.stackSize += 8
		acg.stackVars[assign.Name] = acg.stackSize
		acg.mutableVars[assign.Name] = assign.Mutable
		// x29 points to saved fp location, variables start at offset 16
		offset = int32(16 + acg.stackSize - 8)
	}

	// Store result on stack: str d0, [x29, #offset]
	return acg.out.StrImm64Double("d0", "x29", offset)
}

// compileMatchExpr compiles a match expression (if/else equivalent)
func (acg *ARM64CodeGen) compileMatchExpr(expr *MatchExpr) error {
	// Compile the condition expression (result in d0)
	if err := acg.compileExpression(expr.Condition); err != nil {
		return err
	}

	// Determine the jump condition based on the condition type
	var jumpCond string
	needsZeroCompare := false

	// Check if condition is a comparison (we can use the flags directly)
	if binExpr, ok := expr.Condition.(*BinaryExpr); ok {
		switch binExpr.Operator {
		case "<":
			jumpCond = "ge" // Jump if NOT less than (>=)
		case "<=":
			jumpCond = "gt" // Jump if NOT less or equal (>)
		case ">":
			jumpCond = "le" // Jump if NOT greater than (<=)
		case ">=":
			jumpCond = "lt" // Jump if NOT greater or equal (<)
		case "==":
			jumpCond = "ne" // Jump if NOT equal (!=)
		case "!=":
			jumpCond = "eq" // Jump if NOT not-equal (==)
		default:
			needsZeroCompare = true
		}
	} else {
		needsZeroCompare = true
	}

	// If not a direct comparison, compare d0 with 0.0
	if needsZeroCompare {
		// fmov d1, #0.0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x60, 0x1e}) // fmov d1, #0.0
		// fcmp d0, d1
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
		jumpCond = "eq" // Jump to default if condition is false (== 0.0)
	}

	// Save position for default jump (will be patched later)
	defaultJumpPos := acg.eb.text.Len()
	// Emit placeholder conditional branch to default
	acg.out.BranchCond(jumpCond, 0) // 4 bytes

	// Track positions for end jumps
	var endJumpPositions []int

	// Compile match clauses (only support simple -> result for now)
	if len(expr.Clauses) > 0 {
		for _, clause := range expr.Clauses {
			// For now, skip guard support (simplified implementation)
			if clause.Guard != nil {
				return fmt.Errorf("match guards not yet supported for ARM64")
			}

			// Compile the result expression
			if clause.Result != nil {
				if err := acg.compileExpression(clause.Result); err != nil {
					return err
				}
			}

			// Jump to end after executing this clause
			endJumpPos := acg.eb.text.Len()
			acg.out.Branch(0) // Unconditional branch to end (4 bytes)
			endJumpPositions = append(endJumpPositions, endJumpPos)
		}
	}

	// Default clause position
	defaultPos := acg.eb.text.Len()

	// Compile default expression if present
	if expr.DefaultExpr != nil {
		if err := acg.compileExpression(expr.DefaultExpr); err != nil {
			return err
		}
	} else if len(expr.Clauses) == 0 {
		// No clauses and no default - preserve condition value
		// d0 already has the condition value
	} else {
		// Default is 0.0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x60, 0x1e}) // fmov d0, #0.0
	}

	// End position
	endPos := acg.eb.text.Len()

	// Patch default jump
	defaultOffset := int32(defaultPos - defaultJumpPos)
	acg.patchJumpOffset(defaultJumpPos, defaultOffset)

	// Patch all end jumps
	for _, jumpPos := range endJumpPositions {
		offset := int32(endPos - jumpPos)
		acg.patchJumpOffset(jumpPos, offset)
	}

	return nil
}

// patchJumpOffset patches a branch instruction's offset
func (acg *ARM64CodeGen) patchJumpOffset(pos int, offset int32) {
	// ARM64 branch offsets are in words (4 bytes), not bytes
	if offset%4 != 0 {
		// Offset not aligned - this shouldn't happen but handle gracefully
		offset = (offset >> 2) << 2
	}

	imm := offset >> 2 // Convert to word offset

	textBytes := acg.eb.text.Bytes()

	// Read existing instruction
	instr := uint32(textBytes[pos]) | (uint32(textBytes[pos+1]) << 8) |
		(uint32(textBytes[pos+2]) << 16) | (uint32(textBytes[pos+3]) << 24)

	// Check if it's a conditional branch (B.cond) or unconditional branch (B)
	if (instr & 0xff000010) == 0x54000000 {
		// Conditional branch: B.cond - imm19 at bits [23:5]
		instr = (instr & 0xff00001f) | ((uint32(imm) & 0x7ffff) << 5)
	} else if (instr & 0xfc000000) == 0x14000000 {
		// Unconditional branch: B - imm26 at bits [25:0]
		instr = (instr & 0xfc000000) | (uint32(imm) & 0x3ffffff)
	}

	// Write back patched instruction
	textBytes[pos] = byte(instr)
	textBytes[pos+1] = byte(instr >> 8)
	textBytes[pos+2] = byte(instr >> 16)
	textBytes[pos+3] = byte(instr >> 24)
}

// compileCall compiles a function call
func (acg *ARM64CodeGen) compileCall(call *CallExpr) error {
	switch call.Function {
	case "println":
		return acg.compilePrintln(call)
	case "exit":
		return acg.compileExit(call)
	case "print":
		return acg.compilePrint(call)
	case "getpid":
		return acg.compileGetPid(call)
	case "printf":
		return acg.compilePrintf(call)
	// TODO: Math functions compile but have runtime issues - needs debugging
	// case "sin", "cos", "tan", "asin", "acos", "atan", "sinh", "cosh", "tanh", "exp", "log", "log10", "sqrt", "ceil", "floor", "fabs", "round":
	// 	return acg.compileMathFunction(call)
	// case "pow":
	// 	return acg.compilePowFunction(call)
	default:
		// Check if it's a variable holding a function pointer
		if _, exists := acg.stackVars[call.Function]; exists {
			// Convert to DirectCallExpr and compile
			directCall := &DirectCallExpr{
				Callee: &IdentExpr{Name: call.Function},
				Args:   call.Args,
			}
			return acg.compileDirectCall(directCall)
		}
		return fmt.Errorf("unsupported function for ARM64: %s", call.Function)
	}
}

// compilePrint compiles a print call (without newline)
func (acg *ARM64CodeGen) compilePrint(call *CallExpr) error {
	if len(call.Args) == 0 {
		return fmt.Errorf("print requires an argument")
	}

	arg := call.Args[0]

	switch a := arg.(type) {
	case *StringExpr:
		// Store string in rodata
		label := fmt.Sprintf("str_%d", acg.stringCounter)
		acg.stringCounter++
		content := a.Value // No newline
		acg.eb.Define(label, content)

		// mov x16, #4 (write syscall)
		if err := acg.out.MovImm64("x16", 4); err != nil {
			return err
		}

		// mov x0, #1 (stdout)
		if err := acg.out.MovImm64("x0", 1); err != nil {
			return err
		}

		// Load string address into x1
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: label,
		})
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0x90}) // ADRP x1, #0
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x00, 0x91}) // ADD x1, x1, #0

		// mov x2, length
		if err := acg.out.MovImm64("x2", uint64(len(content))); err != nil {
			return err
		}

		// svc #0x80 (macOS syscall)
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x10, 0x00, 0xd4})

	default:
		return fmt.Errorf("unsupported print argument type for ARM64: %T", arg)
	}

	return nil
}

// compilePrintln compiles a println call
func (acg *ARM64CodeGen) compilePrintln(call *CallExpr) error {
	if len(call.Args) == 0 {
		return fmt.Errorf("println requires an argument")
	}

	arg := call.Args[0]

	// For string literals, use syscall directly (more efficient)
	if strExpr, ok := arg.(*StringExpr); ok {
		// Store string in rodata
		label := fmt.Sprintf("str_%d", acg.stringCounter)
		acg.stringCounter++
		content := strExpr.Value + "\n"
		acg.eb.Define(label, content)

		// mov x16, #4 (write syscall)
		if err := acg.out.MovImm64("x16", 4); err != nil {
			return err
		}

		// mov x0, #1 (stdout)
		if err := acg.out.MovImm64("x0", 1); err != nil {
			return err
		}

		// Load string address into x1
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: label,
		})
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0x90}) // ADRP x1, #0
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x00, 0x91}) // ADD x1, x1, #0

		// mov x2, length
		if err := acg.out.MovImm64("x2", uint64(len(content))); err != nil {
			return err
		}

		// svc #0x80 (macOS syscall)
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x10, 0x00, 0xd4})

		return nil
	}

	// For everything else (numbers, expressions), use printf("%g\n", value)
	// This reuses the existing printf implementation which handles all types correctly

	// Create a synthetic printf call with format "%g\n" and the original argument
	formatStr := &StringExpr{Value: "%g\n"}
	printfCall := &CallExpr{
		Function: "printf",
		Args:     []Expression{formatStr, arg},
	}

	return acg.compilePrintf(printfCall)
}

// compileLoopStatement compiles a loop statement
func (acg *ARM64CodeGen) compileLoopStatement(stmt *LoopStmt) error {
	// Check if iterating over range() or a list
	funcCall, isRangeCall := stmt.Iterable.(*CallExpr)
	if isRangeCall && funcCall.Function == "range" && len(funcCall.Args) == 1 {
		// Range loop
		return acg.compileRangeLoop(stmt, funcCall)
	} else {
		// List iteration
		return acg.compileListLoop(stmt)
	}
}

// compileRangeLoop compiles a range-based loop (@+ i in range(10) { ... })
func (acg *ARM64CodeGen) compileRangeLoop(stmt *LoopStmt, funcCall *CallExpr) error {
	// Increment label counter for uniqueness
	acg.labelCounter++

	// Evaluate the loop limit (argument to range())
	if err := acg.compileExpression(funcCall.Args[0]); err != nil {
		return err
	}

	// Convert d0 (float64) to integer in x0: fcvtzs x0, d0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

	// Allocate stack space for loop limit
	acg.stackSize += 8
	limitOffset := acg.stackSize
	// Store limit: str x0, [x29, #offset] (positive offset from x29)
	offset := int32(16 + limitOffset - 8)
	if err := acg.out.StrImm64("x0", "x29", offset); err != nil {
		return err
	}

	// Allocate stack space for iterator variable
	acg.stackSize += 8
	iterOffset := acg.stackSize
	acg.stackVars[stmt.Iterator] = iterOffset

	// Initialize iterator to 0.0
	// mov x0, #0
	if err := acg.out.MovImm64("x0", 0); err != nil {
		return err
	}
	// scvtf d0, x0 (convert to float64)
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
	// Store iterator: str d0, [x29, #offset] (positive offset from x29)
	offset = int32(16 + iterOffset - 8)
	if err := acg.out.StrImm64Double("d0", "x29", offset); err != nil {
		return err
	}

	// Loop start label
	loopStartPos := acg.eb.text.Len()

	// Register this loop on the active loop stack
	loopLabel := len(acg.activeLoops) + 1
	loopInfo := ARM64LoopInfo{
		Label:            loopLabel,
		StartPos:         loopStartPos,
		EndPatches:       []int{},
		ContinuePatches:  []int{},
		IteratorOffset:   iterOffset,
		UpperBoundOffset: limitOffset,
		IsRangeLoop:      true,
	}
	acg.activeLoops = append(acg.activeLoops, loopInfo)

	// Load iterator value as float: ldr d0, [x29, #offset] (positive offset)
	offset = int32(16 + iterOffset - 8)
	if err := acg.out.LdrImm64Double("d0", "x29", offset); err != nil {
		return err
	}

	// Convert iterator to integer for comparison: fcvtzs x0, d0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

	// Load limit value: ldr x1, [x29, #offset] (positive offset)
	offset = int32(16 + limitOffset - 8)
	if err := acg.out.LdrImm64("x1", "x29", offset); err != nil {
		return err
	}

	// Compare iterator with limit: cmp x0, x1
	acg.out.out.writer.WriteBytes([]byte{0x1f, 0x00, 0x01, 0xeb}) // cmp x0, x1

	// Jump to loop end if iterator >= limit
	loopEndJumpPos := acg.eb.text.Len()
	acg.out.BranchCond("ge", 0) // Placeholder

	// Add this to the loop's end patches
	acg.activeLoops[len(acg.activeLoops)-1].EndPatches = append(
		acg.activeLoops[len(acg.activeLoops)-1].EndPatches,
		loopEndJumpPos,
	)

	// Compile loop body
	for _, s := range stmt.Body {
		if err := acg.compileStatement(s); err != nil {
			return err
		}
	}

	// Mark continue position (increment step)
	continuePos := acg.eb.text.Len()
	acg.activeLoops[len(acg.activeLoops)-1].ContinuePos = continuePos

	// Patch all continue jumps to point here
	for _, patchPos := range acg.activeLoops[len(acg.activeLoops)-1].ContinuePatches {
		offset := int32(continuePos - patchPos)
		acg.patchJumpOffset(patchPos, offset)
	}

	// Increment iterator (add 1.0 to float64 value)
	// ldr d0, [x29, #offset] (positive offset)
	offset = int32(16 + iterOffset - 8)
	if err := acg.out.LdrImm64Double("d0", "x29", offset); err != nil {
		return err
	}
	// Load 1.0 into d1
	// mov x0, #1
	if err := acg.out.MovImm64("x0", 1); err != nil {
		return err
	}
	// scvtf d1, x0
	acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x62, 0x9e})
	// fadd d0, d0, d1
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x28, 0x61, 0x1e})
	// Store incremented value: str d0, [x29, #offset] (positive offset)
	offset = int32(16 + iterOffset - 8)
	if err := acg.out.StrImm64Double("d0", "x29", offset); err != nil {
		return err
	}

	// Jump back to loop start
	loopBackJumpPos := acg.eb.text.Len()
	backOffset := int32(loopStartPos - loopBackJumpPos)
	acg.out.Branch(backOffset)

	// Loop end label
	loopEndPos := acg.eb.text.Len()

	// Patch all end jumps
	for _, patchPos := range acg.activeLoops[len(acg.activeLoops)-1].EndPatches {
		endOffset := int32(loopEndPos - patchPos)
		acg.patchJumpOffset(patchPos, endOffset)
	}

	// Pop loop from active stack
	acg.activeLoops = acg.activeLoops[:len(acg.activeLoops)-1]

	return nil
}

// compileListLoop compiles a list iteration loop (@+ elem in [1,2,3] { ... })
func (acg *ARM64CodeGen) compileListLoop(stmt *LoopStmt) error {
	// Increment label counter for uniqueness
	acg.labelCounter++

	// Evaluate the list expression (returns pointer as float64 in d0)
	if err := acg.compileExpression(stmt.Iterable); err != nil {
		return err
	}

	// Save list pointer to stack
	acg.stackSize += 8
	listPtrOffset := acg.stackSize
	offset := int32(16 + listPtrOffset - 8)
	if err := acg.out.StrImm64Double("d0", "x29", offset); err != nil {
		return err
	}

	// Convert pointer from float64 to integer in x0: fcvtzs x0, d0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

	// Load list length from [x0] (first 8 bytes)
	// ldr d0, [x0]
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x40, 0xfd})

	// Convert length to integer: fcvtzs x0, d0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

	// Store length in stack
	acg.stackSize += 8
	lengthOffset := acg.stackSize
	offset = int32(16 + lengthOffset - 8)
	if err := acg.out.StrImm64("x0", "x29", offset); err != nil {
		return err
	}

	// Allocate stack space for index variable
	acg.stackSize += 8
	indexOffset := acg.stackSize
	// Initialize index to 0: mov x0, #0
	if err := acg.out.MovImm64("x0", 0); err != nil {
		return err
	}
	offset = int32(16 + indexOffset - 8)
	if err := acg.out.StrImm64("x0", "x29", offset); err != nil {
		return err
	}

	// Allocate stack space for iterator variable (the actual value from the list)
	acg.stackSize += 8
	iterOffset := acg.stackSize
	acg.stackVars[stmt.Iterator] = iterOffset

	// Loop start label
	loopStartPos := acg.eb.text.Len()

	// Register this loop on the active loop stack
	loopLabel := len(acg.activeLoops) + 1
	loopInfo := ARM64LoopInfo{
		Label:            loopLabel,
		StartPos:         loopStartPos,
		EndPatches:       []int{},
		ContinuePatches:  []int{},
		IteratorOffset:   iterOffset,
		IndexOffset:      indexOffset,
		UpperBoundOffset: lengthOffset,
		ListPtrOffset:    listPtrOffset,
		IsRangeLoop:      false,
	}
	acg.activeLoops = append(acg.activeLoops, loopInfo)

	// Load index: ldr x0, [x29, #offset] (positive offset)
	offset = int32(16 + indexOffset - 8)
	if err := acg.out.LdrImm64("x0", "x29", offset); err != nil {
		return err
	}

	// Load length: ldr x1, [x29, #offset] (positive offset)
	offset = int32(16 + lengthOffset - 8)
	if err := acg.out.LdrImm64("x1", "x29", offset); err != nil {
		return err
	}

	// Compare index with length: cmp x0, x1
	acg.out.out.writer.WriteBytes([]byte{0x1f, 0x00, 0x01, 0xeb}) // cmp x0, x1

	// Jump to loop end if index >= length
	loopEndJumpPos := acg.eb.text.Len()
	acg.out.BranchCond("ge", 0) // Placeholder

	// Add this to the loop's end patches
	acg.activeLoops[len(acg.activeLoops)-1].EndPatches = append(
		acg.activeLoops[len(acg.activeLoops)-1].EndPatches,
		loopEndJumpPos,
	)

	// Load list pointer from stack to x2
	offset = int32(16 + listPtrOffset - 8)
	if err := acg.out.LdrImm64Double("d0", "x29", offset); err != nil {
		return err
	}
	// Convert to integer: fcvtzs x2, d0
	acg.out.out.writer.WriteBytes([]byte{0x02, 0x00, 0x78, 0x9e})

	// Skip length prefix: x2 += 8
	if err := acg.out.AddImm64("x2", "x2", 8); err != nil {
		return err
	}

	// Load index into x0
	offset = int32(16 + indexOffset - 8)
	if err := acg.out.LdrImm64("x0", "x29", offset); err != nil {
		return err
	}

	// Calculate offset: x0 = x0 << 3 (x0 * 8)
	acg.out.out.writer.WriteBytes([]byte{0x00, 0xf0, 0x7d, 0xd3}) // lsl x0, x0, #3

	// Add to base: x2 = x2 + x0
	acg.out.out.writer.WriteBytes([]byte{0x42, 0x00, 0x00, 0x8b}) // add x2, x2, x0

	// Load element value: ldr d0, [x2]
	acg.out.out.writer.WriteBytes([]byte{0x40, 0x00, 0x40, 0xfd}) // ldr d0, [x2]

	// Store iterator value: str d0, [x29, #offset] (positive offset)
	offset = int32(16 + iterOffset - 8)
	if err := acg.out.StrImm64Double("d0", "x29", offset); err != nil {
		return err
	}

	// Compile loop body
	for _, s := range stmt.Body {
		if err := acg.compileStatement(s); err != nil {
			return err
		}
	}

	// Mark continue position (increment step)
	continuePos := acg.eb.text.Len()
	acg.activeLoops[len(acg.activeLoops)-1].ContinuePos = continuePos

	// Patch all continue jumps to point here
	for _, patchPos := range acg.activeLoops[len(acg.activeLoops)-1].ContinuePatches {
		offset := int32(continuePos - patchPos)
		acg.patchJumpOffset(patchPos, offset)
	}

	// Increment index
	offset = int32(16 + indexOffset - 8)
	if err := acg.out.LdrImm64("x0", "x29", offset); err != nil {
		return err
	}
	if err := acg.out.AddImm64("x0", "x0", 1); err != nil {
		return err
	}
	offset = int32(16 + indexOffset - 8)
	if err := acg.out.StrImm64("x0", "x29", offset); err != nil {
		return err
	}

	// Jump back to loop start
	loopBackJumpPos := acg.eb.text.Len()
	backOffset := int32(loopStartPos - loopBackJumpPos)
	acg.out.Branch(backOffset)

	// Loop end label
	loopEndPos := acg.eb.text.Len()

	// Patch all end jumps
	for _, patchPos := range acg.activeLoops[len(acg.activeLoops)-1].EndPatches {
		endOffset := int32(loopEndPos - patchPos)
		acg.patchJumpOffset(patchPos, endOffset)
	}

	// Pop loop from active stack
	acg.activeLoops = acg.activeLoops[:len(acg.activeLoops)-1]

	return nil
}

// compileExit compiles an exit call via dynamic linking
func (acg *ARM64CodeGen) compileExit(call *CallExpr) error {
	exitCode := uint64(0)

	// Evaluate exit code argument
	if len(call.Args) > 0 {
		if num, ok := call.Args[0].(*NumberExpr); ok {
			exitCode = uint64(int64(num.Value))
		} else {
			// Compile expression and convert to integer
			if err := acg.compileExpression(call.Args[0]); err != nil {
				return err
			}
			// Convert d0 to integer in x0: fcvtzs x0, d0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})
			// x0 now contains exit code, ready for function call
			// Skip the constant load below
			goto callExit
		}
	}

	// Load constant exit code into x0 (first argument register for ARM64)
	if err := acg.out.MovImm64("x0", exitCode); err != nil {
		return err
	}

callExit:
	// Mark that we need dynamic linking
	acg.eb.useDynamicLinking = true

	// Add exit to needed functions list if not already there
	funcName := "exit" // Note: macho.go will add underscore prefix for macOS
	found := false
	for _, f := range acg.eb.neededFunctions {
		if f == funcName {
			found = true
			break
		}
	}
	if !found {
		acg.eb.neededFunctions = append(acg.eb.neededFunctions, funcName)
	}

	// Generate call to exit stub
	stubLabel := funcName + "$stub"
	position := acg.eb.text.Len()
	acg.eb.callPatches = append(acg.eb.callPatches, CallPatch{
		position:   position,
		targetName: stubLabel,
	})

	// Emit placeholder bl instruction (will be patched)
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x94}) // bl #0

	// exit() doesn't return, but we'll never reach here anyway
	return nil
}

// compileDirectCall compiles a direct function call (e.g., lambda invocation)
func (acg *ARM64CodeGen) compileDirectCall(call *DirectCallExpr) error {
	// Compile the callee expression (e.g., a lambda) to get function pointer
	// Result in d0 (function pointer as float64)
	if err := acg.compileExpression(call.Callee); err != nil {
		return err
	}

	// Convert function pointer from float64 to integer in x0
	// fcvtzs x0, d0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

	// Save function pointer to stack (x0 might get clobbered during arg evaluation)
	acg.out.SubImm64("sp", "sp", 16)
	if err := acg.out.StrImm64("x0", "sp", 0); err != nil {
		return err
	}

	// Evaluate all arguments and save to stack
	for _, arg := range call.Args {
		if err := acg.compileExpression(arg); err != nil {
			return err
		}
		// Result in d0, save to stack
		acg.out.SubImm64("sp", "sp", 8)
		// str d0, [sp]
		acg.out.out.writer.WriteBytes([]byte{0xe0, 0x03, 0x00, 0xfd})
	}

	// Load arguments from stack into d0-d7 registers (in reverse order)
	// ARM64 AAPCS64 passes float args in d0-d7
	if len(call.Args) > 8 {
		return fmt.Errorf("too many arguments to direct call (max 8)")
	}

	for i := len(call.Args) - 1; i >= 0; i-- {
		// ldr dN, [sp]
		regNum := uint32(i)
		instr := uint32(0xfd400000) | (regNum) | (31 << 5) // ldr dN, [sp, #0]
		acg.out.out.writer.WriteBytes([]byte{
			byte(instr),
			byte(instr >> 8),
			byte(instr >> 16),
			byte(instr >> 24),
		})
		acg.out.AddImm64("sp", "sp", 8)
	}

	// Load function pointer from stack to x16 (temporary register)
	if err := acg.out.LdrImm64("x16", "sp", 0); err != nil {
		return err
	}
	if err := acg.out.AddImm64("sp", "sp", 16); err != nil {
		return err
	}

	// Call the function pointer in x16: blr x16
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x02, 0x3f, 0xd6})

	// Result is in d0
	return nil
}

// compilePrintf compiles a printf() call via dynamic linking
func (acg *ARM64CodeGen) compilePrintf(call *CallExpr) error {
	if len(call.Args) == 0 {
		return fmt.Errorf("printf requires at least a format string")
	}

	// First argument must be a string (format string)
	formatArg := call.Args[0]
	strExpr, ok := formatArg.(*StringExpr)
	if !ok {
		return fmt.Errorf("printf first argument must be a string literal")
	}

	// Store format string in rodata
	labelName := fmt.Sprintf("str_%d", acg.stringCounter)
	acg.stringCounter++

	// Add null terminator for C string
	formatStr := strExpr.Value + "\x00"
	acg.eb.Define(labelName, formatStr)

	// Mark that we need dynamic linking
	acg.eb.useDynamicLinking = true

	// Add printf to needed functions list if not already there
	funcName := "printf" // Note: macho.go will add underscore prefix for macOS
	found := false
	for _, f := range acg.eb.neededFunctions {
		if f == funcName {
			found = true
			break
		}
	}
	if !found {
		acg.eb.neededFunctions = append(acg.eb.neededFunctions, funcName)
	}

	// Compile additional arguments and push them onto the stack
	// For variadic functions like printf on ARM64, arguments go on the stack
	numArgs := len(call.Args) - 1 // Excluding format string

	// Calculate stack space needed (8 bytes per argument, 16-byte aligned)
	stackSize := ((numArgs * 8) + 15) &^ 15

	if numArgs > 0 {
		// Allocate stack space: sub sp, sp, #stackSize
		// Encoding: 0xd10003ff | (stackSize << 10)
		subInstr := uint32(0xd10003ff) | (uint32(stackSize) << 10)
		acg.out.out.writer.WriteBytes([]byte{
			byte(subInstr),
			byte(subInstr >> 8),
			byte(subInstr >> 16),
			byte(subInstr >> 24),
		})

		// Compile each argument and store on stack (in reverse order to match x86-64 behavior)
		for i := numArgs - 1; i >= 0; i-- {
			arg := call.Args[i+1]

			// Check if this is a string argument
			if strExpr, ok := arg.(*StringExpr); ok {
				// String argument - need to pass pointer, not float
				// Store string in rodata and get its address
				strLabel := fmt.Sprintf("str_%d", acg.stringCounter)
				acg.stringCounter++
				strValue := strExpr.Value + "\x00" // Add null terminator
				acg.eb.Define(strLabel, strValue)

				// Load string address into x9 using PC-relative addressing
				strOffset := uint64(acg.eb.text.Len())
				acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
					offset:     strOffset,
					symbolName: strLabel,
				})
				// ADRP x9, label@PAGE
				acg.out.out.writer.WriteBytes([]byte{0x09, 0x00, 0x00, 0x90})
				// ADD x9, x9, label@PAGEOFF
				acg.out.out.writer.WriteBytes([]byte{0x29, 0x01, 0x00, 0x91})

				// Store x9 (pointer) at [sp, #offset]
				// STR x9, [sp, #offset] - encoding: 0xf9000000 | (offset/8 << 10) | (9 << 0) | (31 << 5)
				// Use (numArgs-1-i) to reverse the order: first evaluated (last arg) goes to highest offset
				stackOffset := uint32((numArgs - 1 - i) * 8)
				strInstr := uint32(0xf90003e9) | ((stackOffset / 8) << 10)
				acg.out.out.writer.WriteBytes([]byte{
					byte(strInstr),
					byte(strInstr >> 8),
					byte(strInstr >> 16),
					byte(strInstr >> 24),
				})
			} else {
				// Numeric argument - compile and store as float
				if err := acg.compileExpression(arg); err != nil {
					return err
				}

				// Store d0 at [sp, #(i*8)]
				// STR d0, [sp, #offset] - encoding: 0xfd000000 | (offset/8 << 10) | 0x3e0
				// Use (numArgs-1-i) to reverse the order: first evaluated (last arg) goes to highest offset
				stackOffset := uint32((numArgs - 1 - i) * 8)
				strInstr := uint32(0xfd0003e0) | ((stackOffset / 8) << 10)
				acg.out.out.writer.WriteBytes([]byte{
					byte(strInstr),
					byte(strInstr >> 8),
					byte(strInstr >> 16),
					byte(strInstr >> 24),
				})
			}
		}
	}

	// Load format string address into x0 (first argument register for ARM64)
	offset := uint64(acg.eb.text.Len())
	acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
		offset:     offset,
		symbolName: labelName,
	})
	// ADRP x0, label@PAGE
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x90})
	// ADD x0, x0, label@PAGEOFF
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x91})

	// Generate call to printf stub
	stubLabel := funcName + "$stub"
	position := acg.eb.text.Len()
	acg.eb.callPatches = append(acg.eb.callPatches, CallPatch{
		position:   position,
		targetName: stubLabel,
	})

	// Emit placeholder bl instruction (will be patched)
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x94}) // bl #0

	// Clean up stack if we allocated space for arguments
	if numArgs > 0 {
		// Restore stack: add sp, sp, #stackSize
		// Encoding: 0x910003ff | (stackSize << 10)
		addInstr := uint32(0x910003ff) | (uint32(stackSize) << 10)
		acg.out.out.writer.WriteBytes([]byte{
			byte(addInstr),
			byte(addInstr >> 8),
			byte(addInstr >> 16),
			byte(addInstr >> 24),
		})
	}

	// printf returns int in x0, convert to float64 in d0
	// scvtf d0, x0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})

	return nil
}

// compileMathFunction compiles a call to a C math library function (sin, cos, sqrt, etc.)
func (acg *ARM64CodeGen) compileMathFunction(call *CallExpr) error {
	if len(call.Args) != 1 {
		return fmt.Errorf("%s requires exactly 1 argument", call.Function)
	}

	// Compile the argument - result will be in d0
	if err := acg.compileExpression(call.Args[0]); err != nil {
		return err
	}

	// Argument is already in d0 (ARM64 ABI: first float arg in d0)

	// Mark that we need dynamic linking
	acg.eb.useDynamicLinking = true

	// Add function to needed functions list if not already there
	funcName := call.Function // e.g., "sin", "cos", "sqrt"
	found := false
	for _, f := range acg.eb.neededFunctions {
		if f == funcName {
			found = true
			break
		}
	}
	if !found {
		acg.eb.neededFunctions = append(acg.eb.neededFunctions, funcName)
	}

	// Generate call to function stub
	stubLabel := funcName + "$stub"
	position := acg.eb.text.Len()
	acg.eb.callPatches = append(acg.eb.callPatches, CallPatch{
		position:   position,
		targetName: stubLabel,
	})

	// Emit placeholder bl instruction (will be patched later)
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x94}) // bl #0

	// Result is returned in d0 (ARM64 ABI: float return value in d0)
	// No conversion needed, d0 already has the result

	return nil
}

// compilePowFunction compiles a call to pow(x, y)
func (acg *ARM64CodeGen) compilePowFunction(call *CallExpr) error {
	if len(call.Args) != 2 {
		return fmt.Errorf("pow requires exactly 2 arguments")
	}

	// Compile first argument (base) - result will be in d0
	if err := acg.compileExpression(call.Args[0]); err != nil {
		return err
	}

	// Save first argument to d1 temporarily (we'll move it back)
	// fmov d8, d0 (use callee-saved register d8)
	acg.out.out.writer.WriteBytes([]byte{0x08, 0x40, 0x60, 0x1e})

	// Compile second argument (exponent) - result will be in d0
	if err := acg.compileExpression(call.Args[1]); err != nil {
		return err
	}

	// Move second argument to d1 (ARM64 ABI: second float arg in d1)
	// fmov d1, d0
	acg.out.out.writer.WriteBytes([]byte{0x01, 0x40, 0x60, 0x1e})

	// Move first argument back to d0 (ARM64 ABI: first float arg in d0)
	// fmov d0, d8
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x41, 0x60, 0x1e})

	// Mark that we need dynamic linking
	acg.eb.useDynamicLinking = true

	// Add pow to needed functions list
	funcName := "pow"
	found := false
	for _, f := range acg.eb.neededFunctions {
		if f == funcName {
			found = true
			break
		}
	}
	if !found {
		acg.eb.neededFunctions = append(acg.eb.neededFunctions, funcName)
	}

	// Generate call to pow stub
	stubLabel := funcName + "$stub"
	position := acg.eb.text.Len()
	acg.eb.callPatches = append(acg.eb.callPatches, CallPatch{
		position:   position,
		targetName: stubLabel,
	})

	// Emit placeholder bl instruction
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x94}) // bl #0

	// Result is returned in d0
	return nil
}

// compileGetPid compiles a getpid() call via dynamic linking
func (acg *ARM64CodeGen) compileGetPid(call *CallExpr) error {
	// Mark that we need dynamic linking
	acg.eb.useDynamicLinking = true

	// Add getpid to needed functions list if not already there
	funcName := "getpid" // Note: macho.go will add underscore prefix for macOS
	found := false
	for _, f := range acg.eb.neededFunctions {
		if f == funcName {
			found = true
			break
		}
	}
	if !found {
		acg.eb.neededFunctions = append(acg.eb.neededFunctions, funcName)
	}

	// Generate a call through the stub
	// We'll create a stub for each imported function
	// For now, use PC-relative branch placeholder (will need stub generation later)
	stubLabel := funcName + "$stub"

	// bl stub (branch with link)
	// This is a placeholder - we'll need to patch it with actual stub address
	position := acg.eb.text.Len()
	acg.eb.callPatches = append(acg.eb.callPatches, CallPatch{
		position:   position,
		targetName: stubLabel,
	})

	// Emit placeholder bl instruction (will be patched)
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x00, 0x94}) // bl #0

	// Result is in x0 (integer), convert to float64 in d0
	// scvtf d0, x0
	acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})

	return nil
}

// generateLambdaFunctions generates code for all lambda functions
func (acg *ARM64CodeGen) generateLambdaFunctions() error {
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG generateLambdaFunctions: generating %d lambdas\n", len(acg.lambdaFuncs))
	}

	for _, lambda := range acg.lambdaFuncs {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG generateLambdaFunctions: generating lambda '%s'\n", lambda.Name)
		}

		// Mark the start of the lambda function with a label
		acg.eb.MarkLabel(lambda.Name)

		// Function prologue - ARM64 ABI
		// Save frame pointer and link register
		if err := acg.out.SubImm64("sp", "sp", 32); err != nil {
			return err
		}
		if err := acg.out.StrImm64("x29", "sp", 0); err != nil {
			return err
		}
		if err := acg.out.StrImm64("x30", "sp", 8); err != nil {
			return err
		}
		// Set frame pointer
		if err := acg.out.AddImm64("x29", "sp", 0); err != nil {
			return err
		}

		// Save previous state
		oldStackVars := acg.stackVars
		oldStackSize := acg.stackSize
		oldCurrentLambda := acg.currentLambda

		// Create new scope for lambda
		acg.stackVars = make(map[string]int)
		acg.stackSize = 0
		acg.currentLambda = &lambda

		// Store parameters from d0-d7 registers to stack
		// Parameters come in d0, d1, d2, d3, d4, d5, d6, d7 (AAPCS64)
		for i, paramName := range lambda.Params {
			if i >= 8 {
				return fmt.Errorf("lambda has too many parameters (max 8)")
			}

			// Allocate stack space for parameter (8 bytes for float64)
			acg.stackSize += 8
			paramOffset := acg.stackSize
			acg.stackVars[paramName] = paramOffset

			// Store parameter from d register to stack
			// str dN, [x29, #-paramOffset]
			regName := fmt.Sprintf("d%d", i)
			if err := acg.out.StrImm64Double(regName, "x29", int32(-paramOffset)); err != nil {
				return err
			}
		}

		// Compile lambda body (result in d0)
		if err := acg.compileExpression(lambda.Body); err != nil {
			return err
		}

		// Clear lambda context
		acg.currentLambda = nil

		// Function epilogue - ARM64 ABI
		// Restore registers and return
		if err := acg.out.LdrImm64("x30", "sp", 8); err != nil {
			return err
		}
		if err := acg.out.LdrImm64("x29", "sp", 0); err != nil {
			return err
		}
		if err := acg.out.AddImm64("sp", "sp", 32); err != nil {
			return err
		}
		if err := acg.out.Return("x30"); err != nil {
			return err
		}

		// Restore previous state
		acg.stackVars = oldStackVars
		acg.stackSize = oldStackSize
		acg.currentLambda = oldCurrentLambda
	}

	return nil
}
