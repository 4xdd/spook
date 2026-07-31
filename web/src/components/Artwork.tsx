import { Music } from "lucide-react";
import { useState } from "react";
import { artworkUrl } from "@/lib/api";
import { cn } from "@/lib/cn";

interface Props {
  artworkId?: string;
  size: 64 | 300 | 1000;
  alt?: string;
  className?: string;
  /** Dominant colour, used to tint the placeholder while loading. */
  color?: string;
  rounded?: "sm" | "md" | "lg";
  eager?: boolean;
}

const radii = {
  sm: "rounded-[4px]",
  md: "rounded-lg",
  lg: "rounded-xl",
};

export function Artwork({ artworkId, size, alt = "", className, color, rounded = "md", eager }: Props) {
  const [loaded, setLoaded] = useState(false);
  const src = artworkUrl(artworkId, size);

  return (
    <div
      className={cn(
        "relative isolate shrink-0 overflow-hidden bg-fill",
        radii[rounded],
        className,
      )}
      style={color ? { backgroundColor: `color-mix(in oklab, ${color} 45%, var(--surface))` } : undefined}
    >
      {src ? (
        <img
          src={src}
          alt={alt}
          loading={eager ? "eager" : "lazy"}
          decoding="async"
          draggable={false}
          onLoad={() => setLoaded(true)}
          className={cn(
            "h-full w-full object-cover transition-opacity duration-300",
            loaded ? "opacity-100" : "opacity-0",
          )}
        />
      ) : (
        <div className="flex h-full w-full items-center justify-center text-tertiary">
          <Music className="h-1/3 w-1/3" strokeWidth={1.5} aria-hidden />
        </div>
      )}
      {/*
        A hairline inset edge keeps light artwork from bleeding into light
        backgrounds without drawing a hard border.
      */}
      <div
        className={cn("pointer-events-none absolute inset-0 ring-1 ring-inset ring-black/10", radii[rounded])}
        aria-hidden
      />
    </div>
  );
}
