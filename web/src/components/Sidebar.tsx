import { Clock, CloudDownload, Disc3, Heart, ListMusic, Loader2, Music2, RefreshCw, Search, Settings, Users } from "lucide-react";
import { forwardRef, useEffect, useState } from "react";
import { NavLink, useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { cn } from "@/lib/cn";
import { plural } from "@/lib/format";
import { useRefreshLibrary, useStartScan, useStats, useDeezerStatus } from "@/lib/queries";
import { LIKED_PLAYLIST_ID } from "@/lib/playlists";
import { usePlaylists } from "@/player/PlaylistProvider";
import { SyncIndicator } from "./SyncIndicator";

const links = [
  { to: "/", label: "Recently Added", icon: Clock, end: true },
  { to: "/artists", label: "Artists", icon: Users, end: false },
  { to: "/albums", label: "Albums", icon: Disc3, end: false },
  { to: "/songs", label: "Songs", icon: ListMusic, end: false },
];

interface Props {
  onOpenSettings(): void;
}

export const Sidebar = forwardRef<HTMLInputElement, Props>(function Sidebar({ onOpenSettings }, ref) {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams] = useSearchParams();
  const { data: stats } = useStats();
  const { data: deezerStatus } = useDeezerStatus();
  const startScan = useStartScan();
  const refreshLibrary = useRefreshLibrary();
  const { playlists } = usePlaylists();
  const userPlaylists = playlists.filter((p) => !p.system);

  const [query, setQuery] = useState(() => searchParams.get("q") ?? "");
  const [syncRequested, setSyncRequested] = useState(false);

  const scanning = stats?.scan.state === "scanning";
  const syncing =
    scanning ||
    stats?.embeddings.state === "running" ||
    stats?.embeddings.state === "pending" ||
    (stats?.embeddings.pending ?? 0) > 0;
  const syncBusy = syncing || startScan.isPending || syncRequested;

  useEffect(() => {
    if (syncRequested && !syncing && !startScan.isPending) {
      setSyncRequested(false);
    }
  }, [syncRequested, syncing, startScan.isPending]);

  // Leaving search clears the field so the sidebar matches the visible page.
  useEffect(() => {
    if (!location.pathname.startsWith("/search")) setQuery("");
  }, [location.pathname]);

  // Pull newly indexed content in as soon as a scan finishes.
  const scanFinishedAt = stats?.scan.finishedAt;
  useEffect(() => {
    if (stats?.scan.state === "done" && scanFinishedAt) refreshLibrary();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scanFinishedAt]);

  function onSearchChange(value: string) {
    setQuery(value);
    if (value.trim()) {
      navigate(`/search?q=${encodeURIComponent(value)}`, { replace: location.pathname === "/search" });
    } else if (location.pathname === "/search") {
      navigate("/", { replace: true });
    }
  }

  return (
    <aside className="flex h-full min-h-0 w-full flex-col gap-4 border-r border-separator bg-canvas px-3 pt-4 pb-3">
      <div className="flex items-center gap-2 px-2">
        <span className="grid h-7 w-7 place-items-center rounded-lg bg-accent text-accent-content">
          <Music2 className="h-4 w-4" aria-hidden />
        </span>
        <span className="text-[15px] font-semibold tracking-[-0.01em]">Spook</span>
      </div>

      <div className="relative px-1">
        <Search className="pointer-events-none absolute top-1/2 left-3.5 h-3.5 w-3.5 -translate-y-1/2 text-tertiary" aria-hidden />
        <input
          ref={ref}
          type="search"
          value={query}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search"
          aria-label="Search library"
          className={cn(
            "w-full rounded-lg bg-fill py-1.5 pr-2.5 pl-8 text-[13px] text-content placeholder:text-tertiary",
            "transition-[background-color,box-shadow] outline-none focus:bg-fill-strong",
          )}
        />
      </div>

      <nav className="flex flex-col gap-0.5" aria-label="Library">
        <div className="px-2.5 pt-1 pb-1 text-[11px] font-semibold tracking-[0.04em] text-tertiary uppercase">
          Library
        </div>
        {links.map(({ to, label, icon: Icon, end }) => (
          <NavLink
            key={to}
            to={to}
            end={end}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-[13px] transition-colors duration-100",
                "active:scale-[0.99]",
                isActive ? "bg-fill-strong text-content" : "text-secondary hover:bg-fill hover:text-content",
              )
            }
          >
            {({ isActive }) => (
              <>
                <Icon className={cn("h-4 w-4", isActive && "text-accent")} aria-hidden />
                {label}
              </>
            )}
          </NavLink>
        ))}
        {deezerStatus?.enabled !== false && (
          <NavLink
            to="/deezer"
            className={({ isActive }) =>
              cn(
                "flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-[13px] transition-colors duration-100",
                "active:scale-[0.99]",
                isActive ? "bg-fill-strong text-content" : "text-secondary hover:bg-fill hover:text-content",
              )
            }
          >
            {({ isActive }) => (
              <>
                <CloudDownload className={cn("h-4 w-4", isActive && "text-accent")} aria-hidden />
                Add from Deezer
              </>
            )}
          </NavLink>
        )}
      </nav>

      <nav className="flex flex-col gap-0.5" aria-label="Playlists">
        <div className="px-2.5 pt-1 pb-1 text-[11px] font-semibold tracking-[0.04em] text-tertiary uppercase">
          Playlists
        </div>
        <NavLink
          to={`/playlists/${LIKED_PLAYLIST_ID}`}
          className={({ isActive }) =>
            cn(
              "flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-[13px] transition-colors duration-100",
              "active:scale-[0.99]",
              isActive ? "bg-fill-strong text-content" : "text-secondary hover:bg-fill hover:text-content",
            )
          }
        >
          {({ isActive }) => (
            <>
              <Heart className={cn("h-4 w-4", isActive && "text-accent")} aria-hidden />
              Liked
            </>
          )}
        </NavLink>
        {userPlaylists.map((playlist) => (
          <NavLink
            key={playlist.id}
            to={`/playlists/${playlist.id}`}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-[13px] transition-colors duration-100",
                "active:scale-[0.99]",
                isActive ? "bg-fill-strong text-content" : "text-secondary hover:bg-fill hover:text-content",
              )
            }
          >
            {({ isActive }) => (
              <>
                <ListMusic className={cn("h-4 w-4", isActive && "text-accent")} aria-hidden />
                <span className="truncate">{playlist.name}</span>
              </>
            )}
          </NavLink>
        ))}
      </nav>

      <div className="mt-auto flex flex-col gap-1.5 px-2.5 text-[11px] text-tertiary">
        {stats && (
          <>
            <SyncIndicator scan={stats.scan} embeddings={stats.embeddings} />
            <span>
              {plural(stats.albums, "album")} · {plural(stats.tracks, "song")}
            </span>
          </>
        )}
        <button
          type="button"
          onClick={onOpenSettings}
          title="Settings (,)"
          className="flex items-center gap-1.5 self-start rounded-md py-0.5 transition-colors hover:text-secondary"
        >
          <Settings className="h-3 w-3" aria-hidden />
          Settings
        </button>
        <button
          type="button"
          onClick={() => {
            if (syncBusy) return;
            setSyncRequested(true);
            startScan.mutate(undefined, {
              onError: () => setSyncRequested(false),
            });
          }}
          disabled={syncBusy}
          className="flex items-center gap-1.5 self-start rounded-md py-0.5 transition-colors hover:text-secondary disabled:pointer-events-none disabled:opacity-70"
        >
          {scanning ? (
            <>
              <Loader2 className="h-3 w-3 animate-spin" aria-hidden />
              Scanning library…
            </>
          ) : syncing ? (
            <>
              <Loader2 className="h-3 w-3 animate-spin" aria-hidden />
              Sync in progress…
            </>
          ) : (
            <>
              <RefreshCw className="h-3 w-3" aria-hidden />
              Rescan library
            </>
          )}
        </button>
      </div>
    </aside>
  );
});
