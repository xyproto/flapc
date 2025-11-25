// Completion: 95% - Writer module for ARM64 ELF
package main

import (
	"fmt"
	"os"
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

	// Generate ELF header and program headers for ARM64
	fc.eb.WriteELFHeader()

	// Write the executable
	elfBytes := fc.eb.Bytes()
	if err := os.WriteFile(outputPath, elfBytes, 0755); err != nil {
		return fmt.Errorf("failed to write executable: %v", err)
	}

	if VerboseMode {
		fmt.Fprintf(os.Stderr, "-> Wrote ARM64 ELF executable: %s\n", outputPath)
		fmt.Fprintf(os.Stderr, "   Text size: %d bytes\n", len(textBytes))
		fmt.Fprintf(os.Stderr, "   Rodata size: %d bytes\n", len(rodataBytes))
	}

	return nil
}
