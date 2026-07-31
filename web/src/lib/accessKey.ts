const STORAGE_KEY = "spook.accessKey";

export function readAccessKey(): string | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (value && value.trim()) return value.trim();
  } catch {
    // Ignore blocked storage.
  }
  return null;
}

export function persistAccessKey(key: string | null) {
  try {
    if (!key) {
      localStorage.removeItem(STORAGE_KEY);
      return;
    }
    localStorage.setItem(STORAGE_KEY, key.trim());
  } catch {
    // Ignore blocked storage.
  }
}

/** Append the stored access key for media URLs that cannot send headers. */
export function withAccessKey(url: string): string {
  const key = readAccessKey();
  if (!key) return url;
  const joiner = url.includes("?") ? "&" : "?";
  return `${url}${joiner}key=${encodeURIComponent(key)}`;
}

export function accessKeyHeaders(): HeadersInit {
  const key = readAccessKey();
  if (!key) return {};
  return { Authorization: `Bearer ${key}` };
}
