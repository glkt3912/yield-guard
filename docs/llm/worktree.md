---
purpose: ブランチ作成・worktree・PR 作業の手順
triggers: [worktree, branch, pr, pull request, rebase, merge]
reads_next: []
last_updated: 2026-05-26
---

## 初回セットアップ（clone 後1回）

```bash
make install          # frontend npm install（メインリポジトリのみ・worktree では不要）
make install-hooks    # lefthook の pre-commit / pre-push フック登録
```

> `make install-hooks` が失敗する場合: `git config --unset-all --local core.hooksPath` を実行してから再試行する。

## Worktree 作成手順

```bash
# メインリポジトリのルートで実行
git worktree add ../<worktree-dir> -b <branch-name>
cd ../<worktree-dir>/frontend && npm ci
```

- `npm ci` は `package-lock.json` を**絶対に変更しない**（lockfile と不一致ならエラーで停止する）
- `npm install` は**実行しない**（`package-lock.json` を書き換える可能性がある）
- `make install-hooks` も不要（メインリポジトリの `.git/hooks/` を自動共有）

> **symlink を使わない理由**: Next.js 16 の Turbopack はプロジェクトルート外を指す symlink を拒否するため、
> `ln -s` で `node_modules` を共有すると `next dev` が起動できない。`npm ci` で実ディレクトリとして持つ。

## ブラウザ動作確認（E2E）をする場合

worktree のフロントエンドを起動するときは**メインリポジトリと別ポート**を使う。

```bash
cd ../<worktree-dir>/frontend && npx next dev -p 3003
```

バックエンドの `ALLOW_ORIGINS` はポートごとに許可が必要なため、`.env` に worktree のポートを追加する。

```bash
# .env（メインリポジトリ）
ALLOW_ORIGINS=http://localhost:3002,http://localhost:3003
```

追加後はバックエンドを再起動する。

## ブランチ命名

```
feature/<content>   新機能
fix/<content>       バグ修正
chore/<content>     依存更新・設定変更
docs/<content>      ドキュメントのみ
refactor/<content>  動作変更なしのリファクタ
test/<content>      テスト追加・修正
```

## PR 作成前チェックリスト

- [ ] `make lint` が通る
- [ ] `make test` が通る
- [ ] Go 型変更がある場合 → `make swagger` 実行 + `swagger.json` をコミットに含める
- [ ] `git rebase origin/main` を**プッシュ直前に1回だけ**実行
- [ ] PR title: 日本語、PR body: `.github/pull_request_template.md` に従う

## マージ方法

```bash
# squash は禁止。必ず --merge を使う
gh pr merge <PR番号> --merge --delete-branch
# ローカル worktree は手動で削除
git worktree remove ../<worktree-dir>
```

## PR 作業時の注意

- `main` への直接プッシュ禁止
- `--force` プッシュ禁止
- コミットに `Co-Authored-By:` 行を追加しない
