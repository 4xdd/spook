import type { Track } from "./api";
import { streamUrl } from "./api";
import { withAccessKey } from "./accessKey";

const MIME: Record<string, string> = {
  MP3: "audio/mpeg",
  FLAC: "audio/flac",
  OPUS: "audio/ogg; codecs=opus",
  OGG: "audio/ogg",
  M4A: "audio/mp4",
  MP4: "audio/mp4",
  AAC: "audio/aac",
  WAV: "audio/wav",
};

let probe: HTMLAudioElement | null = null;

function audioProbe(): HTMLAudioElement | null {
  if (typeof Audio === "undefined") return null;
  if (!probe) probe = new Audio();
  return probe;
}

/** True when the browser cannot decode this container natively. */
export function needsTranscode(format: string): boolean {
  const mime = MIME[format.trim().toUpperCase()];
  if (!mime) return true;
  const audio = audioProbe();
  if (!audio) return false;
  return audio.canPlayType(mime) === "";
}

/** Playback URL: native stream with byte ranges, or a live MP3 transcode pipe. */
export function playbackStreamUrl(track: Pick<Track, "streamUrl" | "format">): string {
  if (!needsTranscode(track.format)) {
    return streamUrl(track.streamUrl);
  }
  const joiner = track.streamUrl.includes("?") ? "&" : "?";
  return withAccessKey(`${track.streamUrl}${joiner}transcode=1`);
}
