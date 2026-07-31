import { MicVocal } from "lucide-react";
import { cn } from "@/lib/cn";
import { useLyrics } from "@/lib/queries";
import { IconButton } from "./IconButton";

interface Props {
  trackId?: string;
  active?: boolean;
  size?: "sm" | "md";
  className?: string;
  onClick(): void;
}

/** Always rendered. Greys out only after lyrics were looked up and none exist. */
export function LyricsButton({ trackId, active, size, className, onClick }: Props) {
  const { data, isPending, isError } = useLyrics(trackId);

  const available = (data?.lines.length ?? 0) > 0;
  const settled = Boolean(trackId) && !isPending;
  const unavailable = settled && !available && !isError;

  let label = "Lyrics";
  if (!trackId) label = "Lyrics";
  else if (isPending) label = "Loading lyrics…";
  else if (isError) label = "Lyrics unavailable";
  else if (!available) label = "No lyrics for this track";
  else if (active) label = "Hide lyrics";
  else label = "Show lyrics";

  return (
    <IconButton
      label={label}
      active={active && available}
      disabled={!trackId}
      onClick={onClick}
      size={size}
      className={cn(
        unavailable && "text-tertiary/45 hover:text-tertiary/45 hover:bg-transparent",
        !trackId && "text-tertiary/35 hover:text-tertiary/35 hover:bg-transparent",
        isPending && "text-secondary",
        className,
      )}
    >
      <MicVocal className={cn("h-4 w-4", size === "md" && "h-4.5 w-4.5")} aria-hidden />
    </IconButton>
  );
}
