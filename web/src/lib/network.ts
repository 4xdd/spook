type NetworkInformation = {
  saveData?: boolean;
  effectiveType?: "slow-2g" | "2g" | "3g" | "4g";
};

function connection(): NetworkInformation | undefined {
  return (navigator as Navigator & { connection?: NetworkInformation }).connection;
}

/** True on save-data or 2G/3G links (when the Network Information API is available). */
export function isSlowConnection(): boolean {
  const conn = connection();
  if (!conn) return false;
  if (conn.saveData) return true;
  const type = conn.effectiveType;
  return type === "slow-2g" || type === "2g" || type === "3g";
}

/** Seconds of audio the player should buffer before starting or resuming. */
export function bufferTargetSeconds(): number {
  const base = isSlowConnection() ? 10 : 4;
  return Math.max(base, adaptiveBufferTarget);
}

/** Raised after a rebuffer so the next start waits for more audio ahead. */
let adaptiveBufferTarget = 0;

export function noteRebuffer(): void {
  adaptiveBufferTarget = Math.min(adaptiveBufferTarget + 3, 20);
}

export function shouldPreloadNextTrack(): boolean {
  return !isSlowConnection();
}

/** Don't compete with the current track until it has this much buffered. */
export function nextTrackPreloadThreshold(): number {
  return isSlowConnection() ? 30 : 12;
}
