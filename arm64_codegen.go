package main

import (
	"fmt"
	"os"
	"strconv"
	"unsafe"
)

// ARM64CodeGen handles ARM64 code generation for macOS
type ARM64CodeGen struct {
	out           *ARM64Out
	eb            *ExecutableBuilder
	stackVars     map[string]int // variable name -> stack offset from fp
	stackSize     int            // current stack size
	stringCounter int            // counter for string labels
	labelCounter  int            // counter for jump labels
}

// NewARM64CodeGen creates a new ARM64 code generator
func NewARM64CodeGen(eb *ExecutableBuilder) *ARM64CodeGen {
	return &ARM64CodeGen{
		out:           &ARM64Out{out: &Out{machine: Platform{Arch: ArchARM64, OS: OSDarwin}, writer: eb.TextWriter(), eb: eb}},
		eb:            eb,
		stackVars:     make(map[string]int),
		stackSize:     0,
		stringCounter: 0,
		labelCounter:  0,
	}
}

// CompileProgram compiles a Flap program to ARM64
func (acg *ARM64CodeGen) CompileProgram(program *Program) error {
	// Function prologue - follow ARM64 ABI
	// Allocate initial stack frame
	if err := acg.out.SubImm64("sp", "sp", 32); err != nil {
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
	if err := acg.out.AddImm64("sp", "sp", 32); err != nil {
		return err
	}
	if err := acg.out.Return("x30"); err != nil {
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
		labelName := fmt.Sprintf("str_%d", acg.stringCounter)
		acg.stringCounter++

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
		offset, exists := acg.stackVars[e.Name]
		if !exists {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Error: undefined variable '%s'\n", e.Name)
			}
			return fmt.Errorf("undefined variable: %s", e.Name)
		}
		// ldr d0, [x29, #-offset]
		if err := acg.out.LdrImm64Double("d0", "x29", int32(-offset)); err != nil {
			return err
		}

	case *BinaryExpr:
		// Compile left operand (result in d0)
		if err := acg.compileExpression(e.Left); err != nil {
			return err
		}

		// Push d0 onto stack to save left operand
		acg.out.SubImm64("sp", "sp", 8)
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
		acg.out.AddImm64("sp", "sp", 8)

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
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x9f, 0x9a})
			// scvtf d0, x0 (convert to float)
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "!=":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, ne
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x10, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "<":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, lt
			acg.out.out.writer.WriteBytes([]byte{0x00, 0xb0, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case "<=":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, le
			acg.out.out.writer.WriteBytes([]byte{0x00, 0xd0, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case ">":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, gt
			acg.out.out.writer.WriteBytes([]byte{0x00, 0xc0, 0x9f, 0x9a})
			// scvtf d0, x0
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x62, 0x9e})
		case ">=":
			// fcmp d0, d1
			acg.out.out.writer.WriteBytes([]byte{0x00, 0x20, 0x61, 0x1e})
			// cset x0, ge
			acg.out.out.writer.WriteBytes([]byte{0x00, 0xa0, 0x9f, 0x9a})
			// scvtf d0, x0
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

		// Save list pointer
		acg.out.SubImm64("sp", "sp", 8)
		acg.out.StrImm64("x0", "sp", 0)

		// Compile index expression
		if err := acg.compileExpression(e.Index); err != nil {
			return err
		}

		// Convert index from float64 to int64: fcvtzs x1, d0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x78, 0x9e})

		// Restore list pointer
		acg.out.LdrImm64("x0", "sp", 0)
		acg.out.AddImm64("sp", "sp", 8)

		// x0 = list pointer, x1 = index
		// Skip past count (8 bytes) and index by (index * 8)
		acg.out.AddImm64("x0", "x0", 8)
		// x1 = x1 << 3 (multiply by 8)
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x1c, 0x00, 0xd3}) // lsl x1, x1, #3
		// x0 = x0 + x1
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x01, 0x8b}) // add x0, x0, x1

		// Load element into d0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x40, 0xfd}) // ldr d0, [x0]

	case *CallExpr:
		return acg.compileCall(e)

	default:
		return fmt.Errorf("unsupported expression type for ARM64: %T", expr)
	}

	return nil
}

// compileAssignment compiles an assignment statement
func (acg *ARM64CodeGen) compileAssignment(assign *AssignStmt) error {
	// Compile the value
	if err := acg.compileExpression(assign.Value); err != nil {
		return err
	}

	// Allocate stack space for variable (8-byte aligned)
	acg.stackSize += 8
	acg.stackVars[assign.Name] = acg.stackSize

	// Store result on stack: str d0, [x29, #-offset]
	return acg.out.StrImm64Double("d0", "x29", int32(-acg.stackSize))
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
	default:
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

		// svc #0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})

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

	switch a := arg.(type) {
	case *StringExpr:
		// Store string in rodata
		label := fmt.Sprintf("str_%d", acg.stringCounter)
		acg.stringCounter++
		content := a.Value + "\n"
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

		// svc #0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})

	case *NumberExpr:
		// For numbers, convert to string and print
		// This is complex - for now, just print the integer part
		numStr := strconv.FormatInt(int64(a.Value), 10) + "\n"

		label := fmt.Sprintf("str_%d", acg.stringCounter)
		acg.stringCounter++
		acg.eb.Define(label, numStr)

		// mov x16, #4
		if err := acg.out.MovImm64("x16", 4); err != nil {
			return err
		}

		// mov x0, #1
		if err := acg.out.MovImm64("x0", 1); err != nil {
			return err
		}

		// Load string address
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: label,
		})
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0x90})
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x00, 0x91})

		// mov x2, length
		if err := acg.out.MovImm64("x2", uint64(len(numStr))); err != nil {
			return err
		}

		// svc #0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})

	default:
		// For other expressions, compile them and then convert the result (in d0) to a string
		if err := acg.compileExpression(arg); err != nil {
			return err
		}

		// d0 contains the result as float64
		// Convert to integer: fcvtzs x0, d0
		acg.out.out.writer.WriteBytes([]byte{0x00, 0x00, 0x78, 0x9e})

		// For now, just convert to decimal string (simplified - only handles positive integers)
		// TODO: Implement proper float-to-string conversion
		// For now, we'll just print "?\n" as a placeholder
		label := fmt.Sprintf("str_%d", acg.stringCounter)
		acg.stringCounter++
		acg.eb.Define(label, "?\n")

		// mov x16, #4
		if err := acg.out.MovImm64("x16", 4); err != nil {
			return err
		}

		// mov x0, #1
		if err := acg.out.MovImm64("x0", 1); err != nil {
			return err
		}

		// Load string address
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: label,
		})
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0x90})
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x00, 0x91})

		// mov x2, 2 (length of "?\n")
		if err := acg.out.MovImm64("x2", 2); err != nil {
			return err
		}

		// svc #0
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})
	}

	return nil
}

// compileExit compiles an exit call
func (acg *ARM64CodeGen) compileExit(call *CallExpr) error {
	exitCode := uint64(0)

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
			// We'll use x0 as exit code below
			// mov x16, #1 (exit syscall)
			if err := acg.out.MovImm64("x16", 1); err != nil {
				return err
			}
			// svc #0
			acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})
			return nil
		}
	}

	// mov x0, exitCode
	if err := acg.out.MovImm64("x0", exitCode); err != nil {
		return err
	}

	// mov x16, #1 (exit syscall)
	if err := acg.out.MovImm64("x16", 1); err != nil {
		return err
	}

	// svc #0
	acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})

	return nil
}
