# Code to markdown

A CLI tool in Go that gathers source code from a repository and converts it into a single markdown file suitable for LLM input. This uses Cobra for command-line interface management.

## Features

**Command-line Interface:**
- Uses Cobra for robust CLI management
- Supports multiple flags for customization
- Provides help and usage information

**Smart File Detection:**
- Automatically detects common source code file extensions
- Excludes common build/dependency directories (node_modules, vendor, etc.)
- Handles special files like Dockerfile, Makefile
- Skips binary files automatically

**Flexible Filtering:**
- `--include` / `-i`: Specify file extensions to include
- `--exclude` / `-e`: Exclude specific file extensions
- `--exclude-dirs` / `-d`: Exclude specific directories
- `--max-size` / `-s`: Set maximum file size limit
- `--hidden` / `-H`: Include hidden files/directories

**Output Options:**
- `--output` / `-o`: Specify output file name
- `--verbose` / `-v`: Enable verbose logging
- Generates table of contents
- Includes file metadata (size, path)
- Uses proper syntax highlighting

## Usage Examples

```bash
# Basic usage - scan current directory
go run main.go

# Scan specific directory
go run main.go /path/to/repo

# Custom output file
go run main.go -o my-codebase.md

# Include only specific extensions
go run main.go -i .go,.py,.js

# Exclude certain extensions
go run main.go -e .log,.tmp

# Verbose output
go run main.go -v

# Include hidden files
go run main.go -H
```

## To use this tool:

1. Save the code as `main.go`
2. Initialize a Go module: `go mod init code2md`
3. Install Cobra: `go get github.com/spf13/cobra`
4. Build: `task build`
5. Run: `./code2md [options] [directory]`

The tool generates a well-structured markdown file that includes:
- Repository metadata
- Table of contents with links
- Each file's content with proper syntax highlighting
- File size and path information

This makes it easy to feed entire codebases to LLMs while maintaining readability and structure.
