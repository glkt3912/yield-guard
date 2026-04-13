# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
# モジュールキャッシュをホストに永続化（go.mod/go.sum が変わらなければダウンロードをスキップ）
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .
# ビルドキャッシュをホストに永続化（変更のないパッケージの再コンパイルをスキップ）
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
