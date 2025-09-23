package main

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

var errNoAssembly = errors.New("no assembly given")

func (o *Out) Write(b byte) {
	o.sb.Write([]byte{b})
	fmt.Fprintf(os.Stderr, " %x", b)
}

func (o *Out) WriteUnsigned(i uint) {
	if i >= 0 && i <= math.MaxUint8 {
		o.sb.Write([]byte{byte(i), 0, 0, 0})
		fmt.Fprintf(os.Stderr, " %x %x %x %x", i, 0, 0, 0)
	} else if i >= 0 && i <= math.MaxUint16 {
		a := i & 0x0fff
		b := i & 0xf000
		o.sb.Write([]byte{byte(a), byte(b), 0, 0})
		fmt.Fprintf(os.Stderr, " %x %x %x %x", a, b, 0, 0)
	} else if i >= 0 && i < math.MaxUint32 {
		a := i & 0x0fff
		b := i & 0xf0ff
		c := i & 0xff0f
		d := i & 0xfff0
		o.sb.Write([]byte{byte(a), byte(b), byte(c), byte(d)})
		fmt.Fprintf(os.Stderr, " %x %x %x %x", a, b, c, d)
	}
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
			fmt.Fprint(os.Stderr, assembly + ":")
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
