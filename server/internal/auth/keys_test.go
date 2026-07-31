package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesKeyWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access_keys")

	keys, created, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if created == "" {
		t.Fatal("expected a newly created key")
	}
	if !keys.Valid(created) {
		t.Fatal("created key should validate")
	}
	if keys.Count() != 1 {
		t.Fatalf("count = %d, want 1", keys.Count())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !containsLine(string(data), created) {
		t.Fatalf("file does not contain created key:\n%s", data)
	}
}

func TestLoadMergesFileAndExtras(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access_keys")
	if err := os.WriteFile(path, []byte("# comment\nfile-key\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	keys, created, err := Load(path, []string{"env-key", "  "})
	if err != nil {
		t.Fatal(err)
	}
	if created != "" {
		t.Fatalf("unexpected created key %q", created)
	}
	if !keys.Valid("file-key") || !keys.Valid("env-key") {
		t.Fatal("both keys should validate")
	}
	if keys.Valid("other") {
		t.Fatal("unknown key should not validate")
	}
}

func TestSplitKeys(t *testing.T) {
	got := SplitKeys(" a ,b, ,c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("SplitKeys = %#v", got)
	}
}

func containsLine(body, line string) bool {
	for _, raw := range splitLines(body) {
		if raw == line {
			return true
		}
	}
	return false
}

func splitLines(body string) []string {
	var out []string
	start := 0
	for i := 0; i < len(body); i++ {
		if body[i] == '\n' {
			out = append(out, body[start:i])
			start = i + 1
		}
	}
	if start < len(body) {
		out = append(out, body[start:])
	}
	return out
}
