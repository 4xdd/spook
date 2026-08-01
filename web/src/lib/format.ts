export function formatTime(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "0:00";
  const total = Math.floor(seconds);
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;
  if (hours > 0) {
    return `${hours}:${String(minutes).padStart(2, "0")}:${String(secs).padStart(2, "0")}`;
  }
  return `${minutes}:${String(secs).padStart(2, "0")}`;
}

export function formatDurationMs(ms: number): string {
  return formatTime(ms / 1000);
}

/** Long-form duration for album and artist summaries. */
export function formatRuntime(ms: number): string {
  const minutes = Math.round(ms / 60000);
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"}`;
  const hours = Math.floor(minutes / 60);
  const rest = minutes % 60;
  if (rest === 0) return `${hours} hour${hours === 1 ? "" : "s"}`;
  return `${hours} hr ${rest} min`;
}

/** Short remaining-time label for progress indicators. */
export function formatEtaRemaining(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return "almost done";
  if (seconds < 45) return `~${Math.max(1, Math.ceil(seconds))} sec left`;
  if (seconds < 3600) {
    const min = Math.max(1, Math.ceil(seconds / 60));
    return `~${min} min left`;
  }
  const hours = Math.floor(seconds / 3600);
  const min = Math.ceil((seconds % 3600) / 60);
  if (min >= 60 || min === 0) return `~${hours + (min === 60 ? 1 : 0)} hr left`;
  return `~${hours} hr ${min} min left`;
}

export function plural(count: number, singular: string, pluralForm = `${singular}s`): string {
  return `${count} ${count === 1 ? singular : pluralForm}`;
}

export function formatReleaseType(type: "album" | "single" | "ep"): string | null {
  if (type === "single") return "Single";
  if (type === "ep") return "EP";
  return null;
}
