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
			w.Write(0x0f)
			w.Write(0x05)
			fmt.Fprintln(os.Stderr)
		}
	} else if len(all) == 3 {
		switch head {
		case "mov":
			fmt.Fprint(os.Stderr, assembly+":")
			w.Write(0x48)
			w.Write(0xc7)
			dest := strings.TrimSuffix(strings.TrimSpace(tail[0]), ",")
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
			val := strings.TrimSpace(tail[1])
			if n, err := strconv.Atoi(val); err == nil { // success
				w.WriteUnsigned(uint(n))
			} else {
				addr := eb.Lookup(val)
				if n, err := strconv.Atoi(addr); err == nil { // success
					w.WriteUnsigned(uint(n))
				}
			}
			fmt.Fprintln(os.Stderr)
		}
	}
	return nil
}
