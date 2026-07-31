import { api } from "./api";

const STORAGE_KEY = "spook.lastfm.v1";
const PENDING_KEY = "spook.lastfm.pending";
const QUEUE_KEY = "spook.lastfm.queue.v1";

export interface LastfmConnection {
  /** When false, session stays stored but nothing is sent. */
  enabled: boolean;
  username: string;
  sessionKey: string;
}

export interface LastfmQueuedScrobble {
  artist: string;
  title: string;
  album?: string;
  albumArtist?: string;
  trackNumber?: number;
  durationSec?: number;
  timestamp: number;
}

export function readLastfmConnection(): LastfmConnection | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<LastfmConnection>;
    if (typeof parsed.username !== "string" || typeof parsed.sessionKey !== "string") return null;
    if (!parsed.username || !parsed.sessionKey) return null;
    return {
      enabled: parsed.enabled !== false,
      username: parsed.username,
      sessionKey: parsed.sessionKey,
    };
  } catch {
    return null;
  }
}

export function persistLastfmConnection(connection: LastfmConnection | null) {
  try {
    if (!connection) {
      localStorage.removeItem(STORAGE_KEY);
      return;
    }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(connection));
  } catch {
    // Ignore blocked storage.
  }
}

export function markLastfmAuthPending() {
  try {
    sessionStorage.setItem(PENDING_KEY, "1");
  } catch {
    // Ignore.
  }
}

export function consumeLastfmAuthPending(): boolean {
  try {
    const pending = sessionStorage.getItem(PENDING_KEY) === "1";
    sessionStorage.removeItem(PENDING_KEY);
    return pending;
  } catch {
    return false;
  }
}

export function readScrobbleQueue(): LastfmQueuedScrobble[] {
  try {
    const raw = localStorage.getItem(QUEUE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as LastfmQueuedScrobble[];
    return Array.isArray(parsed) ? parsed.slice(0, 50) : [];
  } catch {
    return [];
  }
}

export function persistScrobbleQueue(queue: LastfmQueuedScrobble[]) {
  try {
    if (queue.length === 0) {
      localStorage.removeItem(QUEUE_KEY);
      return;
    }
    localStorage.setItem(QUEUE_KEY, JSON.stringify(queue.slice(0, 50)));
  } catch {
    // Ignore.
  }
}

export function enqueueScrobble(item: LastfmQueuedScrobble) {
  const queue = readScrobbleQueue();
  queue.push(item);
  persistScrobbleQueue(queue);
}

/** First name in Spook's "A · B" display credit. */
export function primaryArtist(display: string): string {
  const trimmed = display.trim();
  const cut = trimmed.indexOf(" · ");
  return cut === -1 ? trimmed : trimmed.slice(0, cut).trim();
}

export async function beginLastfmAuth(): Promise<void> {
  const callback = `${window.location.origin}/`;
  const { url } = await api.lastfmAuthURL(callback);
  markLastfmAuthPending();
  window.location.assign(url);
}

export async function completeLastfmAuth(token: string): Promise<LastfmConnection> {
  const session = await api.lastfmSession(token);
  const connection: LastfmConnection = {
    enabled: true,
    username: session.name,
    sessionKey: session.key,
  };
  persistLastfmConnection(connection);
  return connection;
}

export async function flushScrobbleQueue(sessionKey: string): Promise<void> {
  const queue = readScrobbleQueue();
  if (queue.length === 0) return;

  const remaining: LastfmQueuedScrobble[] = [];
  for (const item of queue) {
    try {
      await api.lastfmScrobble({ sessionKey, ...item });
    } catch {
      remaining.push(item);
    }
  }
  persistScrobbleQueue(remaining);
}
