import { defineConfig } from "vitest/config";
import path from "path";

export default defineConfig({
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    globals: true,
    exclude: ["e2e/**", "node_modules/**"],
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary"],
      reportsDirectory: "./coverage",
      // 初期値は実測 (2026-07: Stmts 65.8 / Branch 54.6 / Func 55.4 / Lines 67.6) の
      // 2〜3pt 下。カバレッジ向上に合わせて段階的に引き上げる (#794)
      thresholds: {
        statements: 63,
        branches: 52,
        functions: 53,
        lines: 65,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
});
