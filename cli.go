// Completion: 100% - Utility module complete
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// cli.go - User-friendly command-line interface for c67
//
// This file implements a Go-like CLI interface with subcommands:
// - c67 (default: compile current directory or show help)
// - c67 build <file> (compile to executable)
// - c67 run <file> (compile and run immediately)
// - c67 <file.c67> (shorthand for build)
//
// Also supports shebang execution: #!/usr/bin/c67

// CommandContext holds the execution context for a CLI command
type CommandContext struct {
	Args       []string
	Platform   Platform
	Verbose    bool
	Quiet      bool
	OptTimeout float64
	UpdateDeps bool
	SingleFile bool
	OutputPath string
}

// RunCLI is the main entry point for the user-friendly CLI
// It determines which command to run based on arguments
func RunCLI(args []string, platform Platform, verbose, quiet bool, optTimeout float64, updateDeps, singleFile bool, outputPath string) error {
	ctx := &CommandContext{
		Args:       args,
		Platform:   platform,
		Verbose:    verbose,
		Quiet:      quiet,
		OptTimeout: optTimeout,
		UpdateDeps: updateDeps,
		SingleFile: singleFile,
		OutputPath: outputPath,
	}

	// No arguments - show help
	if len(args) == 0 {
		return cmdHelp(ctx)
	}

	// Check for shebang execution
	// If first arg is a .c67 file and it starts with #!, we're in shebang mode
	if len(args) > 0 && strings.HasSuffix(args[0], ".c67") {
		content, err := os.ReadFile(args[0])
		if err == nil && len(content) > 2 && content[0] == '#' && content[1] == '!' {
			// Shebang mode - run the file with remaining args
			return cmdRunShebang(ctx, args[0], args[1:])
		}
	}

	// Parse subcommand
	subcmd := args[0]

	switch subcmd {
	case "build":
		if len(args) < 2 {
			return fmt.Errorf("usage: c67 build <file.c67> [-o output]")
		}
		return cmdBuild(ctx, args[1:])

	case "run":
		if len(args) < 2 {
			return fmt.Errorf("usage: c67 run <file.c67> [args...]")
		}
		return cmdRun(ctx, args[1:])

	case "help", "--help", "-h":
		return cmdHelp(ctx)

	case "version", "--version", "-V":
		fmt.Println(versionString)
		return nil

	default:
		// Check if it's a .c67 file (shorthand for build)
		if strings.HasSuffix(subcmd, ".c67") {
			return cmdBuild(ctx, args)
		}

		// Check if it's a directory (compile all .c67 files)
		info, err := os.Stat(subcmd)
		if err == nil && info.IsDir() {
			return cmdBuildDir(ctx, subcmd)
		}

		// Unknown command
		return fmt.Errorf("unknown command: %s\n\nRun 'c67 help' for usage information", subcmd)
	}
}

// cmdBuild compiles a C67 source file to an executable
// Confidence that this function is working: 85%
func cmdBuild(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: c67 build <file.c67> [-o output]")
	}

	inputFile := args[0]
	outputPath := ""

	// Parse optional -o flag from args first (takes precedence)
	for i := 1; i < len(args); i++ {
		if args[i] == "-o" && i+1 < len(args) {
			outputPath = args[i+1]
			i++
		}
	}

	// If not in args, use context output path (from main -o flag)
	if outputPath == "" && ctx.OutputPath != "" {
		outputPath = ctx.OutputPath
	}

	// Auto-detect Windows target from .exe extension if outputPath was specified
	if outputPath != "" && strings.HasSuffix(strings.ToLower(outputPath), ".exe") && ctx.Platform.OS != OSWindows {
		// Output ends with .exe but target isn't Windows - auto-detect
		ctx.Platform.OS = OSWindows
		if ctx.Verbose {
			fmt.Fprintf(os.Stderr, "Auto-detected Windows target from .exe output filename\n")
		}
	}

	// If still no output path specified, use input filename without extension
	if outputPath == "" {
		outputPath = strings.TrimSuffix(filepath.Base(inputFile), ".c67")
		// Add .exe extension for Windows targets
		if ctx.Platform.OS == OSWindows {
			outputPath += ".exe"
		}
	}

	// When a specific file is given (not -s flag explicitly passed), enable single-file mode
	// This ensures c67 doesn't look for other .c67 files in the same directory
	oldSingleFlag := SingleFlag
	if !ctx.SingleFile {
		// Only set SingleFlag if not already set via command line
		SingleFlag = true
		defer func() { SingleFlag = oldSingleFlag }()
	}

	if ctx.Verbose {
		fmt.Fprintf(os.Stderr, "Building %s -> %s (single-file mode: %v)\n", inputFile, outputPath, SingleFlag)
	}

	// Compile
	err := CompileC67WithOptions(inputFile, outputPath, ctx.Platform, ctx.OptTimeout)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	if !ctx.Quiet {
		fmt.Printf("Built: %s\n", outputPath)
	}

	return nil
}

// cmdRun compiles a C67 source file to /dev/shm and executes it
func cmdRun(ctx *CommandContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: c67 run <file.c67> [args...]")
	}

	inputFile := args[0]
	programArgs := args[1:]

	// Create temporary executable in /dev/shm (Linux RAM disk for fast execution)
	// Fall back to temp directory if /dev/shm doesn't exist
	tmpDir := "/dev/shm"
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		tmpDir = os.TempDir()
	}

	// Create unique temporary filename
	baseName := strings.TrimSuffix(filepath.Base(inputFile), ".c67")
	tmpExec := filepath.Join(tmpDir, fmt.Sprintf("c67_run_%s_%d", baseName, os.Getpid()))

	// Enable single-file mode when running a specific file
	oldSingleFlag := SingleFlag
	if !ctx.SingleFile {
		SingleFlag = true
		defer func() { SingleFlag = oldSingleFlag }()
	}

	if ctx.Verbose {
		fmt.Fprintf(os.Stderr, "Compiling %s -> %s (single-file mode)\n", inputFile, tmpExec)
	}

	// Compile
	err := CompileC67WithOptions(inputFile, tmpExec, ctx.Platform, ctx.OptTimeout)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	// Ensure cleanup
	defer os.Remove(tmpExec)

	if ctx.Verbose {
		fmt.Fprintf(os.Stderr, "Running %s\n", tmpExec)
	}

	// Execute with arguments
	cmd := exec.Command(tmpExec, programArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Program ran but exited with non-zero status
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("execution failed: %v", err)
	}

	return nil
}

// cmdRunShebang handles shebang execution (#!/usr/bin/c67)
func cmdRunShebang(ctx *CommandContext, scriptPath string, scriptArgs []string) error {
	// In shebang mode, we compile and run immediately
	// This is similar to cmdRun but optimized for shebang use

	tmpDir := "/dev/shm"
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		tmpDir = os.TempDir()
	}

	baseName := strings.TrimSuffix(filepath.Base(scriptPath), ".c67")
	tmpExec := filepath.Join(tmpDir, fmt.Sprintf("c67_shebang_%s_%d", baseName, os.Getpid()))

	// Enable single-file mode for shebang scripts
	oldSingleFlag := SingleFlag
	SingleFlag = true
	defer func() { SingleFlag = oldSingleFlag }()

	// Compile (quietly unless verbose mode)
	err := CompileC67WithOptions(scriptPath, tmpExec, ctx.Platform, ctx.OptTimeout)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	defer os.Remove(tmpExec)

	// Execute with script arguments
	cmd := exec.Command(tmpExec, scriptArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("execution failed: %v", err)
	}

	return nil
}

// cmdBuildDir compiles all .c67 files in a directory
func cmdBuildDir(ctx *CommandContext, dirPath string) error {
	matches, err := filepath.Glob(filepath.Join(dirPath, "*.c67"))
	if err != nil {
		return fmt.Errorf("failed to find .c67 files: %v", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no .c67 files found in %s", dirPath)
	}

	if ctx.Verbose {
		fmt.Fprintf(os.Stderr, "Found %d .c67 file(s) in %s\n", len(matches), dirPath)
	}

	// When compiling a directory, don't enable single-file mode
	// This allows files in the same directory to share definitions
	oldSingleFlag := SingleFlag
	SingleFlag = false
	defer func() { SingleFlag = oldSingleFlag }()

	// Compile each file
	for _, file := range matches {
		outputPath := strings.TrimSuffix(filepath.Base(file), ".c67")

		if ctx.Verbose {
			fmt.Fprintf(os.Stderr, "Building %s -> %s (directory mode)\n", file, outputPath)
		}

		err := CompileC67WithOptions(file, outputPath, ctx.Platform, ctx.OptTimeout)
		if err != nil {
			return fmt.Errorf("compilation of %s failed: %v", file, err)
		}

		if !ctx.Quiet {
			fmt.Printf("Built: %s\n", outputPath)
		}
	}

	return nil
}

// cmdHelp displays usage information
func cmdHelp(ctx *CommandContext) error {
	fmt.Printf(`c67 - The C67 Compiler (Version 1.5.0)

USAGE:
    c67 <command> [arguments]

COMMANDS:
    build <file.c67>      Compile a C67 source file to an executable
    run <file.c67>        Compile and run a C67 program immediately
    help                   Show this help message
    version                Show version information

SHORTHAND:
    c67 <file.c67>      Same as 'c67 build <file.c67>'
    c67                  Show this help message (or build if .c67 files found)

FLAGS (can be used with any command):
    -o, --output <file>    Output executable filename (default: input name without .c67)
    -v, --verbose          Verbose mode (show detailed compilation info)
    -q, --quiet            Quiet mode (suppress progress messages)
    --arch <arch>          Target architecture: amd64, arm64, riscv64 (default: amd64)
    --os <os>              Target OS: linux, darwin, freebsd (default: linux)
    --target <platform>    Target platform: amd64-linux, arm64-macos, etc.
    --opt-timeout <secs>   Optimization timeout in seconds (default: 2.0)
    -u, --update-deps      Update dependency repositories from Git
    -s, --single           Compile single file only (don't load siblings)

EXAMPLES:
    # Compile a program
    c67 build hello.c67
    c67 build hello.c67 -o hello

    # Compile and run immediately
    c67 run hello.c67
    c67 run server.c67 --port 8080

    # Shorthand compilation
    c67 hello.c67

    # Shebang execution (add #!/usr/bin/c67 to first line of .c67 file)
    chmod +x script.c67
    ./script.c67 arg1 arg2

DOCUMENTATION:
    For language documentation, see LANGUAGESPEC.md
    For help or bug reports: https://github.com/anthropics/c67/issues

`)
	return nil
}
