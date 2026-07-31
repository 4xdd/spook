package store

import (
	"context"
	"database/sql"
	"strings"
)

type albumPrimaryArtist struct {
	ID   string
	Name string
}

type primaryCandidate struct {
	id    string
	name  string
	count int
}

// applyAlbumPrimaryArtists picks the most common credited artist per album and
// writes that choice back onto every track in the album.
func (s *Store) applyAlbumPrimaryArtists(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT tr.album_id, ta.artist_id, ta.artist_name, COUNT(*) AS cnt
		FROM track_artists ta
		INNER JOIN tracks tr ON tr.id = ta.track_id
		GROUP BY tr.album_id, ta.artist_id, ta.artist_name
		ORDER BY tr.album_id, cnt DESC`)
	if err != nil {
		return err
	}
	defer rows.Close()

	byAlbum := make(map[string][]primaryCandidate)
	for rows.Next() {
		var albumID, artistID, artistName string
		var count int
		if err := rows.Scan(&albumID, &artistID, &artistName, &count); err != nil {
			return err
		}
		byAlbum[albumID] = append(byAlbum[albumID], primaryCandidate{id: artistID, name: artistName, count: count})
	}
	if err := rows.Err(); err != nil {
		return err
	}

	stmt, err := tx.PrepareContext(ctx, `
		UPDATE tracks SET artist_id = ?, album_artist = ? WHERE album_id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for albumID, candidates := range byAlbum {
		primary := pickPrimaryArtist(candidates)
		if primary.ID == "" {
			continue
		}
		if _, err := stmt.ExecContext(ctx, primary.ID, primary.Name, albumID); err != nil {
			return err
		}
	}
	return nil
}

func pickPrimaryArtist(candidates []primaryCandidate) albumPrimaryArtist {
	if len(candidates) == 0 {
		return albumPrimaryArtist{}
	}

	best := candidates[0]
	bestScore := artistPrimaryScore(best.name, best.count)
	for _, item := range candidates[1:] {
		if score := artistPrimaryScore(item.name, item.count); score > bestScore {
			best = item
			bestScore = score
		}
	}
	return albumPrimaryArtist{ID: best.id, Name: best.name}
}

func artistPrimaryScore(name string, count int) int {
	score := count * 100
	if isLikelyLabel(name) {
		score -= 75
	}
	return score
}

func isLikelyLabel(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	labelWords := []string{
		" records", " record", " label", " entertainment", " productions",
		" publishing", " music group", " collective",
	}
	for _, word := range labelWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return strings.HasSuffix(lower, " records") || strings.HasSuffix(lower, " label")
}
