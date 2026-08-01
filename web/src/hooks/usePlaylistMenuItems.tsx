import { Heart, ListEnd, ListStart, Plus } from "lucide-react";
import { useMemo } from "react";
import type { MenuItem } from "@/components/Menu";
import type { Track } from "@/lib/api";
import { LIKED_PLAYLIST_ID } from "@/lib/playlists";
import { usePlaylists } from "@/player/PlaylistProvider";

export function usePlaylistMenuItems(tracks: Track[]): MenuItem[] {
  const { playlists, addTracks, addToLiked, removeFromLiked, isInPlaylist, createPlaylist } = usePlaylists();

  return useMemo(() => {
    const single = tracks.length === 1 ? tracks[0] : null;
    const inLiked = single ? isInPlaylist(LIKED_PLAYLIST_ID, single.id) : false;

    const playlistChildren: MenuItem[] = playlists.map((playlist) => ({
      label: playlist.name,
      checked: single ? isInPlaylist(playlist.id, single.id) : undefined,
      onSelect: () => addTracks(playlist.id, tracks),
    }));

    playlistChildren.push({
      label: "New Playlist…",
      icon: <Plus className="h-4 w-4" aria-hidden />,
      onSelect: () => {
        const name = window.prompt("Playlist name");
        if (!name?.trim()) return;
        const created = createPlaylist(name);
        addTracks(created.id, tracks);
      },
    });

    const addLabel =
      tracks.length > 1
        ? `Add ${tracks.length} Songs to Playlist`
        : "Add to Playlist";

    return [
      {
        label: inLiked ? "Remove from Liked" : "Add to Liked",
        icon: <Heart className="h-4 w-4" fill={inLiked ? "currentColor" : "none"} aria-hidden />,
        onSelect: () => (inLiked && single ? removeFromLiked(single) : addToLiked(tracks)),
      },
      {
        label: addLabel,
        icon: <Plus className="h-4 w-4" aria-hidden />,
        children: playlistChildren,
      },
    ];
  }, [tracks, playlists, addTracks, addToLiked, removeFromLiked, isInPlaylist, createPlaylist]);
}

export function buildTrackMenuItems(
  options: {
    playNext(): void;
    playLater(): void;
    goToAlbum(): void;
    goToArtist(): void;
    playlistItems: MenuItem[];
  },
): MenuItem[] {
  return [
    {
      label: "Play Next",
      icon: <ListStart className="h-4 w-4" aria-hidden />,
      onSelect: options.playNext,
    },
    {
      label: "Play Later",
      icon: <ListEnd className="h-4 w-4" aria-hidden />,
      onSelect: options.playLater,
    },
    ...options.playlistItems,
    { label: "Go to Album", onSelect: options.goToAlbum },
    { label: "Go to Artist", onSelect: options.goToArtist },
  ];
}

export function buildAlbumMenuItems(options: {
  play(): void;
  shuffle(): void;
  goToArtist?(): void;
  playlistItems: MenuItem[];
}): MenuItem[] {
  const items: MenuItem[] = [
    { label: "Play", onSelect: options.play },
    { label: "Shuffle", onSelect: options.shuffle },
    ...options.playlistItems,
  ];
  if (options.goToArtist) {
    items.push({ label: "Go to Artist", onSelect: options.goToArtist });
  }
  return items;
}
