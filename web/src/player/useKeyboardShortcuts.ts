import { useEffect } from "react";
import { usePlayer, useProgress } from "./PlayerProvider";

function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return (
    target.isContentEditable ||
    target.tagName === "INPUT" ||
    target.tagName === "TEXTAREA" ||
    target.tagName === "SELECT"
  );
}

export function useKeyboardShortcuts(onFocusSearch: () => void, onOpenSettings: () => void) {
  const player = usePlayer();
  const { currentTime } = useProgress();

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.metaKey || event.ctrlKey || event.altKey) return;

      if (isTypingTarget(event.target)) {
        if (event.key === "Escape") (event.target as HTMLElement).blur();
        return;
      }

      switch (event.key) {
        case " ":
          event.preventDefault();
          player.toggle();
          break;
        case "ArrowRight":
          event.preventDefault();
          player.seek(currentTime + (event.shiftKey ? 30 : 10));
          break;
        case "ArrowLeft":
          event.preventDefault();
          player.seek(currentTime - (event.shiftKey ? 30 : 10));
          break;
        case "ArrowUp":
          event.preventDefault();
          player.setVolume(player.volume + 0.05);
          break;
        case "ArrowDown":
          event.preventDefault();
          player.setVolume(player.volume - 0.05);
          break;
        case "n":
          player.next();
          break;
        case "p":
          player.previous();
          break;
        case "m":
          player.toggleMute();
          break;
        case "s":
          player.toggleShuffle();
          break;
        case "r":
          player.cycleRepeat();
          break;
        case "/":
          event.preventDefault();
          onFocusSearch();
          break;
        case ",":
          event.preventDefault();
          onOpenSettings();
          break;
        default:
          break;
      }
    };

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [currentTime, onFocusSearch, onOpenSettings, player]);
}
