import { Loader2 } from "lucide-react";
import { Fragment, useCallback, useEffect, useRef } from "react";
import type { LyricLine, Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import { HELD_WORD_MS, activeLineIndex, activeWordIndex, wordsForLine } from "@/lib/lyrics";
import { useLyrics } from "@/lib/queries";
import { usePlayer, useProgressSelector } from "@/player/PlayerProvider";

interface Props {
  track: Track;
  className?: string;
}

/**
 * How long auto-scroll stands down after the reader scrolls for themselves.
 * Long enough to read ahead, short enough that it comes back on its own.
 */
const MANUAL_SCROLL_GRACE_MS = 8000;

/**
 * Where the live line rests in the pane. Above the middle, so most of what is
 * on screen is what is coming rather than what has gone.
 */
const ACTIVE_ANCHOR = 0.36;

const noLines: LyricLine[] = [];

type LineState = "static" | "past" | "active" | "upcoming";

function lineState(index: number, activeIndex: number, synced: boolean): LineState {
  if (!synced) return "static";
  if (index === activeIndex) return "active";
  return index < activeIndex ? "past" : "upcoming";
}

export function Lyrics({ track, className }: Props) {
  const { seek } = usePlayer();
  const { data, isPending, isError, error } = useLyrics(track.id);

  const lines = data?.lines ?? noLines;
  const synced = data?.synced ?? false;

  const syncKey = useProgressSelector(
    useCallback(
      (progress) => {
        if (!synced) return "-1:-1";
        const timeMs = progress.currentTime * 1000;
        const line = activeLineIndex(lines, timeMs);
        if (line < 0) return "-1:-1";
        const current = lines[line];
        if (!current || current.text === "") return `${line}:-1`;
        const word = activeWordIndex(wordsForLine(current, lines[line + 1]?.timeMs), timeMs);
        return `${line}:${word}`;
      },
      [lines, synced],
    ),
  );
  const [activeIndex, wordProgress] = (() => {
    const [line, word] = syncKey.split(":").map(Number);
    return [line, word] as const;
  })();

  const scrollRef = useRef<HTMLDivElement>(null);
  const manualScrollAt = useRef(0);

  // A new track starts at the top rather than wherever the last one ended.
  // Declared first so the follow-along below can take over from a clean slate.
  useEffect(() => {
    manualScrollAt.current = 0;
    scrollRef.current?.scrollTo({ top: 0 });
  }, [track.id]);

  // Follow the song, but yield to a reader who has taken over the scroll.
  useEffect(() => {
    const container = scrollRef.current;
    if (!container || !synced) return;
    if (Date.now() - manualScrollAt.current < MANUAL_SCROLL_GRACE_MS) return;

    // Before the first timestamp there is no active line; park on the opening
    // one, which the clamped scroll leaves sitting at the top of the pane.
    const target = Math.max(activeIndex, 0);
    const line = container.querySelector<HTMLElement>(`[data-line="${target}"]`);
    if (!line) return;

    const smooth = !window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    container.scrollTo({
      top: line.offsetTop - container.clientHeight * ACTIVE_ANCHOR + line.clientHeight / 2,
      behavior: smooth ? "smooth" : "auto",
    });
  }, [activeIndex, synced, lines]);

  const onManualScroll = () => {
    manualScrollAt.current = Date.now();
  };

  if (isPending) {
    return (
      <Shell className={className}>
        <span className="flex items-center gap-2 text-[13px] text-tertiary">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          Loading lyrics…
        </span>
      </Shell>
    );
  }

  if (isError) {
    return (
      <Shell className={className}>
        <p className="text-[13px] text-secondary">{error.message}</p>
      </Shell>
    );
  }

  if (lines.length === 0) {
    return (
      <Shell className={className}>
        <p className="text-[15px] text-tertiary">No lyrics for this track</p>
      </Shell>
    );
  }

  return (
    <div
      ref={scrollRef}
      onWheel={onManualScroll}
      onTouchMove={onManualScroll}
      onKeyDown={onManualScroll}
      tabIndex={0}
      aria-label={`Lyrics for ${track.title}`}
      // Fading both edges stands in for a hard border against the artwork wash.
      className={cn(
        "lyrics-glow-bleed lyric-fade no-scrollbar size-container relative overflow-y-auto overscroll-contain",
        className,
      )}
    >
      {/*
        The song opens flush with the top of the pane; only the tail needs room
        below it, so the closing line can still climb to the anchor.
      */}
      <div className={cn("flex flex-col items-start gap-3 pt-5", synced ? "pb-[70cqh]" : "pb-5")}>
        {lines.map((line, index) => (
          <LyricRow
            key={index}
            index={index}
            line={line}
            nextTimeMs={lines[index + 1]?.timeMs}
            synced={synced}
            state={lineState(index, activeIndex, synced)}
            activeWord={index === activeIndex ? wordProgress : -1}
            onSeek={seek}
          />
        ))}
      </div>
    </div>
  );
}

interface RowProps {
  index: number;
  line: LyricLine;
  nextTimeMs?: number;
  synced: boolean;
  state: LineState;
  activeWord: number;
  onSeek(seconds: number): void;
}

function LyricRow({ index, line, nextTimeMs, synced, state, activeWord, onSeek }: RowProps) {
  const words =
    synced && state === "active" && line.text !== "" ? wordsForLine(line, nextTimeMs) : null;

  // An empty line is a break: it holds its place so the highlight can move off
  // the line before it, but shows nothing.
  if (line.text === "") {
    return <div data-line={index} className="h-3" aria-hidden />;
  }

  const textClass = cn(
    "block text-left transition-[color,opacity] duration-300 ease-out",
    synced
      ? "text-[22px] leading-[1.2] font-semibold tracking-[-0.02em] sm:text-[26px] sm:leading-[1.15]"
      : "text-[19px] leading-snug font-medium tracking-[-0.01em]",
    state === "active" && "text-content",
    state === "past" && "text-content/35",
    state === "upcoming" && "text-content/55",
    state === "static" && "text-content/85",
  );

  const text =
    words && words.length > 0 ? (
      <span className={textClass} aria-current="true">
        {words.map((word, i) => {
          const sung = activeWord >= 0 && i <= activeWord;
          const current = i === activeWord;
          // A word the singer sits on keeps brightening for as long as the note
          // lasts, so a held note looks held instead of merely lit.
          const held = current && word.durationMs >= HELD_WORD_MS;
          return (
            <Fragment key={`${i}-${word.text}`}>
              <span
                className={cn(
                  "lyric-word transition-[opacity,color,text-shadow] duration-200 ease-out",
                  sung ? "text-content opacity-100" : "text-content/40 opacity-70",
                  current && "lyric-word-current",
                  held && "lyric-word-held",
                )}
                style={held ? { animationDuration: `${Math.round(word.durationMs)}ms` } : undefined}
              >
                {word.text}
              </span>
              {i < words.length - 1 ? " " : ""}
            </Fragment>
          );
        })}
      </span>
    ) : (
      <span className={textClass}>{line.text}</span>
    );

  if (!synced) {
    return (
      <p data-line={index} className="px-1">
        {text}
      </p>
    );
  }

  return (
    <button
      type="button"
      data-line={index}
      onClick={() => onSeek(line.timeMs / 1000)}
      aria-current={state === "active"}
      title="Jump to this line"
      className="cursor-pointer rounded-md px-1 py-0.5 text-left transition-transform duration-100 ease-out hover:text-content active:scale-[0.99]"
    >
      {text}
    </button>
  );
}

function Shell({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn("grid place-items-center py-10", className)}>{children}</div>;
}
