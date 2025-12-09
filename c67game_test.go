package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestC67GameLibraryCompiles tests that c67game library compiles
func TestC67GameLibraryCompiles(t *testing.T) {
	c67gameDir := filepath.Join(os.Getenv("HOME"), "clones", "c67game")
	gamePath := filepath.Join(c67gameDir, "game.c67")

	if _, err := os.Stat(gamePath); os.IsNotExist(err) {
		t.Skip("c67game library not found at ~/clones/c67game")
	}

	tmpDir := t.TempDir()
	binary := filepath.Join(tmpDir, "game_lib")
	cmd := exec.Command("./c67", gamePath, "-o", binary)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Compilation output: %s", output)
		t.Fatalf("Failed to compile c67game: %v", err)
	}

	t.Log("c67game library compiled successfully")
}

// TestC67GameTest tests that c67 test works in c67game directory
func TestC67GameTest(t *testing.T) {
	c67gameDir := filepath.Join(os.Getenv("HOME"), "clones", "c67game")
	testPath := filepath.Join(c67gameDir, "c67game_test.c67")

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("c67game_test.c67 not found")
	}

	// Run c67 test in the c67game directory
	c67Binary := filepath.Join(filepath.Dir(c67gameDir), "c67", "c67")
	cmd := exec.Command(c67Binary, "test")
	cmd.Dir = c67gameDir
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Test output: %s", output)
		t.Fatalf("c67 test failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "PASS") {
		t.Errorf("Expected PASS in output, got: %s", outputStr)
	}

	t.Log("c67game tests passed")
}

// TestC67GameSimpleProgram tests that a simple program using c67game compiles
func TestC67GameSimpleProgram(t *testing.T) {
	cmd := exec.Command("pkg-config", "--exists", "sdl3")
	if err := cmd.Run(); err != nil {
		t.Skip("SDL3 not installed")
	}

	c67gameDir := filepath.Join(os.Getenv("HOME"), "clones", "c67game")
	gamePath := filepath.Join(c67gameDir, "game.c67")

	if _, err := os.Stat(gamePath); os.IsNotExist(err) {
		t.Skip("c67game library not found")
	}

	// Create a simple program that uses c67game
	source := `import "` + gamePath + `"

main = {
    println("Testing c67game import")
}
`

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "test.c67")
	if err := os.WriteFile(srcFile, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}

	binary := filepath.Join(tmpDir, "test")
	cmd = exec.Command("./c67", srcFile, "-o", binary)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Compilation output: %s", output)
		t.Fatalf("Failed to compile: %v", err)
	}

	// Run it
	cmd = exec.Command(binary)
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Logf("Run output: %s", output)
		t.Fatalf("Failed to run: %v", err)
	}

	if !strings.Contains(string(output), "Testing c67game import") {
		t.Errorf("Expected 'Testing c67game import' in output, got: %s", output)
	}

	t.Log("Simple c67game program works")
}
