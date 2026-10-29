package detector

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
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

// GetRealUserHome retrieves the non-root invoking user's home directory even when executed via sudo.
func GetRealUserHome() string {
	// 1. Check SUDO_USER environment variable
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" && sudoUser != "root" {
		if u, err := user.Lookup(sudoUser); err == nil && u.HomeDir != "" {
			return u.HomeDir
		}
		candidate := filepath.Join("/home", sudoUser)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	// 2. Check SUDO_UID environment variable
	if sudoUID := os.Getenv("SUDO_UID"); sudoUID != "" && sudoUID != "0" {
		if u, err := user.LookupId(sudoUID); err == nil && u.HomeDir != "" {
			return u.HomeDir
		}
	}

	// 3. Fallback to normal UserHomeDir if not root
	if h, err := os.UserHomeDir(); err == nil && h != "/root" && h != "" {
		return h
	}

	// 4. Scan /home for valid user directories
	entries, err := os.ReadDir("/home")
	if err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				return filepath.Join("/home", e.Name())
			}
		}
	}

	return "/home/panda"
}

// FindISOFallbacks scans common user directories for .iso files
func FindISOFallbacks() ([]ISOFile, error) {
	var searchDirs []string

	// Add primary real user directories
	realHome := GetRealUserHome()
	if realHome != "" {
		searchDirs = append(searchDirs,
			filepath.Join(realHome, "Downloads"),
			filepath.Join(realHome, "Desktop"),
			filepath.Join(realHome, "Documents"),
		)
	}

	// Also scan all user directories under /home (in case of multiple accounts)
	if entries, err := os.ReadDir("/home"); err == nil {
		for _, e := range entries {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				uHome := filepath.Join("/home", e.Name())
				searchDirs = append(searchDirs,
					filepath.Join(uHome, "Downloads"),
					filepath.Join(uHome, "Desktop"),
				)
			}
		}
	}

	// Add common data mount points
	dataMounts := []string{"/mnt/data", "/media", "/run/media"}
	for _, m := range dataMounts {
		if _, err := os.Stat(m); err == nil {
			searchDirs = append(searchDirs, m)
		}
	}

	// Add current working directory
	if cwd, err := os.Getwd(); err == nil {
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
				if strings.HasPrefix(base, ".") || base == "node_modules" || base == "dist" || base == "build" || base == "snap" {
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
