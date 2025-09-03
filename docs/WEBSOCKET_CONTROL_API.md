# WebSocket 控制 API 文档

## 概述

WebSocket 控制 API 提供了对远程设备的实时控制功能，包括鼠标、键盘、剪贴板和系统操作。

## 连接信息

- **端点**: `ws://{agent_ip}:50052/wscontrol`
- **协议**: WebSocket
- **数据格式**: JSON
- **心跳机制**: 服务器每30秒发送ping，客户端自动响应pong
- **超时设置**: 5分钟无活动自动断开

## 消息格式

### 请求消息结构

```json
{
  "type": "MESSAGE_TYPE",
  "data": {
    // 消息相关数据
  },
  "timestamp": 1640995200000,
  "id": "msg_1640995200000_abc123def"
}
```

### 响应消息结构

```json
{
  "type": "RESPONSE_SUCCESS|RESPONSE_ERROR|RESPONSE_INFO",
  "data": {
    "message": "操作结果描述",
    "details": "详细信息（错误时）"
  },
  "timestamp": 1640995200000
}
```

## 消息类型

### 鼠标控制消息

#### 鼠标移动
```json
{
  "type": "MOUSE_MOVE",
  "data": {
    "x": 100,
    "y": 200
  }
}
```

#### 鼠标点击
```json
{
  "type": "MOUSE_LEFT_CLICK",  // 或 MOUSE_RIGHT_CLICK, MOUSE_MIDDLE_CLICK
  "data": {
    "x": 100,
    "y": 200
  }
}
```

#### 鼠标按下/释放
```json
{
  "type": "MOUSE_LEFT_DOWN",   // 或 MOUSE_LEFT_UP, MOUSE_RIGHT_DOWN, MOUSE_RIGHT_UP, MOUSE_MIDDLE_DOWN, MOUSE_MIDDLE_UP
  "data": {
    "x": 100,
    "y": 200
  }
}
```

#### 鼠标滚轮
```json
{
  "type": "MOUSE_WHEEL_UP",    // 或 MOUSE_WHEEL_DOWN
  "data": {
    "x": 100,
    "y": 200
  }
}
```

#### 重置鼠标状态
```json
{
  "type": "MOUSE_RESET",
  "data": {}
}
```

### 键盘控制消息

#### 按键按下/释放
```json
{
  "type": "KEY_DOWN",          // 或 KEY_UP
  "data": {
    "key": 65,                 // 键码（可选）
    "keyStr": "a"              // 键名（可选）
  }
}
```

#### 按键按下并释放
```json
{
  "type": "KEY_PRESS",
  "data": {
    "key": 65,
    "keyStr": "a"
  }
}
```

#### 组合键
```json
{
  "type": "KEY_COMBO",
  "data": {
    "keys": ["ctrl", "c"]      // 按键组合
  }
}
```

### 剪贴板操作

#### 粘贴文本
```json
{
  "type": "CLIPBOARD_PASTE",
  "data": {
    "text": "要粘贴的文本内容"
  }
}
```

### 系统控制消息

#### 显示桌面
```json
{
  "type": "SYSTEM_DESKTOP",
  "data": {}
}
```

#### 打开任务管理器
```json
{
  "type": "SYSTEM_TASKMANAGER",
  "data": {}
}
```

#### 系统重启
```json
{
  "type": "SYSTEM_REBOOT",
  "data": {}
}
```

## 兼容性

为了保持向后兼容性，系统仍然支持旧格式的消息：

### 旧格式示例

- 鼠标事件: `5.mouse,100,200,1,1640995200000`
- 键盘事件: `3.key,65,1,1640995200000`
- 粘贴操作: `5.paste,文本内容`
- 系统命令: `3.cmd,0./keyboard?cmd=win_d`

## 错误处理

当消息处理失败时，服务器会返回错误响应：

```json
{
  "type": "RESPONSE_ERROR",
  "data": {
    "message": "处理消息失败",
    "details": "具体错误信息"
  },
  "timestamp": 1640995200000
}
```

## 使用示例

### JavaScript 客户端示例

```javascript
const ws = new WebSocket('ws://192.168.1.100:50052/wscontrol');

// 发送鼠标点击
function sendMouseClick(x, y) {
  const message = {
    type: 'MOUSE_LEFT_CLICK',
    data: { x, y },
    timestamp: Date.now(),
    id: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  };
  ws.send(JSON.stringify(message));
}

// 发送键盘按键
function sendKeyPress(keyStr) {
  const message = {
    type: 'KEY_PRESS',
    data: { keyStr },
    timestamp: Date.now(),
    id: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
  };
  ws.send(JSON.stringify(message));
}

// 监听响应
ws.onmessage = (event) => {
  const response = JSON.parse(event.data);
  console.log('收到响应:', response);
};
```

## 注意事项

1. **坐标系统**: 鼠标坐标基于1920x1080分辨率进行缩放
2. **连接管理**: 建议实现心跳机制保持连接活跃
3. **错误处理**: 客户端应处理连接断开和消息发送失败的情况
4. **安全性**: 生产环境中应实现适当的身份验证和授权机制
