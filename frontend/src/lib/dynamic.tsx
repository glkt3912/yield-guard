import React, { lazy, Suspense } from "react";
import { AlertTriangle } from "lucide-react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

const RELOAD_KEY = "chunk-reload-at";
const RELOAD_COOLDOWN_MS = 30_000;

// デプロイ前に開いたタブは旧チャンクを読みに行って失敗する。新しい index.html を取り直すため自動で再読み込みする。
// 失敗し続けるチャンクで無限リロードにならないよう、30 秒以内の再発は ErrorBoundary に任せる
function reloadOnce<T>(loader: () => Promise<T>): () => Promise<T> {
  return () =>
    loader().catch((err) => {
      try {
        const last = Number(sessionStorage.getItem(RELOAD_KEY)) || 0;
        if (Date.now() - last < RELOAD_COOLDOWN_MS) throw err;
        sessionStorage.setItem(RELOAD_KEY, String(Date.now()));
      } catch {
        throw err;
      }
      location.reload();
      return new Promise<T>(() => {});
    });
}

class ChunkErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { failed: boolean }
> {
  state = { failed: false };
  static getDerivedStateFromError() {
    return { failed: true };
  }
  render() {
    if (!this.state.failed) return this.props.children;
    return (
      <Alert>
        <AlertDescription className="flex items-center justify-between gap-2">
          <span className="flex items-center gap-1.5">
            <AlertTriangle className="h-4 w-4 shrink-0" aria-hidden="true" />
            表示に必要なファイルを読み込めませんでした。
          </span>
          <Button size="sm" variant="outline" onClick={() => location.reload()}>
            再読み込み
          </Button>
        </AlertDescription>
      </Alert>
    );
  }
}

// next/dynamic 互換の最小実装（React.lazy + Suspense）
export default function dynamic<C extends React.ComponentType<any>>(
  loader: () => Promise<{ default: C }>,
  options: { loading?: () => React.ReactNode } = {}
) {
  const Lazy = lazy(reloadOnce(loader));
  return function DynamicComponent(props: React.ComponentProps<C>) {
    return (
      <ChunkErrorBoundary>
        <Suspense fallback={options.loading?.() ?? null}>
          <Lazy {...props} />
        </Suspense>
      </ChunkErrorBoundary>
    );
  };
}
