package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/spook/server/internal/httpx"
	"github.com/spook/server/internal/lastfm"
)

func (s *Server) lastfmStatus(w http.ResponseWriter, r *http.Request) {
	configured := s.LastFM != nil && s.LastFM.Configured()
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"enabled":    configured,
		"configured": configured,
	})
}

func (s *Server) lastfmAuthURL(w http.ResponseWriter, r *http.Request) {
	if s.LastFM == nil || !s.LastFM.Configured() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "last.fm is not configured")
		return
	}
	callback := r.URL.Query().Get("callback")
	authURL, err := s.LastFM.AuthURL(callback)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"url": authURL})
}

type lastfmSessionRequest struct {
	Token string `json:"token"`
}

func (s *Server) lastfmSession(w http.ResponseWriter, r *http.Request) {
	if s.LastFM == nil || !s.LastFM.Configured() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "last.fm is not configured")
		return
	}
	var body lastfmSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	session, err := s.LastFM.GetSession(r.Context(), body.Token)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, lastfm.ErrNotConfigured) {
			status = http.StatusServiceUnavailable
		}
		httpx.WriteError(w, status, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, session)
}

type lastfmTrackRequest struct {
	SessionKey  string `json:"sessionKey"`
	Artist      string `json:"artist"`
	Title       string `json:"title"`
	Album       string `json:"album"`
	AlbumArtist string `json:"albumArtist"`
	TrackNumber int    `json:"trackNumber"`
	DurationSec int    `json:"durationSec"`
	Timestamp   int64  `json:"timestamp"`
}

func (req lastfmTrackRequest) track() lastfm.Track {
	return lastfm.Track{
		Artist:      lastfm.PrimaryArtist(req.Artist),
		Title:       req.Title,
		Album:       req.Album,
		AlbumArtist: req.AlbumArtist,
		TrackNumber: req.TrackNumber,
		DurationSec: req.DurationSec,
		Timestamp:   req.Timestamp,
	}
}

func (s *Server) lastfmNowPlaying(w http.ResponseWriter, r *http.Request) {
	s.lastfmWrite(w, r, false)
}

func (s *Server) lastfmScrobble(w http.ResponseWriter, r *http.Request) {
	s.lastfmWrite(w, r, true)
}

func (s *Server) lastfmWrite(w http.ResponseWriter, r *http.Request, scrobble bool) {
	if s.LastFM == nil || !s.LastFM.Configured() {
		httpx.WriteError(w, http.StatusServiceUnavailable, "last.fm is not configured")
		return
	}
	var body lastfmTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	var err error
	if scrobble {
		err = s.LastFM.Scrobble(r.Context(), body.SessionKey, body.track())
	} else {
		err = s.LastFM.UpdateNowPlaying(r.Context(), body.SessionKey, body.track())
	}
	if err != nil {
		httpx.WriteError(w, http.StatusBadGateway, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
