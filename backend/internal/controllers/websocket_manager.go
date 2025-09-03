package controllers

import (
	"fmt"
	"sync"
	"time"
	"winmanager-backend/internal/logger"

	"github.com/gorilla/websocket"
)

// WebSocketManager WebSocket连接管理器
type WebSocketManager struct {
	connections map[string]*WebSocketConnection
	mutex       sync.RWMutex
}

// WebSocketConnection WebSocket连接信息
type WebSocketConnection struct {
	ID          string
	Type        string // "stream", "control", "state"
	InstanceLan string
	FrontendWs  *websocket.Conn
	AgentWs     *websocket.Conn
	CreatedAt   time.Time
	LastActive  time.Time
	IsActive    bool
}

// 全局WebSocket管理器
var wsManager = &WebSocketManager{
	connections: make(map[string]*WebSocketConnection),
}

// AddConnection 添加WebSocket连接
func (wm *WebSocketManager) AddConnection(conn *WebSocketConnection) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	
	wm.connections[conn.ID] = conn
	logger.Infof("添加WebSocket连接: ID=%s, 类型=%s, 实例=%s", conn.ID, conn.Type, conn.InstanceLan)
}

// RemoveConnection 移除WebSocket连接
func (wm *WebSocketManager) RemoveConnection(id string) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	
	if conn, exists := wm.connections[id]; exists {
		conn.IsActive = false
		if conn.FrontendWs != nil {
			conn.FrontendWs.Close()
		}
		if conn.AgentWs != nil {
			conn.AgentWs.Close()
		}
		delete(wm.connections, id)
		logger.Infof("移除WebSocket连接: ID=%s, 类型=%s", id, conn.Type)
	}
}

// GetConnection 获取WebSocket连接
func (wm *WebSocketManager) GetConnection(id string) (*WebSocketConnection, bool) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	
	conn, exists := wm.connections[id]
	return conn, exists
}

// GetConnectionsByInstance 获取实例的所有连接
func (wm *WebSocketManager) GetConnectionsByInstance(instanceLan string) []*WebSocketConnection {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	
	var connections []*WebSocketConnection
	for _, conn := range wm.connections {
		if conn.InstanceLan == instanceLan && conn.IsActive {
			connections = append(connections, conn)
		}
	}
	return connections
}

// GetActiveConnections 获取所有活跃连接
func (wm *WebSocketManager) GetActiveConnections() []*WebSocketConnection {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	
	var connections []*WebSocketConnection
	for _, conn := range wm.connections {
		if conn.IsActive {
			connections = append(connections, conn)
		}
	}
	return connections
}

// UpdateLastActive 更新最后活跃时间
func (wm *WebSocketManager) UpdateLastActive(id string) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	
	if conn, exists := wm.connections[id]; exists {
		conn.LastActive = time.Now()
	}
}

// CleanupInactiveConnections 清理不活跃的连接
func (wm *WebSocketManager) CleanupInactiveConnections() {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	
	timeout := 5 * time.Minute
	now := time.Now()
	
	for id, conn := range wm.connections {
		if now.Sub(conn.LastActive) > timeout {
			logger.Warnf("清理不活跃连接: ID=%s, 类型=%s, 最后活跃=%v",
				id, conn.Type, conn.LastActive)
			
			conn.IsActive = false
			if conn.FrontendWs != nil {
				conn.FrontendWs.Close()
			}
			if conn.AgentWs != nil {
				conn.AgentWs.Close()
			}
			delete(wm.connections, id)
		}
	}
}

// GetStats 获取连接统计信息
func (wm *WebSocketManager) GetStats() map[string]interface{} {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"total_connections": len(wm.connections),
		"by_type":          make(map[string]int),
		"by_instance":      make(map[string]int),
	}
	
	byType := stats["by_type"].(map[string]int)
	byInstance := stats["by_instance"].(map[string]int)
	
	for _, conn := range wm.connections {
		if conn.IsActive {
			byType[conn.Type]++
			byInstance[conn.InstanceLan]++
		}
	}
	
	return stats
}

// StartCleanupRoutine 启动清理例程
func (wm *WebSocketManager) StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			wm.CleanupInactiveConnections()
		}
	}()
	
	logger.Infof("WebSocket连接清理例程已启动")
}

// generateConnectionID 生成连接ID
func generateConnectionID(instanceLan, connType string) string {
	return fmt.Sprintf("%s_%s_%d", instanceLan, connType, time.Now().UnixNano())
}

// GetWebSocketManager 获取全局WebSocket管理器
func GetWebSocketManager() *WebSocketManager {
	return wsManager
}

// 初始化WebSocket管理器
func init() {
	wsManager.StartCleanupRoutine()
}
