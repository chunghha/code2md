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
	"sync"

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
	config       *config.Config
	rootPath     string
	logger       *zap.Logger
	readFileFunc func(path string) ([]byte, error) // For testability
}

// NewFileGatherer creates a new FileGatherer.
func NewFileGatherer(cfg *config.Config, rootPath string, logger *zap.Logger) *FileGatherer {
	return &FileGatherer{
		config:       cfg,
		rootPath:     rootPath,
		logger:       logger,
		readFileFunc: os.ReadFile, // Default to the real os.ReadFile
	}
}

// GatherFiles walks the directory and collects all relevant files concurrently.
func (fg *FileGatherer) GatherFiles(ctx context.Context) ([]FileInfo, error) {
	var (
		files []FileInfo
		mu    sync.Mutex
	)

	extInclude, extExclude := fg.prepareExtensionFilters()
	dirExclude := fg.prepareDirFilters()

	g, ctx := errgroup.WithContext(ctx)
	paths := make(chan string)

	// Start the producer goroutine.
	g.Go(func() error {
		defer close(paths)
		return fg.walkAndProduce(ctx, paths, dirExclude)
	})

	// Start the consumer (worker) goroutines.
	for i := 0; i < runtime.NumCPU(); i++ {
		g.Go(func() error {
			for path := range paths {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					if fileInfo, ok := fg.processFile(path, extInclude, extExclude); ok {
						mu.Lock()

						files = append(files, fileInfo)

						mu.Unlock()
					}
				}
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, nil
}

// walkAndProduce is the producer function that walks the filesystem.
func (fg *FileGatherer) walkAndProduce(ctx context.Context, paths chan<- string, dirExclude map[string]bool) error {
	return filepath.WalkDir(fg.rootPath, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err != nil {
				fg.logger.Warn("Cannot access path", zap.String("path", path), zap.Error(err))
				return nil // Continue walking
			}

			if fg.shouldSkip(d, dirExclude) {
				if d.IsDir() {
					return filepath.SkipDir
				}

				return nil
			}

			if d.IsDir() {
				return nil
			}

			paths <- path

			return nil
		}
	})
}

// shouldSkip determines if a directory entry should be skipped based on its name or type.
func (fg *FileGatherer) shouldSkip(d fs.DirEntry, dirExclude map[string]bool) bool {
	if !fg.config.IncludeHidden && strings.HasPrefix(d.Name(), ".") {
		return true
	}

	if d.IsDir() && dirExclude[d.Name()] {
		return true
	}

	return false
}

// processFile handles the logic for reading and validating a single file.
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

	content, err := fg.readFileFunc(path)
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
		relPath = path
	}

	fg.logger.Debug("Added file", zap.String("path", relPath), zap.Int64("size", info.Size()))

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

	return extInclude, extExclude
}

func (fg *FileGatherer) prepareDirFilters() map[string]bool {
	dirExclude := make(map[string]bool)
	for _, dir := range config.DefaultExcludeDirs() {
		dirExclude[dir] = true
	}

	for _, dir := range fg.config.ExcludeDirs {
		dirExclude[dir] = true
	}

	return dirExclude
}

func (fg *FileGatherer) shouldIncludeFile(path string, extInclude, extExclude map[string]bool) bool {
	fileName := filepath.Base(path)
	ext := filepath.Ext(path)

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
