# HuShu智能网关 - Makefile
# 支持多平台交叉编译

.PHONY: all clean build help deploy deploy-arm32 deploy-arm64 deploy-darwin deploy-windows

# 默认目标
all: help

# 帮助信息
help:
	@echo "HuShu智能网关 - 编译部署工具"
	@echo ""
	@echo "可用目标:"
	@echo "  build          - 编译当前平台版本"
	@echo "  clean          - 清理构建产物"
	@echo "  deploy         - 部署到所有平台"
	@echo "  deploy-arm32   - 部署Linux ARM32版本"
	@echo "  deploy-arm64   - 部署Linux ARM64版本"
	@echo "  deploy-darwin  - 部署macOS版本"
	@echo "  deploy-windows - 部署Windows版本"
	@echo ""
	@echo "用法:"
	@echo "  make build              # 本地编译"
	@echo "  make deploy             # 编译所有平台"
	@echo "  make deploy-arm32       # 仅编译ARM32"

# 项目名称和版本
PROJECT_NAME := gogw
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y%m%d-%H%M%S)

# 源文件
MAIN_SRC := cmd/main.go
CONFIG_SRC := config/config.yaml
MIGRATIONS_DIR := migrations
WEB_DIR := web
DATA_DIR := data

# 部署目录
DEPLOY_DIR := deploy

# 编译当前平台版本
build:
	@echo "=== 构建 $(PROJECT_NAME) $(VERSION) ==="
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(PROJECT_NAME) $(MAIN_SRC)
	@echo "✅ 构建完成: $(PROJECT_NAME)"

# 清理构建产物
clean:
	@echo "=== 清理构建产物 ==="
	rm -f $(PROJECT_NAME)
	rm -rf $(DEPLOY_DIR)
	@echo "✅ 清理完成"

# 创建部署目录结构
prepare-deploy:
	@echo "=== 准备部署目录 ==="
	mkdir -p $(DEPLOY_DIR)/arm32
	mkdir -p $(DEPLOY_DIR)/arm64
	mkdir -p $(DEPLOY_DIR)/darwin
	mkdir -p $(DEPLOY_DIR)/windows
	@echo "✅ 目录准备完成"

# Linux ARM32 编译
deploy-arm32: prepare-deploy
	@echo "=== 编译 Linux ARM32 版本 ==="
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/arm32/$(PROJECT_NAME) $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/arm32/
	cp -r migrations $(DEPLOY_DIR)/arm32/
	cp -r web $(DEPLOY_DIR)/arm32/
	@echo "✅ ARM32 部署包已生成: $(DEPLOY_DIR)/arm32/"

# Linux ARM64 编译
deploy-arm64: prepare-deploy
	@echo "=== 编译 Linux ARM64 版本 ==="
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/arm64/$(PROJECT_NAME) $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/arm64/
	cp -r migrations $(DEPLOY_DIR)/arm64/
	cp -r web $(DEPLOY_DIR)/arm64/
	@echo "✅ ARM64 部署包已生成: $(DEPLOY_DIR)/arm64/"

# macOS (Intel/AMD64) 编译
deploy-darwin: prepare-deploy
	@echo "=== 编译 macOS 版本 ==="
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/darwin/$(PROJECT_NAME) $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/darwin/
	cp -r migrations $(DEPLOY_DIR)/darwin/
	cp -r web $(DEPLOY_DIR)/darwin/
	@echo "✅ macOS 部署包已生成: $(DEPLOY_DIR)/darwin/"

# macOS ARM64 (Apple Silicon) 编译
deploy-darwin-arm64: prepare-deploy
	@echo "=== 编译 macOS ARM64 版本 ==="
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/darwin/$(PROJECT_NAME)-arm64 $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/darwin/
	cp -r migrations $(DEPLOY_DIR)/darwin/
	cp -r web $(DEPLOY_DIR)/darwin/
	@echo "✅ macOS ARM64 部署包已生成: $(DEPLOY_DIR)/darwin/"

# Windows AMD64 编译
deploy-windows: prepare-deploy
	@echo "=== 编译 Windows 版本 ==="
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/windows/$(PROJECT_NAME).exe $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/windows/
	cp -r migrations $(DEPLOY_DIR)/windows/
	cp -r web $(DEPLOY_DIR)/windows/
	@echo "✅ Windows 部署包已生成: $(DEPLOY_DIR)/windows/"

# 部署到所有平台
deploy: prepare-deploy
	@echo "========================================"
	@echo "  HuShu智能网关 - 多平台编译部署"
	@echo "========================================"
	@echo ""
	$(MAKE) deploy-arm32
	@echo ""
	$(MAKE) deploy-arm64
	@echo ""
	$(MAKE) deploy-darwin
	@echo ""
	$(MAKE) deploy-windows
	@echo ""
	@echo "========================================"
	@echo "  ✅ 多平台部署完成!"
	@echo "========================================"
	@echo ""
	@echo "部署目录结构:"
	@ls -la $(DEPLOY_DIR)/*/$(PROJECT_NAME)* 2>/dev/null || echo "  请查看 deploy/ 目录"
	@echo ""
	@echo "快速部署到ARM64设备:"
	@echo "  scp -r deploy/arm64/* root@<设备IP>:/opt/hushu/"

# 快速部署 (仅Linux ARM64)
quick-deploy: deploy-arm64
	@echo ""
	@echo "快速部署提示:"
	@echo "  1. 上传 deploy/arm64/ 目录到设备"
	@echo "  2. 执行: chmod +x gogw"
	@echo "  3. 运行: ./gogw"

# 查看部署文件大小
size:
	@echo "=== 部署文件大小 ==="
	@if [ -d "$(DEPLOY_DIR)" ]; then \
		find $(DEPLOY_DIR) -type f -exec ls -lh {} \; 2>/dev/null | awk '{print $$9, $$5}'; \
	else \
		echo "请先执行: make deploy"; \
	fi

# 列出所有可执行文件
list:
	@echo "=== 可执行文件列表 ==="
	@if [ -d "$(DEPLOY_DIR)" ]; then \
		find $(DEPLOY_DIR) -type f \( -name "$(PROJECT_NAME)" -o -name "$(PROJECT_NAME).exe" \) -exec file {} \; 2>/dev/null; \
	else \
		echo "请先执行: make deploy"; \
	fi

# 创建启动脚本
create-startup-scripts:
	@echo "=== 创建启动脚本 ==="
	@echo '#!/bin/bash' > start.sh
	@echo 'cd "$$(dirname "$$0")"' >> start.sh
	@echo './gogw &' >> start.sh
	@chmod +x start.sh
	@echo "✅ 创建启动脚本: start.sh"
