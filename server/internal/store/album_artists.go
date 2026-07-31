package store

import (
	"context"
	"database/sql"

	"github.com/spook/server/internal/credits"
)

type albumArtistRow struct {
	id    string
	name  string
	count int
}

// applyAlbumArtists credits album headliners so collab albums (e.g. Watch the Throne)
// appear under every lead artist, not just whoever dominates the tags.
func (s *Store) applyAlbumArtists(ctx context.Context, tx *sql.Tx) error {
	if err := s.clearAlbumArtists(ctx, tx); err != nil {
		return err
	}

	byAlbum, err := s.loadPrimaryArtistCounts(ctx, tx)
	if err != nil {
		return err
	}
	if err := s.mergeAlbumArtistTags(ctx, tx, byAlbum); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO album_artists(album_id, artist_id, artist_name, position)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(album_id, artist_id) DO UPDATE SET
			artist_name = excluded.artist_name,
			position = excluded.position`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for albumID, candidates := range byAlbum {
		headliners := albumHeadliners(candidates)
		for i, artist := range headliners {
			if _, err := stmt.ExecContext(ctx, albumID, artist.id, artist.name, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Store) clearAlbumArtists(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, `DELETE FROM album_artists`)
	return err
}

func (s *Store) loadPrimaryArtistCounts(ctx context.Context, tx *sql.Tx) (map[string][]albumArtistRow, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT tr.album_id, ta.artist_id, ta.artist_name, COUNT(*) AS cnt
		FROM track_artists ta
		INNER JOIN tracks tr ON tr.id = ta.track_id
		WHERE ta.position = 0
		GROUP BY tr.album_id, ta.artist_id, ta.artist_name
		ORDER BY tr.album_id, cnt DESC, ta.artist_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byAlbum := make(map[string][]albumArtistRow)
	for rows.Next() {
		var albumID, artistID, artistName string
		var count int
		if err := rows.Scan(&albumID, &artistID, &artistName, &count); err != nil {
			return nil, err
		}
		byAlbum[albumID] = append(byAlbum[albumID], albumArtistRow{
			id: artistID, name: artistName, count: count,
		})
	}
	return byAlbum, rows.Err()
}

func (s *Store) mergeAlbumArtistTags(ctx context.Context, tx *sql.Tx, byAlbum map[string][]albumArtistRow) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT DISTINCT album_id, album_artist FROM tracks WHERE album_artist != ''`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var albumID, albumArtist string
		if err := rows.Scan(&albumID, &albumArtist); err != nil {
			return err
		}
		names := credits.Parse(albumArtist)
		if len(names) < 2 {
			continue
		}
		existing := make(map[string]bool, len(byAlbum[albumID]))
		for _, row := range byAlbum[albumID] {
			existing[row.id] = true
		}
		for _, name := range names {
			id := credits.ArtistID(name)
			if existing[id] {
				continue
			}
			existing[id] = true
			byAlbum[albumID] = append(byAlbum[albumID], albumArtistRow{
				id: id, name: name, count: 1,
			})
		}
	}
	return rows.Err()
}

// albumHeadliners returns co-leads for a collab album.
//
// Being primary on a single track is not enough: DJ sets and mixtapes credit
// every remixer as position-0 on their own cut, and promoting all of them used
// to dump Caribou / Lil B / The 1975 into the Artists grid as if they owned the
// whole set. We keep artists who clearly share billing.
func albumHeadliners(candidates []albumArtistRow) []albumArtistRow {
	if len(candidates) < 2 {
		return nil
	}

	total := 0
	for _, item := range candidates {
		total += item.count
	}
	if total == 0 {
		return nil
	}

	out := make([]albumArtistRow, 0, len(candidates))
	for _, item := range candidates {
		if isAlbumHeadliner(item.count, total, len(candidates)) {
			out = append(out, item)
		}
	}
	if len(out) < 2 {
		return nil
	}
	return out
}

func isAlbumHeadliner(count, totalTracks, candidateCount int) bool {
	if count <= 0 || totalTracks <= 0 {
		return false
	}
	// Real stake in the tracklist. Crowded credits (DJ sets) need a bigger
	// share so a few remix cuts don't mint co-leads of the whole set.
	shareNeed := 15
	if candidateCount > 2 {
		shareNeed = 25
	}
	if count >= 2 && count*100 >= totalTracks*shareNeed {
		return true
	}
	if candidateCount != 2 {
		return false
	}
	// Short collab releases: any second primary shares billing.
	if totalTracks <= 6 {
		return true
	}
	// Long two-artist albums: keep a lopsided junior (Watch the Throne:
	// 1 of 16 ≈ 6%) but drop tiny blips (Twista on College Dropout: 1 of 21)
	// and mid-length sets with a single remix cut (8+1).
	return totalTracks >= 12 && count*100 >= totalTracks*6
}

func (s *Store) applyAlbumDisplayArtists(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT album_id, artist_name
		FROM album_artists
		ORDER BY album_id, position`)
	if err != nil {
		return err
	}
	defer rows.Close()

	namesByAlbum := make(map[string][]string)
	for rows.Next() {
		var albumID, name string
		if err := rows.Scan(&albumID, &name); err != nil {
			return err
		}
		namesByAlbum[albumID] = append(namesByAlbum[albumID], name)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `UPDATE albums SET artist_name = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for albumID, names := range namesByAlbum {
		if len(names) < 2 {
			continue
		}
		if _, err := stmt.ExecContext(ctx, credits.Format(names), albumID); err != nil {
			return err
		}
	}
	return nil
}

const artistAlbumsSubquery = `
	SELECT tr.album_id AS album_id
	FROM track_artists ta
	INNER JOIN tracks tr ON tr.id = ta.track_id
	WHERE ta.artist_id = ?
	UNION
	SELECT aa.album_id FROM album_artists aa WHERE aa.artist_id = ?`

const artistTracksSubquery = `
	SELECT ta.track_id AS track_id
	FROM track_artists ta
	WHERE ta.artist_id = ?
	UNION
	SELECT tr.id FROM tracks tr
	INNER JOIN album_artists aa ON aa.album_id = tr.album_id
	WHERE aa.artist_id = ?`
