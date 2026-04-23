/** 金額を億/万/円の3段階で表示。同一テーブル内で単位を統一するため万円ベースを基本とする。
 * ルール: 1万円未満は円、1億円未満は万円、1億円以上は億円 */
export function fmtYen(v: number): string {
  const rounded = Math.round(v);
  const abs = Math.abs(rounded);
  if (abs >= 100_000_000) return `${(rounded / 100_000_000).toFixed(1)}億円`;
  if (abs >= 10_000) return `${Math.round(rounded / 10_000)}万円`;
  return `${rounded.toLocaleString("ja-JP")}円`;
}

/** 小数第2位パーセント表示（例: 8.25%） */
export function fmtPct(v: number): string {
  return `${(v * 100).toFixed(2)}%`;
}

/** 今日の日付を日本語形式で返す（例: 2026年4月23日） */
export function fmtDate(): string {
  const d = new Date();
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

/** PDFに埋め込む前の文字列サニタイズ（< > & " ' \ を除去） */
export function sanitize(v: string | number | undefined | null): string {
  if (v == null) return "";
  return String(v).replace(/[<>&"'\\]/g, "");
}
