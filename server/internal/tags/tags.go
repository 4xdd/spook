// Package tags reads embedded metadata and cover art from audio files.
package tags

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
)

var supported = map[string]bool{
	".mp3":  true,
	".flac": true,
	".ogg":  true,
	".oga":  true,
	".m4a":  true,
	".mp4":  true,
	".wav":  true,
	".aac":  true,
	".opus": true,
	".wma":  true,
}

func IsSupported(path string) bool {
	return supported[strings.ToLower(filepath.Ext(path))]
}

type Metadata struct {
	Title       string
	Artist      string
	Album       string
	AlbumArtist string
	Genre       string
	Composer    string
	Year        int
	TrackNo     int
	DiscNo      int
	Picture     []byte
	PictureMIME string
}

// Read parses tags, falling back to the filename when a file carries none.
func Read(path string) Metadata {
	ext := strings.ToLower(filepath.Ext(path))
	meta := Metadata{
		Title: strings.TrimSuffix(filepath.Base(path), ext),
	}

	f, err := os.Open(path)
	if err != nil {
		return meta
	}
	defer f.Close()

	parsed, err := tag.ReadFrom(f)
	if err != nil {
		return meta
	}

	if v := clean(parsed.Title()); v != "" {
		meta.Title = v
	}
	meta.Artist = clean(parsed.Artist())
	meta.Album = clean(parsed.Album())
	meta.AlbumArtist = clean(parsed.AlbumArtist())
	meta.Genre = clean(parsed.Genre())
	meta.Composer = clean(parsed.Composer())

	if y := parsed.Year(); y > 0 {
		meta.Year = y
	}
	if n, _ := parsed.Track(); n > 0 {
		meta.TrackNo = n
	}
	if n, _ := parsed.Disc(); n > 0 {
		meta.DiscNo = n
	}

	if pic := parsed.Picture(); pic != nil && len(pic.Data) > 0 {
		meta.Picture = pic.Data
		meta.PictureMIME = pic.MIMEType
	}

	return meta
}

// lyricKeys are the tag names the common containers use for unsynchronised
// lyrics: ID3v2 frames, MP4 atoms, and Vorbis comments (lowercased by the
// tag reader).
var lyricKeys = []string{"USLT", "ULT", "\xa9lyr", "lyrics", "unsyncedlyrics", "unsynced lyrics"}

// Lyrics returns the lyrics embedded in a file's tags, or "" when it has none.
// It is read on demand rather than during a scan, so the text never has to be
// carried through the index.
func Lyrics(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	parsed, err := tag.ReadFrom(f)
	if err != nil {
		return ""
	}

	// Reading Raw() rather than Lyrics() avoids the unchecked type assertions
	// the tag package makes, which panic on files with a malformed frame.
	raw := parsed.Raw()
	for _, key := range lyricKeys {
		if text := lyricText(raw[key]); text != "" {
			return clean(text)
		}
	}
	// A file with several USLT frames stores the rest under a numbered name.
	for key, value := range raw {
		if !strings.HasPrefix(key, "USLT") && !strings.HasPrefix(key, "ULT") {
			continue
		}
		if text := lyricText(value); text != "" {
			return clean(text)
		}
	}
	return ""
}

func lyricText(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case *tag.Comm:
		if v == nil {
			return ""
		}
		return v.Text
	case tag.Comm:
		return v.Text
	default:
		return ""
	}
}

func clean(v string) string {
	return strings.TrimSpace(strings.ReplaceAll(v, "\x00", ""))
}
