import { motion } from "motion/react";
import { useCallback } from "react";
import type { Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import { activeLineIndex } from "@/lib/lyrics";
import { useLyrics } from "@/lib/queries";
import { useProgressSelector } from "@/player/PlayerProvider";

interface Props {
  track: Track;
  onOpen(): void;
  className?: string;
}

/** Three rows of a fixed height, so the middle one is always the live line. */
const ROW_HEIGHT = 30;
const ROWS = 3;

/**
 * The last few words of the song under the cover: enough to follow along
 * without giving up the artwork, and a way in to the full lyrics.
 */
export function LyricsPeek({ track, onOpen, className }: Props) {
  const { data } = useLyrics(track.id);
  const lines = data?.lines;
  const synced = data?.synced ?? false;

  const active = useProgressSelector(
    useCallback(
      (progress) => {
        if (!synced || !lines) return 0;
        // Before the first timestamp, hold the opening line in the slot rather
        // than leaving a blank row where the song has not started.
        return Math.max(activeLineIndex(lines, progress.currentTime * 1000), 0);
      },
      [lines, synced],
    ),
  );

  if (!synced || !lines || lines.length === 0) return null;

  return (
    <button
      type="button"
      onClick={onOpen}
      aria-label="Show full lyrics"
      className={cn(
        "lyric-peek-fade relative w-full cursor-pointer overflow-hidden text-center",
        className,
      )}
      style={{ height: ROWS * ROW_HEIGHT }}
    >
      <motion.div
        animate={{ y: ROW_HEIGHT - active * ROW_HEIGHT }}
        transition={{ type: "spring", bounce: 0, duration: 0.55 }}
      >
        {lines.map((line, index) => (
          <div
            key={index}
            style={{ height: ROW_HEIGHT, lineHeight: `${ROW_HEIGHT}px` }}
            className={cn(
              "truncate px-2 text-[15px] font-medium tracking-[-0.01em] transition-[color,opacity] duration-300",
              index === active ? "text-content" : "text-content/30",
            )}
          >
            {line.text}
          </div>
        ))}
      </motion.div>
    </button>
  );
}
