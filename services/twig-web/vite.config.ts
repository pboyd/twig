import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      "/auth": "http://localhost:8080",
      "/task.v1": "http://localhost:8080",
      "/health.v1": "http://localhost:8080",
      "/plan.v1": "http://localhost:8080",
      "/cli": "http://localhost:8080",
      "/account.v1": "http://localhost:8080",
      "/goal.v1": "http://localhost:8080",
    },
  },
});
