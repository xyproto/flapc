package main

import (
	"fmt"
)

// RiscvCodeGen handles RISC-V64 code generation
type RiscvCodeGen struct {
	out       *RiscvOut
	eb        *ExecutableBuilder
	stackVars map[string]int // variable name -> stack offset from fp
	stackSize int            // current stack size
}

// NewRiscvCodeGen creates a new RISC-V64 code generator
func NewRiscvCodeGen(eb *ExecutableBuilder) *RiscvCodeGen {
	return &RiscvCodeGen{
		out:       &RiscvOut{out: &Out{machine: Platform{Arch: ArchRiscv64, OS: OSLinux}, writer: eb.TextWriter(), eb: eb}},
		eb:        eb,
		stackVars: make(map[string]int),
		stackSize: 0,
	}
}

// CompileProgram compiles a Flap program to RISC-V64
func (rcg *RiscvCodeGen) CompileProgram(program *Program) error {
	// Function prologue
	// addi sp, sp, -32  # Allocate stack frame
	rcg.out.AddImm("sp", "sp", -32)
	// sd ra, 24(sp)     # Save return address
	rcg.out.Store64("ra", "sp", 24)
	// sd s0, 16(sp)     # Save frame pointer
	rcg.out.Store64("s0", "sp", 16)
	// addi s0, sp, 32   # Set frame pointer
	rcg.out.AddImm("s0", "sp", 32)

	// Compile each statement
	for _, stmt := range program.Statements {
		if err := rcg.compileStatement(stmt); err != nil {
			return err
		}
	}

	// Function epilogue (if no explicit exit)
	// li a0, 0          # Exit code 0
	rcg.out.LoadImm("a0", 0)
	// li a7, 93         # Exit syscall number
	rcg.out.LoadImm("a7", 93)
	// ecall
	rcg.out.Ecall()

	return nil
}

// compileStatement compiles a single statement
func (rcg *RiscvCodeGen) compileStatement(stmt Statement) error {
	switch s := stmt.(type) {
	case *ExpressionStmt:
		return rcg.compileExpression(s.Expr)
	case *AssignStmt:
		return rcg.compileAssignment(s)
	default:
		return fmt.Errorf("unsupported statement type for RISC-V64: %T", stmt)
	}
}

// compileExpression compiles an expression
func (rcg *RiscvCodeGen) compileExpression(expr Expression) error {
	switch e := expr.(type) {
	case *NumberExpr:
		// Load number into fa0 (floating-point register)
		// For now, convert to integer and load
		intVal := int64(e.Value)
		return rcg.out.LoadImm("a0", intVal)

	case *StringExpr:
		// Store string in rodata
		label := fmt.Sprintf("str_%d", len(rcg.eb.consts))
		rcg.eb.Define(label, e.Value+"\x00") // Null-terminated

		// Load address of string
		// For now, use a placeholder - will need proper PC-relative addressing
		return rcg.out.LoadImm("a0", 0) // TODO: Load actual address

	case *CallExpr:
		return rcg.compileCall(e)

	default:
		return fmt.Errorf("unsupported expression type for RISC-V64: %T", expr)
	}
}

// compileAssignment compiles an assignment statement
func (rcg *RiscvCodeGen) compileAssignment(assign *AssignStmt) error {
	// Compile the value
	if err := rcg.compileExpression(assign.Value); err != nil {
		return err
	}

	// Allocate stack space for variable
	rcg.stackSize += 8
	rcg.stackVars[assign.Name] = rcg.stackSize

	// Store result on stack
	// sd a0, -offset(s0)
	return rcg.out.Store64("a0", "s0", -int32(rcg.stackSize))
}

// compileCall compiles a function call
func (rcg *RiscvCodeGen) compileCall(call *CallExpr) error {
	switch call.Function {
	case "println":
		return rcg.compilePrintln(call)
	case "exit":
		return rcg.compileExit(call)
	default:
		return fmt.Errorf("unsupported function for RISC-V64: %s", call.Function)
	}
}

// compilePrintln compiles a println call using RISC-V write syscall
func (rcg *RiscvCodeGen) compilePrintln(call *CallExpr) error {
	if len(call.Args) == 0 {
		return fmt.Errorf("println requires an argument")
	}

	arg := call.Args[0]

	switch a := arg.(type) {
	case *StringExpr:
		// Store string in rodata
		label := fmt.Sprintf("str_%d", len(rcg.eb.consts))
		content := a.Value + "\n\x00"
		rcg.eb.Define(label, content)

		// RISC-V write syscall:
		// a7 = 64 (write)
		// a0 = 1 (stdout)
		// a1 = buffer address
		// a2 = length

		// li a7, 64         # write syscall
		if err := rcg.out.LoadImm("a7", 64); err != nil {
			return err
		}

		// li a0, 1          # stdout
		if err := rcg.out.LoadImm("a0", 1); err != nil {
			return err
		}

		// Load string address into a1
		// TODO: Implement PC-relative load for rodata symbols
		// For now, use placeholder
		if err := rcg.out.LoadImm("a1", 0); err != nil {
			return err
		}

		// li a2, length
		if err := rcg.out.LoadImm("a2", int64(len(content)-1)); err != nil {
			return err
		}

		// ecall
		rcg.out.Ecall()

	case *NumberExpr:
		// For numbers, we need to convert to string
		// This is complex - for now, just print a placeholder
		return fmt.Errorf("println(number) not yet implemented for RISC-V64")

	default:
		return fmt.Errorf("unsupported println argument type for RISC-V64: %T", arg)
	}

	return nil
}

// compileExit compiles an exit call
func (rcg *RiscvCodeGen) compileExit(call *CallExpr) error {
	exitCode := int64(0)

	if len(call.Args) > 0 {
		if num, ok := call.Args[0].(*NumberExpr); ok {
			exitCode = int64(num.Value)
		}
	}

	// li a0, exitCode
	if err := rcg.out.LoadImm("a0", exitCode); err != nil {
		return err
	}

	// li a7, 93  # exit syscall
	if err := rcg.out.LoadImm("a7", 93); err != nil {
		return err
	}

	// ecall
	rcg.out.Ecall()

	return nil
}
