package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

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

// MergeEnv loads configuration from a .env file and merges it into the config.
func MergeEnv(cmd *cobra.Command, cfg *Config, dir string) error {
	envPath := filepath.Join(dir, ".env")
	if err := godotenv.Load(envPath); err != nil {
		if os.IsNotExist(err) {
			return nil // .env file is optional, not an error.
		}

		return err // Report other errors (e.g., permissions).
	}

	// flagProcessors defines how to load each flag from an environment variable.
	flagProcessors := map[string]struct {
		envVar string
		apply  func(value string)
	}{
		"output":       {envVar: "CODE2MD_OUTPUT_FILE", apply: func(v string) { cfg.OutputFile = v }},
		"include":      {envVar: "CODE2MD_INCLUDE_EXT", apply: func(v string) { cfg.IncludeExt = strings.Split(v, ",") }},
		"exclude":      {envVar: "CODE2MD_EXCLUDE_EXT", apply: func(v string) { cfg.ExcludeExt = strings.Split(v, ",") }},
		"exclude-dirs": {envVar: "CODE2MD_EXCLUDE_DIRS", apply: func(v string) { cfg.ExcludeDirs = strings.Split(v, ",") }},
		"max-size": {
			envVar: "CODE2MD_MAX_FILE_SIZE",
			apply: func(v string) {
				if intVal, err := strconv.ParseInt(v, 10, 64); err == nil {
					cfg.MaxFileSize = intVal
				}
			},
		},
		"hidden": {
			envVar: "CODE2MD_INCLUDE_HIDDEN",
			apply: func(v string) {
				if boolVal, err := strconv.ParseBool(v); err == nil {
					cfg.IncludeHidden = boolVal
				}
			},
		},
		"verbose": {
			envVar: "CODE2MD_VERBOSE",
			apply: func(v string) {
				if boolVal, err := strconv.ParseBool(v); err == nil {
					cfg.Verbose = boolVal
				}
			},
		},
	}

	// Iterate over the map and process each flag if it wasn't set on the command line.
	for flagName, p := range flagProcessors {
		if !cmd.Flags().Changed(flagName) {
			if val, ok := os.LookupEnv(p.envVar); ok {
				p.apply(val)
			}
		}
	}

	return nil
}
