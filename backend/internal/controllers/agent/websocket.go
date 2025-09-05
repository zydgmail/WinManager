package agent

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"winmanager-backend/internal/config"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

// WebSocketStream WebSocket视频流代理
func WebSocketStream(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("WebSocket流参数错误: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 获取实例信息
	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "实例不存在"})
		return
	}

	// 升级HTTP连接为WebSocket
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("升级WebSocket失败: %v", err)
		return
	}
	defer clientConn.Close()

	// 构建Agent WebSocket URL
	agentHTTPPort := config.GetAgentHTTPPort()
	agentWSURL := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", instance.Lan, agentHTTPPort),
		Path:   "/wsstream",
	}

	logger.Infof("连接到Agent WebSocket: %s", agentWSURL.String())

	// 连接到Agent的WebSocket
	agentConn, _, err := websocket.DefaultDialer.Dial(agentWSURL.String(), nil)
	if err != nil {
		logger.Errorf("连接Agent WebSocket失败: %v", err)
		clientConn.WriteMessage(websocket.TextMessage, []byte(`{"error": "连接Agent失败"}`))
		return
	}
	defer agentConn.Close()

	// 创建双向代理
	errChan := make(chan error, 2)

	// 客户端到Agent
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("客户端到Agent代理panic: %v", r)
			}
		}()
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Errorf("客户端连接异常关闭: %v", err)
				}
				errChan <- err
				return
			}
			if err := agentConn.WriteMessage(messageType, message); err != nil {
				logger.Errorf("向Agent发送消息失败: %v", err)
				errChan <- err
				return
			}
		}
	}()

	// Agent到客户端
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("Agent到客户端代理panic: %v", r)
			}
		}()
		for {
			messageType, message, err := agentConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Errorf("Agent连接异常关闭: %v", err)
				}
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(messageType, message); err != nil {
				logger.Errorf("向客户端发送消息失败: %v", err)
				errChan <- err
				return
			}
		}
	}()

	// 等待任一方向的连接出错
	<-errChan
	logger.Infof("WebSocket代理连接结束: ID=%d", id)
}

// WebSocketControl WebSocket控制代理
func WebSocketControl(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("WebSocket控制参数错误: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 获取实例信息
	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "实例不存在"})
		return
	}

	// 升级HTTP连接为WebSocket
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Errorf("升级WebSocket失败: %v", err)
		return
	}
	defer clientConn.Close()

	// 构建Agent WebSocket URL
	agentHTTPPort := config.GetAgentHTTPPort()
	agentWSURL := url.URL{
		Scheme: "ws",
		Host:   fmt.Sprintf("%s:%d", instance.Lan, agentHTTPPort),
		Path:   "/wscontrol",
	}

	logger.Infof("连接到Agent控制WebSocket: %s", agentWSURL.String())

	// 连接到Agent的WebSocket
	agentConn, _, err := websocket.DefaultDialer.Dial(agentWSURL.String(), nil)
	if err != nil {
		logger.Errorf("连接Agent控制WebSocket失败: %v", err)
		clientConn.WriteMessage(websocket.TextMessage, []byte(`{"error": "连接Agent失败"}`))
		return
	}
	defer agentConn.Close()

	// 创建双向代理
	errChan := make(chan error, 2)

	// 客户端到Agent
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("客户端到Agent控制代理panic: %v", r)
			}
		}()
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Errorf("客户端控制连接异常关闭: %v", err)
				}
				errChan <- err
				return
			}
			if err := agentConn.WriteMessage(messageType, message); err != nil {
				logger.Errorf("向Agent发送控制消息失败: %v", err)
				errChan <- err
				return
			}
		}
	}()

	// Agent到客户端
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("Agent到客户端控制代理panic: %v", r)
			}
		}()
		for {
			messageType, message, err := agentConn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logger.Errorf("Agent控制连接异常关闭: %v", err)
				}
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(messageType, message); err != nil {
				logger.Errorf("向客户端发送控制消息失败: %v", err)
				errChan <- err
				return
			}
		}
	}()

	// 等待任一方向的连接出错
	<-errChan
	logger.Infof("WebSocket控制代理连接结束: ID=%d", id)
}
