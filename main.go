package main

import (
	"fmt"
	"strconv"
	"strings"
)

// A C compiler written in GO

type Out struct {
	platform string
	env      map[string]string
	sb       strings.Builder
}

func New(platform string) *Out {
	var o Out
	o.env = make(map[string]string)
	o.platform = platform
	return &o
}

func GlobalLookup(what string) string {
	switch what {
	case "SYS_WRITE":
		return "1"
	case "SYS_EXIT":
		return "60"
	case "STDOUT":
		return "1"
	}
	return ""
}

func (o *Out) Lookup(what string) string {
	if v := GlobalLookup(what); v != "" {
		return v
	}
	if v, ok := o.env[what]; ok {
		return v
	}
	return "0"
}

func (o *Out) Get() string {
	return o.sb.String()
}

func (o *Out) Emit(assembly string, comment ...string) {
	if len(comment) == 0 {
		o.sb.WriteString(assembly + "\n")
	} else {
		o.sb.WriteString(assembly + "\t;\t" + comment[0] + "\n")
	}
}

func (o *Out) Define(symbol, value string) {
	o.env[symbol] = value
	o.env[symbol+"_len"] = strconv.Itoa(len(value))
}

func (o *Out) SysWrite(what_data string, what_data_len ...string) {
	o.Emit("mov rax, " + GlobalLookup("SYS_WRITE"))
	o.Emit("mov rdi, " + GlobalLookup("STDOUT"))
	o.Emit("mov rsi, " + what_data)
	if len(what_data_len) == 0 {
		o.Emit("mov rdx, " + what_data + "_len")
	} else {
		o.Emit("mov rdx, " + what_data_len[0])
	}
	o.Emit("syscall")
}

func (o *Out) SysExit(code ...string) {
	o.Emit("mov rax, " + GlobalLookup("SYS_EXIT"))
	if len(code) == 0 {
		o.Emit("mov rdi, 0")
	} else {
		o.Emit("mov rdi, " + code[0])
	}
	o.Emit("syscall")
}

func main() {
	o := New("x86_64")
	o.Define("hello", "Hello, World!")
	o.SysWrite("hello")
	o.SysExit()
	fmt.Println(o.Get())
}
