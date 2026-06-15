#!/usr/bin/env python3
"""
PDF用 NotoSansJP フォントサブセット再生成スクリプト

【実行条件】
  pip install fonttools brotli
  npm install --no-save @fontsource/noto-sans-jp   （frontend/ で実行）

【使い方】
  cd frontend
  python3 scripts/regenerate-pdf-fonts.py

【背景 / 設計判断（重要）】
  pdfmake はフォントファイル単体で全グリフを解決するため、Web用に分割された
  woff2 チャンク（ブラウザがオンデマンドロードする前提）をそのまま使うと
  PDF に含まれる漢字が欠落する。

  さらに **可変フォント（fvar あり）を instancer で static 化した TTF は、
  pdfmake がバンドルする pdfkit のフォント埋め込み処理を無限ハングさせる**
  （Issue #783）。サブセットして小さくしてもハングは解消しない＝サイズではなく
  フォント構造の非互換が原因。

  そこで本スクリプトは instancer を一切使わず、@fontsource が配布する
  **ウェイト別（400 / 700）の static woff2 チャンク**から必要グリフを集めて
  クリーンな static TTF を生成する。
    - Regular = weight 400 チャンク群
    - Bold    = weight 700 チャンク群（pdfkit は wght axis 非対応のため、
                太字は「本物の 700 ウェイトの static フォント」を別ファイルで持つ）

  グリフ欠落の whack-a-mole を避けるため、収録範囲は手書きの文字列ではなく
  **JIS X 0208（第1+第2水準, 約6,900漢字）+ かな + 記号 + 半角英数**を採用する。
  これで実務日本語（バックエンド生成メッセージ含む）の欠落はまず起きない。
"""

import glob
import subprocess
from pathlib import Path
from fontTools.ttLib import TTFont
from fontTools.merge import Merger

# ── 設定 ────────────────────────────────────────────────────────────────────

SCRIPT_DIR = Path(__file__).parent
FRONTEND_DIR = SCRIPT_DIR.parent
FONTS_OUT_DIR = FRONTEND_DIR / "public" / "fonts"
FONTSOURCE_DIR = FRONTEND_DIR / "node_modules" / "@fontsource" / "noto-sans-jp" / "files"

WEIGHTS = {
    "Regular": "400",
    "Bold": "700",
}

# pdfmake は横書き・OpenType シェーピング不要なので GSUB/GPOS は落としてサイズを抑える
LAYOUT_FEATURES = ""


def build_charset() -> str:
    """JIS X 0208（第1+第2水準）+ かな・記号・全角形 + ASCII を網羅した文字集合を返す。"""
    chars: set[str] = set()
    # JIS X 0208 全区点を EUC-JP 経由で Unicode 化
    for hi in range(0x21, 0x7F):
        for lo in range(0x21, 0x7F):
            try:
                chars.add(bytes([hi + 0x80, lo + 0x80]).decode("euc_jp"))
            except UnicodeDecodeError:
                pass
    # ASCII
    for cp in range(0x20, 0x7F):
        chars.add(chr(cp))
    # CJK記号・かな・全角形・囲み数字・幾何記号（PDFのラベルや凡例で使用）
    for lo, hi in [
        (0x3000, 0x30FF),  # CJK記号・ひらがな・カタカナ
        (0x2460, 0x24FF),  # 囲み英数字
        (0x25A0, 0x25FF),  # 幾何学模様（凡例マーカー等）
        (0xFF00, 0xFFEF),  # 半角・全角形
    ]:
        for cp in range(lo, hi):
            chars.add(chr(cp))
    return "".join(sorted(chars))


def find_chunks(weight: str, needed_cps: set[int]) -> list[str]:
    """指定ウェイトの woff2 チャンクのうち、必要コードポイントを含むものを返す。"""
    chunks = []
    for wf in sorted(glob.glob(str(FONTSOURCE_DIR / f"*-{weight}-normal.woff2"))):
        try:
            cmap = TTFont(wf).getBestCmap() or {}
        except Exception:
            continue
        if needed_cps & set(cmap.keys()):
            chunks.append(wf)
    return chunks


def build_font(label: str, weight: str, chars_file: Path, needed_cps: set[int]) -> None:
    print(f"\n[{label}] ビルド開始 (weight={weight})")
    if not FONTSOURCE_DIR.exists():
        raise FileNotFoundError(
            f"{FONTSOURCE_DIR} が見つかりません。\n"
            "frontend/ で `npm install --no-save @fontsource/noto-sans-jp` を実行してください。"
        )

    chunks = find_chunks(weight, needed_cps)
    print(f"  対象チャンク数: {len(chunks)}")
    if not chunks:
        raise RuntimeError(f"weight={weight} のチャンクが見つかりません")

    # チャンクをマージしてから必要文字に絞り込む（instancer は使わない）
    tmp_merged = Path(f"/tmp/NotoSansJP-{label}-merged.ttf")
    if len(chunks) == 1:
        TTFont(chunks[0]).save(str(tmp_merged))
    else:
        Merger().merge(chunks).save(str(tmp_merged))

    dst = FONTS_OUT_DIR / f"NotoSansJP-{label}.ttf"
    subprocess.run(
        [
            "pyftsubset", str(tmp_merged),
            f"--text-file={chars_file}",
            f"--output-file={dst}",
            f"--layout-features={LAYOUT_FEATURES}",
            "--no-hinting",
        ],
        check=True,
    )

    ft = TTFont(dst)
    cmap = ft.getBestCmap() or {}
    missing = sorted(c for c in chars_file.read_text(encoding="utf-8") if ord(c) > 0x7F and ord(c) not in cmap)
    size_kb = dst.stat().st_size // 1024
    print(
        f"  → {dst} ({size_kb} KB, weight={ft['OS/2'].usWeightClass}, "
        f"glyphs={ft['maxp'].numGlyphs}, missing={len(missing)})"
    )
    if missing:
        print(f"  ⚠ 欠落: {''.join(missing[:60])}")


def main() -> None:
    print("=== PDF用 NotoSansJP サブセット再生成（clean static / instancer 不使用）===")
    charset = build_charset()
    print(f"収録文字数: {len(charset)}")
    chars_file = Path("/tmp/pdf_chars.txt")
    chars_file.write_text(charset, encoding="utf-8")
    needed_cps = {ord(c) for c in charset if ord(c) > 0x7F}

    for label, weight in WEIGHTS.items():
        build_font(label, weight, chars_file, needed_cps)

    print("\n完了。git diff --stat frontend/public/fonts/ で変化を確認してください。")


if __name__ == "__main__":
    main()
