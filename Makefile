# HuShu智能网关 - Makefile
# 支持多平台交叉编译

.PHONY: all clean build build-mini test fmt vet ui ui-install ui-dev run help northbound-plugins \
        deploy deploy-arm32 deploy-arm64 deploy-darwin deploy-darwin-arm64 deploy-windows

# 默认目标: 编译前端 + 本地运行
all: ui run
	@echo ""
	@echo "========================================"
	@echo "  ✅ 前端构建完成，后端已启动!"
	@echo "========================================"
	@echo ""
	@echo "访问 http://localhost:8080 查看应用"
	@echo "按 Ctrl+C 停止服务器"

# 帮助信息
help:
	@echo "HuShu智能网关 - 编译部署工具"
	@echo ""
	@echo "可用目标:"
	@echo "  (无参数)      - 编译前端 + 启动后端 (默认)"
	@echo "  all           - 同默认目标"
	@echo "  build         - 编译当前平台后端 (CGO=0)"
	@echo "  build-mini    - 编译最小体积后端 (trimpath + 精简 ldflags)"
	@echo "  northbound-plugins - 编译北向插件"
	@echo "  test          - go test ./..."
	@echo "  fmt           - gofmt + goimports"
	@echo "  vet           - go vet"
	@echo "  ui-install    - 安装前端依赖 (SolidJS)"
	@echo "  ui            - 构建前端 (SolidJS)"
	@echo "  ui-dev        - 前端开发服务器 (热重载)"
	@echo "  run           - 仅启动后端服务"
	@echo "  clean         - 清理构建产物"
	@echo "  deploy        - 部署到所有平台"
	@echo "  deploy-arm32  - 部署Linux ARM32版本"
	@echo "  deploy-arm64  - 部署Linux ARM64版本"
	@echo "  deploy-darwin - 部署macOS版本"
	@echo "  deploy-darwin-arm64 - 部署macOS ARM64版本"
	@echo "  deploy-windows - 部署Windows版本"
	@echo ""
	@echo "用法:"
	@echo "  make              # 前端构建 + 启动后端"
	@echo "  make all          # 同 make"
	@echo "  make ui-dev       # 前端开发服务器 (热重载)"
	@echo "  make deploy       # 编译所有平台"

# 项目名称和版本
PROJECT_NAME := gogw
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date +%Y%m%d-%H%M%S)
NORTHBOUND_PLUGIN_DIR := plugin_north
NORTHBOUND_PLUGIN_CMDS := northbound-xunji northbound-http northbound-mqtt
NORTHBOUND_PLUGINS := $(addprefix $(NORTHBOUND_PLUGIN_DIR)/,$(NORTHBOUND_PLUGIN_CMDS))

# 源文件
MAIN_SRC := cmd/main.go
CONFIG_SRC := config/config.yaml
MIGRATIONS_DIR := migrations
WEB_DIR := web
DATA_DIR := data

# 部署目录
DEPLOY_DIR := deploy

# 构建优化参数
COMMON_LDFLAGS := -s -w -buildid=
BUILD_FLAGS := -trimpath -ldflags "$(COMMON_LDFLAGS)"

# 编译当前平台版本
build:
	@echo "=== 构建 $(PROJECT_NAME) $(VERSION) ==="
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(PROJECT_NAME) $(MAIN_SRC)
	@echo "✅ 构建完成: $(PROJECT_NAME)"

# 编译最小体积版本
build-mini:
	@echo "=== 构建最小体积 $(PROJECT_NAME) $(VERSION) ==="
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(PROJECT_NAME) $(MAIN_SRC)
	@echo "✅ 构建完成: $(PROJECT_NAME)"
	@if command -v upx >/dev/null 2>&1; then \
		echo "=== 使用 upx 压缩可执行文件 ==="; \
		upx --best --lzma $(PROJECT_NAME) >/dev/null 2>&1 || true; \
	fi
	@ls -lh $(PROJECT_NAME)

# 北向插件编译
northbound-plugins: $(NORTHBOUND_PLUGINS)
	@echo "✅ 北向插件构建完成: $(NORTHBOUND_PLUGIN_DIR)"

$(NORTHBOUND_PLUGIN_DIR)/northbound-%: plugin_north/src/northbound-%/main.go
	@mkdir -p $(NORTHBOUND_PLUGIN_DIR)
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $@ ./plugin_north/src/northbound-$*
	@if command -v upx >/dev/null 2>&1; then upx --best --lzma $@ >/dev/null 2>&1 || true; fi

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')
	@if command -v goimports >/dev/null 2>&1; then goimports -w $$(find . -name '*.go' -not -path './vendor/*'); else echo "goimports 未安装，跳过"; fi

vet:
	go vet ./...

# 前端依赖安装 (SolidJS)
ui-install:
	@echo "=== 安装前端依赖 (SolidJS) ==="
	@if [ -d "ui/frontend/node_modules" ]; then \
		echo "依赖已存在，使用 npm install 更新..."; \
		npm --prefix ui/frontend install; \
	else \
		npm --prefix ui/frontend install; \
	fi
	@echo "✅ 前端依赖安装完成"

# 前端构建 (SolidJS)
ui:
	@echo "=== 构建前端 (SolidJS) ==="
	@if [ ! -d "ui/frontend/node_modules" ]; then \
		echo "依赖未安装，先执行: make ui-install"; \
		exit 1; \
	fi
	npm --prefix ui/frontend run build
	@echo "✅ 前端构建完成: ui/static/dist/main.js"

# 前端开发服务器 (热重载)
ui-dev:
	@echo "=== 启动前端开发服务器 (SolidJS) ==="
	@echo "访问 http://localhost:5173 查看前端"
	@echo "按 Ctrl+C 停止服务器"
	@echo ""
	npm --prefix ui/frontend run dev --host

# 启动后端服务
run:
	@echo "=== 启动服务 (go run ./cmd/main.go) ==="
	@echo "访问 http://localhost:8080"
	@echo "按 Ctrl+C 停止服务"
	@echo ""
	go run ./cmd/main.go

# 清理构建产物
clean:
	@echo "=== 清理构建产物 ==="
	rm -f $(PROJECT_NAME)
	rm -rf $(DEPLOY_DIR)
	rm -rf ui/frontend/node_modules
	rm -rf ui/frontend/dist
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
deploy-arm32: prepare-deploy ui
	@echo "=== 编译 Linux ARM32 版本 ==="
	CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/arm32/$(PROJECT_NAME) $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/arm32/
	cp -r migrations $(DEPLOY_DIR)/arm32/
	cp -r web $(DEPLOY_DIR)/arm32/
	@echo "✅ ARM32 部署包已生成: $(DEPLOY_DIR)/arm32/"

# Linux ARM64 编译
deploy-arm64: prepare-deploy ui
	@echo "=== 编译 Linux ARM64 版本 ==="
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/arm64/$(PROJECT_NAME) $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/arm64/
	cp -r migrations $(DEPLOY_DIR)/arm64/
	cp -r web $(DEPLOY_DIR)/arm64/
	@echo "✅ ARM64 部署包已生成: $(DEPLOY_DIR)/arm64/"

# macOS (Intel/AMD64) 编译
deploy-darwin: prepare-deploy ui
	@echo "=== 编译 macOS 版本 ==="
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/darwin/$(PROJECT_NAME) $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/darwin/
	cp -r migrations $(DEPLOY_DIR)/darwin/
	cp -r web $(DEPLOY_DIR)/darwin/
	@echo "✅ macOS 部署包已生成: $(DEPLOY_DIR)/darwin/"

# macOS ARM64 (Apple Silicon) 编译
deploy-darwin-arm64: prepare-deploy ui
	@echo "=== 编译 macOS ARM64 版本 ==="
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/darwin/$(PROJECT_NAME)-arm64 $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/darwin/
	cp -r migrations $(DEPLOY_DIR)/darwin/
	cp -r web $(DEPLOY_DIR)/darwin/
	@echo "✅ macOS ARM64 部署包已生成: $(DEPLOY_DIR)/darwin/"

# Windows AMD64 编译
deploy-windows: prepare-deploy ui
	@echo "=== 编译 Windows 版本 ==="
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(COMMON_LDFLAGS) -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(DEPLOY_DIR)/windows/$(PROJECT_NAME).exe $(MAIN_SRC)
	@echo "复制配置文件..."
	cp -f config/config.yaml $(DEPLOY_DIR)/windows/
	cp -r migrations $(DEPLOY_DIR)/windows/
	cp -r web $(DEPLOY_DIR)/windows/
	@echo "✅ Windows 部署包已生成: $(DEPLOY_DIR)/windows/"

# 部署到所有平台
deploy: prepare-deploy ui
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
