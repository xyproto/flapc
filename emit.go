package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

var errNoAssembly = errors.New("no Assembly given")

func (o *Out) Write(b byte) int {
	o.buf.Write([]byte{b})
	fmt.Fprintf(os.Stderr, " %x", b)
	return 1
}

func (o *Out) WriteN(b byte, n int) int {
	for i := 0; i < n; i++ {
		o.Write(b)
	}
	return n
}

func (o *Out) Write2(b byte) int {
	o.buf.Write([]byte{b, 0})
	fmt.Fprintf(os.Stderr, " %x %x", b, 0)
	return 2
}

func (o *Out) Write4(b byte) int {
	o.buf.Write([]byte{b, 0, 0, 0})
	fmt.Fprintf(os.Stderr, " %x %x %x %x", b, 0, 0, 0)
	return 4
}

func (o *Out) Write8(b byte) int {
	o.buf.Write([]byte{b, 0, 0, 0, 0, 0, 0, 0})
	fmt.Fprintf(os.Stderr, " %x %x %x %x %x %x %x %x", b, 0, 0, 0, 0, 0, 0, 0)
	return 8
}

func (o *Out) Write8u(v uint64) int {
	binary.Write(&o.buf, binary.LittleEndian, v)
	fmt.Fprintf(os.Stderr, " %x", v)
	return 8
}

func (o *Out) WriteUnsigned(i uint) int {
	n := 0
	if i >= 0 && i <= math.MaxUint8 {
		o.buf.Write([]byte{byte(i), 0, 0, 0})
		fmt.Fprintf(os.Stderr, " %x %x %x %x", i, 0, 0, 0)
		n = 4
	} else if i >= 0 && i <= math.MaxUint16 {
		a := i & 0x0fff
		b := i & 0xf000
		o.buf.Write([]byte{byte(a), byte(b), 0, 0})
		fmt.Fprintf(os.Stderr, " %x %x %x %x", a, b, 0, 0)
		n = 4
	} else if i >= 0 && i < math.MaxUint32 {
		a := i & 0x0fff
		b := i & 0xf0ff
		c := i & 0xff0f
		d := i & 0xfff0
		o.buf.Write([]byte{byte(a), byte(b), byte(c), byte(d)})
		fmt.Fprintf(os.Stderr, " %x %x %x %x", a, b, c, d)
		n = 4
	}
	return n
}

func (o *Out) Emit(assembly string) error {
	all := strings.Fields(assembly)
	if len(all) == 0 {
		return errNoAssembly
	}
	head := strings.TrimSpace(all[0])
	var tail []string
	if len(all) > 1 {
		tail = all[1:]
	}
	if len(all) == 3 {
		switch head {
		case "mov":
			fmt.Fprint(os.Stderr, assembly+":")
			o.Write(0x48)
			o.Write(0xc7)
			dest := strings.TrimSuffix(strings.TrimSpace(tail[0]), ",")
			switch dest {
			case "rax":
				o.Write(0xc0)
			case "rbx":
				o.Write(0xc3)
			case "rcx":
				o.Write(0xc1)
			case "rdx":
				o.Write(0xc2)
			}
			val := strings.TrimSpace(tail[1])
			if n, err := strconv.Atoi(val); err == nil { // success
				o.WriteUnsigned(uint(n))
			}
			fmt.Fprintln(os.Stderr)
		}
	}
	return nil
}
