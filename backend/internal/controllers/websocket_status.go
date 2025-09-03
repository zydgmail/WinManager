package controllers

import (
	"winmanager-backend/internal/logger"

	"github.com/gin-gonic/gin"
)

// GetWebSocketStats 获取WebSocket连接统计
func GetWebSocketStats(c *gin.Context) {
	wsManager := GetWebSocketManager()
	stats := wsManager.GetStats()
	
	logger.Infof("获取WebSocket统计信息: %+v", stats)
	SuccessRes(c, stats)
}

// GetInstanceConnections 获取实例的WebSocket连接
func GetInstanceConnections(c *gin.Context) {
	instanceLan := c.Param("id")
	
	wsManager := GetWebSocketManager()
	connections := wsManager.GetConnectionsByInstance(instanceLan)
	
	// 构建响应数据
	var result []map[string]interface{}
	for _, conn := range connections {
		result = append(result, map[string]interface{}{
			"id":           conn.ID,
			"type":         conn.Type,
			"instance_lan": conn.InstanceLan,
			"created_at":   conn.CreatedAt,
			"last_active":  conn.LastActive,
			"is_active":    conn.IsActive,
		})
	}
	
	logger.Infof("获取实例WebSocket连接: 实例=%s, 连接数=%d", instanceLan, len(result))
	SuccessRes(c, result)
}

// CloseInstanceConnections 关闭实例的所有WebSocket连接
func CloseInstanceConnections(c *gin.Context) {
	instanceLan := c.Param("id")
	
	wsManager := GetWebSocketManager()
	connections := wsManager.GetConnectionsByInstance(instanceLan)
	
	// 关闭所有连接
	for _, conn := range connections {
		wsManager.RemoveConnection(conn.ID)
	}
	
	logger.Infof("关闭实例WebSocket连接: 实例=%s, 关闭数=%d", instanceLan, len(connections))
	SuccessRes(c, gin.H{
		"message": "连接已关闭",
		"count":   len(connections),
	})
}

// CloseWebSocketConnection 关闭指定的WebSocket连接
func CloseWebSocketConnection(c *gin.Context) {
	connID := c.Param("conn_id")
	
	wsManager := GetWebSocketManager()
	if conn, exists := wsManager.GetConnection(connID); exists {
		wsManager.RemoveConnection(connID)
		logger.Infof("关闭WebSocket连接: ID=%s, 类型=%s", connID, conn.Type)
		SuccessRes(c, gin.H{"message": "连接已关闭"})
	} else {
		logger.Warnf("WebSocket连接不存在: ID=%s", connID)
		ErrorRes(c, ErrNotFound, "连接不存在")
	}
}
