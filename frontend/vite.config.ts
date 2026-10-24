import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

const proxy = {
  "/api": process.env.BACKEND_URL ?? "http://localhost:8080",
};

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: { "@": path.resolve(__dirname, "./src") },
  },
  server: { port: 3000, proxy },
  preview: { port: 3000, proxy },
});
