package gatherer

import (
	"code2md/internal/config"
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"go.uber.org/zap"
)

// setupTestFileSystem is a helper to create a consistent test environment.
func setupTestFileSystem(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	createTestFile := func(filePath string, content string, size int64) {
		fullPath := filepath.Join(tmpDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", fullPath, err)
		}

		file, err := os.Create(fullPath)
		if err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}

		defer func() {
			if err := file.Close(); err != nil {
				t.Fatalf("Failed to close test file %s: %v", fullPath, err)
			}
		}()

		if size > 0 {
			if err := file.Truncate(size); err != nil {
				t.Fatalf("Failed to truncate file %s: %v", fullPath, err)
			}
		}

		if content != "" {
			if _, err := file.WriteString(content); err != nil {
				t.Fatalf("Failed to write to file %s: %v", fullPath, err)
			}
		}
	}
	createTestFile("main.go", "package main", 0)
	createTestFile("README.md", "# Test", 0)
	createTestFile("data.log", "some log data", 0)
	createTestFile("node_modules/lib.js", "// js", 0)
	createTestFile(".env", "SECRET=123", 0)
	createTestFile("src/bigfile.txt", "", 200)
	createTestFile("src/binary.bin", "hello\x00world", 0)
	createTestFile("Makefile", "build: all", 0)

	return tmpDir
}

func TestFileGatherer_GatherFiles_Default(t *testing.T) {
	tmpDir := setupTestFileSystem(t)
	cfg := &config.Config{
		MaxFileSize:   100,
		IncludeHidden: false,
	}
	// Create a no-op logger for this test
	logger := zap.NewNop()
	// Pass the logger to the gatherer
	gatherer := NewFileGatherer(cfg, tmpDir, logger)

	// Pass a context to GatherFiles
	files, err := gatherer.GatherFiles(context.Background())
	if err != nil {
		t.Fatalf("GatherFiles() returned an unexpected error: %v", err)
	}

	expectedFiles := []string{"Makefile", "README.md", "main.go"}
	assertFilePathsMatch(t, files, expectedFiles)
}

func TestFileGatherer_GatherFiles_WithHidden(t *testing.T) {
	tmpDir := setupTestFileSystem(t)
	cfg := &config.Config{
		MaxFileSize:   100,
		IncludeHidden: true,
	}
	logger := zap.NewNop()
	gatherer := NewFileGatherer(cfg, tmpDir, logger)

	files, err := gatherer.GatherFiles(context.Background())
	if err != nil {
		t.Fatalf("GatherFiles() returned an unexpected error: %v", err)
	}

	expectedFiles := []string{".env", "Makefile", "README.md", "main.go"}
	assertFilePathsMatch(t, files, expectedFiles)
}

func TestFileGatherer_Concurrency(t *testing.T) {
	tmpDir := setupTestFileSystem(t)
	cfg := &config.Config{
		MaxFileSize: 1024 * 1024,
	}
	logger := zap.NewNop()

	gatherer := NewFileGatherer(cfg, tmpDir, logger)
	// Override the default file reader with one that has an artificial delay
	gatherer.readFileFunc = func(path string) ([]byte, error) {
		time.Sleep(100 * time.Millisecond) // Artificial delay
		return os.ReadFile(path)
	}

	startTime := time.Now()

	_, err := gatherer.GatherFiles(context.Background())
	if err != nil {
		t.Fatalf("GatherFiles returned an unexpected error: %v", err)
	}

	duration := time.Since(startTime)

	// With concurrency, processing 4 valid files with a 100ms delay each
	// should take significantly less than 400ms (ideally just over 100ms).
	if duration > 300*time.Millisecond {
		t.Errorf("Expected concurrent processing to take less than 300ms, but it took %v", duration)
	}
}

// assertFilePathsMatch is a helper to avoid duplicating assertion logic.
func assertFilePathsMatch(t *testing.T, actualFiles []FileInfo, expectedPaths []string) {
	t.Helper()

	if len(actualFiles) != len(expectedPaths) {
		for _, f := range actualFiles {
			t.Logf("Got file: %s", f.Path)
		}

		t.Fatalf("Expected %d files, but got %d", len(expectedPaths), len(actualFiles))
	}

	gatheredPaths := make([]string, len(actualFiles))
	for i, f := range actualFiles {
		gatheredPaths[i] = f.Path
	}

	sort.Strings(gatheredPaths)
	sort.Strings(expectedPaths)

	for i, expectedPath := range expectedPaths {
		if gatheredPaths[i] != expectedPath {
			t.Errorf("Expected file %q at index %d, but got %q", expectedPath, i, gatheredPaths[i])
		}
	}
}
