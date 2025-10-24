package main

type X86_64CodeGen struct {
	writer Writer
	eb     *ExecutableBuilder
}

func NewX86_64CodeGen(writer Writer, eb *ExecutableBuilder) *X86_64CodeGen {
	return &X86_64CodeGen{
		writer: writer,
		eb:     eb,
	}
}

func (x *X86_64CodeGen) write(b uint8) {
	x.writer.(*BufferWrapper).Write(b)
}

func (x *X86_64CodeGen) writeUnsigned(i uint) {
	x.writer.(*BufferWrapper).WriteUnsigned(i)
}

func (x *X86_64CodeGen) emit(bytes []byte) {
	for _, b := range bytes {
		x.write(b)
	}
}

func (x *X86_64CodeGen) Ret() {
	x.write(0xC3)
}

func (x *X86_64CodeGen) Syscall() {
	x.write(0x0F)
	x.write(0x05)
}

func (x *X86_64CodeGen) CallSymbol(symbol string) {
	x.write(0xE8)
	
	callPos := x.eb.text.Len()
	x.writeUnsigned(0x00000000)
	
	x.eb.callPatches = append(x.eb.callPatches, CallPatch{
		position:   callPos,
		targetName: symbol,
	})
}

func (x *X86_64CodeGen) CallRelative(offset int32) {
	x.write(0xE8)
	x.writeUnsigned(uint(offset))
}

func (x *X86_64CodeGen) CallRegister(reg string) {
	r, ok := x86_64Registers[reg]
	if !ok {
		compilerError("Unknown register: %s", reg)
	}
	x.write(0xFF)
	x.write(0xD0 + r.Encoding)
}

func (x *X86_64CodeGen) JumpUnconditional(offset int32) {
	x.write(0xE9)
	x.writeUnsigned(uint(offset))
}

func (x *X86_64CodeGen) JumpConditional(condition JumpCondition, offset int32) {
	var opcode byte
	switch condition {
	case JumpEqual:
		opcode = 0x84
	case JumpNotEqual:
		opcode = 0x85
	case JumpLess:
		opcode = 0x8C
	case JumpLessOrEqual:
		opcode = 0x8E
	case JumpGreater:
		opcode = 0x8F
	case JumpGreaterOrEqual:
		opcode = 0x8D
	default:
		compilerError("Unknown jump condition: %v", condition)
	}
	x.write(0x0F)
	x.write(opcode)
	x.writeUnsigned(uint(offset))
}
