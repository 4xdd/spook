import type { Track } from "@/lib/api";

export type RepeatMode = "off" | "all" | "one";

export interface QueueContext {
  /** Where the queue came from, shown above the queue list. */
  label: string;
  id: string;
}

export interface PlayerSnapshot {
  queue: Track[];
  index: number;
  current: Track | null;
  isPlaying: boolean;
  isLoading: boolean;
  shuffle: boolean;
  repeat: RepeatMode;
  volume: number;
  muted: boolean;
  context: QueueContext | null;
  error: string | null;
}

export interface Progress {
  currentTime: number;
  duration: number;
}
