package config

// Config holds all the configuration for the application.
type Config struct {
	OutputFile    string
	IncludeExt    []string
	ExcludeExt    []string
	ExcludeDirs   []string
	MaxFileSize   int64
	IncludeHidden bool
	Verbose       bool
}

// DefaultExtensions returns the default list of source code extensions.
func DefaultExtensions() []string {
	return []string{
		".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".h", ".hpp",
		".cs", ".php", ".rb", ".rs", ".swift", ".kt", ".scala", ".sh",
		".sql", ".html", ".css", ".scss", ".less", ".vue", ".jsx", ".tsx",
		".yaml", ".yml", ".json", ".xml", ".toml", ".ini", ".cfg", ".conf",
		".md", ".txt", ".rst", ".dockerfile", "Dockerfile", "Makefile",
	}
}

// DefaultExcludeDirs returns the default list of directories to exclude.
func DefaultExcludeDirs() []string {
	return []string{
		".git", ".svn", ".hg", "node_modules", "vendor", "target", "build",
		"dist", "out", ".idea", ".vscode", "__pycache__", ".pytest_cache",
		".tox", "venv", ".env", ".venv", "env", ".DS_Store", "thumbs.db",
	}
}
