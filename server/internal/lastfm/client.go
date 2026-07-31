// Package lastfm talks to the Last.fm 2.0 web service for auth and scrobbling.
package lastfm

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const apiBase = "https://ws.audioscrobbler.com/2.0/"

var ErrNotConfigured = errors.New("last.fm api key not configured")

type Client struct {
	APIKey string
	Secret string
	HTTP   *http.Client
}

func New(apiKey, secret string) *Client {
	return &Client{
		APIKey: strings.TrimSpace(apiKey),
		Secret: strings.TrimSpace(secret),
		HTTP:   &http.Client{Timeout: 12 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c != nil && c.APIKey != "" && c.Secret != ""
}

func (c *Client) AuthURL(callback string) (string, error) {
	if !c.Configured() {
		return "", ErrNotConfigured
	}
	u, err := url.Parse("https://www.last.fm/api/auth/")
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("api_key", c.APIKey)
	if callback != "" {
		q.Set("cb", callback)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

type Session struct {
	Name string `json:"name"`
	Key  string `json:"key"`
}

type sessionResponse struct {
	Session Session `json:"session"`
	Message string  `json:"message"`
	Error   int     `json:"error"`
}

func (c *Client) GetSession(ctx context.Context, token string) (Session, error) {
	if !c.Configured() {
		return Session{}, ErrNotConfigured
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return Session{}, errors.New("missing auth token")
	}

	params := map[string]string{
		"method":  "auth.getSession",
		"api_key": c.APIKey,
		"token":   token,
	}
	var body sessionResponse
	if err := c.call(ctx, http.MethodGet, params, &body); err != nil {
		return Session{}, err
	}
	if body.Error != 0 {
		return Session{}, fmt.Errorf("last.fm: %s", body.Message)
	}
	if body.Session.Key == "" || body.Session.Name == "" {
		return Session{}, errors.New("last.fm: empty session")
	}
	return body.Session, nil
}

type Track struct {
	Artist      string
	Title       string
	Album       string
	AlbumArtist string
	TrackNumber int
	DurationSec int
	Timestamp   int64 // unix seconds; required for scrobble
}

type writeResponse struct {
	Message string `json:"message"`
	Error   int    `json:"error"`
}

func (c *Client) UpdateNowPlaying(ctx context.Context, sessionKey string, track Track) error {
	params, err := c.trackParams("track.updateNowPlaying", sessionKey, track, false)
	if err != nil {
		return err
	}
	var body writeResponse
	if err := c.call(ctx, http.MethodPost, params, &body); err != nil {
		return err
	}
	if body.Error != 0 {
		return fmt.Errorf("last.fm: %s", body.Message)
	}
	return nil
}

func (c *Client) Scrobble(ctx context.Context, sessionKey string, track Track) error {
	params, err := c.trackParams("track.scrobble", sessionKey, track, true)
	if err != nil {
		return err
	}
	var body writeResponse
	if err := c.call(ctx, http.MethodPost, params, &body); err != nil {
		return err
	}
	if body.Error != 0 {
		return fmt.Errorf("last.fm: %s", body.Message)
	}
	return nil
}

func (c *Client) trackParams(method, sessionKey string, track Track, scrobble bool) (map[string]string, error) {
	if !c.Configured() {
		return nil, ErrNotConfigured
	}
	sessionKey = strings.TrimSpace(sessionKey)
	if sessionKey == "" {
		return nil, errors.New("missing session key")
	}
	artist := strings.TrimSpace(track.Artist)
	title := strings.TrimSpace(track.Title)
	if artist == "" || title == "" {
		return nil, errors.New("artist and track are required")
	}

	params := map[string]string{
		"method":  method,
		"api_key": c.APIKey,
		"sk":      sessionKey,
		"artist":  artist,
		"track":   title,
	}
	if album := strings.TrimSpace(track.Album); album != "" {
		params["album"] = album
	}
	if albumArtist := strings.TrimSpace(track.AlbumArtist); albumArtist != "" {
		params["albumArtist"] = albumArtist
	}
	if track.TrackNumber > 0 {
		params["trackNumber"] = fmt.Sprintf("%d", track.TrackNumber)
	}
	if track.DurationSec > 0 {
		params["duration"] = fmt.Sprintf("%d", track.DurationSec)
	}
	if scrobble {
		if track.Timestamp <= 0 {
			return nil, errors.New("timestamp is required")
		}
		params["timestamp"] = fmt.Sprintf("%d", track.Timestamp)
	}
	return params, nil
}

func (c *Client) call(ctx context.Context, method string, params map[string]string, dest any) error {
	params["api_sig"] = Sign(params, c.Secret)
	params["format"] = "json"

	var req *http.Request
	var err error
	if method == http.MethodGet {
		u, parseErr := url.Parse(apiBase)
		if parseErr != nil {
			return parseErr
		}
		q := u.Query()
		for key, value := range params {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	} else {
		form := url.Values{}
		for key, value := range params {
			form.Set(key, value)
		}
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, apiBase, strings.NewReader(form.Encode()))
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		}
	}
	if err != nil {
		return err
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("last.fm: decode response: %w", err)
	}
	return nil
}

// Sign builds the Last.fm api_sig: md5(sorted key+value pairs + secret).
// The format parameter is never signed.
func Sign(params map[string]string, secret string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		if key == "format" || key == "callback" || key == "api_sig" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
		b.WriteString(params[key])
	}
	b.WriteString(secret)
	sum := md5.Sum([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// PrimaryArtist picks the first credited name from Spook's "A · B" display form.
func PrimaryArtist(display string) string {
	display = strings.TrimSpace(display)
	if display == "" {
		return display
	}
	if before, _, ok := strings.Cut(display, " · "); ok {
		return strings.TrimSpace(before)
	}
	return display
}
