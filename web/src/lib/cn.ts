import clsx, { type ClassValue } from "clsx";

export function cn(...values: ClassValue[]): string {
  return clsx(values);
}

/** Builds a translucent tint from an album's dominant colour. */
export function tint(color: string | undefined, alpha: number): string | undefined {
  if (!color) return undefined;
  return `color-mix(in oklab, ${color} ${Math.round(alpha * 100)}%, transparent)`;
}
