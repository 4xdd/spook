// Package scan indexes a music directory into the store.
//
// Scans are incremental: a file whose size and modification time are unchanged
// is never reopened, so rescanning a large library costs little more than a walk.
package scan

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spook/server/internal/artwork"
	"github.com/spook/server/internal/audio"
	"github.com/spook/server/internal/credits"
	"github.com/spook/server/internal/store"
	"github.com/spook/server/internal/tags"
)

const (
	StateIdle     = "idle"
	StateScanning = "scanning"
	StateDone     = "done"
	StateError    = "error"

	batchSize = 250
)

type Progress struct {
	State      string
	Total      int
	Processed  int
	Indexed    int
	Removed    int
	StartedAt  time.Time
	FinishedAt time.Time
	Error      string
}

type Scanner struct {
	root   string
	store  *store.Store
	art    *artwork.Cache
	prober *audio.Prober

	// A scan outlives the request that triggered it, so it runs under the
	// scanner's own context rather than the caller's.
	ctx    context.Context
	cancel context.CancelFunc

	mu       sync.Mutex
	running  bool
	progress Progress
	done     chan struct{}
}

func New(root string, st *store.Store, art *artwork.Cache, prober *audio.Prober) *Scanner {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scanner{
		root:     root,
		store:    st,
		art:      art,
		prober:   prober,
		ctx:      ctx,
		cancel:   cancel,
		progress: Progress{State: StateIdle},
	}
}

func (s *Scanner) Progress() Progress {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.progress
}

// Shutdown cancels an in-flight scan and waits for it to unwind.
func (s *Scanner) Shutdown() {
	s.cancel()

	s.mu.Lock()
	done := s.done
	s.mu.Unlock()

	if done != nil {
		<-done
	}
}

// Trigger starts a scan in the background, reporting whether it began.
func (s *Scanner) Trigger() bool {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return false
	}
	s.running = true
	s.progress = Progress{State: StateScanning, StartedAt: time.Now()}
	s.done = make(chan struct{})
	done := s.done
	s.mu.Unlock()

	go func() {
		defer close(done)
		err := s.run(s.ctx)

		s.mu.Lock()
		defer s.mu.Unlock()
		s.running = false
		s.progress.FinishedAt = time.Now()
		if err != nil {
			s.progress.State = StateError
			s.progress.Error = err.Error()
			log.Printf("scan failed: %v", err)
			return
		}
		s.progress.State = StateDone
		log.Printf("scan complete: %d indexed, %d removed, %d total in %s",
			s.progress.Indexed, s.progress.Removed, s.progress.Total,
			s.progress.FinishedAt.Sub(s.progress.StartedAt).Round(time.Millisecond))
	}()

	return true
}

type candidate struct {
	path    string
	size    int64
	modTime int64
	dir     string
}

func (s *Scanner) run(ctx context.Context) error {
	info, err := os.Stat(s.root)
	if err != nil {
		return fmt.Errorf("music directory %q: %w", s.root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("music directory %q is not a directory", s.root)
	}

	candidates, err := s.walk()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.progress.Total = len(candidates)
	s.mu.Unlock()

	states, err := s.store.FileStates(ctx)
	if err != nil {
		return fmt.Errorf("read index: %w", err)
	}

	if records, err := s.store.AllArtwork(ctx); err == nil {
		seeds := make([]artwork.Record, 0, len(records))
		for _, record := range records {
			seeds = append(seeds, artwork.Record{
				ID: record.ID, Mime: record.Mime, Width: record.Width, Height: record.Height,
				Color: record.Color, IsDark: record.IsDark, Source: record.Source,
			})
		}
		s.art.Seed(seeds)
	}
	s.art.Reset()

	pending := make([]candidate, 0, len(candidates))
	seen := make(map[string]bool, len(candidates))
	for _, item := range candidates {
		seen[item.path] = true
		if state, ok := states[item.path]; ok && state.Size == item.size && state.ModTime == item.modTime {
			continue
		}
		pending = append(pending, item)
	}

	s.mu.Lock()
	s.progress.Processed = len(candidates) - len(pending)
	s.mu.Unlock()

	if err := s.index(ctx, pending, states); err != nil {
		return err
	}

	var removed []string
	for path := range states {
		if !seen[path] {
			removed = append(removed, path)
		}
	}
	if err := s.store.DeleteTracks(ctx, removed); err != nil {
		return fmt.Errorf("prune removed tracks: %w", err)
	}

	s.mu.Lock()
	s.progress.Removed = len(removed)
	s.mu.Unlock()

	if err := s.store.Rebuild(ctx); err != nil {
		return err
	}
	if err := s.store.SetMeta(ctx, "last_scan", strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
		return err
	}
	if err := s.store.SetMeta(ctx, "root", s.root); err != nil {
		return err
	}
	return s.store.Vacuum(ctx)
}

func (s *Scanner) walk() ([]candidate, error) {
	var out []candidate

	err := filepath.WalkDir(s.root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			log.Printf("skip %q: %v", path, walkErr)
			return nil
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") && path != s.root {
				return filepath.SkipDir
			}
			return nil
		}
		if !tags.IsSupported(path) || strings.HasPrefix(entry.Name(), ".") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		out = append(out, candidate{
			path:    path,
			size:    info.Size(),
			modTime: info.ModTime().Unix(),
			dir:     filepath.Dir(path),
		})
		return nil
	})

	return out, err
}

func (s *Scanner) index(ctx context.Context, pending []candidate, states map[string]store.FileState) error {
	if len(pending) == 0 {
		return nil
	}

	workers := runtime.NumCPU()
	if workers > 8 {
		workers = 8
	}
	if workers > len(pending) {
		workers = len(pending)
	}

	jobs := make(chan candidate)
	results := make(chan store.Track, workers*2)

	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for item := range jobs {
				track := s.buildTrack(item, states)

				s.mu.Lock()
				s.progress.Processed++
				s.mu.Unlock()

				select {
				case results <- track:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, item := range pending {
			select {
			case jobs <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		group.Wait()
		close(results)
	}()

	batch := make([]store.Track, 0, batchSize)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := s.store.UpsertTracks(ctx, batch); err != nil {
			return err
		}
		s.mu.Lock()
		s.progress.Indexed += len(batch)
		s.mu.Unlock()
		batch = batch[:0]
		return nil
	}

	for track := range results {
		batch = append(batch, track)
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	return s.persistArtwork(ctx)
}

func (s *Scanner) persistArtwork(ctx context.Context) error {
	for _, record := range s.art.Records() {
		err := s.store.PutArtwork(ctx, store.Artwork{
			ID: record.ID, Mime: record.Mime, Width: record.Width, Height: record.Height,
			Color: record.Color, IsDark: record.IsDark, Source: record.Source,
		})
		if err != nil {
			return fmt.Errorf("record artwork: %w", err)
		}
	}
	return nil
}

func buildArtistCredits(names []string) []store.ArtistCredit {
	out := make([]store.ArtistCredit, 0, len(names))
	for i, name := range names {
		out = append(out, store.ArtistCredit{
			ID:       artistID(name),
			Name:     name,
			Position: i,
		})
	}
	return out
}

func (s *Scanner) buildTrack(item candidate, states map[string]store.FileState) store.Track {
	meta := tags.Read(item.path)
	info := s.prober.Probe(item.path, item.size)

	container := albumContainerName(s.root, item.dir)
	title := firstNonEmpty(meta.Title, strings.TrimSuffix(filepath.Base(item.path), filepath.Ext(item.path)))
	// Tagged credits decide the album primary; title "feat." guests are appended
	// afterwards so a one-line (feat. X) never owns the release.
	names := credits.Merge(meta.Artist, meta.AlbumArtist)
	albumArtist := credits.Primary(names)
	if albumArtist == "" {
		albumArtist = resolveAlbumArtist(meta.Artist, meta.AlbumArtist, container)
		names = credits.Merge(albumArtist)
	}
	names = credits.Dedupe(append(names, credits.FromTitle(title)...))
	artist := credits.Format(names)
	if artist == "" {
		artist = firstNonEmpty(meta.Artist, albumArtist, "Unknown Artist")
	}
	album := resolveAlbumName(meta.Album, container)
	if album == "" || album == "Unknown Album" {
		album = firstNonEmpty(folderAlbumName(item.dir), "Unknown Album")
	}

	addedAt := time.Now().Unix()
	if state, ok := states[item.path]; ok && state.AddedAt > 0 {
		addedAt = state.AddedAt
	}

	track := store.Track{
		ID:           trackID(item.path),
		Path:         item.path,
		Filename:     filepath.Base(item.path),
		Title:        title,
		SortTitle:    sortKey(title),
		Artist:       artist,
		AlbumArtist:  albumArtist,
		ArtistID:     artistID(albumArtist),
		Credits:      buildArtistCredits(names),
		AlbumID:      albumID(albumArtist, album, container),
		AlbumName:    album,
		Genre:        meta.Genre,
		Year:         sanitizeYear(meta.Year),
		TrackNo:      meta.TrackNo,
		DiscNo:       meta.DiscNo,
		DurationMS:   info.DurationMS,
		BitrateKbps:  info.BitrateKbps,
		SampleRateHz: info.SampleRateHz,
		Channels:     info.Channels,
		Format:       strings.TrimPrefix(strings.ToLower(filepath.Ext(item.path)), "."),
		SizeBytes:    item.size,
		ModTime:      item.modTime,
		AddedAt:      addedAt,
	}
	track.SortTitle = sortKey(track.Title)
	track.ArtworkID = s.resolveArtwork(meta, item.dir)

	return track
}

// resolveArtwork prefers the picture embedded in the file and falls back to an
// image sitting in the album folder.
func (s *Scanner) resolveArtwork(meta tags.Metadata, dir string) string {
	if len(meta.Picture) > 0 {
		if record, err := s.art.Put(meta.Picture, artwork.SourceEmbedded); err == nil {
			return record.ID
		} else if !errors.Is(err, os.ErrNotExist) {
			log.Printf("embedded artwork in %s: %v", dir, err)
		}
	}
	if record, ok := s.art.PutFolderImage(dir); ok {
		return record.ID
	}
	return ""
}

// sanitizeYear rejects the nonsense values that malformed date tags produce,
// which would otherwise win the MAX() used to date an album.
func sanitizeYear(year int) int {
	if year < 1500 || year > time.Now().Year()+1 {
		return 0
	}
	return year
}

// folderAlbumName treats the containing directory as the album for untagged
// files, which reads far better than "Unknown Album" for ripped folders.
func folderAlbumName(dir string) string {
	name := filepath.Base(dir)
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
