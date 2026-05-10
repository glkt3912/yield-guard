# CLAUDE.md

## Project overview

Yield-Guard is a real estate investment decision-support tool that fetches land transaction data from Japan's MLIT (国土交通省) API and visualizes gross yield, dead-cross timing, and exit strategy equity in real time.

## Stack

- **Backend**: Go 1.25 / Gin / Clean Architecture → Cloud Run (`backend/`)
- **Frontend**: Next.js 16 (App Router) / TypeScript / Tailwind CSS v4 / Shadcn/UI / Recharts → Vercel (`frontend/`)
- **Data**: 国土交通省 不動産情報ライブラリ API (`reinfolib.mlit.go.jp`) — requires `MLIT_API_KEY`

## Key commands

```bash
make dev          # backend :8080 + frontend :3000 (local, no Docker)
make docker-up    # Docker Compose build & start
make docker-down  # stop containers
make test         # go test -race ./... + vitest run
make lint         # golangci-lint + eslint + tsc --noEmit
make build        # go build + next build
make integration  # integration tests against real MLIT API (needs MLIT_API_KEY)
```

**Backend only:**
```bash
cd backend && go run ./cmd/server          # :8080
cd backend && go test -race ./... -timeout 120s
```

**Frontend only:**
```bash
cd frontend && npm run dev                 # :3000
cd frontend && npm test
```

**MLIT API debug:**
```bash
make mlit-land-prices area=13 year=2024 quarter=1 to_year=2024 to_quarter=4
make mlit-municipalities area=13
```

## Environment variables

Copy `.env.example` → `.env` (project root).
- `MLIT_API_KEY` — required for all MLIT data endpoints
- `APP_INTERNAL_API_KEY` — Vercel→Cloud Run auth (omit for local dev)
- `ALLOW_ORIGINS` — comma-separated CORS origins (default: `http://localhost:3000`)

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check |
| `GET` | `/api/land-prices/stats` | Land transaction stats |
| `GET` | `/api/land-prices/compare` | Compare vs. market price |
| `GET` | `/api/land-prices/estimate` | Theoretical price estimate |
| `POST` | `/api/investment/analyze` | Investment simulation |
| `GET` | `/api/investment/rent-decline-hint` | Rent decline rate hint from land appraisals (area, municipality) |
| `GET` | `/api/municipalities` | Municipality list |
| `GET` | `/api/station-ridership` | Station ridership (tile coords) |
| `GET` | `/api/population-forecast` | Population forecast (tile coords) |
| `GET` | `/api/land-appraisals` | Official land appraisals |
| `GET` | `/api/urban-risks` | Urban planning risks (tile coords) |
| `GET` | `/api/hazard` | Hazard info: flood/storm/tsunami/landslide (tile coords) |
| `GET` | `/api/investment-score` | Investment suitability score (tile coords) |
| `GET` | `/api/investment-score-heatmap` | Batch investment scores for viewport bbox (minLat, maxLat, minLng, maxLng, z=11-15) |
| `GET` | `/api/rent-stats` | Area rent market stats: median/average/count from MLIT rental data (area, municipality, area_sqm) |

## Directory structure

```
yield-guard/
├── backend/
│   ├── cmd/server/main.go          # entry point
│   └── internal/
│       ├── domain/
│       │   ├── types.go            # domain models
│       │   ├── investment.go       # yield / dead-cross / exit calc logic
│       │   └── investment_test.go
│       ├── mlit/
│       │   ├── client.go           # MLIT API client (retry + cache)
│       │   ├── cache.go            # in-memory TTL cache (24h)
│       │   └── types.go            # API response types
│       └── api/
│           ├── handler.go          # HTTP handlers
│           ├── rent_handler.go     # GET /api/rent-stats handler
│           └── router.go           # Gin router + CORS + auth middleware
├── frontend/src/
│   ├── app/                        # Next.js App Router pages
│   ├── components/                 # UI components (see below)
│   ├── lib/                        # API client, calc utilities
│   └── types/                      # TypeScript types
├── docs/                           # Design docs (read via docs MCP)
├── terraform/                      # Cloud Run / infra
├── docker-compose.yml
├── Makefile
└── .env.example
```

**Key frontend components:**
- `Dashboard.tsx` — top-level layout
- `InvestmentForm.tsx` — input form
- `LandPriceAnalysis.tsx` — market comparison
- `YieldAnalysis.tsx` — gross yield + 8% threshold
- `DeadCrossChart.tsx` — dead-cross visualization
- `CashFlowChart.tsx` — stress-test cash flow
- `CostBreakdown.tsx` — cost breakdown
- `WatchlistPanel.tsx` — property watchlist (localStorage persistence)

## Conventions

- **Branch**: `feature/<content>` (no issue numbers)
- **Commits**: English, small logical units (types / logic / API / frontend / tests)
- **PR body**: Japanese, follow `.github/pull_request_template.md`
- **No `Co-Authored-By`** lines in commits
- **色は単独の情報源にしない** — Badge/アイコンを必ず併記する（WCAG 1.4.1）

## CI (GitHub Actions)

| Workflow | Trigger paths | Checks |
|----------|--------------|--------|
| `backend-ci.yml` | `backend/**` | golangci-lint, go test -race, go build |
| `frontend-ci.yml` | `frontend/**` | lint, tsc, vitest, next build |

## MCP servers

`docs` server is configured in `.mcp.json` — use `mcp__docs__*` tools to read `docs/` design documents (overview, API reference, domain specs, etc.).

## Claude Code operating rules

These rules define the safety boundary for autonomous Claude Code operations.

### Git operations

- **Never push directly to `main`** — always work on a `feature/*` branch and open a PR
- **Never force-push** (`git push --force`) — prevents overwriting remote history
- **Never skip hooks** (`git commit --no-verify`) — pre-commit hooks must always run
- **`git reset --hard` only on explicit instruction** — prevents loss of uncommitted work

### Secrets and environment variables

- **Never read, display, or log the contents of `.env`** — `MLIT_API_KEY`, `APP_INTERNAL_API_KEY`, and other secrets must not appear in output
- **Never stage or commit `.env`** — it is gitignored; do not force-add it
- **Never include secret values in API responses, test output, or error messages**

### Infrastructure

- **`terraform destroy` only on explicit instruction** — prevents accidental deletion of Cloud Run services or IAM resources
- **`terraform apply` requires confirmation** — always review the plan output before applying
- **Never modify GCS bucket contents or Secret Manager secrets directly** — these are managed by Terraform
- **Destructive local operations** (`docker volume rm`, `rm -rf`, etc.) require confirmation before execution

### PR and external actions

- **Never merge a PR without explicit instruction** — even if CI passes
- **Never post comments to Issues or PRs without explicit instruction** — unintended external communication must be avoided
