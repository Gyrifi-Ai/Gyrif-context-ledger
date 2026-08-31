import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath } from "node:url";

// In Docker dev the compose file sets API_PROXY_TARGET to the gyrifi service name.
// Outside Docker, fall back to the local dev server address.
const apiProxy = process.env.API_PROXY_TARGET ?? "http://127.0.0.1:18080";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": fileURLToPath(new URL("./src", import.meta.url)) },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": apiProxy,
      "/events": apiProxy,
    },
  },
});
