import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import path from "node:path";

// Build to dist/, which the Go binary go:embeds and serves from the in-process
// handler at "/". Relative base so assets resolve under the Wails custom scheme.
export default defineConfig({
  base: "./",
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "src") },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
