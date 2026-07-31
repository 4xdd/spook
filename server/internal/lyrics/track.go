package lyrics

// Track describes a library item for an online lyrics lookup.
type Track struct {
	Title      string
	Artist     string
	Album      string
	DurationMS int64
}
