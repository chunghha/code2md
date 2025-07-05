package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all the configuration for the application.
type Config struct {
	OutputFile    string   `envconfig:"OUTPUT_FILE"`
	IncludeExt    []string `envconfig:"INCLUDE_EXT"`
	ExcludeExt    []string `envconfig:"EXCLUDE_EXT"`
	ExcludeDirs   []string `envconfig:"EXCLUDE_DIRS"`
	MaxFileSize   int64    `envconfig:"MAX_SIZE"`
	IncludeHidden bool     `envconfig:"INCLUDE_HIDDEN"`
	Verbose       bool     `envconfig:"VERBOSE"`
	DryRun        bool     `envconfig:"DRY_RUN"`
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
		"coverage",
	}
}

// DefaultExcludeFiles returns the default list of specific files to exclude.
func DefaultExcludeFiles() []string {
	return []string{
		"pnpm-lock.yaml",
		"bun.lockb",
		"codebase.md",
	}
}

// Load populates a Config struct from environment variables and a .env file.
// It follows the standard precedence: .env file < actual environment variables.
func Load() (*Config, error) {
	// Load .env file. It's okay if it doesn't exist.
	_ = godotenv.Load()

	var c Config

	err := envconfig.Process("CODE2MD", &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}
