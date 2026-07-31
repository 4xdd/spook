import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { BrowserRouter } from "react-router-dom";
import { App } from "./App";
import { EqualizerProvider } from "./player/EqualizerProvider";
import { PlayerProvider } from "./player/PlayerProvider";
import { applyTheme, readTheme } from "@/lib/theme";
import "./index.css";

applyTheme(readTheme());

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // The library only changes when a scan runs, which invalidates explicitly.
      staleTime: 60_000,
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
});

const root = document.getElementById("root");
if (!root) throw new Error("missing #root");

/**
 * Keep large desktop displays at roughly 1080p UI density. Browser and OS
 * scaling already reduce the CSS viewport, so they naturally opt out.
 */
function fitDesktopDensity(container: HTMLElement) {
  const scale = Math.max(1, Math.min(window.innerWidth / 1920, window.innerHeight / 1080));
  const inverse = 100 / scale;

  container.style.width = `${inverse}%`;
  container.style.height = `${inverse}%`;
  container.style.transform = `scale(${scale})`;
}

fitDesktopDensity(root);
window.addEventListener("resize", () => fitDesktopDensity(root));

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <PlayerProvider>
          <EqualizerProvider>
            <App />
          </EqualizerProvider>
        </PlayerProvider>
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
);
