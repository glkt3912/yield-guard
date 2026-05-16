import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React, { createRef } from "react";
import { FormSheet } from "@/components/FormSheet";

describe("FormSheet", () => {
  it("isOpen=false のとき何も表示されない", () => {
    const { container } = render(
      <FormSheet isOpen={false} onClose={vi.fn()} closeButtonRef={createRef()}>
        <p>コンテンツ</p>
      </FormSheet>
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("isOpen=true のときダイアログが表示される", () => {
    render(
      <FormSheet isOpen={true} onClose={vi.fn()} closeButtonRef={createRef()}>
        <p>コンテンツ</p>
      </FormSheet>
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("コンテンツ")).toBeInTheDocument();
  });

  it("ダイアログのaria-labelが設定されている", () => {
    render(
      <FormSheet isOpen={true} onClose={vi.fn()} closeButtonRef={createRef()}>
        <p>内容</p>
      </FormSheet>
    );
    expect(screen.getByRole("dialog", { name: "投資条件入力" })).toBeInTheDocument();
  });

  it("「投資条件を編集」タイトルが表示される", () => {
    render(
      <FormSheet isOpen={true} onClose={vi.fn()} closeButtonRef={createRef()}>
        <p>内容</p>
      </FormSheet>
    );
    expect(screen.getByText("投資条件を編集")).toBeInTheDocument();
  });

  it("閉じるボタンをクリックすると onClose が呼ばれる", async () => {
    const onClose = vi.fn();
    render(
      <FormSheet isOpen={true} onClose={onClose} closeButtonRef={createRef()}>
        <p>内容</p>
      </FormSheet>
    );
    await userEvent.click(screen.getByRole("button", { name: "閉じる" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("バックドロップをクリックすると onClose が呼ばれる", async () => {
    const onClose = vi.fn();
    const { container } = render(
      <FormSheet isOpen={true} onClose={onClose} closeButtonRef={createRef()}>
        <p>内容</p>
      </FormSheet>
    );
    // The backdrop is the first absolute div child of the fixed container
    const backdrop = container.querySelector(".absolute.inset-0.bg-black\\/50");
    if (backdrop) await userEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("子要素が表示される", () => {
    render(
      <FormSheet isOpen={true} onClose={vi.fn()} closeButtonRef={createRef()}>
        <input aria-label="テスト入力" />
      </FormSheet>
    );
    expect(screen.getByLabelText("テスト入力")).toBeInTheDocument();
  });

  it("aria-modal=true が設定されている", () => {
    render(
      <FormSheet isOpen={true} onClose={vi.fn()} closeButtonRef={createRef()}>
        <p>内容</p>
      </FormSheet>
    );
    expect(screen.getByRole("dialog")).toHaveAttribute("aria-modal", "true");
  });
});
