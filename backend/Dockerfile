# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
# モジュールキャッシュをホストに永続化（go.mod/go.sum が変わらなければダウンロードをスキップ）
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
# --mount=type=cache: ローカル開発専用（CI では ephemeral ランナーのため無効）
# CI でのキャッシュは deploy-backend.yml の --cache-from/to で対応
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o server ./cmd/server

# alpine:latest ではなくバージョンを固定して再現性を確保
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
# root実行を避けるために専用ユーザーを作成
RUN adduser -D -u 1000 appuser
WORKDIR /home/appuser/
COPY --from=builder /app/server .
RUN chown appuser:appuser server
USER appuser
EXPOSE 8080
CMD ["./server"]
