// Package artwork resolves, deduplicates and caches album covers.
//
// Covers come from embedded picture frames first and a folder image second.
// Identical bytes anywhere in the library collapse to a single cache entry, so
// an album's cover is decoded and resized once no matter how many tracks carry it.
package artwork

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// Sizes are the cached square variants, in pixels.
var Sizes = []int{64, 300, 1000}

const (
	SourceEmbedded = "embedded"
	SourceFolder   = "folder"

	maxImageBytes = 32 << 20
)

type Record struct {
	ID     string
	Mime   string
	Width  int
	Height int
	Color  string
	IsDark bool
	Source string
}

type Cache struct {
	dir string

	mu    sync.Mutex
	known map[string]Record
	dirs  map[string]string // directory -> artwork id, "" when the folder has none
}

func NewCache(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create art cache: %w", err)
	}
	return &Cache{
		dir:   dir,
		known: make(map[string]Record),
		dirs:  make(map[string]string),
	}, nil
}

// Seed registers artwork already recorded in the index so a rescan does not
// decode images it has seen before.
func (c *Cache) Seed(records []Record) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, record := range records {
		if c.filesExist(record.ID) {
			c.known[record.ID] = record
		}
	}
}

// Reset drops the per-directory memo between scans.
func (c *Cache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dirs = make(map[string]string)
}

func (c *Cache) Records() []Record {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Record, 0, len(c.known))
	for _, record := range c.known {
		out = append(out, record)
	}
	return out
}

// Put decodes, resizes and caches an image, returning the shared record.
func (c *Cache) Put(data []byte, source string) (Record, error) {
	if len(data) == 0 || len(data) > maxImageBytes {
		return Record{}, errors.New("artwork: empty or oversized image")
	}

	sum := sha256.Sum256(data)
	id := hex.EncodeToString(sum[:8])

	c.mu.Lock()
	if record, ok := c.known[id]; ok {
		c.mu.Unlock()
		return record, nil
	}
	c.mu.Unlock()

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return Record{}, fmt.Errorf("artwork: decode: %w", err)
	}

	bounds := img.Bounds()
	color, isDark := dominant(img)
	record := Record{
		ID:     id,
		Mime:   mimeFor(format),
		Width:  bounds.Dx(),
		Height: bounds.Dy(),
		Color:  color,
		IsDark: isDark,
		Source: source,
	}

	if err := c.writeVariants(id, img); err != nil {
		return Record{}, err
	}

	c.mu.Lock()
	c.known[id] = record
	c.mu.Unlock()

	return record, nil
}

// PutFolderImage resolves and caches the cover image sitting next to a track,
// looking one directory up so multi-disc folders inherit the album cover.
func (c *Cache) PutFolderImage(dir string) (Record, bool) {
	c.mu.Lock()
	if id, ok := c.dirs[dir]; ok {
		record, known := c.known[id]
		c.mu.Unlock()
		return record, known
	}
	c.mu.Unlock()

	record, ok := c.resolveFolder(dir)

	c.mu.Lock()
	c.dirs[dir] = record.ID
	c.mu.Unlock()

	return record, ok
}

func (c *Cache) resolveFolder(dir string) (Record, bool) {
	candidates := []string{dir}
	if parent := filepath.Dir(dir); parent != dir && parent != "." && parent != string(filepath.Separator) {
		candidates = append(candidates, parent)
	}

	for _, candidate := range candidates {
		path, ok := findCoverFile(candidate)
		if !ok {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		record, err := c.Put(data, SourceFolder)
		if err != nil {
			continue
		}
		return record, true
	}
	return Record{}, false
}

var coverNames = []string{"cover", "folder", "front", "album", "albumart", "albumartsmall", "artwork", "art", "scan", "thumb"}

var imageExtensions = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true, ".gif": true, ".bmp": true,
}

// findCoverFile prefers conventional cover filenames, then falls back to the
// only image in the directory when there is exactly one.
func findCoverFile(dir string) (string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", false
	}

	var images []string
	best, bestRank := "", len(coverNames)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !imageExtensions[ext] {
			continue
		}
		full := filepath.Join(dir, name)
		images = append(images, full)

		stem := strings.ToLower(strings.TrimSuffix(name, ext))
		for rank, candidate := range coverNames {
			if stem == candidate || strings.HasPrefix(stem, candidate) {
				if rank < bestRank {
					best, bestRank = full, rank
				}
				break
			}
		}
	}

	if best != "" {
		return best, true
	}
	if len(images) == 1 {
		return images[0], true
	}
	return "", false
}

func (c *Cache) writeVariants(id string, img image.Image) error {
	dir := c.variantDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	bounds := img.Bounds()
	longest := bounds.Dx()
	if bounds.Dy() > longest {
		longest = bounds.Dy()
	}

	for _, size := range Sizes {
		target := size
		if longest < size {
			// Never upscale; the variant just mirrors the source.
			target = longest
		}
		scale := float64(target) / float64(longest)
		width := int(float64(bounds.Dx()) * scale)
		height := int(float64(bounds.Dy()) * scale)
		if width < 1 {
			width = 1
		}
		if height < 1 {
			height = 1
		}

		resized := image.NewRGBA(image.Rect(0, 0, width, height))
		// Flatten onto white so transparent PNG covers do not encode as black.
		draw.Draw(resized, resized.Bounds(), image.White, image.Point{}, draw.Src)
		draw.CatmullRom.Scale(resized, resized.Bounds(), img, bounds, draw.Over, nil)

		path := c.Path(id, size)
		file, err := os.Create(path)
		if err != nil {
			return err
		}
		err = jpeg.Encode(file, resized, &jpeg.Options{Quality: 88})
		closeErr := file.Close()
		if err != nil {
			os.Remove(path)
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return nil
}

// Path is the on-disk location of a cached variant.
func (c *Cache) Path(id string, size int) string {
	return filepath.Join(c.variantDir(id), fmt.Sprintf("%s-%d.jpg", id, size))
}

// NearestSize maps a requested width onto a cached variant.
func NearestSize(requested int) int {
	for _, size := range Sizes {
		if requested <= size {
			return size
		}
	}
	return Sizes[len(Sizes)-1]
}

func (c *Cache) variantDir(id string) string {
	shard := id
	if len(shard) > 2 {
		shard = shard[:2]
	}
	return filepath.Join(c.dir, shard)
}

func (c *Cache) filesExist(id string) bool {
	for _, size := range Sizes {
		if _, err := os.Stat(c.Path(id, size)); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return false
			}
			return false
		}
	}
	return true
}

func mimeFor(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
