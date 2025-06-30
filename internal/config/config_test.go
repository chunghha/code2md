package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestCmd is a helper to create a dummy cobra command with all our flags.
func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("output", "", "output file")
	cmd.Flags().Int64("max-size", 0, "max file size")
	cmd.Flags().Bool("hidden", false, "include hidden")
	cmd.Flags().Bool("verbose", false, "verbose output")
	cmd.Flags().StringSlice("exclude-dirs", []string{}, "dirs to exclude")
	cmd.Flags().StringSlice("include", []string{}, "exts to include")
	cmd.Flags().StringSlice("exclude", []string{}, "exts to exclude")

	return cmd
}

// setupTestEnvFile creates a temporary .env file for testing.
func setupTestEnvFile(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	envContent := []string{
		"CODE2MD_OUTPUT_FILE=from_env.md",
		"CODE2MD_MAX_FILE_SIZE=512000",
		"CODE2MD_INCLUDE_HIDDEN=true",
		"CODE2MD_VERBOSE=true",
		"CODE2MD_EXCLUDE_DIRS=.git,vendor,tmp",
	}

	envPath := filepath.Join(tmpDir, ".env")
	// G306: Use more restrictive permissions for .env file.
	if err := os.WriteFile(envPath, []byte(strings.Join(envContent, "\n")), 0600); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	return tmpDir
}

func TestMergeEnv_LoadsWhenFlagsNotSet(t *testing.T) {
	tmpDir := setupTestEnvFile(t)
	cmd := newTestCmd()
	cfg := &Config{
		OutputFile:  "default.md",
		MaxFileSize: 1024,
	}

	err := MergeEnv(cmd, cfg, tmpDir)
	if err != nil {
		t.Fatalf("MergeEnv returned an unexpected error: %v", err)
	}

	if cfg.OutputFile != "from_env.md" {
		t.Errorf("Expected OutputFile to be 'from_env.md', got %q", cfg.OutputFile)
	}

	if cfg.MaxFileSize != 512000 {
		t.Errorf("Expected MaxFileSize to be 512000, got %d", cfg.MaxFileSize)
	}

	if !cfg.IncludeHidden {
		t.Error("Expected IncludeHidden to be true, but it was false")
	}

	if len(cfg.ExcludeDirs) != 3 {
		t.Errorf("Expected ExcludeDirs to be parsed correctly, got %v", cfg.ExcludeDirs)
	}
}

func TestMergeEnv_RespectsUserFlags(t *testing.T) {
	tmpDir := setupTestEnvFile(t)
	cmd := newTestCmd()
	cfg := &Config{
		OutputFile:  "from_flag.md",
		MaxFileSize: 999,
	}

	// Mark flags as changed by the user, as Cobra would.
	if err := cmd.Flags().Set("output", "from_flag.md"); err != nil {
		t.Fatalf("test setup failed: could not set flag 'output': %v", err)
	}

	if err := cmd.Flags().Set("max-size", "999"); err != nil {
		t.Fatalf("test setup failed: could not set flag 'max-size': %v", err)
	}

	err := MergeEnv(cmd, cfg, tmpDir)
	if err != nil {
		t.Fatalf("MergeEnv returned an unexpected error: %v", err)
	}

	// Assert that the flag values are kept and not overwritten by .env.
	if cfg.OutputFile != "from_flag.md" {
		t.Errorf("Expected OutputFile to be 'from_flag.md', got %q", cfg.OutputFile)
	}

	if cfg.MaxFileSize != 999 {
		t.Errorf("Expected MaxFileSize to be 999, got %d", cfg.MaxFileSize)
	}
}
