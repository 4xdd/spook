package scan

import (
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	yearPartRE    = regexp.MustCompile(`^\d{4}$`)
	variantSuffix = regexp.MustCompile(`^(\d+|[\s(].+)$`)
)

// albumContainerName returns the on-disk folder that represents a track's album,
// skipping common disc subfolders such as CD1 or Disc 2.
func albumContainerName(musicRoot, fileDir string) string {
	rel, err := filepath.Rel(musicRoot, fileDir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return filepath.Base(fileDir)
	}

	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) == 0 {
		return ""
	}
	if len(parts) >= 2 && isDiscFolder(parts[len(parts)-1]) {
		return parts[len(parts)-2]
	}
	return parts[len(parts)-1]
}

func isDiscFolder(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(n, "cd") {
		rest := strings.TrimSpace(n[2:])
		_, err := strconv.Atoi(rest)
		return err == nil
	}
	if strings.HasPrefix(n, "disc") {
		rest := strings.TrimSpace(n[4:])
		_, err := strconv.Atoi(rest)
		return err == nil
	}
	return false
}

// resolveAlbumName prefers a more specific on-disk folder name over a generic tag.
// This keeps "Jealous" and "Jealous (2025)" separate, and splits numbered variants
// like name / name2 / name3 even when tags reuse the same album title.
func resolveAlbumName(tagAlbum, container string) string {
	tagAlbum = strings.TrimSpace(tagAlbum)
	container = strings.TrimSpace(container)

	if container == "" {
		return firstNonEmpty(tagAlbum, "Unknown Album")
	}
	if tagAlbum == "" {
		if _, album, ok := parseFolderArtistAlbum(container); ok {
			return album
		}
		return container
	}
	if folderDisambiguates(tagAlbum, container) {
		if _, album, ok := parseFolderArtistAlbum(container); ok {
			return album
		}
		return container
	}
	return tagAlbum
}

// folderDisambiguates reports whether container should win over tagAlbum.
func folderDisambiguates(tagAlbum, container string) bool {
	tagFold := fold(tagAlbum)
	containerFold := fold(container)
	if tagFold == containerFold {
		return false
	}
	if strings.HasPrefix(containerFold, tagFold) {
		suffix := strings.TrimSpace(containerFold[len(tagFold):])
		if suffix != "" && variantSuffix.MatchString(suffix) {
			return true
		}
	}
	if _, album, ok := parseFolderArtistAlbum(container); ok && fold(album) != tagFold {
		return true
	}
	return false
}

// parseFolderArtistAlbum extracts artist and album from common folder patterns such as
// "Artist - Album" or "Artist - 2010 - Album Title".
func parseFolderArtistAlbum(folderName string) (artist, album string, ok bool) {
	parts := strings.Split(folderName, " - ")
	if len(parts) < 2 {
		return "", "", false
	}
	if len(parts) == 2 {
		artist = strings.TrimSpace(parts[0])
		album = strings.TrimSpace(parts[1])
		return artist, album, artist != "" && album != ""
	}
	if yearPartRE.MatchString(strings.TrimSpace(parts[1])) {
		artist = strings.TrimSpace(parts[0])
		album = strings.TrimSpace(strings.Join(parts[2:], " - "))
		return artist, album, artist != "" && album != ""
	}
	artist = strings.TrimSpace(parts[0])
	album = strings.TrimSpace(strings.Join(parts[1:], " - "))
	return artist, album, artist != "" && album != ""
}

// resolveAlbumArtist prefers tags but can infer artist from folder names.
func resolveAlbumArtist(tagArtist, tagAlbumArtist, container string) string {
	artist := firstNonEmpty(tagAlbumArtist, tagArtist)
	if artist != "" && artist != "Unknown Artist" {
		return artist
	}
	if parsedArtist, _, ok := parseFolderArtistAlbum(container); ok {
		return parsedArtist
	}
	return firstNonEmpty(artist, "Unknown Artist")
}
