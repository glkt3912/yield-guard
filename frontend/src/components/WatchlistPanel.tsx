"use client";

import React, { useEffect, useRef, useState } from "react";
import {
  collection,
  onSnapshot,
  addDoc,
  updateDoc,
  deleteDoc,
  doc,
  serverTimestamp,
  query,
  orderBy,
} from "firebase/firestore";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { AlertDialog } from "@/components/ui/alert-dialog";
import { useToast } from "@/components/ui/toast";
import {
  Trash2,
  ClipboardList,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  BarChart2,
  Search,
  Ban,
  Home,
} from "lucide-react";
import type { WatchlistItem, WatchlistStatus, InvestmentResult } from "@/types/investment";
import WatchlistCompareTable from "@/components/WatchlistCompareTable";
import AuthStatusBadge from "@/components/AuthStatusBadge";
import { useAuthContext } from "@/components/RootLayoutClient";
import { getFirebaseDb } from "@/lib/firebase";

// localStorage keys
const STORAGE_KEY = "yg_watchlist";
const MIGRATION_FLAG_PREFIX = "yg_watchlist_migrated";

const STATUS_OPTIONS: { value: WatchlistStatus; label: string }[] = [
  { value: "検討中", label: "検討中" },
  { value: "見送り", label: "見送り" },
  { value: "購入済み", label: "購入済み" },
];

const STATUS_BADGE_VARIANT: Record<WatchlistStatus, "default" | "warning" | "outline" | "success"> =
  {
    検討中: "default",
    見送り: "outline",
    購入済み: "success",
  };

function StatusIcon({ status }: { status: WatchlistStatus }) {
  if (status === "検討中") return <Search className="h-3 w-3 shrink-0" />;
  if (status === "見送り") return <Ban className="h-3 w-3 shrink-0" />;
  return <Home className="h-3 w-3 shrink-0" />;
}

function loadLocalItems(): WatchlistItem[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    return JSON.parse(raw) as WatchlistItem[];
  } catch {
    return [];
  }
}

function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleDateString("ja-JP", { year: "numeric", month: "2-digit", day: "2-digit" });
}

function getDscrColor(dscr: number): string {
  if (dscr >= 1.2) return "text-green-600";
  if (dscr >= 1.0) return "text-yellow-600";
  return "text-red-600";
}

function DscrIcon({ dscr }: { dscr: number }) {
  if (dscr >= 1.2) return <CheckCircle2 className="h-3 w-3 shrink-0" />;
  if (dscr >= 1.0) return <AlertTriangle className="h-3 w-3 shrink-0" />;
  return <XCircle className="h-3 w-3 shrink-0" />;
}

// Returns the Firestore collection path for a user's watchlist items.
// Returns null when Firebase is not configured (env vars missing).
function itemsCollection(uid: string) {
  const db = getFirebaseDb();
  if (!db) return null;
  return collection(db, "watchlists", uid, "items");
}

function itemDoc(uid: string, itemId: string) {
  const db = getFirebaseDb();
  if (!db) return null;
  return doc(db, "watchlists", uid, "items", itemId);
}

interface WatchlistPanelProps {
  currentResult?: InvestmentResult;
}

export default function WatchlistPanel({ currentResult }: WatchlistPanelProps) {
  const { user, loading: authLoading } = useAuthContext();
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [firestoreLoading, setFirestoreLoading] = useState(() => getFirebaseDb() !== null);
  const [nameInput, setNameInput] = useState("");
  const [memoInput, setMemoInput] = useState("");
  const [nameError, setNameError] = useState("");
  const [pendingDeleteId, setPendingDeleteId] = useState<string | null>(null);
  const [compareMode, setCompareMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const { toast } = useToast();
  const migrationDoneRef = useRef(false);

  // Subscribe to Firestore once the user is available
  useEffect(() => {
    if (authLoading || !user) return;

    const uid = user.uid;
    const col = itemsCollection(uid);
    if (!col) return;
    const q = query(col, orderBy("addedAt", "desc"));

    const unsubscribe = onSnapshot(
      q,
      (snapshot) => {
        const fetched: WatchlistItem[] = snapshot.docs.map((d) => {
          const data = d.data();
          return {
            id: d.id,
            name: data.name as string,
            memo: (data.memo as string) ?? "",
            status: (data.status as WatchlistStatus) ?? "検討中",
            addedAt: data.addedAt as string,
            metrics: data.metrics as WatchlistItem["metrics"] | undefined,
          };
        });
        setItems(fetched);
        setFirestoreLoading(false);

        // One-time migration from localStorage → Firestore
        const migrationKey = `${MIGRATION_FLAG_PREFIX}_${uid}`;
        if (!migrationDoneRef.current && !localStorage.getItem(migrationKey)) {
          migrationDoneRef.current = true;
          const localItems = loadLocalItems();
          if (localItems.length > 0) {
            const existingIds = new Set(fetched.map((item) => item.id));
            const toMigrate = localItems.filter((item) => !existingIds.has(item.id));
            const migrationCol = itemsCollection(uid);
            if (toMigrate.length > 0 && migrationCol) {
              Promise.all(
                toMigrate.map((item) =>
                  addDoc(migrationCol, {
                    name: item.name,
                    memo: item.memo,
                    status: item.status,
                    addedAt: item.addedAt,
                    ...(item.metrics ? { metrics: item.metrics } : {}),
                    migratedAt: serverTimestamp(),
                  })
                )
              )
                .then(() => {
                  localStorage.setItem(migrationKey, uid);
                  localStorage.removeItem(STORAGE_KEY);
                })
                .catch((err) => {
                  console.error("[WatchlistPanel] migration failed:", err);
                });
            } else {
              localStorage.setItem(migrationKey, uid);
            }
          } else {
            localStorage.setItem(migrationKey, uid);
          }
        }
      },
      (err) => {
        console.error("[WatchlistPanel] onSnapshot error:", err);
        setFirestoreLoading(false);
      }
    );

    return () => unsubscribe();
  }, [user, authLoading]);

  async function handleAdd() {
    const trimmed = nameInput.trim();
    if (!trimmed) {
      setNameError("物件名を入力してください");
      return;
    }
    if (!user) return;

    setNameError("");

    const newItem: Omit<WatchlistItem, "id"> = {
      name: trimmed,
      memo: memoInput.trim(),
      status: "検討中",
      addedAt: new Date().toISOString(),
      ...(currentResult
        ? {
            metrics: {
              grossYield: currentResult.grossYield,
              netYield: currentResult.netYield,
              dscr: currentResult.dscr,
              irr: currentResult.irr ?? null,
              totalInvestment: currentResult.totalInvestment,
              exitTotalEquity: currentResult.exitTotalEquity,
              deadCrossYear: currentResult.deadCrossYear,
              npv: currentResult.npv,
            },
          }
        : {}),
    };

    // Optimistic: add a temporary item immediately
    const tempId = `temp_${Date.now()}`;
    setItems((prev) => [{ ...newItem, id: tempId }, ...prev]);
    setNameInput("");
    setMemoInput("");
    toast({ message: `「${trimmed}」をウォッチリストに追加しました`, variant: "success" });

    try {
      const col = itemsCollection(user.uid);
      if (col) {
        await addDoc(col, { ...newItem, createdAt: serverTimestamp() });
      }
      // onSnapshot will replace the optimistic item with the server version
    } catch (err) {
      console.error("[WatchlistPanel] addDoc failed:", err);
      // Roll back optimistic update on error
      setItems((prev) => prev.filter((item) => item.id !== tempId));
      toast({ message: "追加に失敗しました。再度お試しください。", variant: "danger" });
    }
  }

  async function handleStatusChange(id: string, status: WatchlistStatus) {
    if (!user) return;
    // Optimistic update
    setItems((prev) => prev.map((item) => (item.id === id ? { ...item, status } : item)));
    try {
      const d = itemDoc(user.uid, id);
      if (d) await updateDoc(d, { status });
    } catch (err) {
      console.error("[WatchlistPanel] updateDoc failed:", err);
    }
  }

  async function handleDeleteConfirm() {
    if (!pendingDeleteId || !user) return;
    const target = items.find((item) => item.id === pendingDeleteId);
    const deletedId = pendingDeleteId;
    // Optimistic update
    setItems((prev) => prev.filter((item) => item.id !== deletedId));
    setSelectedIds((prev) => prev.filter((id) => id !== deletedId));
    setPendingDeleteId(null);
    toast({
      message: target ? `「${target.name}」を削除しました` : "削除しました",
      variant: "warning",
    });
    try {
      const d = itemDoc(user.uid, deletedId);
      if (d) await deleteDoc(d);
    } catch (err) {
      console.error("[WatchlistPanel] deleteDoc failed:", err);
      // Roll back optimistic delete on error
      if (target) {
        setItems((prev) => {
          const already = prev.some((item) => item.id === target.id);
          return already ? prev : [target, ...prev];
        });
      }
      toast({ message: "削除に失敗しました。再度お試しください。", variant: "danger" });
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") handleAdd();
  }

  function toggleCompareMode() {
    setCompareMode((prev) => {
      if (prev) setSelectedIds([]);
      return !prev;
    });
  }

  function toggleSelectItem(id: string) {
    setSelectedIds((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      if (prev.length >= 4) return prev; // max 4
      return [...prev, id];
    });
  }

  const selectedItems = items.filter((item) => selectedIds.includes(item.id));
  const pendingDeleteName = items.find((item) => item.id === pendingDeleteId)?.name ?? "";
  const isLoading = authLoading || firestoreLoading;

  return (
    <>
      <Card className="rounded-xl shadow-sm">
        <CardHeader className="pb-2">
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <CardTitle className="flex items-center gap-2 text-base font-semibold">
                <ClipboardList className="h-4 w-4 text-primary" />
                物件候補ウォッチリスト
              </CardTitle>
              {items.length >= 2 && !isLoading && (
                <Button
                  type="button"
                  size="sm"
                  variant={compareMode ? "default" : "outline"}
                  onClick={toggleCompareMode}
                  className="flex items-center gap-1.5 text-xs"
                >
                  <BarChart2 className="h-3.5 w-3.5" />
                  {compareMode ? "比較終了" : "比較表示"}
                </Button>
              )}
            </div>
            {/* Auth status */}
            <AuthStatusBadge />
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Loading skeleton */}
          {isLoading ? (
            <div className="space-y-3" aria-label="ウォッチリストを読み込み中">
              <Skeleton className="h-9 w-full" />
              <Skeleton className="h-9 w-full" />
              <div className="divide-y divide-border">
                {[...Array(3)].map((_, i) => (
                  <div key={i} className="flex flex-col gap-2 py-3">
                    <Skeleton className="h-4 w-2/3" />
                    <Skeleton className="h-3 w-1/2" />
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <>
              {/* Add form */}
              <div className="space-y-2">
                <div className="flex gap-2">
                  <div className="flex-1">
                    <input
                      type="text"
                      placeholder="物件名（必須）"
                      value={nameInput}
                      onChange={(e) => {
                        setNameInput(e.target.value);
                        if (nameError) setNameError("");
                      }}
                      onKeyDown={handleKeyDown}
                      className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                      aria-label="物件名"
                    />
                    {nameError && <p className="mt-0.5 text-xs text-destructive">{nameError}</p>}
                  </div>
                  <Button
                    type="button"
                    size="sm"
                    onClick={handleAdd}
                    className="shrink-0"
                    data-testid="watchlist-add-button"
                  >
                    追加
                  </Button>
                </div>
                <input
                  type="text"
                  placeholder="メモ（任意）"
                  value={memoInput}
                  onChange={(e) => setMemoInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  aria-label="メモ"
                />
              </div>

              {/* Compare mode hint */}
              {compareMode && (
                <p className="rounded-md bg-blue-50 px-3 py-2 text-xs text-blue-700">
                  比較する物件を最大4件選択してください（{selectedIds.length}/4 件選択中）
                </p>
              )}

              {/* List */}
              {items.length === 0 ? (
                <p className="py-6 text-center text-sm text-muted-foreground">
                  まだ物件が登録されていません
                </p>
              ) : (
                <ul className="divide-y divide-border">
                  {items.map((item) => (
                    <li
                      key={item.id}
                      className="flex flex-col gap-2 py-3 sm:flex-row sm:items-start sm:gap-3"
                    >
                      {/* Compare mode checkbox */}
                      {compareMode && (
                        <div className="flex shrink-0 items-center pt-0.5">
                          <input
                            type="checkbox"
                            id={`compare-${item.id}`}
                            checked={selectedIds.includes(item.id)}
                            onChange={() => toggleSelectItem(item.id)}
                            disabled={!selectedIds.includes(item.id) && selectedIds.length >= 4}
                            className="h-4 w-4 rounded border-input accent-primary"
                            aria-label={`${item.name} を比較対象に追加`}
                          />
                        </div>
                      )}
                      {/* Left: name + meta */}
                      <div className="min-w-0 flex-1 space-y-0.5">
                        <div className="flex flex-wrap items-center gap-2">
                          <span
                            className="truncate text-sm font-medium"
                            data-testid="watchlist-item-name"
                          >
                            {item.name}
                          </span>
                          <Badge
                            variant={STATUS_BADGE_VARIANT[item.status]}
                            className="flex shrink-0 items-center gap-1"
                          >
                            <StatusIcon status={item.status} />
                            {item.status}
                          </Badge>
                        </div>
                        {item.metrics && (
                          <div className="flex flex-wrap gap-2 pt-0.5">
                            <span className="text-xs text-blue-600">
                              表面利回り: {(item.metrics.grossYield * 100).toFixed(1)}%
                            </span>
                            <span
                              className={`inline-flex items-center gap-1 text-xs ${getDscrColor(item.metrics.dscr)}`}
                            >
                              <DscrIcon dscr={item.metrics.dscr} />
                              DSCR: {item.metrics.dscr.toFixed(2)}
                            </span>
                            <span className="text-xs text-purple-600">
                              IRR:{" "}
                              {item.metrics.irr !== null
                                ? `${(item.metrics.irr * 100).toFixed(1)}%`
                                : "-"}
                            </span>
                          </div>
                        )}
                        {item.memo && (
                          <p className="truncate text-xs text-muted-foreground">{item.memo}</p>
                        )}
                        <p className="text-xs text-muted-foreground">
                          登録日: {formatDate(item.addedAt)}
                        </p>
                      </div>

                      {/* Right: status selector + delete */}
                      <div className="flex shrink-0 items-center gap-2">
                        <select
                          value={item.status}
                          onChange={(e) =>
                            handleStatusChange(item.id, e.target.value as WatchlistStatus)
                          }
                          className="h-8 rounded-md border border-input bg-background px-2 text-xs focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                          aria-label={`${item.name} のステータスを変更`}
                        >
                          {STATUS_OPTIONS.map((o) => (
                            <option key={o.value} value={o.value}>
                              {o.label}
                            </option>
                          ))}
                        </select>
                        <button
                          type="button"
                          onClick={() => setPendingDeleteId(item.id)}
                          className="flex h-8 w-8 items-center justify-center rounded-md text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                          aria-label={`${item.name} を削除`}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}
        </CardContent>
      </Card>

      {/* Comparison table */}
      {compareMode && selectedItems.length >= 2 && <WatchlistCompareTable items={selectedItems} />}

      <AlertDialog
        open={pendingDeleteId !== null}
        onClose={() => setPendingDeleteId(null)}
        title="削除の確認"
        description={`「${pendingDeleteName}」をウォッチリストから削除しますか？`}
        confirmLabel="削除"
        onConfirm={handleDeleteConfirm}
      />
    </>
  );
}
