"use client";

import React, { useState } from "react";
import { UserRound, Mail, CheckCircle2, AlertCircle, Loader2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAuthContext } from "@/components/RootLayoutClient";

export default function AuthStatusBadge() {
  const { user, loading, linkWithEmail } = useAuthContext();
  const [expanded, setExpanded] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  if (loading) {
    return (
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground" aria-live="polite">
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
        <span>認証中...</span>
      </div>
    );
  }

  // Permanent account (non-anonymous or already linked)
  const isLinked = user && !user.isAnonymous;
  if (isLinked) {
    return (
      <div className="flex items-center gap-1.5">
        <Badge variant="success" className="flex items-center gap-1">
          <CheckCircle2 className="h-3 w-3" aria-hidden="true" />
          <span>同期済み</span>
        </Badge>
        <span className="text-xs text-muted-foreground">{user.email}</span>
      </div>
    );
  }

  // Anonymous user — show upgrade prompt
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Badge variant="warning" className="flex items-center gap-1">
          <UserRound className="h-3 w-3" aria-hidden="true" />
          <span>匿名ログイン中</span>
        </Badge>
        <button
          type="button"
          onClick={() => setExpanded((prev) => !prev)}
          className="text-xs text-primary underline underline-offset-2 hover:text-primary/80"
          aria-expanded={expanded}
          aria-controls="email-link-form"
        >
          {expanded ? "閉じる" : "メール登録でデバイス間同期"}
        </button>
      </div>

      {expanded && (
        <div
          id="email-link-form"
          className="rounded-lg border border-border bg-card p-3 space-y-2 shadow-sm"
          role="region"
          aria-label="メールアドレス登録フォーム"
        >
          {success ? (
            <div className="flex items-center gap-2 text-sm text-green-700">
              <CheckCircle2 className="h-4 w-4 shrink-0" aria-hidden="true" />
              <span>
                メールアドレスを登録しました。複数デバイスでウォッチリストが共有されます。
              </span>
            </div>
          ) : (
            <>
              <p className="text-xs text-muted-foreground">
                メールアドレスを登録するとウォッチリストが複数デバイスで同期されます。
              </p>
              <div className="space-y-1.5">
                <label htmlFor="auth-email" className="text-xs font-medium text-foreground">
                  メールアドレス
                </label>
                <div className="flex items-center gap-1.5 rounded-md border border-input bg-background px-2">
                  <Mail className="h-3.5 w-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
                  <input
                    id="auth-email"
                    type="email"
                    value={email}
                    onChange={(e) => {
                      setEmail(e.target.value);
                      setError(null);
                    }}
                    placeholder="example@email.com"
                    autoComplete="email"
                    className="flex-1 bg-transparent py-1.5 text-xs placeholder:text-muted-foreground focus:outline-none"
                    aria-required="true"
                  />
                </div>
              </div>
              <div className="space-y-1.5">
                <label htmlFor="auth-password" className="text-xs font-medium text-foreground">
                  パスワード（6文字以上）
                </label>
                <input
                  id="auth-password"
                  type="password"
                  value={password}
                  onChange={(e) => {
                    setPassword(e.target.value);
                    setError(null);
                  }}
                  placeholder="パスワード"
                  autoComplete="new-password"
                  className="flex h-8 w-full rounded-md border border-input bg-background px-3 text-xs placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  aria-required="true"
                />
              </div>

              {error && (
                <div className="flex items-center gap-1.5 text-xs text-destructive" role="alert">
                  <AlertCircle className="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
                  <span>{error}</span>
                </div>
              )}

              <Button
                type="button"
                size="sm"
                className="w-full text-xs"
                disabled={submitting || !email || !password}
                onClick={async () => {
                  setSubmitting(true);
                  setError(null);
                  try {
                    await linkWithEmail(email, password);
                    setSuccess(true);
                  } catch (err: unknown) {
                    const msg =
                      err instanceof Error
                        ? err.message
                        : "登録に失敗しました。再度お試しください。";
                    setError(msg);
                  } finally {
                    setSubmitting(false);
                  }
                }}
                aria-label="メールアドレスを登録してデバイス間同期を有効化する"
              >
                {submitting ? (
                  <span className="flex items-center gap-1.5">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                    登録中...
                  </span>
                ) : (
                  "登録してデバイス間同期を有効化"
                )}
              </Button>
            </>
          )}
        </div>
      )}
    </div>
  );
}
