---
purpose: フロントエンドのホスティング基盤 Cloudflare Workers の仕組み・設定・運用知見・ハマりどころ
triggers: [cloudflare, workers, wrangler, wrangler.jsonc, worker/index.ts, deploy, preview, HTMLRewriter, 1027]
audience: frontend-dev
token_weight: medium
reads_next: [docs/llm/frontend.md, docs/security.md]
---

# Cloudflare Workers

フロントエンドは Vite でビルドした SPA を Cloudflare Workers で配信している（2026-10 に Next.js / Vercel から移行）。
初回セットアップ手順は `docs/overview.md`「デプロイフロー」を参照。

## Cloudflare Workers とは

Cloudflare の世界 300 拠点以上のエッジで JS/TS を実行するサーバーレス基盤。

| | Cloudflare Workers | Vercel (Next.js) | Cloud Run |
|---|---|---|---|
| 実行環境 | V8 isolate（Chrome の JS エンジンを小さく区切った環境） | Node.js | コンテナ |
| コールドスタート | ほぼ無し（数 ms） | 数百 ms〜 | 秒単位もあり |
| 実行場所 | ユーザーに最も近い拠点 | 指定リージョン | 指定リージョン |

- ランタイムは Node.js ではなく `workerd`。`fetch` / `Request` / `Response` などの Web 標準 API が中心で、Node API は一部のみ
- 重い処理は置かない（無料プランは CPU 10 ms / リクエスト）。計算・外部 API 集約は従来どおり Go バックエンド（Cloud Run）

## このプロジェクトでの構成

```
ブラウザ ─┬─ /assets/*, /icons/*, /fonts/* など ─→ 静的アセット（Worker を通らない・無料）
          ├─ /api/*  ─→ worker/index.ts ─(X-Internal-Key 付与)─→ Cloud Run (Go)
          └─ /       ─→ worker/index.ts ─→ 共有 URL なら HTMLRewriter で OG タグを書き換え
```

| ファイル | 役割 |
|---|---|
| `frontend/wrangler.jsonc` | Worker 設定（アセット・ルーティング・環境） |
| `frontend/worker/index.ts` | `/api/*` プロキシ（旧 `middleware.ts`）と OG 書き換え（旧 `generateMetadata`） |
| `frontend/worker/tsconfig.json` | Worker 用の型設定（`@cloudflare/workers-types`）。`npm run type-check` で一緒に検査される |
| `frontend/vite.config.ts` | ローカル開発（`vite` / `vite preview`）用の `/api` プロキシ |

### wrangler.jsonc の要点

| 設定 | 意味 |
|---|---|
| `assets.directory: "./dist"` | `vite build` の出力をそのまま静的アセットとして配信 |
| `not_found_handling: "single-page-application"` | 存在しないパスには `index.html` を 200 で返す |
| `run_worker_first: ["/api/*", "/"]` | この 2 パスだけ Worker コードを先に通す。それ以外は Worker を起動せず配信（課金対象外） |
| `vars.BACKEND_URL` | 既定値は `wrangler dev` 用。CI は `--var BACKEND_URL:...` で上書き |
| `env.stg` | 旧 PR プレビュー用の別 Worker `yield-guard-stg`（CI からは使っていない） |

### 環境と URL

| 環境 | Worker | URL |
|---|---|---|
| 本番 | `yield-guard` | `https://yield-guard.glkt3912.workers.dev` |

## 設定値の置き場所

| 値 | 置き場所 | 設定方法 |
|---|---|---|
| `APP_INTERNAL_API_KEY` | Worker secret（暗号化） | 本番: `npx wrangler secret put APP_INTERNAL_API_KEY` / stg: `npx wrangler versions secret put APP_INTERNAL_API_KEY --env stg` |
| `BACKEND_URL` | Worker var（CI がデプロイ時に指定） | GitHub Variables `BACKEND_URL_PROD` / `BACKEND_URL_STG` |
| `VITE_FIREBASE_*` | ビルド時にバンドルへ埋め込み（公開値） | GitHub Variables |
| `CLOUDFLARE_API_TOKEN` / `CLOUDFLARE_ACCOUNT_ID` | CI 用 | GitHub Environment `production` の Secrets（main 限定。トークンは「Edit Cloudflare Workers」テンプレートで発行） |

- 本番の secret は `wrangler secret put` した瞬間に新バージョンとしてデプロイされる（再デプロイ不要）
- `VITE_` で始まる変数はブラウザに配られる。秘密の値を `VITE_` にしてはいけない

## デプロイ

`.github/workflows/frontend-ci.yml` の `deploy` ジョブ（CI・E2E 通過後）。

| トリガー | コマンド | 結果 |
|---|---|---|
| main push | `wrangler deploy --var BACKEND_URL:$BACKEND_URL_PROD` | 本番に即時反映 |

- PR プレビューは廃止（`CLOUDFLARE_API_TOKEN` を main 以外のブランチに渡さないため）。動作確認はローカルの `make dev` で行う
- ロールバック: ダッシュボードの Deployments から、または `npx wrangler rollback`

## ローカル開発

| コマンド | 用途 |
|---|---|
| `npm run dev`（`make dev`） | 普段の開発。Vite が `/api` を `BACKEND_URL`（既定 `localhost:8080`）へプロキシ。Worker は通らない |
| `npm run build && npx wrangler dev` | Worker の動作確認（OG 書き換え・`X-Internal-Key` 付与）。`--var APP_INTERNAL_API_KEY:xxx` で鍵も試せる |

## 料金と上限（無料プラン）

| 項目 | 上限 | このアプリ |
|---|---|---|
| Worker を通るリクエスト | 10 万件 / 日（UTC 0 時リセット） | `/` 表示と `/api/*` だけが対象 |
| 静的アセット | 無料・無制限 | `run_worker_first` 以外は全てこちら |
| CPU 時間 | 10 ms / リクエスト | プロキシの待ち時間は数えない。OG 書き換えも軽量 |
| サブリクエスト | 50 / 実行 | 1（Cloud Run へ） |
| アセット | 2 万ファイル・1 ファイル 25 MiB | 約 20 ファイル・合計 約 9 MB |

- 上限超過時はその日の残り `/` と `/api` が **Error 1027** になる。stg（プレビュー）も同じアカウント枠を使う
- 利用量はダッシュボードの Workers & Pages → Metrics で確認。10 万件に近づいたら Workers Paid（月 5 ドル程度）を検討

## ハマりどころ（移行時に得た知見）

- **存在しないアセットにも `index.html` が 200 で返る**: SPA フォールバックのため、デプロイ前に開いたタブが旧チャンク（`/assets/xxx-<hash>.js`）を読むと HTML が返る。
  対策として `public/sw.js` は `text/html` をアセットとしてキャッシュせず、`src/lib/dynamic.tsx` は読み込み失敗時に 1 回だけ再読み込みする
- **`wrangler secret put` は既存の Worker にしか使えない**: 初回は先に `wrangler deploy` が必要。その間 `/api` はキー無しで 401 になる（公開前なら問題なし）
- **workers.dev サブドメインはダッシュボードを一度開くまで存在しない**: API で `code 10007` が返ったら Workers & Pages を開く
- **HTMLRewriter は `after()` などで挿入した HTML を再処理しない**: 挿入するタグは最終的な値で書く。`setInnerContent` / `setAttribute` は自動でエスケープされるが、`{ html: true }` で挿入する文字列は自分でエスケープする（現状は数値由来の固定文言のみ）
- **CORS**: ブラウザは同一オリジンでも POST に `Origin` を付け、Worker はそれを Cloud Run へ転送する。Cloud Run の `ALLOW_ORIGINS`（Terraform `frontend_url`）に workers.dev の URL が無いと 403 になる。プレビューは `*.glkt3912.workers.dev` の suffix 一致で許可
- **プロキシ時の `Host` ヘッダー**: `new Request(url, request)` は元のヘッダーを全部コピーするので `host` を削除してから転送する
- **PR プレビューは誰でも開ける**: Vercel のような標準のログイン保護は無い。内部キー付きで stg バックエンドに繋がるため、必要なら Cloudflare Access で保護する（未対応）
- **レート制限の送信元 IP**: バックエンドは `X-Forwarded-For` の末尾（＝Cloudflare のエッジ IP）で制限するため、利用者間で枠を共有しやすい。`CF-Connecting-IP` を渡す改善は未対応
- **`!` で実行した `gh secret set` は値の入力を待たずに空で登録する**: `pbpaste | gh secret set NAME` か GitHub の GUI で設定する

## トラブルシューティング

| 症状 | 原因と対処 |
|---|---|
| `/api/*` が 401 | Worker secret `APP_INTERNAL_API_KEY` 未設定、または Cloud Run 側と値が不一致 |
| `/api/*` が 403 | `ALLOW_ORIGINS` に今の URL が無い → `FRONTEND_URL*` を更新して Terraform apply |
| `/api/*` が 502 `BACKEND_URL is not configured` | デプロイ時の `--var BACKEND_URL` が空（GitHub Variables 未設定） |
| Error 1027 | 無料プランの 1 日上限超過。翌 UTC 0 時に復旧、または有料プランへ |
| CI で `Preview alias URL not found` | stg Worker が未作成、または workers.dev が無効。`npx wrangler deploy --env stg --var "BACKEND_URL:<stg Cloud Run URL>"` を一度実行（vars は env に継承されないため `--var` 必須） |
| 画面は出るがログインできない | Firebase Authentication の承認済みドメインに workers.dev のドメインが無い |
