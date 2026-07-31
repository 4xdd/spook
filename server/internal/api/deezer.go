package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/spook/server/internal/deezer"
	"github.com/spook/server/internal/httpx"
)

var errDeezerUnavailable = errors.New("deezer unavailable")

func (s *Server) deezerStatus(w http.ResponseWriter, r *http.Request) {
	if s.Deezer == nil {
		httpx.WriteJSON(w, http.StatusOK, deezer.Status{Enabled: false, Error: "Deezer integration disabled"})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, s.Deezer.Status())
}

func (s *Server) deezerSearch(w http.ResponseWriter, r *http.Request) {
	if err := s.requireDeezer(w); err != nil {
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		httpx.WriteError(w, http.StatusBadRequest, "query is required")
		return
	}

	searchType := deezer.SearchType(r.URL.Query().Get("type"))
	switch searchType {
	case "", deezer.SearchTrack:
		searchType = deezer.SearchTrack
	case deezer.SearchAlbum, deezer.SearchArtist:
	default:
		httpx.WriteError(w, http.StatusBadRequest, "type must be track, album, or artist")
		return
	}

	client, err := s.Deezer.Client()
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	results, err := client.Search(r.Context(), searchType, query)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"type":    searchType,
		"results": results,
	})
}

type deezerDownloadRequest struct {
	Type     deezer.DownloadType `json:"type"`
	MusicID  int                 `json:"musicId"`
	MusicID2 int                 `json:"music_id"`
}

func (s *Server) deezerDownload(w http.ResponseWriter, r *http.Request) {
	if err := s.requireDeezer(w); err != nil {
		return
	}

	var body deezerDownloadRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json body")
		return
	}

	musicID := body.MusicID
	if musicID == 0 {
		musicID = body.MusicID2
	}
	if musicID <= 0 {
		httpx.WriteError(w, http.StatusBadRequest, "musicId is required")
		return
	}
	if body.Type != deezer.DownloadTrack && body.Type != deezer.DownloadAlbum {
		httpx.WriteError(w, http.StatusBadRequest, "type must be track or album")
		return
	}

	client, err := s.Deezer.Client()
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	taskID, err := client.Download(r.Context(), body.Type, musicID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusAccepted, map[string]any{
		"taskId": taskID,
		"type":   body.Type,
		"musicId": musicID,
	})
}

func (s *Server) deezerJobs(w http.ResponseWriter, r *http.Request) {
	if err := s.requireDeezer(w); err != nil {
		return
	}

	client, err := s.Deezer.Client()
	if err != nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
		return
	}

	jobs, err := client.Jobs(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}

	httpx.WriteJSON(w, http.StatusOK, map[string]any{"jobs": jobs})
}

func (s *Server) requireDeezer(w http.ResponseWriter) error {
	if s.Deezer == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Deezer integration is disabled")
		return errDeezerUnavailable
	}
	status := s.Deezer.Status()
	if !status.Enabled {
		httpx.WriteError(w, http.StatusServiceUnavailable, "Deezer integration is disabled")
		return errDeezerUnavailable
	}
	if !status.Configured {
		httpx.WriteError(w, http.StatusServiceUnavailable, status.Error)
		return errDeezerUnavailable
	}
	if !status.Running {
		message := status.Error
		if message == "" {
			message = "deezer subworker is not running"
		}
		httpx.WriteError(w, http.StatusServiceUnavailable, message)
		return errDeezerUnavailable
	}
	return nil
}
