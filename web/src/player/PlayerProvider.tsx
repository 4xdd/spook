import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import type { Track } from "@/lib/api";
import { artworkUrl, streamUrl } from "@/lib/api";
import { equalizer } from "./equalizer";
import type { PlayerSnapshot, Progress, QueueContext, RepeatMode } from "./types";

interface PlayerActions {
  /** Replaces the queue and starts playing at the given position. */
  play(tracks: Track[], startIndex?: number, context?: QueueContext): void;
  playShuffled(tracks: Track[], context?: QueueContext): void;
  toggle(): void;
  pause(): void;
  next(): void;
  previous(): void;
  jumpTo(index: number): void;
  playNext(tracks: Track[]): void;
  playLater(tracks: Track[]): void;
  removeAt(index: number): void;
  clearQueue(): void;
  seek(seconds: number): void;
  setVolume(volume: number): void;
  toggleMute(): void;
  toggleShuffle(): void;
  cycleRepeat(): void;
}

type PlayerValue = PlayerSnapshot & PlayerActions;

const PlayerContext = createContext<PlayerValue | null>(null);

const STORAGE_KEY = "spook.player.v1";

interface PersistedState {
  queue: Track[];
  index: number;
  position: number;
  volume: number;
  muted: boolean;
  shuffle: boolean;
  repeat: RepeatMode;
  context: QueueContext | null;
}

function readPersisted(): Partial<PersistedState> {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as Partial<PersistedState>) : {};
  } catch {
    return {};
  }
}

function shuffled<T>(items: T[]): T[] {
  const copy = items.slice();
  for (let i = copy.length - 1; i > 0; i -= 1) {
    const j = Math.floor(Math.random() * (i + 1));
    [copy[i], copy[j]] = [copy[j], copy[i]];
  }
  return copy;
}

export function PlayerProvider({ children }: { children: ReactNode }) {
  const persisted = useRef(readPersisted()).current;

  const audioRef = useRef<HTMLAudioElement | null>(null);
  const preloadRef = useRef<HTMLAudioElement | null>(null);
  /** Queue order before shuffling, so turning shuffle off restores it. */
  const unshuffledRef = useRef<Track[] | null>(null);

  const [queue, setQueue] = useState<Track[]>(persisted.queue ?? []);
  const [index, setIndex] = useState(persisted.index ?? 0);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [shuffle, setShuffle] = useState(persisted.shuffle ?? false);
  const [repeat, setRepeat] = useState<RepeatMode>(persisted.repeat ?? "off");
  const [volume, setVolumeState] = useState(persisted.volume ?? 1);
  const [muted, setMuted] = useState(persisted.muted ?? false);
  const [context, setContext] = useState<QueueContext | null>(persisted.context ?? null);
  const [error, setError] = useState<string | null>(null);

  const current = queue[index] ?? null;

  // Progress lives outside React state: at 60fps it would re-render the app.
  const progressRef = useRef<Progress>({ currentTime: 0, duration: 0 });
  const subscribersRef = useRef(new Set<(progress: Progress) => void>());

  const publishProgress = useCallback((progress: Progress) => {
    progressRef.current = progress;
    subscribersRef.current.forEach((notify) => notify(progress));
  }, []);

  const subscribeProgress = useCallback((notify: (progress: Progress) => void) => {
    subscribersRef.current.add(notify);
    notify(progressRef.current);
    return () => {
      subscribersRef.current.delete(notify);
    };
  }, []);

  if (audioRef.current === null && typeof Audio !== "undefined") {
    const audio = new Audio();
    audio.preload = "metadata";
    audio.volume = persisted.volume ?? 1;
    audio.muted = persisted.muted ?? false;
    audioRef.current = audio;
    equalizer.attach(audio);
  }

  // A rAF loop gives the scrubber continuous motion; timeupdate alone is choppy.
  useEffect(() => {
    const audio = audioRef.current;
    if (!audio || !isPlaying) return;

    let frame = 0;
    const tick = () => {
      publishProgress({
        currentTime: audio.currentTime,
        duration: Number.isFinite(audio.duration) ? audio.duration : 0,
      });
      frame = requestAnimationFrame(tick);
    };
    frame = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(frame);
  }, [isPlaying, publishProgress]);

  const advance = useCallback(
    (direction: 1 | -1, auto: boolean) => {
      setIndex((position) => {
        if (queue.length === 0) return position;

        if (auto && repeat === "one") return position;

        const nextPosition = position + direction;
        if (nextPosition >= queue.length) {
          if (repeat === "all") return 0;
          if (auto) {
            setIsPlaying(false);
            return position;
          }
          return position;
        }
        if (nextPosition < 0) return 0;
        return nextPosition;
      });
    },
    [queue.length, repeat],
  );

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    const onEnded = () => {
      if (repeat === "one") {
        audio.currentTime = 0;
        void audio.play();
        return;
      }
      if (index + 1 >= queue.length && repeat !== "all") {
        setIsPlaying(false);
        publishProgress({ currentTime: 0, duration: progressRef.current.duration });
        return;
      }
      advance(1, true);
    };

    const onLoaded = () => {
      setIsLoading(false);
      publishProgress({
        currentTime: audio.currentTime,
        duration: Number.isFinite(audio.duration) ? audio.duration : 0,
      });
    };
    const onPlay = () => setIsPlaying(true);
    const onPause = () => setIsPlaying(false);
    const onWaiting = () => setIsLoading(true);
    const onPlaying = () => setIsLoading(false);
    const onTimeUpdate = () => {
      if (!isPlaying) {
        publishProgress({
          currentTime: audio.currentTime,
          duration: Number.isFinite(audio.duration) ? audio.duration : 0,
        });
      }
    };
    const onError = () => {
      setIsLoading(false);
      setIsPlaying(false);
      setError(current ? `Could not play "${current.title}"` : "Playback failed");
    };

    audio.addEventListener("ended", onEnded);
    audio.addEventListener("loadedmetadata", onLoaded);
    audio.addEventListener("play", onPlay);
    audio.addEventListener("pause", onPause);
    audio.addEventListener("waiting", onWaiting);
    audio.addEventListener("playing", onPlaying);
    audio.addEventListener("timeupdate", onTimeUpdate);
    audio.addEventListener("error", onError);

    return () => {
      audio.removeEventListener("ended", onEnded);
      audio.removeEventListener("loadedmetadata", onLoaded);
      audio.removeEventListener("play", onPlay);
      audio.removeEventListener("pause", onPause);
      audio.removeEventListener("waiting", onWaiting);
      audio.removeEventListener("playing", onPlaying);
      audio.removeEventListener("timeupdate", onTimeUpdate);
      audio.removeEventListener("error", onError);
    };
  }, [advance, current, index, isPlaying, publishProgress, queue.length, repeat]);

  /** Swaps the source when the track changes, leaving playback state alone. */
  const loadedIdRef = useRef<string | null>(null);
  const shouldAutoplayRef = useRef(false);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    if (!current) {
      audio.removeAttribute("src");
      loadedIdRef.current = null;
      return;
    }
    if (loadedIdRef.current === current.id) return;

    loadedIdRef.current = current.id;
    setError(null);
    setIsLoading(true);
    audio.src = streamUrl(current.streamUrl);
    audio.load();
    publishProgress({ currentTime: 0, duration: (current.durationMs ?? 0) / 1000 });

    if (shouldAutoplayRef.current) {
      void audio.play().catch(() => setIsPlaying(false));
    }
  }, [current, publishProgress]);

  // Warm the next track so a track change does not stall on the first bytes.
  useEffect(() => {
    const upcoming = queue[index + 1];
    if (!upcoming) {
      preloadRef.current = null;
      return;
    }
    const preload = new Audio();
    preload.preload = "auto";
    preload.src = streamUrl(upcoming.streamUrl);
    preloadRef.current = preload;
    return () => {
      preload.removeAttribute("src");
    };
  }, [index, queue]);

  useEffect(() => {
    const audio = audioRef.current;
    if (audio) {
      audio.volume = volume;
      audio.muted = muted;
    }
  }, [muted, volume]);

  // Persist enough to restore the session, but never auto-resume playback.
  useEffect(() => {
    const state: PersistedState = {
      queue,
      index,
      position: progressRef.current.currentTime,
      volume,
      muted,
      shuffle,
      repeat,
      context,
    };
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch {
      // Storage can be full or blocked; playback should not care.
    }
  }, [context, index, muted, queue, repeat, shuffle, volume]);

  const play = useCallback((tracks: Track[], startIndex = 0, queueContext?: QueueContext) => {
    if (tracks.length === 0) return;
    shouldAutoplayRef.current = true;
    unshuffledRef.current = null;
    setShuffle(false);
    setQueue(tracks);
    setIndex(Math.min(Math.max(startIndex, 0), tracks.length - 1));
    setContext(queueContext ?? null);

    // Same track re-selected: restart it rather than sit paused at the end.
    const audio = audioRef.current;
    if (audio && loadedIdRef.current === tracks[startIndex]?.id) {
      audio.currentTime = 0;
      void audio.play();
    }
  }, []);

  const playShuffled = useCallback(
    (tracks: Track[], queueContext?: QueueContext) => {
      if (tracks.length === 0) return;
      unshuffledRef.current = tracks;
      shouldAutoplayRef.current = true;
      setShuffle(true);
      setQueue(shuffled(tracks));
      setIndex(0);
      setContext(queueContext ?? null);
      const audio = audioRef.current;
      if (audio) audio.currentTime = 0;
    },
    [],
  );

  const toggle = useCallback(() => {
    const audio = audioRef.current;
    if (!audio || !current) return;
    shouldAutoplayRef.current = true;
    if (audio.paused) {
      void audio.play().catch(() => setIsPlaying(false));
    } else {
      audio.pause();
    }
  }, [current]);

  const pause = useCallback(() => {
    audioRef.current?.pause();
  }, []);

  const next = useCallback(() => {
    shouldAutoplayRef.current = true;
    advance(1, false);
  }, [advance]);

  const previous = useCallback(() => {
    const audio = audioRef.current;
    // Matches every music player: restart the track unless you are near its start.
    if (audio && audio.currentTime > 3) {
      audio.currentTime = 0;
      publishProgress({ currentTime: 0, duration: progressRef.current.duration });
      return;
    }
    shouldAutoplayRef.current = true;
    advance(-1, false);
  }, [advance, publishProgress]);

  const jumpTo = useCallback((position: number) => {
    shouldAutoplayRef.current = true;
    setIndex(position);
  }, []);

  const playNext = useCallback(
    (tracks: Track[]) => {
      if (tracks.length === 0) return;
      setQueue((existing) => {
        if (existing.length === 0) {
          shouldAutoplayRef.current = true;
          return tracks;
        }
        const copy = existing.slice();
        copy.splice(index + 1, 0, ...tracks);
        return copy;
      });
    },
    [index],
  );

  const playLater = useCallback((tracks: Track[]) => {
    if (tracks.length === 0) return;
    setQueue((existing) => {
      if (existing.length === 0) shouldAutoplayRef.current = true;
      return [...existing, ...tracks];
    });
  }, []);

  const removeAt = useCallback(
    (position: number) => {
      setQueue((existing) => existing.filter((_, i) => i !== position));
      if (position < index) setIndex((value) => value - 1);
    },
    [index],
  );

  const clearQueue = useCallback(() => {
    audioRef.current?.pause();
    setQueue([]);
    setIndex(0);
    setContext(null);
    loadedIdRef.current = null;
    publishProgress({ currentTime: 0, duration: 0 });
  }, [publishProgress]);

  const seek = useCallback(
    (seconds: number) => {
      const audio = audioRef.current;
      if (!audio) return;
      const duration = Number.isFinite(audio.duration) ? audio.duration : progressRef.current.duration;
      const clamped = Math.min(Math.max(seconds, 0), duration || seconds);
      audio.currentTime = clamped;
      publishProgress({ currentTime: clamped, duration });
    },
    [publishProgress],
  );

  const setVolume = useCallback((value: number) => {
    const clamped = Math.min(Math.max(value, 0), 1);
    setVolumeState(clamped);
    if (clamped > 0) setMuted(false);
  }, []);

  const toggleMute = useCallback(() => setMuted((value) => !value), []);

  const toggleShuffle = useCallback(() => {
    setShuffle((enabled) => {
      const playing = queue[index];
      if (!enabled) {
        unshuffledRef.current = queue;
        // Keep the current track playing and shuffle everything around it.
        const rest = shuffled(queue.filter((_, i) => i !== index));
        setQueue(playing ? [playing, ...rest] : rest);
        setIndex(playing ? 0 : 0);
        return true;
      }

      const original = unshuffledRef.current;
      unshuffledRef.current = null;
      if (!original) return false;
      setQueue(original);
      const restored = playing ? original.findIndex((track) => track.id === playing.id) : 0;
      setIndex(restored >= 0 ? restored : 0);
      return false;
    });
  }, [index, queue]);

  const cycleRepeat = useCallback(() => {
    setRepeat((mode) => (mode === "off" ? "all" : mode === "all" ? "one" : "off"));
  }, []);

  // OS media keys, lock screen and notification controls.
  useEffect(() => {
    if (!("mediaSession" in navigator)) return;

    if (!current) {
      navigator.mediaSession.metadata = null;
      navigator.mediaSession.playbackState = "none";
      return;
    }

    const artwork = current.artworkId
      ? ([64, 300, 1000] as const).map((size) => ({
          src: artworkUrl(current.artworkId, size) ?? "",
          sizes: `${size}x${size}`,
          type: "image/jpeg",
        }))
      : [];

    navigator.mediaSession.metadata = new MediaMetadata({
      title: current.title,
      artist: current.artist,
      album: current.album,
      artwork,
    });
    navigator.mediaSession.playbackState = isPlaying ? "playing" : "paused";
  }, [current, isPlaying]);

  useEffect(() => {
    if (!("mediaSession" in navigator)) return;
    const handlers: [MediaSessionAction, MediaSessionActionHandler][] = [
      ["play", () => toggle()],
      ["pause", () => pause()],
      ["nexttrack", () => next()],
      ["previoustrack", () => previous()],
      ["seekto", (details) => details.seekTime !== undefined && seek(details.seekTime)],
      ["seekforward", () => seek(progressRef.current.currentTime + 10)],
      ["seekbackward", () => seek(progressRef.current.currentTime - 10)],
    ];

    for (const [action, handler] of handlers) {
      try {
        navigator.mediaSession.setActionHandler(action, handler);
      } catch {
        // Not every browser supports every action.
      }
    }
    return () => {
      for (const [action] of handlers) {
        try {
          navigator.mediaSession.setActionHandler(action, null);
        } catch {
          // Ignore.
        }
      }
    };
  }, [next, pause, previous, seek, toggle]);

  const value = useMemo<PlayerValue>(
    () => ({
      queue,
      index,
      current,
      isPlaying,
      isLoading,
      shuffle,
      repeat,
      volume,
      muted,
      context,
      error,
      play,
      playShuffled,
      toggle,
      pause,
      next,
      previous,
      jumpTo,
      playNext,
      playLater,
      removeAt,
      clearQueue,
      seek,
      setVolume,
      toggleMute,
      toggleShuffle,
      cycleRepeat,
    }),
    [
      clearQueue,
      context,
      current,
      cycleRepeat,
      error,
      index,
      isLoading,
      isPlaying,
      jumpTo,
      muted,
      next,
      pause,
      play,
      playLater,
      playNext,
      playShuffled,
      previous,
      queue,
      removeAt,
      repeat,
      seek,
      setVolume,
      shuffle,
      toggle,
      toggleMute,
      toggleShuffle,
      volume,
    ],
  );

  return (
    <PlayerContext.Provider value={value}>
      <ProgressContext.Provider value={subscribeProgress}>{children}</ProgressContext.Provider>
    </PlayerContext.Provider>
  );
}

type Subscribe = (notify: (progress: Progress) => void) => () => void;

const ProgressContext = createContext<Subscribe | null>(null);

export function usePlayer(): PlayerValue {
  const value = useContext(PlayerContext);
  if (!value) throw new Error("usePlayer must be used inside PlayerProvider");
  return value;
}

/** Subscribes only the calling component to playback position updates. */
export function useProgress(): Progress {
  const subscribe = useContext(ProgressContext);
  if (!subscribe) throw new Error("useProgress must be used inside PlayerProvider");

  const [progress, setProgress] = useState<Progress>({ currentTime: 0, duration: 0 });
  useEffect(() => subscribe(setProgress), [subscribe]);
  return progress;
}

/**
 * Like useProgress, but re-renders only when the derived value changes, which
 * is what a view keyed to something coarser than the frame rate wants.
 * Pass a stable selector: it doubles as the resubscribe key.
 */
export function useProgressSelector<T>(select: (progress: Progress) => T): T {
  const subscribe = useContext(ProgressContext);
  if (!subscribe) throw new Error("useProgressSelector must be used inside PlayerProvider");

  const [value, setValue] = useState<T>(() => select({ currentTime: 0, duration: 0 }));

  useEffect(
    () =>
      subscribe((progress) => {
        const next = select(progress);
        setValue((previous) => (Object.is(previous, next) ? previous : next));
      }),
    [select, subscribe],
  );

  return value;
}
