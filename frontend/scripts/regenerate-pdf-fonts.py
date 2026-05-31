#!/usr/bin/env python3
"""
PDF用 NotoSansJP フォントサブセット再生成スクリプト

【実行条件】
  pip install fonttools
  npm install @fontsource/noto-sans-jp  （frontend/ で実行）

【使い方】
  cd frontend
  python3 scripts/regenerate-pdf-fonts.py

【背景】
  pdfmake はフォントファイル単体で全グリフを解決するため、Web用サブセット
  （ブラウザが複数チャンクをオンデマンドロードする前提で生成されたもの）では
  PDFに含まれる漢字が欠落する。
  本スクリプトは generatePdf.ts / verdict.ts / charts.ts で実際に使われる
  全文字を列挙し、@fontsource の woff2 チャンクから必要グリフを抽出して
  public/fonts/ に配置する。

【新しい漢字を追加した場合】
  NEEDED_TEXT に該当文字を追記してから本スクリプトを再実行する。
"""

import glob
import subprocess
from pathlib import Path
from fontTools.ttLib import TTFont
from fontTools.merge import Merger
from fontTools.varLib import instancer

# ── 設定 ────────────────────────────────────────────────────────────────────

SCRIPT_DIR = Path(__file__).parent
FRONTEND_DIR = SCRIPT_DIR.parent
FONTS_OUT_DIR = FRONTEND_DIR / "public" / "fonts"
FONTSOURCE_DIR = FRONTEND_DIR / "node_modules" / "@fontsource" / "noto-sans-jp" / "files"

# PDF出力で使われる全文字（generatePdf.ts / verdict.ts / charts.ts の文字列リテラルから収集）
# ※ 新しい漢字をPDFに追加した場合はここに追記して再実行すること
NEEDED_TEXT = (
    # generatePdf.ts — ラベル・セクションタイトル
    "不動産投資分析レポート物件概要価格土地建物費用築年数構造最寄り駅徒歩月額賃料"
    "想定空室率ローン金額利金期間分析情報実施日総投資額表面利回り"
    "P1投資サマリーP2年間キャッシュフローP3ストレステスト結果P4取得コスト内訳"
    "DSCR基本複合実質LTV出口収益"
    "売却後総CF累積益戦略年後売却想定価格譲渡所得税手取り"
    "ストレステスト要約シナリオ判定安全危険ベースライン"
    "賃料収入返済運営経費税引前後残債"
    "初期内訳諸合計取得時明細仲介手数料込印紙登録免許税概算"
    "固定資産日割精算年間費用目"
    "総合判定適格要交渉見送り推奨"
    # verdict.ts — autoComment / reasons
    "DSCR基準達成未達超過以上下回水準満最低余裕注意"
    "重大リスク検出現時点複数確認信号デッドクロス発生投資回収負債減価償却"
    # charts.ts — グラフ凡例
    "ローン減価償却残額渉"
    # 記号・特殊文字
    "×÷―…≦≧　！％＋－＝？勧号略奨"
    # ひらがな全域
    "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん"
    "ゃゅょっ"
    # カタカナ全域
    "アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン"
    "ァィゥェォャュョッー"
    # 記号
    "（）「」【】・、。"
    # ASCII（pdfmakeが参照するため念のため含める）
    " 0123456789%.-/:ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

WEIGHTS = {
    "Regular": "400",
    "Bold": "700",
}


# ── ロジック ─────────────────────────────────────────────────────────────────

def write_chars_file(path: Path) -> None:
    chars = "".join(sorted(set(NEEDED_TEXT)))
    path.write_text(chars, encoding="utf-8")
    print(f"  文字数: {len(chars)}")


def find_chunks_with_missing(current_ttf: Path, weight: str, missing_cps: set) -> list[str]:
    pattern = str(FONTSOURCE_DIR / f"*-{weight}-normal.woff2")
    chunks = []
    for wf in sorted(glob.glob(pattern)):
        try:
            font = TTFont(wf)
            cmap = font.getBestCmap() or {}
            if missing_cps & set(cmap.keys()):
                chunks.append(wf)
        except Exception:
            pass
    return chunks


def ensure_static(src_ttf: Path, weight_int: int) -> None:
    """variable font（fvar あり）を static インスタンスに変換して上書き保存する。
    pdfkit は weight axis を解釈しないため、Bold スロットには wght=700 の static TTF が必要。
    tmp ファイル経由で atomic に書き込むことで、失敗時にオリジナルが破損しないようにする。"""
    font = TTFont(src_ttf)
    if "fvar" not in font:
        return
    print(f"  variable font を検出 → wght={weight_int} で static 化")
    instancer.instantiateVariableFont(font, {"wght": weight_int})
    tmp = src_ttf.with_suffix(".tmp")
    font.save(str(tmp))
    tmp.replace(src_ttf)  # atomic rename（失敗時はオリジナルを保持）
    size_kb = src_ttf.stat().st_size // 1024
    print(f"  → static 化完了 ({size_kb} KB)")


def build_subset(label: str, weight: str, chars_file: Path) -> None:
    print(f"\n[{label}] ビルド開始 (weight={weight})")
    src_ttf = FONTS_OUT_DIR / f"NotoSansJP-{label}.ttf"
    tmp_merged = Path(f"/tmp/NotoSansJP-{label}-merged.ttf")
    tmp_subset = Path(f"/tmp/NotoSansJP-{label}-subset.ttf")

    # variable font の場合は先に static 化する（pdfkit は wght axis 非対応）
    ensure_static(src_ttf, int(weight))

    # 現在のフォントで欠落しているグリフを特定
    current = TTFont(src_ttf)
    current_cmap = current.getBestCmap() or {}
    needed_cps = set(ord(c) for c in NEEDED_TEXT if ord(c) > 0x7F)
    missing_cps = needed_cps - set(current_cmap.keys())
    print(f"  欠落グリフ数: {len(missing_cps)}")

    if not missing_cps:
        print("  → 欠落なし、スキップ")
        return

    if not FONTSOURCE_DIR.exists():
        raise FileNotFoundError(
            f"{FONTSOURCE_DIR} が見つかりません。\n"
            "frontend/ で `npm install @fontsource/noto-sans-jp` を実行してください。"
        )

    # 必要なチャンクを特定してマージ
    chunks = find_chunks_with_missing(src_ttf, weight, missing_cps)
    print(f"  マージ対象チャンク数: {len(chunks)}")
    merger = Merger()
    merged = merger.merge([str(src_ttf)] + chunks)
    merged.save(str(tmp_merged))

    # 必要文字のみに絞り込む
    subprocess.run(
        [
            "pyftsubset", str(tmp_merged),
            f"--text-file={chars_file}",
            f"--output-file={tmp_subset}",
            "--layout-features=*",
            "--no-hinting",
        ],
        check=True,
    )

    # 配置
    dst = FONTS_OUT_DIR / f"NotoSansJP-{label}.ttf"
    dst.write_bytes(tmp_subset.read_bytes())
    size_kb = dst.stat().st_size // 1024
    print(f"  → {dst} ({size_kb} KB)")


def main() -> None:
    chars_file = Path("/tmp/pdf_chars.txt")
    print("=== PDF用 NotoSansJP サブセット再生成 ===")
    write_chars_file(chars_file)

    for label, weight in WEIGHTS.items():
        build_subset(label, weight, chars_file)

    print("\n完了。git diff --stat frontend/public/fonts/ で変化を確認してください。")


if __name__ == "__main__":
    main()
