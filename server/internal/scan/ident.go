package scan

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/spook/server/internal/credits"
)

// id keeps public identifiers short and stable without leaking filesystem paths.
func id(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:8])
}

func trackID(path string) string  { return id("track", path) }
func artistID(name string) string { return credits.ArtistID(name) }

func albumID(albumArtist, album, container string) string {
	container = strings.TrimSpace(container)
	if container == "" {
		return id("album", credits.Fold(albumArtist), credits.Fold(album))
	}
	return id("album", credits.Fold(albumArtist), credits.Fold(album), credits.Fold(container))
}

func fold(v string) string { return credits.Fold(v) }

// sortKey drops a leading article so "The Beatles" sorts under B, matching the
// expression the store uses for derived tables.
func sortKey(v string) string {
	folded := fold(v)
	if strings.HasPrefix(folded, "the ") {
		return folded[4:]
	}
	return folded
}
