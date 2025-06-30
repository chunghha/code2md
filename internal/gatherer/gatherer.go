package gatherer

import (
	"code2md/internal/config"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// FileInfo holds the details of a gathered file.
type FileInfo struct {
	Path    string
	Size    int64
	Content string
}

// FileGatherer is responsible for collecting files from the filesystem.
type FileGatherer struct {
	config   *config.Config
	rootPath string
}

// NewFileGatherer creates a new FileGatherer.
func NewFileGatherer(cfg *config.Config, rootPath string) *FileGatherer {
	return &FileGatherer{
		config:   cfg,
		rootPath: rootPath,
	}
}

// GatherFiles walks the directory and collects all relevant files.
func (fg *FileGatherer) GatherFiles() ([]FileInfo, error) {
	var files []FileInfo

	extInclude, extExclude := fg.prepareExtensionFilters()
	dirExclude := fg.prepareDirFilters()

	err := filepath.WalkDir(fg.rootPath, func(path string, d fs.DirEntry, err error) error {
		return fg.processWalkEntry(path, d, err, &files, extInclude, extExclude, dirExclude)
	})

	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	return files, err
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

func (fg *FileGatherer) processWalkEntry(path string, d fs.DirEntry, err error, files *[]FileInfo,
	extInclude, extExclude, dirExclude map[string]bool) error {
	if err != nil {
		if fg.config.Verbose {
			fmt.Printf("Warning: Cannot access %s: %v\n", path, err)
		}

		return nil // Continue walking
	}

	if fg.shouldSkipHidden(d.Name()) {
		if d.IsDir() {
			return filepath.SkipDir
		}

		return nil
	}

	if d.IsDir() && dirExclude[d.Name()] {
		return filepath.SkipDir
	}

	if d.IsDir() {
		return nil
	}

	return fg.processFile(path, d, files, extInclude, extExclude)
}

func (fg *FileGatherer) shouldSkipHidden(name string) bool {
	return !fg.config.IncludeHidden && strings.HasPrefix(name, ".")
}

func (fg *FileGatherer) processFile(path string, d fs.DirEntry, files *[]FileInfo,
	extInclude, extExclude map[string]bool) error {
	if !fg.shouldIncludeFile(path, extInclude, extExclude) {
		return nil
	}

	info, err := d.Info()
	if err != nil {
		if fg.config.Verbose {
			fmt.Printf("Warning: Cannot get info for %s: %v\n", path, err)
		}

		return nil
	}

	if info.Size() > fg.config.MaxFileSize {
		if fg.config.Verbose {
			fmt.Printf("Skipping %s (size: %d bytes, max: %d bytes)\n",
				path, info.Size(), fg.config.MaxFileSize)
		}

		return nil
	}

	return fg.addFileToCollection(path, info, files)
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

func (fg *FileGatherer) addFileToCollection(path string, info os.FileInfo, files *[]FileInfo) error {
	content, err := os.ReadFile(path)
	if err != nil {
		if fg.config.Verbose {
			fmt.Printf("Warning: Cannot read %s: %v\n", path, err)
		}

		return nil
	}

	if isBinary(content) {
		if fg.config.Verbose {
			fmt.Printf("Skipping binary file: %s\n", path)
		}

		return nil
	}

	relPath, err := filepath.Rel(fg.rootPath, path)
	if err != nil {
		relPath = path
	}

	*files = append(*files, FileInfo{
		Path:    relPath,
		Size:    info.Size(),
		Content: string(content),
	})

	if fg.config.Verbose {
		fmt.Printf("Added: %s (%d bytes)\n", relPath, info.Size())
	}

	return nil
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
