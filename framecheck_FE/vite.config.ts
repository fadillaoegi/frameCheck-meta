/// <reference types="vitest/config" />
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

const apiPort = process.env.FRAMECHECK_API_PORT ?? "8089";
const webPort = Number(process.env.FRAMECHECK_WEB_PORT ?? "5181");

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: "127.0.0.1",
    port: webPort,
    strictPort: true,
    // Same-origin proxy: the browser never talks to the API port directly,
    // so no CORS configuration is needed.
    proxy: { "/v1": `http://127.0.0.1:${apiPort}` },
  },
  test: { environment: "node" },
});
