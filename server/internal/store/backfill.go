package store

import (
	"context"
	"database/sql"

	"github.com/spook/server/internal/credits"
)

// backfillTrackArtists rebuilds per-track credit rows from stored tags and
// titles. It always refreshes every track so newly recognised patterns
// (title "feat." credits, etc.) land without requiring a full re-index.
func (s *Store) backfillTrackArtists(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT t.id, t.title, t.artist, t.album_artist
		FROM tracks t`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type trackCredits struct {
		id          string
		title       string
		artist      string
		albumArtist string
	}
	var pending []trackCredits
	for rows.Next() {
		var item trackCredits
		if err := rows.Scan(&item.id, &item.title, &item.artist, &item.albumArtist); err != nil {
			return err
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	deleteStmt, err := tx.PrepareContext(ctx, `DELETE FROM track_artists WHERE track_id = ?`)
	if err != nil {
		return err
	}
	defer deleteStmt.Close()

	insertStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO track_artists(track_id, artist_id, artist_name, position)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(track_id, artist_id) DO UPDATE SET
			artist_name = excluded.artist_name,
			position = excluded.position`)
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	updateArtist, err := tx.PrepareContext(ctx, `UPDATE tracks SET artist = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer updateArtist.Close()

	for _, item := range pending {
		names := credits.All(item.artist, item.albumArtist, item.title)
		if len(names) == 0 {
			names = []string{firstNonEmpty(item.albumArtist, item.artist, "Unknown Artist")}
		}
		if _, err := deleteStmt.ExecContext(ctx, item.id); err != nil {
			return err
		}
		for i, name := range names {
			if _, err := insertStmt.ExecContext(ctx, item.id, credits.ArtistID(name), name, i); err != nil {
				return err
			}
		}
		// Rewrite the display artist so "underscores feat. X" becomes
		// "underscores · X" once credits have been split.
		if formatted := credits.Format(names); formatted != "" && formatted != item.artist {
			if _, err := updateArtist.ExecContext(ctx, formatted, item.id); err != nil {
				return err
			}
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
