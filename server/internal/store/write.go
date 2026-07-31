package store

import (
	"context"
	"fmt"
)

// sortExpr strips a leading article so "The Beatles" files under B.
const sortExpr = `lower(CASE WHEN lower(substr(%[1]s, 1, 4)) = 'the ' THEN substr(%[1]s, 5) ELSE %[1]s END)`

// FileStates returns the indexed state of every known file, keyed by path.
func (s *Store) FileStates(ctx context.Context) (map[string]FileState, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT path, id, size_bytes, mod_time, added_at FROM tracks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make(map[string]FileState)
	for rows.Next() {
		var path string
		var st FileState
		if err := rows.Scan(&path, &st.ID, &st.Size, &st.ModTime, &st.AddedAt); err != nil {
			return nil, err
		}
		states[path] = st
	}
	return states, rows.Err()
}

func (s *Store) PutArtwork(ctx context.Context, art Artwork) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO artwork(id, mime, width, height, color, is_dark, source)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			mime = excluded.mime, width = excluded.width, height = excluded.height,
			color = excluded.color, is_dark = excluded.is_dark`,
		art.ID, art.Mime, art.Width, art.Height, art.Color, boolToInt(art.IsDark), art.Source)
	return err
}

// UpsertTracks writes a batch of scanned tracks in a single transaction.
func (s *Store) UpsertTracks(ctx context.Context, tracks []Track) error {
	if len(tracks) == 0 {
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tracks(
			id, path, filename, title, sort_title, artist, album_artist, artist_id,
			album_id, album_name, genre, year, track_no, disc_no, duration_ms,
			bitrate_kbps, sample_rate_hz, channels, format, size_bytes, mod_time,
			artwork_id, added_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			path = excluded.path, filename = excluded.filename, title = excluded.title,
			sort_title = excluded.sort_title, artist = excluded.artist,
			album_artist = excluded.album_artist, artist_id = excluded.artist_id,
			album_id = excluded.album_id, album_name = excluded.album_name,
			genre = excluded.genre, year = excluded.year, track_no = excluded.track_no,
			disc_no = excluded.disc_no, duration_ms = excluded.duration_ms,
			bitrate_kbps = excluded.bitrate_kbps, sample_rate_hz = excluded.sample_rate_hz,
			channels = excluded.channels, format = excluded.format,
			size_bytes = excluded.size_bytes, mod_time = excluded.mod_time,
			artwork_id = excluded.artwork_id`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	creditDelete, err := tx.PrepareContext(ctx, `DELETE FROM track_artists WHERE track_id = ?`)
	if err != nil {
		return err
	}
	defer creditDelete.Close()

	creditInsert, err := tx.PrepareContext(ctx, `
		INSERT INTO track_artists(track_id, artist_id, artist_name, position)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(track_id, artist_id) DO UPDATE SET
			artist_name = excluded.artist_name,
			position = excluded.position`)
	if err != nil {
		return err
	}
	defer creditInsert.Close()

	for _, t := range tracks {
		var artworkID any
		if t.ArtworkID != "" {
			artworkID = t.ArtworkID
		}
		_, err := stmt.ExecContext(ctx,
			t.ID, t.Path, t.Filename, t.Title, t.SortTitle, t.Artist, t.AlbumArtist, t.ArtistID,
			t.AlbumID, t.AlbumName, t.Genre, t.Year, t.TrackNo, t.DiscNo, t.DurationMS,
			t.BitrateKbps, t.SampleRateHz, t.Channels, t.Format, t.SizeBytes, t.ModTime,
			artworkID, t.AddedAt)
		if err != nil {
			return fmt.Errorf("upsert %q: %w", t.Path, err)
		}

		if _, err := creditDelete.ExecContext(ctx, t.ID); err != nil {
			return fmt.Errorf("clear credits for %q: %w", t.Path, err)
		}
		for _, credit := range t.Credits {
			if _, err := creditInsert.ExecContext(ctx, t.ID, credit.ID, credit.Name, credit.Position); err != nil {
				return fmt.Errorf("credit %q: %w", t.Path, err)
			}
		}
	}

	return tx.Commit()
}

func (s *Store) DeleteTracks(ctx context.Context, paths []string) error {
	if len(paths) == 0 {
		return nil
	}

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `DELETE FROM tracks WHERE path = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, path := range paths {
		if _, err := stmt.ExecContext(ctx, path); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Rebuild derives the artist and album tables from tracks and refreshes the
// search index. Deriving keeps aggregates honest without incremental bookkeeping.
func (s *Store) Rebuild(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	steps := []string{
		`DELETE FROM artists`,
		`DELETE FROM albums`,
		`DELETE FROM album_artists`,
	}

	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step); err != nil {
			return fmt.Errorf("rebuild: %w", err)
		}
	}

	if err := s.backfillTrackArtists(ctx, tx); err != nil {
		return fmt.Errorf("rebuild track artists: %w", err)
	}

	if err := s.applyAlbumArtists(ctx, tx); err != nil {
		return fmt.Errorf("rebuild album artists: %w", err)
	}

	if err := s.applyAlbumPrimaryArtists(ctx, tx); err != nil {
		return fmt.Errorf("rebuild album primary artists: %w", err)
	}

	steps = []string{
		fmt.Sprintf(`
			INSERT INTO artists(id, name, sort_name)
			SELECT ta.artist_id, ta.artist_name, %s
			FROM track_artists ta
			INNER JOIN (
				SELECT artist_id, MIN(position) AS min_pos
				FROM track_artists
				GROUP BY artist_id
			) first ON first.artist_id = ta.artist_id AND first.min_pos = ta.position
			GROUP BY ta.artist_id`, fmt.Sprintf(sortExpr, "ta.artist_name")),
		fmt.Sprintf(`
			INSERT INTO albums(id, name, sort_name, artist_id, artist_name, year, genre, artwork_id, release_type)
			SELECT t.album_id, t.album_name, %s, t.artist_id, t.album_artist,
			       MAX(t.year), MAX(t.genre),
			       (SELECT t2.artwork_id FROM tracks t2
			         WHERE t2.album_id = t.album_id AND t2.artwork_id IS NOT NULL
			         GROUP BY t2.artwork_id ORDER BY COUNT(*) DESC LIMIT 1),
			       CASE
			         WHEN COUNT(*) <= 1 THEN 'single'
			         WHEN COUNT(*) <= 6 THEN 'ep'
			         ELSE 'album'
			       END
			FROM tracks t GROUP BY t.album_id`, fmt.Sprintf(sortExpr, "t.album_name")),
		`DELETE FROM search_fts`,
		`INSERT INTO search_fts(kind, ref_id, title, subtitle) SELECT 'artist', id, name, '' FROM artists`,
		`INSERT INTO search_fts(kind, ref_id, title, subtitle) SELECT 'album', id, name, artist_name FROM albums`,
		`INSERT INTO search_fts(kind, ref_id, title, subtitle) SELECT 'track', id, title, artist || ' ' || album_name FROM tracks`,
		// Orphaned artwork rows would otherwise linger after files are removed.
		`DELETE FROM artwork WHERE id NOT IN (SELECT artwork_id FROM tracks WHERE artwork_id IS NOT NULL)`,
	}

	for _, step := range steps {
		if _, err := tx.ExecContext(ctx, step); err != nil {
			return fmt.Errorf("rebuild: %w", err)
		}
	}

	if err := s.applyAlbumDisplayArtists(ctx, tx); err != nil {
		return fmt.Errorf("rebuild album display artists: %w", err)
	}

	return tx.Commit()
}

func (s *Store) Vacuum(ctx context.Context) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
