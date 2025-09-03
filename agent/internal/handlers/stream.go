package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1500000,
	WriteBufferSize: 1500000,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// StartStreamHandler starts video streaming using global Hub
func StartStreamHandler(c *gin.Context) {
	log.Info("收到启动视频流请求")

	// 检查是否已经在运行
	if IsGlobalStreamingRunning() {
		log.Info("全局视频流已在运行，返回当前状态")
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"message": "视频流已在运行",
			"data": gin.H{
				"running": true,
				"stats": GetStreamingStats(),
			},
		})
		return
	}

	// 启动全局流媒体
	log.Info("开始启动全局视频流...")
	if err := StartGlobalStreaming(); err != nil {
		log.WithError(err).Error("启动全局流媒体失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": -1,
			"message": "启动流媒体失败",
			"error": err.Error(),
		})
		return
	}

	log.Info("全局视频流启动成功")
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "视频流启动成功",
		"data": gin.H{
			"running": IsGlobalStreamingRunning(),
			"stats": GetStreamingStats(),
			"websocket_url": "/wsstream",
			"instructions": "现在可以连接到 /wsstream 接收视频流",
		},
	})
}

// StopStreamHandler stops video streaming
func StopStreamHandler(c *gin.Context) {
	log.Info("收到停止视频流请求（前端兼容接口）")

	// 获取当前统计信息
	stats := GetStreamingStats()
	clientCount := stats["client_count"].(int)

	if !IsGlobalStreamingRunning() {
		log.Info("全局视频流未在运行")
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"message": "视频流未在运行",
			"data": gin.H{
				"running": false,
				"stats": stats,
			},
		})
		return
	}

	// 如果还有其他用户在使用，拒绝停止
	if clientCount > 0 {
		log.Infof("仍有 %d 个用户在观看视频流，拒绝停止操作", clientCount)
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"message": "其他用户正在观看，无法停止视频流",
		})
		return
	}

	// 没有其他用户，可以安全停止
	log.Info("没有其他用户，开始停止全局视频流")
	if err := StopGlobalStreaming(); err != nil {
		log.WithError(err).Error("停止全局流媒体失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code": -1,
			"message": "停止流媒体失败",
			"error": err.Error(),
		})
		return
	}

	log.Info("全局视频流停止成功")
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "视频流停止成功",
	})
}

// StreamStatusHandler returns current stream status
func StreamStatusHandler(c *gin.Context) {
	running := IsGlobalStreamingRunning()
	stats := GetStreamingStats()

	// 构建详细的状态信息
	statusData := gin.H{
		"running": running,
		"stats": stats,
		"timestamp": time.Now().Unix(),
	}

	// 如果正在运行，添加更多详细信息
	if running {
		statusData["websocket_url"] = "/wsstream"
		statusData["available_actions"] = []string{"stop", "connect_websocket"}
		statusData["message"] = "视频流正在运行，可以连接WebSocket"
	} else {
		statusData["available_actions"] = []string{"start"}
		statusData["message"] = "视频流未运行，请先调用 /api/startstream"
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": statusData,
	})
}

// WebSocketStreamHandler handles WebSocket video streaming using Hub pattern
func WebSocketStreamHandler(c *gin.Context) {
	log.Info("WebSocket流连接请求")

	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.WithError(err).Error("升级WebSocket失败")
		return
	}

	// 设置WebSocket参数
	ws.SetReadLimit(512)
	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	defer func() {
		log.Info("关闭WebSocket连接")
		ws.Close()
	}()

	// 使用Hub模式处理连接
	handleWebSocketConnection(ws)
}

// handleWebSocketConnection 使用Hub模式处理WebSocket连接
func handleWebSocketConnection(ws *websocket.Conn) {
	// 检查全局流媒体是否已启动
	if !IsGlobalStreamingRunning() {
		log.Warn("全局流媒体未启动，拒绝WebSocket连接")
		ws.WriteMessage(websocket.TextMessage, []byte(`{
			"error": "stream_not_started",
			"message": "请先调用 /api/startstream 启动流媒体服务",
			"code": 4001
		}`))
		ws.Close()
		return
	}

	// 获取全局Hub
	hub := GetGlobalHub()
	if hub == nil {
		log.Error("全局Hub未初始化")
		ws.WriteMessage(websocket.TextMessage, []byte(`{
			"error": "hub_not_initialized",
			"message": "流媒体服务未正确初始化",
			"code": 4002
		}`))
		ws.Close()
		return
	}

	// 创建连接对象
	conn := &Connection{
		ws:   ws,
		send: make(chan []byte, 256),
		hub:  hub,
		run:  true,
		stop: make(chan struct{}),
	}

	// 注册连接到Hub
	hub.register <- conn
	conn.key = hub.setHubConnName(conn)

	log.Infof("WebSocket客户端已连接到Hub: %s", conn.key)

	// 启动发送循环
	go conn.writePump()

	// 启动读取循环（处理客户端消息）
	conn.readPump()
}

// Connection的写入循环
func (c *Connection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.ws.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.ws.WriteMessage(websocket.BinaryMessage, message); err != nil {
				log.WithError(err).Error("WebSocket写入失败")
				return
			}

		case <-ticker.C:
			c.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.stop:
			return
		}
	}
}

// Connection的读取循环
func (c *Connection) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.ws.Close()
		close(c.stop)
	}()

	c.ws.SetReadLimit(512)
	c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.WithError(err).Error("WebSocket读取错误")
			}
			break
		}
	}
}
