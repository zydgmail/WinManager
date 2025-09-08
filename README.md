# WinManager

基于 Go + Vue3 的企业级远程桌面与设备管理解决方案。支持高性能视频编码、多协议通信和分布式设备管理。

## 项目架构

- **Backend (Go)**：设备管理、会话协调、WebSocket 网关、API 服务
- **Agent (Go)**：Windows 端屏幕采集、硬件编码、远程控制执行
- **Frontend (Vue3 + TypeScript)**：Web 管理界面，设备监控、会话控制

---

## 核心特性

### 设备管理

- 🔗 设备自动注册与心跳维护
- 📊 实时状态监控与离线检测
- 🗂️ 分组管理与批量操作
- 💾 SQLite/PostgreSQL 数据持久化

### 远程控制

- 🖥️ 高性能屏幕采集（DXGI/GDI）
- 🎞️ 多编码器支持：H.264、VP8、JPEG、JPEG-Turbo、NVENC
- ⌨️ 完整键鼠控制与剪贴板同步
- 🌐 WebSocket 实时通信与视频流传输

### 性能优化

- ⚡ 硬件加速编码（NVENC、Quick Sync）
- 📈 自适应码率与帧率控制
- 🚀 WebSocket 连接复用
- 📊 Prometheus 指标监控

### 系统管理

- 🔧 远程命令执行与脚本运行
- 🔄 远程重启/关机控制
- 📊 设备状态监控与心跳检测
- 📁 文件上传/下载管理

---

## 技术栈

### 后端技术

- **框架**：Gin (HTTP)、gRPC (RPC)、Gorilla WebSocket
- **数据库**：GORM (SQLite/PostgreSQL)
- **监控**：Prometheus、Logrus
- **配置**：JSON 配置 + CLI 参数

### Agent 核心

- **屏幕采集**：robotgo、DXGI、GDI+
- **视频编码**：x264、libvpx、libjpeg-turbo、FFmpeg
- **系统控制**：Windows API、WMI
- **通信协议**：gRPC、WebSocket、HTTP

### 前端技术

- **框架**：Vue 3.5 + TypeScript + Pinia
- **UI 组件**：Element Plus + TailwindCSS
- **构建工具**：Vite 6 + ESLint + Prettier
- **基础架构**：基于 [pure-admin](https://github.com/pure-admin/vue-pure-admin) 深度定制

---

## 目录结构

```text
WinManager/
  agent/           # 设备端（Windows）
  backend/         # 后端 API / WebSocket 服务
  frontend/        # 前端管理界面（Vue3 + Vite）
```

---

## 快速开始

### 1) 后端

#### 使用已构建二进制

```powershell
# 进入后端目录
cd backend

# 直接运行（默认 :8080）
./build/backend.exe --http=:8080
```

或从源码构建：

```powershell
cd backend
make build
./build/backend.exe --http=:8080
```

配置文件：`backend/config.json`

```json
{
  "database": { "path": "./data.db" },
  "server": { "port": ":8080" },
  "agent": {
    "http_port": 50052,
    "grpc_port": 50051,
    "heartbeat_timeout_seconds": 90,
    "offline_check_interval": 60
  },
  "log": {
    "level": "debug",
    "file": "./logs/backend.log",
    "max_size": 10,
    "max_backups": 5,
    "compress": false
  }
}
```

启动后将提供以下基础接口（前缀均为 `/api`）：

- 健康检查：`GET /api/health`
- 版本信息：`GET /api/version`
- 实例注册：`POST /api/register`
- 心跳上报：`PATCH /api/heartbeat/:id`
- 实例管理：`GET /api/instances` 等
- 分组管理：`GET /api/groups` 等
- WebSocket：`GET /api/ws/:id`、`GET /api/ws/:id/stream`
- 代理转发：`ANY /api/proxy/:id/*path`
- Agent 交互：`/api/agent/:id/*`（截图、启动/停止流、执行命令、WS 视频流等）

更多路由可参考 `backend/internal/controllers/router.go`。

---

### 2) Agent（Windows 端）

#### 快速部署（使用预构建）

```powershell
# 进入 Agent 目录
cd agent

# 配置后端地址（必须）
notepad config.json
# 修改 server.url 为实际后端地址，如：http://192.168.1.100:8080

# 直接运行
./build/full-agent.exe --debug
```

#### 从源码构建

**前置条件**：确保已安装 [环境要求](#环境要求) 中的开发环境

```bash
# 在 MSYS2 MINGW64 终端中执行
cd agent

# 1. 安装编码器依赖
make install-deps-windows

# 2. 验证依赖是否正确安装
pkg-config --exists x264 && echo "x264: OK" || echo "x264: 缺失"
pkg-config --exists vpx && echo "libvpx: OK" || echo "libvpx: 缺失"  
pkg-config --exists libjpeg && echo "libjpeg-turbo: OK" || echo "libjpeg-turbo: 缺失"

# 3. 构建完整版本（包含所有编码器）
make build-full

# 4. 运行测试
./build/full-agent.exe --debug
```

#### 构建标签说明

Agent 支持不同的构建标签以包含不同的编码器：

```bash
# 仅包含 H.264 编码器（最常用）
go build -tags "h264enc" -o build/agent-h264.exe .

# 包含所有软件编码器
go build -tags "h264enc,vp8enc,jpegturbo" -o build/agent-full.exe .

# 包含硬件编码器（需要 NVIDIA GPU）
go build -tags "h264enc,nvenc" -o build/agent-nvenc.exe .

# 不包含任何高级编码器（仅基础 JPEG）
go build -o build/agent-basic.exe .
```

#### 编码器依赖详解

| 编码器 | 库依赖 | 用途 | 性能 | 兼容性 |
|--------|--------|------|------|--------|
| **JPEG** | 内置 Go | 截图、低延迟 | 中 | 最佳 |
| **JPEG-Turbo** | libjpeg-turbo | 高速截图 | 高 | 优秀 |
| **H.264** | x264 | 视频流、高压缩 | 高 | 优秀 |
| **VP8** | libvpx | WebRTC 兼容 | 中 | 良好 |
| **NVENC** | FFmpeg+CUDA | 硬件加速 | 极高 | 仅 NVIDIA |

#### 常见构建问题

**问题 1**：`fatal error: x264.h: No such file or directory`

```bash
# 解决：确保安装了 x264 开发库
pacman -S mingw-w64-x86_64-x264
export PKG_CONFIG_PATH="/mingw64/lib/pkgconfig:$PKG_CONFIG_PATH"
```

**问题 2**：`undefined reference to 'x264_encoder_open'`

```bash
# 解决：确保 CGO 链接器能找到库文件
export CGO_LDFLAGS="-L/mingw64/lib"
export LD_LIBRARY_PATH="/mingw64/lib:$LD_LIBRARY_PATH"
```

**问题 3**：运行时 DLL 缺失

```bash
# 解决：复制必要的 DLL 到 agent/dll/ 目录
cp /mingw64/bin/libx264-*.dll ./dll/
cp /mingw64/bin/libvpx-*.dll ./dll/
cp /mingw64/bin/libjpeg-*.dll ./dll/
```

---

### 3) 前端（管理界面）

开发环境（默认通过 Vite 代理到 `http://localhost:8080`）：

```powershell
cd frontend
pnpm install
pnpm dev
```

生产构建：

```powershell
cd frontend
pnpm build
# 产物在 frontend/dist，可用任意静态服务器托管
npx serve -s dist -l 5173
```

前端基础配置：`frontend/public/platform-config.json`

```json
{
  "Version": "6.0.0",
  "Title": "WinManager",
  "FixedHeader": true,
  "Layout": "vertical",
  "Theme": "light",
  "DarkMode": false,
  "ShowLogo": true
}
```

Vite 代理（开发模式）位于 `frontend/vite.config.ts`：

```ts
proxy: {
  "/api": { target: "http://localhost:8080", changeOrigin: true, ws: true, secure: false }
}
```

---

## 配置详解

### Backend 配置 (`backend/config.json`)

```json
{
  "database": {
    "path": "./data.db"                    // SQLite 数据库文件路径
  },
  "server": {
    "port": ":9090"                        // HTTP 服务端口
  },
  "agent": {
    "http_port": 50052,                    // Agent HTTP 端口（用于截图、命令等）
    "grpc_port": 50051,                    // Agent gRPC 端口（用于流传输）
    "heartbeat_timeout_seconds": 90,       // 心跳超时时间（秒）
    "offline_check_interval": 60           // 离线检测间隔（秒）
  },
  "log": {
    "level": "debug",                      // 日志级别：debug, info, warn, error
    "file": "./logs/backend.log",          // 日志文件路径
    "max_size": 10,                        // 单个日志文件最大大小（MB）
    "max_backups": 5,                      // 保留的日志文件数量
    "compress": false                      // 是否压缩旧日志文件
  }
}
```

### Agent 配置 (`agent/config.json`)

```json
{
  "version": "1.0.0",                      // 配置版本
  "server": {
    "url": "http://172.17.1.242:9090",    // Backend 服务地址（必须配置）
    "timeout": 30,                         // 请求超时时间（秒）
    "retry_interval": 5                    // 重试间隔（秒）
  },
  "agent": {
    "http_port": 50052,                    // Agent HTTP 服务端口
    "grpc_port": 50051,                    // Agent gRPC 服务端口
    "debug": false,                        // 是否启用调试模式
    "log_level": "debug"                   // 日志级别
  },
  "screen": {
    "jpeg_quality": 80,                    // JPEG 质量 (1-100)
    "capture_method": "robotgo"            // 屏幕采集方法: robotgo, dxgi
  },
  "encoder": {
    "default_codec": "h264",               // 默认编码器
    "jpeg_quality": 80,                    // JPEG 编码质量
    "h264_preset": "medium",               // H.264 预设: ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow
    "h264_tune": "zerolatency",            // H.264 调优: film, animation, grain, stillimage, fastdecode, zerolatency
    "h264_profile": "baseline",            // H.264 配置: baseline, main, high
    "h264_bitrate": 200000,                // H.264 码率 (bps)
    "vp8_bitrate": 8192,                   // VP8 码率 (bps)
    "nvenc_bitrate": 50000000,             // NVENC 码率 (bps)
    "nvenc_preset": "fast",                // NVENC 预设: slow, medium, fast, hp, hq, bd, ll, llhq, llhp
    "frame_rate": 20,                      // 目标帧率 (fps)
    "enabled_codecs": ["h264", "jpeg", "jpeg-turbo", "vp8"],  // 启用的编码器列表
    "codec_priority": ["h264", "vp8", "jpeg-turbo", "jpeg"],  // 编码器优先级
    "custom_settings": {},                 // 自定义编码器设置
    "debug": {
      "save_path": "./debug",              // 调试文件保存路径
      "save_video_duration": 5             // 调试视频保存时长（秒）
    }
  },
  "input": {
    "mouse_enabled": true,                 // 是否启用鼠标控制
    "keyboard_enabled": true,              // 是否启用键盘控制
    "paste_enabled": true                  // 是否启用剪贴板粘贴
  },
  "proxy": {
    "enabled": false,                      // 是否启用代理
    "auto_detect": true,                   // 是否自动检测代理
    "url": "",                             // 代理 URL
    "username": "",                        // 代理用户名
    "password": ""                         // 代理密码
  },
  "monitoring": {
    "metrics_enabled": true,               // 是否启用指标收集
    "metrics_interval": 15,                // 指标收集间隔（秒）
    "health_check_enabled": true           // 是否启用健康检查
  },
  "system": {
    "reboot_enabled": true,                // 是否允许重启系统
    "reboot_delay": 3,                     // 重启延迟（秒）
    "shutdown_enabled": true,              // 是否允许关闭系统
    "commands_enabled": true               // 是否允许执行系统命令
  }
}
```

### Frontend 配置 (`frontend/public/platform-config.json`)

```json
{
  "Version": "6.0.0",                     // 版本号
  "Title": "WinManager",                  // 应用标题
  "FixedHeader": true,                    // 是否固定头部
  "Layout": "vertical",                   // 布局模式: vertical, horizontal
  "Theme": "light",                       // 主题: light, dark
  "DarkMode": false,                      // 是否启用暗色模式
  "ShowLogo": true                        // 是否显示 Logo
}
```

### 环境变量配置

```bash
# Agent 环境变量
export CGO_ENABLED=1                     # 启用 CGO（必须）
export WINMANAGER_SERVER_URL="http://backend:8080"  # Backend 地址
export WINMANAGER_DEBUG=true             # 调试模式
export WINMANAGER_LOG_LEVEL=debug        # 日志级别

# Backend 环境变量  
export WINMANAGER_PORT=:8080             # 服务端口
export WINMANAGER_DB_PATH="./data.db"    # 数据库路径
export WINMANAGER_LOG_LEVEL=info         # 日志级别
```

---

## 端口与网络

- Backend HTTP：默认 `:8080`
- Agent HTTP：默认 `:50052`
- Agent gRPC：默认 `:50051`
- 前端开发：默认 `:5173`（Vite）

确保后端与 Agent 之间网络互通（后端仅需要能访问到 Agent 的 `http_port` 与 `grpc_port`）。在 `backend/config.json` 中修改 `agent.http_port` 与 `agent.grpc_port`，并确保 Agent 端保持一致。

---

## API 快速验证

```powershell
# 健康检查
curl http://localhost:8080/api/health

# 版本
curl http://localhost:8080/api/version

# 实例列表
curl http://localhost:8080/api/instances
```

更多接口见 `backend/internal/controllers/router.go`。

---

## 环境要求

### 基础环境

- **Go**: 1.23+ （必须启用 CGO）
- **Node.js**: 18.18.0 / 20.9.0+ / 22.0.0+
- **包管理**: pnpm 9+
- **构建工具**: Make (Windows 推荐 MinGW64)

### Windows 开发环境（Agent 构建）

#### 方式一：MSYS2 (推荐)

```bash
# 安装 MSYS2 (https://www.msys2.org/)
# 打开 MSYS2 MINGW64 终端

# 安装构建工具链
pacman -S --needed base-devel mingw-w64-x86_64-toolchain
pacman -S --needed mingw-w64-x86_64-cmake
pacman -S --needed mingw-w64-x86_64-pkg-config

# 安装编码器依赖
pacman -S --needed mingw-w64-x86_64-x264
pacman -S --needed mingw-w64-x86_64-libvpx  
pacman -S --needed mingw-w64-x86_64-libjpeg-turbo
pacman -S --needed mingw-w64-x86_64-ffmpeg

# 设置环境变量
export CGO_ENABLED=1
export CC=x86_64-w64-mingw32-gcc
export CXX=x86_64-w64-mingw32-g++
```

#### 方式二：MinGW-w64 + 手动编译

```bash
# 安装 MinGW-w64 toolchain
# 手动编译或下载预编译的：
# - x264 (https://www.videolan.org/developers/x264.html)
# - libvpx (https://github.com/webmproject/libvpx)  
# - libjpeg-turbo (https://github.com/libjpeg-turbo/libjpeg-turbo)
# - FFmpeg (https://ffmpeg.org/download.html)

# 确保 CGO 可以找到头文件和库文件
export CGO_CFLAGS="-I/path/to/include"
export CGO_LDFLAGS="-L/path/to/lib"
export CGO_ENABLED=1
```

### Linux 开发环境

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install build-essential pkg-config
sudo apt-get install libx264-dev libvpx-dev libjpeg-turbo8-dev
sudo apt-get install libavcodec-dev libavformat-dev libavutil-dev

# CentOS/RHEL/Fedora
sudo yum groupinstall "Development Tools"
sudo yum install pkgconfig x264-devel libvpx-devel libjpeg-turbo-devel
sudo yum install ffmpeg-devel

# 设置 CGO
export CGO_ENABLED=1
```

### 环境验证

```bash
# 验证 Go 和 CGO
go version
go env CGO_ENABLED  # 应该返回 "1"

# 验证编译工具
gcc --version
make --version
pkg-config --version

# 验证编码器库 (Linux)
pkg-config --libs --cflags x264 libvpx 

# 验证编码器库 (Windows MSYS2)
pkg-config --libs --cflags x264 vpx libjpeg
```

---

## 故障排除

### 构建相关问题

#### CGO 编译错误

```bash
# 错误：CGO_ENABLED is not set
export CGO_ENABLED=1
go env CGO_ENABLED  # 确认返回 "1"

# 错误：gcc: command not found
# Windows: 安装 MinGW-w64 或 MSYS2
# Linux: sudo apt-get install build-essential
```

#### 编码器库缺失

```bash
# 错误：cannot find -lx264
# 确认库文件存在
find /usr -name "libx264*" 2>/dev/null  # Linux
find /mingw64 -name "libx264*" 2>/dev/null  # MSYS2

# 设置库路径
export PKG_CONFIG_PATH="/mingw64/lib/pkgconfig:$PKG_CONFIG_PATH"
export LD_LIBRARY_PATH="/mingw64/lib:$LD_LIBRARY_PATH"
```

#### Make 命令失败

```bash
# Windows 下确保在正确的环境中
# MSYS2: 使用 MSYS2 MINGW64 终端
# Git Bash: 可能缺少 make，建议使用 MSYS2

# 验证 Make 环境
make --version
which make
```

### 运行时问题

#### Agent 无法连接 Backend

```bash
# 1. 检查配置文件
cat agent/config.json | grep -A5 '"server"'
# 确保 server.url 指向正确的 Backend 地址

# 2. 网络连通性测试
curl http://192.168.1.100:8080/api/health
telnet 192.168.1.100 8080

# 3. 防火墙检查
# Windows: 检查 Windows Defender 防火墙
# Linux: 检查 iptables/ufw

# 4. 查看详细日志
./build/full-agent.exe --debug --log-level=debug
```

#### 视频流问题

```bash
# 黑屏或无法显示
# 1. 检查编码器可用性
./build/full-agent.exe --debug 2>&1 | grep -i encoder

# 2. 尝试不同的编码器
# 编辑 config.json:
"default_codec": "jpeg",  # 先用最兼容的
"enabled_codecs": ["jpeg"],

# 3. 降低性能要求
"frame_rate": 10,
"h264_bitrate": 100000,  # 降低码率
"jpeg_quality": 60       # 降低质量
```

#### DLL 缺失 (Windows)

```bash
# 错误：The program can't start because xxx.dll is missing
# 解决：复制 MSYS2 中的 DLL 到 agent/dll/
cp /mingw64/bin/libx264-*.dll ./dll/
cp /mingw64/bin/libvpx-*.dll ./dll/
cp /mingw64/bin/libjpeg-*.dll ./dll/
cp /mingw64/bin/libgcc_s_seh-1.dll ./dll/
cp /mingw64/bin/libwinpthread-1.dll ./dll/
```

### 性能调优

#### 降低 CPU 使用率

```json
// config.json - Agent
{
  "encoder": {
    "frame_rate": 15,           // 降低帧率
    "h264_preset": "fast",      // 使用更快的预设
    "default_codec": "jpeg"     // 使用 JPEG 而非 H.264
  }
}
```

#### 降低内存使用

```json
{
  "screen": {
    "capture_method": "robotgo"  // 而非 dxgi
  },
  "encoder": {
    "h264_profile": "baseline"   // 使用基础配置
  }
}
```

#### 提高编码性能

```json
{
  "encoder": {
    "default_codec": "nvenc",    // 使用硬件编码器
    "nvenc_preset": "fast",
    "enabled_codecs": ["nvenc", "h264", "jpeg"]
  }
}
```

### 日志调试

#### 启用详细日志

```bash
# Agent 详细日志
./build/full-agent.exe --debug --log-level=debug

# Backend 详细日志
./build/backend.exe --log-level=debug

# 查看特定模块日志
./build/full-agent.exe --debug 2>&1 | grep -i "encoder\|stream\|error"
```

#### 日志文件位置

```bash
# Backend 日志
ls -la backend/logs/backend*.log

# Agent 日志 (如果配置了文件输出)
ls -la agent/logs/agent*.log

# 系统日志 (Linux)
journalctl -u winmanager-agent
```

### 常见问题 FAQ

**Q: 为什么必须启用 CGO？**
A: Agent 需要调用 C 库（x264、libvpx 等）进行硬件编码，这些库只能通过 CGO 访问。

**Q: 可以在没有编码器的情况下运行吗？**
A: 可以，使用基础构建：`go build -o agent-basic.exe .`，仅支持基础 JPEG 编码。

**Q: 如何选择最佳编码器？**
A:

- 兼容性优先：JPEG
- 性能优先：NVENC (需要 N 卡)
- 平衡选择：H.264

**Q: 前端无法连接怎么办？**

A:

- 开发环境：确保 `pnpm dev` 正常启动，检查 Vite 代理配置
- 生产环境：确保静态文件正确部署，API 路径正确

**Q: 多台 Agent 如何管理？**
A: 每台 Agent 都需要指向同一个 Backend，Backend 会自动识别和管理多个设备。

---

## 免责声明

本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。任何人或组织不得将本仓库的内容用于非法用途或侵犯他人合法权益。因使用本项目造成的任何直接或间接损失，作者不承担责任。

---

## 许可证

本项目采用 Apache License 2.0 开源许可证。详见根目录的 `LICENSE` 文件。

简要说明：

- 您可以自由使用、复制、修改、分发本项目
- 必须在分发时保留许可证与版权声明
- 若修改了源码或以本项目为基础再次分发，应明确说明变更
- 许可证不授予商标使用权

---

## 致谢

- 社区开源库与编码器组件（x264、libvpx、libjpeg-turbo、FFmpeg 等）
