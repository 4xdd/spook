/**
 * A six-band graphic equaliser matching the band centres Spotify uses, so its
 * preset curves transfer over directly.
 */
export interface Band {
  frequency: number;
  /** Short axis label, e.g. "2.4K". */
  label: string;
  type: BiquadFilterType;
  q: number;
}

export const EQ_BANDS: readonly Band[] = [
  // The outer bands are shelves: a peak at 60Hz leaves the sub-bass untouched,
  // and a peak at 15K leaves the very top of the spectrum behind.
  { frequency: 60, label: "60", type: "lowshelf", q: 0.7 },
  { frequency: 150, label: "150", type: "peaking", q: 1 },
  { frequency: 400, label: "400", type: "peaking", q: 1 },
  { frequency: 1000, label: "1K", type: "peaking", q: 1 },
  { frequency: 2400, label: "2.4K", type: "peaking", q: 1 },
  { frequency: 15000, label: "15K", type: "highshelf", q: 0.7 },
];

export const EQ_MIN_DB = -12;
export const EQ_MAX_DB = 12;

export const FLAT_GAINS: readonly number[] = EQ_BANDS.map(() => 0);

export interface Preset {
  name: string;
  gains: readonly number[];
}

/** Named curves, in Spotify's order. `Custom` is implicit and never listed. */
export const EQ_PRESETS: readonly Preset[] = [
  { name: "Flat", gains: [0, 0, 0, 0, 0, 0] },
  { name: "Acoustic", gains: [5, 4.5, 3, 1.5, 3.5, 3] },
  { name: "Bass booster", gains: [8, 6, 3, 0.5, 0, 0] },
  { name: "Bass reducer", gains: [-8, -6, -3, -0.5, 0, 0] },
  { name: "Classical", gains: [5, 4, 3, -1.5, -1, 3] },
  { name: "Dance", gains: [6.5, 5, 1.5, 2, 3, 1.5] },
  { name: "Deep", gains: [7, 5, 2.5, 1, -2, -6] },
  { name: "Electronic", gains: [6, 4, 0, 1.5, 2.5, 5] },
  { name: "Hip-hop", gains: [7, 5.5, 1.5, 2, 1, 3] },
  { name: "Jazz", gains: [5, 3.5, 1, 2, 3.5, 4] },
  { name: "Latin", gains: [6, 3.5, 0, 0, 3, 6] },
  { name: "Loudness", gains: [8, 6, 0, 1.5, -3, 6] },
  { name: "Lounge", gains: [-3, -1, 2, 4, 2, -1] },
  { name: "Piano", gains: [3, 2, 0.5, 2.5, 4, 3] },
  { name: "Pop", gains: [-2, -1, 3, 4, 2, -1.5] },
  { name: "R&B", gains: [7, 6, 2.5, -1.5, 2, 4] },
  { name: "Rock", gains: [6, 4.5, -0.5, -1, 2.5, 5] },
  { name: "Small speakers", gains: [8, 6, 3, 1, 0, -1] },
  { name: "Spoken word", gains: [-4, -1, 2.5, 5, 4.5, 1] },
  { name: "Treble booster", gains: [0, 0, 0, 2, 5, 8] },
  { name: "Treble reducer", gains: [0, 0, 0, -2, -5, -8] },
  { name: "Vocal booster", gains: [-3, -2, 3, 5, 4, -1.5] },
];

export const CUSTOM_PRESET = "Custom";

export interface EqSettings {
  enabled: boolean;
  /** A preset name, or `Custom` once a band has been dragged. */
  preset: string;
  gains: number[];
}

export function defaultEqSettings(): EqSettings {
  return { enabled: false, preset: "Flat", gains: FLAT_GAINS.slice() };
}

export function clampGain(db: number): number {
  // A tenth of a dB is finer than anyone can hear, and keeps readouts short.
  return Math.round(Math.min(Math.max(db, EQ_MIN_DB), EQ_MAX_DB) * 10) / 10;
}

/** Names the curve if it matches a preset, so hand-tuning reveals itself. */
export function presetFor(gains: readonly number[]): string {
  const match = EQ_PRESETS.find((preset) =>
    preset.gains.every((gain, index) => Math.abs(gain - (gains[index] ?? 0)) < 0.05),
  );
  return match?.name ?? CUSTOM_PRESET;
}

export function formatGain(db: number): string {
  const rounded = Math.round(db * 10) / 10;
  const value = Number.isInteger(rounded) ? rounded.toFixed(0) : rounded.toFixed(1);
  return rounded > 0 ? `+${value}` : value;
}

const STORAGE_KEY = "spook.eq.v1";

export function readEqSettings(): EqSettings {
  const fallback = defaultEqSettings();
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;

    const parsed = JSON.parse(raw) as Partial<EqSettings>;
    const gains = EQ_BANDS.map((_band, index) => {
      const gain = parsed.gains?.[index];
      return typeof gain === "number" && Number.isFinite(gain) ? clampGain(gain) : 0;
    });
    return {
      enabled: parsed.enabled === true,
      // Gains are authoritative; a stale label would leave the listbox blank.
      preset: presetFor(gains),
      gains,
    };
  } catch {
    return fallback;
  }
}

export function persistEqSettings(settings: EqSettings) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // Storage can be full or blocked; playback should not care.
  }
}
