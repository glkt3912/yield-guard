# syntax=docker/dockerfile:1
# CVE-2026-31431 パッチ適用済み alpine3.22 以降を使用すること
FROM node:22-alpine3.22 AS deps
WORKDIR /app
COPY package.json package-lock.json ./
# npmキャッシュをホストに永続化
RUN --mount=type=cache,target=/root/.npm \
    npm ci

FROM node:22-alpine3.22 AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine3.22 AS runner
WORKDIR /app
ENV NODE_ENV=production
# root実行を避けるために専用ユーザーを作成
RUN addgroup -g 1001 appgroup && adduser -u 1001 -G appgroup -D appuser
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
RUN chown -R appuser:appgroup .
USER appuser
EXPOSE 3000
CMD ["npm", "start"]
