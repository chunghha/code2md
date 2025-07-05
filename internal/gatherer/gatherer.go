package gatherer

import (
	"code2md/internal/config"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// FileInfo holds the details of a gathered file.
type FileInfo struct {
	Path    string
	Size    int64
	Content string
}

// FileGatherer is responsible for collecting files from the filesystem.
type FileGatherer struct {
	config          *config.Config
	rootPath        string
	logger          *zap.Logger
	gitignoreParser *GitignoreParser
	gitignoreExists bool // Flag to track if .gitignore was found.
}

// NewFileGatherer creates a new FileGatherer.
func NewFileGatherer(cfg *config.Config, rootPath string, logger *zap.Logger) *FileGatherer {
	gitignoreParser := NewGitignoreParser(rootPath)
	err := gitignoreParser.LoadGitignore()

	// Check if the error was specifically "file does not exist".
	gitignoreExists := !os.IsNotExist(err)
	if err != nil && gitignoreExists {
		logger.Warn("Failed to load or parse .gitignore", zap.Error(err))
	}

	return &FileGatherer{
		config:          cfg,
		rootPath:        rootPath,
		logger:          logger,
		gitignoreParser: gitignoreParser,
		gitignoreExists: gitignoreExists,
	}
}

// GatherFiles orchestrates the concurrent file gathering pipeline.
func (fg *FileGatherer) GatherFiles(ctx context.Context) ([]FileInfo, error) {
	extInclude, extExclude := fg.prepareExtensionFilters()
	// Pass the gitignore existence flag to the directory filter preparation.
	dirExclude := fg.prepareDirFilters(fg.gitignoreExists)

	paths := make(chan string)
	results := make(chan FileInfo)
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return fg.producer(ctx, paths, dirExclude)
	})

	for i := 0; i < runtime.NumCPU(); i++ {
		g.Go(func() error {
			return fg.worker(ctx, paths, results, extInclude, extExclude)
		})
	}

	go func() {
		_ = g.Wait()

		close(results)
	}()

	var files []FileInfo //nolint:prealloc // The final size is unknown as files are received from a channel.
	for file := range results {
		files = append(files, file)
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// producer walks the filesystem and sends candidate file paths to the paths channel.
func (fg *FileGatherer) producer(ctx context.Context, paths chan<- string, dirExclude map[string]bool) error {
	defer close(paths)

	return filepath.WalkDir(fg.rootPath, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err != nil {
				fg.logger.Warn("Cannot access path", zap.String("path", path), zap.Error(err))
				return nil
			}

			// Always check gitignore first. This is the highest priority.
			if fg.gitignoreParser.ShouldIgnore(path) {
				if d.IsDir() {
					fg.logger.Debug("Skipping directory tree (gitignore)", zap.String("dir", path))
					return filepath.SkipDir
				}

				fg.logger.Debug("Skipping file (gitignore)", zap.String("file", path))

				return nil
			}

			// Handle default directory and hidden directory exclusions.
			if d.IsDir() {
				if dirExclude[d.Name()] || fg.shouldSkipHidden(d.Name()) {
					fg.logger.Debug("Skipping directory tree", zap.String("dir", d.Name()))
					return filepath.SkipDir
				}

				return nil
			}

			if fg.shouldSkipHidden(d.Name()) {
				return nil
			}

			paths <- path

			return nil
		}
	})
}

// worker receives file paths and performs the heavy processing.
func (fg *FileGatherer) worker(
	ctx context.Context,
	paths <-chan string,
	results chan<- FileInfo,
	extInclude, extExclude map[string]bool,
) error {
	for path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			fileInfo, shouldAdd := fg.processFile(path, extInclude, extExclude)
			if shouldAdd {
				results <- fileInfo
			}
		}
	}

	return nil
}

// processFile performs the "heavy" work on a single file path.
func (fg *FileGatherer) processFile(path string, extInclude, extExclude map[string]bool) (FileInfo, bool) {
	if !fg.shouldIncludeFile(path, extInclude, extExclude) {
		return FileInfo{}, false
	}

	info, err := os.Stat(path)
	if err != nil {
		fg.logger.Warn("Cannot get info for file", zap.String("path", path), zap.Error(err))
		return FileInfo{}, false
	}

	if info.Size() > fg.config.MaxFileSize {
		fg.logger.Debug("Skipping large file",
			zap.String("path", path),
			zap.Int64("size", info.Size()),
			zap.Int64("max_size", fg.config.MaxFileSize),
		)

		return FileInfo{}, false
	}

	content, err := os.ReadFile(path)
	if err != nil {
		fg.logger.Warn("Cannot read file", zap.String("path", path), zap.Error(err))
		return FileInfo{}, false
	}

	if isBinary(content) {
		fg.logger.Debug("Skipping binary file", zap.String("path", path))
		return FileInfo{}, false
	}

	relPath, err := filepath.Rel(fg.rootPath, path)
	if err != nil {
		relPath = path // Fallback to absolute path if Rel fails
	}

	fg.logger.Debug("Added file", zap.String("path", relPath))

	return FileInfo{
		Path:    relPath,
		Size:    info.Size(),
		Content: string(content),
	}, true
}

func (fg *FileGatherer) prepareExtensionFilters() (extInclude, extExclude map[string]bool) {
	extInclude = make(map[string]bool)
	extExclude = make(map[string]bool)

	if len(fg.config.IncludeExt) == 0 {
		for _, ext := range config.DefaultExtensions() {
			extInclude[ext] = true
		}
	} else {
		for _, ext := range fg.config.IncludeExt {
			extInclude[ext] = true
		}
	}

	for _, ext := range fg.config.ExcludeExt {
		extExclude[ext] = true
	}

	for _, file := range config.DefaultExcludeFiles() {
		extExclude[file] = true
	}

	return extInclude, extExclude
}

// prepareDirFilters now chooses which exclusion list to use.
func (fg *FileGatherer) prepareDirFilters(gitignoreExists bool) map[string]bool {
	dirExclude := make(map[string]bool)

	var defaultDirs []string

	if gitignoreExists {
		// .gitignore exists, so be minimal. Only exclude VCS directories.
		defaultDirs = []string{".git", ".svn", ".hg"}
	} else {
		// No .gitignore, so use the comprehensive "helpful" list.
		defaultDirs = config.DefaultExcludeDirs()
	}

	for _, dir := range defaultDirs {
		dirExclude[dir] = true
	}
	// Always add user-provided exclusions from the command line.
	for _, dir := range fg.config.ExcludeDirs {
		dirExclude[dir] = true
	}

	return dirExclude
}

func (fg *FileGatherer) shouldSkipHidden(name string) bool {
	return !fg.config.IncludeHidden && strings.HasPrefix(name, ".")
}

func (fg *FileGatherer) shouldIncludeFile(path string, extInclude, extExclude map[string]bool) bool {
	fileName := filepath.Base(path)
	ext := filepath.Ext(path)

	if extExclude[fileName] {
		return false
	}

	if fg.config.IncludeHidden && strings.HasPrefix(fileName, ".") {
		if ext != "" && extExclude[ext] {
			return false
		}

		if extExclude[fileName] {
			return false
		}

		return true
	}

	if ext == "" {
		return extInclude[fileName]
	}

	return extInclude[ext] && !extExclude[ext]
}

func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}

	nonPrintable := 0

	for _, b := range data {
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
	}

	const maxNonPrintableRatio = 0.3
	if len(data) > 0 && float64(nonPrintable)/float64(len(data)) > maxNonPrintableRatio {
		return true
	}

	return false
}
