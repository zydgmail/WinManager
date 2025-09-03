# WinManager Backend Makefile

BINARY_NAME=backend.exe
BUILD_DIR=build

# 标准构建
.PHONY: build
build:
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .

# 开发构建 (包含调试信息)
.PHONY: build-dev
build-dev:
	@mkdir -p $(BUILD_DIR)
	@go build -race -o $(BUILD_DIR)/dev-$(BINARY_NAME) .

# 构建 Linux 可执行文件
.PHONY: build-linux
build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/linux-backend .

# 运行
.PHONY: run
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) --http=:8080

# 帮助
.PHONY: help
help:
	@echo "常用命令:"
	@echo "  build     		- 构建程序"
	@echo "  build-dev 		- 构建开发版本"
	@echo "  build-linux    - 构建Linux环境版本"
	@echo "  run       		- 构建并运行"
