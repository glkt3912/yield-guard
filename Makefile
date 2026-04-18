.PHONY: dev test lint build help \
        mlit-land-prices mlit-municipalities mlit-station-ridership mlit-population-forecast \
        api-station-ridership api-estimate-ridership api-population-forecast \
        integration integration-population

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

## mlit-station-ridership: 駅別乗降客数を取得 (XKT015) ※国交省APIへ直接リクエスト
##   使い方: make mlit-station-ridership z=14 x=14547 y=6451
##   緯度経度→タイル変換例(z=14): 渋谷付近 → x=14547 y=6451
mlit-station-ridership:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT015?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq .

## mlit-population-forecast: 将来推計人口を取得 (XKT013) ※国交省APIへ直接リクエスト
##   使い方: make mlit-population-forecast z=14 x=14547 y=6451
##   緯度経度→タイル変換例(z=14): 渋谷付近 → x=14547 y=6451 / 前橋付近 → x=14479 y=6412
mlit-population-forecast:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT013?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:2] | map(.properties | {MESH_ID, PTN_2020, PTN_2030, PTN_2040, PTN_2050}))}'

# ---------------------------------------------------------------------------
# ローカル開発サーバー向けAPIテスト (backend が :8080 で起動中であること)
# ---------------------------------------------------------------------------
API_BASE := http://localhost:8080/api

## api-station-ridership: ローカルの /api/station-ridership を呼び出す
##   使い方: make api-station-ridership lat=35.6762 lng=139.6503 [z=14]
api-station-ridership:
	@test -n "$(lat)" || (echo "ERROR: lat は必須です (例: lat=35.6762)"; exit 1)
	@test -n "$(lng)" || (echo "ERROR: lng は必須です (例: lng=139.6503)"; exit 1)
	curl -s \
	  "$(API_BASE)/station-ridership?lat=$(lat)&lng=$(lng)$(if $(z),&z=$(z),)" \
	  | jq .

## api-population-forecast: ローカルの /api/population-forecast を呼び出す
##   使い方: make api-population-forecast lat=35.6762 lng=139.6503 [z=14]
api-population-forecast:
	@test -n "$(lat)" || (echo "ERROR: lat は必須です (例: lat=35.6762)"; exit 1)
	@test -n "$(lng)" || (echo "ERROR: lng は必須です (例: lng=139.6503)"; exit 1)
	curl -s \
	  "$(API_BASE)/population-forecast?lat=$(lat)&lng=$(lng)$(if $(z),&z=$(z),)" \
	  | jq .

## api-estimate-ridership: 需要スコア補正付き理論価格推定を呼び出す
##   使い方: make api-estimate-ridership area=13 city=13113 price=50000000 area_sqm=100 building_age=10 station_minutes=5 ridership_score=A
api-estimate-ridership:
	@test -n "$(area)"           || (echo "ERROR: area は必須です";           exit 1)
	@test -n "$(price)"          || (echo "ERROR: price は必須です";          exit 1)
	@test -n "$(area_sqm)"       || (echo "ERROR: area_sqm は必須です";       exit 1)
	@test -n "$(building_age)"   || (echo "ERROR: building_age は必須です";   exit 1)
	curl -s \
	  "$(API_BASE)/land-prices/estimate?area=$(area)$(if $(city),&city=$(city),)&year=2024&quarter=1&to_year=2024&to_quarter=4&price=$(price)&area_sqm=$(area_sqm)&building_age=$(building_age)$(if $(station_minutes),&station_minutes=$(station_minutes),)$(if $(ridership_score),&ridership_score=$(ridership_score),)" \
	  | jq .

# ---------------------------------------------------------------------------
# 統合テスト (実際の国交省APIへ疎通確認)
# ---------------------------------------------------------------------------

## integration: 全統合テストを実行 (MLIT_API_KEY 必須)
integration:
	cd backend && source ../.env 2>/dev/null; \
	go test -tags=integration ./internal/mlit/... -v -timeout 120s

## integration-population: 将来推計人口 (XKT013) の統合テストのみ実行
integration-population:
	cd backend && source ../.env 2>/dev/null; \
	go test -tags=integration ./internal/mlit/... -v -timeout 60s -run TestFetchPopulationForecast
