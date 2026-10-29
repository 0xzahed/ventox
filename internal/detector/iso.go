package detector

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ISOFile struct {
	Path          string
	Name          string
	Size          int64
	FormattedSize string
	ModTime       time.Time
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FindISOFallbacks scans common user directories for .iso files
func FindISOFallbacks() ([]ISOFile, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "/home/panda"
	}

	searchDirs := []string{
		filepath.Join(homeDir, "Downloads"),
		filepath.Join(homeDir, "Desktop"),
		filepath.Join(homeDir, "Documents"),
	}

	cwd, err := os.Getwd()
	if err == nil && cwd != homeDir {
		searchDirs = append(searchDirs, cwd)
	}

	var found []ISOFile
	seen := make(map[string]bool)

	for _, dir := range searchDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}

			// Don't descend into hidden dirs, git, or node_modules
			if d.IsDir() {
				base := d.Name()
				if strings.HasPrefix(base, ".") || base == "node_modules" || base == "dist" || base == "build" {
					return filepath.SkipDir
				}
				// Limit depth to 3 levels below search directory
				rel, _ := filepath.Rel(dir, path)
				if strings.Count(rel, string(os.PathSeparator)) >= 3 {
					return filepath.SkipDir
				}
				return nil
			}

			if strings.EqualFold(filepath.Ext(d.Name()), ".iso") {
				if seen[path] {
					return nil
				}
				seen[path] = true

				info, err := d.Info()
				if err != nil {
					return nil
				}

				found = append(found, ISOFile{
					Path:          path,
					Name:          d.Name(),
					Size:          info.Size(),
					FormattedSize: formatBytes(info.Size()),
					ModTime:       info.ModTime(),
				})
			}
			return nil
		})
	}

	// Sort by newest modified first
	sort.Slice(found, func(i, j int) bool {
		return found[i].ModTime.After(found[j].ModTime)
	})

	return found, nil
}
