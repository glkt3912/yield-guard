"use client";
import { useEffect, useState } from "react";
import { Sun, Moon } from "lucide-react";

export function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    try {
      const stored = localStorage.getItem("theme");
      const prefersDark = window.matchMedia?.("(prefers-color-scheme: dark)").matches ?? false;
      const isDark = stored ? stored === "dark" : prefersDark;
      setDark(isDark);
      document.documentElement.classList.toggle("dark", isDark);
    } catch {
      // localStorage/matchMedia unavailable (SSR or test env)
    }
  }, []);

  function toggle() {
    const next = !dark;
    setDark(next);
    document.documentElement.classList.toggle("dark", next);
    try {
      localStorage.setItem("theme", next ? "dark" : "light");
    } catch {
      // ignore
    }
  }

  return (
    <button
      onClick={toggle}
      aria-label={dark ? "ライトモードに切り替え" : "ダークモードに切り替え"}
      className="rounded-md p-2 hover:bg-secondary"
    >
      {dark ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
    </button>
  );
}
