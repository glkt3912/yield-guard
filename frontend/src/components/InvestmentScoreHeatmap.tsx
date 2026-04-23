"use client";

import { useCallback, useState } from "react";
import { MapContainer, TileLayer, Rectangle, Tooltip, useMapEvents } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import { fetchInvestmentScoreHeatmap } from "@/lib/api";
import type { HeatmapTile } from "@/types/investment";

function scoreToColor(score: number): string {
  if (score >= 80) return "#22c55e";   // green-500
  if (score >= 65) return "#86efac";   // green-300
  if (score >= 50) return "#fde68a";   // yellow-200
  if (score >= 35) return "#fdba74";   // orange-300
  return "#f87171";                     // red-400
}

// Tile bounds: the rectangle a tile covers (not just its center)
function tileBounds(x: number, y: number, z: number): [[number, number], [number, number]] {
  const n = Math.pow(2, z);
  const lng1 = (x / n) * 360 - 180;
  const lng2 = ((x + 1) / n) * 360 - 180;
  const lat1 = (Math.atan(Math.sinh(Math.PI * (1 - (2 * y) / n))) * 180) / Math.PI;
  const lat2 = (Math.atan(Math.sinh(Math.PI * (1 - (2 * (y + 1)) / n))) * 180) / Math.PI;
  return [[lat2, lng1], [lat1, lng2]];
}

function AnalyzeButton({ onAnalyze, loading }: { onAnalyze: (bounds: { minLat: number; maxLat: number; minLng: number; maxLng: number }) => void; loading: boolean }) {
  const map = useMapEvents({});
  const handleClick = useCallback(() => {
    const b = map.getBounds();
    onAnalyze({
      minLat: b.getSouth(),
      maxLat: b.getNorth(),
      minLng: b.getWest(),
      maxLng: b.getEast(),
    });
  }, [map, onAnalyze]);

  return (
    <div className="leaflet-top leaflet-right" style={{ zIndex: 1000, marginTop: 10, marginRight: 10 }}>
      <div className="leaflet-control">
        <button
          onClick={handleClick}
          disabled={loading}
          className="px-3 py-2 bg-blue-600 text-white text-sm rounded shadow hover:bg-blue-700 disabled:opacity-50"
        >
          {loading ? "取得中..." : "このエリアを分析"}
        </button>
      </div>
    </div>
  );
}

interface Props {
  centerLat?: number;
  centerLng?: number;
}

export default function InvestmentScoreHeatmap({ centerLat = 35.6812, centerLng = 139.7671 }: Props) {
  const [tiles, setTiles] = useState<HeatmapTile[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleAnalyze = useCallback(async (bounds: { minLat: number; maxLat: number; minLng: number; maxLng: number }) => {
    setLoading(true);
    setError(null);
    try {
      const data = await fetchInvestmentScoreHeatmap({ ...bounds, z: 13 });
      setTiles(data.tiles);
    } catch (e) {
      setError(e instanceof Error ? e.message : "取得失敗");
    } finally {
      setLoading(false);
    }
  }, []);

  return (
    <div className="rounded-lg border bg-card p-4">
      <h3 className="text-lg font-semibold mb-3">エリア別投資スコア</h3>
      {error && <p className="text-red-500 text-sm mb-2">{error}</p>}
      <div style={{ height: 480 }}>
        <MapContainer
          center={[centerLat, centerLng]}
          zoom={13}
          style={{ height: "100%", width: "100%" }}
        >
          <TileLayer
            attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
          />
          <AnalyzeButton onAnalyze={handleAnalyze} loading={loading} />
          {tiles.map((tile) => (
            <Rectangle
              key={`${tile.z}-${tile.x}-${tile.y}`}
              bounds={tileBounds(tile.x, tile.y, tile.z)}
              pathOptions={{
                color: scoreToColor(tile.totalScore),
                fillColor: scoreToColor(tile.totalScore),
                fillOpacity: 0.4,
                weight: 0.5,
              }}
            >
              <Tooltip sticky>
                {tile.grade} / スコア {tile.totalScore}
              </Tooltip>
            </Rectangle>
          ))}
        </MapContainer>
      </div>
      {tiles.length > 0 && (
        <div className="mt-2 flex gap-4 text-xs text-muted-foreground flex-wrap">
          {[
            { label: "優良 (80+)", color: "#22c55e" },
            { label: "良好 (65-79)", color: "#86efac" },
            { label: "普通 (50-64)", color: "#fde68a" },
            { label: "注意 (35-49)", color: "#fdba74" },
            { label: "要注意 (<35)", color: "#f87171" },
          ].map(({ label, color }) => (
            <span key={label} className="flex items-center gap-1">
              <span style={{ background: color, width: 12, height: 12, display: "inline-block", borderRadius: 2 }} />
              {label}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
