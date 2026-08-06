.PHONY: help build build-all build-server build-server-all build-mcp build-mcp-all build-cli build-cli-all clean version

# 默认版本，可通过 make build VERSION=v2.1.0 覆盖
VERSION ?= dev

# CLI 产物目录（与历史约定保持一致）
CLI_OUT  = data/cli

# server / mcp 产物目录
OUT_DIR  = data/dist

# 交叉编译的目标平台（server/mcp/cli 通用）。
# 需要更多平台在这里加，格式 os/arch。
PLATFORMS = darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64

# ---------- 一键打包 ----------

## 构建当前平台三件套：server + mcp + cli
build:
	@$(MAKE) build-server
	@$(MAKE) build-mcp
	@$(MAKE) build-cli

## 构建全部 6 平台三件套（server + mcp + cli）
build-all:
	@$(MAKE) build-server-all
	@$(MAKE) build-mcp-all
	@$(MAKE) build-cli-all

# ---------- Server 构建 ----------

## 构建当前平台 server 并放入 $(OUT_DIR)/
build-server:
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/qim-server-$(shell go env GOOS)-$(shell go env GOARCH)$(shell if [ "$(shell go env GOOS)" = "windows" ]; then echo ".exe"; fi) ./main.go
	@echo "✅ $(OUT_DIR)/qim-server-$(shell go env GOOS)-$(shell go env GOARCH)"

## 构建全部 6 平台 server
build-server-all:
	@mkdir -p $(OUT_DIR)
	@for pair in $(PLATFORMS); do \
		os=$${pair%/*}; arch=$${pair#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "  → qim-server-$$os-$$arch$$ext"; \
		GOOS=$$os GOARCH=$$arch go build \
			-o $(OUT_DIR)/qim-server-$$os-$$arch$$ext ./main.go; \
	done
	@echo "✅ server 全部构建完成 ($(VERSION))"

# ---------- MCP 构建 ----------

## 构建当前平台 mcp 并放入 $(OUT_DIR)/
build-mcp:
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/qim-mcp-$(shell go env GOOS)-$(shell go env GOARCH)$(shell if [ "$(shell go env GOOS)" = "windows" ]; then echo ".exe"; fi) ./cmd/qim-mcp
	@echo "✅ $(OUT_DIR)/qim-mcp-$(shell go env GOOS)-$(shell go env GOARCH)"

## 构建全部 6 平台 mcp
build-mcp-all:
	@mkdir -p $(OUT_DIR)
	@for pair in $(PLATFORMS); do \
		os=$${pair%/*}; arch=$${pair#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "  → qim-mcp-$$os-$$arch$$ext"; \
		GOOS=$$os GOARCH=$$arch go build \
			-o $(OUT_DIR)/qim-mcp-$$os-$$arch$$ext ./cmd/qim-mcp; \
	done
	@echo "✅ mcp 全部构建完成 ($(VERSION))"

# ---------- CLI 构建（保持原语义） ----------

## 构建当前平台 CLI 并放入 data/cli/
build-cli:
	@mkdir -p $(CLI_OUT)
	go build -ldflags "-X main.version=$(VERSION)" \
		-o $(CLI_OUT)/qim-$(shell go env GOOS)-$(shell go env GOARCH)$(shell if [ "$(shell go env GOOS)" = "windows" ]; then echo ".exe"; fi) \
		./cmd/qim
	@echo "$(VERSION)" > $(CLI_OUT)/version.txt
	@echo "✅ $(CLI_OUT)/qim-$(shell go env GOOS)-$(shell go env GOARCH)"

## 构建全部 6 个平台的 CLI
build-cli-all:
	@mkdir -p $(CLI_OUT)
	@for pair in $(PLATFORMS); do \
		os=$${pair%/*}; arch=$${pair#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "  → qim-$$os-$$arch$$ext"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "-X main.version=$(VERSION)" \
			-o $(CLI_OUT)/qim-$$os-$$arch$$ext ./cmd/qim; \
	done
	@echo "$(VERSION)" > $(CLI_OUT)/version.txt
	@echo "✅ CLI 全部构建完成 $(VERSION)"

# ---------- 清理 / 帮助 ----------

## 清理构建产物
clean:
	@test -n "$(CLI_OUT)" || { echo "CLI_OUT 未设置，拒绝清理"; exit 1; }
	@test -n "$(OUT_DIR)" || { echo "OUT_DIR 未设置，拒绝清理"; exit 1; }
	rm -rf $(CLI_OUT)/qim-* $(CLI_OUT)/version.txt
	rm -rf $(OUT_DIR)/qim-server-* $(OUT_DIR)/qim-mcp-*

## 打印当前版本
version:
	@echo "$(VERSION)"

## 列出可用目标
help:
	@echo "可用目标:"
	@echo "  build            构建当前平台三件套 (server + mcp + cli)"
	@echo "  build-all        构建全部 6 平台三件套"
	@echo "  build-server     构建当前平台 server"
	@echo "  build-server-all 构建全部 6 平台 server"
	@echo "  build-mcp        构建当前平台 mcp"
	@echo "  build-mcp-all    构建全部 6 平台 mcp"
	@echo "  build-cli        构建当前平台 CLI"
	@echo "  build-cli-all    构建全部 6 平台 CLI"
	@echo ""
	@echo "用法: make build VERSION=v2.1.0"
