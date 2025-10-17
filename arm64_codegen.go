package main

import (
	"fmt"
)

// ARM64CodeGen handles ARM64 code generation for macOS
type ARM64CodeGen struct {
	out       *ARM64Out
	eb        *ExecutableBuilder
	stackVars map[string]int // variable name -> stack offset from fp
	stackSize int            // current stack size
}

// NewARM64CodeGen creates a new ARM64 code generator
func NewARM64CodeGen(eb *ExecutableBuilder) *ARM64CodeGen {
	return &ARM64CodeGen{
		out:       &ARM64Out{out: &Out{machine: MachineARM64, writer: eb.TextWriter(), eb: eb}},
		eb:        eb,
		stackVars: make(map[string]int),
		stackSize: 0,
	}
}

// CompileProgram compiles a Flap program to ARM64
func (acg *ARM64CodeGen) CompileProgram(program *Program) error {
	// Function prologue - follow ARM64 ABI
	// stp x29, x30, [sp, #-32]!  // Save frame pointer and link register
	// mov x29, sp                 // Set frame pointer
	// sub sp, sp, #32            // Allocate stack space

	// For now, use simple approach with individual instructions
	// sub sp, sp, #32
	if err := acg.out.SubImm64("sp", "sp", 32); err != nil {
		return err
	}
	// str x29, [sp, #0]
	if err := acg.out.StrImm64("x29", "sp", 0); err != nil {
		return err
	}
	// str x30, [sp, #8]
	if err := acg.out.StrImm64("x30", "sp", 8); err != nil {
		return err
	}
	// add x29, sp, #0 (mov x29, sp)
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
	// mov x0, #0              // Exit code 0
	if err := acg.out.MovImm64("x0", 0); err != nil {
		return err
	}
	// ldr x30, [sp, #8]       // Restore link register
	if err := acg.out.LdrImm64("x30", "sp", 8); err != nil {
		return err
	}
	// ldr x29, [sp, #0]       // Restore frame pointer
	if err := acg.out.LdrImm64("x29", "sp", 0); err != nil {
		return err
	}
	// add sp, sp, #32         // Deallocate stack
	if err := acg.out.AddImm64("sp", "sp", 32); err != nil {
		return err
	}
	// ret
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

// compileExpression compiles an expression
func (acg *ARM64CodeGen) compileExpression(expr Expression) error {
	switch e := expr.(type) {
	case *NumberExpr:
		// Load number into x0
		intVal := int64(e.Value)
		return acg.out.MovImm64("x0", uint64(intVal))

	case *StringExpr:
		// Store string in rodata
		label := fmt.Sprintf("str_%d", len(acg.eb.consts))
		acg.eb.Define(label, e.Value+"\x00") // Null-terminated

		// For now, just load 0 - will need PC-relative addressing later
		return acg.out.MovImm64("x0", 0) // TODO: Load actual address

	case *CallExpr:
		return acg.compileCall(e)

	default:
		return fmt.Errorf("unsupported expression type for ARM64: %T", expr)
	}
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

	// Store result on stack
	// str x0, [x29, #-offset]
	return acg.out.StrImm64("x0", "x29", int32(-acg.stackSize))
}

// compileCall compiles a function call
func (acg *ARM64CodeGen) compileCall(call *CallExpr) error {
	switch call.Function {
	case "println":
		return acg.compilePrintln(call)
	case "exit":
		return acg.compileExit(call)
	default:
		return fmt.Errorf("unsupported function for ARM64: %s", call.Function)
	}
}

// compilePrintln compiles a println call using macOS write syscall
func (acg *ARM64CodeGen) compilePrintln(call *CallExpr) error {
	if len(call.Args) == 0 {
		return fmt.Errorf("println requires an argument")
	}

	arg := call.Args[0]

	switch a := arg.(type) {
	case *StringExpr:
		// Store string in rodata
		label := fmt.Sprintf("str_%d", len(acg.eb.consts))
		content := a.Value + "\n\x00"
		acg.eb.Define(label, content)

		// macOS ARM64 write syscall:
		// x16 = 4 (write syscall number on macOS)
		// x0 = 1 (stdout)
		// x1 = buffer address
		// x2 = length
		// svc #0

		// mov x16, #4          // write syscall on macOS
		if err := acg.out.MovImm64("x16", 4); err != nil {
			return err
		}

		// mov x0, #1           // stdout
		if err := acg.out.MovImm64("x0", 1); err != nil {
			return err
		}

		// Load string address into x1 using PC-relative addressing
		// ADRP x1, symbol@PAGE
		// ADD x1, x1, symbol@PAGEOFF
		offset := uint64(acg.eb.text.Len())
		acg.eb.pcRelocations = append(acg.eb.pcRelocations, PCRelocation{
			offset:     offset,
			symbolName: label,
		})
		// Emit placeholder ADRP + ADD instructions (will be patched later)
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0x90}) // ADRP x1, #0
		acg.out.out.writer.WriteBytes([]byte{0x21, 0x00, 0x00, 0x91}) // ADD x1, x1, #0

		// mov x2, length
		if err := acg.out.MovImm64("x2", uint64(len(content)-1)); err != nil {
			return err
		}

		// svc #0 (syscall)
		acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})

	case *NumberExpr:
		// For numbers, we need to convert to string
		// This is complex - for now, just print a placeholder
		return fmt.Errorf("println(number) not yet implemented for ARM64")

	default:
		return fmt.Errorf("unsupported println argument type for ARM64: %T", arg)
	}

	return nil
}

// compileExit compiles an exit call
func (acg *ARM64CodeGen) compileExit(call *CallExpr) error {
	exitCode := uint64(0)

	if len(call.Args) > 0 {
		if num, ok := call.Args[0].(*NumberExpr); ok {
			exitCode = uint64(int64(num.Value))
		}
	}

	// macOS ARM64 exit syscall:
	// x16 = 1 (exit syscall number on macOS)
	// x0 = exit code
	// svc #0

	// mov x0, exitCode
	if err := acg.out.MovImm64("x0", exitCode); err != nil {
		return err
	}

	// mov x16, #1  // exit syscall on macOS
	if err := acg.out.MovImm64("x16", 1); err != nil {
		return err
	}

	// svc #0 (syscall)
	acg.out.out.writer.WriteBytes([]byte{0x01, 0x00, 0x00, 0xd4})

	return nil
}
