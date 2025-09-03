# WinManager Agent

远程桌面控制代理，提供屏幕捕获、输入模拟和系统管理功能。

## 功能

- 🖥️ 多种屏幕捕获方式 (RobotGo, DXGI, GDI, WGC)
- 🎥 多编码器支持 (JPEG, JPEG Turbo, H.264, VP8, NVENC)
- 🖱️ 鼠标键盘控制
- 📊 系统信息收集
- 🌐 HTTP/gRPC 双协议
- 📈 性能监控
- ⚡ 硬件加速编码 (NVIDIA NVENC)
- 🔧 条件编译支持

## 环境要求

### 基础要求
- Go 1.23+
- Git (Go modules 依赖)
- CGO 环境 (`set CGO_ENABLED=1`)

### Windows 环境

#### 方法 1: 自动安装 (推荐)
```bash
# 1. 下载安装 MSYS2: https://www.msys2.org/
# 2. 打开 MSYS2 Shell，运行自动安装
make install-deps-windows

# 3. 设置环境变量
export CGO_ENABLED=1
export PATH=$PATH:/mingw64/bin
export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:/mingw64/lib/pkgconfig
```

#### 方法 2: 手动安装
```bash
# 1. 下载安装 MSYS2: https://www.msys2.org/
# 2. 打开 MSYS2 Shell，安装工具链
pacman -S mingw-w64-x86_64-toolchain
pacman -S mingw-w64-x86_64-libvpx      # VP8 编码器
pacman -S mingw-w64-x86_64-libjpeg-turbo # JPEG Turbo
pacman -S mingw-w64-x86_64-x264        # H.264 编码器
pacman -S mingw-w64-x86_64-ffmpeg      # NVENC 支持
pacman -S mingw-w64-x86_64-pkg-config  # 包配置
pacman -S make git

# 3. 设置环境变量
export CGO_ENABLED=1
export PATH=$PATH:/mingw64/bin
export PKG_CONFIG_PATH=$PKG_CONFIG_PATH:/mingw64/lib/pkgconfig
```

#### 检查依赖
```bash
# 检查所有编码器依赖是否安装
make check-deps
```

### Linux (Ubuntu) 环境
```bash
# 安装依赖
apt-get install libvips-dev

# 检查版本 (需要 libvips 8.10+)
pkg-config --modversion vips
vips --version
```

## 快速开始

### 构建

#### 基础构建 (仅 JPEG 编码器)
```bash
# 使用 Go 直接构建
go build -o build/winmanager-agent .

# 或使用 make
make build
```

#### 完整构建 (所有编码器)
```bash
# 构建包含所有编码器的版本
make build-full

# 或手动指定构建标签
go build -tags "jpegturbo,h264enc,vp8enc,nvenc" -o build/full-agent .
```

#### 特定编码器构建
```bash
make build-h264     # 仅 H.264 编码器
make build-turbo    # 仅 JPEG Turbo 编码器
make build-vp8      # 仅 VP8 编码器
make build-nvenc    # 仅 NVENC 编码器
```

### 运行
```bash
# 调试模式
./build/winmanager-agent --debug

# 连接服务器
./build/winmanager-agent --server http://localhost:8080
```

## 命令行选项

- `--debug, -d`: 调试模式
- `--server, -s`: 服务器地址
- `--http, -a`: HTTP端口 (默认:8080)
- `--grpc, -g`: gRPC端口 (默认:50051)

## API

### HTTP 接口

#### 基础接口
- `GET /health` - 健康检查
- `GET /api/info` - 系统信息
- `GET /metrics` - 监控指标

#### 屏幕捕获接口
- `GET /api/screenshot` - 传统屏幕截图 (JPEG/PNG)
- `GET /api/encoded-screenshot` - 编码器屏幕截图
- `GET /api/encoders` - 编码器信息
- `GET /api/stream` - 视频流 (计划中)

#### 编码器参数
```bash
# 获取编码器信息
curl http://localhost:50052/api/encoders

# JPEG 截图
curl http://localhost:50052/api/encoded-screenshot?codec=jpeg&quality=80

# H.264 编码截图
curl http://localhost:50052/api/encoded-screenshot?codec=h264&method=auto

# NVENC 硬件编码
curl http://localhost:50052/api/encoded-screenshot?codec=nvenc&quality=95
```

### gRPC 服务
- `Mouse` - 鼠标控制
- `Key` - 键盘控制
- `Screenshot` - 屏幕捕获
- `Paste` - 剪贴板操作

## 编码器支持

### 支持的编码器

| 编码器 | 构建标签 | 描述 | 性能 |
|--------|----------|------|------|
| JPEG | (默认) | 标准 JPEG 编码 | 中等 |
| JPEG Turbo | `jpegturbo` | 高性能 JPEG 编码 | 高 |
| H.264 | `h264enc` | x264 视频编码 | 高 |
| VP8 | `vp8enc` | WebRTC 兼容编码 | 中等 |
| NVENC | `nvenc` | NVIDIA 硬件编码 | 极高 |

### 编码器配置

编码器可通过 `config.json` 配置：

```json
{
  "encoder": {
    "default_codec": "jpeg",
    "jpeg_quality": 80,
    "h264_preset": "superfast",
    "h264_tune": "zerolatency",
    "vp8_bitrate": 8192,
    "nvenc_bitrate": 50000000,
    "frame_rate": 30,
    "enabled_codecs": ["jpeg", "jpeg-turbo", "h264", "vp8", "nvenc"],
    "codec_priority": ["nvenc", "h264", "vp8", "jpeg-turbo", "jpeg"]
  }
}
```

### 屏幕捕获方法

| 方法 | 平台 | 性能 | 兼容性 |
|------|------|------|--------|
| RobotGo | 跨平台 | 中等 | 高 |
| DXGI | Windows 8+ | 高 | 中等 |
| GDI | Windows | 低 | 高 |
| WGC | Windows 10+ | 极高 | 低 |

### 构建选项
```bash
make build          # 基础版本 (仅 JPEG)
make build-full     # 完整版本 (所有编码器)
make build-dev      # 开发版本 (包含调试信息)
make build-h264     # H.264 编码器版本
make build-turbo    # JPEG Turbo 版本
make build-vp8      # VP8 编码器版本
make build-nvenc    # NVENC 版本
```

### 运行和测试
```bash
make run            # 调试模式运行
make run-prod       # 生产模式运行
make test           # 运行测试
make test-encoders  # 测试编码器
```

## 常见问题

### 编译错误
```bash
# Git 未安装
exec: "git": executable file not found in %PATH%
# 解决: 安装 Git 或设置代理
go env -w GOPROXY=https://goproxy.cn,direct

# robotgo 编译失败
undefined: Bitmap
# 解决: 安装完整的 MinGW 工具链 (见上方环境要求)

# CGO 未启用
CGO_ENABLED=0
# 解决: set CGO_ENABLED=1
```

### 运行错误
```bash
# 服务器注册失败
json: cannot unmarshal number into Go struct field
# 解决: 已修复，支持多种服务器响应格式
```

## 项目结构

```
agent/
├── main.go              # 入口文件
├── internal/            # 私有代码
│   ├── api/            # API 实现
│   ├── config/         # 配置管理
│   ├── controllers/    # HTTP 控制器
│   ├── handlers/       # 请求处理
│   └── logger/         # 日志配置
├── pkg/                # 公共包
│   ├── device/         # 设备信息
│   ├── encoders/       # 编码器实现
│   ├── input/          # 输入模拟
│   └── screen/         # 屏幕捕获
├── protos/             # gRPC 定义
└── build/              # 构建输出
```

## 监控指标

Agent 在 `/metrics` 端点提供 Prometheus 指标：

- `agent_cpu_usage_percent` - CPU 使用率
- `agent_memory_usage_bytes` - 内存使用量 (字节)
- `agent_memory_total_bytes` - 总内存 (字节)
- `agent_goroutines_count` - 协程数量
- `agent_uptime_seconds` - 运行时间 (秒)
- `agent_requests_total` - HTTP 请求总数
- `agent_request_duration_seconds` - 请求耗时分布

## 安全注意事项

- Agent 需要适当的用户权限运行
- 生产环境应使用 TLS 加密网络通信
- 访问控制应在服务器端实现
- 定期应用安全更新

## 故障排除

### 常见问题

1. **权限拒绝**: 确保 Agent 有屏幕捕获和输入模拟的必要权限
2. **网络问题**: 检查防火墙设置和代理配置
3. **CPU 使用率高**: 监控编码设置和屏幕捕获频率

### 日志

- 调试模式: 输出到控制台
- 生产模式: 输出到 `logs/agent.log` 并自动轮转

## 开发贡献

1. Fork 仓库
2. 创建功能分支
3. 提交更改
4. 添加测试 (如适用)
5. 提交 Pull Request
