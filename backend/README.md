# WinManager Backend

WinManager 后端服务器，提供实例管理和分组管理功能。

## 功能特性

- **实例管理**: 注册、更新、删除 Windows 实例
- **分组管理**: 创建和管理实例分组
- **WebSocket 串流**: 支持 Guacamole 远程桌面协议
- **代理功能**: HTTP 和 WebSocket 代理转发
- **多实例控制**: 支持并行操作多个实例
- **SQLite 数据库**: 轻量级数据存储
- **中文日志格式**: 便于调试和监控

## 系统架构

### 整体架构流程图

```mermaid
graph TB
    subgraph "前端 Web Browser"
        A[Vue.js 前端页面]
        B[WebsocketStream.vue]
        C[JMuxer 视频解码器]
        D[Guacamole.js 键鼠处理]
    end

    subgraph "Backend 后端服务器"
        E[Gin HTTP Server :8080]
        F[/ws/:id 路由]
        G[/ws/state/:id 路由]
        H[/ws/:id/stream 代理路由]
        I[Guacamole 协议处理]
        J[WebSocket 升级器]
        K[代理转发模块]
    end

    subgraph "Guacamole 中间件"
        L[Guacd 服务器 :4822]
        M[协议转换器]
        N[VNC/RDP 客户端]
    end

    subgraph "Agent 客户端机器"
        O[Agent HTTP Server :50052]
        P[/wsstream WebSocket 端点]
        Q[屏幕截取模块]
        R[H.264 编码器]
        S[鼠标键盘控制]
        T[DXGI/GDI 屏幕抓取]
        U[VNC Server :5900]
    end

    %% 视频流路径
    A -->|1. 请求视频流| B
    B -->|2. WebSocket连接<br/>ws://backend/ws/192.168.1.100/stream| H
    H -->|3. 代理转发<br/>ws://192.168.1.100:50052/wsstream| P
    P -->|4. 启动屏幕抓取| Q
    Q -->|5. 截取屏幕| T
    T -->|6. 原始图像数据| R
    R -->|7. H.264编码| P
    P -->|8. 编码后视频流| H
    H -->|9. 转发视频流| B
    B -->|10. 解码显示| C

    %% 控制流路径
    A -->|11. 键鼠事件| D
    D -->|12. Guacamole协议<br/>ws://backend/ws/state/192.168.1.100| G
    G -->|13. 协议转换| I
    I -->|14. 连接Guacd| L
    L -->|15. 协议转换| M
    M -->|16. VNC协议| N
    N -->|17. VNC连接| U
    U -->|18. 系统调用| S
    S -->|19. 鼠标键盘操作| T
```

### 三者角色分工

#### 🌐 **Web 前端**
- **视频显示**: 使用 JMuxer 解码 H.264 视频流
- **用户交互**: 捕获键盘鼠标事件
- **协议处理**: 使用 Guacamole.js 处理远程桌面协议
- **连接管理**: 管理两个 WebSocket 连接（视频流 + 控制流）

#### 🔄 **Backend 后端**
- **路由代理**: 将前端请求代理到对应的 Agent
- **协议转换**: Guacamole 协议与 VNC/RDP 协议转换
- **连接管理**: 管理前端与 Agent 之间的 WebSocket 连接
- **负载均衡**: 处理多个 Agent 的连接和状态同步

#### 💻 **Agent 客户端**
- **屏幕抓取**: 使用 DXGI/GDI 实时截取屏幕
- **视频编码**: H.264 硬件/软件编码
- **输入模拟**: 接收并执行鼠标键盘操作
- **VNC 服务**: 提供 VNC 服务供 Guacamole 连接

### 数据流说明

#### 📹 **视频流路径** (Agent → Web)
```
Agent屏幕 → DXGI抓取 → H.264编码 → WebSocket → Backend代理 → Web解码显示
```

#### 🎮 **控制流路径** (Web → Agent)
```
Web键鼠 → Guacamole.js → WebSocket → Backend → Guacd → VNC → Agent执行
```

### 关键技术特点

1. **双协议设计**:
   - 视频流: 直接 WebSocket + H.264 (高效)
   - 控制流: Guacamole 协议 + VNC (标准化)

2. **代理架构**:
   - Backend 作为代理，前端不直接连接 Agent
   - 支持内网访问和连接管理

3. **实时性优化**:
   - DXGI 硬件加速屏幕抓取
   - H.264 硬件编码
   - WebSocket 二进制传输

4. **多实例同步**:
   - 支持同时控制多个 Agent
   - 状态同步和并行操作

## 项目结构

```
backend/
├── main.go              # 主程序入口
├── go.mod              # Go模块定义
├── go.sum              # Go模块依赖锁定
├── Makefile            # 构建脚本
├── README.md           # 项目说明
├── config.json         # 配置文件
├── build/              # 构建输出目录
├── logs/               # 日志文件目录
├── internal/           # 内部包
│   ├── config/         # 配置管理
│   ├── models/         # 数据模型
│   └── controllers/    # 控制器
└── data.db             # SQLite数据库文件
```

## 环境要求

- Go 1.23.0 或更高版本
- Windows 操作系统（推荐使用 MSYS2/MinGW64）

## 安装和运行

### 1. 克隆项目

```bash
git clone <repository-url>
cd backend
```

### 2. 安装依赖

```bash
make deps
```

### 3. 构建项目

```bash
make build
```

### 4. 运行项目

```bash
make run
```

或者直接开发模式运行：

```bash
make dev
```

## 配置说明

项目使用 `config.json` 文件进行配置：

```json
{
  "database": {
    "path": "./data.db"
  },
  "server": {
    "port": ":8080"
  },
  "log": {
    "level": "debug",
    "file": "./logs/backend.log"
  }
}
```

## API 接口

### 实例管理

- `GET /api/instances` - 获取实例列表
- `GET /api/instances/:id` - 获取单个实例
- `POST /api/register` - 注册新实例
- `PATCH /api/instances/:id` - 更新实例信息
- `DELETE /api/instances/:id` - 删除实例
- `PATCH /api/instances/move-group` - 移动实例到分组

### 分组管理

- `GET /api/groups` - 获取分组列表
- `GET /api/groups/:id` - 获取单个分组
- `POST /api/groups` - 创建新分组
- `PATCH /api/groups/:id` - 更新分组信息
- `DELETE /api/groups/:id` - 删除分组

### WebSocket 串流

- `GET /api/ws/:id` - Guacamole远程桌面连接
- `GET /api/ws/state/:id` - 状态同步和多实例控制
- `GET /api/ws/:id/stream` - 视频流代理连接

### HTTP 代理

- `ANY /api/proxy/:id/*path` - HTTP请求代理转发
- `GET /api/stream/:id/start` - 启动视频流
- `GET /api/stream/:id/stop` - 停止视频流

### Agent 交互

- `POST /api/instances/:id/screenshot` - 获取截图
- `POST /api/instances/:id/execute` - 执行命令
- `GET /api/instances/:id/system/info` - 获取系统信息

### WebSocket 状态管理

- `GET /api/websocket/stats` - 获取连接统计
- `GET /api/websocket/instances/:id` - 获取实例连接
- `DELETE /api/websocket/instances/:id` - 关闭实例连接
- `DELETE /api/websocket/connections/:conn_id` - 关闭指定连接

### 系统接口

- `GET /api/health` - 健康检查
- `GET /api/version` - 版本信息

## 开发指南

### 构建项目

```bash
make build
```

### 开发模式运行

```bash
go run . --http=:8080
```

## 日志格式

项目使用中文日志格式：

```
日志等级-[日期时间]-[文件路径]-[函数名]-日志消息
```

示例：
```
INFO-[2024-01-01 12:00:00]-[main.go]-[init]-启动 winmanager-backend 版本 1.0.0
```

## 数据库

项目使用 SQLite 数据库，数据库文件位于 `./data.db`。

### 数据表结构

#### instances 表
- id: 主键
- uuid: 设备唯一标识
- os: 操作系统
- arch: 架构
- lan: 内网IP
- wan: 外网IP
- mac: MAC地址
- cpu: CPU信息
- cores: CPU核心数
- memory: 内存大小
- uptime: 运行时间
- hostname: 主机名
- username: 用户名
- status: 状态
- version: Agent版本
- watchdog_version: Watchdog版本
- group_id: 分组ID
- created_at: 创建时间
- updated_at: 更新时间

#### groups 表
- id: 主键
- name: 分组名称
- total: 实例总数
- created_at: 创建时间
- updated_at: 更新时间

## 许可证

本项目采用 MIT 许可证。
