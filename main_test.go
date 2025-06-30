package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Bytes", 500, "500 B"},
		{"Kilobytes", 1536, "1.5 KB"},
		{"Megabytes", 1048576, "1.0 MB"},
		{"Gigabytes", 1610612736, "1.5 GB"},
		{"Edge case unit", 1023, "1023 B"},
		{"Exact unit", 1024, "1.0 KB"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := formatBytes(tc.bytes)
			if actual != tc.expected {
				t.Errorf("formatBytes(%d): expected %q, got %q", tc.bytes, tc.expected, actual)
			}
		})
	}
}

func TestSanitizeAnchor(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Normal path", "path/to/file.go", "path-to-file-go"},
		{"Path with backslashes", "path\\to\\file.go", "path-to-file-go"},
		{"Path with spaces", "path to/my file.go", "path-to-my-file-go"},
		{"Path with underscores", "my_path/to_file.go", "my-path-to-file-go"},
		{"Mixed special characters", "a.b_c/d\\e f", "a-b-c-d-e-f"},
		{"Empty string", "", ""},
		{"Single dot", ".", "-"},
		{"Starts with slash", "/path/to/file", "-path-to-file"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := sanitizeAnchor(tc.input)
			if actual != tc.expected {
				t.Errorf("sanitizeAnchor(%q): expected %q, got %q", tc.input, tc.expected, actual)
			}
		})
	}
}

func TestGetLanguageFromPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected string
	}{
		{"Go file", "main.go", "go"},
		{"Python file", "script.py", "python"},
		{"JavaScript file", "app.js", "javascript"},
		{"TypeScript file", "component.ts", "typescript"},
		{"C++ header", "utils.hpp", "cpp"},
		{"C# file", "Program.cs", "csharp"},
		{"Dockerfile", "Dockerfile", "dockerfile"},
		{"Makefile", "Makefile", "makefile"},
		{"YAML file", "config.yml", "yaml"},
		{"Unknown extension", "file.unknown", "text"},
		{"No extension", "README", "text"},
		{"Uppercase extension", "IMAGE.JPG", "text"}, // Assuming .jpg is not in the map
		{"Path with folder", "src/app/main.go", "go"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := getLanguageFromPath(tc.path)
			if actual != tc.expected {
				t.Errorf("getLanguageFromPath(%q): expected %q, got %q", tc.path, tc.expected, actual)
			}
		})
	}
}

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
		// This defer is now updated to check the error from Close()
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

	// Create a diverse set of files and directories
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

	config := &Config{
		MaxFileSize:   100,
		IncludeHidden: false,
	}
	gatherer := NewFileGatherer(config, tmpDir)

	files, err := gatherer.GatherFiles()
	if err != nil {
		t.Fatalf("GatherFiles() returned an unexpected error: %v", err)
	}

	expectedFiles := []string{"Makefile", "README.md", "main.go"}
	assertFilePathsMatch(t, files, expectedFiles)
}

func TestFileGatherer_GatherFiles_WithHidden(t *testing.T) {
	tmpDir := setupTestFileSystem(t)

	config := &Config{
		MaxFileSize:   100,
		IncludeHidden: true,
	}
	gatherer := NewFileGatherer(config, tmpDir)

	files, err := gatherer.GatherFiles()
	if err != nil {
		t.Fatalf("GatherFiles() returned an unexpected error: %v", err)
	}

	expectedFiles := []string{".env", "Makefile", "README.md", "main.go"}
	assertFilePathsMatch(t, files, expectedFiles)
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
