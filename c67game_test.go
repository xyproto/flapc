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

// TestC67GameTest tests that c67game_test.c67 compiles and runs
func TestC67GameTest(t *testing.T) {
	c67gameDir := filepath.Join(os.Getenv("HOME"), "clones", "c67game")
	testPath := filepath.Join(c67gameDir, "c67game_test.c67")

	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Skip("c67game_test.c67 not found")
	}

	tmpDir := t.TempDir()
	binary := filepath.Join(tmpDir, "c67game_test")

	cmd := exec.Command("./c67", testPath, "-o", binary)
	output, err := cmd.CombinedOutput()

	if err != nil {
		t.Logf("Compilation output: %s", output)
		t.Fatalf("Failed to compile c67game_test.c67: %v", err)
	}

	// Run the test
	cmd = exec.Command(binary)
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Logf("Test output: %s", output)
		t.Fatalf("c67game_test.c67 failed: %v", err)
	}

	if !strings.Contains(string(output), "compiles successfully") {
		t.Errorf("Expected success message in output, got: %s", output)
	}

	t.Log("c67game_test.c67 passed")
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
