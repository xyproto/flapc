package main

// SysWrite generates a write system call using architecture-specific registers
func (eb *ExecutableBuilder) SysWrite(what_data string, what_data_len ...string) {
	switch eb.platform.Arch {
	case ArchX86_64:
		eb.SysWriteX86_64(what_data, what_data_len...)
	case ArchARM64:
		eb.SysWriteARM64(what_data, what_data_len...)
	case ArchRiscv64:
		eb.SysWriteRiscv64(what_data, what_data_len...)
	}
}

// SysExit generates an exit system call using architecture-specific registers
func (eb *ExecutableBuilder) SysExit(code ...string) {
	switch eb.platform.Arch {
	case ArchX86_64:
		eb.SysExitX86_64(code...)
	case ArchARM64:
		eb.SysExitARM64(code...)
	case ArchRiscv64:
		eb.SysExitRiscv64(code...)
	}
}
