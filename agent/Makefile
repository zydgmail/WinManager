# WinManager Agent Makefile

BINARY_NAME=agent.exe
BUILD_DIR=build

# Build tags for different encoder support (temporarily disable nvenc due to FFmpeg compatibility)
BUILD_TAGS_FULL=h264enc # ,vp8enc,jpegturbo

# 完整构建 (包含所有编码器)
.PHONY: build-full
build-full:
	@mkdir -p $(BUILD_DIR)
	@echo "Building full version (all encoders)..."
	@echo "Note: Requires x264, libvpx, libjpeg-turbo, and FFmpeg libraries"
	@go build -tags "$(BUILD_TAGS_FULL)" -ldflags "-X main.autoRegister=true -H windowsgui" -o $(BUILD_DIR)/full-$(BINARY_NAME) .

# 运行 (调试模式，自动注册，所有编码器)
.PHONY: run
run: build-full
	@echo "Running in debug mode with auto-registration and all encoders..."
	@$(BUILD_DIR)/full-$(BINARY_NAME) --debug

# 安装 Windows 编码器依赖
.PHONY: install-deps-windows
install-deps-windows:
	@echo "Installing encoder dependencies for Windows (MSYS2)..."
	@echo "Make sure you're running this in MSYS2 shell"
	pacman -S --needed mingw-w64-x86_64-x264
	pacman -S --needed mingw-w64-x86_64-libvpx
	pacman -S --needed mingw-w64-x86_64-libjpeg-turbo
	pacman -S --needed mingw-w64-x86_64-ffmpeg
	pacman -S --needed mingw-w64-x86_64-pkg-config

# 帮助
.PHONY: help
help:
	@echo "常用命令:"
	@echo "  build-full  - 构建程序 (所有编码器)"
	@echo "  run         - 构建并运行 (调试模式，自动注册，所有编码器)"
	@echo "  install-deps-windows - 安装 Windows 编码器依赖"
