import type { ProxyOptions } from "vite";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath, URL } from "node:url";

// The Go server embeds whatever lands in server/internal/web/dist.
const outDir = fileURLToPath(new URL("../server/internal/web/dist", import.meta.url));
const apiTarget = process.env.SPOOK_API ?? "http://localhost:8080";

/** Keep range/chunk responses flowing through the dev proxy without buffering. */
function streamingProxy(): ProxyOptions {
  return {
    target: apiTarget,
    changeOrigin: true,
    configure(proxy) {
      proxy.on("proxyRes", (proxyRes) => {
        if (!proxyRes.headers["content-range"] && !proxyRes.headers["accept-ranges"]) {
          return;
        }
        proxyRes.headers["x-accel-buffering"] = "no";
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  build: {
    outDir,
    emptyOutDir: true,
    sourcemap: false,
  },
  server: {
    port: 5173,
    proxy: {
      "/api/v1/stream": streamingProxy(),
      "/api": { target: apiTarget, changeOrigin: true },
      "/health": { target: apiTarget, changeOrigin: true },
    },
  },
});
