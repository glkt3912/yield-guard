"use client";
import React from "react";
import { X } from "lucide-react";

interface FormSheetProps {
  isOpen: boolean;
  onClose: () => void;
  closeButtonRef: React.RefObject<HTMLButtonElement | null>;
  children: React.ReactNode;
}

export function FormSheet({ isOpen, onClose, closeButtonRef, children }: FormSheetProps) {
  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 lg:hidden"
      role="dialog"
      aria-modal="true"
      aria-label="投資条件入力"
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      {/* Sheet */}
      <div className="absolute bottom-0 left-0 right-0 max-h-[90vh] overflow-y-auto rounded-t-2xl bg-background shadow-xl">
        <div className="sticky top-0 flex items-center justify-between border-b bg-background px-4 py-3">
          <h2 className="text-base font-semibold">投資条件を編集</h2>
          <button
            ref={closeButtonRef}
            onClick={onClose}
            className="flex h-[44px] w-[44px] items-center justify-center rounded-full hover:bg-muted transition-colors"
            aria-label="閉じる"
          >
            <X className="h-5 w-5" />
          </button>
        </div>
        <div className="px-4 py-4">{children}</div>
      </div>
    </div>
  );
}
