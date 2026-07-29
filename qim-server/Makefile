.PHONY: build-cli build-cli-all clean version

# 默认版本，可通过 make build-cli VERSION=v2.1.0 覆盖
VERSION ?= dev
OUT_DIR  = data/cli

# ---------- CLI 构建 ----------

## 构建当前平台 CLI 并放入 data/cli/
build-cli:
	@mkdir -p $(OUT_DIR)
	go build -ldflags "-X main.version=$(VERSION)" \
		-o $(OUT_DIR)/qim-$(shell go env GOOS)-$(shell go env GOARCH)$(shell if [ "$(shell go env GOOS)" = "windows" ]; then echo ".exe"; fi) \
		cmd/qim/main.go
	@echo "$(VERSION)" > $(OUT_DIR)/version.txt
	@echo "✅ $(OUT_DIR)/qim-$(shell go env GOOS)-$(shell go env GOARCH)"

## 构建全部 6 个平台的 CLI
build-cli-all:
	@mkdir -p $(OUT_DIR)
	@for pair in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do \
		os=$${pair%/*}; arch=$${pair#*/}; ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		echo "  → qim-$$os-$$arch$$ext"; \
		GOOS=$$os GOARCH=$$arch go build -ldflags "-X main.version=$(VERSION)" \
			-o $(OUT_DIR)/qim-$$os-$$arch$$ext cmd/qim/main.go; \
	done
	@echo "$(VERSION)" > $(OUT_DIR)/version.txt
	@echo "✅ 全部构建完成 ($(VERSION))"

## 清理构建产物
clean:
	rm -rf $(OUT_DIR)/qim-* $(OUT_DIR)/version.txt

## 打印当前版本
version:
	@echo "$(VERSION)"
