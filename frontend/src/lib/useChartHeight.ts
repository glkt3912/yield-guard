"use client";
import { useEffect, useState } from "react";

interface ResponsiveChartState {
  height: number;
  isMobile: boolean;
}

/**
 * Returns responsive chart height and mobile flag from a single resize listener.
 * Initializes with desktop values to avoid SSR hydration mismatch.
 */
export function useResponsiveChart(
  mobile: number,
  tablet: number,
  desktop: number
): ResponsiveChartState {
  const [state, setState] = useState<ResponsiveChartState>({
    height: desktop,
    isMobile: false,
  });

  useEffect(() => {
    function update() {
      const w = window.innerWidth;
      const isMobile = w < 640;
      setState({
        height: isMobile ? mobile : w < 1024 ? tablet : desktop,
        isMobile,
      });
    }
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, [mobile, tablet, desktop]);

  return state;
}

/** Convenience wrapper for components that only need responsive height. */
export function useChartHeight(mobile: number, tablet: number, desktop: number): number {
  return useResponsiveChart(mobile, tablet, desktop).height;
}
