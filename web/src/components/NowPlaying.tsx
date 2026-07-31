import { ChevronDown, ListMusic, Repeat, Repeat1, Shuffle, SkipBack, SkipForward } from "lucide-react";
import { AnimatePresence, motion, type PanInfo } from "motion/react";
import { useNavigate } from "react-router-dom";
import { cn } from "@/lib/cn";
import { useLyricsAvailable, useSyncedLyrics } from "@/lib/queries";
import { usePlayer } from "@/player/PlayerProvider";
import { Artwork } from "./Artwork";
import { IconButton } from "./IconButton";
import { Lyrics } from "./Lyrics";
import { LyricsButton } from "./LyricsButton";
import { LyricsPeek } from "./LyricsPeek";
import { NowPlayingBackdrop } from "./NowPlayingBackdrop";
import { PlayButton } from "./PlayButton";
import { Scrubber } from "./Scrubber";
import { VolumeControl } from "./VolumeControl";

interface Props {
  open: boolean;
  lyricsOpen: boolean;
  onClose(): void;
  onToggleLyrics(): void;
  onShowQueue(): void;
}

/**
 * Projects where a flick would come to rest, matching the deceleration curve
 * used by native scroll views, so a fast swipe dismisses even if it is short.
 */
function project(velocity: number, decelerationRate = 0.998): number {
  return ((velocity / 1000) * decelerationRate) / (1 - decelerationRate);
}

/** A non-overshooting layout curve keeps the large column swap visually stable. */
const SWAP = { duration: 0.5, ease: [0.22, 1, 0.36, 1] } as const;

/** Peek lyrics squash down under the controls instead of fading in place. */
const PEEK_TUCK = { duration: 0.42, ease: [0.22, 1, 0.36, 1] } as const;

/** Full lyrics wait for the peek row to collapse so the two do not collide. */
const LYRICS_IN = { duration: 0.5, ease: [0.22, 1, 0.36, 1], delay: 0.14 } as const;

export function NowPlaying({ open, lyricsOpen, onClose, onToggleLyrics, onShowQueue }: Props) {
  const { current, isPlaying, toggle, next, previous, shuffle, repeat, toggleShuffle, cycleRepeat } = usePlayer();
  const navigate = useNavigate();

  const RepeatIcon = repeat === "one" ? Repeat1 : Repeat;
  // Stick the preference across tracks, but fall back to full artwork when the
  // current song has nothing to show.
  const lyricsAvailable = useLyricsAvailable(current?.id);
  const showLyrics = lyricsOpen && lyricsAvailable;
  // Only timed lyrics can hold a line in the middle slot, so only they peek.
  const syncedLyrics = useSyncedLyrics(current?.id);
  const showPeek = !showLyrics && syncedLyrics;

  function onDragEnd(_event: unknown, info: PanInfo) {
    // Decide from where the gesture is going, not from where it stopped.
    const projected = info.offset.y + project(info.velocity.y);
    if (projected > 180) onClose();
  }

  function goTo(path: string) {
    onClose();
    navigate(path);
  }

  return (
    <AnimatePresence>
      {open && current && (
        <motion.section
          key="now-playing"
          role="dialog"
          aria-modal="true"
          aria-label="Now Playing"
          initial={{ y: "100%" }}
          animate={{ y: 0 }}
          exit={{ y: "100%" }}
          // Enter and exit travel the same path, so the panel returns the way it came.
          transition={{ type: "spring", bounce: 0, duration: 0.45 }}
          drag="y"
          dragDirectionLock
          dragConstraints={{ top: 0, bottom: 0 }}
          dragElastic={{ top: 0, bottom: 0.6 }}
          onDragEnd={onDragEnd}
          className="fixed inset-0 z-50 flex touch-none flex-col overflow-hidden bg-canvas"
        >
          <NowPlayingBackdrop track={current} playing={isPlaying} />

          <header className="flex items-center justify-between px-4 py-3 pt-safe sm:px-4">
            <IconButton label="Close Now Playing" onClick={onClose}>
              <ChevronDown className="h-5 w-5" aria-hidden />
            </IconButton>
            <div
              className="h-1 w-9 cursor-grab rounded-full bg-content/25 active:cursor-grabbing"
              aria-hidden
            />
            <div className="flex items-center gap-1">
              <IconButton label="Show queue" onClick={onShowQueue}>
                <ListMusic className="h-4.5 w-4.5" aria-hidden />
              </IconButton>
            </div>
          </header>

          <div
            className="np-stage min-h-0 flex-1 overflow-hidden px-4 pb-8 pb-safe sm:px-6 sm:pb-10"
            data-lyrics={showLyrics}
            data-peek={showPeek}
          >
            <div className="np-art flex min-h-0 w-full max-w-[26rem] flex-col items-center justify-center">
              <motion.div
                layout
                // Paused artwork sits back; playing artwork comes forward.
                animate={{ scale: isPlaying ? 1 : 0.92 }}
                transition={SWAP}
                // Fits whichever axis of the cell runs out first, so the art
                // shrinks on a short window instead of spilling over. With the
                // peek showing it reserves the room that peek needs below it.
                className={cn(
                  "np-cover aspect-square shrink-0",
                  showPeek ? "w-[min(100cqw,calc(100cqh_-_9rem))]" : "w-[min(100cqw,100cqh)]",
                )}
              >
                <Artwork
                  artworkId={current.artworkId}
                  size={1000}
                  alt={`${current.album} by ${current.artist}`}
                  color={current.color}
                  rounded="lg"
                  eager
                  className="h-full w-full shadow-art"
                />
              </motion.div>
            </div>

            <AnimatePresence initial={false} mode="popLayout">
              {showPeek && (
                <motion.div
                  key="lyrics-peek"
                  className="np-peek flex w-full max-w-[26rem] shrink-0 items-center justify-center overflow-hidden"
                  style={{ transformOrigin: "50% 100%" }}
                  initial={{ scaleY: 0, opacity: 0, y: 8 }}
                  animate={{ scaleY: 1, opacity: 1, y: 0 }}
                  exit={{ scaleY: 0, opacity: 0, y: 10 }}
                  transition={PEEK_TUCK}
                >
                  <LyricsPeek track={current} onOpen={onToggleLyrics} />
                </motion.div>
              )}

              {showLyrics && (
                <motion.div
                  key="full-lyrics"
                  layout
                  className="np-lyrics flex min-h-0 w-full"
                  initial={{ opacity: 0, x: 16 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 16, transition: SWAP }}
                  transition={{ layout: SWAP, opacity: LYRICS_IN, x: LYRICS_IN }}
                >
                  <Lyrics track={current} className="min-h-0 w-full flex-1" />
                </motion.div>
              )}
            </AnimatePresence>

            <motion.div layout transition={SWAP} className="np-controls flex w-full max-w-[26rem] flex-col gap-6">
              <div className="flex items-start justify-between gap-4">
                <div className="min-w-0">
                  <h2 className="truncate text-[22px] font-semibold">{current.title}</h2>
                  <button
                    type="button"
                    onClick={() => goTo(`/artists/${current.artistId}`)}
                    className="np-track-meta-secondary block max-w-full truncate text-left text-[17px] text-accent hover:underline"
                  >
                    {current.artist}
                  </button>
                  <button
                    type="button"
                    onClick={() => goTo(`/albums/${current.albumId}`)}
                    className="np-track-meta-secondary block max-w-full truncate text-left text-[15px] text-secondary hover:underline"
                  >
                    {current.album}
                  </button>
                </div>
                <span className="np-track-meta-secondary mt-1 shrink-0 rounded-md bg-fill px-2 py-0.5 text-[11px] font-medium tracking-wide text-secondary uppercase">
                  {current.format}
                  {current.sampleRateHz ? ` · ${(current.sampleRateHz / 1000).toFixed(1)} kHz` : ""}
                </span>
              </div>

              <Scrubber size="lg" />

              <div className="flex items-center justify-between">
                <div className="flex items-center gap-0.5">
                  <LyricsButton
                    trackId={current.id}
                    active={showLyrics}
                    onClick={onToggleLyrics}
                  />
                  <IconButton label="Shuffle" active={shuffle} onClick={toggleShuffle}>
                    <Shuffle className="h-4.5 w-4.5" aria-hidden />
                  </IconButton>
                </div>

                <div className="flex items-center gap-6">
                  <IconButton label="Previous" onClick={previous} size="md">
                    <SkipBack className="h-6 w-6" fill="currentColor" strokeWidth={0} aria-hidden />
                  </IconButton>
                  <PlayButton playing={isPlaying} onClick={toggle} size="lg" />
                  <IconButton label="Next" onClick={next} size="md">
                    <SkipForward className="h-6 w-6" fill="currentColor" strokeWidth={0} aria-hidden />
                  </IconButton>
                </div>

                <IconButton
                  label={repeat === "off" ? "Repeat off" : repeat === "all" ? "Repeat all" : "Repeat one"}
                  active={repeat !== "off"}
                  onClick={cycleRepeat}
                >
                  <RepeatIcon className={cn("h-4.5 w-4.5")} aria-hidden />
                </IconButton>
              </div>

              <VolumeControl className="justify-center" />
            </motion.div>
          </div>
        </motion.section>
      )}
    </AnimatePresence>
  );
}
