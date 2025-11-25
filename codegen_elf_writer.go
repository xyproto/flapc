// Completion: 100% - Writer module complete
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

// codegen_elf_writer.go - ELF executable generation for x86_64 Linux/Unix
//
// This file handles the generation of ELF (Executable and Linkable Format)
// executables for Linux/Unix systems on x86_64 architecture.

// Confidence that this function is working: 85%
func (fc *FlapCompiler) writeELF(program *Program, outputPath string) error {
	// Enable dynamic linking for ELF (required for WriteCompleteDynamicELF)
	fc.eb.useDynamicLinking = true

	// Build pltFunctions list from all called functions
	// Start with essential functions that runtime helpers need
	pltFunctions := []string{"printf", "exit", "malloc", "free", "realloc", "strlen", "pow", "fflush"}

	// Add all functions from usedFunctions (includes call() dynamic calls)
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
		// Skip internal runtime functions (start with _flap or flap_) - they are resolved directly
		if strings.HasPrefix(funcName, "_flap") || strings.HasPrefix(funcName, "flap_") {
			continue
		}
		if !pltSet[funcName] {
			pltFunctions = append(pltFunctions, funcName)
			pltSet[funcName] = true
		}
	}

	// Build mapping from actual calls to PLT indices
	callToPLT := make(map[string]int)
	for i, f := range pltFunctions {
		callToPLT[f] = i
	}

	// Set up dynamic sections
	ds := NewDynamicSections()
	fc.dynamicSymbols = ds // Store for later symbol updates

	// Only add NEEDED libraries if their functions are actually used
	// libc.so.6 is always needed for basic functionality
	ds.AddNeeded("libc.so.6")

	// Check if pthread functions are used (parallel loops with @@)
	if fc.usedFunctions["pthread_create"] || fc.usedFunctions["pthread_join"] {
		ds.AddNeeded("libpthread.so.0")
	}

	// Check if any libm functions are called (via call() FFI)
	// Note: builtin math functions like sqrt(), sin(), cos() use hardware instructions, not libm
	// But call("sqrt", ...) calls libm's sqrt
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
		if libName != "linked" { // Skip our marker value
			// Skip "c" - standard C library functions are already in libc.so.6
			if libName == "c" {
				continue
			}

			// If library name already contains .so, it's a direct .so file - use it as-is
			libSoName := libName
			if strings.Contains(libSoName, ".so") {
				// Direct .so file (e.g., "libmanyargs.so" from import "/tmp/libmanyargs.so" as mylib)
				// Use it directly for DT_NEEDED
				if VerboseMode {
					fmt.Fprintf(os.Stderr, "Adding custom C library dependency: %s\n", libSoName)
				}
				ds.AddNeeded(libSoName)
				continue
			}

			// Add .so.X suffix if not present (standard library mapping)
			if !strings.Contains(libSoName, ".so") {
				// Try to get library name from pkg-config
				cmd := exec.Command("pkg-config", "--libs-only-l", libName)
				if output, err := cmd.Output(); err == nil {
					// Parse output like "-lSDL3" to get "SDL3"
					libs := strings.TrimSpace(string(output))
					if strings.HasPrefix(libs, "-l") {
						libSoName = "lib" + strings.TrimPrefix(libs, "-l") + ".so"
					} else {
						// Fallback to standard naming
						if !strings.HasPrefix(libSoName, "lib") {
							libSoName = "lib" + libSoName
						}
						libSoName += ".so"
					}
				} else {
					// pkg-config failed, try to find versioned .so using ldconfig
					if !strings.HasPrefix(libSoName, "lib") {
						libSoName = "lib" + libSoName
					}

					// Try to find the actual .so file with ldconfig
					ldconfigCmd := exec.Command("ldconfig", "-p")
					if ldOutput, ldErr := ldconfigCmd.Output(); ldErr == nil {
						// Search for libname.so in ldconfig output
						lines := strings.Split(string(ldOutput), "\n")
						for _, line := range lines {
							if strings.Contains(line, libSoName) && strings.Contains(line, "=>") {
								// Extract the path after =>
								parts := strings.Split(line, "=>")
								if len(parts) == 2 {
									actualPath := strings.TrimSpace(parts[1])
									// Extract just the filename from the path
									pathParts := strings.Split(actualPath, "/")
									if len(pathParts) > 0 {
										libSoName = pathParts[len(pathParts)-1]
									}
									break
								}
							}
						}
					}

					// If still no version, just add .so
					if !strings.Contains(libSoName, ".so") {
						libSoName += ".so"
					}
				}
			}
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Adding C library dependency: %s\n", libSoName)
			}
			ds.AddNeeded(libSoName)
		}
	}

	// Note: dlopen/dlsym/dlclose are part of libc.so.6 on modern glibc (2.34+)
	// No need to link libdl.so.2 separately

	// Add symbols for PLT functions
	for _, funcName := range pltFunctions {
		ds.AddSymbol(funcName, STB_GLOBAL, STT_FUNC)
	}

	// Add symbols for lambda functions so they can be resolved at runtime
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Adding %d lambda function symbols to dynsym\n", len(fc.lambdaFuncs))
	}
	for _, lambda := range fc.lambdaFuncs {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: Adding lambda symbol '%s' to dynsym\n", lambda.Name)
		}
		ds.AddSymbol(lambda.Name, STB_GLOBAL, STT_FUNC)
	}

	// Add cache pointer storage to rodata (8 bytes of zeros for each cache)
	if len(fc.memoCaches) > 0 {
		for cacheName := range fc.memoCaches {
			fc.eb.Define(cacheName, "\x00\x00\x00\x00\x00\x00\x00\x00")
		}
	}

	// Check if hot functions are used with WPO disabled
	if len(fc.hotFunctions) > 0 && fc.wpoTimeout == 0 {
		return fmt.Errorf("hot functions require whole-program optimization (do not use --opt-timeout=0)")
	}

	fc.buildHotFunctionTable()
	fc.generateHotFunctionTable()

	rodataSymbols := fc.eb.RodataSection()

	// Create sorted list of symbol names for deterministic ordering
	var symbolNames []string
	for name := range rodataSymbols {
		symbolNames = append(symbolNames, name)
	}
	sort.Strings(symbolNames)

	// DEBUG: Print what symbols we're writing

	// Clear rodata buffer before writing sorted symbols
	// (in case any data was written during code generation)
	fc.eb.rodata.Reset()

	estimatedRodataAddr := uint64(0x403000 + 0x100)
	currentAddr := estimatedRodataAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]

		// Align string literals to 8-byte boundaries for proper float64 access
		if strings.HasPrefix(symbol, "str_") {
			padding := (8 - (currentAddr % 8)) % 8
			if padding > 0 {
				fc.eb.WriteRodata(make([]byte, padding))
				currentAddr += padding
			}
		}

		fc.eb.WriteRodata([]byte(value))
		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
	}
	if fc.eb.rodata.Len() > 0 {
		previewLen := 32
		if fc.eb.rodata.Len() < previewLen {
			previewLen = fc.eb.rodata.Len()
		}
	}

	// Assign addresses to .data section symbols (writable data like closures)
	// Need to sort data symbols for consistent addresses
	dataSymbols := fc.eb.DataSection()
	if len(dataSymbols) > 0 {
		// Clear data buffer before writing sorted symbols
		fc.eb.data.Reset()

		dataBaseAddr := currentAddr // Follows .rodata
		dataSymbolNames := make([]string, 0, len(dataSymbols))
		for symbol := range dataSymbols {
			dataSymbolNames = append(dataSymbolNames, symbol)
		}
		sort.Strings(dataSymbolNames)

		for _, symbol := range dataSymbolNames {
			value := dataSymbols[symbol]
			// Write data to buffer first
			fc.eb.WriteData([]byte(value))
			// Then assign address
			fc.eb.DefineAddr(symbol, dataBaseAddr)
			dataBaseAddr += uint64(len(value))
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Wrote and assigned .data symbol %s to 0x%x (%d bytes)\n", symbol, fc.eb.consts[symbol].addr, len(value))
			}
		}
		currentAddr = dataBaseAddr
	}

	// Write complete dynamic ELF with unique PLT functions
	// Note: We pass pltFunctions (unique) for building PLT/GOT structure
	// We'll use fc.callOrder (with duplicates) later for patching actual call sites
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "\n=== First compilation callOrder: %v ===\n", fc.callOrder)
		}
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "=== pltFunctions (unique): %v ===\n", pltFunctions)
		}
	}

	gotBase, rodataBaseAddr, textAddr, pltBase, err := fc.eb.WriteCompleteDynamicELF(ds, pltFunctions)
	if err != nil {
		return err
	}

	// Update rodata addresses using same sorted order
	currentAddr = rodataBaseAddr
	for _, symbol := range symbolNames {
		value := rodataSymbols[symbol]

		// Apply same alignment as when writing rodata
		if strings.HasPrefix(symbol, "str_") {
			padding := (8 - (currentAddr % 8)) % 8
			currentAddr += padding
		}

		fc.eb.DefineAddr(symbol, currentAddr)
		currentAddr += uint64(len(value))
	}

	// Update .data addresses similarly
	dataSymbols = fc.eb.DataSection()
	if len(dataSymbols) > 0 {
		dataBaseAddr := currentAddr // Follows .rodata
		for symbol, value := range dataSymbols {
			fc.eb.DefineAddr(symbol, dataBaseAddr)
			dataBaseAddr += uint64(len(value))
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Updated .data symbol %s to 0x%x\n", symbol, fc.eb.consts[symbol].addr)
			}
		}
		currentAddr = dataBaseAddr
	}

	// Regenerate code with correct addresses
	fc.eb.text.Reset()
	// DON'T reset rodata - it already has correct addresses from first pass
	// Resetting rodata causes all symbols to move, breaking PC-relative addressing
	fc.eb.pcRelocations = []PCRelocation{} // Reset PC relocations for recompilation
	fc.eb.callPatches = []CallPatch{}      // Reset call patches for recompilation
	fc.eb.labels = make(map[string]int)    // Reset labels for recompilation
	fc.callOrder = []string{}              // Clear call order for recompilation
	fc.stringCounter = 0                   // Reset string counter for recompilation
	fc.labelCounter = 0                    // Reset label counter for recompilation
	fc.lambdaCounter = 0                   // Reset lambda counter for recompilation
	// DON'T clear lambdaFuncs - we need them for second pass lambda generation
	fc.lambdaOffsets = make(map[string]int) // Reset lambda offsets
	fc.variables = make(map[string]int)     // Reset variables map
	fc.mutableVars = make(map[string]bool)  // Reset mutability tracking
	fc.stackOffset = 0                      // Reset stack offset
	// Set up stack frame
	fc.out.PushReg("rbp")
	fc.out.MovRegToReg("rbp", "rsp")
	fc.out.SubImmFromReg("rsp", StackSlotSize) // Align stack to 16 bytes
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rdi", "rdi")
	fc.out.XorRegWithReg("rsi", "rsi")

	// DON'T re-define rodata symbols - they already exist from first pass
	// Re-defining them would change their addresses and break PC-relative references

	// ===== AVX-512 CPU DETECTION (regenerated) =====
	fc.out.MovImmToReg("rax", "7")              // CPUID leaf 7
	fc.out.XorRegWithReg("rcx", "rcx")          // subleaf 0
	fc.out.Emit([]byte{0x0f, 0xa2})             // cpuid
	fc.out.Emit([]byte{0xf6, 0xc3, 0x01})       // test bl, 1
	fc.out.Emit([]byte{0x0f, 0xba, 0xe3, 0x10}) // bt ebx, 16
	fc.out.Emit([]byte{0x0f, 0x92, 0xc0})       // setc al
	fc.out.LeaSymbolToReg("rbx", "cpu_has_avx512")
	fc.out.MovByteRegToMem("rax", "rbx", 0) // Write only AL, not full RAX
	fc.out.XorRegWithReg("rax", "rax")
	fc.out.XorRegWithReg("rbx", "rbx")
	fc.out.XorRegWithReg("rcx", "rcx")
	// ===== END AVX-512 DETECTION =====

	// Recompile with correct addresses
	// NOTE: Use the original program parameter (which includes imports),
	// not a reparsed version from source which would lose imported statements

	// Reset compiler state for second pass
	fc.variables = make(map[string]int)
	fc.mutableVars = make(map[string]bool)
	fc.varTypes = make(map[string]string)
	fc.stackOffset = 0
	fc.lambdaFuncs = nil // Clear lambda list so collectSymbols can repopulate it
	fc.lambdaCounter = 0
	fc.labelCounter = 0                                       // Reset label counter for consistent loop labels
	fc.movedVars = make(map[string]bool)                      // Reset moved variables tracking
	fc.scopedMoved = []map[string]bool{make(map[string]bool)} // Reset scoped tracking

	// Collect symbols again (two-pass compilation for second regeneration)
	for _, stmt := range program.Statements {
		if err := fc.collectSymbols(stmt); err != nil {
			return err
		}
	}

	// Reset labelCounter after collectSymbols so compilation uses same labels
	fc.labelCounter = 0

	// DON'T rebuild hot function table - it already exists in rodata from first pass
	// Rebuilding it would change its address and break PC-relative references

	fc.pushDeferScope()

	// Initialize default arena at program start (before any user code)
	// This is where _start jumps to, so it will execute first
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Generating arena init at text offset %d (after _start jump target)\n", fc.eb.text.Len())
	}
	fc.out.LeaSymbolToReg("rax", "_flap_default_arena_struct") // rax = arena struct
	fc.out.LeaSymbolToReg("rbx", "_flap_default_arena_buffer") // rbx = buffer
	fc.out.MovRegToMem("rbx", "rax", 0)                        // [struct+0] = buffer_ptr
	fc.out.MovImmToMem(65536, "rax", 8)                        // [struct+8] = capacity (64KB)
	fc.out.MovImmToMem(0, "rax", 16)                           // [struct+16] = offset (0)
	fc.out.MovImmToMem(8, "rax", 24)                           // [struct+24] = alignment (8)
	// Store arena struct pointer in _flap_default_arena
	fc.out.LeaSymbolToReg("rcx", "_flap_default_arena")
	fc.out.MovRegToMem("rax", "rcx", 0)

	// Generate code with symbols collected
	for _, stmt := range program.Statements {
		fc.compileStatement(stmt)
	}

	fc.popDeferScope()

	// Always add implicit exit at the end of the program
	// Even if there's an exit() call in the code, it might be conditional
	// If an unconditional exit() is called, it will never return, so this code is harmless
	// If we've used printf or other libc functions, call exit() to ensure proper cleanup
	// Otherwise use direct syscall for minimal programs
	if fc.usedFunctions["printf"] || fc.usedFunctions["exit"] || len(fc.usedFunctions) > 0 {
		// Use libc's exit() for proper cleanup (flushes buffers)
		fc.out.XorRegWithReg("rdi", "rdi") // exit code 0
		fc.trackFunctionCall("exit")
		fc.eb.GenerateCallInstruction("exit")
	} else {
		// Use direct syscall for minimal programs without libc dependencies
		fc.out.MovImmToReg("rax", "60")    // syscall number for exit
		fc.out.XorRegWithReg("rdi", "rdi") // exit code 0
		fc.eb.Emit("syscall")              // invoke syscall directly
	}

	// Generate lambda functions
	// (lambdas are generated after the main code, so they can reference main code)
	fc.generateLambdaFunctions()

	// Generate pattern lambda functions
	fc.generatePatternLambdaFunctions()

	// Generate runtime helper functions AFTER lambda generation
	fc.generateRuntimeHelpers()

	// Collect rodata symbols again (lambda/runtime functions may have created new ones)
	rodataSymbols = fc.eb.RodataSection()

	// Find any NEW symbols that weren't in the original list
	var newSymbols []string
	for symbol := range rodataSymbols {
		found := false
		for _, existingSym := range symbolNames {
			if symbol == existingSym {
				found = true
				break
			}
		}
		if !found {
			newSymbols = append(newSymbols, symbol)
		}
	}

	if len(newSymbols) > 0 {
		sort.Strings(newSymbols)

		// Append new symbols to rodata and assign addresses
		for _, symbol := range newSymbols {
			value := rodataSymbols[symbol]
			fc.eb.WriteRodata([]byte(value))
			fc.eb.DefineAddr(symbol, currentAddr)
			currentAddr += uint64(len(value))
			symbolNames = append(symbolNames, symbol)
		}
	}

	// Handle new .data symbols similarly
	dataSymbols = fc.eb.DataSection()
	newDataSymbols := []string{}
	for symbol := range dataSymbols {
		// Check if already assigned
		if _, ok := fc.eb.consts[symbol]; ok && fc.eb.consts[symbol].addr != 0 {
			continue
		}
		newDataSymbols = append(newDataSymbols, symbol)
	}
	if len(newDataSymbols) > 0 {
		sort.Strings(newDataSymbols)
		for _, symbol := range newDataSymbols {
			value := dataSymbols[symbol]
			fc.eb.DefineAddr(symbol, currentAddr)
			// Write the actual data to the .data buffer
			fc.eb.WriteData([]byte(value))
			currentAddr += uint64(len(value))
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "Assigned new .data symbol %s to 0x%x, wrote %d bytes\n", symbol, fc.eb.consts[symbol].addr, len(value))
			}
		}
	}

	// Set lambda function addresses
	if VerboseMode {
		fmt.Fprintf(os.Stderr, "DEBUG: Setting lambda function addresses, have %d lambdas\n", len(fc.lambdaOffsets))
	}
	for lambdaName, offset := range fc.lambdaOffsets {
		lambdaAddr := textAddr + uint64(offset)
		fc.eb.DefineAddr(lambdaName, lambdaAddr)

		// Update the symbol value in the dynamic symbol table
		if fc.dynamicSymbols != nil {
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG: Calling UpdateSymbolValue for lambda '%s' at address 0x%x\n", lambdaName, lambdaAddr)
			}
			success := fc.dynamicSymbols.UpdateSymbolValue(lambdaName, lambdaAddr)
			if VerboseMode {
				fmt.Fprintf(os.Stderr, "DEBUG: UpdateSymbolValue returned %v\n", success)
			}
		} else if VerboseMode {
			fmt.Fprintf(os.Stderr, "DEBUG: fc.dynamicSymbols is nil, cannot update symbol\n")
		}
	}

	// Rebuild and repatch the symbol table with updated lambda addresses
	if fc.dynamicSymbols != nil {
		fc.dynamicSymbols.buildSymbolTable()
		fc.eb.patchDynsymInELF(fc.dynamicSymbols)
	}

	// Patch PLT calls using callOrder (actual sequence of calls)
	// patchPLTCalls will look up each function name in the PLT to get its offset
	// This handles duplicate calls (e.g., two calls to exit) correctly
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "\n=== Second compilation callOrder: %v ===\n", fc.callOrder)
		}
	}
	fc.eb.patchPLTCalls(ds, textAddr, pltBase, fc.callOrder)

	// Patch PC-relative relocations
	rodataSize := fc.eb.rodata.Len()
	fc.eb.PatchPCRelocations(textAddr, rodataBaseAddr, rodataSize)

	// Patch function calls in regenerated code
	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "\n=== Patching function calls (regenerated code) ===\n")
		}
	}
	// Patch hot function pointer table
	fc.patchHotFunctionTable()

	// Update ELF with regenerated code (copies eb.text into ELF buffer)
	fc.eb.patchTextInELF()
	fc.eb.patchRodataInELF()
	fc.eb.patchDataInELF()

	// Output the executable file
	elfBytes := fc.eb.Bytes()
	if err := os.WriteFile(outputPath, elfBytes, 0o755); err != nil {
		return err
	}

	if fc.debug {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Final GOT base: 0x%x\n", gotBase)
		}
	}
	return nil
}

// Confidence that this function is working: 50%
// writePE generates a Windows PE (Portable Executable) file for x86_64
