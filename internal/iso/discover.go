package iso

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xzahed/ventox/internal/domain"
)

// SupportedExtensions lists recognized disk image extensions.
var SupportedExtensions = map[string]domain.ImageFormat{
	".iso": domain.FormatISO,
	".img": domain.FormatRawIMG,
	".raw": domain.FormatRawIMG,
}

// GetDefaultSearchDirs returns standard directories likely to contain disk images,
// handling sudo privilege escalation cleanly by checking the invoking user's home.
func GetDefaultSearchDirs() []string {
	var dirs []string
	home := getInvokingUserHome()

	if home != "" {
		downloads := filepath.Join(home, "Downloads")
		desktop := filepath.Join(home, "Desktop")
		documents := filepath.Join(home, "Documents")

		for _, d := range []string{downloads, desktop, documents, home} {
			if stat, err := os.Stat(d); err == nil && stat.IsDir() {
				dirs = append(dirs, d)
			}
		}
	}

	// Current working directory
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, cwd)
	}

	return deduplicatePaths(dirs)
}

// DiscoverImages scans the given directories (non-recursively for performance)
// and returns metadata for discovered disk images.
func DiscoverImages(ctx context.Context, searchDirs ...string) ([]domain.ImageMetadata, error) {
	if len(searchDirs) == 0 {
		searchDirs = GetDefaultSearchDirs()
	}

	var results []domain.ImageMetadata
	seen := make(map[string]bool)

	for _, dir := range searchDirs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // Skip unreadable directories
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			ext := strings.ToLower(filepath.Ext(entry.Name()))
			format, ok := SupportedExtensions[ext]
			if !ok {
				continue
			}

			fullPath := filepath.Join(dir, entry.Name())
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			meta, err := InspectImage(fullPath)
			if err != nil {
				// Still include basic file info even if deep inspection fails
				info, statErr := entry.Info()
				if statErr != nil {
					continue
				}
				results = append(results, domain.ImageMetadata{
					Path:           fullPath,
					Name:           entry.Name(),
					SizeBytes:      info.Size(),
					ModTime:        info.ModTime(),
					Format:         format,
					Classification: domain.ClassUnknown,
				})
				continue
			}

			results = append(results, meta)
		}
	}

	return results, nil
}

// getInvokingUserHome resolves the actual user's home directory even when running under sudo.
func getInvokingUserHome() string {
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser != "" && sudoUser != "root" {
		// Common Linux user home
		linuxHome := filepath.Join("/home", sudoUser)
		if stat, err := os.Stat(linuxHome); err == nil && stat.IsDir() {
			return linuxHome
		}
		// macOS user home
		darwinHome := filepath.Join("/Users", sudoUser)
		if stat, err := os.Stat(darwinHome); err == nil && stat.IsDir() {
			return darwinHome
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		return home
	}
	return ""
}

func deduplicatePaths(paths []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, p := range paths {
		cleaned := filepath.Clean(p)
		if !seen[cleaned] {
			seen[cleaned] = true
			out = append(out, cleaned)
		}
	}
	return out
}
