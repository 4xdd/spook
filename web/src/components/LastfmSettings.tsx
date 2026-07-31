import { ExternalLink, Loader2 } from "lucide-react";
import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import {
  beginLastfmAuth,
  persistLastfmConnection,
  readLastfmConnection,
  type LastfmConnection,
} from "@/lib/lastfm";
import { Switch } from "./Switch";

export function LastfmSettings({
  authError,
  refreshKey,
}: {
  authError?: string | null;
  refreshKey?: number;
}) {
  const [configured, setConfigured] = useState<boolean | null>(null);
  const [connection, setConnection] = useState<LastfmConnection | null>(() => readLastfmConnection());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authError) setError(authError);
    setConnection(readLastfmConnection());
  }, [authError, refreshKey]);

  useEffect(() => {
    let cancelled = false;
    void api
      .lastfmStatus()
      .then((status) => {
        if (!cancelled) setConfigured(status.configured);
      })
      .catch(() => {
        if (!cancelled) setConfigured(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  async function connect() {
    setBusy(true);
    setError(null);
    try {
      await beginLastfmAuth();
    } catch (err) {
      setBusy(false);
      setError(err instanceof Error ? err.message : "Could not start Last.fm auth");
    }
  }

  function disconnect() {
    persistLastfmConnection(null);
    setConnection(null);
  }

  function setEnabled(enabled: boolean) {
    if (!connection) return;
    const next = { ...connection, enabled };
    persistLastfmConnection(next);
    setConnection(next);
  }

  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-[13px] font-semibold">Last.fm</h3>
          <p className="text-[11.5px] text-tertiary">
            {configured === false
              ? "Add LASTFM_API_KEY and LASTFM_API_SECRET on the server to enable."
              : connection
                ? `Scrobbling as ${connection.username}. Session stays in this browser.`
                : "Connect to scrobble plays from this browser."}
          </p>
        </div>
        {connection && (
          <Switch
            checked={connection.enabled}
            onChange={setEnabled}
            label="Last.fm scrobbling"
          />
        )}
      </div>

      {error && <p className="text-[12px] text-accent">{error}</p>}

      <div className="flex flex-wrap items-center gap-2">
        {connection ? (
          <>
            <a
              href={`https://www.last.fm/user/${encodeURIComponent(connection.username)}`}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 rounded-lg bg-fill px-2.5 py-1.5 text-[12.5px] text-secondary transition-colors hover:bg-fill-strong hover:text-content"
            >
              View profile
              <ExternalLink className="h-3 w-3" aria-hidden />
            </a>
            <button
              type="button"
              onClick={disconnect}
              className="rounded-lg px-2.5 py-1.5 text-[12.5px] text-secondary transition-colors hover:bg-fill hover:text-content"
            >
              Disconnect
            </button>
          </>
        ) : (
          <button
            type="button"
            onClick={() => void connect()}
            disabled={busy || configured === false}
            className="inline-flex items-center gap-1.5 rounded-lg bg-fill px-2.5 py-1.5 text-[12.5px] text-content transition-colors hover:bg-fill-strong disabled:cursor-not-allowed disabled:opacity-50"
          >
            {busy ? <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden /> : null}
            Connect Last.fm
          </button>
        )}
      </div>
    </section>
  );
}
