package internal

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/benfiola/homelab-images/shared/pkg/logging"
)

var mediaExtensions = map[string]bool{
	".mkv": true, ".mp4": true, ".m4v": true, ".avi": true, ".ts": true, ".mov": true,
}

const tmpFilePrefix = ".loudnorm.tmp."

// TmpPathFor returns the same-directory temp path used while replacing file,
// so the final rename is atomic and same-filesystem.
func TmpPathFor(file string) string {
	return filepath.Join(filepath.Dir(file), tmpFilePrefix+filepath.Base(file))
}

// BackupPathFor mirrors file's absolute path under backupDir.
func BackupPathFor(backupDir, file string) string {
	return filepath.Join(backupDir, file) + ".mka"
}

func isTmpFileName(name string) bool {
	return strings.HasPrefix(name, tmpFilePrefix)
}

func isMediaFileName(name string) bool {
	if isTmpFileName(name) {
		return false
	}
	return mediaExtensions[strings.ToLower(filepath.Ext(name))]
}

func walkDirs(ctx context.Context, dirs []string, visit func(path string)) {
	logger := logging.FromContext(ctx)

	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					logger.Warn("media directory not found, skipping", "dir", path)
					return nil
				}
				logger.Warn("failed to walk media directory", "path", path, "error", err)
				return nil
			}
			if !d.IsDir() {
				visit(path)
			}
			return nil
		})
		if err != nil {
			logger.Warn("failed to scan media directory", "dir", dir, "error", err)
		}
	}
}

// ScanDirs walks mediaDirs and returns every matching media file, in a
// deterministic order. Missing/unreadable directories are logged and skipped,
// not treated as fatal.
func ScanDirs(ctx context.Context, mediaDirs []string) []string {
	var files []string
	walkDirs(ctx, mediaDirs, func(path string) {
		if isMediaFileName(filepath.Base(path)) {
			files = append(files, path)
		}
	})
	sort.Strings(files)
	return files
}

// CleanupTmpFiles removes any leftover .loudnorm.tmp.* files under mediaDirs,
// left behind by a crash/OOM/kill during a previous run.
func CleanupTmpFiles(ctx context.Context, mediaDirs []string) int {
	logger := logging.FromContext(ctx)

	removed := 0
	walkDirs(ctx, mediaDirs, func(path string) {
		if !isTmpFileName(filepath.Base(path)) {
			return
		}
		if err := os.Remove(path); err != nil {
			logger.Warn("failed to remove orphaned tmp file", "path", path, "error", err)
			return
		}
		removed++
	})
	return removed
}
