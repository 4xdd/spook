package artwork

import (
	"fmt"
	"image"
	"math"
)

// dominant picks a representative colour by bucketing pixels in a coarse RGB
// histogram, weighting each bucket by saturation so a colourful accent beats a
// large flat background, then averaging the winning bucket's pixels.
func dominant(img image.Image) (hex string, isDark bool) {
	bounds := img.Bounds()
	if bounds.Empty() {
		return "#1c1c1e", true
	}

	const buckets = 6
	type accumulator struct {
		r, g, b float64
		count   float64
		weight  float64
	}
	histogram := make(map[int]*accumulator, 256)

	stepX := max(bounds.Dx()/48, 1)
	stepY := max(bounds.Dy()/48, 1)

	var totalPixels float64

	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			r16, g16, b16, a16 := img.At(x, y).RGBA()
			if a16 < 0x8000 {
				continue
			}
			r := float64(r16 >> 8)
			g := float64(g16 >> 8)
			b := float64(b16 >> 8)

			luminance := (0.2126*r + 0.7152*g + 0.0722*b) / 255
			totalPixels++

			maxC := math.Max(r, math.Max(g, b))
			minC := math.Min(r, math.Min(g, b))
			saturation := 0.0
			if maxC > 0 {
				saturation = (maxC - minC) / maxC
			}
			// Near-black and near-white pixels rarely make good accents.
			weight := 0.25 + saturation
			if luminance < 0.08 || luminance > 0.95 {
				weight *= 0.15
			}

			key := int(r)/((256/buckets)+1)*buckets*buckets +
				int(g)/((256/buckets)+1)*buckets +
				int(b)/((256/buckets)+1)

			bucket := histogram[key]
			if bucket == nil {
				bucket = &accumulator{}
				histogram[key] = bucket
			}
			bucket.r += r
			bucket.g += g
			bucket.b += b
			bucket.count++
			bucket.weight += weight
		}
	}

	if totalPixels == 0 {
		return "#1c1c1e", true
	}

	var best *accumulator
	for _, bucket := range histogram {
		if best == nil || bucket.weight > best.weight {
			best = bucket
		}
	}
	if best == nil || best.count == 0 {
		return "#1c1c1e", true
	}

	r := best.r / best.count
	g := best.g / best.count
	b := best.b / best.count

	// isDark describes the colour itself, since that is what the UI tints
	// surfaces with and has to pick readable foreground text against.
	luminance := (0.2126*r + 0.7152*g + 0.0722*b) / 255

	return fmt.Sprintf("#%02x%02x%02x", int(r), int(g), int(b)), luminance < 0.5
}
