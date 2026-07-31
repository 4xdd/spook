package credits

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var featSplitRE = regexp.MustCompile(`(?i)\s+(?:feat\.?|ft\.?|featuring|with|w/)\s+`)

// titleFeatRE pulls credited guests out of titles like
// "Bangarang (feat. Sirah)" or "Years Of War (Feat. A & B)".
var titleFeatRE = regexp.MustCompile(`(?i)(?:\(|\[)\s*(?:feat\.?|ft\.?|featuring)\s+([^)\]]+)(?:\)|\])`)

// ArtistID returns a stable identifier for an artist name.
func ArtistID(name string) string {
	sum := sha256.Sum256([]byte("artist\x00" + Fold(name)))
	return hex.EncodeToString(sum[:8])
}

func Fold(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// Parse splits a raw credit string into individual artist names.
func Parse(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// feat./ft./with → commas, then also undo our own " · " display join so a
	// later rebuild can re-split credits that were already formatted.
	normalized := featSplitRE.ReplaceAllString(raw, ", ")
	normalized = strings.ReplaceAll(normalized, " · ", ", ")
	var parts []string
	for _, segment := range strings.Split(normalized, ",") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		if strings.Contains(segment, " & ") {
			for _, side := range strings.Split(segment, " & ") {
				if name := strings.TrimSpace(side); name != "" {
					parts = append(parts, name)
				}
			}
			continue
		}
		parts = append(parts, segment)
	}
	return Dedupe(parts)
}

// FromTitle extracts featured artists embedded in a track title.
func FromTitle(title string) []string {
	matches := titleFeatRE.FindAllStringSubmatch(title, -1)
	if len(matches) == 0 {
		return nil
	}
	var out []string
	for _, match := range matches {
		out = append(out, Parse(match[1])...)
	}
	return Dedupe(out)
}

// Merge parses and deduplicates artists from multiple tag fields.
func Merge(fields ...string) []string {
	var all []string
	for _, field := range fields {
		all = append(all, Parse(field)...)
	}
	return Dedupe(all)
}

// All combines artist/album-artist tags with featured artists from the title.
// Title guests are appended after the tagged credits so they never become
// the primary/album artist.
func All(artist, albumArtist, title string) []string {
	return Dedupe(append(Merge(artist, albumArtist), FromTitle(title)...))
}

func Dedupe(names []string) []string {
	seen := make(map[string]bool, len(names))
	out := make([]string, 0, len(names))
	for _, name := range names {
		key := Fold(name)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, strings.TrimSpace(name))
	}
	return out
}

func Primary(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

func Format(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	default:
		return strings.Join(names, " · ")
	}
}
