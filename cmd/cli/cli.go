package cli

import (
	"code2md/internal/config"
	"code2md/internal/gatherer"
	"code2md/internal/generator"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var version = "dev"

const defaultMaxFileSize = 1024 * 1024 // 1MB

func Execute() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading configuration from environment: %w", err)
	}

	var logger *zap.Logger
	if cfg.Verbose {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	// Defer a function that syncs the logger but specifically ignores the
	// benign "inappropriate ioctl for device" error.
	defer func() {
		if syncErr := logger.Sync(); syncErr != nil {
			// Check if the error is the specific syscall error we want to ignore.
			// This is a known issue when stdout/stderr is not a TTY.
			if !errors.Is(syncErr, syscall.ENOTTY) {
				fmt.Fprintf(os.Stderr, "Error syncing logger: %v\n", syncErr)
			}
		}
	}()

	return createRootCommand(cfg, logger).Execute()
}

func createRootCommand(cfg *config.Config, logger *zap.Logger) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "code2md [directory]",
		Short: "Convert source code repository to markdown for LLM consumption",
		Long: `A CLI tool that gathers all source code files from a repository
and converts them into a single markdown file suitable for feeding to Large Language Models.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCode2MD(cmd.Context(), cfg, logger, args)
		},
	}

	rootCmd.Version = version

	rootCmd.Flags().StringVarP(&cfg.OutputFile, "output", "o", "codebase.md", "Output markdown file")

	if cfg.OutputFile != "" {
		rootCmd.Flag("output").DefValue = cfg.OutputFile
	}

	rootCmd.Flags().StringSliceVarP(&cfg.IncludeExt, "include", "i", []string{}, "File extensions to include (e.g., .go,.py)")
	rootCmd.Flags().StringSliceVarP(&cfg.ExcludeExt, "exclude", "e", []string{}, "File extensions to exclude")
	rootCmd.Flags().StringSliceVarP(&cfg.ExcludeDirs, "exclude-dirs", "d", []string{}, "Directories to exclude")
	rootCmd.Flags().Int64VarP(&cfg.MaxFileSize, "max-size", "s", defaultMaxFileSize, "Maximum file size in bytes (default: 1MB)")

	if cfg.MaxFileSize != 0 {
		rootCmd.Flag("max-size").DefValue = fmt.Sprintf("%d", cfg.MaxFileSize)
	}

	rootCmd.Flags().BoolVarP(&cfg.IncludeHidden, "hidden", "H", false, "Include hidden files and directories")
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().BoolVar(&cfg.DryRun, "dry-run", false, "List files that would be included without generating the output file")

	return rootCmd
}

func runCode2MD(ctx context.Context, cfg *config.Config, logger *zap.Logger, args []string) error {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("error resolving path: %w", err)
	}

	logger.Info("Starting file gathering", zap.String("path", absPath))

	g := gatherer.NewFileGatherer(cfg, absPath, logger)

	files, err := g.GatherFiles(ctx)
	if err != nil {
		return fmt.Errorf("error gathering files: %w", err)
	}

	logger.Info("File gathering complete", zap.Int("file_count", len(files)))

	if cfg.DryRun {
		fmt.Println("Dry Run: The following files would be included in the output:")

		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.Path
		}

		sort.Strings(paths)

		for _, path := range paths {
			fmt.Println(path)
		}

		return nil
	}

	gen := generator.NewMarkdownGenerator(cfg)

	err = gen.GenerateMarkdown(files, absPath)
	if err != nil {
		return fmt.Errorf("error generating markdown: %w", err)
	}

	fmt.Printf("Successfully generated %s with %d files\n", cfg.OutputFile, len(files))

	return nil
}
