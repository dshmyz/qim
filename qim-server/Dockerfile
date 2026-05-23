# ====================================================
# QIM Server — 多阶段 Docker 构建
# ====================================================
# 构建 amd64:
#   docker build --build-arg TARGETARCH=amd64 -t qim-server:latest .
#
# 构建 arm64（信创/飞腾）:
#   docker build --build-arg TARGETARCH=arm64 -t qim-server:latest .
#
# 启动:
#   docker run -d -p 8080:8080 \
#     -v ./config.yaml:/app/config.yaml \
#     -v ./qim.db:/app/qim.db \
#     -v ./uploads:/app/uploads \
#     qim-server:latest
# ====================================================

# ---- 构建阶段 ----
FROM golang:1.25-alpine3.20 AS builder

ARG TARGETARCH=amd64

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 纯 Go 编译，无需 CGO
RUN GOOS=linux GOARCH=${TARGETARCH} \
    go build -ldflags="-s -w" -o qim-server .

# ---- 运行阶段 ----
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /build/qim-server .

# 默认配置文件（用户在运行时通过 -v 挂载覆盖）
COPY config.yaml /app/config.yaml

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/swagger/index.html || exit 1

EXPOSE 8080

ENTRYPOINT ["./qim-server"]
