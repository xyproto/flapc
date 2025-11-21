package main

import (
"fmt"
"os"
"strings"
)

// codegen_pe_writer.go - PE executable generation for x86_64 Windows
//
// This file handles the generation of PE (Portable Executable) files
// for Windows systems on x86_64 architecture.

// Confidence that this function is working: 70%
func (fc *FlapCompiler) writePE(program *Program, outputPath string) error {
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Generating Windows PE executable\n")
	}

	// For Windows PE, we need to handle imports differently than ELF
	// Windows uses import tables instead of PLT/GOT

	// Build list of required imports from C runtime (msvcrt.dll)
	requiredImports := []string{"printf", "exit", "malloc", "free", "realloc", "strlen", "pow", "fflush"}

	// Add all functions from usedFunctions
	lambdaSet := make(map[string]bool)
	for _, lambda := range fc.lambdaFuncs {
		lambdaSet[lambda.Name] = true
	}

	for funcName := range fc.usedFunctions {
		// Skip lambda functions - they are internal
		if lambdaSet[funcName] {
			continue
		}
		// Skip internal runtime functions
		if strings.HasPrefix(funcName, "_flap") || strings.HasPrefix(funcName, "flap_") {
			continue
		}
		requiredImports = append(requiredImports, funcName)
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "Required Windows imports: %v\n", requiredImports)
	}

	// For now, use the simple PE writer without full import table support
	// This is a minimal PE that will need Wine or Windows to run
	// Runtime helpers will need to be added similarly to ELF
	if err := fc.eb.WritePE(outputPath); err != nil {
		return fmt.Errorf("failed to write PE file: %v", err)
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "PE executable written to %s\n", outputPath)
		fmt.Fprintf(os.Stderr, "Note: Full import table support is work in progress\n")
	}

	return nil
}
