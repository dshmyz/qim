# ====================================================
# QIM Server — 多阶段 Docker 构建（含前端 embed）
# ====================================================
# 构建 amd64:
#   docker build --build-arg TARGETARCH=amd64 -t qim-server:latest .
#
# 构建 arm64（信创/飞腾）:
#   docker build --build-arg TARGETARCH=arm64 -t qim-server:latest .
#
# 跳过前端构建（已有 webroot/ 目录）:
#   docker build --build-arg SKIP_FRONTEND=1 -t qim-server:latest .
#
# 启动:
#   docker run -d -p 8080:8080 \
#     -v ./config.yaml:/app/config.yaml \
#     -v ./qim.db:/app/qim.db \
#     -v ./uploads:/app/uploads \
#     qim-server:latest
# ====================================================

# ---- 阶段 1: 前端构建 ----
FROM node:20-alpine AS frontend-builder

ARG SKIP_FRONTEND=

WORKDIR /build

# 只拷贝 package.json 先装依赖（利用 Docker 缓存）
COPY qim-admin/package.json qim-admin/package-lock.json* ./qim-admin/

RUN if [ -z "$SKIP_FRONTEND" ]; then \
      cd qim-admin && npm install --legacy-peer-deps 2>/dev/null || npm install; \
    fi

# 拷贝源码并构建
COPY qim-admin/ ./qim-admin/

RUN if [ -z "$SKIP_FRONTEND" ]; then \
      cd qim-admin && \
      npm run build && \
      npx vite build --config vite.config.landing.ts && \
      mkdir -p /build/web/webroot/admin /build/web/webroot/landing && \
      cp -r dist/* /build/web/webroot/admin/ && \
      cp -r dist-landing/* /build/web/webroot/landing/ && \
      if [ -f /build/web/webroot/landing/landing.html ]; then \
        mv /build/web/webroot/landing/landing.html /build/web/webroot/landing/index.html; \
      fi; \
    fi

# ---- 阶段 2: Go 后端构建 ----
FROM golang:1.25-alpine3.20 AS backend-builder

ARG TARGETARCH=amd64

WORKDIR /build
COPY qim-server/go.mod qim-server/go.sum ./
RUN go mod download

COPY qim-server/ .

# 拷贝前端构建产物到 embed 目录
COPY --from=frontend-builder /build/web/ ./web/

# 纯 Go 编译，无需 CGO
RUN GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o qim-server .

# ---- 运行阶段 ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=backend-builder /build/qim-server .

# 默认配置文件（用户在运行时通过 -v 挂载覆盖）
COPY qim-server/config.yaml /app/config.yaml

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

EXPOSE 8080

ENTRYPOINT ["./qim-server"]
