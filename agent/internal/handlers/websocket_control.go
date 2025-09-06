package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"winmanager-agent/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/go-vgo/robotgo"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// WebSocket控制消息类型常量
const (
	// 鼠标控制消息类型
	MSG_MOUSE_MOVE         = "MOUSE_MOVE"         // 鼠标移动
	MSG_MOUSE_LEFT_DOWN    = "MOUSE_LEFT_DOWN"    // 鼠标左键按下
	MSG_MOUSE_LEFT_UP      = "MOUSE_LEFT_UP"      // 鼠标左键释放
	MSG_MOUSE_LEFT_CLICK   = "MOUSE_LEFT_CLICK"   // 鼠标左键点击
	MSG_MOUSE_RIGHT_DOWN   = "MOUSE_RIGHT_DOWN"   // 鼠标右键按下
	MSG_MOUSE_RIGHT_UP     = "MOUSE_RIGHT_UP"     // 鼠标右键释放
	MSG_MOUSE_RIGHT_CLICK  = "MOUSE_RIGHT_CLICK"  // 鼠标右键点击
	MSG_MOUSE_MIDDLE_DOWN  = "MOUSE_MIDDLE_DOWN"  // 鼠标中键按下
	MSG_MOUSE_MIDDLE_UP    = "MOUSE_MIDDLE_UP"    // 鼠标中键释放
	MSG_MOUSE_MIDDLE_CLICK = "MOUSE_MIDDLE_CLICK" // 鼠标中键点击
	MSG_MOUSE_WHEEL_UP     = "MOUSE_WHEEL_UP"     // 鼠标滚轮向上
	MSG_MOUSE_WHEEL_DOWN   = "MOUSE_WHEEL_DOWN"   // 鼠标滚轮向下
	MSG_MOUSE_RESET        = "MOUSE_RESET"        // 重置鼠标状态

	// 键盘控制消息类型
	MSG_KEY_DOWN  = "KEY_DOWN"  // 按键按下
	MSG_KEY_UP    = "KEY_UP"    // 按键释放
	MSG_KEY_PRESS = "KEY_PRESS" // 按键按下并释放
	MSG_KEY_COMBO = "KEY_COMBO" // 组合键

	// 剪贴板消息类型
	MSG_CLIPBOARD_PASTE  = "CLIPBOARD_PASTE"  // 将文本注入输入目标（打字输入）
	MSG_CLIPBOARD_COPY   = "CLIPBOARD_COPY"   // 复制文本（保留，兼容）
	MSG_CLIPBOARD_SET    = "CLIPBOARD_SET"    // 设置Agent剪贴板（不打字）
	MSG_CLIPBOARD_GET    = "CLIPBOARD_GET"    // 请求Agent当前剪贴板
	MSG_CLIPBOARD_UPDATE = "CLIPBOARD_UPDATE" // Agent->客户端：剪贴板变更通知

	// 系统控制消息类型
	MSG_SYSTEM_DESKTOP     = "SYSTEM_DESKTOP"     // 显示桌面
	MSG_SYSTEM_TASKMANAGER = "SYSTEM_TASKMANAGER" // 打开任务管理器
	MSG_SYSTEM_REBOOT      = "SYSTEM_REBOOT"      // 系统重启
	MSG_SYSTEM_SHUTDOWN    = "SYSTEM_SHUTDOWN"    // 系统关机
	MSG_SYSTEM_LOCK        = "SYSTEM_LOCK"        // 锁定系统

	// 应用程序控制消息类型
	MSG_APP_START    = "APP_START"    // 启动应用程序
	MSG_APP_STOP     = "APP_STOP"     // 停止应用程序
	MSG_APP_MINIMIZE = "APP_MINIMIZE" // 最小化应用程序
	MSG_APP_MAXIMIZE = "APP_MAXIMIZE" // 最大化应用程序

	// 响应消息类型
	MSG_RESPONSE_SUCCESS = "RESPONSE_SUCCESS" // 操作成功响应
	MSG_RESPONSE_ERROR   = "RESPONSE_ERROR"   // 操作错误响应
	MSG_RESPONSE_INFO    = "RESPONSE_INFO"    // 信息响应

	// 坐标映射查询消息类型
	MSG_COORDINATE_MAPPING_STATUS = "COORDINATE_MAPPING_STATUS" // 查询坐标映射状态
)

// 控制消息结构
type ControlMessage struct {
	Type      string                 `json:"type"`      // 消息类型
	Data      map[string]interface{} `json:"data"`      // 消息数据
	Timestamp int64                  `json:"timestamp"` // 时间戳
	ID        string                 `json:"id"`        // 消息ID（可选）
}

// 鼠标事件数据结构
type MouseEventData struct {
	X      int `json:"x"`      // X坐标
	Y      int `json:"y"`      // Y坐标
	Button int `json:"button"` // 按键状态
}

// 键盘事件数据结构
type KeyEventData struct {
	Key    int    `json:"key"`    // 键码
	State  int    `json:"state"`  // 按键状态 (0=释放, 1=按下)
	KeyStr string `json:"keyStr"` // 键名（可选）
}

// 系统命令数据结构
type SystemCommandData struct {
	Command string            `json:"command"` // 命令类型
	Args    map[string]string `json:"args"`    // 命令参数
	Timeout int               `json:"timeout"` // 超时时间（秒）
}

// 坐标映射配置
type CoordinateMapping struct {
	// 前端发送的坐标基于的分辨率（编码分辨率）
	EncodedWidth  int
	EncodedHeight int

	// 实际屏幕分辨率
	ScreenWidth  int
	ScreenHeight int

	// 缩放比例
	ScaleX float64
	ScaleY float64

	// 是否已初始化
	Initialized bool
}

// WebSocket控制接口升级器
var controlUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域，生产环境应该限制
	},
}

// 全局坐标映射实例
var globalCoordinateMapping *CoordinateMapping
var coordinateMappingMutex sync.RWMutex

// WebSocketControlHandler 处理WebSocket控制连接
func WebSocketControlHandler(c *gin.Context) {
	log.Info("WebSocket控制连接请求")

	// 升级HTTP连接为WebSocket
	ws, err := controlUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.WithError(err).Error("升级WebSocket控制连接失败")
		return
	}

	defer func() {
		log.Info("关闭WebSocket控制连接")
		ws.Close()
	}()

	// 设置WebSocket参数
	ws.SetReadLimit(1024)
	ws.SetReadDeadline(time.Now().Add(300 * time.Second)) // 增加到5分钟
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(300 * time.Second)) // 增加到5分钟
		return nil
	})

	// 启动心跳机制
	go func() {
		ticker := time.NewTicker(30 * time.Second) // 每30秒发送一次心跳
		defer ticker.Stop()

		for range ticker.C {
			if err := ws.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				log.WithError(err).Debug("发送心跳失败，连接可能已断开")
				return
			}
			log.Debug("💓 发送WebSocket心跳")
		}
	}()

	// 发送连接成功消息
	successMsg := ControlMessage{
		Type:      MSG_RESPONSE_SUCCESS,
		Data:      map[string]interface{}{"message": "WebSocket控制连接建立成功"},
		Timestamp: time.Now().Unix(),
	}
	if err := sendControlMessage(ws, successMsg); err != nil {
		log.WithError(err).Error("发送连接成功消息失败")
		return
	}

	log.Info("WebSocket控制连接建立成功")

	// 初始化坐标映射（用于将编码坐标转换为屏幕坐标）
	if err := initCoordinateMapping(); err != nil {
		log.WithError(err).Warn("初始化坐标映射失败，将使用原始坐标")
	}

	// 剪贴板同步改为事件驱动：不启动轮询监听

	// 处理控制消息
	handleControlMessages(ws)
}

// handleControlMessages 处理控制消息循环
func handleControlMessages(ws *websocket.Conn) {
	for {
		// 读取消息
		_, messageData, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithError(err).Error("WebSocket控制连接读取错误")
			}
			break
		}

		log.WithFields(log.Fields{
			"raw_message":    string(messageData),
			"message_length": len(messageData),
		}).Debug("收到原始控制消息")

		// 尝试解析新格式的JSON消息
		var msg ControlMessage
		if err := json.Unmarshal(messageData, &msg); err == nil {
			// 剪贴板消息特殊日志
			if msg.Type == MSG_CLIPBOARD_SET || msg.Type == MSG_CLIPBOARD_GET || msg.Type == MSG_CLIPBOARD_PASTE {
				log.WithFields(log.Fields{
					"type":        msg.Type,
					"data":        msg.Data,
					"timestamp":   msg.Timestamp,
					"raw_message": string(messageData),
				}).Info("📋 [Agent] 收到剪贴板消息")
			}

			if err := handleNewControlMessage(ws, msg); err != nil {
				log.WithError(err).Errorf("处理新格式控制消息失败: %s", msg.Type)
				sendErrorResponse(ws, "处理消息失败", err.Error())
			} else {
				log.WithField("type", msg.Type).Debug("新格式控制消息处理成功")
			}
		} else {
			// 兼容旧格式消息
			message := string(messageData)
			log.WithFields(log.Fields{
				"message":     message,
				"parse_error": err.Error(),
			}).Debug("收到旧格式控制消息")

			if err := handleLegacyControlMessage(message); err != nil {
				log.WithError(err).Error("处理旧格式控制消息失败")
				sendErrorResponse(ws, "处理消息失败", err.Error())
			} else {
				log.WithField("message", message).Debug("旧格式控制消息处理成功")
			}
		}
	}
}

// sendControlMessage 发送控制消息
func sendControlMessage(ws *websocket.Conn, msg ControlMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	return ws.WriteMessage(websocket.TextMessage, data)
}

// sendErrorResponse 发送错误响应
func sendErrorResponse(ws *websocket.Conn, message, details string) {
	errorMsg := ControlMessage{
		Type: MSG_RESPONSE_ERROR,
		Data: map[string]interface{}{
			"message": message,
			"details": details,
		},
		Timestamp: time.Now().Unix(),
	}
	sendControlMessage(ws, errorMsg)
}

// sendSuccessResponse 发送成功响应
func sendSuccessResponse(ws *websocket.Conn, message string, data map[string]interface{}) {
	if data == nil {
		data = make(map[string]interface{})
	}
	data["message"] = message

	successMsg := ControlMessage{
		Type:      MSG_RESPONSE_SUCCESS,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
	sendControlMessage(ws, successMsg)
}

// handleNewControlMessage 处理新格式控制消息
func handleNewControlMessage(ws *websocket.Conn, msg ControlMessage) error {
	log.WithField("type", msg.Type).Debug("处理新格式控制消息")

	// 剪贴板相关消息特殊日志
	if msg.Type == MSG_CLIPBOARD_SET || msg.Type == MSG_CLIPBOARD_GET || msg.Type == MSG_CLIPBOARD_PASTE {
		log.WithFields(log.Fields{
			"type":      msg.Type,
			"data":      msg.Data,
			"timestamp": msg.Timestamp,
		}).Info("📋 [Agent] 收到剪贴板消息")
	}

	switch msg.Type {
	// 鼠标事件
	case MSG_MOUSE_MOVE:
		return handleNewMouseMove(msg.Data)
	case MSG_MOUSE_LEFT_CLICK:
		return handleNewMouseClick(msg.Data, "left")
	case MSG_MOUSE_RIGHT_CLICK:
		return handleNewMouseClick(msg.Data, "right")
	case MSG_MOUSE_MIDDLE_CLICK:
		return handleNewMouseClick(msg.Data, "middle")
	case MSG_MOUSE_LEFT_DOWN:
		return handleNewMouseDown(msg.Data, "left")
	case MSG_MOUSE_LEFT_UP:
		return handleNewMouseUp(msg.Data, "left")
	case MSG_MOUSE_RIGHT_DOWN:
		return handleNewMouseDown(msg.Data, "right")
	case MSG_MOUSE_RIGHT_UP:
		return handleNewMouseUp(msg.Data, "right")
	case MSG_MOUSE_MIDDLE_DOWN:
		return handleNewMouseDown(msg.Data, "middle")
	case MSG_MOUSE_MIDDLE_UP:
		return handleNewMouseUp(msg.Data, "middle")
	case MSG_MOUSE_WHEEL_UP:
		return handleNewMouseWheel(msg.Data, 1)
	case MSG_MOUSE_WHEEL_DOWN:
		return handleNewMouseWheel(msg.Data, -1)
	case MSG_MOUSE_RESET:
		return handleNewMouseReset()

	// 键盘事件
	case MSG_KEY_DOWN:
		return handleNewKeyDown(msg.Data)
	case MSG_KEY_UP:
		return handleNewKeyUp(msg.Data)
	case MSG_KEY_PRESS:
		return handleNewKeyPress(msg.Data)
	case MSG_KEY_COMBO:
		return handleNewKeyCombo(msg.Data)

	// 剪贴板事件
	case MSG_CLIPBOARD_PASTE:
		return handleNewClipboardPaste(msg.Data)
	case MSG_CLIPBOARD_SET:
		return handleClipboardSet(msg.Data)
	case MSG_CLIPBOARD_GET:
		return handleClipboardGet(ws)

	// 系统控制事件
	case MSG_SYSTEM_DESKTOP:
		return handleNewSystemDesktop()
	case MSG_SYSTEM_TASKMANAGER:
		return handleNewSystemTaskManager()
	case MSG_SYSTEM_REBOOT:
		return handleNewSystemReboot()

	// 坐标映射状态查询
	case MSG_COORDINATE_MAPPING_STATUS:
		return handleCoordinateMappingStatus(ws, msg)

	default:
		return fmt.Errorf("未知的消息类型: %s", msg.Type)
	}
}

// handleLegacyControlMessage 处理旧格式控制消息（兼容性）
func handleLegacyControlMessage(message string) error {
	// 解析消息格式：type.param1,param2,param3,...
	parts := strings.SplitN(message, ".", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid message format: %s", message)
	}

	msgType := parts[0]
	params := strings.Split(parts[1], ",")

	switch msgType {
	case "5": // 鼠标事件或特殊操作
		if len(params) > 0 && params[0] == "paste" {
			return handlePasteMessage(params)
		}
		return handleMouseMessage(params)
	case "3": // 键盘事件或命令
		return handleKeyboardOrCommand(params)
	default:
		return fmt.Errorf("unknown message type: %s", msgType)
	}
}

// handleMouseMessage 处理鼠标消息
// 格式：5.mouse,x,y,button,timestamp
func handleMouseMessage(params []string) error {
	if len(params) < 4 {
		return fmt.Errorf("invalid mouse message params: %v", params)
	}

	if params[0] != "mouse" {
		if params[0] == "reset" {
			// 重置鼠标状态
			log.Debug("重置鼠标状态")
			return nil
		}
		return fmt.Errorf("unknown mouse action: %s", params[0])
	}

	// 解析坐标
	x, err := strconv.Atoi(params[1])
	if err != nil {
		return fmt.Errorf("invalid x coordinate: %s", params[1])
	}

	y, err := strconv.Atoi(params[2])
	if err != nil {
		return fmt.Errorf("invalid y coordinate: %s", params[2])
	}

	// 解析按键
	button, err := strconv.Atoi(params[3])
	if err != nil {
		return fmt.Errorf("invalid button: %s", params[3])
	}

	log.WithFields(log.Fields{
		"x":      x,
		"y":      y,
		"button": button,
	}).Debug("处理鼠标事件")

	// 移动鼠标到指定位置
	robotgo.Move(x, y)

	// 处理鼠标按键
	switch button {
	case 0:
		// 鼠标移动或释放，不需要点击
		break
	case 1:
		// 左键点击
		robotgo.Click("left")
	case 2:
		// 中键点击
		robotgo.Click("middle")
	case 4:
		// 右键点击
		robotgo.Click("right")
	case 8:
		// 向下滚动
		robotgo.Scroll(0, -3)
	case 16:
		// 向上滚动
		robotgo.Scroll(0, 3)
	default:
		log.WithField("button", button).Warn("未知的鼠标按键")
	}

	return nil
}

// handleKeyboardOrCommand 处理键盘事件或命令
func handleKeyboardOrCommand(params []string) error {
	if len(params) < 1 {
		return fmt.Errorf("invalid keyboard/command params: %v", params)
	}

	action := params[0]

	switch action {
	case "key":
		// 键盘事件：3.key,keysym,pressed,timestamp
		return handleKeyboardEvent(params)
	case "cmd":
		// 命令事件：3.cmd,0.command
		return handleCommandEvent(params)
	default:
		return fmt.Errorf("unknown keyboard/command action: %s", action)
	}
}

// handleKeyboardEvent 处理键盘事件
func handleKeyboardEvent(params []string) error {
	if len(params) < 3 {
		return fmt.Errorf("invalid keyboard event params: %v", params)
	}

	// 解析按键码
	keysym, err := strconv.Atoi(params[1])
	if err != nil {
		return fmt.Errorf("invalid keysym: %s", params[1])
	}

	// 解析按键状态（1=按下，0=释放）
	pressed, err := strconv.Atoi(params[2])
	if err != nil {
		return fmt.Errorf("invalid pressed state: %s", params[2])
	}

	log.WithFields(log.Fields{
		"keysym":  keysym,
		"pressed": pressed,
	}).Debug("处理键盘事件")

	// 转换keysym到robotgo按键字符串
	keyStr := convertKeysymToRobotgo(keysym)
	if keyStr == "" {
		log.WithField("keysym", keysym).Warn("不支持的按键码")
		return nil
	}

	// 执行按键操作
	if pressed == 1 {
		robotgo.KeyDown(keyStr)
	} else {
		robotgo.KeyUp(keyStr)
	}

	return nil
}

// handleCommandEvent 处理命令事件
func handleCommandEvent(params []string) error {
	if len(params) < 2 {
		return fmt.Errorf("invalid command event params: %v", params)
	}

	// 解析命令：3.cmd,0.command
	command := params[1]

	log.WithField("command", command).Info("处理命令事件")

	// 处理不同类型的命令
	if strings.Contains(command, "/keyboard?cmd=") {
		// 键盘快捷键命令
		return handleKeyboardCommand(command)
	} else if strings.Contains(command, "/process?") {
		// 进程管理命令
		return handleProcessCommand(command)
	} else if strings.Contains(command, "/reboot") {
		// 重启命令
		return handleRebootCommand()
	}

	return fmt.Errorf("unknown command: %s", command)
}

// handleKeyboardCommand 处理键盘快捷键命令
func handleKeyboardCommand(command string) error {
	if strings.Contains(command, "cmd=win_d") {
		// 显示桌面 (Win+D)
		log.Info("执行显示桌面命令")
		robotgo.KeyDown("cmd")
		robotgo.KeyDown("d")
		time.Sleep(50 * time.Millisecond)
		robotgo.KeyUp("d")
		robotgo.KeyUp("cmd")
	}
	return nil
}

// handleProcessCommand 处理进程管理命令
func handleProcessCommand(command string) error {
	if strings.Contains(command, "name=taskmgr") && strings.Contains(command, "action=start") {
		// 启动任务管理器
		log.Info("启动任务管理器")
		robotgo.KeyDown("ctrl")
		robotgo.KeyDown("shift")
		robotgo.KeyDown("esc")
		time.Sleep(50 * time.Millisecond)
		robotgo.KeyUp("esc")
		robotgo.KeyUp("shift")
		robotgo.KeyUp("ctrl")
	}
	return nil
}

// handleRebootCommand 处理重启命令
func handleRebootCommand() error {
	log.Info("收到重启命令")
	// 这里可以实现实际的重启逻辑
	// 为了安全起见，暂时只记录日志
	return nil
}

// convertKeysymToRobotgo 将Guacamole keysym转换为robotgo按键字符串
func convertKeysymToRobotgo(keysym int) string {
	// 特殊按键映射（优先处理）
	keyMap := map[int]string{
		65288: "backspace",
		65289: "tab",
		65293: "enter",
		65505: "shift",
		65507: "ctrl",
		65513: "alt",
		65515: "cmd", // 左Win键
		65516: "cmd", // 右Win键
		65307: "esc",
		32:    "space",
		96:    "`", // 反引号键 - 特殊处理
		65361: "left",
		65362: "up",
		65363: "right",
		65364: "down",
		65535: "delete",
		65360: "home",
		65367: "end",
		65365: "pageup",
		65366: "pagedown",
	}

	if key, exists := keyMap[keysym]; exists {
		return key
	}

	// 基本字符（ASCII）- 排除已在特殊映射中处理的字符
	if keysym >= 32 && keysym <= 126 {
		return string(rune(keysym))
	}

	// 功能键 F1-F12
	if keysym >= 65470 && keysym <= 65481 {
		return fmt.Sprintf("f%d", keysym-65469)
	}

	return ""
}

// 新格式鼠标事件处理函数
func handleNewMouseMove(data map[string]interface{}) error {
	x, y, err := extractMouseCoordinates(data)
	if err != nil {
		log.WithError(err).Error("❌ 提取鼠标坐标失败")
		return err
	}

	// log.WithFields(log.Fields{
	// 	"x":          x,
	// 	"y":          y,
	// 	"action":     "move",
	// 	"event_type": "MOUSE_MOVE",
	// 	"raw_data":   fmt.Sprintf("%+v", data),
	// }).Info("🖱️ 执行鼠标移动")

	robotgo.Move(x, y)

	log.WithFields(log.Fields{
		"x":      x,
		"y":      y,
		"result": "success",
	}).Info("✅ 鼠标移动完成")
	return nil
}

func handleNewMouseClick(data map[string]interface{}, button string) error {
	x, y, err := extractMouseCoordinates(data)
	if err != nil {
		log.WithError(err).Error("❌ 提取鼠标坐标失败")
		return err
	}

	log.WithFields(log.Fields{
		"x":           x,
		"y":           y,
		"button":      button,
		"action":      "click",
		"event_type":  fmt.Sprintf("MOUSE_%s_CLICK", strings.ToUpper(button)),
		"raw_data":    fmt.Sprintf("%+v", data),
		"coordinates": fmt.Sprintf("(%d, %d)", x, y),
	}).Info("🖱️ 执行鼠标点击")

	robotgo.Move(x, y)
	robotgo.Click(button)

	log.WithFields(log.Fields{
		"x":           x,
		"y":           y,
		"button":      button,
		"result":      "success",
		"coordinates": fmt.Sprintf("(%d, %d)", x, y),
	}).Info("✅ 鼠标点击完成")
	return nil
}

func handleNewMouseDown(data map[string]interface{}, button string) error {
	x, y, err := extractMouseCoordinates(data)
	if err != nil {
		log.WithError(err).Error("❌ 提取鼠标坐标失败")
		return err
	}

	log.WithFields(log.Fields{
		"x":           x,
		"y":           y,
		"button":      button,
		"action":      "down",
		"event_type":  fmt.Sprintf("MOUSE_%s_DOWN", strings.ToUpper(button)),
		"raw_data":    fmt.Sprintf("%+v", data),
		"coordinates": fmt.Sprintf("(%d, %d)", x, y),
	}).Info("🖱️ 执行鼠标按下")

	robotgo.Move(x, y)
	robotgo.MouseDown(button)

	log.WithFields(log.Fields{
		"x":           x,
		"y":           y,
		"button":      button,
		"result":      "success",
		"coordinates": fmt.Sprintf("(%d, %d)", x, y),
	}).Info("✅ 鼠标按下完成")
	return nil
}

func handleNewMouseUp(data map[string]interface{}, button string) error {
	x, y, err := extractMouseCoordinates(data)
	if err != nil {
		log.WithError(err).Error("❌ 提取鼠标坐标失败")
		return err
	}

	log.WithFields(log.Fields{
		"x":           x,
		"y":           y,
		"button":      button,
		"action":      "up",
		"event_type":  fmt.Sprintf("MOUSE_%s_UP", strings.ToUpper(button)),
		"raw_data":    fmt.Sprintf("%+v", data),
		"coordinates": fmt.Sprintf("(%d, %d)", x, y),
	}).Info("🖱️ 执行鼠标释放")

	robotgo.Move(x, y)
	robotgo.MouseUp(button)

	log.WithFields(log.Fields{
		"x":           x,
		"y":           y,
		"button":      button,
		"result":      "success",
		"coordinates": fmt.Sprintf("(%d, %d)", x, y),
	}).Info("✅ 鼠标释放完成")
	return nil
}

func handleNewMouseWheel(data map[string]interface{}, direction int) error {
	x, y, err := extractMouseCoordinates(data)
	if err != nil {
		log.WithError(err).Error("❌ 提取鼠标坐标失败")
		return err
	}

	wheelDirection := "UP"
	if direction < 0 {
		wheelDirection = "DOWN"
	}

	log.WithFields(log.Fields{
		"x":               x,
		"y":               y,
		"direction":       direction,
		"wheel_direction": wheelDirection,
		"scroll_amount":   direction * 3,
		"event_type":      fmt.Sprintf("MOUSE_WHEEL_%s", wheelDirection),
		"raw_data":        fmt.Sprintf("%+v", data),
		"coordinates":     fmt.Sprintf("(%d, %d)", x, y),
	}).Info("🖱️ 执行鼠标滚轮")

	robotgo.Move(x, y)
	robotgo.Scroll(0, direction*3)

	log.WithFields(log.Fields{
		"x":         x,
		"y":         y,
		"direction": wheelDirection,
		"result":    "success",
	}).Info("✅ 鼠标滚轮完成")
	return nil
}

func handleNewMouseReset() error {
	log.WithFields(log.Fields{
		"action":           "reset",
		"event_type":       "MOUSE_RESET",
		"buttons_released": []string{"left", "right", "middle"},
	}).Info("🖱️ 重置鼠标状态")

	// 释放所有可能按下的鼠标按键
	robotgo.MouseUp("left")
	// robotgo.MouseUp("right") // cause refresh panel to appear
	robotgo.MouseUp("middle")

	log.Info("✅ 鼠标状态重置完成")
	return nil
}

// 初始化坐标映射
func initCoordinateMapping() error {
	coordinateMappingMutex.Lock()
	defer coordinateMappingMutex.Unlock()

	// 获取实际屏幕分辨率
	width, height := robotgo.GetScreenSize()

	// 获取编码器期望的分辨率（从全局编码器获取）
	var encodedWidth, encodedHeight int = 1920, 1080 // 默认值

	// 尝试从全局编码器获取实际期望尺寸
	if globalEncoder != nil {
		if expectedSize, err := globalEncoder.VideoSize(); err == nil {
			encodedWidth = expectedSize.X
			encodedHeight = expectedSize.Y
		}
	}

	// 计算缩放比例
	scaleX := float64(width) / float64(encodedWidth)
	scaleY := float64(height) / float64(encodedHeight)

	globalCoordinateMapping = &CoordinateMapping{
		EncodedWidth:  encodedWidth,
		EncodedHeight: encodedHeight,
		ScreenWidth:   width,
		ScreenHeight:  height,
		ScaleX:        scaleX,
		ScaleY:        scaleY,
		Initialized:   true,
	}

	log.WithFields(log.Fields{
		"encoded_resolution": fmt.Sprintf("%dx%d", encodedWidth, encodedHeight),
		"screen_resolution":  fmt.Sprintf("%dx%d", width, height),
		"scale_x":            scaleX,
		"scale_y":            scaleY,
	}).Info("坐标映射初始化完成")

	return nil
}

// 转换坐标：从编码分辨率坐标转换为实际屏幕坐标
func convertCoordinates(encodedX, encodedY int) (int, int) {
	coordinateMappingMutex.RLock()
	defer coordinateMappingMutex.RUnlock()

	if globalCoordinateMapping == nil || !globalCoordinateMapping.Initialized {
		log.Warn("坐标映射未初始化，使用原始坐标")
		return encodedX, encodedY
	}

	// 应用缩放比例
	actualX := int(float64(encodedX) * globalCoordinateMapping.ScaleX)
	actualY := int(float64(encodedY) * globalCoordinateMapping.ScaleY)

	// 确保坐标在屏幕范围内
	actualX = max(0, min(actualX, globalCoordinateMapping.ScreenWidth-1))
	actualY = max(0, min(actualY, globalCoordinateMapping.ScreenHeight-1))

	log.WithFields(log.Fields{
		"encoded": fmt.Sprintf("(%d, %d)", encodedX, encodedY),
		"actual":  fmt.Sprintf("(%d, %d)", actualX, actualY),
		"scale":   fmt.Sprintf("(%.3f, %.3f)", globalCoordinateMapping.ScaleX, globalCoordinateMapping.ScaleY),
	}).Debug("坐标转换")

	return actualX, actualY
}

// 辅助函数：max和min
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractMouseCoordinates 从数据中提取鼠标坐标并自动转换
func extractMouseCoordinates(data map[string]interface{}) (int, int, error) {
	xVal, ok := data["x"]
	if !ok {
		return 0, 0, fmt.Errorf("缺少x坐标")
	}

	yVal, ok := data["y"]
	if !ok {
		return 0, 0, fmt.Errorf("缺少y坐标")
	}

	x, ok := xVal.(float64)
	if !ok {
		return 0, 0, fmt.Errorf("x坐标格式错误")
	}

	y, ok := yVal.(float64)
	if !ok {
		return 0, 0, fmt.Errorf("y坐标格式错误")
	}

	// 转换为整数坐标
	encodedX := int(x)
	encodedY := int(y)

	// 应用坐标转换
	actualX, actualY := convertCoordinates(encodedX, encodedY)

	return actualX, actualY, nil
}

// handlePasteMessage 处理粘贴消息
// 格式：5.paste,text_content
func handlePasteMessage(params []string) error {
	if len(params) < 2 {
		return fmt.Errorf("invalid paste message params: %v", params)
	}

	text := params[1]
	log.WithField("text_length", len(text)).Debug("处理粘贴事件")

	// 使用robotgo输入文本
	robotgo.TypeStr(text)

	return nil
}

// 新格式键盘事件处理函数
func handleNewKeyDown(data map[string]interface{}) error {
	key, keyStr, err := extractKeyData(data)
	if err != nil {
		log.WithError(err).Error("❌ 提取按键数据失败")
		return err
	}

	log.WithFields(log.Fields{
		"key":        key,
		"keyStr":     keyStr,
		"action":     "down",
		"event_type": "KEY_DOWN",
		"raw_data":   fmt.Sprintf("%+v", data),
		"key_code":   key,
		"key_string": keyStr,
	}).Info("⌨️ 执行按键按下")

	if keyStr != "" {
		robotgo.KeyDown(keyStr)
		log.WithFields(log.Fields{
			"key":    key,
			"keyStr": keyStr,
			"result": "success",
		}).Info("✅ 按键按下完成")
	} else {
		log.WithFields(log.Fields{
			"key":    key,
			"reason": "invalid_key_string",
		}).Warn("⚠️ 无效的按键字符串，跳过按键按下操作")
	}
	return nil
}

func handleNewKeyUp(data map[string]interface{}) error {
	key, keyStr, err := extractKeyData(data)
	if err != nil {
		log.WithError(err).Error("提取按键数据失败")
		return err
	}

	log.WithFields(log.Fields{"key": key, "keyStr": keyStr, "action": "up"}).Debug("执行按键释放")
	if keyStr != "" {
		robotgo.KeyUp(keyStr)
		log.WithFields(log.Fields{"key": key, "keyStr": keyStr}).Debug("按键释放完成")
	} else {
		log.WithField("key", key).Warn("无效的按键字符串，跳过按键释放操作")
	}
	return nil
}

func handleNewKeyPress(data map[string]interface{}) error {
	key, keyStr, err := extractKeyData(data)
	if err != nil {
		log.WithError(err).Error("提取按键数据失败")
		return err
	}

	log.WithFields(log.Fields{"key": key, "keyStr": keyStr, "action": "press"}).Debug("执行按键按下并释放")
	if keyStr != "" {
		robotgo.KeyTap(keyStr)
		log.WithFields(log.Fields{"key": key, "keyStr": keyStr}).Debug("按键按下并释放完成")
	} else {
		log.WithField("key", key).Warn("无效的按键字符串，跳过按键操作")
	}
	return nil
}

func handleNewKeyCombo(data map[string]interface{}) error {
	keysVal, ok := data["keys"]
	if !ok {
		return fmt.Errorf("缺少keys参数")
	}

	keys, ok := keysVal.([]interface{})
	if !ok {
		return fmt.Errorf("keys参数格式错误")
	}

	keyStrs := make([]string, 0, len(keys))
	for _, keyVal := range keys {
		keyStr, ok := keyVal.(string)
		if !ok {
			continue
		}
		keyStrs = append(keyStrs, keyStr)
	}

	log.WithField("keys", keyStrs).Debug("组合键")
	if len(keyStrs) > 0 {
		// robotgo.KeyTap 需要最后一个键作为主键，前面的作为修饰键
		if len(keyStrs) == 1 {
			robotgo.KeyTap(keyStrs[0])
		} else {
			// 将 []string 转换为 []interface{}
			modifiers := make([]interface{}, len(keyStrs)-1)
			for i, key := range keyStrs[:len(keyStrs)-1] {
				modifiers[i] = key
			}
			robotgo.KeyTap(keyStrs[len(keyStrs)-1], modifiers...)
		}
	}
	return nil
}

// extractKeyData 从数据中提取按键信息
func extractKeyData(data map[string]interface{}) (int, string, error) {
	var key int
	var keyStr string

	// 尝试获取键码
	if keyVal, ok := data["key"]; ok {
		if keyFloat, ok := keyVal.(float64); ok {
			key = int(keyFloat)
			keyStr = convertKeysymToRobotgo(key)
		}
	}

	// 尝试获取键名
	if keyStrVal, ok := data["keyStr"]; ok {
		if str, ok := keyStrVal.(string); ok {
			keyStr = str
		}
	}

	if keyStr == "" && key == 0 {
		return 0, "", fmt.Errorf("缺少有效的按键信息")
	}

	return key, keyStr, nil
}

// 新格式剪贴板事件处理函数
func handleNewClipboardPaste(data map[string]interface{}) error {
	textVal, ok := data["text"]
	if !ok {
		log.Error("❌ 缺少text参数")
		return fmt.Errorf("缺少text参数")
	}

	text, ok := textVal.(string)
	if !ok {
		log.Error("❌ text参数格式错误")
		return fmt.Errorf("text参数格式错误")
	}

	// 截取文本预览（避免日志过长）
	preview := text
	if len(text) > 100 {
		preview = text[:100] + "..."
	}

	log.WithFields(log.Fields{
		"text_length":  len(text),
		"text_preview": preview,
		"event_type":   "CLIPBOARD_PASTE",
		"raw_data":     fmt.Sprintf("%+v", data),
		"char_count":   len([]rune(text)), // Unicode字符数
	}).Info("📋 执行粘贴文本")

	robotgo.TypeStr(text)

	log.WithFields(log.Fields{
		"text_length": len(text),
		"result":      "success",
	}).Info("✅ 粘贴文本完成")
	return nil
}

// handleClipboardSet 设置Agent剪贴板（不打字）
func handleClipboardSet(data map[string]interface{}) error {
	textVal, ok := data["text"]
	if !ok {
		log.Error("📋 [Agent] CLIPBOARD_SET失败：缺少text参数")
		return fmt.Errorf("缺少text参数")
	}
	text, ok := textVal.(string)
	if !ok {
		log.Error("📋 [Agent] CLIPBOARD_SET失败：text参数格式错误")
		return fmt.Errorf("text参数格式错误")
	}

	preview := text
	if len(text) > 100 {
		preview = text[:100] + "..."
	}
	log.WithFields(log.Fields{
		"text_length":  len(text),
		"text_preview": preview,
		"event_type":   "CLIPBOARD_SET",
	}).Info("📋 [Agent] 设置Agent剪贴板")

	err := robotgo.WriteAll(text)
	if err != nil {
		log.WithError(err).Error("📋 [Agent] 设置剪贴板失败")
		return err
	}
	log.Info("📋 [Agent] 剪贴板设置成功")
	return nil
}

// handleClipboardGet 读取Agent剪贴板，并回写到当前ws客户端
func handleClipboardGet(ws *websocket.Conn) error {
	log.Info("📋 [Agent] 收到CLIPBOARD_GET请求，开始读取剪贴板")

	// 添加更详细的剪贴板状态检查
	log.WithFields(log.Fields{
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		"action":    "clipboard_read_attempt",
	}).Info("📋 [Agent] 尝试读取剪贴板")

	text, err := robotgo.ReadAll()
	if err != nil {
		log.WithError(err).Error("📋 [Agent] 读取剪贴板失败")
		return fmt.Errorf("读取剪贴板失败: %w", err)
	}

	preview := text
	if len(text) > 100 {
		preview = text[:100] + "..."
	}

	log.WithFields(log.Fields{
		"text_length":  len(text),
		"text_preview": preview,
		"is_empty":     text == "",
		"raw_text":     fmt.Sprintf("%q", text), // 显示原始文本，包括特殊字符
	}).Info("📋 [Agent] 读取剪贴板成功，发送CLIPBOARD_UPDATE")

	msg := ControlMessage{
		Type: MSG_CLIPBOARD_UPDATE,
		Data: map[string]interface{}{
			"text":        text,
			"text_length": len(text),
			"char_count":  len([]rune(text)),
		},
		Timestamp: time.Now().Unix(),
	}
	err = sendControlMessage(ws, msg)
	if err != nil {
		log.WithError(err).Error("📋 [Agent] 发送CLIPBOARD_UPDATE失败")
		return err
	}
	log.Info("📋 [Agent] CLIPBOARD_UPDATE发送成功")
	return nil
}

// 轮询剪贴板监听已移除，改为事件驱动：收到 MSG_CLIPBOARD_GET 时读取并返回

// 新格式系统控制事件处理函数
func handleNewSystemDesktop() error {
	log.WithFields(log.Fields{
		"event_type": "SYSTEM_DESKTOP",
		"action":     "show_desktop",
		"shortcut":   "Win+D",
		"keys":       []string{"cmd", "d"},
	}).Info("🖥️ 执行显示桌面命令")

	log.Debug("🔽 按下 Win+D 组合键")
	robotgo.KeyDown("cmd")
	robotgo.KeyDown("d")
	time.Sleep(50 * time.Millisecond)
	robotgo.KeyUp("d")
	robotgo.KeyUp("cmd")

	log.WithFields(log.Fields{
		"result":      "success",
		"duration_ms": 50,
	}).Info("✅ 显示桌面命令执行完成")
	return nil
}

func handleNewSystemTaskManager() error {
	log.WithFields(log.Fields{
		"event_type": "SYSTEM_TASKMANAGER",
		"action":     "open_task_manager",
		"shortcut":   "Ctrl+Shift+Esc",
		"keys":       []string{"ctrl", "shift", "esc"},
	}).Info("📊 执行打开任务管理器命令")

	log.Debug("🔽 按下 Ctrl+Shift+Esc 组合键")
	robotgo.KeyDown("ctrl")
	robotgo.KeyDown("shift")
	robotgo.KeyDown("esc")
	time.Sleep(50 * time.Millisecond)
	robotgo.KeyUp("esc")
	robotgo.KeyUp("shift")
	robotgo.KeyUp("ctrl")

	log.WithFields(log.Fields{
		"result":      "success",
		"duration_ms": 50,
	}).Info("✅ 打开任务管理器命令执行完成")
	return nil
}

func handleNewSystemReboot() error {
	log.WithFields(log.Fields{
		"event_type":     "SYSTEM_REBOOT",
		"action":         "reboot_request",
		"security_level": "config_controlled",
		"execution":      "enabled",
	}).Info("🔄 收到系统重启命令")

	// 检查配置是否启用重启功能
	cfg := config.GetGlobalConfig()
	if !cfg.IsRebootEnabled() {
		log.WithFields(log.Fields{
			"reason": "reboot_disabled_in_config",
			"status": "rejected",
		}).Warn("⚠️ 重启功能已在配置中禁用")
		return nil
	}

	delay := cfg.GetRebootDelay()
	log.WithFields(log.Fields{
		"delay_seconds": delay,
		"status":        "scheduled",
	}).Info("🔄 系统重启已安排")

	// 在后台执行重启，避免阻塞WebSocket
	go func() {
		time.Sleep(time.Duration(delay) * time.Second)
		log.Info("🔄 开始执行系统重启")

		cmd := exec.Command("shutdown", "/r", "/t", "0")
		err := cmd.Run()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("❌ 系统重启执行失败")
		} else {
			log.Info("✅ 系统重启命令已执行")
		}
	}()

	return nil
}

// 处理坐标映射状态查询
func handleCoordinateMappingStatus(ws *websocket.Conn, msg ControlMessage) error {
	log.WithField("event_type", "COORDINATE_MAPPING_STATUS").Debug("查询坐标映射状态")

	coordinateMappingMutex.RLock()
	defer coordinateMappingMutex.RUnlock()

	if globalCoordinateMapping == nil || !globalCoordinateMapping.Initialized {
		sendErrorResponse(ws, "坐标映射未初始化", "请等待视频流启动完成")
		return nil
	}

	status := map[string]interface{}{
		"encoded_resolution": fmt.Sprintf("%dx%d", globalCoordinateMapping.EncodedWidth, globalCoordinateMapping.EncodedHeight),
		"screen_resolution":  fmt.Sprintf("%dx%d", globalCoordinateMapping.ScreenWidth, globalCoordinateMapping.ScreenHeight),
		"scale_x":            globalCoordinateMapping.ScaleX,
		"scale_y":            globalCoordinateMapping.ScaleY,
		"initialized":        globalCoordinateMapping.Initialized,
		"timestamp":          time.Now().Unix(),
	}

	sendSuccessResponse(ws, "坐标映射状态", status)
	return nil
}

// UpdateCoordinateMapping 更新坐标映射（供外部调用）
func UpdateCoordinateMapping(encodedWidth, encodedHeight int) error {
	coordinateMappingMutex.Lock()
	defer coordinateMappingMutex.Unlock()

	// 获取实际屏幕分辨率
	width, height := robotgo.GetScreenSize()

	// 计算缩放比例
	scaleX := float64(width) / float64(encodedWidth)
	scaleY := float64(height) / float64(encodedHeight)

	globalCoordinateMapping = &CoordinateMapping{
		EncodedWidth:  encodedWidth,
		EncodedHeight: encodedHeight,
		ScreenWidth:   width,
		ScreenHeight:  height,
		ScaleX:        scaleX,
		ScaleY:        scaleY,
		Initialized:   true,
	}

	log.WithFields(log.Fields{
		"encoded_resolution": fmt.Sprintf("%dx%d", encodedWidth, encodedHeight),
		"screen_resolution":  fmt.Sprintf("%dx%d", width, height),
		"scale_x":            scaleX,
		"scale_y":            scaleY,
		"caller":             "external_update",
	}).Info("坐标映射已更新")

	return nil
}

// GetCoordinateMappingStatus 获取坐标映射状态（供外部调用）
func GetCoordinateMappingStatus() map[string]interface{} {
	coordinateMappingMutex.RLock()
	defer coordinateMappingMutex.RUnlock()

	if globalCoordinateMapping == nil || !globalCoordinateMapping.Initialized {
		return map[string]interface{}{
			"initialized": false,
			"message":     "坐标映射未初始化",
		}
	}

	return map[string]interface{}{
		"encoded_resolution": fmt.Sprintf("%dx%d", globalCoordinateMapping.EncodedWidth, globalCoordinateMapping.EncodedHeight),
		"screen_resolution":  fmt.Sprintf("%dx%d", globalCoordinateMapping.ScreenWidth, globalCoordinateMapping.ScreenHeight),
		"scale_x":            globalCoordinateMapping.ScaleX,
		"scale_y":            globalCoordinateMapping.ScaleY,
		"initialized":        globalCoordinateMapping.Initialized,
		"timestamp":          time.Now().Unix(),
	}
}

// CoordinateMappingStatusHandler HTTP处理器：获取坐标映射状态
func CoordinateMappingStatusHandler(c *gin.Context) {
	status := GetCoordinateMappingStatus()

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "坐标映射状态查询成功",
		"data":      status,
		"timestamp": time.Now().Unix(),
	})
}
