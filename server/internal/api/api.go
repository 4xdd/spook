// Package api exposes the library over a typed JSON HTTP interface.
package api

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spook/server/internal/artwork"
	"github.com/spook/server/internal/audio"
	"github.com/spook/server/internal/deezer"
	"github.com/spook/server/internal/httpx"
	"github.com/spook/server/internal/lastfm"
	"github.com/spook/server/internal/lyrics"
	"github.com/spook/server/internal/scan"
	"github.com/spook/server/internal/store"
)

type Server struct {
	Store     *store.Store
	Art       *artwork.Cache
	Scanner   *scan.Scanner
	Deezer    *deezer.Worker
	Lyrics    *lyrics.Online
	LastFM    *lastfm.Client
	Stream    *audio.Streamer
	Root      string
	ChunkSize int64
}

func (s *Server) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /api/v1/stats", s.stats)
	mux.HandleFunc("GET /api/v1/albums", s.albums)
	mux.HandleFunc("GET /api/v1/albums/{id}", s.album)
	mux.HandleFunc("GET /api/v1/artists", s.artists)
	mux.HandleFunc("GET /api/v1/artists/{id}", s.artist)
	mux.HandleFunc("GET /api/v1/tracks", s.tracks)
	mux.HandleFunc("GET /api/v1/tracks/{id}", s.track)
	mux.HandleFunc("GET /api/v1/tracks/{id}/lyrics", s.lyrics)
	mux.HandleFunc("GET /api/v1/search", s.search)
	mux.HandleFunc("GET /api/v1/art/{id}", s.art)
	mux.HandleFunc("GET /api/v1/stream/{id}", s.stream)
	mux.HandleFunc("HEAD /api/v1/stream/{id}", s.stream)
	mux.HandleFunc("GET /api/v1/scan", s.scanStatus)
	mux.HandleFunc("POST /api/v1/scan", s.startScan)
	mux.HandleFunc("GET /api/v1/deezer/status", s.deezerStatus)
	mux.HandleFunc("GET /api/v1/deezer/search", s.deezerSearch)
	mux.HandleFunc("POST /api/v1/deezer/download", s.deezerDownload)
	mux.HandleFunc("GET /api/v1/deezer/jobs", s.deezerJobs)

	mux.HandleFunc("GET /api/v1/lastfm/status", s.lastfmStatus)
	mux.HandleFunc("GET /api/v1/lastfm/auth-url", s.lastfmAuthURL)
	mux.HandleFunc("POST /api/v1/lastfm/session", s.lastfmSession)
	mux.HandleFunc("POST /api/v1/lastfm/now-playing", s.lastfmNowPlaying)
	mux.HandleFunc("POST /api/v1/lastfm/scrobble", s.lastfmScrobble)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	counts, err := s.Store.Stats(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lastScan, _ := s.Store.Meta(r.Context(), "last_scan")
	parsed, _ := strconv.ParseInt(lastScan, 10, 64)

	httpx.WriteJSON(w, http.StatusOK, Stats{
		Root:       s.Root,
		Tracks:     counts.Tracks,
		Albums:     counts.Albums,
		Artists:    counts.Artists,
		DurationMs: counts.DurationMS,
		LastScan:   parsed,
		Scan:       toScanStatus(s.Scanner.Progress()),
	})
}

func (s *Server) albums(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	albums, err := s.Store.Albums(r.Context(), query.Get("sort"),
		intParam(query.Get("limit"), 0), intParam(query.Get("offset"), 0))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toAlbums(albums))
}

func (s *Server) album(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	album, err := s.Store.Album(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	tracks, err := s.Store.TracksByAlbum(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, AlbumDetail{Album: toAlbum(album), Tracks: toTracks(tracks)})
}

func (s *Server) artists(w http.ResponseWriter, r *http.Request) {
	artists, err := s.Store.Artists(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toArtists(artists))
}

func (s *Server) artist(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	artist, err := s.Store.Artist(r.Context(), id)
	if err != nil {
		s.fail(w, err)
		return
	}
	albums, err := s.Store.AlbumsByArtist(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	tracks, err := s.Store.TracksByArtist(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, ArtistDetail{
		Artist: toArtist(artist),
		Albums: toAlbums(albums),
		Tracks: toTracks(tracks),
	})
}

func (s *Server) tracks(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	tracks, err := s.Store.Tracks(r.Context(), query.Get("sort"),
		intParam(query.Get("limit"), 0), intParam(query.Get("offset"), 0))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTracks(tracks))
}

func (s *Server) track(w http.ResponseWriter, r *http.Request) {
	track, err := s.Store.Track(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, toTrack(track))
}

// lyrics reads from the file first, then online providers with a hard deadline.
func (s *Server) lyrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	track, err := s.Store.Track(ctx, r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	found := lyrics.Lookup(ctx, track.Path, lyrics.Track{
		Title:      track.Title,
		Artist:     track.Artist,
		Album:      track.AlbumName,
		DurationMS: track.DurationMS,
	}, s.Lyrics)
	httpx.WriteJSON(w, http.StatusOK, toLyrics(track.ID, found))
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	results, err := s.Store.Search(r.Context(), query, intParam(r.URL.Query().Get("limit"), 20))
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, SearchResults{
		Query:   query,
		Artists: toArtists(results.Artists),
		Albums:  toAlbums(results.Albums),
		Tracks:  toTracks(results.Tracks),
	})
}

func (s *Server) art(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	size := artwork.NearestSize(intParam(r.URL.Query().Get("size"), 300))
	path := s.Art.Path(id, size)

	file, err := os.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Cache keys are content hashes, so a cached image can never go stale.
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(w, r, path, info.ModTime(), file)
}

func (s *Server) stream(w http.ResponseWriter, r *http.Request) {
	track, err := s.Store.Track(r.Context(), r.PathValue("id"))
	if err != nil {
		s.fail(w, err)
		return
	}
	w.Header().Set("X-Spook-Track-Id", track.ID)

	if transcodeRequested(r) {
		if s.Stream == nil || !s.Stream.Available() {
			httpx.WriteError(w, http.StatusServiceUnavailable, "transcoding requires ffmpeg")
			return
		}
		s.Stream.StreamMP3(w, r, track.Path, s.ChunkSize)
		return
	}

	httpx.ServeAudio(w, r, track.Path, s.ChunkSize)
}

func transcodeRequested(r *http.Request) bool {
	switch r.URL.Query().Get("transcode") {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func (s *Server) scanStatus(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, toScanStatus(s.Scanner.Progress()))
}

func (s *Server) startScan(w http.ResponseWriter, r *http.Request) {
	started := s.Scanner.Trigger()
	status := http.StatusAccepted
	if !started {
		status = http.StatusConflict
	}
	httpx.WriteJSON(w, status, toScanStatus(s.Scanner.Progress()))
}

func (s *Server) fail(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		httpx.WriteError(w, http.StatusNotFound, "not found")
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, err.Error())
}

func intParam(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}
