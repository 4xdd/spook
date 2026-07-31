import { AnimatePresence, motion } from "motion/react";
import { useRef } from "react";
import type { Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import { useArtworkPalette } from "@/lib/palette";
import { useAudioPulse } from "@/player/useAudioPulse";

interface Props {
  track: Track;
  playing: boolean;
}

/**
 * Each blob answers to one part of the mix, so the wash moves the way the song
 * does: bass swells the big one, the top end flickers the small one.
 */
const BLOBS = [
  { band: "low", swell: 0.18, drift: "34s", opacity: 0.55, box: "-top-[25%] -left-[20%] size-[95vmax]" },
  { band: "mid", swell: 0.12, drift: "26s", opacity: 0.42, box: "top-[5%] -right-[30%] size-[80vmax]" },
  { band: "high", swell: 0.09, drift: "21s", opacity: 0.36, box: "-bottom-[35%] left-[5%] size-[75vmax]" },
] as const;

export function NowPlayingBackdrop({ track, playing }: Props) {
  const host = useRef<HTMLDivElement>(null);
  const palette = useArtworkPalette(track.artworkId, track.color);

  useAudioPulse(host, playing);

  return (
    <div ref={host} className="pointer-events-none absolute inset-0 -z-10 overflow-hidden" aria-hidden>
      <AnimatePresence initial={false}>
        <motion.div
          key={palette.join("|")}
          className="absolute inset-0"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.9, ease: "easeInOut" }}
        >
          {BLOBS.map((blob, index) => {
            const color = palette[index % palette.length];
            return (
              <div
                key={blob.band}
                className={cn("blob absolute rounded-full", blob.box)}
                style={{
                  background: `radial-gradient(closest-side, ${color} 0%, color-mix(in oklab, ${color} 40%, transparent) 55%, transparent 78%)`,
                  opacity: blob.opacity,
                  scale: `calc(1 + var(--pulse-${blob.band}, 0) * ${blob.swell})`,
                  animationDuration: blob.drift,
                  animationDelay: `${index * -7}s`,
                }}
              />
            );
          })}
        </motion.div>
      </AnimatePresence>
    </div>
  );
}
