import { useEffect, useState } from "react";
import { artworkUrl } from "./api";

/**
 * Pulls a handful of colours out of a cover so the Now Playing wash is lit by
 * the record itself. The server's dominant colour is one flat tone; a sleeve
 * usually has two or three that play off each other, which is what makes the
 * backdrop feel like it belongs to the artwork.
 */

/** How many colours the backdrop asks for. */
const SWATCHES = 3;

/** Covers are sampled small: we want the broad strokes, not the detail. */
const SAMPLE_SIZE = 28;

/** Hue buckets, wide enough that a gradient counts as one colour. */
const BUCKETS = 18;

/** Below this the pixel is grey and tells us nothing about the palette. */
const MIN_SATURATION = 0.14;

/** Kept clear of both ends so every wash reads on dark and light canvases. */
const WASH_LIGHTNESS: [number, number] = [0.42, 0.64];
const WASH_SATURATION: [number, number] = [0.5, 0.92];

interface Hsl {
  h: number;
  s: number;
  l: number;
}

const cache = new Map<string, string[]>();
const inFlight = new Map<string, Promise<string[]>>();

function rgbToHsl(r: number, g: number, b: number): Hsl {
  const red = r / 255;
  const green = g / 255;
  const blue = b / 255;
  const max = Math.max(red, green, blue);
  const min = Math.min(red, green, blue);
  const l = (max + min) / 2;
  const delta = max - min;

  if (delta === 0) return { h: 0, s: 0, l };

  const s = delta / (1 - Math.abs(2 * l - 1));
  let h: number;
  if (max === red) h = ((green - blue) / delta) % 6;
  else if (max === green) h = (blue - red) / delta + 2;
  else h = (red - green) / delta + 4;

  return { h: (h * 60 + 360) % 360, s, l };
}

function hexToHsl(hex: string): Hsl | null {
  const match = /^#?([\da-f]{6})$/i.exec(hex.trim());
  if (!match) return null;
  const value = Number.parseInt(match[1], 16);
  return rgbToHsl((value >> 16) & 255, (value >> 8) & 255, value & 255);
}

function clamp(value: number, [min, max]: [number, number]): number {
  return Math.min(Math.max(value, min), max);
}

function wash({ h, s, l }: Hsl): string {
  const saturation = clamp(Math.max(s, 0.35), WASH_SATURATION);
  const lightness = clamp(l, WASH_LIGHTNESS);
  return `hsl(${Math.round(h)} ${Math.round(saturation * 100)}% ${Math.round(lightness * 100)}%)`;
}

/** Rounds a palette out when a cover only really has one colour in it. */
function spread(seed: Hsl, count: number): string[] {
  return Array.from({ length: count }, (_, i) =>
    wash({
      h: (seed.h + i * 42) % 360,
      s: seed.s * (1 - i * 0.12),
      l: seed.l + (i % 2 === 0 ? 0.04 : -0.06) * i,
    }),
  );
}

export function fallbackPalette(color: string | undefined): string[] {
  const seed = color ? hexToHsl(color) : null;
  return spread(seed ?? { h: 250, s: 0.35, l: 0.5 }, SWATCHES);
}

function paletteFrom(pixels: Uint8ClampedArray): string[] {
  const weights = new Float64Array(BUCKETS);
  const sums = Array.from({ length: BUCKETS }, () => ({ h: 0, s: 0, l: 0 }));

  for (let i = 0; i < pixels.length; i += 4) {
    if (pixels[i + 3] < 128) continue;
    const hsl = rgbToHsl(pixels[i], pixels[i + 1], pixels[i + 2]);
    if (hsl.s < MIN_SATURATION || hsl.l < 0.06 || hsl.l > 0.95) continue;

    // Vivid mid-tones say more about a cover than washed-out edges do.
    const weight = hsl.s * (1 - Math.abs(hsl.l - 0.5));
    const bucket = Math.floor((hsl.h / 360) * BUCKETS) % BUCKETS;
    weights[bucket] += weight;
    sums[bucket].h += hsl.h * weight;
    sums[bucket].s += hsl.s * weight;
    sums[bucket].l += hsl.l * weight;
  }

  const ranked = Array.from(weights, (weight, bucket) => ({ weight, bucket }))
    .filter((entry) => entry.weight > 0)
    .sort((a, b) => b.weight - a.weight);

  const picked: Hsl[] = [];
  for (const { bucket } of ranked) {
    if (picked.length === SWATCHES) break;
    // Neighbouring buckets are the same colour twice; keep the blobs distinct.
    const tooClose = picked.some((seen) => {
      const gap = Math.abs(seen.h - (sums[bucket].h / weights[bucket]));
      return Math.min(gap, 360 - gap) < 25;
    });
    if (tooClose) continue;
    picked.push({
      h: sums[bucket].h / weights[bucket],
      s: sums[bucket].s / weights[bucket],
      l: sums[bucket].l / weights[bucket],
    });
  }

  if (picked.length === 0) return [];
  if (picked.length < SWATCHES) return spread(picked[0], SWATCHES);
  return picked.map(wash);
}

function extract(url: string): Promise<string[]> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.decoding = "async";
    image.onload = () => {
      const canvas = document.createElement("canvas");
      canvas.width = SAMPLE_SIZE;
      canvas.height = SAMPLE_SIZE;
      const context = canvas.getContext("2d", { willReadFrequently: true });
      if (!context) {
        reject(new Error("no 2d context"));
        return;
      }
      context.drawImage(image, 0, 0, SAMPLE_SIZE, SAMPLE_SIZE);
      resolve(paletteFrom(context.getImageData(0, 0, SAMPLE_SIZE, SAMPLE_SIZE).data));
    };
    image.onerror = () => reject(new Error("artwork failed to load"));
    image.src = url;
  });
}

/**
 * Colours for the current cover, falling back to the server's dominant tone.
 * The previous palette stays on screen until the next one is ready, so a track
 * change crossfades rather than blinking through a default.
 */
export function useArtworkPalette(artworkId: string | undefined, color: string | undefined): string[] {
  const [palette, setPalette] = useState<string[]>(() =>
    artworkId ? (cache.get(artworkId) ?? fallbackPalette(color)) : fallbackPalette(color),
  );

  useEffect(() => {
    const url = artworkId ? artworkUrl(artworkId, 300) : undefined;
    if (!artworkId || !url) {
      setPalette(fallbackPalette(color));
      return;
    }

    const cached = cache.get(artworkId);
    if (cached) {
      setPalette(cached);
      return;
    }

    let cancelled = false;
    const work =
      inFlight.get(artworkId) ??
      extract(url)
        .then((colors) => {
          const resolved = colors.length > 0 ? colors : fallbackPalette(color);
          cache.set(artworkId, resolved);
          return resolved;
        })
        .finally(() => inFlight.delete(artworkId));
    inFlight.set(artworkId, work);

    void work
      .then((colors) => {
        if (!cancelled) setPalette(colors);
      })
      .catch(() => {
        if (!cancelled) setPalette(fallbackPalette(color));
      });

    return () => {
      cancelled = true;
    };
  }, [artworkId, color]);

  return palette;
}
