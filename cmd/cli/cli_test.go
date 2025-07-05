package cli

import (
	"bytes"
	"code2md/internal/config"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// setupTestFileSystem is a helper to create a temporary directory with files for testing.
func setupTestFileSystem(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	createTestFile := func(filePath string, content string) {
		fullPath := filepath.Join(tmpDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", fullPath, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write file %s: %v", err, filePath)
		}
	}

	createTestFile("main.go", "package main")
	createTestFile("README.md", "# Test")
	createTestFile("internal/helper.go", "package internal")
	// This file should be excluded by default
	createTestFile("node_modules/lib.js", "// js")

	return tmpDir
}

func TestRunCode2MD_DryRun(t *testing.T) {
	// 1. Setup
	tmpDir := setupTestFileSystem(t)
	logger := zap.NewNop() // Use a no-op logger to keep test output clean.
	outputFileName := "test_output.md"

	cfg := &config.Config{
		DryRun:      true,
		OutputFile:  outputFileName,
		MaxFileSize: 1024 * 1024, // 1MB, a sensible default
	}

	// 2. Redirect standard output to capture the printed list of files.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// 3. Execute the function
	err := runCode2MD(context.Background(), cfg, logger, []string{tmpDir})
	if err != nil {
		t.Fatalf("runCode2MD returned an unexpected error: %v", err)
	}

	// 4. Stop redirecting and read the captured output.
	if err := w.Close(); err != nil {
		t.Fatalf("Failed to close pipe writer: %v", err)
	}

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("Failed to read from pipe reader: %v", err)
	}

	os.Stdout = oldStdout

	// 5. Assert the results
	output := buf.String()

	expectedFiles := []string{"README.md", "internal/helper.go", "main.go"}
	for _, expected := range expectedFiles {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected dry run output to contain %q, but it did not.\nOutput was:\n%s", expected, output)
		}
	}

	// Assert that the output file was NOT created.
	finalOutputPath := filepath.Join(tmpDir, outputFileName)
	if _, err := os.Stat(finalOutputPath); !os.IsNotExist(err) {
		t.Errorf("Expected output file %q NOT to be created in dry run mode, but it was.", finalOutputPath)
	}
}
