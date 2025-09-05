# WinManager

跨平台远程桌面与设备管理解决方案。项目由三部分组成：

- Backend（Go）：设备与会话管理、WebSocket 转发、HTTP 代理与统一 API
- Agent（Go）：运行在受控 Windows 设备上的屏幕采集、编码与控制执行
- Frontend（Vue3 + Vite）：管理端 UI，提供设备分组、在线状态、会话控制等

---

## 功能特性

- 设备注册与心跳，离线自动检测与状态维护（SQLite 默认存储）
- 实例与分组管理，增删改查接口完善
- WebSocket 会话：Guacamole 协议转发、视频流代理、状态管理
- Windows 屏幕采集与多编码器（H264、VP8、JPEG/JPEG-Turbo），帧率/码率可配
- 键鼠输入、剪贴板粘贴等控制能力（可独立开关）
- 指标采集与性能监控（Prometheus 客户端集成）
- 前端开发代理到后端 API，生产构建体积优化

---

## 技术栈与来源

- 前端：基于 [pure-vue-admin](https://github.com/pure-admin/vue-pure-admin) 进行二次开发与集成
- 后端：基于 [Gin](github.com/gin-gonic/gin) 框架构建路由、配置与服务能力
- Agent：基于 [robotgo](https://github.com/go-vgo/robotgo) 实现屏幕采集与输入控制

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

### 2) Agent（Windows）

#### 使用已构建二进制运行

```powershell
# 进入 Agent 目录
cd agent

# 修改配置（最重要的是 server.url 指向后端）
notepad.exe .\config.json

# 运行（读取默认 config.json）
./build/full-agent.exe --debug
```

示例配置：`agent/config.json`

```json
{
  "version": "1.0.0",
  "server": {
    "url": "http://127.0.0.1:8080",
    "timeout": 30,
    "retry_interval": 5
  },
  "agent": {
    "http_port": 50052,
    "grpc_port": 50051,
    "debug": true,
    "log_level": "debug"
  },
  "screen": { "jpeg_quality": 80, "capture_method": "robotgo" },
  "encoder": {
    "default_codec": "h264",
    "jpeg_quality": 80,
    "h264_preset": "medium",
    "h264_tune": "zerolatency",
    "h264_profile": "baseline",
    "h264_bitrate": 200000,
    "vp8_bitrate": 8192,
    "nvenc_bitrate": 50000000,
    "nvenc_preset": "fast",
    "frame_rate": 20,
    "enabled_codecs": ["h264", "jpeg", "jpeg-turbo", "vp8"],
    "codec_priority": ["h264", "vp8", "jpeg-turbo", "jpeg"],
    "custom_settings": {},
    "debug": { "save_path": "./debug", "save_video_duration": 5 }
  },
  "input": { "mouse_enabled": true, "keyboard_enabled": true, "paste_enabled": true },
  "proxy": { "enabled": false, "auto_detect": true, "url": "", "username": "", "password": "" },
  "monitoring": { "metrics_enabled": true, "metrics_interval": 15, "health_check_enabled": true }
}
```

常用启动参数（命令行优先级高于配置）：

```powershell
./build/full-agent.exe ^
  --config .\config.json ^
  --server http://127.0.0.1:8080 ^
  --grpc :50051 ^
  --http :50052 ^
  --debug
```

从源码构建（Windows，MSYS2 推荐）：

```powershell
cd agent
make install-deps-windows   # 安装 x264 / libvpx / libjpeg-turbo / ffmpeg 等依赖
make build-full              # 构建完整版本（包含编码器）
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

## 开发环境要求

- Go 1.23+
- Node.js 18/20/22，pnpm 9+
- Windows 平台建议通过 MSYS2 安装编码器依赖（Agent 编码器构建）

---

## 常见问题（FAQ）

- Agent 无法注册/心跳不更新？
  - 检查 `agent/config.json` 中的 `server.url` 是否可从 Agent 访问（例如内网 IP）。
  - 后端是否监听在正确地址（`--http=:8080`）且防火墙放行？
- 视频流黑屏/延迟高？
  - 调整 `encoder.frame_rate`、`h264_bitrate`、`default_codec` 与 `codec_priority`。
  - 使用 `jpeg`/`jpeg-turbo` 做兼容性验证。
- 前端 404 或跨域？
  - 开发模式下使用 `pnpm dev`，依赖 Vite 代理 `/api` 到后端。
  - 生产环境请将 `frontend/dist` 以静态资源方式部署，API 走后端域名的 `/api`。

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
