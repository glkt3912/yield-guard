.PHONY: dev test lint build help

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

## help: 利用可能なコマンドを表示
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'
