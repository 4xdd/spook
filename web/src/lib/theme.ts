export type Theme = "system" | "light" | "dark";

const STORAGE_KEY = "spook.theme";

export function readTheme(): Theme {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    if (value === "light" || value === "dark" || value === "system") return value;
  } catch {
    // Ignore blocked storage.
  }
  return "system";
}

export function applyTheme(theme: Theme) {
  const root = document.documentElement;
  if (theme === "system") {
    root.removeAttribute("data-theme");
    return;
  }
  root.setAttribute("data-theme", theme);
}

export function persistTheme(theme: Theme) {
  try {
    localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    // Ignore blocked storage.
  }
}

export function cycleTheme(current: Theme): Theme {
  if (current === "system") return "dark";
  if (current === "dark") return "light";
  return "system";
}
