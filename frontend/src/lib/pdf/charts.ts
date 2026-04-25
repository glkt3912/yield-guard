import type { AcquisitionCostBreakdown, YearlyResult } from "@/types/investment";

const PAD = { top: 20, right: 10, bottom: 30, left: 55 };

function escapeXml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function fmtLabel(v: number): string {
  const abs = Math.abs(v);
  if (abs >= 1_000_000) return `${(v / 1_000_000).toFixed(0)}百万`;
  if (abs >= 10_000) return `${Math.round(v / 10_000)}万`;
  return String(Math.round(v));
}

/** P2用: 税引後CF棒グラフ（最大10年分） */
export function buildCfBarChartSvg(
  yearlyResults: YearlyResult[],
  width = 480,
  height = 160
): string {
  const data = yearlyResults.slice(0, 10);
  if (data.length === 0)
    return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}"></svg>`;

  const innerW = width - PAD.left - PAD.right;
  const innerH = height - PAD.top - PAD.bottom;
  const barW = innerW / data.length;

  const values = data.map((r) => r.afterTaxCashFlow);
  const maxAbs = Math.max(...values.map(Math.abs), 1);
  const midY = PAD.top + innerH * 0.5;
  const scale = (innerH * 0.45) / maxAbs;

  const bars = data
    .map((r, i) => {
      const h = Math.abs(r.afterTaxCashFlow) * scale;
      const x = PAD.left + i * barW + barW * 0.1;
      const bw = barW * 0.8;
      const fill = r.isInDeadCrossZone
        ? "#fca5a5"
        : r.afterTaxCashFlow >= 0
          ? "#60a5fa"
          : "#f87171";
      const y = r.afterTaxCashFlow >= 0 ? midY - h : midY;
      const label = fmtLabel(r.afterTaxCashFlow);
      const labelY = r.afterTaxCashFlow >= 0 ? y - 2 : y + h + 10;
      return [
        `<rect x="${x.toFixed(1)}" y="${y.toFixed(1)}" width="${bw.toFixed(1)}" height="${Math.max(h, 1).toFixed(1)}" fill="${fill}"/>`,
        `<text x="${(x + bw / 2).toFixed(1)}" y="${labelY.toFixed(1)}" text-anchor="middle" font-size="7" fill="#64748b">${escapeXml(label)}</text>`,
        `<text x="${(x + bw / 2).toFixed(1)}" y="${(height - 4).toFixed(1)}" text-anchor="middle" font-size="7" fill="#64748b">${r.year}年</text>`,
      ].join("\n");
    })
    .join("\n");

  // 軸
  const axis = [
    `<line x1="${PAD.left}" y1="${midY.toFixed(1)}" x2="${(PAD.left + innerW).toFixed(1)}" y2="${midY.toFixed(1)}" stroke="#e2e8f0" stroke-width="1"/>`,
    `<line x1="${PAD.left}" y1="${PAD.top}" x2="${PAD.left}" y2="${(PAD.top + innerH).toFixed(1)}" stroke="#e2e8f0" stroke-width="1"/>`,
    `<text x="${(PAD.left - 4).toFixed(1)}" y="${(midY + 4).toFixed(1)}" text-anchor="end" font-size="7" fill="#64748b">0</text>`,
    `<text x="${(PAD.left - 4).toFixed(1)}" y="${(PAD.top + 4).toFixed(1)}" text-anchor="end" font-size="7" fill="#64748b">${fmtLabel(maxAbs)}</text>`,
  ].join("\n");

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}">
${axis}
${bars}
</svg>`;
}

/** P3用: 元金返済 vs 減価償却 折れ線グラフ（最大35年分） */
export function buildDeadCrossLineSvg(
  yearlyResults: YearlyResult[],
  deadCrossYear: number,
  width = 480,
  height = 160
): string {
  const data = yearlyResults.slice(0, 35);
  if (data.length === 0)
    return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}"></svg>`;

  const innerW = width - PAD.left - PAD.right;
  const innerH = height - PAD.top - PAD.bottom;

  const principals = data.map((r) => r.annualPrincipal);
  const deprs = data.map((r) => r.annualDepreciation);
  const maxVal = Math.max(...principals, ...deprs, 1);
  const scale = innerH / maxVal;
  const stepX = innerW / (data.length - 1 || 1);

  function polyline(values: number[], color: string): string {
    const pts = values
      .map(
        (v, i) =>
          `${(PAD.left + i * stepX).toFixed(1)},${(PAD.top + innerH - v * scale).toFixed(1)}`
      )
      .join(" ");
    return `<polyline points="${pts}" fill="none" stroke="${color}" stroke-width="1.5"/>`;
  }

  const crossLine =
    deadCrossYear > 0 && deadCrossYear <= data.length
      ? `<line x1="${(PAD.left + (deadCrossYear - 1) * stepX).toFixed(1)}" y1="${PAD.top}" x2="${(PAD.left + (deadCrossYear - 1) * stepX).toFixed(1)}" y2="${(PAD.top + innerH).toFixed(1)}" stroke="#dc2626" stroke-width="1" stroke-dasharray="3,2"/>
         <text x="${(PAD.left + (deadCrossYear - 1) * stepX + 3).toFixed(1)}" y="${(PAD.top + 10).toFixed(1)}" font-size="7" fill="#dc2626">${deadCrossYear}年目</text>`
      : "";

  const axis = [
    `<line x1="${PAD.left}" y1="${(PAD.top + innerH).toFixed(1)}" x2="${(PAD.left + innerW).toFixed(1)}" y2="${(PAD.top + innerH).toFixed(1)}" stroke="#e2e8f0" stroke-width="1"/>`,
    `<line x1="${PAD.left}" y1="${PAD.top}" x2="${PAD.left}" y2="${(PAD.top + innerH).toFixed(1)}" stroke="#e2e8f0" stroke-width="1"/>`,
    `<text x="${(PAD.left - 4).toFixed(1)}" y="${(PAD.top + 4).toFixed(1)}" text-anchor="end" font-size="7" fill="#64748b">${fmtLabel(maxVal)}</text>`,
    `<text x="${(PAD.left - 4).toFixed(1)}" y="${(PAD.top + innerH).toFixed(1)}" text-anchor="end" font-size="7" fill="#64748b">0</text>`,
  ].join("\n");

  // 凡例
  const legend = [
    `<rect x="${PAD.left}" y="${(PAD.top + innerH + 16).toFixed(1)}" width="8" height="3" fill="#3b82f6"/>`,
    `<text x="${PAD.left + 10}" y="${(PAD.top + innerH + 19).toFixed(1)}" font-size="7" fill="#64748b">元金返済</text>`,
    `<rect x="${PAD.left + 55}" y="${(PAD.top + innerH + 16).toFixed(1)}" width="8" height="3" fill="#f59e0b"/>`,
    `<text x="${PAD.left + 65}" y="${(PAD.top + innerH + 19).toFixed(1)}" font-size="7" fill="#64748b">減価償却</text>`,
  ].join("\n");

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}">
${axis}
${polyline(principals, "#3b82f6")}
${polyline(deprs, "#f59e0b")}
${crossLine}
${legend}
</svg>`;
}

/** P5用: 取得コスト内訳ドーナツチャート */
export function buildCostDonutSvg(
  landPrice: number,
  buildingCost: number,
  acquisitionCosts: AcquisitionCostBreakdown,
  width = 200,
  height = 200
): string {
  const segments = [
    { label: "土地", value: landPrice, color: "#3b82f6" },
    { label: "建物", value: buildingCost, color: "#60a5fa" },
    { label: "諸経費", value: acquisitionCosts.total, color: "#f59e0b" },
  ].filter((s) => s.value > 0);

  const total = segments.reduce((sum, s) => sum + s.value, 0);
  if (total === 0)
    return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}"></svg>`;

  const cx = width / 2;
  const cy = height / 2 - 10;
  const r = Math.min(cx, cy) - 10;
  const innerR = r * 0.55;

  function arc(startAngle: number, endAngle: number, color: string): string {
    const s = (Math.PI * 2 * startAngle) / 360 - Math.PI / 2;
    const e = (Math.PI * 2 * endAngle) / 360 - Math.PI / 2;
    const x1 = cx + r * Math.cos(s);
    const y1 = cy + r * Math.sin(s);
    const x2 = cx + r * Math.cos(e);
    const y2 = cy + r * Math.sin(e);
    const ix1 = cx + innerR * Math.cos(s);
    const iy1 = cy + innerR * Math.sin(s);
    const ix2 = cx + innerR * Math.cos(e);
    const iy2 = cy + innerR * Math.sin(e);
    const largeArc = endAngle - startAngle > 180 ? 1 : 0;
    return `<path d="M${x1.toFixed(1)},${y1.toFixed(1)} A${r},${r} 0 ${largeArc},1 ${x2.toFixed(1)},${y2.toFixed(1)} L${ix2.toFixed(1)},${iy2.toFixed(1)} A${innerR},${innerR} 0 ${largeArc},0 ${ix1.toFixed(1)},${iy1.toFixed(1)} Z" fill="${color}"/>`;
  }

  let currentAngle = 0;
  const arcs = segments.map((seg) => {
    const angle = (seg.value / total) * 360;
    const result = arc(currentAngle, currentAngle + angle, seg.color);
    currentAngle += angle;
    return result;
  });

  // 凡例
  const legendItems = segments
    .map(
      (seg, i) =>
        `<rect x="4" y="${(height - 22 + i * 12 - (segments.length - 1) * 6).toFixed(0)}" width="8" height="8" fill="${seg.color}"/>` +
        `<text x="14" y="${(height - 15 + i * 12 - (segments.length - 1) * 6).toFixed(0)}" font-size="7" fill="#64748b">${escapeXml(seg.label)}: ${Math.round((seg.value / total) * 100)}%</text>`
    )
    .join("\n");

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}">
${arcs.join("\n")}
${legendItems}
</svg>`;
}
