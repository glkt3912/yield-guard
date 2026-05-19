.PHONY: dev backend frontend install install-hooks logs test lint build clean help \
        swagger swagger-check \
        mlit-land-prices mlit-municipalities mlit-station-ridership mlit-population-forecast mlit-land-appraisals \
        mlit-urban-zoning mlit-liquefaction mlit-flood-hazard mlit-storm-hazard mlit-tsunami-hazard mlit-landslide-hazard \
        api-station-ridership api-estimate-ridership api-population-forecast api-land-appraisals api-investment-score \
        integration integration-population integration-land-appraisals \
        e2e e2e-ui e2e-report

## dev: バックエンド・フロントエンドの開発サーバーを起動
dev:
	@if [ ! -f .env ]; then \
	  echo "ERROR: .env が見つかりません。worktree の場合は以下を実行してください:"; \
	  echo "  cp ../yield-guard/.env .env"; \
	  exit 1; \
	fi
	@trap 'kill 0' INT TERM EXIT; \
	echo "==> Starting backend..."; \
	(cd backend && set -a; . ../.env; set +a; go run ./cmd/server) & \
	BACKEND_PID=$$!; \
	echo "==> Waiting for backend (up to 30s)..."; \
	for i in $$(seq 1 30); do \
	  if curl -sf http://localhost:8080/health > /dev/null 2>&1; then \
	    echo "==> Backend ready (PID=$$BACKEND_PID)"; \
	    break; \
	  fi; \
	  if ! kill -0 $$BACKEND_PID 2>/dev/null; then \
	    echo "ERROR: バックエンドの起動に失敗しました。ログを確認してください。"; \
	    exit 1; \
	  fi; \
	  sleep 1; \
	done; \
	if ! curl -sf http://localhost:8080/health > /dev/null 2>&1; then \
	  echo "ERROR: バックエンドが 30 秒以内に応答しませんでした。"; \
	  kill $$BACKEND_PID 2>/dev/null; \
	  exit 1; \
	fi; \
	echo "==> Starting frontend..."; \
	(cd frontend && npm run dev) & \
	wait

## backend: バックエンド開発サーバーのみ起動
backend:
	cd backend && go run ./cmd/server

## frontend: フロントエンド開発サーバーのみ起動
frontend:
	cd frontend && npm run dev

## swagger: OpenAPI スキーマを生成 (docs/openapi/swagger.json)
##   型ファイルも更新するには別途 cd frontend && npm run generate:types を実行すること
swagger:
	cd backend && swag init -g cmd/server/main.go -o ../docs/openapi --outputTypes json
	@echo "==> Generated docs/openapi/swagger.json"

## swagger-check: swagger.json のドリフトをローカルで確認 (PR 前チェック用)
swagger-check: swagger
	git diff --exit-code docs/openapi/swagger.json

## install: フロントエンド依存関係をインストール
install:
	cd frontend && npm install

## install-hooks: lefthook git フックをインストール
install-hooks:
	@if ! command -v lefthook >/dev/null 2>&1; then \
	  echo "==> Installing lefthook via brew..."; \
	  brew install lefthook; \
	fi
	lefthook install

## logs: Dockerコンテナのログを表示（未起動の場合は案内）
logs:
	@if docker compose ps --quiet 2>/dev/null | grep -q .; then \
	  docker compose logs -f; \
	else \
	  echo "Docker コンテナが起動していません。先に 'make docker-up' を実行してください。"; \
	fi

## test: 全ユニットテストを実行（E2E は make e2e で実行）
test:
	@echo "==> Backend tests"
	cd backend && go test -race ./... -timeout 120s
	@echo "==> Frontend unit tests"
	cd frontend && npm test

## e2e: フロントエンド E2E テスト実行 (Playwright / Chromium)
e2e:
	cd frontend && npm run e2e

## e2e-ui: Playwright UI モードで E2E テストを起動
e2e-ui:
	cd frontend && npm run e2e:ui

## e2e-report: 最後の Playwright レポートをブラウザで表示
e2e-report:
	cd frontend && npx playwright show-report playwright-report

## lint: 全lintを実行
lint:
	@echo "==> Backend lint"
	cd backend && golangci-lint run ./...
	@echo "==> Frontend lint"
	cd frontend && npm run format:check
	cd frontend && npm run lint
	cd frontend && npm run type-check

## build: 全ビルドを実行
build:
	@echo "==> Backend build"
	cd backend && go build -o yield-guard-server ./cmd/server
	@echo "==> Frontend build"
	cd frontend && npm run build

## clean: ビルド成果物を削除
clean:
	rm -f backend/yield-guard-server
	rm -rf frontend/.next
	rm -rf frontend/out

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

## mlit-land-appraisals: 地価公示情報を取得 (XCT001) ※国交省APIへ直接リクエスト
##   使い方: make mlit-land-appraisals area=13 year=2024 [division=00]
##   division: 00=住宅地(デフォルト) 05=商業地 07=準工業地 09=工業地
mlit-land-appraisals:
	@test -n "$(area)" || (echo "ERROR: area は必須です (例: area=13)"; exit 1)
	@test -n "$(year)" || (echo "ERROR: year は必須です (例: year=2024)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XCT001?area=$(area)&year=$(year)&division=$(or $(division),00)" \
	   | jq '{status: .status, count: (.data | length), sample: (.data[:3] | map({"地域名": .["標準地番号 地域名"], "価格時点": .["価格時点"], "公示価格": .["公示価格"], "1㎡当たりの価格": .["1㎡当たりの価格"], "変動率": .["変動率"], "用途区分": .["標準地番号 用途区分"]}))}'

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

## mlit-urban-zoning: 都市計画区域/区域区分を取得 (XKT001) ※フィールド値確認用
##   使い方: make mlit-urban-zoning z=14 x=14547 y=6451
mlit-urban-zoning:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT001?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:3] | map(.properties | {area_classification_ja, kubun_id, city_name}))}'

## mlit-liquefaction: 液状化発生傾向図を取得 (XKT025) ※liquefaction_tendency_level の実値確認用
##   使い方: make mlit-liquefaction z=14 x=14547 y=6451
mlit-liquefaction:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT025?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:5] | map(.properties | {liquefaction_tendency_level, note, mesh_code}))}'

## mlit-flood-hazard: 洪水浸水想定区域を取得 (XKT026) ※A31a_205 浸水深ランクの実値確認用
##   使い方: make mlit-flood-hazard z=14 x=14547 y=6451
mlit-flood-hazard:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT026?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:3] | map(.properties | {A31a_205, A31a_202, A31a_204}))}'

## mlit-storm-hazard: 高潮浸水想定区域を取得 (XKT027) ※A49_003 浸水深区分の実値確認用
##   使い方: make mlit-storm-hazard z=14 x=14547 y=6451
mlit-storm-hazard:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT027?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:3] | map(.properties | {A49_003, A49_001, target_year}))}'

## mlit-tsunami-hazard: 津波浸水想定を取得 (XKT028) ※A40_003 浸水深区分の実値確認用
##   使い方: make mlit-tsunami-hazard z=14 x=14547 y=6451
mlit-tsunami-hazard:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT028?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:3] | map(.properties | {A40_003, A40_001, target_year}))}'

## mlit-landslide-hazard: 土砂災害警戒区域を取得 (XKT029) ※A33_001/A33_002 の実値確認用
##   使い方: make mlit-landslide-hazard z=14 x=14547 y=6451
mlit-landslide-hazard:
	@test -n "$(z)" || (echo "ERROR: z は必須です (例: z=14)"; exit 1)
	@test -n "$(x)" || (echo "ERROR: x は必須です (例: x=14547)"; exit 1)
	@test -n "$(y)" || (echo "ERROR: y は必須です (例: y=6451)"; exit 1)
	@source .env 2>/dev/null; \
	 curl -s \
	   -H "Ocp-Apim-Subscription-Key: $$MLIT_API_KEY" \
	   --compressed \
	   "$(MLIT_BASE)/XKT029?response_format=geojson&z=$(z)&x=$(x)&y=$(y)" \
	   | jq '{feature_count: (.features | length), sample: (.features[:3] | map(.properties | {A33_001, A33_002, A33_005, A33_007}))}'

## api-station-ridership: ローカルの /api/station-ridership を呼び出す
##   使い方: make api-station-ridership lat=35.6762 lng=139.6503 [z=14]
api-station-ridership:
	@test -n "$(lat)" || (echo "ERROR: lat は必須です (例: lat=35.6762)"; exit 1)
	@test -n "$(lng)" || (echo "ERROR: lng は必須です (例: lng=139.6503)"; exit 1)
	curl -s \
	  "$(API_BASE)/station-ridership?lat=$(lat)&lng=$(lng)$(if $(z),&z=$(z),)" \
	  | jq .

## api-land-appraisals: ローカルの /api/land-appraisals を呼び出す
##   使い方: make api-land-appraisals area=13 city=13101 year=2024 [division=00]
##   division: 00=住宅地(デフォルト) 05=商業地 07=準工業地 09=工業地
api-land-appraisals:
	@test -n "$(area)" || (echo "ERROR: area は必須です (例: area=13)"; exit 1)
	@test -n "$(year)" || (echo "ERROR: year は必須です (例: year=2024)"; exit 1)
	curl -s \
	  "$(API_BASE)/land-appraisals?area=$(area)$(if $(city),&city=$(city),)&year=$(year)$(if $(division),&division=$(division),)" \
	  | jq .

## api-population-forecast: ローカルの /api/population-forecast を呼び出す
##   使い方: make api-population-forecast lat=35.6762 lng=139.6503 [z=14]
api-population-forecast:
	@test -n "$(lat)" || (echo "ERROR: lat は必須です (例: lat=35.6762)"; exit 1)
	@test -n "$(lng)" || (echo "ERROR: lng は必須です (例: lng=139.6503)"; exit 1)
	curl -s \
	  "$(API_BASE)/population-forecast?lat=$(lat)&lng=$(lng)$(if $(z),&z=$(z),)" \
	  | jq .

## api-investment-score: ローカルの /api/investment-score を呼び出す
##   使い方: make api-investment-score lat=35.6580 lng=139.7016
##   渋谷付近: lat=35.6580 lng=139.7016 / 前橋付近: lat=36.3897 lng=139.0607
api-investment-score:
	@test -n "$(lat)" || (echo "ERROR: lat は必須です (例: lat=35.6580)"; exit 1)
	@test -n "$(lng)" || (echo "ERROR: lng は必須です (例: lng=139.7016)"; exit 1)
	curl -s \
	  "$(API_BASE)/investment-score?lat=$(lat)&lng=$(lng)" \
	  | jq '{totalScore: .totalScore, grade: .grade, breakdown: (.breakdown | {population: .population, ridership: .ridership, urbanArea: .urbanArea, locationOptimization: .locationOptimization, hazardRisk: .hazardRisk, liquefactionRisk: .liquefactionRisk, embankment: .embankment, disasterHistory: .disasterHistory})}'

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
	cd backend && set -a; . ../.env 2>/dev/null; set +a; \
	go test -tags=integration ./internal/mlit/... -v -timeout 120s

## integration-population: 将来推計人口 (XKT013) の統合テストのみ実行
integration-population:
	cd backend && set -a; . ../.env 2>/dev/null; set +a; \
	go test -tags=integration ./internal/mlit/... -v -timeout 60s -run TestFetchPopulationForecast

## integration-land-appraisals: 地価公示 (XCT001) の統合テストのみ実行
integration-land-appraisals:
	cd backend && set -a; . ../.env 2>/dev/null; set +a; \
	go test -tags=integration ./internal/mlit/... -v -timeout 60s -run TestFetchLandAppraisals
