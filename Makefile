.PHONY: dev test lint build help \
        mlit-land-prices mlit-municipalities mlit-prefectures

## dev: バックエンド・フロントエンドの開発サーバーを起動
dev:
	@echo "==> Starting backend..."
	cd backend && go run ./cmd/server &
	@echo "==> Starting frontend..."
	cd frontend && npm run dev

## test: 全テストを実行
test:
	@echo "==> Backend tests"
	cd backend && go test -race ./... -timeout 120s
	@echo "==> Frontend tests"
	cd frontend && npm test

## lint: 全lintを実行
lint:
	@echo "==> Backend lint"
	cd backend && golangci-lint run ./...
	@echo "==> Frontend lint"
	cd frontend && npm run lint
	cd frontend && npx tsc --noEmit

## build: 全ビルドを実行
build:
	@echo "==> Backend build"
	cd backend && go build -o yield-guard-server ./cmd/server
	@echo "==> Frontend build"
	cd frontend && npm run build

## docker-up: Dockerコンテナをビルドして起動
docker-up:
	docker compose up --build

## docker-down: Dockerコンテナを停止・削除
docker-down:
	docker compose down

## help: 利用可能なコマンドを表示
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'

# ---------------------------------------------------------------------------
# 国交省 不動産情報ライブラリ API 開発用リクエスト
# APIキーは .env の MLIT_API_KEY から読み込む（git管理外）
# ---------------------------------------------------------------------------
MLIT_BASE := https://www.reinfolib.mlit.go.jp/ex-api/external
_mlit_key  = $$(source .env 2>/dev/null && echo $$MLIT_API_KEY)

## mlit-land-prices: 土地取引価格を取得 (XIT001)
##   使い方: make mlit-land-prices area=13 year=2024 quarter=1 to_year=2024 to_quarter=4 [city=13101]
mlit-land-prices:
	@test -n "$(area)"      || (echo "ERROR: area は必須です (例: area=13)"; exit 1)
	@test -n "$(year)"      || (echo "ERROR: year は必須です (例: year=2024)"; exit 1)
	@test -n "$(quarter)"   || (echo "ERROR: quarter は必須です (例: quarter=1)"; exit 1)
	@test -n "$(to_year)"   || (echo "ERROR: to_year は必須です (例: to_year=2024)"; exit 1)
	@test -n "$(to_quarter)"|| (echo "ERROR: to_quarter は必須です (例: to_quarter=4)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XIT001?area=$(area)&year=$(year)&quarter=$(quarter)&toYear=$(to_year)&toQuarter=$(to_quarter)&priceClassification=01$(if $(city),&city=$(city),)" \
	   | jq .

## mlit-municipalities: 市区町村一覧を取得 (XIT002)
##   使い方: make mlit-municipalities area=13
mlit-municipalities:
	@test -n "$(area)" || (echo "ERROR: area は必須です (例: area=13)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XIT002?area=$(area)" \
	   | jq .
