import { MoreHorizontal } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import type { Album, Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import { formatReleaseType } from "@/lib/format";
import { buildAlbumMenuItems, usePlaylistMenuItems } from "@/hooks/usePlaylistMenuItems";
import { usePlayer } from "@/player/PlayerProvider";
import { api } from "@/lib/api";
import { Artwork } from "./Artwork";
import { ContextMenu, useContextMenu } from "./ContextMenu";
import { Menu } from "./Menu";
import { PlayButton } from "./PlayButton";

interface Props {
  album: Album;
  /** When known (e.g. album detail), skips a fetch for playlist actions. */
  tracks?: Track[];
  /** Hides the artist line on pages already scoped to one artist. */
  showArtist?: boolean;
  className?: string;
}

export function AlbumCard({ album, tracks: tracksProp, showArtist = true, className }: Props) {
  const { play, playShuffled, current, isPlaying, toggle } = usePlayer();
  const navigate = useNavigate();
  const contextMenu = useContextMenu();
  const [menuTracks, setMenuTracks] = useState<Track[]>(tracksProp ?? []);
  const playlistItems = usePlaylistMenuItems(menuTracks);

  useEffect(() => {
    if (tracksProp) setMenuTracks(tracksProp);
  }, [tracksProp]);

  const isCurrent = current?.albumId === album.id;

  async function loadTracks(): Promise<Track[]> {
    if (tracksProp?.length) return tracksProp;
    if (menuTracks.length) return menuTracks;
    const detail = await api.album(album.id);
    setMenuTracks(detail.tracks);
    return detail.tracks;
  }

  async function playAlbum(event?: React.MouseEvent) {
    event?.preventDefault();
    event?.stopPropagation();
    if (isCurrent) {
      toggle();
      return;
    }
    const tracks = await loadTracks();
    play(tracks, 0, { label: album.name, id: album.id });
  }

  async function shuffleAlbum() {
    const tracks = await loadTracks();
    playShuffled(tracks, { label: album.name, id: album.id });
  }

  const menuItems = buildAlbumMenuItems({
    play: () => void playAlbum(),
    shuffle: () => void shuffleAlbum(),
    goToArtist: () => navigate(`/artists/${album.artistId}`),
    playlistItems,
  });

  async function openContextMenu(event: React.MouseEvent) {
    event.preventDefault();
    event.stopPropagation();
    await loadTracks();
    contextMenu.openAt(event);
  }

  return (
    <>
      <div
        className={cn("group flex flex-col gap-2.5 outline-none", className)}
        onMouseEnter={() => void loadTracks()}
      >
        <div className="relative">
          <Link
            to={`/albums/${album.id}`}
            onContextMenu={(event) => void openContextMenu(event)}
            className="block outline-none"
          >
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
          </Link>
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
          <div
            className="absolute top-2 right-2 opacity-0 transition-opacity duration-150 group-hover:opacity-100 group-focus-within:opacity-100"
            onClick={(event) => event.stopPropagation()}
            onContextMenu={(event) => event.stopPropagation()}
          >
            <Menu
              label={`More options for ${album.name}`}
              items={menuItems}
              align="right"
              onBeforeOpen={async () => {
                await loadTracks();
              }}
              className="[&>button]:bg-black/45 [&>button]:text-white [&>button]:backdrop-blur-sm [&>button]:hover:bg-black/60 [&>button]:hover:text-white"
            >
              <MoreHorizontal className="h-4 w-4" aria-hidden />
            </Menu>
          </div>
        </div>

        <Link to={`/albums/${album.id}`} className="min-w-0 outline-none">
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
        </Link>
      </div>

      <ContextMenu
        open={contextMenu.isOpen}
        x={contextMenu.position?.x ?? 0}
        y={contextMenu.position?.y ?? 0}
        label={`Options for ${album.name}`}
        items={menuItems}
        onClose={contextMenu.close}
      />
    </>
  );
}
