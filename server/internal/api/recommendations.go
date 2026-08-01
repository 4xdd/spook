package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/spook/server/internal/httpx"
)

func (s *Server) recommendations(w http.ResponseWriter, r *http.Request) {
	if s.Recommend == nil || s.Recommend.Len() == 0 {
		httpx.WriteJSON(w, http.StatusOK, Recommendations{Tracks: []Track{}})
		return
	}

	q := r.URL.Query()
	seed := splitCSV(q.Get("seed"))
	exclude := splitCSV(q.Get("exclude"))
	limit := intParam(q.Get("limit"), 10)
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	var nonce int64
	if raw := q.Get("nonce"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			nonce = parsed
		}
	}

	tracks, err := s.Recommend.Tracks(r.Context(), seed, exclude, limit, nonce)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, Recommendations{Tracks: toTracks(tracks)})
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			out = append(out, id)
		}
	}
	return out
}
