# WebSocket 控制消息重构说明

## 重构目标

将原有的数字类型控制消息（1,2,3,4,5）重构为更具体、更易理解的消息类型，提高代码可读性和前后端对接效率。

## 主要改进

### 1. 消息类型细化

#### 原有格式（数字类型）
```
5.mouse,100,200,1,1640995200000    // 鼠标事件
3.key,65,1,1640995200000           // 键盘事件
5.paste,文本内容                   // 粘贴操作
3.cmd,0./keyboard?cmd=win_d        // 系统命令
```

#### 新格式（语义化类型）
```json
{
  "type": "MOUSE_LEFT_CLICK",
  "data": { "x": 100, "y": 200 },
  "timestamp": 1640995200000,
  "id": "msg_1640995200000_abc123"
}
```

### 2. 新增消息类型

#### 鼠标控制（13种类型）
- `MOUSE_MOVE` - 鼠标移动
- `MOUSE_LEFT_CLICK` - 左键点击
- `MOUSE_RIGHT_CLICK` - 右键点击
- `MOUSE_MIDDLE_CLICK` - 中键点击
- `MOUSE_LEFT_DOWN` - 左键按下
- `MOUSE_LEFT_UP` - 左键释放
- `MOUSE_RIGHT_DOWN` - 右键按下
- `MOUSE_RIGHT_UP` - 右键释放
- `MOUSE_MIDDLE_DOWN` - 中键按下
- `MOUSE_MIDDLE_UP` - 中键释放
- `MOUSE_WHEEL_UP` - 滚轮向上
- `MOUSE_WHEEL_DOWN` - 滚轮向下
- `MOUSE_RESET` - 重置鼠标状态

#### 键盘控制（4种类型）
- `KEY_DOWN` - 按键按下
- `KEY_UP` - 按键释放
- `KEY_PRESS` - 按键按下并释放
- `KEY_COMBO` - 组合键

#### 剪贴板操作（2种类型）
- `CLIPBOARD_PASTE` - 粘贴文本
- `CLIPBOARD_COPY` - 复制文本

#### 系统控制（5种类型）
- `SYSTEM_DESKTOP` - 显示桌面
- `SYSTEM_TASKMANAGER` - 打开任务管理器
- `SYSTEM_REBOOT` - 系统重启
- `SYSTEM_SHUTDOWN` - 系统关机
- `SYSTEM_LOCK` - 锁定系统

#### 响应消息（3种类型）
- `RESPONSE_SUCCESS` - 操作成功
- `RESPONSE_ERROR` - 操作错误
- `RESPONSE_INFO` - 信息响应

## 文件修改清单

### 后端文件（Agent）

#### 1. `agent/internal/handlers/websocket_control.go`
- **新增**: 消息类型常量定义
- **新增**: `ControlMessage` 结构体
- **新增**: 鼠标、键盘、系统事件数据结构
- **重构**: `handleControlMessages()` 函数，支持新旧格式
- **新增**: `handleNewControlMessage()` 处理新格式消息
- **保留**: `handleLegacyControlMessage()` 兼容旧格式
- **新增**: 各种新格式事件处理函数
- **新增**: 消息发送和响应处理函数

#### 2. `agent/internal/controllers/routes.go`
- **确认**: `/ws/control` 路由已存在

### 前端文件

#### 1. `frontend/src/views/device-dashboard/components/StreamDialog.vue`
- **新增**: `MSG_TYPES` 常量定义
- **重构**: `sendControlMessage()` 函数，支持新格式
- **新增**: `sendLegacyControlMessage()` 兼容函数
- **重构**: 所有鼠标事件处理函数
- **重构**: 所有键盘事件处理函数
- **重构**: 设备操控方法
- **新增**: `convertKeysymToString()` 键码转换函数

### 文档文件

#### 1. `docs/WEBSOCKET_CONTROL_API.md`
- **新增**: 完整的API文档
- **包含**: 消息格式说明
- **包含**: 所有消息类型示例
- **包含**: JavaScript使用示例

#### 2. `docs/CONTROL_MESSAGE_REFACTOR.md`
- **新增**: 重构说明文档

## 兼容性保证

### 向后兼容
- 保留旧格式消息处理逻辑
- 前端可以同时发送新旧格式消息
- 后端自动识别消息格式并相应处理

### 渐进式迁移
1. **阶段1**: 部署新版本，同时支持新旧格式
2. **阶段2**: 前端逐步迁移到新格式
3. **阶段3**: 移除旧格式支持（可选）

## 优势对比

### 原有数字格式的问题
- 消息类型不直观（5.mouse, 3.key）
- 参数解析复杂
- 前后端对接困难
- 扩展性差
- 调试困难

### 新语义化格式的优势
- 消息类型一目了然
- JSON格式标准化
- 类型安全
- 易于扩展
- 便于调试和日志记录
- 更好的IDE支持

## 测试建议

### 功能测试
1. 测试所有鼠标操作（移动、点击、滚轮）
2. 测试所有键盘操作（按键、组合键）
3. 测试剪贴板操作
4. 测试系统控制功能

### 兼容性测试
1. 验证新格式消息正常工作
2. 验证旧格式消息仍然有效
3. 测试混合使用新旧格式

### 性能测试
1. 测试高频鼠标移动事件
2. 测试连续键盘输入
3. 验证WebSocket连接稳定性

## 后续扩展

### 可能的新消息类型
- 文件操作类型（上传、下载、删除）
- 应用程序控制（启动、停止、最小化、最大化）
- 网络操作（ping、连接测试）
- 硬件状态查询（CPU、内存、磁盘）

### 消息增强
- 添加消息优先级
- 实现消息确认机制
- 支持批量操作
- 添加操作撤销功能

## 总结

通过这次重构，我们将原有的数字类型控制消息升级为语义化的消息类型，大大提高了代码的可读性和可维护性。同时保持了向后兼容性，确保现有功能不受影响。新的消息格式为未来的功能扩展奠定了良好的基础。
