package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// codegen_macho_writer.go - Mach-O executable generation for ARM64 macOS
//
// This file handles the generation of Mach-O executables for macOS
// on ARM64 (Apple Silicon) architecture.

// Confidence that this function is working: 60%
func (fc *FlapCompiler) writeMachOARM64(outputPath string) error {
	// Build neededFunctions list from call patches (actual function calls made)
	// Extract unique function names from callPatches
	neededSet := make(map[string]bool)
	for _, patch := range fc.eb.callPatches {
		// patch.targetName is like "malloc$stub" or "printf$stub"
		// Strip the "$stub" suffix to get the function name
		funcName := patch.targetName
		if strings.HasSuffix(funcName, "$stub") {
			funcName = funcName[:len(funcName)-5] // Remove "$stub"
		}
		neededSet[funcName] = true
	}

	// Convert set to slice
	neededFuncs := make([]string, 0, len(neededSet))
	for funcName := range neededSet {
		neededFuncs = append(neededFuncs, funcName)
	}

	// Assign to executable builder for Mach-O generation
	fc.eb.neededFunctions = neededFuncs
	if len(neededFuncs) > 0 {
		fc.eb.useDynamicLinking = true
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> ARM64 neededFunctions: %v\n", neededFuncs)
	}

	// First, write all rodata symbols to the rodata buffer and assign addresses
	pageSize := uint64(0x4000) // 16KB page size for ARM64
	textSize := uint64(fc.eb.text.Len())
	textSizeAligned := (textSize + pageSize - 1) &^ (pageSize - 1)

	// Calculate rodata address (comes after __TEXT segment)
	rodataAddr := pageSize + textSizeAligned

	if VerboseMode {
		fmt.Fprintln(os.Stderr, "-> Writing rodata symbols")
	}

	// Get all rodata symbols and write them
	rodataSymbols := fc.eb.RodataSection()
	currentAddr := rodataAddr
	for symbol, value := range rodataSymbols {
		fc.eb.WriteRodata([]byte(value))
		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "   %s at 0x%x (%d bytes)\n", symbol, fc.eb.consts[symbol].addr, len(value))
		}
	}

	rodataSize := fc.eb.rodata.Len()

	// Set lambda function addresses from labels
	textAddr := pageSize
	for labelName, offset := range fc.eb.labels {
		if strings.HasPrefix(labelName, "lambda_") {
			lambdaAddr := textAddr + uint64(offset)
			fc.eb.DefineAddr(labelName, lambdaAddr)
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG: Setting %s address to 0x%x (offset %d)\n", labelName, lambdaAddr, offset)
			}
		}
	}

	// Now patch PC-relative relocations in the text
	if VerboseMode && len(fc.eb.pcRelocations) > 0 {
		fmt.Fprintf(os.Stderr, "-> Patching %d PC-relative relocations\n", len(fc.eb.pcRelocations))
	}
	fc.eb.PatchPCRelocations(textAddr, rodataAddr, rodataSize)

	// Use the existing Mach-O writer infrastructure
	if err := fc.eb.WriteMachO(); err != nil {
		return fmt.Errorf("failed to write Mach-O: %v", err)
	}

	// Write the executable
	machoBytes := fc.eb.elf.Bytes()

	if err := os.WriteFile(outputPath, machoBytes, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %v", err)
	}

	cmd := exec.Command("ldid", "-S", outputPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Warning: ldid signing failed: %v\n%s\n", err, output)
		}
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Wrote ARM64 Mach-O executable: %s\n", outputPath)
		fmt.Fprintf(os.Stderr, "   Text size: %d bytes\n", fc.eb.text.Len())
		fmt.Fprintf(os.Stderr, "   Rodata size: %d bytes\n", rodataSize)
	}

	return nil
}

// writeELFRiscv64 writes a RISC-V64 ELF executable
