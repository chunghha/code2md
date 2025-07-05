package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	// 1. Setup: Create a temporary .env file
	tmpDir := t.TempDir()
	envFilePath := filepath.Join(tmpDir, ".env")

	envContent := "CODE2MD_OUTPUT_FILE=test_from_env_file.md\nCODE2MD_VERBOSE=true"
	if err := os.WriteFile(envFilePath, []byte(envContent), 0600); err != nil {
		t.Fatalf("Failed to write .env file: %v", err)
	}

	// 2. Set a conflicting environment variable to test precedence
	t.Setenv("CODE2MD_OUTPUT_FILE", "test_from_env.md")
	// Set a new variable to test direct env loading
	t.Setenv("CODE2MD_MAX_SIZE", "512")

	// 3. Change to the directory with the .env file to ensure it's found
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if err = os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	defer func() {
		if err = os.Chdir(originalWd); err != nil {
			t.Fatalf("Failed to change back to original directory: %v", err)
		}
	}()

	// 4. Run the function to be tested
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned an unexpected error: %v", err)
	}

	// 5. Assert the results
	if cfg.OutputFile != "test_from_env.md" {
		t.Errorf("Expected OutputFile to be 'test_from_env.md' (from env), but got %q", cfg.OutputFile)
	}

	if cfg.MaxFileSize != 512 {
		t.Errorf("Expected MaxFileSize to be 512 (from env), but got %d", cfg.MaxFileSize)
	}

	if !cfg.Verbose {
		t.Error("Expected Verbose to be true (from .env file), but got false")
	}
}
