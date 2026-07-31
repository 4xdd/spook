package lyrics

import "context"

// Lookup returns lyrics for a track: local files first, then LRCLIB synced lookup.
func Lookup(ctx context.Context, path string, track Track, online *Online) Lyrics {
	if found := Find(path); !found.Empty() {
		return found
	}
	if online == nil {
		return Lyrics{}
	}
	return online.Fetch(ctx, track)
}
