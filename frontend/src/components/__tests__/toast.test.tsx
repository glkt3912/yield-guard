import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import React from "react";
import { ToastProvider, useToast } from "@/components/ui/toast";

// Helper component that calls toast() on mount
function ToastTrigger({
  message,
  variant,
}: {
  message: string;
  variant: "success" | "warning" | "danger";
}) {
  const { toast } = useToast();
  React.useEffect(() => {
    toast({ message, variant });
  }, []); // eslint-disable-line react-hooks/exhaustive-deps
  return null;
}

describe("Toast", () => {
  describe("ToastProvider / useToast", () => {
    it('toast({ message, variant: "success" }) → role="alert" element with message appears', () => {
      render(
        <ToastProvider>
          <ToastTrigger message="保存しました" variant="success" />
        </ToastProvider>
      );
      expect(screen.getByRole("alert")).toBeInTheDocument();
      expect(screen.getByText("保存しました")).toBeInTheDocument();
    });

    it("auto-dismisses after 4000ms", () => {
      vi.useFakeTimers();
      render(
        <ToastProvider>
          <ToastTrigger message="テスト" variant="success" />
        </ToastProvider>
      );
      expect(screen.getByRole("alert")).toBeInTheDocument();

      act(() => {
        vi.advanceTimersByTime(4000);
      });

      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
      vi.useRealTimers();
    });

    it("X button click → immediate dismiss", () => {
      render(
        <ToastProvider>
          <ToastTrigger message="閉じるテスト" variant="warning" />
        </ToastProvider>
      );
      expect(screen.getByRole("alert")).toBeInTheDocument();

      const closeButton = screen.getByRole("button", { name: "閉じる" });
      fireEvent.click(closeButton);

      expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    });

    it("multiple toasts are all rendered", () => {
      function MultiTrigger() {
        const { toast } = useToast();
        React.useEffect(() => {
          toast({ message: "first", variant: "success" });
          toast({ message: "second", variant: "danger" });
        }, []); // eslint-disable-line react-hooks/exhaustive-deps
        return null;
      }
      render(
        <ToastProvider>
          <MultiTrigger />
        </ToastProvider>
      );
      const alerts = screen.getAllByRole("alert");
      expect(alerts).toHaveLength(2);
      expect(screen.getByText("first")).toBeInTheDocument();
      expect(screen.getByText("second")).toBeInTheDocument();
    });
  });

  describe("useToast outside ToastProvider", () => {
    it("throws Error when used outside ToastProvider", () => {
      function BadComponent() {
        useToast();
        return null;
      }
      // Suppress console.error for expected error boundary output
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      expect(() => render(<BadComponent />)).toThrow("useToast must be used within ToastProvider");
      consoleSpy.mockRestore();
    });
  });
});
