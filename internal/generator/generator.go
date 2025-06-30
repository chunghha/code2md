package generator

import (
	"bufio"
	"code2md/internal/config"
	"code2md/internal/gatherer"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MarkdownGenerator is responsible for creating the markdown file.
type MarkdownGenerator struct {
	config *config.Config
}

// NewMarkdownGenerator creates a new MarkdownGenerator.
func NewMarkdownGenerator(cfg *config.Config) *MarkdownGenerator {
	return &MarkdownGenerator{config: cfg}
}

// GenerateMarkdown creates the final markdown file from the gathered file info.
func (mg *MarkdownGenerator) GenerateMarkdown(files []gatherer.FileInfo, rootPath string) error {
	f, err := os.Create(mg.config.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}

	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", closeErr)
		}
	}()

	writer := bufio.NewWriter(f)

	defer func() {
		if flushErr := writer.Flush(); flushErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to flush buffer: %v\n", flushErr)
		}
	}()

	if err := writeHeader(writer, files, rootPath); err != nil {
		return err
	}

	if err := writeTableOfContents(writer, files); err != nil {
		return err
	}

	return writeFileContents(writer, files)
}

func writeHeader(writer *bufio.Writer, files []gatherer.FileInfo, rootPath string) error {
	if _, err := fmt.Fprintf(writer, "# Codebase Analysis\n\n"); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "**Repository:** %s  \n", rootPath); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "**Generated:** %s  \n", time.Now().Format("2006-01-02 15:04:05")); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "**Files:** %d  \n", len(files)); err != nil {
		return err
	}

	totalSize := calculateTotalSize(files)
	if _, err := fmt.Fprintf(writer, "**Total Size:** %s  \n\n", formatBytes(totalSize)); err != nil {
		return err
	}

	return nil
}

func calculateTotalSize(files []gatherer.FileInfo) int64 {
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	return totalSize
}

func writeTableOfContents(writer *bufio.Writer, files []gatherer.FileInfo) error {
	if _, err := fmt.Fprintf(writer, "## Table of Contents\n\n"); err != nil {
		return err
	}

	for _, file := range files {
		if _, err := fmt.Fprintf(writer, "- [%s](#%s)\n", file.Path, sanitizeAnchor(file.Path)); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(writer, "\n"); err != nil {
		return err
	}

	return nil
}

func writeFileContents(writer *bufio.Writer, files []gatherer.FileInfo) error {
	if _, err := fmt.Fprintf(writer, "## File Contents\n\n"); err != nil {
		return err
	}

	for _, file := range files {
		if err := writeFileSection(writer, file); err != nil {
			return err
		}
	}

	return nil
}

func writeFileSection(writer *bufio.Writer, file gatherer.FileInfo) error {
	if _, err := fmt.Fprintf(writer, "### %s\n\n", file.Path); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "**Size:** %s  \n", formatBytes(file.Size)); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "**Path:** `%s`  \n\n", file.Path); err != nil {
		return err
	}

	lang := getLanguageFromPath(file.Path)
	if _, err := fmt.Fprintf(writer, "```%s\n", lang); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(writer, "%s", file.Content); err != nil {
		return err
	}

	if !strings.HasSuffix(file.Content, "\n") {
		if _, err := fmt.Fprintf(writer, "\n"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(writer, "```\n\n"); err != nil {
		return err
	}

	return nil
}

func getLanguageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	fileName := strings.ToLower(filepath.Base(path))
	langMap := map[string]string{
		// --- THIS IS THE FIX ---
		".go": "go",
		// -----------------------
		".py": "python", ".js": "javascript", ".ts": "typescript",
		".jsx": "jsx", ".tsx": "tsx", ".java": "java", ".c": "c", ".cpp": "cpp",
		".cc": "cpp", ".cxx": "cpp", ".h": "c", ".hpp": "cpp", ".cs": "csharp",
		".php": "php", ".rb": "ruby", ".rs": "rust", ".swift": "swift", ".kt": "kotlin",
		".scala": "scala", ".sh": "bash", ".bash": "bash", ".zsh": "zsh", ".fish": "fish",
		".sql": "sql", ".html": "html", ".htm": "html", ".css": "css", ".scss": "scss",
		".sass": "sass", ".less": "less", ".vue": "vue", ".yaml": "yaml", ".yml": "yaml",
		".json": "json", ".xml": "xml", ".toml": "toml", ".ini": "ini", ".cfg": "ini",
		".conf": "ini", ".md": "markdown", ".txt": "text", ".rst": "rst",
		".dockerfile": "dockerfile",
	}

	if fileName == "dockerfile" || fileName == "makefile" {
		return strings.ToLower(fileName)
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	return "text"
}

func sanitizeAnchor(text string) string {
	result := strings.ToLower(text)
	result = strings.ReplaceAll(result, "/", "-")
	result = strings.ReplaceAll(result, "\\", "-")
	result = strings.ReplaceAll(result, ".", "-")
	result = strings.ReplaceAll(result, "_", "-")
	result = strings.ReplaceAll(result, " ", "-")

	return result
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
