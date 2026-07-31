import { useEffect, type RefObject } from "react";
import { equalizer, type AudioLevels } from "./equalizer";

/**
 * How quickly a band follows the music. Rising fast and falling slowly is what
 * makes a hit land as a beat instead of a flicker.
 */
const ATTACK = 0.4;
const RELEASE = 0.06;

/** Quiet passages should still settle to nothing rather than hover. */
const FLOOR = 0.04;

const BANDS = ["low", "mid", "high"] as const;

function shape(level: number): number {
  // Perceived loudness is closer to a curve than to raw bin energy, and the
  // bottom of the range is mostly room noise.
  const above = Math.max(level - FLOOR, 0) / (1 - FLOOR);
  return Math.min(above ** 0.7, 1);
}

/**
 * Writes smoothed per-band levels onto an element as `--pulse-low`, `--pulse-mid`
 * and `--pulse-high`, for CSS to do what it likes with. Kept off React state:
 * this updates every frame.
 */
export function useAudioPulse(target: RefObject<HTMLElement | null>, active: boolean): void {
  useEffect(() => {
    const element = target.current;
    if (!element) return;

    const clear = () => {
      for (const band of BANDS) element.style.setProperty(`--pulse-${band}`, "0");
    };

    if (!active || window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      clear();
      return;
    }

    // Playing already counts as a gesture, so the context can start here.
    if (!equalizer.ensureAnalyser()) {
      clear();
      return;
    }

    const smoothed: AudioLevels = { low: 0, mid: 0, high: 0 };
    let frame = 0;

    const tick = () => {
      const levels = equalizer.levels();
      if (levels) {
        for (const band of BANDS) {
          const next = shape(levels[band]);
          smoothed[band] += (next - smoothed[band]) * (next > smoothed[band] ? ATTACK : RELEASE);
          element.style.setProperty(`--pulse-${band}`, smoothed[band].toFixed(3));
        }
      }
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);

    return () => {
      cancelAnimationFrame(frame);
      clear();
    };
  }, [active, target]);
}
