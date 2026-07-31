import { bufferTargetSeconds } from "@/lib/network";

/** Seconds of media downloaded ahead of the playhead. */
export function bufferedAhead(audio: HTMLAudioElement): number {
  const ranges = audio.buffered;
  if (ranges.length === 0) return 0;

  const time = audio.currentTime;
  for (let i = 0; i < ranges.length; i += 1) {
    if (time >= ranges.start(i) && time <= ranges.end(i)) {
      return ranges.end(i) - time;
    }
  }

  // Playhead hasn't entered the first range yet (common right after load()).
  return Math.max(0, ranges.end(ranges.length - 1) - time);
}

/** Resolves once enough audio is buffered to survive a slow link. */
export function waitUntilReady(audio: HTMLAudioElement, signal?: AbortSignal): Promise<void> {
  const target = bufferTargetSeconds();

  return new Promise((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException("Aborted", "AbortError"));
      return;
    }

    let timeout = 0;

    const finish = () => {
      cleanup();
      resolve();
    };

    const cleanup = () => {
      audio.removeEventListener("canplay", check);
      audio.removeEventListener("progress", check);
      audio.removeEventListener("canplaythrough", finish);
      signal?.removeEventListener("abort", onAbort);
      window.clearTimeout(timeout);
    };

    const onAbort = () => {
      cleanup();
      reject(new DOMException("Aborted", "AbortError"));
    };

    const ready = () =>
      audio.readyState >= HTMLMediaElement.HAVE_ENOUGH_DATA || bufferedAhead(audio) >= target;

    const check = () => {
      if (ready()) finish();
    };

    audio.addEventListener("canplay", check);
    audio.addEventListener("progress", check);
    audio.addEventListener("canplaythrough", finish, { once: true });
    signal?.addEventListener("abort", onAbort);

    // Never block playback forever on a stalled stream.
    timeout = window.setTimeout(finish, 60_000);
    check();
  });
}

/** Buffer if needed, then start playback. */
export async function safePlay(audio: HTMLAudioElement, signal?: AbortSignal): Promise<void> {
  const target = bufferTargetSeconds();
  if (audio.readyState < HTMLMediaElement.HAVE_FUTURE_DATA || bufferedAhead(audio) < target) {
    await waitUntilReady(audio, signal);
  }
  if (signal?.aborted) throw new DOMException("Aborted", "AbortError");
  await audio.play();
}
