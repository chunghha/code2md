# Code to Markdown

A CLI tool in Go that gathers source code from a repository and converts it into a single markdown file suitable for LLM input. This tool is fast, flexible, and highly configurable.

## Features

**Smart File Detection:**
- Automatically detects common source code file extensions.
- Excludes common build/dependency directories (`node_modules`, `vendor`, etc.).
- Handles special files like `Dockerfile` and `Makefile`.
- Skips binary files automatically.

**Flexible Filtering:**
- `--include` / `-i`: Specify file extensions to include.
- `--exclude` / `-e`: Exclude specific file extensions.
- `--exclude-dirs` / `-d`: Exclude specific directories.
- `--max-size` / `-s`: Set a maximum file size limit.
- `--hidden` / `-H`: Include hidden files and directories.

**Powerful Configuration:**
- All flags can be configured via a `.env` file for project-specific defaults.
- Command-line flags always take precedence over `.env` settings.

**High Performance & Observability:**
- **Concurrent file processing** for significantly faster scans on large repositories.
- **Structured logging** (`zap`) for clear and useful output, especially with the `--verbose` flag.

**Rich Output:**
- Generates a clean, well-structured markdown file.
- Includes repository metadata (path, file count, total size).
- Creates a clickable table of contents with links to each file.
- Uses proper syntax highlighting hints for each code block.

## Configuration

For project-specific settings, you can create a `.env` file in the directory you are scanning. The tool will automatically load it. Command-line flags will always override settings from the `.env` file.

Here are the available environment variables:

| Variable                 | Description                                      | Example                               |
| ------------------------ | ------------------------------------------------ | ------------------------------------- |
| `CODE2MD_OUTPUT_FILE`    | The name of the output markdown file.            | `CODE2MD_OUTPUT_FILE=project.md`      |
| `CODE2MD_MAX_FILE_SIZE`  | Maximum file size in bytes.                      | `CODE2MD_MAX_FILE_SIZE=2048000`       |
| `CODE2MD_INCLUDE_HIDDEN` | Set to `true` to include hidden files.           | `CODE2MD_INCLUDE_HIDDEN=true`         |
| `CODE2MD_VERBOSE`        | Set to `true` for detailed log output.           | `CODE2MD_VERBOSE=true`                |
| `CODE2MD_INCLUDE_EXT`    | Comma-separated list of file extensions to add.  | `CODE2MD_INCLUDE_EXT=.go,.js,.ts`     |
| `CODE2MD_EXCLUDE_EXT`    | Comma-separated list of file extensions to skip. | `CODE2MD_EXCLUDE_EXT=.log,.tmp`       |
| `CODE2MD_EXCLUDE_DIRS`   | Comma-separated list of directories to skip.     | `CODE2MD_EXCLUDE_DIRS=.git,build,dist` |

## Usage Examples

```bash
# Basic usage - scan current directory
./code2md

# Scan a specific directory
./code2md /path/to/your/project

# Override .env settings with a flag
./code2md -o custom-output.md

# Include only specific extensions
./code2md -i .go,.py,.js

# Exclude certain extensions
./code2md -e .log,.tmp

# Get detailed logging output
./code2md -v

# Include hidden files like .bashrc or .env
./code2md -H
```

## Installation & Building

**Using `go-task` (Recommended):**

This project uses `go-task` for simple build automation.

1.  [Install go-task](https://taskfile.dev/installation/).
2.  Run the build command from the project root:
    ```bash
    task build
    ```

**Using standard Go commands:**

1.  Clone the repository.
2.  From the project root, run the build command:
    ```bash
    go build -o code2md .
    ```

After building, you can run the tool with `./code2md [options] [directory]`.

---

### Credit

The prompt in GEMINI.md is from [Augmented Coding: Beyond the Vibes](https://tidyfirst.substack.com/p/augmented-coding-beyond-the-vibes)
