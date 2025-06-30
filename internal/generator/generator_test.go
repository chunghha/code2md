package generator

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"Zero bytes", 0, "0 B"},
		{"Kilobytes", 1536, "1.5 KB"},
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
		{"Path with spaces", "path to/my file.go", "path-to-my-file-go"},
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
		{"Dockerfile", "Dockerfile", "dockerfile"},
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
