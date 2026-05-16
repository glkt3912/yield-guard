"use client";
import { useSyncExternalStore } from "react";

function subscribe(callback: () => void) {
  window.addEventListener("online", callback);
  window.addEventListener("offline", callback);
  return () => {
    window.removeEventListener("online", callback);
    window.removeEventListener("offline", callback);
  };
}

export function useNetworkStatus(): boolean | null {
  return useSyncExternalStore(
    subscribe,
    () => navigator.onLine, // クライアントスナップショット
    () => null // サーバースナップショット（SSR時はnull）
  );
}
