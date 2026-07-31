package deezer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deezer-downloader.ini")

	if err := WriteConfig(path, "/home/user/Music", "abc123", "mp3", "/usr/bin/yt-dlp", 5001); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	text := string(body)
	for _, want := range []string{
		"base = /home/user/Music",
		"cookie_arl = abc123",
		"quality = mp3",
		"port = 5001",
		"command = /usr/bin/yt-dlp",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("config missing %q:\n%s", want, text)
		}
	}
}

func TestSettingsConfigured(t *testing.T) {
	if (Settings{URL: "http://127.0.0.1:5000"}).Configured() != true {
		t.Fatal("expected external URL to count as configured")
	}
	if (Settings{ARL: "x"}).Configured() != true {
		t.Fatal("expected ARL to count as configured")
	}
	if (Settings{}).Configured() != false {
		t.Fatal("expected empty settings to be unconfigured")
	}
}

func TestSanitizeARL(t *testing.T) {
	raw := "abc123def   # inline comment with — em dash"
	if got := sanitizeARL(raw); got != "abc123def" {
		t.Fatalf("got %q", got)
	}
}

func TestWriteConfigStripsInlineCommentFromARL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deezer-downloader.ini")
	arl := "deadbeef   # not part of cookie"

	if err := WriteConfig(path, "/music", arl, "mp3", "/usr/bin/yt-dlp", 5001); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "cookie_arl = deadbeef") {
		t.Fatalf("unexpected config:\n%s", body)
	}
	if strings.Contains(string(body), "# not part") {
		t.Fatal("inline comment leaked into config")
	}
}
