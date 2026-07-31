import { useCallback, useEffect, useRef, useState } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { useQueryClient } from "@tanstack/react-query";
import { AccessGate } from "@/components/AccessGate";
import { NowPlaying } from "@/components/NowPlaying";
import { PlayerBar } from "@/components/PlayerBar";
import { QueuePanel } from "@/components/QueuePanel";
import { SettingsPanel } from "@/components/SettingsPanel";
import { MobileNav, type MobileNavHandle } from "@/components/MobileNav";
import { Sidebar } from "@/components/Sidebar";
import { AlbumDetail } from "@/routes/AlbumDetail";
import { Albums } from "@/routes/Albums";
import { ArtistDetail } from "@/routes/ArtistDetail";
import { Artists } from "@/routes/Artists";
import { RecentlyAdded } from "@/routes/RecentlyAdded";
import { DeezerAdd } from "@/routes/DeezerAdd";
import { SearchResults } from "@/routes/SearchResults";
import { Songs } from "@/routes/Songs";
import { ApiError, api } from "@/lib/api";
import { persistAccessKey, readAccessKey } from "@/lib/accessKey";
import { useKeyboardShortcuts } from "@/player/useKeyboardShortcuts";
import { useLastfmAuthCallback } from "@/player/useLastfmAuthCallback";
import { useLastfmScrobbler } from "@/player/useLastfmScrobbler";

export function App() {
  const queryClient = useQueryClient();
  const [unlocked, setUnlocked] = useState(() => !!readAccessKey());

  const lock = useCallback(() => {
    persistAccessKey(null);
    queryClient.clear();
    setUnlocked(false);
  }, [queryClient]);

  useEffect(() => {
    if (!unlocked) return;
    let cancelled = false;
    void api.stats().catch((err: unknown) => {
      if (cancelled) return;
      if (err instanceof ApiError && err.status === 401) lock();
    });
    return () => {
      cancelled = true;
    };
  }, [lock, unlocked]);

  if (!unlocked) {
    return <AccessGate onUnlocked={() => setUnlocked(true)} />;
  }

  return <LibraryApp onLock={lock} />;
}

function LibraryApp({ onLock }: { onLock(): void }) {
  const [queueOpen, setQueueOpen] = useState(false);
  const [nowPlayingOpen, setNowPlayingOpen] = useState(false);
  const [lyricsOpen, setLyricsOpen] = useState(false);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [lastfmRefresh, setLastfmRefresh] = useState(0);
  const searchRef = useRef<HTMLInputElement>(null);
  const mobileNavRef = useRef<MobileNavHandle>(null);

  const focusSearch = useCallback(() => {
    if (window.matchMedia("(min-width: 640px)").matches) {
      searchRef.current?.focus();
    } else {
      mobileNavRef.current?.focusSearch();
    }
  }, []);
  const openSettings = useCallback(() => setSettingsOpen(true), []);
  const afterLastfmAuth = useCallback(() => {
    setLastfmRefresh((value) => value + 1);
    setSettingsOpen(true);
  }, []);
  useKeyboardShortcuts(focusSearch, openSettings);
  useLastfmScrobbler();
  const lastfmAuthError = useLastfmAuthCallback(afterLastfmAuth);

  return (
    <div className="grid h-full grid-rows-[auto_minmax(0,1fr)_auto] bg-canvas sm:grid-rows-[minmax(0,1fr)_auto]">
      <MobileNav ref={mobileNavRef} onOpenSettings={openSettings} />

      <div className="relative grid min-h-0 grid-cols-1 sm:grid-cols-[auto_minmax(0,1fr)_auto]">
        <div className="hidden w-60 sm:block">
          <Sidebar ref={searchRef} onOpenSettings={openSettings} />
        </div>

        <main className="min-h-0">
          <Routes>
            <Route path="/" element={<RecentlyAdded />} />
            <Route path="/albums" element={<Albums />} />
            <Route path="/albums/:id" element={<AlbumDetail />} />
            <Route path="/artists" element={<Artists />} />
            <Route path="/artists/:id" element={<ArtistDetail />} />
            <Route path="/songs" element={<Songs />} />
            <Route path="/search" element={<SearchResults />} />
            <Route path="/deezer" element={<DeezerAdd />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </main>

        <QueuePanel open={queueOpen} onClose={() => setQueueOpen(false)} />
      </div>

      <PlayerBar
        onExpand={() => setNowPlayingOpen(true)}
        onToggleLyrics={() => setLyricsOpen((open) => !open)}
        onToggleQueue={() => setQueueOpen((open) => !open)}
        lyricsOpen={lyricsOpen}
        queueOpen={queueOpen}
      />

      <SettingsPanel
        open={settingsOpen}
        onClose={() => setSettingsOpen(false)}
        lastfmError={lastfmAuthError}
        lastfmRefresh={lastfmRefresh}
        onLock={onLock}
      />

      <NowPlaying
        open={nowPlayingOpen}
        lyricsOpen={lyricsOpen}
        onClose={() => setNowPlayingOpen(false)}
        onToggleLyrics={() => setLyricsOpen((open) => !open)}
        onShowQueue={() => {
          setNowPlayingOpen(false);
          setQueueOpen(true);
        }}
      />
    </div>
  );
}
