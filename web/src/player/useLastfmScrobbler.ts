import { useEffect, useRef } from "react";
import { api } from "@/lib/api";
import {
  enqueueScrobble,
  flushScrobbleQueue,
  primaryArtist,
  readLastfmConnection,
} from "@/lib/lastfm";
import { usePlayer, useProgress } from "./PlayerProvider";

const MIN_DURATION_SEC = 30;
const MAX_LISTEN_BEFORE_SCROBBLE = 4 * 60;

function scrobbleThreshold(durationSec: number): number {
  return Math.min(durationSec / 2, MAX_LISTEN_BEFORE_SCROBBLE);
}

/**
 * Sends Now Playing + scrobbles to Last.fm when a local session is connected.
 * The session key lives in localStorage; the server only holds the API secret.
 */
export function useLastfmScrobbler() {
  const { current, isPlaying } = usePlayer();
  const { duration } = useProgress();

  const startedAtRef = useRef<number>(0);
  const nowPlayingIdRef = useRef<string | null>(null);
  const scrobbledIdRef = useRef<string | null>(null);
  /** Actual playback time while playing; ignores seek jumps. */
  const listenedSecondsRef = useRef(0);
  const listenTickRef = useRef(0);

  // Flush any scrobbles that failed while offline whenever we gain a session.
  useEffect(() => {
    const connection = readLastfmConnection();
    if (!connection?.enabled) return;
    void flushScrobbleQueue(connection.sessionKey);
  }, [current?.id]);

  useEffect(() => {
    if (!current) {
      nowPlayingIdRef.current = null;
      scrobbledIdRef.current = null;
      return;
    }

    const connection = readLastfmConnection();
    if (!connection?.enabled) return;

    if (nowPlayingIdRef.current === current.id) return;
    nowPlayingIdRef.current = current.id;
    scrobbledIdRef.current = null;
    startedAtRef.current = Math.floor(Date.now() / 1000);

    const durationSec = Math.round((current.durationMs || duration * 1000) / 1000);
    void api
      .lastfmNowPlaying({
        sessionKey: connection.sessionKey,
        artist: primaryArtist(current.artist),
        title: current.title,
        album: current.album,
        albumArtist: current.albumArtist || undefined,
        trackNumber: current.trackNo,
        durationSec: durationSec > 0 ? durationSec : undefined,
      })
      .catch(() => {
        // Now Playing failures are not retried.
      });
  }, [current, duration]);

  useEffect(() => {
    listenedSecondsRef.current = 0;
    listenTickRef.current = 0;
  }, [current?.id]);

  useEffect(() => {
    if (!current || !isPlaying) return;

    let frame = 0;
    const tick = () => {
      const now = performance.now();
      if (listenTickRef.current > 0) {
        listenedSecondsRef.current += (now - listenTickRef.current) / 1000;
      }
      listenTickRef.current = now;

      const connection = readLastfmConnection();
      if (connection?.enabled && scrobbledIdRef.current !== current.id) {
        const durationSec = Math.round(
          (Number.isFinite(duration) && duration > 0 ? duration * 1000 : current.durationMs) / 1000,
        );
        if (durationSec >= MIN_DURATION_SEC) {
          const threshold = scrobbleThreshold(durationSec);
          if (listenedSecondsRef.current >= threshold) {
            scrobbledIdRef.current = current.id;
            const payload = {
              sessionKey: connection.sessionKey,
              artist: primaryArtist(current.artist),
              title: current.title,
              album: current.album,
              albumArtist: current.albumArtist || undefined,
              trackNumber: current.trackNo,
              durationSec,
              timestamp: startedAtRef.current || Math.floor(Date.now() / 1000),
            };

            void api.lastfmScrobble(payload).catch(() => {
              enqueueScrobble({
                artist: payload.artist,
                title: payload.title,
                album: payload.album,
                albumArtist: payload.albumArtist,
                trackNumber: payload.trackNumber,
                durationSec: payload.durationSec,
                timestamp: payload.timestamp,
              });
            });
            return;
          }
        }
      }

      frame = requestAnimationFrame(tick);
    };

    frame = requestAnimationFrame(tick);
    return () => {
      cancelAnimationFrame(frame);
      if (listenTickRef.current > 0) {
        listenedSecondsRef.current += (performance.now() - listenTickRef.current) / 1000;
        listenTickRef.current = 0;
      }
    };
  }, [current, duration, isPlaying]);
}
