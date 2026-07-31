package api

import (
	"github.com/spook/server/internal/lyrics"
	"github.com/spook/server/internal/scan"
	"github.com/spook/server/internal/store"
)

type Track struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Artist       string `json:"artist"`
	ArtistID     string `json:"artistId"`
	Album        string `json:"album"`
	AlbumID      string `json:"albumId"`
	AlbumArtist  string `json:"albumArtist"`
	Genre        string `json:"genre,omitempty"`
	Year         int    `json:"year,omitempty"`
	TrackNo      int    `json:"trackNo,omitempty"`
	DiscNo       int    `json:"discNo,omitempty"`
	DurationMs   int64  `json:"durationMs"`
	Format       string `json:"format"`
	BitrateKbps  int    `json:"bitrateKbps,omitempty"`
	SampleRateHz int    `json:"sampleRateHz,omitempty"`
	SizeBytes    int64  `json:"sizeBytes"`
	ArtworkID    string `json:"artworkId,omitempty"`
	Color        string `json:"color,omitempty"`
	StreamURL    string `json:"streamUrl"`
	AddedAt      int64  `json:"addedAt"`
}

type Album struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Artist     string `json:"artist"`
	ArtistID   string `json:"artistId"`
	Genre      string `json:"genre,omitempty"`
	Year       int    `json:"year,omitempty"`
	ArtworkID  string `json:"artworkId,omitempty"`
	Color      string `json:"color,omitempty"`
	IsDark       bool   `json:"isDark"`
	ReleaseType  string `json:"releaseType"`
	TrackCount   int    `json:"trackCount"`
	DiscCount  int    `json:"discCount"`
	DurationMs int64  `json:"durationMs"`
	AddedAt    int64  `json:"addedAt"`
}

type Artist struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ArtworkID  string `json:"artworkId,omitempty"`
	Color      string `json:"color,omitempty"`
	IsDark     bool   `json:"isDark"`
	AlbumCount int    `json:"albumCount"`
	TrackCount int    `json:"trackCount"`
	DurationMs int64  `json:"durationMs"`
}

type AlbumDetail struct {
	Album  Album   `json:"album"`
	Tracks []Track `json:"tracks"`
}

type ArtistDetail struct {
	Artist Artist  `json:"artist"`
	Albums []Album `json:"albums"`
	Tracks []Track `json:"tracks"`
}

type SearchResults struct {
	Query   string   `json:"query"`
	Artists []Artist `json:"artists"`
	Albums  []Album  `json:"albums"`
	Tracks  []Track  `json:"tracks"`
}

// LyricLine carries a timestamp of -1 when the lyrics are not timed.
type LyricLine struct {
	TimeMs int64  `json:"timeMs"`
	Text   string `json:"text"`
}

type Lyrics struct {
	TrackID string      `json:"trackId"`
	Source  string      `json:"source,omitempty"`
	Synced  bool        `json:"synced"`
	Lines   []LyricLine `json:"lines"`
}

type ScanStatus struct {
	State      string `json:"state"`
	Total      int    `json:"total"`
	Processed  int    `json:"processed"`
	Indexed    int    `json:"indexed"`
	Removed    int    `json:"removed"`
	StartedAt  int64  `json:"startedAt,omitempty"`
	FinishedAt int64  `json:"finishedAt,omitempty"`
	Error      string `json:"error,omitempty"`
}

type Stats struct {
	Root       string     `json:"root"`
	Tracks     int        `json:"tracks"`
	Albums     int        `json:"albums"`
	Artists    int        `json:"artists"`
	DurationMs int64      `json:"durationMs"`
	LastScan   int64      `json:"lastScan,omitempty"`
	Scan       ScanStatus `json:"scan"`
}

func toTrack(t store.Track) Track {
	return Track{
		ID:           t.ID,
		Title:        t.Title,
		Artist:       t.Artist,
		ArtistID:     t.ArtistID,
		Album:        t.AlbumName,
		AlbumID:      t.AlbumID,
		AlbumArtist:  t.AlbumArtist,
		Genre:        t.Genre,
		Year:         t.Year,
		TrackNo:      t.TrackNo,
		DiscNo:       t.DiscNo,
		DurationMs:   t.DurationMS,
		Format:       t.Format,
		BitrateKbps:  t.BitrateKbps,
		SampleRateHz: t.SampleRateHz,
		SizeBytes:    t.SizeBytes,
		ArtworkID:    t.ArtworkID,
		Color:        t.Color,
		StreamURL:    "/api/v1/stream/" + t.ID,
		AddedAt:      t.AddedAt,
	}
}

func toTracks(tracks []store.Track) []Track {
	out := make([]Track, 0, len(tracks))
	for _, track := range tracks {
		out = append(out, toTrack(track))
	}
	return out
}

func toAlbum(a store.Album) Album {
	return Album{
		ID:         a.ID,
		Name:       a.Name,
		Artist:     a.ArtistName,
		ArtistID:   a.ArtistID,
		Genre:      a.Genre,
		Year:       a.Year,
		ArtworkID:  a.ArtworkID,
		Color:      a.Color,
		IsDark:      a.IsDark,
		ReleaseType: a.ReleaseType,
		TrackCount:  a.TrackCount,
		DiscCount:  a.DiscCount,
		DurationMs: a.DurationMS,
		AddedAt:    a.AddedAt,
	}
}

func toAlbums(albums []store.Album) []Album {
	out := make([]Album, 0, len(albums))
	for _, album := range albums {
		out = append(out, toAlbum(album))
	}
	return out
}

func toArtist(a store.Artist) Artist {
	return Artist{
		ID:         a.ID,
		Name:       a.Name,
		ArtworkID:  a.ArtworkID,
		Color:      a.Color,
		IsDark:     a.IsDark,
		AlbumCount: a.AlbumCount,
		TrackCount: a.TrackCount,
		DurationMs: a.DurationMS,
	}
}

func toArtists(artists []store.Artist) []Artist {
	out := make([]Artist, 0, len(artists))
	for _, artist := range artists {
		out = append(out, toArtist(artist))
	}
	return out
}

func toLyrics(trackID string, found lyrics.Lyrics) Lyrics {
	lines := make([]LyricLine, 0, len(found.Lines))
	for _, line := range found.Lines {
		lines = append(lines, LyricLine{TimeMs: line.TimeMs, Text: line.Text})
	}
	return Lyrics{TrackID: trackID, Source: found.Source, Synced: found.Synced, Lines: lines}
}

func toScanStatus(p scan.Progress) ScanStatus {
	status := ScanStatus{
		State:     p.State,
		Total:     p.Total,
		Processed: p.Processed,
		Indexed:   p.Indexed,
		Removed:   p.Removed,
		Error:     p.Error,
	}
	if !p.StartedAt.IsZero() {
		status.StartedAt = p.StartedAt.Unix()
	}
	if !p.FinishedAt.IsZero() {
		status.FinishedAt = p.FinishedAt.Unix()
	}
	return status
}
