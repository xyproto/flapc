package main

// SysWrite generates a write system call using architecture-specific registers
func (eb *ExecutableBuilder) SysWrite(what_data string, what_data_len ...string) {
	switch eb.machine {
	case MachineX86_64:
		eb.SysWriteX86_64(what_data, what_data_len...)
	case MachineARM64:
		eb.SysWriteARM64(what_data, what_data_len...)
	case MachineRiscv64:
		eb.SysWriteRiscv64(what_data, what_data_len...)
	}
}

// SysExit generates an exit system call using architecture-specific registers
func (eb *ExecutableBuilder) SysExit(code ...string) {
	switch eb.machine {
	case MachineX86_64:
		eb.SysExitX86_64(code...)
	case MachineARM64:
		eb.SysExitARM64(code...)
	case MachineRiscv64:
		eb.SysExitRiscv64(code...)
	}
}
