import { Loader2 } from "lucide-react";
import { Fragment, useCallback, useEffect, useRef } from "react";
import type { LyricLine, Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import {
  HELD_WORD_MS,
  activeLineIndex,
  activeWordIndex,
  groupLyricLines,
  type LyricDisplayRow,
  wordsForLine,
} from "@/lib/lyrics";
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
  const rows = groupLyricLines(lines);
  const synced = data?.synced ?? false;

  const syncKey = useProgressSelector(
    useCallback(
      (progress) => {
        if (!synced) return "-1:-1";
        const timeMs = progress.currentTime * 1000;
        const line = activeLineIndex(rows, timeMs);
        if (line < 0) return `${timeMs}:-1:-1`;
        const current = rows[line];
        if (!current || current.text === "") return `${timeMs}:${line}:-1`;
        const word = activeWordIndex(
          wordsForLine({ timeMs: current.timeMs, text: current.text }, rows[line + 1]?.timeMs),
          timeMs,
        );
        return `${timeMs}:${line}:${word}`;
      },
      [rows, synced],
    ),
  );
  const [timeMs, activeIndex, wordProgress] = (() => {
    const [time, line, word] = syncKey.split(":").map(Number);
    return [time, line, word] as const;
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
  }, [activeIndex, synced, rows]);

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

  if (rows.length === 0) {
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
      <div className={cn("flex flex-col items-stretch gap-3 pt-5", synced ? "pb-[70cqh]" : "pb-5")}>
        {rows.map((row, index) => (
          <LyricRow
            key={index}
            index={index}
            row={row}
            nextTimeMs={rows[index + 1]?.timeMs}
            synced={synced}
            state={lineState(index, activeIndex, synced)}
            activeWord={index === activeIndex ? wordProgress : -1}
            timeMs={synced ? timeMs : undefined}
            onSeek={seek}
          />
        ))}
      </div>
    </div>
  );
}

interface RowProps {
  index: number;
  row: LyricDisplayRow;
  nextTimeMs?: number;
  synced: boolean;
  state: LineState;
  activeWord: number;
  timeMs?: number;
  onSeek(seconds: number): void;
}

function LyricRow({ index, row, nextTimeMs, synced, state, activeWord, timeMs, onSeek }: RowProps) {
  const line = { timeMs: row.timeMs, text: row.text };
  const words =
    synced && state === "active" && row.text !== ""
      ? wordsForLine(line, nextTimeMs)
      : null;

  // An empty line is a break: it holds its place so the highlight can move off
  // the line before it, but shows nothing.
  if (row.text === "") {
    return <div data-line={index} className="h-3" aria-hidden />;
  }

  const mainClass = lyricTextClass(synced, state, "main");
  const secondaryClass = lyricTextClass(synced, state, "secondary");
  const layered = row.secondaryTexts.length > 0;

  const content = (
    <div
      className={cn(
        "flex w-full items-baseline gap-4",
        layered ? "justify-between" : "justify-start",
      )}
    >
      {renderLyricText(row.text, mainClass, words, activeWord, state === "active")}
      {layered && (
        <div className="flex min-w-0 shrink-0 flex-col items-end gap-0.5 text-right">
          {row.secondaryTexts.map((secondary, i) => {
            const secondaryWords =
              synced && state === "active"
                ? wordsForLine({ timeMs: row.timeMs, text: secondary }, nextTimeMs)
                : null;
            const secondaryActiveWord =
              state === "active" && timeMs !== undefined && secondaryWords
                ? activeWordIndex(secondaryWords, timeMs)
                : -1;
            return renderLyricText(
              secondary,
              secondaryClass,
              secondaryWords,
              secondaryActiveWord,
              false,
              `${index}-secondary-${i}`,
            );
          })}
        </div>
      )}
    </div>
  );

  if (!synced) {
    return (
      <p data-line={index} className="w-full px-1">
        {content}
      </p>
    );
  }

  return (
    <button
      type="button"
      data-line={index}
      onClick={() => onSeek(row.timeMs / 1000)}
      aria-current={state === "active"}
      title="Jump to this line"
      className="w-full cursor-pointer rounded-md px-1 py-0.5 text-left transition-transform duration-100 ease-out hover:text-content active:scale-[0.99]"
    >
      {content}
    </button>
  );
}

function lyricTextClass(synced: boolean, state: LineState, role: "main" | "secondary") {
  return cn(
    "block transition-[color,opacity] duration-300 ease-out",
    role === "main" ? "min-w-0 flex-1 text-left" : "text-right",
    synced
      ? role === "main"
        ? "text-[22px] leading-[1.2] font-semibold tracking-[-0.02em] sm:text-[26px] sm:leading-[1.15]"
        : "text-[18px] leading-[1.2] font-medium tracking-[-0.02em] sm:text-[22px] sm:leading-[1.15]"
      : role === "main"
        ? "text-[19px] leading-snug font-medium tracking-[-0.01em]"
        : "text-[16px] leading-snug font-normal tracking-[-0.01em]",
    state === "active" && "text-content",
    state === "past" && "text-content/35",
    state === "upcoming" && "text-content/55",
    state === "static" && "text-content/85",
  );
}

function renderLyricText(
  text: string,
  textClass: string,
  words: ReturnType<typeof wordsForLine> | null,
  activeWord: number,
  ariaCurrent: boolean,
  keyPrefix = "text",
) {
  if (words && words.length > 0) {
    return (
      <span className={textClass} aria-current={ariaCurrent || undefined}>
        {words.map((word, i) => {
          const sung = activeWord >= 0 && i <= activeWord;
          const current = i === activeWord;
          const held = current && word.durationMs >= HELD_WORD_MS;
          return (
            <Fragment key={`${keyPrefix}-${i}-${word.text}`}>
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
    );
  }

  return <span className={textClass}>{text}</span>;
}

function Shell({ children, className }: { children: React.ReactNode; className?: string }) {
  return <div className={cn("grid place-items-center py-10", className)}>{children}</div>;
}
