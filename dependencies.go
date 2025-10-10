package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FunctionRepository maps function names to Git repository URLs
// When the compiler encounters an unknown function, it looks it up here
// and automatically fetches the repository containing the implementation
var FunctionRepository = map[string]string{
	// Math functions
	// Keep these as auto-dependencies (can be pure Flap implementations):
	"abs": "github.com/xyproto/flap_math", // Simple: x < 0 { -> -x ~> x }
	"min": "github.com/xyproto/flap_math",
	"max": "github.com/xyproto/flap_math",

	// These have excellent x87 FPU instruction support - consider making builtin:
	// sqrt: SQRTSD (SSE2)
	// sin, cos, tan: FSIN, FCOS, FPTAN (x87)
	// asin, acos, atan: FPATAN (x87)
	// log, exp: FYL2X, F2XM1 (x87)
	// pow: FYL2X + F2XM1 (x87)
	// floor, ceil, round: FRNDINT (x87)
	"sqrt":  "github.com/xyproto/flap_math",
	"pow":   "github.com/xyproto/flap_math",
	"sin":   "github.com/xyproto/flap_math",
	"cos":   "github.com/xyproto/flap_math",
	"tan":   "github.com/xyproto/flap_math",
	"asin":  "github.com/xyproto/flap_math",
	"acos":  "github.com/xyproto/flap_math",
	"atan":  "github.com/xyproto/flap_math",
	"atan2": "github.com/xyproto/flap_math",
	"log":   "github.com/xyproto/flap_math",
	"log10": "github.com/xyproto/flap_math",
	"exp":   "github.com/xyproto/flap_math",
	"floor": "github.com/xyproto/flap_math",
	"ceil":  "github.com/xyproto/flap_math",
	"round": "github.com/xyproto/flap_math",

	// Standard library
	// "println": "github.com/xyproto/flap_core",  // Commented out - println is now a builtin

	// Graphics (example)
	"InitWindow":    "github.com/xyproto/flap_raylib",
	"CloseWindow":   "github.com/xyproto/flap_raylib",
	"DrawRectangle": "github.com/xyproto/flap_raylib",
}

// GetFunctionRepository returns the repository URL for a function
// Checks environment variable FLAPC_FUNCTIONNAME first, then falls back to FunctionRepository map
// Example: FLAPC_PRINTLN=github.com/xyproto/flap_alternative_core overrides the default
func GetFunctionRepository(funcName string) (string, bool) {
	// Check for environment variable override
	// Convert function name to uppercase for env var: println -> PRINTLN
	envVarName := "FLAPC_" + strings.ToUpper(funcName)
	if repoURL := os.Getenv(envVarName); repoURL != "" {
		if VerboseMode {
			fmt.Fprintf(os.Stderr, "Using environment override for %s: %s=%s\n", funcName, envVarName, repoURL)
		}
		return repoURL, true
	}

	// Fall back to FunctionRepository map
	repoURL, ok := FunctionRepository[funcName]
	return repoURL, ok
}

// GetCachePath returns the cache directory for flapc dependencies
// Respects XDG_CACHE_HOME environment variable
// Default: $XDG_CACHE_HOME/flapc or ~/.cache/flapc/
func GetCachePath() (string, error) {
	// Check XDG_CACHE_HOME first
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "flapc"), nil
	}

	// Fall back to ~/.cache/flapc
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	cachePath := filepath.Join(homeDir, ".cache", "flapc")
	return cachePath, nil
}

// GetRepoCachePath returns the local path for a cloned repository
// Example: "github.com/xyproto/flap_math" -> "~/.cache/flapc/github.com/xyproto/flap_math"
func GetRepoCachePath(repoURL string) (string, error) {
	cachePath, err := GetCachePath()
	if err != nil {
		return "", err
	}

	// Remove protocol prefix if present
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "http://")
	repoURL = strings.TrimPrefix(repoURL, "git://")

	// Remove .git suffix if present
	repoURL = strings.TrimSuffix(repoURL, ".git")

	return filepath.Join(cachePath, repoURL), nil
}

// EnsureRepoCloned ensures a repository is cloned to the cache
// If already cloned, does nothing (unless updateDeps is true)
func EnsureRepoCloned(repoURL string, updateDeps bool) (string, error) {
	repoPath, err := GetRepoCachePath(repoURL)
	if err != nil {
		return "", err
	}

	// Check if repo already exists
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Repo exists
		if updateDeps {
			if err := GitPull(repoPath); err != nil {
				return "", fmt.Errorf("failed to update %s: %w", repoURL, err)
			}
			fmt.Fprintf(os.Stderr, "Updated dependency: %s\n", repoURL)
		}
		return repoPath, nil
	}

	// Repo doesn't exist, clone it
	if err := GitClone(repoURL, repoPath); err != nil {
		return "", fmt.Errorf("failed to clone %s: %w", repoURL, err)
	}

	fmt.Fprintf(os.Stderr, "Cloned dependency: %s\n", repoURL)
	return repoPath, nil
}

// GitClone clones a Git repository to the specified path
// Strategy: Use latest tag if available, otherwise main branch, otherwise master
func GitClone(repoURL, destPath string) error {
	// Create parent directory
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Build full clone URL (add https:// if needed)
	cloneURL := repoURL
	if !strings.HasPrefix(repoURL, "http://") &&
		!strings.HasPrefix(repoURL, "https://") &&
		!strings.HasPrefix(repoURL, "git://") &&
		!strings.HasPrefix(repoURL, "git@") {
		cloneURL = "https://" + repoURL
	}

	// First, do a shallow clone (just enough to discover tags and branches)
	cmd := exec.Command("git", "clone", "--bare", cloneURL, destPath+".tmp")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w (tried to clone %s)", err, cloneURL)
	}

	// Determine what to checkout: latest tag > main > master
	checkoutRef, err := determineCheckoutRef(destPath + ".tmp")
	if err != nil {
		// Cleanup temp bare repo
		os.RemoveAll(destPath + ".tmp")
		return fmt.Errorf("failed to determine checkout ref: %w", err)
	}

	// Remove the temporary bare clone
	os.RemoveAll(destPath + ".tmp")

	// Now clone with the determined ref
	var cloneCmd *exec.Cmd
	if checkoutRef != "" {
		fmt.Fprintf(os.Stderr, "Cloning %s at %s...\n", repoURL, checkoutRef)
		cloneCmd = exec.Command("git", "clone", "--depth=1", "--branch", checkoutRef, cloneURL, destPath)
	} else {
		// No specific ref, let git use default
		cloneCmd = exec.Command("git", "clone", "--depth=1", cloneURL, destPath)
	}
	cloneCmd.Stdout = os.Stderr
	cloneCmd.Stderr = os.Stderr

	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w (tried to clone %s at %s)", err, cloneURL, checkoutRef)
	}

	return nil
}

// determineCheckoutRef determines which ref to checkout
// Priority: latest tag > main > master > empty (git default)
func determineCheckoutRef(bareRepoPath string) (string, error) {
	// Try to get the latest tag
	latestTag, err := getLatestTag(bareRepoPath)
	if err == nil && latestTag != "" {
		return latestTag, nil
	}

	// Try main branch
	if branchExists(bareRepoPath, "main") {
		return "main", nil
	}

	// Try master branch
	if branchExists(bareRepoPath, "master") {
		return "master", nil
	}

	// Let git use its default
	return "", nil
}

// getLatestTag returns the latest semver tag from a bare repository
func getLatestTag(bareRepoPath string) (string, error) {
	cmd := exec.Command("git", "--git-dir", bareRepoPath, "tag", "--sort=-version:refname")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) > 0 && tags[0] != "" {
		return tags[0], nil
	}

	return "", fmt.Errorf("no tags found")
}

// branchExists checks if a branch exists in a bare repository
func branchExists(bareRepoPath, branchName string) bool {
	cmd := exec.Command("git", "--git-dir", bareRepoPath, "show-ref", "--verify", "refs/heads/"+branchName)
	err := cmd.Run()
	return err == nil
}

// GitPull updates an existing Git repository
// Fetches latest tags and updates to latest tag, or latest main/master
func GitPull(repoPath string) error {
	// Fetch all updates including tags
	fetchCmd := exec.Command("git", "-C", repoPath, "fetch", "--all", "--tags")
	fetchCmd.Stdout = os.Stderr
	fetchCmd.Stderr = os.Stderr

	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// Determine the best ref to checkout
	latestTag, err := getLatestTagInRepo(repoPath)
	var checkoutRef string
	if err == nil && latestTag != "" {
		checkoutRef = latestTag
	} else {
		// Try origin/main, then origin/master
		if remoteBranchExists(repoPath, "origin/main") {
			checkoutRef = "origin/main"
		} else if remoteBranchExists(repoPath, "origin/master") {
			checkoutRef = "origin/master"
		} else {
			return fmt.Errorf("no suitable branch found (tried main, master)")
		}
	}

	// Checkout the determined ref
	checkoutCmd := exec.Command("git", "-C", repoPath, "checkout", checkoutRef)
	checkoutCmd.Stdout = os.Stderr
	checkoutCmd.Stderr = os.Stderr

	if err := checkoutCmd.Run(); err != nil {
		return fmt.Errorf("git checkout %s failed: %w", checkoutRef, err)
	}

	fmt.Fprintf(os.Stderr, "Updated to %s\n", checkoutRef)
	return nil
}

// getLatestTagInRepo returns the latest tag from a working repository
func getLatestTagInRepo(repoPath string) (string, error) {
	cmd := exec.Command("git", "-C", repoPath, "tag", "--sort=-version:refname")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(tags) > 0 && tags[0] != "" {
		return tags[0], nil
	}

	return "", fmt.Errorf("no tags found")
}

// remoteBranchExists checks if a remote branch exists
func remoteBranchExists(repoPath, branchName string) bool {
	cmd := exec.Command("git", "-C", repoPath, "show-ref", "--verify", "refs/remotes/"+branchName)
	err := cmd.Run()
	return err == nil
}

// FindFlapFiles returns all .flap files in a directory (recursively)
func FindFlapFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (like .git)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && path != dir {
			return filepath.SkipDir
		}

		// Add .flap files
		if !info.IsDir() && strings.HasSuffix(path, ".flap") {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// ResolveFunction looks up a function name and returns its repository URL
// Returns empty string if function is not in the repository map
// Checks environment variable FLAPC_FUNCTIONNAME first via GetFunctionRepository
func ResolveFunction(funcName string) string {
	if repoURL, ok := GetFunctionRepository(funcName); ok {
		return repoURL
	}
	return ""
}

// ResolveDependencies takes a list of unknown functions and returns
// unique repository URLs that need to be cloned
func ResolveDependencies(unknownFunctions []string) []string {
	repoSet := make(map[string]bool)

	for _, funcName := range unknownFunctions {
		if repoURL := ResolveFunction(funcName); repoURL != "" {
			repoSet[repoURL] = true
		}
	}

	// Convert set to slice
	repos := make([]string, 0, len(repoSet))
	for repo := range repoSet {
		repos = append(repos, repo)
	}

	return repos
}
