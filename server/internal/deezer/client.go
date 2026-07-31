package deezer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client talks to a running deezer-downloader HTTP API.
type Client struct {
	base   string
	client *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		base: strings.TrimRight(baseURL, "/"),
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (c *Client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/queue", nil)
	if err != nil {
		return err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("deezer-downloader returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) Search(ctx context.Context, searchType SearchType, query string) ([]Result, error) {
	payload := map[string]string{
		"type":  string(searchType),
		"query": query,
	}
	var raw []map[string]any
	if err := c.postJSON(ctx, "/search", payload, &raw); err != nil {
		return nil, err
	}
	return normalizeResults(searchType, raw), nil
}

func (c *Client) Download(ctx context.Context, downloadType DownloadType, musicID int) (int, error) {
	payload := map[string]any{
		"type":            string(downloadType),
		"music_id":        musicID,
		"add_to_playlist": false,
		"create_zip":      false,
	}
	var resp struct {
		TaskID int `json:"task_id"`
	}
	if err := c.postJSON(ctx, "/download", payload, &resp); err != nil {
		return 0, err
	}
	return resp.TaskID, nil
}

func (c *Client) Jobs(ctx context.Context) ([]Job, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/queue", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("queue request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw []struct {
		ID          int           `json:"id"`
		Description string        `json:"description"`
		State       string        `json:"state"`
		Exception   string        `json:"exception"`
		Progress    []json.Number `json:"progress"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	jobs := make([]Job, 0, len(raw))
	for _, item := range raw {
		job := Job{
			ID:          item.ID,
			Description: item.Description,
			State:       item.State,
			Error:       item.Exception,
		}
		if len(item.Progress) > 0 {
			if v, err := item.Progress[0].Int64(); err == nil {
				job.Progress = int(v)
			}
		}
		if len(item.Progress) > 1 {
			if v, err := item.Progress[1].Int64(); err == nil {
				job.ProgressMax = int(v)
			}
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		var errBody struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errBody)
		if errBody.Error != "" {
			return fmt.Errorf("%s", errBody.Error)
		}
		return fmt.Errorf("request failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(raw, out)
}

func normalizeResults(searchType SearchType, raw []map[string]any) []Result {
	out := make([]Result, 0, len(raw))
	for _, item := range raw {
		result := Result{
			ID:         stringField(item, "id"),
			Type:       searchType,
			Title:      stringField(item, "title"),
			Album:      stringField(item, "album"),
			AlbumID:    stringField(item, "album_id"),
			Artist:     stringField(item, "artist"),
			ArtistID:   stringField(item, "artist_id"),
			ImageURL:   stringField(item, "img_url"),
			PreviewURL: stringField(item, "preview_url"),
		}
		if idType := stringField(item, "id_type"); idType != "" {
			switch idType {
			case "album":
				result.Type = SearchAlbum
			case "artist":
				result.Type = SearchArtist
			default:
				result.Type = SearchTrack
			}
		}
		if result.ID == "" {
			continue
		}
		out = append(out, result)
	}
	return out
}

func stringField(item map[string]any, key string) string {
	value, ok := item[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatInt(int64(typed), 10)
	case json.Number:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}
