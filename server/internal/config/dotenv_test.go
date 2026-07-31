package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDotEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := `# comment
MUSIC_DIR=/tmp/from-dotenv
DEEZER_ARL=abc123
QUOTED="hello world"
INLINE=value  # trailing comment
ALREADY=from-file
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("ALREADY", "from-env")

	if err := parseDotEnvFile(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("DEEZER_ARL"); got != "abc123" {
		t.Fatalf("DEEZER_ARL = %q", got)
	}
	if got := os.Getenv("MUSIC_DIR"); got != "/tmp/from-dotenv" {
		t.Fatalf("MUSIC_DIR = %q", got)
	}
	if got := os.Getenv("INLINE"); got != "value" {
		t.Fatalf("INLINE = %q, want value without comment", got)
	}
	if got := os.Getenv("QUOTED"); got != "hello world" {
		t.Fatalf("QUOTED = %q", got)
	}
	if got := os.Getenv("ALREADY"); got != "from-env" {
		t.Fatalf("ALREADY should not be overwritten, got %q", got)
	}
}
