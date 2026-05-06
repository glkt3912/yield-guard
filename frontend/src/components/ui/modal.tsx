"use client";

import * as React from "react";
import { X } from "lucide-react";
import { cn } from "@/lib/utils";

const FOCUSABLE = 'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])';

interface ModalProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
}

export function Modal({ open, onClose, title, children }: ModalProps) {
  const titleId = React.useId();
  const panelRef = React.useRef<HTMLDivElement>(null);
  // onClose を ref 経由で参照することで、インライン arrow の再生成による
  // useEffect の不要な再実行を防ぐ
  const onCloseRef = React.useRef(onClose);
  React.useEffect(() => {
    onCloseRef.current = onClose;
  });

  // ESC キー + Tab フォーカストラップ
  React.useEffect(() => {
    if (!open) return;

    function getFocusables() {
      return Array.from(panelRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE) ?? []);
    }

    function handleKey(e: KeyboardEvent) {
      if (e.key === "Escape") {
        onCloseRef.current();
        return;
      }
      if (e.key !== "Tab") return;
      const focusables = getFocusables();
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }

    document.addEventListener("keydown", handleKey);
    return () => document.removeEventListener("keydown", handleKey);
  }, [open]);

  // body スクロールロック（複数モーダル対応: 参照カウンタで管理）
  React.useEffect(() => {
    if (!open) return;
    const count = Number(document.body.dataset.modalCount ?? 0) + 1;
    document.body.dataset.modalCount = String(count);
    document.body.style.overflow = "hidden";
    return () => {
      const next = Number(document.body.dataset.modalCount ?? 1) - 1;
      document.body.dataset.modalCount = String(next);
      if (next === 0) document.body.style.overflow = "";
    };
  }, [open]);

  // フォーカスを最初のフォーカス可能要素へ移動
  React.useEffect(() => {
    if (!open) return;
    panelRef.current?.querySelector<HTMLElement>(FOCUSABLE)?.focus();
  }, [open]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-end sm:items-center justify-center"
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? titleId : undefined}
    >
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50"
        onClick={() => onCloseRef.current()}
        aria-hidden="true"
      />

      {/* Panel */}
      <div
        ref={panelRef}
        className={cn(
          "relative z-10 w-full bg-background shadow-xl",
          "rounded-t-2xl sm:rounded-xl sm:max-w-md",
          "animate-in slide-in-from-bottom-4 sm:slide-in-from-bottom-0 sm:zoom-in-95 fade-in-0"
        )}
      >
        {/* Header */}
        {title && (
          <div className="flex items-center justify-between border-b px-4 py-3">
            <h2 id={titleId} className="text-sm font-semibold text-foreground">
              {title}
            </h2>
            <button
              type="button"
              onClick={() => onCloseRef.current()}
              className="rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
              aria-label="閉じる"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        )}

        {/* Body */}
        <div className="px-4 py-4">{children}</div>
      </div>
    </div>
  );
}
