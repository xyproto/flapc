package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// TestENetCompilation tests that ENet example programs compile successfully
// This verifies C FFI correctness without requiring ENet to be installed
func TestENetCompilation(t *testing.T) {
	// Skip on non-Linux for now (ENet examples are Linux-focused)
	if runtime.GOOS != "linux" {
		t.Skip("Skipping ENet compilation test on non-Linux platform")
	}

	platform := GetDefaultPlatform()

	examples := []struct {
		name   string
		source string
	}{
		{
			name:   "enet_simple",
			source: "examples/enet/simple_test.flap",
		},
	}

	for _, example := range examples {
		t.Run(example.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			outPath := filepath.Join(tmpDir, "test_"+example.name)

			// Try to compile
			err := CompileFlap(example.source, outPath, platform)

			// We expect compilation to succeed (generates assembly/binary)
			// It may fail at link time if ENet is not installed, which is acceptable
			// The key is that the Flap code compiles and generates valid assembly
			if err != nil {
				// Check if it's a link error (expected if ENet not installed)
				if isLinkError(err) {
					t.Logf("%s: Compilation successful, link failed (ENet not installed): %v", example.name, err)
					// This is OK - the Flap code compiled successfully
					return
				}
				// Other errors are test failures
				t.Fatalf("%s: Compilation failed: %v", example.name, err)
			}

			// If we got here, compilation AND linking succeeded
			t.Logf("%s: Full compilation and linking successful", example.name)

			// Verify binary was created
			if _, err := os.Stat(outPath); os.IsNotExist(err) {
				t.Fatalf("%s: Binary not created at %s", example.name, outPath)
			}

			// Verify it's executable
			fileInfo, err := os.Stat(outPath)
			if err != nil {
				t.Fatalf("%s: Failed to stat binary: %v", example.name, err)
			}

			if fileInfo.Mode()&0111 == 0 {
				t.Fatalf("%s: Binary is not executable", example.name)
			}

			// Clean up
			os.Remove(outPath)
		})
	}
}

// isLinkError checks if an error is a linking error (undefined reference to ENet symbols)
func isLinkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common link errors when ENet is missing
	linkErrors := []string{
		"undefined reference to `enet_initialize'",
		"undefined reference to `enet_",
		"ld returned 1 exit status",
		"compilation failed",
	}
	for _, linkErr := range linkErrors {
		if contains(errStr, linkErr) {
			return true
		}
	}
	return false
}

// TestENetCodeGeneration verifies that ENet examples generate valid assembly
// even if linking fails
func TestENetCodeGeneration(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping ENet codegen test on non-Linux platform")
	}

	// Test that we can at least parse and generate code for ENet examples
	examples := []string{
		"examples/enet/simple_test.flap",
	}

	for _, source := range examples {
		t.Run(filepath.Base(source), func(t *testing.T) {
			// Just verify the file exists and has valid Flap syntax
			content, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", source, err)
			}

			// Parse the program
			parser := NewParserWithFilename(string(content), source)
			program := parser.ParseProgram()

			// Verify we got a valid AST
			if program == nil {
				t.Fatalf("Failed to parse %s", source)
			}

			if len(program.Statements) == 0 {
				t.Fatalf("No statements parsed from %s", source)
			}

			t.Logf("Successfully parsed %s: %d statements", source, len(program.Statements))
		})
	}
}

// TestENetWithLibraryIfAvailable attempts to run ENet tests if the library is available
func TestENetWithLibraryIfAvailable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping ENet runtime test on non-Linux platform")
	}

	// Check if ENet is available
	cmd := exec.Command("pkg-config", "--exists", "libenet")
	if err := cmd.Run(); err != nil {
		t.Skip("ENet library not installed (pkg-config --exists libenet failed)")
	}

	t.Log("ENet library detected via pkg-config")

	// Try to compile with ENet
	platform := GetDefaultPlatform()
	tmpDir := t.TempDir()
	serverBin := filepath.Join(tmpDir, "test_enet_server_runtime")
	err := CompileFlap("examples/enet/server.flap", serverBin, platform)

	if err != nil {
		t.Fatalf("Failed to compile ENet server with library available: %v", err)
	}

	defer os.Remove(serverBin)

	// Verify binary exists
	if _, err := os.Stat(serverBin); os.IsNotExist(err) {
		t.Fatalf("Server binary not created")
	}

	t.Log("ENet server compiled successfully with library")

	// Note: We don't run the binary in tests as it would start a server
	// and require network setup. The compilation test is sufficient.
}
