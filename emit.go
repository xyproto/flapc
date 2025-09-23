package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var errNoAssembly = errors.New("no Assembly given")

func (bw *BufferWrapper) Write(b byte) int {
	bw.buf.Write([]byte{b})
	fmt.Fprintf(os.Stderr, " %x", b)
	return 1
}

func (bw *BufferWrapper) WriteN(b byte, n int) int {
	for i := 0; i < n; i++ {
		bw.Write(b)
	}
	return n
}

func (bw *BufferWrapper) Write2(b byte) int {
	bw.buf.Write([]byte{b, 0})
	fmt.Fprintf(os.Stderr, " %x %x", b, 0)
	return 2
}

func (bw *BufferWrapper) Write4(b byte) int {
	bw.buf.Write([]byte{b, 0, 0, 0})
	fmt.Fprintf(os.Stderr, " %x %x %x %x", b, 0, 0, 0)
	return 4
}

func (bw *BufferWrapper) Write8(b byte) int {
	bw.buf.Write([]byte{b, 0, 0, 0, 0, 0, 0, 0})
	fmt.Fprintf(os.Stderr, " %x %x %x %x %x %x %x %x", b, 0, 0, 0, 0, 0, 0, 0)
	return 8
}

func (bw *BufferWrapper) Write8u(v uint64) int {
	binary.Write(bw.buf, binary.LittleEndian, v)
	fmt.Fprintf(os.Stderr, " %x", v)
	return 8
}

func (bw *BufferWrapper) WriteBytes(bs []byte) int {
	bw.buf.Write(bs)
	for _, b := range bs {
		fmt.Fprintf(os.Stderr, " %x", b)
	}
	return 1
}

func (o *ExecutableBuilder) PrependBytes(bs []byte) {
	var newBuf bytes.Buffer
	newBuf.Write(bs)
	newBuf.Write(o.text.Bytes())
	o.text = newBuf
}

func (bw *BufferWrapper) WriteUnsigned(i uint) int {
	a := byte(i & 0xff)
	b := byte((i >> 8) & 0xff)
	c := byte((i >> 16) & 0xff)
	d := byte((i >> 24) & 0xff)
	bw.buf.Write([]byte{a, b, c, d})
	fmt.Fprintf(os.Stderr, " %x %x %x %x", a, b, c, d)
	return 4
}

func (eb *ExecutableBuilder) Emit(assembly string) error {
	w := eb.TextWriter()
	all := strings.Fields(assembly)
	if len(all) == 0 {
		return errNoAssembly
	}
	head := strings.TrimSpace(all[0])
	var tail []string
	if len(all) > 1 {
		tail = all[1:]
	}
	if len(all) == 1 {
		switch head {
		case "syscall":
			fmt.Fprint(os.Stderr, assembly+":")
			switch eb.machine {
			case "x86_64":
				w.Write(0x0f) // syscall instruction for x86_64
				w.Write(0x05)
			case "aarch64":
				w.Write(0xd4) // svc #0 instruction for aarch64
				w.Write(0x00)
				w.Write(0x00)
				w.Write(0x01)
			case "riscv64":
				w.Write(0x73) // ecall instruction for riscv64
				w.Write(0x00)
				w.Write(0x00)
				w.Write(0x00)
			}
			fmt.Fprintln(os.Stderr)
		}
	} else if len(all) == 3 {
		switch head {
		case "mov":
			fmt.Fprint(os.Stderr, assembly+":")
			dest := strings.TrimSuffix(strings.TrimSpace(tail[0]), ",")
			val := strings.TrimSpace(tail[1])

			switch eb.machine {
			case "x86_64":
				w.Write(0x48)
				w.Write(0xc7)
				switch dest {
				case "rax":
					w.Write(0xc0)
				case "rbx":
					w.Write(0xc3)
				case "rcx":
					w.Write(0xc1)
				case "rdx":
					w.Write(0xc2)
				case "rdi":
					w.Write(0xc7)
				case "rsi":
					w.Write(0xc6)
				}
				if n, err := strconv.Atoi(val); err == nil {
					w.WriteUnsigned(uint(n))
				} else {
					addr := eb.Lookup(val)
					if n, err := strconv.Atoi(addr); err == nil {
						w.WriteUnsigned(uint(n))
					}
				}
			case "aarch64":
				// For aarch64, we'll emit mov immediate instructions
				// This is a simplified implementation
				switch dest {
				case "x8", "x0", "x1", "x2": // Common ARM64 registers for syscalls
					w.Write(0xd2) // mov immediate instruction family
					w.Write(0x80) // placeholder for now
					w.Write(0x00)
					w.Write(0x08) // placeholder register encoding
				}
				// For now, just write a placeholder value
				if n, err := strconv.Atoi(val); err == nil {
					w.WriteUnsigned(uint(n))
				} else {
					addr := eb.Lookup(val)
					if n, err := strconv.Atoi(addr); err == nil {
						w.WriteUnsigned(uint(n))
					}
				}
			case "riscv64":
				// For riscv64, we'll emit addi instructions (load immediate)
				// This is a simplified implementation
				switch dest {
				case "a7", "a0", "a1", "a2": // Common RISC-V registers for syscalls
					w.Write(0x13) // addi instruction family
					w.Write(0x08) // placeholder for now
					w.Write(0x00)
					w.Write(0x00) // placeholder register encoding
				}
				// For now, just write a placeholder value
				if n, err := strconv.Atoi(val); err == nil {
					w.WriteUnsigned(uint(n))
				} else {
					addr := eb.Lookup(val)
					if n, err := strconv.Atoi(addr); err == nil {
						w.WriteUnsigned(uint(n))
					}
				}
			}
			fmt.Fprintln(os.Stderr)
		}
	}
	return nil
}
