package gatherer

import (
	"code2md/internal/config"
	"context"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

// assertFilePathsMatch is a helper function to compare the gathered file paths with an expected list.
func assertFilePathsMatch(t *testing.T, files []FileInfo, expected []string) {
	t.Helper()

	if len(files) != len(expected) {
		actualPaths := make([]string, len(files))
		for i, f := range files {
			actualPaths[i] = f.Path
		}

		t.Errorf("Expected %d files, but got %d", len(expected), len(files))
		t.Logf("Expected: %v", expected)
		t.Logf("Actual:   %v", actualPaths)

		return
	}

	for i, file := range files {
		if file.Path != expected[i] {
			t.Errorf("Expected file at index %d to be %q, but got %q", i, expected[i], file.Path)
		}
	}
}

func TestFileGatherer_GatherFiles_WithGitignore(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := zap.NewDevelopment()

	createTestFile := func(filePath string, content string) {
		fullPath := filepath.Join(tmpDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", fullPath, err)
		}
		// Corrected file permissions to satisfy gosec linter.
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	createTestFile(".gitignore", "*.log\nbuild/\ntemp.txt\n")
	createTestFile("main.go", "package main")
	createTestFile("README.md", "# Test")
	createTestFile("debug.log", "log content")
	createTestFile("build/output.txt", "build output")
	createTestFile("temp.txt", "temporary file")
	createTestFile("config.yaml", "key: value")

	cfg := &config.Config{
		MaxFileSize:   1024 * 1024,
		IncludeHidden: false,
	}
	gatherer := NewFileGatherer(cfg, tmpDir, logger)

	files, err := gatherer.GatherFiles(context.Background())
	if err != nil {
		t.Fatalf("GatherFiles() returned an unexpected error: %v", err)
	}

	expectedFiles := []string{"README.md", "config.yaml", "main.go"}
	assertFilePathsMatch(t, files, expectedFiles)
}

func TestFileGatherer_GitignoreComplexPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	logger, _ := zap.NewDevelopment()

	createTestFile := func(filePath string, content string) {
		fullPath := filepath.Join(tmpDir, filePath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", fullPath, err)
		}
		// Corrected file permissions to satisfy gosec linter.
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	gitignoreContent := `
# Ignore all .log files
*.log
# Ignore the build directory at the root
/build/
# Ignore all 'docs' directories
docs/
# Ignore files in any 'tmp' directory
**/tmp/data.txt
`
	createTestFile(".gitignore", gitignoreContent)
	createTestFile("main.go", "package main")
	createTestFile("src/build/somefile.txt", "not in root build dir")
	createTestFile("debug.log", "log content")
	createTestFile("build/output.txt", "in root build dir")
	createTestFile("src/docs/guide.md", "in a docs dir")
	createTestFile("app/tmp/data.txt", "in a tmp dir")

	cfg := &config.Config{
		MaxFileSize:   1024 * 1024,
		IncludeHidden: false,
	}
	gatherer := NewFileGatherer(cfg, tmpDir, logger)

	files, err := gatherer.GatherFiles(context.Background())
	if err != nil {
		t.Fatalf("GatherFiles() returned an unexpected error: %v", err)
	}

	expectedFiles := []string{"main.go", "src/build/somefile.txt"}
	assertFilePathsMatch(t, files, expectedFiles)
}
