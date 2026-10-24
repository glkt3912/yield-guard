# syntax=docker/dockerfile:1
# CVE-2026-31431 パッチ適用済み alpine3.22 以降を使用すること
FROM node:22-alpine3.22
WORKDIR /app
COPY package.json package-lock.json ./
# npmキャッシュをホストに永続化
RUN --mount=type=cache,target=/root/.npm \
    npm ci
COPY . .
RUN npm run build
# root実行を避けるために専用ユーザーを作成
RUN addgroup -g 1001 appgroup && adduser -u 1001 -G appgroup -D appuser
USER appuser
EXPOSE 3000
# ローカル確認用: 静的ビルドを配信し /api を BACKEND_URL へプロキシ（本番は Cloudflare Workers）
CMD ["npx", "vite", "preview", "--host", "0.0.0.0"]
