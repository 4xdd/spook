import { Heart } from "lucide-react";
import type { Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import { LIKED_PLAYLIST_ID } from "@/lib/playlists";
import { usePlaylists } from "@/player/PlaylistProvider";
import { IconButton } from "./IconButton";

interface Props {
  track: Track | null | undefined;
  className?: string;
  iconClassName?: string;
  size?: "sm" | "md";
}

export function LikeButton({ track, className, iconClassName = "h-4 w-4", size = "md" }: Props) {
  const { addToLiked, removeFromLiked, isInPlaylist } = usePlaylists();
  const liked = track ? isInPlaylist(LIKED_PLAYLIST_ID, track.id) : false;

  function toggle() {
    if (!track) return;
    if (liked) removeFromLiked(track);
    else addToLiked(track);
  }

  return (
    <IconButton
      label={liked ? "Remove from Liked" : "Add to Liked"}
      active={liked}
      onClick={toggle}
      disabled={!track}
      size={size}
      className={className}
    >
      <Heart className={cn(iconClassName)} fill={liked ? "currentColor" : "none"} aria-hidden />
    </IconButton>
  );
}
