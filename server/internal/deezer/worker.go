package deezer

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	healthCheckInterval = 5 * time.Second
	minRestartBackoff   = 5 * time.Second
)

// ScanTrigger rescans the music library after downloads finish.
type ScanTrigger interface {
	Trigger() bool
}

// Worker manages a deezer-downloader child process and watches its download queue.
type Worker struct {
	settings Settings
	scanner  ScanTrigger

	mu      sync.RWMutex
	client  *Client
	cmd     *exec.Cmd
	running bool
	lastErr string

	restartMu       sync.Mutex
	lastRestartFail time.Time
	manageProcess   bool

	ctx    context.Context
	cancel context.CancelFunc
}

func NewWorker(settings Settings, scanner ScanTrigger) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		settings:      settings,
		scanner:       scanner,
		ctx:           ctx,
		cancel:        cancel,
		manageProcess: strings.TrimSpace(settings.URL) == "",
	}
}

// Start prepares config, launches deezer-downloader when needed, and begins queue polling.
func (w *Worker) Start() error {
	if !w.settings.Enabled || !w.settings.Configured() {
		return nil
	}

	if err := w.ensureReady(context.Background()); err != nil {
		return err
	}

	if moved, err := OrganizeAlbumDownloads(w.settings.MusicDir); err != nil {
		log.Printf("deezer organize albums: %v", err)
	} else if moved && w.scanner != nil {
		w.scanner.Trigger()
	}

	go w.pollQueue()
	log.Printf("deezer subworker ready at %s (downloads -> %s)", w.settings.SpawnURL(), w.settings.MusicDir)
	return nil
}

// Stop shuts down polling and the child process if Spook started it.
func (w *Worker) Stop() {
	w.cancel()
	w.stopProcess()
}

func (w *Worker) Status() Status {
	w.mu.RLock()
	defer w.mu.RUnlock()

	status := Status{
		Enabled:    w.settings.Enabled,
		Configured: w.settings.Configured(),
		BaseURL:    w.settings.SpawnURL(),
		MusicDir:   w.settings.MusicDir,
		Quality:    w.settings.Quality,
		Running:    w.running && w.client != nil,
		Error:      w.lastErr,
	}
	if !w.settings.Enabled {
		status.Error = "Deezer integration disabled"
	} else if !w.settings.Configured() {
		status.Error = "Set DEEZER_ARL or -deezer-arl with your Deezer ARL cookie"
	}
	return status
}

func (w *Worker) Client() (*Client, error) {
	if err := w.ensureReady(context.Background()); err != nil {
		return nil, err
	}

	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.client == nil {
		return nil, fmt.Errorf("deezer subworker is not running")
	}
	return w.client, nil
}

func (w *Worker) ensureReady(ctx context.Context) error {
	w.mu.RLock()
	client := w.client
	running := w.running
	w.mu.RUnlock()

	if client != nil && running {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := client.Ping(pingCtx)
		cancel()
		if err == nil {
			return nil
		}
		w.markUnavailable(fmt.Sprintf("deezer subworker unreachable: %v", err))
	}

	if !w.manageProcess {
		client = NewClient(w.settings.SpawnURL())
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := client.Ping(pingCtx)
		cancel()
		if err == nil {
			w.mu.Lock()
			w.client = client
			w.running = true
			w.lastErr = ""
			w.mu.Unlock()
			return nil
		}
		return fmt.Errorf("deezer subworker is not running")
	}

	w.restartMu.Lock()
	defer w.restartMu.Unlock()

	if time.Since(w.lastRestartFail) < minRestartBackoff {
		w.mu.RLock()
		errMsg := w.lastErr
		w.mu.RUnlock()
		if errMsg == "" {
			errMsg = "deezer subworker is restarting"
		}
		return fmt.Errorf("%s", errMsg)
	}

	w.stopProcessLocked()
	if err := w.spawnLocked(); err != nil {
		w.lastRestartFail = time.Now()
		w.mu.Lock()
		w.lastErr = err.Error()
		w.mu.Unlock()
		return err
	}
	return nil
}

func (w *Worker) spawnLocked() error {
	client := NewClient(w.settings.SpawnURL())

	if w.manageProcess {
		ytDLP := findExecutable("yt-dlp", "/usr/bin/yt-dlp")
		command := w.settings.Command
		if command == "" {
			command = findExecutable("deezer-downloader")
		}
		if command == "" {
			return fmt.Errorf("deezer-downloader not found in PATH")
		}
		if strings.TrimSpace(w.settings.ARL) == "" {
			return fmt.Errorf("DEEZER_ARL is required when Spook spawns deezer-downloader")
		}
		if err := WriteConfig(w.settings.ConfigPath(), w.settings.MusicDir, w.settings.ARL, w.settings.Quality, ytDLP, w.settings.Port); err != nil {
			return fmt.Errorf("write deezer config: %w", err)
		}

		cmd := exec.CommandContext(w.ctx, command, "--config", w.settings.ConfigPath())
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("start deezer-downloader: %w", err)
		}
		w.cmd = cmd
		w.watchChild(cmd)
	}

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		pingCtx, cancel := context.WithTimeout(w.ctx, 3*time.Second)
		err := client.Ping(pingCtx)
		cancel()
		if err == nil {
			w.mu.Lock()
			w.client = client
			w.running = true
			w.lastErr = ""
			w.mu.Unlock()
			return nil
		}
		if w.cmd != nil && w.cmd.ProcessState != nil && w.cmd.ProcessState.Exited() {
			return fmt.Errorf("deezer-downloader exited before becoming ready")
		}
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return fmt.Errorf("deezer-downloader not reachable at %s", w.settings.SpawnURL())
}

func (w *Worker) watchChild(cmd *exec.Cmd) {
	go func() {
		err := cmd.Wait()
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.cmd != cmd {
			return
		}
		w.cmd = nil
		w.client = nil
		w.running = false
		if err != nil {
			w.lastErr = fmt.Sprintf("deezer-downloader exited: %v", err)
		} else {
			w.lastErr = "deezer-downloader exited"
		}
		log.Printf("deezer subworker: %s", w.lastErr)
	}()
}

func (w *Worker) markUnavailable(reason string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.running = false
	w.client = nil
	if reason != "" {
		w.lastErr = reason
	}
}

func (w *Worker) stopProcess() {
	w.restartMu.Lock()
	defer w.restartMu.Unlock()
	w.stopProcessLocked()
}

func (w *Worker) stopProcessLocked() {
	w.mu.Lock()
	cmd := w.cmd
	w.cmd = nil
	w.client = nil
	w.running = false
	w.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}
}

func (w *Worker) pollQueue() {
	ticker := time.NewTicker(healthCheckInterval)
	defer ticker.Stop()

	active := false

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			if err := w.ensureReady(w.ctx); err != nil {
				continue
			}

			client, err := w.Client()
			if err != nil {
				continue
			}
			jobs, err := client.Jobs(w.ctx)
			if err != nil {
				w.markUnavailable(fmt.Sprintf("deezer queue check failed: %v", err))
				continue
			}

			busy := false
			for _, job := range jobs {
				switch job.State {
				case "waiting", "active":
					busy = true
				}
			}

			if active && !busy {
				w.organizeAndScan()
			}
			active = busy
		}
	}
}

func (w *Worker) organizeAndScan() {
	moved, err := OrganizeAlbumDownloads(w.settings.MusicDir)
	if err != nil {
		log.Printf("deezer organize albums: %v", err)
	}
	if moved && w.scanner != nil {
		w.scanner.Trigger()
	}
}
