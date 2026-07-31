package deezer

// SearchType selects what Deezer entity to search for.
type SearchType string

const (
	SearchTrack  SearchType = "track"
	SearchAlbum  SearchType = "album"
	SearchArtist SearchType = "artist"
)

// DownloadType selects a single track or full album download.
type DownloadType string

const (
	DownloadTrack DownloadType = "track"
	DownloadAlbum DownloadType = "album"
)

// Result is a normalized Deezer search hit.
type Result struct {
	ID         string     `json:"id"`
	Type       SearchType `json:"type"`
	Title      string     `json:"title,omitempty"`
	Album      string     `json:"album,omitempty"`
	AlbumID    string     `json:"albumId,omitempty"`
	Artist     string     `json:"artist,omitempty"`
	ArtistID   string     `json:"artistId,omitempty"`
	ImageURL   string     `json:"imageUrl,omitempty"`
	PreviewURL string     `json:"previewUrl,omitempty"`
}

// Job mirrors a task from deezer-downloader's /queue endpoint.
type Job struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	State       string `json:"state"`
	Progress    int    `json:"progress"`
	ProgressMax int    `json:"progressMax"`
	Error       string `json:"error,omitempty"`
}

// Status reports whether the Deezer subworker is configured and reachable.
type Status struct {
	Enabled   bool   `json:"enabled"`
	Running   bool   `json:"running"`
	Configured bool  `json:"configured"`
	BaseURL   string `json:"baseUrl,omitempty"`
	MusicDir  string `json:"musicDir,omitempty"`
	Quality   string `json:"quality,omitempty"`
	Error     string `json:"error,omitempty"`
}
