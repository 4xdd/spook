import { Link } from "react-router-dom";
import type { Album } from "@/lib/api";
import { cn } from "@/lib/cn";
import { formatReleaseType } from "@/lib/format";
import { usePlayer } from "@/player/PlayerProvider";
import { api } from "@/lib/api";
import { Artwork } from "./Artwork";
import { PlayButton } from "./PlayButton";

interface Props {
  album: Album;
  /** Hides the artist line on pages already scoped to one artist. */
  showArtist?: boolean;
  className?: string;
}

export function AlbumCard({ album, showArtist = true, className }: Props) {
  const { play, current, isPlaying, toggle } = usePlayer();
  const isCurrent = current?.albumId === album.id;

  async function playAlbum(event: React.MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    if (isCurrent) {
      toggle();
      return;
    }
    const detail = await api.album(album.id);
    play(detail.tracks, 0, { label: detail.album.name, id: detail.album.id });
  }

  return (
    <Link
      to={`/albums/${album.id}`}
      className={cn("group flex flex-col gap-2.5 outline-none", className)}
    >
      <div className="relative">
        <Artwork
          artworkId={album.artworkId}
          size={300}
          alt={`${album.name} by ${album.artist}`}
          color={album.color}
          rounded="lg"
          className={cn(
            "aspect-square w-full shadow-art transition-transform duration-200 ease-out",
            "group-hover:-translate-y-0.5 group-active:translate-y-0 group-active:scale-[0.985]",
          )}
        />
        <div
          className={cn(
            "absolute right-2 bottom-2 transition-[opacity,transform] duration-200 ease-out",
            isCurrent && isPlaying
              ? "opacity-100"
              : "translate-y-1 opacity-0 group-hover:translate-y-0 group-hover:opacity-100 group-focus-visible:translate-y-0 group-focus-visible:opacity-100",
          )}
        >
          <PlayButton
            playing={isCurrent && isPlaying}
            onClick={playAlbum}
            label={`Play ${album.name}`}
          />
        </div>
      </div>

      <div className="min-w-0">
        <div className="truncate text-[13px] leading-snug text-content group-hover:underline">{album.name}</div>
        {showArtist ? (
          <div className="truncate text-[13px] leading-snug text-secondary">{album.artist}</div>
        ) : (
          (() => {
            const release = formatReleaseType(album.releaseType);
            const subtitle = [release, album.year?.toString()].filter(Boolean).join(" · ");
            return subtitle ? (
              <div className="text-[13px] leading-snug text-secondary">{subtitle}</div>
            ) : null;
          })()
        )}
      </div>
    </Link>
  );
}
