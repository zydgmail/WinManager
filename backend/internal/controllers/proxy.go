package controllers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"winmanager-backend/internal/config"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// StreamController WebSocket视频流代理控制器
func StreamController() gin.HandlerFunc {
	upgrade := websocket.Upgrader{
		ReadBufferSize:  1500000, // 与Agent保持一致
		WriteBufferSize: 1500000,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return func(c *gin.Context) {
		// /ws/:id/stream
		instanceParam := c.Param("id")
		logger.Infof("视频流代理连接到实例: %s", instanceParam)

		var instance *models.Instance
		var err error

		// 尝试将参数解析为ID（数字）
		if id, parseErr := strconv.Atoi(instanceParam); parseErr == nil {
			// 参数是数字，按ID查询
			instance, err = models.GetInstance(id)
			if err != nil {
				logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
				c.JSON(404, gin.H{"error": "实例未找到"})
				return
			}
		} else {
			// 参数不是数字，按LAN地址查询（兼容旧版本）
			instance, err = models.GetInstanceByLan(instanceParam)
			if err != nil {
				logger.Errorf("获取实例失败: LAN=%s, 错误=%v", instanceParam, err)
				c.JSON(404, gin.H{"error": "实例未找到"})
				return
			}
		}

		// 升级前端连接为WebSocket
		frontendWs, err := upgrade.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			logger.Errorf("升级前端WebSocket失败: %v", err)
			return
		}
		defer frontendWs.Close()

		// 连接到Agent的WebSocket视频流
		agentHTTPPort := config.GetAgentHTTPPort()
		agentURL := fmt.Sprintf("ws://%s:%d/wsstream", instance.Lan, agentHTTPPort)
		agentWs, _, err := websocket.DefaultDialer.Dial(agentURL, nil)
		if err != nil {
			logger.Errorf("连接Agent WebSocket失败: %s, 错误=%v", agentURL, err)
			frontendWs.WriteMessage(websocket.CloseMessage, []byte("无法连接到Agent"))
			return
		}
		defer agentWs.Close()

		// 创建连接管理
		connID := generateConnectionID(instance.Lan, "stream")
		conn := &WebSocketConnection{
			ID:          connID,
			Type:        "stream",
			InstanceLan: instance.Lan,
			FrontendWs:  frontendWs,
			AgentWs:     agentWs,
			CreatedAt:   time.Now(),
			LastActive:  time.Now(),
			IsActive:    true,
		}

		// 添加到管理器
		wsManager := GetWebSocketManager()
		wsManager.AddConnection(conn)
		defer wsManager.RemoveConnection(connID)

		logger.Infof("视频流代理建立成功: 前端 ↔ Backend ↔ Agent(%s), 连接ID=%s", instance.Lan, connID)

		// 启动双向代理
		go proxyWebSocketWithManager(frontendWs, agentWs, "前端→Agent", connID)
		proxyWebSocketWithManager(agentWs, frontendWs, "Agent→前端", connID)
	}
}

// ProxyController HTTP代理控制器
func ProxyController() gin.HandlerFunc {
	return func(c *gin.Context) {
		// /api/proxy/:id/*path
		instanceParam := c.Param("id")
		proxyPath := c.Param("path")

		logger.Infof("HTTP代理请求: %s %s%s", c.Request.Method, instanceParam, proxyPath)

		var instance *models.Instance
		var err error

		// 尝试将参数解析为ID（数字）
		if id, parseErr := strconv.Atoi(instanceParam); parseErr == nil {
			// 参数是数字，按ID查询
			instance, err = models.GetInstance(id)
			if err != nil {
				logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
				ErrorRes(c, ErrNotFound, "实例未找到")
				return
			}
		} else {
			// 参数不是数字，按LAN地址查询（兼容旧版本）
			instance, err = models.GetInstanceByLan(instanceParam)
			if err != nil {
				logger.Errorf("获取实例失败: LAN=%s, 错误=%v", instanceParam, err)
				ErrorRes(c, ErrNotFound, "实例未找到")
				return
			}
		}

		// 构建目标URL
		agentHTTPPort := config.GetAgentHTTPPort()
		targetURL := fmt.Sprintf("http://%s:%d%s", instance.Lan, agentHTTPPort, proxyPath)

		// 解析查询参数
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		// 创建代理请求
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			logger.Errorf("创建代理请求失败: %v", err)
			InternalErrorRes(c, "创建代理请求失败")
			return
		}

		// 复制请求头
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// 发送请求
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			logger.Errorf("代理请求失败: %s, 错误=%v", targetURL, err)
			InternalErrorRes(c, "代理请求失败")
			return
		}
		defer resp.Body.Close()

		// 复制响应头
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// 设置状态码
		c.Status(resp.StatusCode)

		// 复制响应体
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			logger.Errorf("复制响应体失败: %v", err)
		}

		logger.Infof("HTTP代理成功: %s %s → %d", c.Request.Method, targetURL, resp.StatusCode)
	}
}

// StartStreamProxy 启动视频流代理
func StartStreamProxy(c *gin.Context) {
	instanceParam := c.Param("id")

	var instance *models.Instance
	var err error

	// 尝试将参数解析为ID（数字）
	if id, parseErr := strconv.Atoi(instanceParam); parseErr == nil {
		// 参数是数字，按ID查询
		instance, err = models.GetInstance(id)
		if err != nil {
			logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
			ErrorRes(c, ErrNotFound, "实例未找到")
			return
		}
	} else {
		// 参数不是数字，按LAN地址查询（兼容旧版本）
		instance, err = models.GetInstanceByLan(instanceParam)
		if err != nil {
			logger.Errorf("获取实例失败: LAN=%s, 错误=%v", instanceParam, err)
			ErrorRes(c, ErrNotFound, "实例未找到")
			return
		}
	}

	// 代理到Agent的startstream接口
	agentHTTPPort := config.GetAgentHTTPPort()
	targetURL := fmt.Sprintf("http://%s:%d/api/startstream", instance.Lan, agentHTTPPort)

	logger.Infof("启动视频流代理: %s", targetURL)

	// 设置超时时间
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		logger.Errorf("启动视频流失败: %s, 错误=%v", targetURL, err)
		InternalErrorRes(c, "启动视频流失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Infof("启动视频流成功: %s", instance.Lan)
		SuccessRes(c, gin.H{"message": "视频流启动成功"})
	} else {
		logger.Errorf("启动视频流失败: %s, 状态码=%d", instance.Lan, resp.StatusCode)
		ErrorRes(c, ErrInternal, "启动视频流失败")
	}
}

// StopStreamProxy 停止视频流代理
func StopStreamProxy(c *gin.Context) {
	instanceParam := c.Param("id")

	var instance *models.Instance
	var err error

	// 尝试将参数解析为ID（数字）
	if id, parseErr := strconv.Atoi(instanceParam); parseErr == nil {
		// 参数是数字，按ID查询
		instance, err = models.GetInstance(id)
		if err != nil {
			logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
			ErrorRes(c, ErrNotFound, "实例未找到")
			return
		}
	} else {
		// 参数不是数字，按LAN地址查询（兼容旧版本）
		instance, err = models.GetInstanceByLan(instanceParam)
		if err != nil {
			logger.Errorf("获取实例失败: LAN=%s, 错误=%v", instanceParam, err)
			ErrorRes(c, ErrNotFound, "实例未找到")
			return
		}
	}

	// 代理到Agent的stopstream接口
	agentHTTPPort := config.GetAgentHTTPPort()
	targetURL := fmt.Sprintf("http://%s:%d/api/stopstream", instance.Lan, agentHTTPPort)

	logger.Infof("停止视频流代理: %s", targetURL)

	// 设置超时时间
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(targetURL)
	if err != nil {
		logger.Errorf("停止视频流失败: %s, 错误=%v", targetURL, err)
		InternalErrorRes(c, "停止视频流失败")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		logger.Infof("停止视频流成功: %s", instance.Lan)
		SuccessRes(c, gin.H{"message": "视频流停止成功"})
	} else {
		logger.Errorf("停止视频流失败: %s, 状态码=%d", instance.Lan, resp.StatusCode)
		ErrorRes(c, ErrInternal, "停止视频流失败")
	}
}

// proxyWebSocket 代理WebSocket消息
func proxyWebSocket(from, to *websocket.Conn, direction string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("WebSocket代理异常 %s: %v", direction, r)
		}
	}()

	for {
		messageType, data, err := from.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("WebSocket读取错误 %s: %v", direction, err)
			}
			break
		}

		err = to.WriteMessage(messageType, data)
		if err != nil {
			logger.Errorf("WebSocket写入错误 %s: %v", direction, err)
			break
		}

		// 记录数据传输（仅在调试模式下）
		if messageType == websocket.BinaryMessage {
			logger.Debugf("WebSocket代理 %s: 传输 %d 字节", direction, len(data))
		}
	}

	logger.Infof("WebSocket代理结束: %s", direction)
}

// proxyWebSocketWithManager 带管理器的WebSocket代理
func proxyWebSocketWithManager(from, to *websocket.Conn, direction, connID string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("WebSocket代理异常 %s: %v", direction, r)
		}
	}()

	wsManager := GetWebSocketManager()

	for {
		messageType, data, err := from.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Errorf("WebSocket读取错误 %s: %v", direction, err)
			}
			break
		}

		err = to.WriteMessage(messageType, data)
		if err != nil {
			logger.Errorf("WebSocket写入错误 %s: %v", direction, err)
			break
		}

		// 更新活跃时间
		wsManager.UpdateLastActive(connID)

		// 记录数据传输（仅在调试模式下）
		if messageType == websocket.BinaryMessage {
			logger.Debugf("WebSocket代理 %s: 传输 %d 字节", direction, len(data))
		}
	}

	logger.Infof("WebSocket代理结束: %s", direction)
}
