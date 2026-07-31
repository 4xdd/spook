package store

// ArtistListedInBrowse reports whether an artist should appear in the browse grid.
//
// primaryReleases counts albums they own. directReleases / directTracks count
// rows in track_artists only — never album_artists — so a remixer who was
// wrongly promoted to album headliner cannot inflate past the one-off filter.
func ArtistListedInBrowse(primaryReleases, directReleases, directTracks int) bool {
	if primaryReleases > 0 {
		return true
	}
	// Multi-credit collaborators stay visible; a single guest feature does not.
	return directTracks > 1 || directReleases > 1
}

// Browse listing uses direct credits so album-headliner inflation cannot sneak
// one-track remix guests into the Artists grid.
const artistBrowseWhere = `
	WHERE x.primary_release_count > 0
	   OR x.direct_track_count > 1
	   OR x.direct_album_count > 1`
