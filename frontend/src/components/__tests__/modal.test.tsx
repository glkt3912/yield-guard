import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, act } from "@testing-library/react";
import { Modal } from "@/components/ui/modal";

describe("Modal", () => {
  afterEach(() => {
    // Clean up body styles and dataset between tests
    document.body.style.overflow = "";
    delete document.body.dataset.modalCount;
  });

  it("open=true → renders role=dialog and children visible", () => {
    render(
      <Modal open={true} onClose={vi.fn()} title="テストモーダル">
        <p>モーダルのコンテンツ</p>
      </Modal>
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("モーダルのコンテンツ")).toBeInTheDocument();
    expect(screen.getByText("テストモーダル")).toBeInTheDocument();
  });

  it("open=false → renders nothing (null)", () => {
    const { container } = render(
      <Modal open={false} onClose={vi.fn()}>
        <p>非表示コンテンツ</p>
      </Modal>
    );
    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("ESC key → onClose called", () => {
    const onClose = vi.fn();
    render(
      <Modal open={true} onClose={onClose}>
        <p>コンテンツ</p>
      </Modal>
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("backdrop click → onClose called", () => {
    const onClose = vi.fn();
    render(
      <Modal open={true} onClose={onClose}>
        <p>コンテンツ</p>
      </Modal>
    );
    const backdrop = document.querySelector('[aria-hidden="true"]') as HTMLElement;
    expect(backdrop).not.toBeNull();
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("open=true → document.body.style.overflow === 'hidden'", () => {
    render(
      <Modal open={true} onClose={vi.fn()}>
        <p>コンテンツ</p>
      </Modal>
    );
    expect(document.body.style.overflow).toBe("hidden");
  });

  it("after close → overflow reset", () => {
    const { rerender } = render(
      <Modal open={true} onClose={vi.fn()}>
        <p>コンテンツ</p>
      </Modal>
    );
    expect(document.body.style.overflow).toBe("hidden");

    rerender(
      <Modal open={false} onClose={vi.fn()}>
        <p>コンテンツ</p>
      </Modal>
    );
    expect(document.body.style.overflow).toBe("");
  });

  it("Tab key focus trap: last focusable element + Tab → focus moves to first", async () => {
    render(
      <Modal open={true} onClose={vi.fn()} title="フォーカストラップ">
        <button>ボタン1</button>
        <button>ボタン2</button>
        <button>ボタン3</button>
      </Modal>
    );

    const dialog = screen.getByRole("dialog");
    const focusableElements = dialog.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    const focusables = Array.from(focusableElements);
    expect(focusables.length).toBeGreaterThan(0);

    const lastFocusable = focusables[focusables.length - 1];
    const firstFocusable = focusables[0];

    // Focus the last element
    act(() => {
      lastFocusable.focus();
    });
    expect(document.activeElement).toBe(lastFocusable);

    // Tab from last → should wrap to first
    fireEvent.keyDown(document, { key: "Tab", shiftKey: false });
    expect(document.activeElement).toBe(firstFocusable);
  });

  it("Shift+Tab key focus trap: first focusable element + Shift+Tab → focus moves to last", async () => {
    render(
      <Modal open={true} onClose={vi.fn()} title="フォーカストラップ">
        <button>ボタン1</button>
        <button>ボタン2</button>
        <button>ボタン3</button>
      </Modal>
    );

    const dialog = screen.getByRole("dialog");
    const focusableElements = dialog.querySelectorAll<HTMLElement>(
      'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
    );
    const focusables = Array.from(focusableElements);

    const firstFocusable = focusables[0];
    const lastFocusable = focusables[focusables.length - 1];

    // Focus the first element
    act(() => {
      firstFocusable.focus();
    });
    expect(document.activeElement).toBe(firstFocusable);

    // Shift+Tab from first → should wrap to last
    fireEvent.keyDown(document, { key: "Tab", shiftKey: true });
    expect(document.activeElement).toBe(lastFocusable);
  });

  it("ESC key does not fire onClose when modal is not open", () => {
    const onClose = vi.fn();
    render(
      <Modal open={false} onClose={onClose}>
        <p>コンテンツ</p>
      </Modal>
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).not.toHaveBeenCalled();
  });
});
