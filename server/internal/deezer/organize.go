package deezer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stagingQuietPeriod = 30 * time.Second

// OrganizeAlbumDownloads moves album downloads from deezer-downloader's
// "Artist - Album" staging folders into MusicDir/AlbumTitle/.
// The returned bool is true when at least one album folder was reorganized.
func OrganizeAlbumDownloads(musicDir string) (bool, error) {
	staging := filepath.Join(musicDir, "albums")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return false, err
	}

	entries, err := os.ReadDir(staging)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	var firstErr error
	moved := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		organized, err := organizeAlbumFolder(musicDir, staging, entry.Name())
		if err != nil {
			log.Printf("deezer organize %q: %v", entry.Name(), err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if organized {
			moved = true
		}
	}
	return moved, firstErr
}

func organizeAlbumFolder(musicDir, stagingDir, folderName string) (bool, error) {
	albumTitle, ok := albumTitleFromFolder(folderName)
	if !ok {
		return false, fmt.Errorf("unexpected folder name %q", folderName)
	}

	source := filepath.Join(stagingDir, folderName)
	if stagingFolderActive(source) {
		return false, nil
	}

	target := filepath.Join(musicDir, sanitizeFolderName(albumTitle))

	if err := os.MkdirAll(target, 0o755); err != nil {
		return false, err
	}

	files, err := os.ReadDir(source)
	if err != nil {
		return false, err
	}
	if len(files) == 0 {
		return false, nil
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}
		from := filepath.Join(source, file.Name())
		to := filepath.Join(target, file.Name())
		if err := moveFile(from, to); err != nil {
			return false, err
		}
	}

	if err := os.Remove(source); err != nil {
		return false, err
	}
	return true, nil
}

// stagingFolderActive reports whether a staging folder was touched recently and
// may still be receiving tracks from an in-progress album download.
func stagingFolderActive(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return true
	}
	if len(entries) == 0 {
		return false
	}

	info, err := os.Stat(dir)
	if err != nil {
		return true
	}
	latest := info.ModTime()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}
		if fileInfo.ModTime().After(latest) {
			latest = fileInfo.ModTime()
		}
	}
	return time.Since(latest) < stagingQuietPeriod
}

func albumTitleFromFolder(folderName string) (string, bool) {
	parts := strings.SplitN(folderName, " - ", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return strings.TrimSpace(parts[1]), true
}

func sanitizeFolderName(name string) string {
	replacer := strings.NewReplacer(
		"/", "",
		":", "",
		"\"", "",
		"?", "",
		"\t", " ",
	)
	return replacer.Replace(strings.TrimSpace(name))
}

func moveFile(from, to string) error {
	if _, err := os.Stat(to); err == nil {
		// Already organized — drop the duplicate staging copy.
		return os.Remove(from)
	}
	if err := os.Rename(from, to); err == nil {
		return nil
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		return err
	}
	return os.Remove(from)
}
