# code2md

A high-performance, concurrent CLI tool that gathers all relevant source code from a repository and converts it into a single, well-structured markdown file. This output is perfect for providing context to Large Language Models (LLMs).

The tool is fast, configurable, and uses smart defaults to ignore irrelevant files and directories, focusing only on the code that matters.

## Features

**Smart & Fast Processing:**
- **Concurrent Scanning:** Processes files in parallel for maximum speed, using all available CPU cores.
- **Intelligent Filtering:** Automatically ignores common dependency directories (`node_modules`, `vendor`), build artifacts (`target`, `dist`), and VCS folders (`.git`).
- **Content-Aware Skipping:** Detects and skips binary files to keep the output clean.
- **Default Exclusions:** Ignores common lockfiles (`pnpm-lock.yaml`, `bun.lockb`) and its own output (`codebase.md`) by default.

**Powerful Configuration:**
- **Command-Line Flags:** Customize behavior on the fly for specific, one-off tasks.
- **Environment Variables:** Configure the tool globally using `CODE2MD_` prefixed variables.
- **`.env` File Support:** Automatically loads configuration from a `.env` file in the project root for repository-specific settings.

**Flexible Output:**
- **Custom Output File:** Specify the name of the generated markdown file.
- **File & Directory Filtering:** Use flags or environment variables to include/exclude specific file extensions or directories.
- **Size & Visibility Control:** Set a maximum file size to ignore large assets and choose whether to include hidden files and folders.
- **Structured Markdown:** Generates a clean markdown file with a header, a linked table of contents, and properly syntax-highlighted code blocks for each file.
- **Verbose Logging:** Use the `--verbose` flag to see detailed logs of the scanning process.

## Installation

Ensure you have Go installed (version 1.18 or newer).

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/chunghha/code2md.git
    cd code2md
    ```

2.  **Build the binary:**
    ```bash
    task build
    ```
    or
    ```bash
    go build -o code2md .
    ```

3.  **(Optional) Move the binary to a directory in your PATH:**
    To install code2md to **~/bin/**
    ```bash
    task install
    ```
    or
    ```bash
    mv code2md /usr/local/bin/
    ```

## Usage

**Basic Usage:**
Scan the current directory and create `codebase.md`.
```bash
code2md
```

**Scan a specific directory:**
```bash
code2md /path/to/your/project
```

**Customizing with Flags:**
```bash
# Specify a different output file
code2md -o my_project.md

# Include only Go and Python files
code2md -i .go,.py

# Exclude all test files
code2md -e _test.go

# Exclude the 'dist' and 'coverage' directories
code2md -d dist,coverage

# Include hidden files (e.g., .github, .vscode)
code2md -H

# See everything the tool is doing
code2md --verbose
```

## Configuration

The tool uses a flexible configuration system with a clear order of precedence.

### Configuration Precedence

Settings are applied in the following order. Each level overrides the previous one:
1.  **Defaults:** Sensible built-in values.
2.  **`.env` File:** Values loaded from a `.env` file in the directory where `code2md` is run.
3.  **Environment Variables:** System-wide variables prefixed with `CODE2MD_`.
4.  **Command-Line Flags:** The highest precedence, for specific, one-time overrides.

### Environment Variables

You can set the following environment variables. For example, you can add `export CODE2MD_EXCLUDE_DIRS="dist,build,coverage"` to your `.bashrc` or `.zshrc`.

| Variable                  | Flag (`--`)    | Type           | Description                                      |
| ------------------------- | -------------- | -------------- | ------------------------------------------------ |
| `CODE2MD_OUTPUT_FILE`     | `output`       | `string`       | Path for the output markdown file.               |
| `CODE2MD_INCLUDE_EXT`     | `include`      | `string` (csv) | Comma-separated list of file extensions to include. |
| `CODE2MD_EXCLUDE_EXT`     | `exclude`      | `string` (csv) | Comma-separated list of file extensions to exclude. |
| `CODE2MD_EXCLUDE_DIRS`    | `exclude-dirs` | `string` (csv) | Comma-separated list of directories to exclude.  |
| `CODE2MD_MAX_SIZE`        | `max-size`     | `int`          | Maximum file size in bytes.                      |
| `CODE2MD_INCLUDE_HIDDEN`  | `hidden`       | `bool`         | Set to `true` to include hidden files.           |
| `CODE2MD_VERBOSE`         | `verbose`      | `bool`         | Set to `true` for detailed logging.              |

## Development

This project uses `go-task` for task automation.

-   **Run all tests:** `task test`
-   **Run linters:** `task lint`
-   **Build the binary:** `task build`
---

### Credit

The prompt in GEMINI.md is from [Augmented Coding: Beyond the Vibes](https://tidyfirst.substack.com/p/augmented-coding-beyond-the-vibes)
