package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrNotFound = errors.New("not found")

const albumSelect = `
	SELECT al.id, al.name, al.artist_id, al.artist_name, al.year, COALESCE(al.genre, ''),
	       COALESCE(al.artwork_id, ''), COALESCE(aw.color, ''), COALESCE(aw.is_dark, 0),
	       COALESCE(al.release_type, 'album'),
	       COUNT(t.id), COALESCE(SUM(t.duration_ms), 0),
	       COALESCE(MAX(t.added_at), 0), MAX(COALESCE(MAX(t.disc_no), 1), 1)
	FROM albums al
	LEFT JOIN tracks t ON t.album_id = al.id
	LEFT JOIN artwork aw ON aw.id = al.artwork_id`

const trackSelect = `
	SELECT t.id, t.path, t.filename, t.title, t.artist, t.album_artist, t.artist_id,
	       t.album_id, t.album_name, COALESCE(t.genre, ''), t.year, t.track_no, t.disc_no,
	       t.duration_ms, t.bitrate_kbps, t.sample_rate_hz, t.channels, t.format,
	       t.size_bytes, COALESCE(t.artwork_id, ''), COALESCE(aw.color, ''), t.added_at
	FROM tracks t
	LEFT JOIN artwork aw ON aw.id = t.artwork_id`

// AlbumOrder maps an API sort key to SQL.
func AlbumOrder(sort string) string {
	switch sort {
	case "recent":
		return `MAX(t.added_at) DESC, al.sort_name`
	case "artist":
		return `(SELECT ar.sort_name FROM artists ar WHERE ar.id = al.artist_id), al.year, al.sort_name`
	case "year":
		return `al.year DESC, al.sort_name`
	default:
		return `al.sort_name`
	}
}

func TrackOrder(sort string) string {
	switch sort {
	case "recent":
		return `t.added_at DESC, t.sort_title`
	case "artist":
		return `t.artist, t.album_name, t.disc_no, t.track_no`
	case "album":
		return `t.album_name, t.disc_no, t.track_no`
	default:
		return `t.sort_title`
	}
}

func (s *Store) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	err := s.db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM tracks),
		       (SELECT COUNT(*) FROM albums),
		       (SELECT COUNT(*) FROM artists ar
		         WHERE (SELECT COUNT(*) FROM albums al WHERE al.artist_id = ar.id) > 0
		            OR (SELECT COUNT(*) FROM track_artists ta WHERE ta.artist_id = ar.id) > 1
		            OR (SELECT COUNT(DISTINCT tr.album_id) FROM track_artists ta
		                  INNER JOIN tracks tr ON tr.id = ta.track_id
		                 WHERE ta.artist_id = ar.id) > 1),
		       (SELECT COALESCE(SUM(duration_ms), 0) FROM tracks)`).
		Scan(&st.Tracks, &st.Albums, &st.Artists, &st.DurationMS)
	return st, err
}

func (s *Store) Albums(ctx context.Context, sort string, limit, offset int) ([]Album, error) {
	query := fmt.Sprintf(`%s GROUP BY al.id ORDER BY %s LIMIT ? OFFSET ?`, albumSelect, AlbumOrder(sort))
	rows, err := s.db.QueryContext(ctx, query, clampLimit(limit), offset)
	if err != nil {
		return nil, err
	}
	return scanAlbums(rows)
}

func (s *Store) Album(ctx context.Context, id string) (Album, error) {
	query := fmt.Sprintf(`%s WHERE al.id = ? GROUP BY al.id`, albumSelect)
	rows, err := s.db.QueryContext(ctx, query, id)
	if err != nil {
		return Album{}, err
	}
	albums, err := scanAlbums(rows)
	if err != nil {
		return Album{}, err
	}
	if len(albums) == 0 {
		return Album{}, ErrNotFound
	}
	return albums[0], nil
}

func (s *Store) AlbumsByArtist(ctx context.Context, artistID string) ([]Album, error) {
	query := fmt.Sprintf(`%s
		WHERE al.id IN (`+artistAlbumsSubquery+`)
		GROUP BY al.id ORDER BY al.year DESC, al.sort_name`, albumSelect)
	rows, err := s.db.QueryContext(ctx, query, artistID, artistID)
	if err != nil {
		return nil, err
	}
	return scanAlbums(rows)
}

func (s *Store) AlbumsByIDs(ctx context.Context, ids []string) ([]Album, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`%s WHERE al.id IN (%s) GROUP BY al.id`, albumSelect, placeholders(len(ids)))
	rows, err := s.db.QueryContext(ctx, query, toAny(ids)...)
	if err != nil {
		return nil, err
	}
	albums, err := scanAlbums(rows)
	if err != nil {
		return nil, err
	}
	order := indexOf(ids)
	sortBy(albums, func(a Album) int { return order[a.ID] })
	return albums, nil
}

func (s *Store) Artists(ctx context.Context) ([]Artist, error) {
	rows, err := s.db.QueryContext(ctx, artistQuery(artistBrowseWhere, `x.sort_name`))
	if err != nil {
		return nil, err
	}
	return scanArtists(rows)
}

func (s *Store) Artist(ctx context.Context, id string) (Artist, error) {
	rows, err := s.db.QueryContext(ctx, artistQuery(`WHERE x.id = ?`, `x.sort_name`), id)
	if err != nil {
		return Artist{}, err
	}
	artists, err := scanArtists(rows)
	if err != nil {
		return Artist{}, err
	}
	if len(artists) == 0 {
		return Artist{}, ErrNotFound
	}
	return artists[0], nil
}

func (s *Store) ArtistsByIDs(ctx context.Context, ids []string) ([]Artist, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := artistQuery(fmt.Sprintf(`WHERE x.id IN (%s)`, placeholders(len(ids))), `x.sort_name`)
	rows, err := s.db.QueryContext(ctx, query, toAny(ids)...)
	if err != nil {
		return nil, err
	}
	artists, err := scanArtists(rows)
	if err != nil {
		return nil, err
	}
	order := indexOf(ids)
	sortBy(artists, func(a Artist) int { return order[a.ID] })
	return artists, nil
}

// artistQuery builds the aggregate query in two stages so the artwork row can be
// joined against the album-derived artwork id.
func artistQuery(where, order string) string {
	return fmt.Sprintf(`
		SELECT x.id, x.name, x.artwork_id, COALESCE(aw.color, ''), COALESCE(aw.is_dark, 0),
		       x.album_count, x.track_count, x.duration_ms
		FROM (
			SELECT ar.id, ar.name, ar.sort_name,
			       COALESCE((SELECT al.artwork_id FROM albums al
			                  WHERE al.artwork_id IS NOT NULL
			                    AND al.id IN (
			                      SELECT tr.album_id FROM track_artists ta
			                      INNER JOIN tracks tr ON tr.id = ta.track_id
			                      WHERE ta.artist_id = ar.id
			                      UNION
			                      SELECT aa.album_id FROM album_artists aa WHERE aa.artist_id = ar.id
			                    )
			                  ORDER BY al.year DESC LIMIT 1), '') AS artwork_id,
			       (SELECT COUNT(DISTINCT album_id) FROM (
			          SELECT tr.album_id AS album_id FROM track_artists ta
			          INNER JOIN tracks tr ON tr.id = ta.track_id
			          WHERE ta.artist_id = ar.id
			          UNION
			          SELECT aa.album_id FROM album_artists aa WHERE aa.artist_id = ar.id
			        )) AS album_count,
			       (SELECT COUNT(*) FROM albums al WHERE al.artist_id = ar.id) AS primary_release_count,
			       (SELECT COUNT(*) FROM track_artists ta WHERE ta.artist_id = ar.id) AS direct_track_count,
			       (SELECT COUNT(DISTINCT tr.album_id) FROM track_artists ta
			          INNER JOIN tracks tr ON tr.id = ta.track_id
			         WHERE ta.artist_id = ar.id) AS direct_album_count,
			       (SELECT COUNT(*) FROM (
			          SELECT ta.track_id AS track_id FROM track_artists ta WHERE ta.artist_id = ar.id
			          UNION
			          SELECT tr.id FROM tracks tr
			          INNER JOIN album_artists aa ON aa.album_id = tr.album_id
			          WHERE aa.artist_id = ar.id
			        )) AS track_count,
			       (SELECT COALESCE(SUM(tr.duration_ms), 0) FROM tracks tr
			         WHERE tr.id IN (
			           SELECT ta.track_id FROM track_artists ta WHERE ta.artist_id = ar.id
			           UNION
			           SELECT tr2.id FROM tracks tr2
			           INNER JOIN album_artists aa ON aa.album_id = tr2.album_id
			           WHERE aa.artist_id = ar.id
			         )) AS duration_ms
			FROM artists ar
		) x
		LEFT JOIN artwork aw ON aw.id = x.artwork_id
		%s
		ORDER BY %s`, where, order)
}

func (s *Store) Tracks(ctx context.Context, sort string, limit, offset int) ([]Track, error) {
	query := fmt.Sprintf(`%s ORDER BY %s LIMIT ? OFFSET ?`, trackSelect, TrackOrder(sort))
	rows, err := s.db.QueryContext(ctx, query, clampLimit(limit), offset)
	if err != nil {
		return nil, err
	}
	return scanTracks(rows)
}

func (s *Store) Track(ctx context.Context, id string) (Track, error) {
	rows, err := s.db.QueryContext(ctx, trackSelect+` WHERE t.id = ?`, id)
	if err != nil {
		return Track{}, err
	}
	tracks, err := scanTracks(rows)
	if err != nil {
		return Track{}, err
	}
	if len(tracks) == 0 {
		return Track{}, ErrNotFound
	}
	return tracks[0], nil
}

func (s *Store) TracksByAlbum(ctx context.Context, albumID string) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		trackSelect+` WHERE t.album_id = ? ORDER BY t.disc_no, t.track_no, t.sort_title`, albumID)
	if err != nil {
		return nil, err
	}
	return scanTracks(rows)
}

func (s *Store) TracksByArtist(ctx context.Context, artistID string) ([]Track, error) {
	rows, err := s.db.QueryContext(ctx,
		trackSelect+` WHERE t.id IN (`+artistTracksSubquery+`)
		 ORDER BY t.album_name, t.disc_no, t.track_no`, artistID, artistID)
	if err != nil {
		return nil, err
	}
	return scanTracks(rows)
}

func (s *Store) TracksByIDs(ctx context.Context, ids []string) ([]Track, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := fmt.Sprintf(`%s WHERE t.id IN (%s)`, trackSelect, placeholders(len(ids)))
	rows, err := s.db.QueryContext(ctx, query, toAny(ids)...)
	if err != nil {
		return nil, err
	}
	tracks, err := scanTracks(rows)
	if err != nil {
		return nil, err
	}
	order := indexOf(ids)
	sortBy(tracks, func(t Track) int { return order[t.ID] })
	return tracks, nil
}

func (s *Store) Artwork(ctx context.Context, id string) (Artwork, error) {
	var art Artwork
	var isDark int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, mime, width, height, color, is_dark, source FROM artwork WHERE id = ?`, id).
		Scan(&art.ID, &art.Mime, &art.Width, &art.Height, &art.Color, &isDark, &art.Source)
	if err == sql.ErrNoRows {
		return Artwork{}, ErrNotFound
	}
	art.IsDark = isDark == 1
	return art, err
}

func (s *Store) AllArtwork(ctx context.Context) ([]Artwork, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, mime, width, height, color, is_dark, source FROM artwork`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Artwork
	for rows.Next() {
		var art Artwork
		var isDark int
		if err := rows.Scan(&art.ID, &art.Mime, &art.Width, &art.Height, &art.Color, &isDark, &art.Source); err != nil {
			return nil, err
		}
		art.IsDark = isDark == 1
		out = append(out, art)
	}
	return out, rows.Err()
}

func (s *Store) Search(ctx context.Context, query string, limit int) (SearchResults, error) {
	match := ftsMatch(query)
	if match == "" {
		return SearchResults{}, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	ids := func(kind string, n int) ([]string, error) {
		rows, err := s.db.QueryContext(ctx,
			`SELECT ref_id FROM search_fts WHERE search_fts MATCH ? AND kind = ? ORDER BY rank LIMIT ?`,
			match, kind, n)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				return nil, err
			}
			out = append(out, id)
		}
		return out, rows.Err()
	}

	var results SearchResults

	artistIDs, err := ids("artist", limit)
	if err != nil {
		return results, err
	}
	if results.Artists, err = s.ArtistsByIDs(ctx, artistIDs); err != nil {
		return results, err
	}

	albumIDs, err := ids("album", limit)
	if err != nil {
		return results, err
	}
	if results.Albums, err = s.AlbumsByIDs(ctx, albumIDs); err != nil {
		return results, err
	}

	trackIDs, err := ids("track", limit)
	if err != nil {
		return results, err
	}
	if results.Tracks, err = s.TracksByIDs(ctx, trackIDs); err != nil {
		return results, err
	}

	return results, nil
}

// ftsMatch turns free text into a prefix query, quoting each term so
// punctuation in names cannot be read as FTS5 syntax.
func ftsMatch(query string) string {
	fields := strings.Fields(query)
	terms := make([]string, 0, len(fields))
	for _, field := range fields {
		cleaned := strings.Map(func(r rune) rune {
			if r == '"' || r == '*' {
				return -1
			}
			return r
		}, field)
		if cleaned == "" {
			continue
		}
		terms = append(terms, `"`+cleaned+`"*`)
	}
	return strings.Join(terms, " ")
}

func scanAlbums(rows *sql.Rows) ([]Album, error) {
	defer rows.Close()
	albums := []Album{}
	for rows.Next() {
		var a Album
		var isDark int
		if err := rows.Scan(&a.ID, &a.Name, &a.ArtistID, &a.ArtistName, &a.Year, &a.Genre,
			&a.ArtworkID, &a.Color, &isDark, &a.ReleaseType, &a.TrackCount, &a.DurationMS, &a.AddedAt, &a.DiscCount); err != nil {
			return nil, err
		}
		a.IsDark = isDark == 1
		albums = append(albums, a)
	}
	return albums, rows.Err()
}

func scanArtists(rows *sql.Rows) ([]Artist, error) {
	defer rows.Close()
	artists := []Artist{}
	for rows.Next() {
		var a Artist
		var isDark int
		if err := rows.Scan(&a.ID, &a.Name, &a.ArtworkID, &a.Color, &isDark,
			&a.AlbumCount, &a.TrackCount, &a.DurationMS); err != nil {
			return nil, err
		}
		a.IsDark = isDark == 1
		artists = append(artists, a)
	}
	return artists, rows.Err()
}

func scanTracks(rows *sql.Rows) ([]Track, error) {
	defer rows.Close()
	tracks := []Track{}
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Path, &t.Filename, &t.Title, &t.Artist, &t.AlbumArtist,
			&t.ArtistID, &t.AlbumID, &t.AlbumName, &t.Genre, &t.Year, &t.TrackNo, &t.DiscNo,
			&t.DurationMS, &t.BitrateKbps, &t.SampleRateHz, &t.Channels, &t.Format,
			&t.SizeBytes, &t.ArtworkID, &t.Color, &t.AddedAt); err != nil {
			return nil, err
		}
		tracks = append(tracks, t)
	}
	return tracks, rows.Err()
}

func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func toAny(values []string) []any {
	out := make([]any, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func indexOf(ids []string) map[string]int {
	out := make(map[string]int, len(ids))
	for i, id := range ids {
		out[id] = i
	}
	return out
}

func sortBy[T any](items []T, key func(T) int) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && key(items[j]) < key(items[j-1]); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func clampLimit(limit int) int {
	if limit <= 0 {
		return 100000
	}
	return limit
}
