package store

// ReleaseType classifies a release from its track count.
func ReleaseType(trackCount int) string {
	switch {
	case trackCount <= 1:
		return "single"
	case trackCount <= 6:
		return "ep"
	default:
		return "album"
	}
}
