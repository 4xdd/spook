package store

type ArtistCredit struct {
	ID       string
	Name     string
	Position int
}

type Track struct {
	ID           string
	Path         string
	Filename     string
	Title        string
	SortTitle    string
	Artist       string
	AlbumArtist  string
	ArtistID     string
	Credits      []ArtistCredit
	AlbumID      string
	AlbumName    string
	Genre        string
	Year         int
	TrackNo      int
	DiscNo       int
	DurationMS   int64
	BitrateKbps  int
	SampleRateHz int
	Channels     int
	Format       string
	SizeBytes    int64
	ModTime      int64
	ArtworkID    string
	Color        string
	AddedAt      int64
}

type Album struct {
	ID         string
	Name       string
	ArtistID   string
	ArtistName string
	Genre      string
	Year       int
	ArtworkID  string
	Color      string
	IsDark       bool
	ReleaseType  string
	TrackCount   int
	DiscCount  int
	DurationMS int64
	AddedAt    int64
}

type Artist struct {
	ID         string
	Name       string
	ArtworkID  string
	Color      string
	IsDark     bool
	AlbumCount int
	TrackCount int
	DurationMS int64
}

type Artwork struct {
	ID     string
	Mime   string
	Width  int
	Height int
	Color  string
	IsDark bool
	Source string
}

// FileState is the minimum needed to decide whether a file must be re-read.
type FileState struct {
	ID      string
	Size    int64
	ModTime int64
	AddedAt int64
}

type Stats struct {
	Tracks     int
	Albums     int
	Artists    int
	DurationMS int64
}

type SearchResults struct {
	Artists []Artist
	Albums  []Album
	Tracks  []Track
}
