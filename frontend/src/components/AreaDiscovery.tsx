"use client";
import React, { useState } from "react";
import { fetchAreaDiscovery, type AreaDiscoveryItem } from "@/lib/api";

const PREFECTURES = [
  { value: "01", label: "北海道" },
  { value: "02", label: "青森県" },
  { value: "03", label: "岩手県" },
  { value: "04", label: "宮城県" },
  { value: "05", label: "秋田県" },
  { value: "06", label: "山形県" },
  { value: "07", label: "福島県" },
  { value: "08", label: "茨城県" },
  { value: "09", label: "栃木県" },
  { value: "10", label: "群馬県" },
  { value: "11", label: "埼玉県" },
  { value: "12", label: "千葉県" },
  { value: "13", label: "東京都" },
  { value: "14", label: "神奈川県" },
  { value: "15", label: "新潟県" },
  { value: "16", label: "富山県" },
  { value: "17", label: "石川県" },
  { value: "18", label: "福井県" },
  { value: "19", label: "山梨県" },
  { value: "20", label: "長野県" },
  { value: "21", label: "岐阜県" },
  { value: "22", label: "静岡県" },
  { value: "23", label: "愛知県" },
  { value: "24", label: "三重県" },
  { value: "25", label: "滋賀県" },
  { value: "26", label: "京都府" },
  { value: "27", label: "大阪府" },
  { value: "28", label: "兵庫県" },
  { value: "29", label: "奈良県" },
  { value: "30", label: "和歌山県" },
  { value: "31", label: "鳥取県" },
  { value: "32", label: "島根県" },
  { value: "33", label: "岡山県" },
  { value: "34", label: "広島県" },
  { value: "35", label: "山口県" },
  { value: "36", label: "徳島県" },
  { value: "37", label: "香川県" },
  { value: "38", label: "愛媛県" },
  { value: "39", label: "高知県" },
  { value: "40", label: "福岡県" },
  { value: "41", label: "佐賀県" },
  { value: "42", label: "長崎県" },
  { value: "43", label: "熊本県" },
  { value: "44", label: "大分県" },
  { value: "45", label: "宮崎県" },
  { value: "46", label: "鹿児島県" },
  { value: "47", label: "沖縄県" },
];

interface Props {
  onMunicipalitySelect?: (
    municipalityCode: string,
    municipalityName: string,
    prefecture: string,
    centerLat: number,
    centerLng: number
  ) => void;
}

function DifficultyBadge({ difficulty, label }: { difficulty: string; label: string }) {
  const colorMap: Record<string, string> = {
    achievable: "bg-green-100 text-green-800 border border-green-200",
    "slightly-difficult": "bg-yellow-100 text-yellow-800 border border-yellow-200",
    difficult: "bg-red-100 text-red-800 border border-red-200",
    unknown: "bg-gray-100 text-gray-600 border border-gray-200",
  };
  const cls = colorMap[difficulty] ?? "bg-muted text-foreground border";
  return (
    <span className={`inline-block rounded-full px-2 py-0.5 text-xs font-medium ${cls}`}>
      {label}
    </span>
  );
}

export function AreaDiscovery({ onMunicipalitySelect }: Props) {
  const [prefecture, setPrefecture] = useState("13");
  const [budgetMan, setBudgetMan] = useState("");
  const [yieldPct, setYieldPct] = useState("8");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [items, setItems] = useState<AreaDiscoveryItem[] | null>(null);

  const handleSearch = async () => {
    setLoading(true);
    setError(null);
    setItems(null);
    try {
      const params: Parameters<typeof fetchAreaDiscovery>[0] = { prefecture };
      if (budgetMan !== "") {
        const v = parseFloat(budgetMan);
        if (!isNaN(v) && v > 0) params.budget = v * 10000;
      }
      if (yieldPct !== "") {
        const v = parseFloat(yieldPct);
        if (!isNaN(v) && v > 0) params.yield = v / 100;
      }
      const res = await fetchAreaDiscovery(params);
      setItems(res.items);
    } catch (e) {
      setError(e instanceof Error ? e.message : "エリアデータの取得に失敗しました");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="rounded-xl border bg-card p-5 shadow-sm space-y-4">
      <h2 className="text-base font-semibold text-foreground">エリアを探す</h2>
      <p className="text-xs text-muted-foreground">
        予算と目標利回りを入力して、候補エリアをランキング表示します。
      </p>

      {/* Form */}
      <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-foreground">都道府県</label>
          <select
            value={prefecture}
            onChange={(e) => setPrefecture(e.target.value)}
            className="rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          >
            {PREFECTURES.map((p) => (
              <option key={p.value} value={p.value}>
                {p.label}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-foreground">予算（万円）</label>
          <input
            type="number"
            placeholder="例: 3000"
            value={budgetMan}
            onChange={(e) => setBudgetMan(e.target.value)}
            className="rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
            min={0}
          />
        </div>

        <div className="flex flex-col gap-1">
          <label className="text-xs font-medium text-foreground">目標利回り（%）</label>
          <input
            type="number"
            placeholder="例: 8"
            value={yieldPct}
            onChange={(e) => setYieldPct(e.target.value)}
            step="0.1"
            min={0}
            max={50}
            className="rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-ring"
          />
        </div>
      </div>

      <button
        onClick={handleSearch}
        disabled={loading}
        className="flex min-h-[40px] items-center gap-2 rounded-md bg-primary px-5 py-2 text-sm font-semibold text-white shadow-sm hover:bg-primary/90 disabled:opacity-60 transition-colors"
      >
        {loading ? "検索中..." : "エリアを探す"}
      </button>

      {error && (
        <div className="rounded-md border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800">
          {error}
        </div>
      )}

      {/* Results */}
      {items && items.length === 0 && (
        <p className="text-sm text-muted-foreground">
          該当するエリアのデータが見つかりませんでした。
        </p>
      )}

      {items && items.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b bg-muted/40 text-xs font-medium text-muted-foreground">
                <th className="px-3 py-2 text-left">市区町村</th>
                <th className="px-3 py-2 text-right">坪単価中央値</th>
                <th className="px-3 py-2 text-right">取引件数</th>
                <th className="px-3 py-2 text-center">利回り達成難易度</th>
                <th className="px-3 py-2 text-center">土地価格トレンド</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr
                  key={item.municipalityCode}
                  onClick={() =>
                    onMunicipalitySelect?.(
                      item.municipalityCode,
                      item.municipalityName,
                      prefecture,
                      item.centerLat,
                      item.centerLng
                    )
                  }
                  className="cursor-pointer border-b hover:bg-muted/30 transition-colors"
                >
                  <td className="px-3 py-2 font-medium text-foreground">
                    {item.municipalityName}
                    {!item.dataSufficient && (
                      <span className="ml-1 text-xs text-muted-foreground">（データ少）</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-right text-foreground">
                    {item.medianTsubo > 0
                      ? `${Math.round(item.medianTsubo / 10000).toLocaleString()}万円`
                      : "—"}
                  </td>
                  <td className="px-3 py-2 text-right text-foreground">{item.transactionCount}</td>
                  <td className="px-3 py-2 text-center">
                    <DifficultyBadge
                      difficulty={item.yieldDifficulty}
                      label={item.yieldDifficultyLabel}
                    />
                  </td>
                  <td className="px-3 py-2 text-center text-foreground">{item.landPriceTrend}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {items && items.length > 0 && (
        <p className="text-xs text-muted-foreground">
          市区町村を選択すると地図が表示されます。地図上で物件位置をクリックすると投資スコアが取得できます。
        </p>
      )}
    </div>
  );
}
