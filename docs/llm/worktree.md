---
purpose: ブランチ作成・worktree・PR 作業の手順
triggers: [worktree, branch, pr, pull request, rebase, merge]
reads_next: []
last_updated: 2026-05-21
---

## Worktree 作成手順

```bash
# メインリポジトリのルートで実行
git worktree add ../<worktree-dir> -b <branch-name>
ln -s "$(pwd)/frontend/node_modules" "../<worktree-dir>/frontend/node_modules"
```

- `npm install` は worktree 内で**絶対に実行しない**（`package-lock.json` が壊れる）
- `make install-hooks` も不要（メインリポジトリの `.git/hooks/` を自動共有）

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
