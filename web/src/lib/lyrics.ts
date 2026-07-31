import type { LyricLine } from "./api";

/** Span given to the last line of a song, which has nothing following it. */
const LAST_LINE_FALLBACK_MS = 4000;

/** Nothing reads as a word below this, however short it is written. */
const MIN_WORD_MS = 160;

/** Rough speaking pace, used to share a line's span out between its words. */
const MS_PER_CHAR = 62;

/**
 * A word left ringing for longer than this is a held note rather than a slow
 * delivery, and is worth drawing differently.
 */
export const HELD_WORD_MS = 800;

export interface TimedWord {
  text: string;
  startMs: number;
  durationMs: number;
}

/** The last line whose timestamp has passed, or -1 before the first one. */
export function activeLineIndex(lines: LyricLine[], timeMs: number): number {
  let active = -1;
  for (let i = 0; i < lines.length; i += 1) {
    if (lines[i].timeMs > timeMs) break;
    active = i;
  }
  return active;
}

/**
 * Spread a line's words across the gap until the next timestamp. Synced LRC is
 * almost always line-timed, so this is the karaoke approximation we can make:
 * words take about as long as they are to say, and whatever time is left over
 * lands on the last word, which is where a singer actually holds a note.
 */
export function wordsForLine(line: LyricLine, nextTimeMs: number | undefined): TimedWord[] {
  const tokens = line.text.trim().split(/\s+/).filter(Boolean);
  if (tokens.length === 0) return [];

  const start = line.timeMs;
  const end = nextTimeMs ?? start + LAST_LINE_FALLBACK_MS;
  const span = Math.max(end - start, tokens.length * MIN_WORD_MS);

  const natural = tokens.map((token) => Math.max(MIN_WORD_MS, token.length * MS_PER_CHAR));
  const spoken = natural.reduce((sum, ms) => sum + ms, 0);
  // Too little room: everything speeds up. Too much: the tail carries it.
  const squeeze = span < spoken ? span / spoken : 1;
  const durations = natural.map((ms) => ms * squeeze);
  durations[durations.length - 1] += Math.max(span - spoken, 0);

  let at = start;
  return tokens.map((text, i) => {
    const word = { text, startMs: at, durationMs: durations[i] };
    at += durations[i];
    return word;
  });
}

/** Index of the word currently being sung within a timed line, or -1. */
export function activeWordIndex(words: TimedWord[], timeMs: number): number {
  let active = -1;
  for (let i = 0; i < words.length; i += 1) {
    if (words[i].startMs > timeMs) break;
    active = i;
  }
  return active;
}
