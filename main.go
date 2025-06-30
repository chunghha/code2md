package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultMaxFileSize = 1024 * 1024 // 1MB
)

type Config struct {
	OutputFile    string
	IncludeExt    []string
	ExcludeExt    []string
	ExcludeDirs   []string
	MaxFileSize   int64
	IncludeHidden bool
	Verbose       bool
}

type FileInfo struct {
	Path    string
	Size    int64
	Content string
}

type FileGatherer struct {
	config   *Config
	rootPath string
}

type MarkdownGenerator struct {
	config *Config
}

func main() {
	if err := createRootCommand().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createRootCommand() *cobra.Command {
	var config Config

	rootCmd := &cobra.Command{
		Use:   "code2md [directory]",
		Short: "Convert source code repository to markdown for LLM consumption",
		Long: `A CLI tool that gathers all source code files from a repository
and converts them into a single markdown file suitable for feeding to Large Language Models.

The tool automatically detects common source code file extensions and excludes
common build/dependency directories to focus on the actual source code.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runCode2MD(&config, args)
		},
	}

	// Add flags
	rootCmd.Flags().StringVarP(&config.OutputFile, "output", "o", "codebase.md", "Output markdown file")
	rootCmd.Flags().StringSliceVarP(&config.IncludeExt, "include", "i", []string{}, "File extensions to include (e.g., .go,.py)")
	rootCmd.Flags().StringSliceVarP(&config.ExcludeExt, "exclude", "e", []string{}, "File extensions to exclude")
	rootCmd.Flags().StringSliceVarP(&config.ExcludeDirs, "exclude-dirs", "d", []string{}, "Directories to exclude")
	rootCmd.Flags().Int64VarP(&config.MaxFileSize, "max-size", "s", defaultMaxFileSize, "Maximum file size in bytes (default: 1MB)")
	rootCmd.Flags().BoolVarP(&config.IncludeHidden, "hidden", "H", false, "Include hidden files and directories")
	rootCmd.Flags().BoolVarP(&config.Verbose, "verbose", "v", false, "Verbose output")

	return rootCmd
}

func runCode2MD(config *Config, args []string) error {
	// Determine target directory
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	// Resolve absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("error resolving path: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Scanning directory: %s\n", absPath)
	}

	// Gather files
	gatherer := NewFileGatherer(config, absPath)

	files, err := gatherer.GatherFiles()
	if err != nil {
		return fmt.Errorf("error gathering files: %w", err)
	}

	if config.Verbose {
		fmt.Printf("Found %d files\n", len(files))
	}

	// Generate markdown
	generator := NewMarkdownGenerator(config)

	err = generator.GenerateMarkdown(files, absPath)
	if err != nil {
		return fmt.Errorf("error generating markdown: %w", err)
	}

	fmt.Printf("Successfully generated %s with %d files\n", config.OutputFile, len(files))

	return nil
}

func NewFileGatherer(config *Config, rootPath string) *FileGatherer {
	return &FileGatherer{
		config:   config,
		rootPath: rootPath,
	}
}

func (fg *FileGatherer) GatherFiles() ([]FileInfo, error) {
	var files []FileInfo

	// Prepare filters
	extInclude, extExclude := fg.prepareExtensionFilters()
	dirExclude := fg.prepareDirFilters()

	err := filepath.WalkDir(fg.rootPath, func(path string, d fs.DirEntry, err error) error {
		return fg.processWalkEntry(path, d, err, &files, extInclude, extExclude, dirExclude)
	})

	// Sort files by path for consistent output
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, err
}

func (fg *FileGatherer) prepareExtensionFilters() (extInclude, extExclude map[string]bool) {
	// Common source code extensions.
	defaultExtensions := []string{
		".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h", ".hpp",
		".cs", ".php", ".rb", ".rs", ".swift", ".kt", ".scala", ".sh",
		".sql", ".html", ".css", ".scss", ".less", ".vue", ".jsx", ".tsx",
		".yaml", ".yml", ".json", ".xml", ".toml", ".ini", ".cfg", ".conf",
		".md", ".txt", ".rst", ".dockerfile", "Dockerfile", "Makefile",
	}

	extInclude = make(map[string]bool)
	extExclude = make(map[string]bool)

	// If no include extensions specified, use defaults
	if len(fg.config.IncludeExt) == 0 {
		for _, ext := range defaultExtensions {
			extInclude[ext] = true
		}
	} else {
		for _, ext := range fg.config.IncludeExt {
			extInclude[ext] = true
		}
	}

	// Add exclude extensions
	for _, ext := range fg.config.ExcludeExt {
		extExclude[ext] = true
	}

	return extInclude, extExclude
}

func (fg *FileGatherer) prepareDirFilters() map[string]bool {
	// Common directories to exclude.
	defaultExcludeDirs := []string{
		".git", ".svn", ".hg", "node_modules", "vendor", "target", "build",
		"dist", "out", ".idea", ".vscode", "__pycache__", ".pytest_cache",
		".tox", "venv", ".env", ".venv", "env", ".DS_Store", "thumbs.db",
	}

	dirExclude := make(map[string]bool)
	for _, dir := range defaultExcludeDirs {
		dirExclude[dir] = true
	}

	for _, dir := range fg.config.ExcludeDirs {
		dirExclude[dir] = true
	}

	return dirExclude
}

func (fg *FileGatherer) processWalkEntry(path string, d fs.DirEntry, err error, files *[]FileInfo,
	extInclude, extExclude, dirExclude map[string]bool) error {
	if err != nil {
		if fg.config.Verbose {
			fmt.Printf("Warning: Cannot access %s: %v\n", path, err)
		}

		return nil // Continue walking
	}

	if fg.shouldSkipHidden(d.Name()) {
		if d.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	if d.IsDir() && dirExclude[d.Name()] {
		return filepath.SkipDir
	}

	if d.IsDir() {
		return nil
	}

	return fg.processFile(path, d, files, extInclude, extExclude)
}

func (fg *FileGatherer) shouldSkipHidden(name string) bool {
	return !fg.config.IncludeHidden && strings.HasPrefix(name, ".")
}

func (fg *FileGatherer) processFile(path string, d fs.DirEntry, files *[]FileInfo,
	extInclude, extExclude map[string]bool) error {
	// *** CHANGE IS HERE: Call the new method on fg ***
	if !fg.shouldIncludeFile(path, extInclude, extExclude) {
		return nil
	}

	info, err := d.Info()
	if err != nil {
		if fg.config.Verbose {
			fmt.Printf("Warning: Cannot get info for %s: %v\n", path, err)
		}

		return nil
	}

	if info.Size() > fg.config.MaxFileSize {
		if fg.config.Verbose {
			fmt.Printf("Skipping %s (size: %d bytes, max: %d bytes)\n",
				path, info.Size(), fg.config.MaxFileSize)
		}

		return nil
	}

	return fg.addFileToCollection(path, info, files)
}

func (fg *FileGatherer) shouldIncludeFile(path string, extInclude, extExclude map[string]bool) bool {
	fileName := filepath.Base(path)
	ext := filepath.Ext(path)

	// If we are including hidden files and this is a hidden file, prioritize it.
	// It should be included unless its extension is explicitly excluded.
	if fg.config.IncludeHidden && strings.HasPrefix(fileName, ".") {
		// For a file like `.env`, ext is `.env`. Check if it's excluded.
		if ext != "" && extExclude[ext] {
			return false
		}
		// For a file like `.bashrc`, ext is `.bashrc`. Check if it's excluded.
		if extExclude[fileName] {
			return false
		}

		return true // It's a hidden file we want, so include it.
	}

	// Special handling for files without extensions (e.g., Makefile)
	if ext == "" {
		return extInclude[fileName]
	}

	// Default behavior: check if extension is in the include list and not the exclude list.
	return extInclude[ext] && !extExclude[ext]
}

func (fg *FileGatherer) addFileToCollection(path string, info os.FileInfo, files *[]FileInfo) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if fg.config.Verbose {
			fmt.Printf("Warning: Cannot read %s: %v\n", path, err)
		}

		return nil
	}

	if isBinary(content) {
		if fg.config.Verbose {
			fmt.Printf("Skipping binary file: %s\n", path)
		}

		return nil
	}

	relPath, err := filepath.Rel(fg.rootPath, path)
	if err != nil {
		relPath = path
	}

	*files = append(*files, FileInfo{
		Path:    relPath,
		Size:    info.Size(),
		Content: string(content),
	})

	if fg.config.Verbose {
		fmt.Printf("Added: %s (%d bytes)\n", relPath, info.Size())
	}

	return nil
}

func NewMarkdownGenerator(config *Config) *MarkdownGenerator {
	return &MarkdownGenerator{config: config}
}

func (mg *MarkdownGenerator) GenerateMarkdown(files []FileInfo, rootPath string) error {
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

func writeHeader(writer *bufio.Writer, files []FileInfo, rootPath string) error {
	if _, err := fmt.Fprintf(writer, "# Codebase Analysis\n\n"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := fmt.Fprintf(writer, "**Repository:** %s  \n", rootPath); err != nil {
		return fmt.Errorf("failed to write repository info: %w", err)
	}

	if _, err := fmt.Fprintf(writer, "**Generated:** %s  \n", time.Now().Format("2006-01-02 15:04:05")); err != nil {
		return fmt.Errorf("failed to write generation time: %w", err)
	}

	if _, err := fmt.Fprintf(writer, "**Files:** %d  \n", len(files)); err != nil {
		return fmt.Errorf("failed to write file count: %w", err)
	}

	totalSize := calculateTotalSize(files)
	if _, err := fmt.Fprintf(writer, "**Total Size:** %s  \n\n", formatBytes(totalSize)); err != nil {
		return fmt.Errorf("failed to write total size: %w", err)
	}

	return nil
}

func calculateTotalSize(files []FileInfo) int64 {
	var totalSize int64
	for _, file := range files {
		totalSize += file.Size
	}

	return totalSize
}

func writeTableOfContents(writer *bufio.Writer, files []FileInfo) error {
	if _, err := fmt.Fprintf(writer, "## Table of Contents\n\n"); err != nil {
		return fmt.Errorf("failed to write table of contents header: %w", err)
	}

	for _, file := range files {
		if _, err := fmt.Fprintf(writer, "- [%s](#%s)\n", file.Path, sanitizeAnchor(file.Path)); err != nil {
			return fmt.Errorf("failed to write table of contents entry: %w", err)
		}
	}

	if _, err := fmt.Fprintf(writer, "\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

func writeFileContents(writer *bufio.Writer, files []FileInfo) error {
	if _, err := fmt.Fprintf(writer, "## File Contents\n\n"); err != nil {
		return fmt.Errorf("failed to write file contents header: %w", err)
	}

	for _, file := range files {
		if err := writeFileSection(writer, file); err != nil {
			return err
		}
	}

	return nil
}

func writeFileSection(writer *bufio.Writer, file FileInfo) error {
	if _, err := fmt.Fprintf(writer, "### %s\n\n", file.Path); err != nil {
		return fmt.Errorf("failed to write file header: %w", err)
	}

	if _, err := fmt.Fprintf(writer, "**Size:** %s  \n", formatBytes(file.Size)); err != nil {
		return fmt.Errorf("failed to write file size: %w", err)
	}

	if _, err := fmt.Fprintf(writer, "**Path:** `%s`  \n\n", file.Path); err != nil {
		return fmt.Errorf("failed to write file path: %w", err)
	}

	lang := getLanguageFromPath(file.Path)
	if _, err := fmt.Fprintf(writer, "```%s\n", lang); err != nil {
		return fmt.Errorf("failed to write code block start: %w", err)
	}

	if _, err := fmt.Fprintf(writer, "%s", file.Content); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	if !strings.HasSuffix(file.Content, "\n") {
		if _, err := fmt.Fprintf(writer, "\n"); err != nil {
			return fmt.Errorf("failed to write newline: %w", err)
		}
	}

	if _, err := fmt.Fprintf(writer, "```\n\n"); err != nil {
		return fmt.Errorf("failed to write code block end: %w", err)
	}

	return nil
}

func isBinary(data []byte) bool {
	// Simple heuristic: if file contains null bytes, it's likely binary
	for _, b := range data {
		if b == 0 {
			return true
		}
	}

	// Check for very high ratio of non-printable characters
	nonPrintable := 0

	for _, b := range data {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Allow tab, newline, carriage return
			nonPrintable++
		}
	}

	const maxNonPrintableRatio = 0.3
	if len(data) > 0 && float64(nonPrintable)/float64(len(data)) > maxNonPrintableRatio {
		return true
	}

	return false
}

func getLanguageFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	fileName := strings.ToLower(filepath.Base(path))

	langMap := map[string]string{
		".go":         "go",
		".py":         "python",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "jsx",
		".tsx":        "tsx",
		".java":       "java",
		".c":          "c",
		".cpp":        "cpp",
		".cc":         "cpp",
		".cxx":        "cpp",
		".h":          "c",
		".hpp":        "cpp",
		".cs":         "csharp",
		".php":        "php",
		".rb":         "ruby",
		".rs":         "rust",
		".swift":      "swift",
		".kt":         "kotlin",
		".scala":      "scala",
		".sh":         "bash",
		".bash":       "bash",
		".zsh":        "zsh",
		".fish":       "fish",
		".sql":        "sql",
		".html":       "html",
		".htm":        "html",
		".css":        "css",
		".scss":       "scss",
		".sass":       "sass",
		".less":       "less",
		".vue":        "vue",
		".yaml":       "yaml",
		".yml":        "yaml",
		".json":       "json",
		".xml":        "xml",
		".toml":       "toml",
		".ini":        "ini",
		".cfg":        "ini",
		".conf":       "ini",
		".md":         "markdown",
		".txt":        "text",
		".rst":        "rst",
		".dockerfile": "dockerfile",
	}

	// Special file names
	if fileName == "dockerfile" || fileName == "makefile" {
		return strings.ToLower(fileName)
	}

	if lang, exists := langMap[ext]; exists {
		return lang
	}

	return "text"
}

func sanitizeAnchor(text string) string {
	// Convert to lowercase and replace non-alphanumeric with hyphens
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
