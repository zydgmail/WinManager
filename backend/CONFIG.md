# WinManager Backend 配置说明

## 配置文件结构

后端使用 `config.json` 文件进行配置，支持以下配置项：

```json
{
  "database": {
    "path": "./data.db"
  },
  "server": {
    "port": ":8080"
  },
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

## 配置项说明

### database
- `path`: SQLite数据库文件路径

### server
- `port`: 后端HTTP服务器监听端口

### agent
- `http_port`: Agent的HTTP端口（用于截图、代理等HTTP请求）
- `grpc_port`: Agent的gRPC端口（用于gRPC通信）
- `heartbeat_timeout_seconds`: 心跳超时时间（秒），超过此时间判断设备离线，默认90秒
- `offline_check_interval`: 离线检测间隔（秒），后端定时检查设备离线状态的间隔，默认60秒

### log
- `level`: 日志级别 (debug, info, warn, error)
- `file`: 日志文件路径
- `max_size`: 日志文件最大大小(MB)
- `max_backups`: 保留的日志文件数量
- `compress`: 是否压缩旧日志文件

## Agent端口配置的重要性

Agent端口配置统一管理了后端与Agent通信时使用的端口号，包括：

1. **截图请求**: `http://{agent_ip}:{http_port}/api/screenshot`
2. **WebSocket视频流**: `ws://{agent_ip}:{http_port}/wsstream`
3. **HTTP代理**: `http://{agent_ip}:{http_port}/api/*`
4. **视频流控制**: `http://{agent_ip}:{http_port}/api/startstream`、`stopstream`
5. **同步命令**: `http://{agent_ip}:{http_port}/api/toggle`

## 设备离线检测机制

后端实现了自动的设备离线检测机制：

1. **心跳机制**: Agent每30秒向后端发送一次心跳，更新`last_heartbeat_at`字段
2. **离线检测**: 后端定时检查设备的最后心跳时间，超过配置的超时时间则判断为离线
3. **状态更新**: 自动将超时设备的状态从在线(1)更新为离线(0)

### 配置说明
- `heartbeat_timeout_seconds`: 心跳超时时间，建议设置为心跳间隔的3倍（默认90秒）
- `offline_check_interval`: 检测间隔，不宜过短以避免频繁数据库操作（默认60秒）

### 工作流程
1. Agent启动后向后端注册，状态设为在线(1)
2. Agent每30秒发送心跳，更新`last_heartbeat_at`和状态为在线(1)
3. 后端每60秒检查一次，将超过90秒未心跳的设备状态设为离线(0)
4. 当Agent恢复心跳时，状态会自动恢复为在线(1)

## 配置修改

如果需要修改Agent端口，只需要：

1. 修改 `backend/config.json` 中的 `agent.http_port` 和 `agent.grpc_port`
2. 确保Agent也使用相同的端口配置
3. 重启后端服务

## 默认端口

- 后端HTTP服务: 8080
- Agent HTTP服务: 50052  
- Agent gRPC服务: 50051

这些端口可以根据实际部署环境进行调整。
