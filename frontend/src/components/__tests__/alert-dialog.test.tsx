import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AlertDialog } from "@/components/ui/alert-dialog";

describe("AlertDialog", () => {
  afterEach(() => {
    document.body.style.overflow = "";
    delete document.body.dataset.modalCount;
  });

  it("open=true → role=alertdialog が表示される", () => {
    render(
      <AlertDialog
        open={true}
        onClose={vi.fn()}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
      />
    );
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
    expect(screen.getByText("削除の確認")).toBeInTheDocument();
    expect(screen.getByText("本当に削除しますか？")).toBeInTheDocument();
  });

  it("open=false → 何も描画されない", () => {
    const { container } = render(
      <AlertDialog
        open={false}
        onClose={vi.fn()}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
      />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("確認ボタン押下 → onConfirm と onClose が呼ばれる", () => {
    const onConfirm = vi.fn();
    const onClose = vi.fn();
    render(
      <AlertDialog
        open={true}
        onClose={onClose}
        title="削除の確認"
        description="本当に削除しますか？"
        confirmLabel="削除"
        onConfirm={onConfirm}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "削除" }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("キャンセルボタン押下 → onClose が呼ばれ onConfirm は呼ばれない", () => {
    const onConfirm = vi.fn();
    const onClose = vi.fn();
    render(
      <AlertDialog
        open={true}
        onClose={onClose}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={onConfirm}
      />
    );
    fireEvent.click(screen.getByRole("button", { name: "キャンセル" }));
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("ESC キー → onClose が呼ばれる", () => {
    const onClose = vi.fn();
    render(
      <AlertDialog
        open={true}
        onClose={onClose}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
      />
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("backdrop クリック → onClose は呼ばれない（WAI-ARIA alertdialog）", () => {
    const onClose = vi.fn();
    render(
      <AlertDialog
        open={true}
        onClose={onClose}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
      />
    );
    const backdrop = document.querySelector('[aria-hidden="true"]') as HTMLElement;
    fireEvent.click(backdrop);
    expect(onClose).not.toHaveBeenCalled();
  });

  it("open=true → document.body.style.overflow === 'hidden'", () => {
    render(
      <AlertDialog
        open={true}
        onClose={vi.fn()}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
      />
    );
    expect(document.body.style.overflow).toBe("hidden");
  });

  it("variant=destructive → AlertTriangle アイコンが表示される", () => {
    render(
      <AlertDialog
        open={true}
        onClose={vi.fn()}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
        variant="destructive"
      />
    );
    // AlertTriangle は aria-hidden なので role でなく存在確認
    const dialog = screen.getByRole("alertdialog");
    const svgs = dialog.querySelectorAll('svg[aria-hidden="true"]');
    expect(svgs.length).toBeGreaterThan(0);
  });

  it("aria-labelledby と aria-describedby が正しく設定されている", () => {
    render(
      <AlertDialog
        open={true}
        onClose={vi.fn()}
        title="削除の確認"
        description="本当に削除しますか？"
        onConfirm={vi.fn()}
      />
    );
    const dialog = screen.getByRole("alertdialog");
    const labelledBy = dialog.getAttribute("aria-labelledby");
    const describedBy = dialog.getAttribute("aria-describedby");
    expect(labelledBy).toBeTruthy();
    expect(describedBy).toBeTruthy();
    expect(document.getElementById(labelledBy!)).toHaveTextContent("削除の確認");
    expect(document.getElementById(describedBy!)).toHaveTextContent("本当に削除しますか？");
  });
});
