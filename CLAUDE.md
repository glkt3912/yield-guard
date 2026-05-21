# CLAUDE.md

## Project overview

Yield-Guard: Japan real estate investment tool. Fetches MLIT land transaction data; visualizes gross yield, dead-cross, exit equity.

- **Backend**: Go 1.25 / Gin / Clean Architecture → Cloud Run (`backend/`)
- **Frontend**: Next.js 16 (App Router) / TypeScript / Tailwind CSS v4 / Shadcn/UI → Vercel (`frontend/`)
- **Data**: 国土交通省 不動産情報ライブラリ API (`reinfolib.mlit.go.jp`) — requires `MLIT_API_KEY`

## Key commands

```bash
make dev            # backend :8080 + frontend :3000
make test           # go test -race ./... + vitest run
make lint           # golangci-lint + eslint + tsc --noEmit
make swagger        # regenerate docs/openapi/swagger.json (run after any Go type change)
make swagger-check  # verify no schema drift
make integration    # real MLIT API tests (needs MLIT_API_KEY)
```

## Environment variables

Copy `.env.example` → `.env`. Key vars: `MLIT_API_KEY`, `APP_INTERNAL_API_KEY`, `ALLOW_ORIGINS`.

## Conventions

- **Branch**: `feature/*` | `fix/*` | `chore/*` | `docs/*` | `refactor/*` | `test/*`
- **Commits**: English, small logical units
- **PR title/body**: Japanese; follow `.github/pull_request_template.md`
- **No `Co-Authored-By`** in commits

## MCP servers

`docs` server reads `docs/` via `.mcp.json` — use `mcp__docs__*` tools for design documents.

## Task-specific entry points

Before starting work, read the relevant entry doc in `docs/llm/`:

| Task | Entry doc |
|------|-----------|
| Backend / API / Go | `docs/llm/backend.md` |
| Frontend / UI / TSX | `docs/llm/frontend.md` |
| Investment calc / domain logic | `docs/llm/domain.md` |
| Branch / worktree / PR | `docs/llm/worktree.md` |

## Claude Code operating rules

### Git
- **Never push to `main` directly** — use a typed branch + PR
- **Never force-push** or skip hooks (`--no-verify`)
- **`git reset --hard` only on explicit instruction**

### Secrets
- **Never read, display, or commit `.env`** — secrets must not appear in output or responses

### Infrastructure
- **`terraform destroy` / `terraform apply`** — always confirm before running
- **Destructive local ops** (`docker volume rm`, `rm -rf`) — confirm before running

### PR and external actions
- **Never merge a PR** without explicit instruction
- **Never post to Issues/PRs** without explicit instruction
- **Never use `--squash`** when merging — use `--merge --delete-branch`
