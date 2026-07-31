// Package auth gates the API behind shared access keys stored on disk.
package auth

import (
	"bufio"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Keys is a set of valid access keys loaded from a local file and/or env.
type Keys struct {
	mu   sync.RWMutex
	set  map[string]struct{}
	path string
}

// Load reads keys from path (one per line) and merges any extras. Blank lines
// and # comments are ignored. When the resulting set is empty, a new key is
// generated, written to path, and returned as created.
func Load(path string, extras []string) (*Keys, string, error) {
	k := &Keys{
		set:  make(map[string]struct{}),
		path: path,
	}

	if path != "" {
		if err := k.readFile(path); err != nil && !os.IsNotExist(err) {
			return nil, "", err
		}
	}
	for _, extra := range extras {
		k.add(extra)
	}

	var created string
	if len(k.set) == 0 {
		key, err := generateKey()
		if err != nil {
			return nil, "", err
		}
		if path == "" {
			return nil, "", fmt.Errorf("no access keys configured and no access-keys file path")
		}
		if err := writeInitial(path, key); err != nil {
			return nil, "", err
		}
		k.add(key)
		created = key
	}

	return k, created, nil
}

// Valid reports whether candidate matches a configured key.
func (k *Keys) Valid(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || k == nil {
		return false
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	for key := range k.set {
		if secureEqual(key, candidate) {
			return true
		}
	}
	return false
}

// Count returns how many keys are configured.
func (k *Keys) Count() int {
	if k == nil {
		return 0
	}
	k.mu.RLock()
	defer k.mu.RUnlock()
	return len(k.set)
}

// Path returns the on-disk keys file, if any.
func (k *Keys) Path() string {
	if k == nil {
		return ""
	}
	return k.path
}

func (k *Keys) add(raw string) {
	key := strings.TrimSpace(raw)
	if key == "" {
		return
	}
	k.mu.Lock()
	k.set[key] = struct{}{}
	k.mu.Unlock()
}

func (k *Keys) readFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k.add(line)
	}
	return scanner.Err()
}

func writeInitial(path string, key string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body := "# Spook access keys — one per line.\n" +
		"# Anyone with a key can use the library API and player.\n" +
		key + "\n"
	return os.WriteFile(path, []byte(body), 0o600)
}

func generateKey() (string, error) {
	var buf [24]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}

func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// SplitKeys parses a comma-separated access-key list from the environment.
func SplitKeys(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if key := strings.TrimSpace(part); key != "" {
			out = append(out, key)
		}
	}
	return out
}
