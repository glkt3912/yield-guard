"use client";
import { useEffect, useState } from "react";

/** Returns a responsive chart height based on viewport width (sm/lg breakpoints). */
export function useChartHeight(mobile: number, tablet: number, desktop: number): number {
  const [height, setHeight] = useState(desktop);

  useEffect(() => {
    function update() {
      const w = window.innerWidth;
      if (w < 640) setHeight(mobile);
      else if (w < 1024) setHeight(tablet);
      else setHeight(desktop);
    }
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, [mobile, tablet, desktop]);

  return height;
}

/** Returns true when viewport is narrower than Tailwind's sm breakpoint (640px). */
export function useIsMobile(): boolean {
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    function update() {
      setIsMobile(window.innerWidth < 640);
    }
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);

  return isMobile;
}
