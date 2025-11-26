// Completion: 95% - Writer module for ARM64 ELF
package main

import (
	"fmt"
	"os"
	"strings"
)

// codegen_arm64_writer.go - ELF executable generation for ARM64 Linux
//
// This file handles the generation of ELF executables for Linux
// on ARM64 architecture.

func (fc *FlapCompiler) writeELFARM64(outputPath string) error {
	// Enable dynamic linking for ARM64 ELF
	fc.eb.useDynamicLinking = true

	textBytes := fc.eb.text.Bytes()
	rodataBytes := fc.eb.rodata.Bytes()

	// Build pltFunctions list from all called functions
	pltFunctions := []string{"printf", "exit", "malloc", "free", "realloc", "strlen", "pow", "fflush"}

	// Add all functions from usedFunctions
	pltSet := make(map[string]bool)
	for _, f := range pltFunctions {
		pltSet[f] = true
	}

	// Build set of lambda function names to exclude from PLT
	lambdaSet := make(map[string]bool)
	for _, lambda := range fc.lambdaFuncs {
		lambdaSet[lambda.Name] = true
	}

	for funcName := range fc.usedFunctions {
		// Skip lambda functions - they are internal, not external PLT functions
		if lambdaSet[funcName] {
			continue
		}
		// Skip internal runtime functions
		if strings.HasPrefix(funcName, "_flap") || strings.HasPrefix(funcName, "flap_") {
			continue
		}
		if !pltSet[funcName] {
			pltFunctions = append(pltFunctions, funcName)
			pltSet[funcName] = true
		}
	}

	// Set up dynamic sections
	ds := NewDynamicSections(fc.eb.target.Arch())
	fc.dynamicSymbols = ds

	// Add NEEDED libraries
	ds.AddNeeded("libc.so.6")

	// Check if pthread functions are used
	if fc.usedFunctions["pthread_create"] || fc.usedFunctions["pthread_join"] {
		ds.AddNeeded("libpthread.so.0")
	}

	// Check if any libm functions are called
	libmFunctions := map[string]bool{
		"sqrt": true, "sin": true, "cos": true, "tan": true,
		"asin": true, "acos": true, "atan": true, "atan2": true,
		"sinh": true, "cosh": true, "tanh": true,
		"log": true, "log10": true, "exp": true, "pow": true,
		"fabs": true, "fmod": true, "ceil": true, "floor": true,
	}
	needsLibm := false
	for funcName := range fc.usedFunctions {
		if libmFunctions[funcName] {
			needsLibm = true
			break
		}
	}
	if needsLibm {
		ds.AddNeeded("libm.so.6")
	}

	// Add C library dependencies from imports
	for libName := range fc.cLibHandles {
		if libName != "linked" && libName != "c" {
			ds.AddNeeded(libName)
		}
	}

	// Add symbols to dynamic sections (STB_GLOBAL = 1, STT_FUNC = 2)
	for _, funcName := range pltFunctions {
		ds.AddSymbol(funcName, 1, 2) // STB_GLOBAL, STT_FUNC
	}

	// Add symbols for lambda functions
	for _, lambda := range fc.lambdaFuncs {
		ds.AddSymbol(lambda.Name, 1, 2) // STB_GLOBAL, STT_FUNC
	}

	// Write complete dynamic ELF with PLT/GOT
	_, rodataAddr, textAddr, pltBase, err := fc.eb.WriteCompleteDynamicELF(ds, pltFunctions)
	if err != nil {
		return fmt.Errorf("failed to write ARM64 ELF: %v", err)
	}

	// Patch PLT calls in the generated code
	fc.eb.patchPLTCalls(ds, textAddr, pltBase, pltFunctions)

	// Patch PC-relative relocations (for rodata access)
	rodataSize := fc.eb.rodata.Len()
	fc.eb.PatchPCRelocations(textAddr, rodataAddr, rodataSize)

	// Update ELF with patched text
	fc.eb.patchTextInELF()

	// Write the final executable to file
	elfBytes := fc.eb.Bytes()
	if err := os.WriteFile(outputPath, elfBytes, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %v", err)
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Wrote ARM64 dynamic ELF executable: %s\n", outputPath)
		fmt.Fprintf(os.Stderr, "   Text size: %d bytes\n", len(textBytes))
		fmt.Fprintf(os.Stderr, "   Rodata size: %d bytes\n", len(rodataBytes))
		fmt.Fprintf(os.Stderr, "   PLT functions: %d\n", len(pltFunctions))
	}

	return nil
}
