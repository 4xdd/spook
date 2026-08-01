package store

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
)

// TrackEmbedding is a MERT audio vector for similarity search.
type TrackEmbedding struct {
	TrackID string
	ModTime int64
	Dim     int
	Vector  []float32
}

type PendingEmbedding struct {
	ID      string
	Path    string
	ModTime int64
}

// PendingEmbeddings lists tracks missing or stale embeddings.
func (s *Store) PendingEmbeddings(ctx context.Context, limit int) ([]PendingEmbedding, error) {
	if limit <= 0 {
		limit = 100000
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.path, t.mod_time
		FROM tracks t
		LEFT JOIN track_embeddings e ON e.track_id = t.id
		WHERE e.track_id IS NULL OR e.mod_time != t.mod_time
		ORDER BY t.added_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PendingEmbedding
	for rows.Next() {
		var p PendingEmbedding
		if err := rows.Scan(&p.ID, &p.Path, &p.ModTime); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) PutEmbedding(ctx context.Context, emb TrackEmbedding) error {
	return s.PutEmbeddings(ctx, []TrackEmbedding{emb})
}

// PutEmbeddings stores many vectors in one transaction.
func (s *Store) PutEmbeddings(ctx context.Context, embs []TrackEmbedding) error {
	if len(embs) == 0 {
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
		INSERT INTO track_embeddings(track_id, mod_time, dim, vector)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(track_id) DO UPDATE SET
			mod_time = excluded.mod_time,
			dim = excluded.dim,
			vector = excluded.vector`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, emb := range embs {
		if emb.TrackID == "" || len(emb.Vector) == 0 {
			return fmt.Errorf("invalid embedding for track %q", emb.TrackID)
		}
		if emb.Dim == 0 {
			emb.Dim = len(emb.Vector)
		}
		blob := encodeFloat32(emb.Vector)
		if _, err := stmt.ExecContext(ctx, emb.TrackID, emb.ModTime, emb.Dim, blob); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) AllEmbeddings(ctx context.Context) ([]TrackEmbedding, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT track_id, mod_time, dim, vector FROM track_embeddings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TrackEmbedding
	for rows.Next() {
		var emb TrackEmbedding
		var blob []byte
		if err := rows.Scan(&emb.TrackID, &emb.ModTime, &emb.Dim, &blob); err != nil {
			return nil, err
		}
		vec, err := decodeFloat32(blob, emb.Dim)
		if err != nil {
			return nil, err
		}
		emb.Vector = vec
		out = append(out, emb)
	}
	return out, rows.Err()
}

type EmbeddingCounts struct {
	Tracks     int
	Embedded   int
	Pending    int
}

func (s *Store) EmbeddingCounts(ctx context.Context) (EmbeddingCounts, error) {
	var counts EmbeddingCounts
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tracks`).Scan(&counts.Tracks)
	if err != nil {
		return counts, err
	}
	err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM track_embeddings`).Scan(&counts.Embedded)
	if err != nil {
		return counts, err
	}
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM tracks t
		LEFT JOIN track_embeddings e ON e.track_id = t.id
		WHERE e.track_id IS NULL OR e.mod_time != t.mod_time`).Scan(&counts.Pending)
	return counts, err
}

func encodeFloat32(v []float32) []byte {
	b := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

func decodeFloat32(b []byte, dim int) ([]float32, error) {
	if dim <= 0 {
		dim = len(b) / 4
	}
	if len(b) != dim*4 {
		return nil, fmt.Errorf("embedding blob length mismatch")
	}
	out := make([]float32, dim)
	for i := 0; i < dim; i++ {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}
