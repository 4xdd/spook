// Command spook indexes a music directory and serves it to the web player.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/spook/server/internal/api"
	"github.com/spook/server/internal/artwork"
	"github.com/spook/server/internal/audio"
	"github.com/spook/server/internal/auth"
	"github.com/spook/server/internal/config"
	"github.com/spook/server/internal/deezer"
	"github.com/spook/server/internal/httpx"
	"github.com/spook/server/internal/lastfm"
	"github.com/spook/server/internal/lyrics"
	"github.com/spook/server/internal/scan"
	"github.com/spook/server/internal/store"
	"github.com/spook/server/internal/web"
)

func main() {
	cfg := config.Load()

	db, err := store.Open(cfg.DatabasePath())
	if err != nil {
		log.Fatalf("open library: %v", err)
	}
	defer db.Close()

	art, err := artwork.NewCache(cfg.ArtDir)
	if err != nil {
		log.Fatalf("open art cache: %v", err)
	}

	scanner := scan.New(cfg.MusicDir, db, art, audio.NewProber(cfg.UseFFprobe))

	deezerWorker := deezer.NewWorker(deezer.Settings{
		Enabled:   cfg.Deezer.Enabled,
		ARL:       cfg.Deezer.ARL,
		URL:       cfg.Deezer.URL,
		Port:      cfg.Deezer.Port,
		Quality:   cfg.Deezer.Quality,
		Command:   cfg.Deezer.Command,
		MusicDir:  cfg.MusicDir,
		ConfigDir: cfg.DataDir,
	}, scanner)
	if cfg.Deezer.Enabled && (cfg.Deezer.ARL != "" || cfg.Deezer.URL != "") {
		if err := deezerWorker.Start(); err != nil {
			log.Printf("deezer subworker: %v", err)
		}
	} else if cfg.Deezer.Enabled {
		log.Printf("deezer: disabled until DEEZER_ARL or -deezer-arl is set")
	}
	defer deezerWorker.Stop()

	lastfmClient := lastfm.New(cfg.LastFM.APIKey, cfg.LastFM.Secret)

	apiServer := &api.Server{
		Store:     db,
		Art:       art,
		Scanner:   scanner,
		Deezer:    deezerWorker,
		Lyrics:    lyrics.NewOnline(cfg.Lyrics.Enabled, cfg.Lyrics.BaseURL),
		LastFM:    lastfmClient,
		Root:      cfg.MusicDir,
		ChunkSize: cfg.ChunkSize,
	}

	accessKeys, createdKey, err := auth.Load(cfg.AccessKeysFile, cfg.AccessKeys)
	if err != nil {
		log.Fatalf("access keys: %v", err)
	}

	webHandler, err := web.Handler(cfg.WebDir)
	if err != nil {
		log.Fatalf("web assets: %v", err)
	}

	mux := http.NewServeMux()
	apiServer.Routes(mux)
	mux.Handle("/", webHandler)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpx.LogRequests(httpx.CORS(auth.Require(accessKeys, mux))),
		ReadHeaderTimeout: 10 * time.Second,
	}

	uiURL := "http://" + displayHost(cfg.Addr) + "/"
	log.Printf("spook listening on %s", uiURL)
	log.Printf("music: %s", cfg.MusicDir)
	log.Printf("library: %s", cfg.DatabasePath())
	log.Printf("art cache: %s", cfg.ArtDir)
	if createdKey != "" {
		log.Printf("access key created (saved to %s): %s", accessKeys.Path(), createdKey)
	} else {
		log.Printf("access keys: %d configured (%s)", accessKeys.Count(), accessKeys.Path())
	}
	if cfg.Lyrics.Enabled {
		log.Printf("lyrics: LRCLIB (https://lrclib.net)")
	} else {
		log.Printf("lyrics: online lookup disabled")
	}
	if lastfmClient.Configured() {
		log.Printf("last.fm: scrobbling enabled")
	} else {
		log.Printf("last.fm: disabled until LASTFM_API_KEY and LASTFM_API_SECRET are set")
	}

	if cfg.ScanOnStart {
		// The scan runs in the background so the UI is usable immediately,
		// serving whatever the previous scan already indexed.
		scanner.Trigger()
	}
	if cfg.OpenBrowser {
		go openBrowser(uiURL)
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	<-signals

	log.Println("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	scanner.Shutdown()
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return
	}
	if err := cmd.Start(); err != nil {
		log.Printf("open browser: %v", err)
	}
}

func displayHost(addr string) string {
	if addr == "" {
		return "localhost:8080"
	}
	if addr[0] == ':' {
		return "localhost" + addr
	}
	return addr
}
