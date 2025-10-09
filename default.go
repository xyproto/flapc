package main

import (
	"fmt"
	"log"
	"os"
)

func (eb *ExecutableBuilder) CompileDefaultProgram(outputFile string) error {
	eb.Define("hello", "Hello, World!\n\x00")
	// Enable dynamic linking for glibc
	eb.useDynamicLinking = true
	eb.neededFunctions = []string{"printf", "exit"}
	if eb.useDynamicLinking && len(eb.neededFunctions) > 0 {
		fmt.Fprintln(os.Stderr, "-> .rodata")
		rodataSymbols := eb.RodataSection()
		estimatedRodataAddr := uint64(0x403000 + 0x100)
		currentAddr := estimatedRodataAddr
		for symbol, value := range rodataSymbols {
			eb.WriteRodata([]byte(value))
			eb.DefineAddr(symbol, currentAddr)
			currentAddr += uint64(len(value))
			fmt.Fprintf(os.Stderr, "%s = %q at ~0x%x (estimated)\n", symbol, value, eb.consts[symbol].addr)
		}
		// Generate text with estimated BSS addresses
		fmt.Fprintln(os.Stderr, "-> .text")
		err := eb.GenerateGlibcHelloWorld()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating glibc hello world: %v\n", err)
			// Fallback to syscalls
			eb.SysWrite("hello")
			eb.SysExit()
		}
		fmt.Fprintln(os.Stderr, "-> ELF generation")
		// Set up complete dynamic sections
		ds := NewDynamicSections()
		ds.AddNeeded("libc.so.6")
		// Add symbols
		for _, funcName := range eb.neededFunctions {
			ds.AddSymbol(funcName, STB_GLOBAL, STT_FUNC)
		}
		gotBase, rodataBaseAddr, textAddr, pltBase, err := eb.WriteCompleteDynamicELF(ds, eb.neededFunctions)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Fprintln(os.Stderr, "-> .rodata (final addresses) and regenerating code")
		currentAddr = rodataBaseAddr
		for symbol, value := range rodataSymbols {
			eb.DefineAddr(symbol, currentAddr)
			currentAddr += uint64(len(value))
			fmt.Fprintf(os.Stderr, "%s = %q at 0x%x\n", symbol, value, eb.consts[symbol].addr)
		}
		eb.text.Reset()
		err = eb.GenerateGlibcHelloWorld()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error regenerating code: %v\n", err)
		}
		fmt.Fprintln(os.Stderr, "-> Patching PLT calls in regenerated code")
		eb.patchPLTCalls(ds, textAddr, pltBase, eb.neededFunctions)
		fmt.Fprintln(os.Stderr, "-> Patching RIP-relative relocations in regenerated code")
		rodataSize := eb.rodata.Len()
		eb.PatchPCRelocations(textAddr, rodataBaseAddr, rodataSize)
		fmt.Fprintln(os.Stderr, "-> Updating ELF with regenerated code")
		eb.patchTextInELF()
		fmt.Fprintf(os.Stderr, "Final GOT base: 0x%x\n", gotBase)
	} else {
		fmt.Fprintln(os.Stderr, "-> .rodata")
		rodataSymbols := eb.RodataSection()
		rodataAddr := baseAddr + headerSize
		currentAddr := uint64(rodataAddr)
		for symbol, value := range rodataSymbols {
			eb.DefineAddr(symbol, currentAddr)
			currentAddr += eb.WriteRodata([]byte(value))
			fmt.Fprintf(os.Stderr, "%s = %q\n", symbol, value)
		}
		fmt.Fprintln(os.Stderr, "-> .text")
		if err := eb.GenerateGlibcHelloWorld(); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating glibc hello world: %v\n", err)
			eb.SysWrite("hello")
			eb.SysExit()
		}
		if len(eb.dynlinker.Libraries) > 0 {
			eb.WriteDynamicELF()
		} else {
			eb.WriteELFHeader()
		}
	}
	// Output the executable file
	if err := os.WriteFile(outputFile, eb.Bytes(), 0o755); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Wrote %s\n", outputFile)
	return nil
}
