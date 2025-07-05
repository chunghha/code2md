package gatherer

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
)

// GitignoreParser handles parsing and matching gitignore patterns.
type GitignoreParser struct {
	patterns []glob.Glob
	basePath string
}

// NewGitignoreParser creates a new parser for the given directory.
func NewGitignoreParser(basePath string) *GitignoreParser {
	return &GitignoreParser{
		basePath: basePath,
	}
}

// LoadGitignore loads and translates patterns from a .gitignore file.
func (gp *GitignoreParser) LoadGitignore() (err error) {
	gitignorePath := filepath.Join(gp.basePath, ".gitignore")

	file, openErr := os.Open(gitignorePath)
	if openErr != nil {
		if os.IsNotExist(openErr) {
			return nil // No .gitignore file is not an error.
		}

		return openErr
	}

	defer func() {
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments, empty lines, and negation patterns.
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}

		// A single gitignore pattern can result in multiple glob patterns.
		patternsToCompile := translateGitignoreToGlobs(line)
		for _, p := range patternsToCompile {
			// We must compile with the separator to handle `**` correctly.
			if g, compileErr := glob.Compile(p, '/'); compileErr == nil {
				gp.patterns = append(gp.patterns, g)
			}
		}
	}

	return scanner.Err()
}

// translateGitignoreToGlobs converts a single .gitignore pattern into one or more glob patterns.
func translateGitignoreToGlobs(line string) []string {
	// A pattern ending with "/" signifies that it should only match directories.
	isDirPattern := strings.HasSuffix(line, "/")
	if isDirPattern {
		line = strings.TrimSuffix(line, "/")
	}

	// If a pattern does not contain a slash, it should match in any directory.
	// We use glob brace expansion `{,**/}` to match either the root or any subdirectory.
	if !strings.Contains(line, "/") {
		line = "{,**/}" + line
	} else if strings.HasPrefix(line, "/") {
		// A leading slash anchors the pattern to the root directory.
		line = strings.TrimPrefix(line, "/")
	}

	// A directory pattern must match the directory itself and everything inside it.
	// A file pattern must match the file and also a directory of the same name.
	return []string{line, line + "/**"}
}

// ShouldIgnore checks if a file path should be ignored based on gitignore patterns.
func (gp *GitignoreParser) ShouldIgnore(filePath string) bool {
	relPath, err := filepath.Rel(gp.basePath, filePath)
	if err != nil || relPath == "." {
		return false
	}
	// Use the system's native separator for matching, as the glob was compiled with it.
	relPath = filepath.ToSlash(relPath)

	for _, g := range gp.patterns {
		if g.Match(relPath) {
			return true
		}
	}

	return false
}
