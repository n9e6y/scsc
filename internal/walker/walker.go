// File: internal/walker/walker.go
// Description: Scans the repository for source files, ignoring common vendor and hidden directories.

package walker

import (
	"io/fs"
	"path/filepath"
	"strings"
)

// commonIgnores are directories we almost never want to index.
// In a v2, this would parse an actual .gitignore file.
var commonIgnores = map[string]bool{
	".git":         true,
	"node_modules": true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".semcode":     true, // Never index our own database!
}

// Walk traverses a directory and returns all file paths that match the target extensions.
// If validExts is empty, it returns all non-ignored files.
func Walk(root string, validExts []string) ([]string, error) {
	var files []string

	extMap := make(map[string]bool)
	for _, ext := range validExts {
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[ext] = true
	}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err // Can't access this file/dir
		}

		if d.IsDir() {
			// Never skip the root itself: a root like ".." or "./.hidden-repo" would
			// otherwise trip the hidden-directory rule and index nothing.
			if path == root {
				return nil
			}
			if commonIgnores[d.Name()] {
				return filepath.SkipDir
			}
			// Skip hidden directories (like .idea, .vscode)
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// It's a file. Check extensions if we have a filter.
		if len(extMap) > 0 {
			ext := filepath.Ext(d.Name())
			if !extMap[ext] {
				return nil
			}
		}

		// Optionally skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}
