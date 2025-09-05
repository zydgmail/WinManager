package controllers

import (
	"winmanager-backend/internal/controllers/agent"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/services"

	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由
func InitRouter(ctx *gin.RouterGroup) {
	logger.Infof("初始化路由配置")

	// 健康检查
	ctx.GET("/health", func(c *gin.Context) {
		logger.Infof("健康检查请求")
		SuccessRes(c, gin.H{"status": "ok", "message": "WinManager Backend is running"})
	})

	// 系统状态
	setupSystemRoutes(ctx)

	// 版本信息
	ctx.GET("/version", func(c *gin.Context) {
		logger.Infof("版本信息请求")
		SuccessRes(c, gin.H{"version": "1.0.0", "name": "winmanager-backend"})
	})

	// 实例管理路由
	setupInstanceRoutes(ctx)

	// 分组管理路由
	setupGroupRoutes(ctx)

	// WebSocket路由
	setupWebSocketRoutes(ctx)

	// Agent交互路由（包含网关转发功能）
	setupAgentRoutes(ctx)

	logger.Infof("路由配置完成")
}

// setupSystemRoutes 设置系统状态路由
func setupSystemRoutes(ctx *gin.RouterGroup) {
	system := ctx.Group("/system")
	{
		// 离线检测服务状态
		system.GET("/offline-detector/status", func(c *gin.Context) {
			logger.Infof("获取离线检测服务状态请求")
			status := services.GetOfflineDetectorStatus()
			SuccessRes(c, status)
		})
	}
}

// setupInstanceRoutes 设置实例相关路由
func setupInstanceRoutes(ctx *gin.RouterGroup) {
	logger.Infof("设置实例管理路由")

	ctx.POST("/register", Register)

	ctx.PATCH("/heartbeat/:id", Heartbeat)

	// 实例管理
	ctx.GET("/instances", ListInstances)
	ctx.GET("/instances/:id", GetInstance)
	ctx.PATCH("/instances/:id", PatchInstance)
	ctx.DELETE("/instances/:id", DeleteInstance)
	ctx.PATCH("/instances/move-group", MoveGroupInstance)
}

// setupGroupRoutes 设置分组相关路由
func setupGroupRoutes(ctx *gin.RouterGroup) {
	logger.Infof("设置分组管理路由")

	// 分组管理
	ctx.GET("/groups", ListGroups)
	ctx.GET("/groups/:id", GetGroup)
	ctx.POST("/groups", CreateGroup)
	ctx.PATCH("/groups/:id", PatchGroup)
	ctx.DELETE("/groups/:id", DeleteGroup)
}

// setupWebSocketRoutes 设置WebSocket相关路由
func setupWebSocketRoutes(ctx *gin.RouterGroup) {
	logger.Infof("设置WebSocket路由")

	// Guacamole WebSocket连接
	ctx.GET("/ws/:id", WsController())
	ctx.GET("/ws/state/:id", StateController())

	// 视频流WebSocket代理
	ctx.GET("/ws/:id/stream", agent.WebSocketStream)
}

// setupAgentRoutes 设置Agent交互相关路由
func setupAgentRoutes(ctx *gin.RouterGroup) {
	logger.Infof("设置Agent交互路由")

	// Agent路由组 - 所有Agent相关接口都以/agent/开头
	agentGroup := ctx.Group("/agent")
	{
		// 系统信息
		agentGroup.GET("/:id/info", agent.GetSystemInfo)

		// 截图接口
		agentGroup.POST("/:id/screenshot", agent.Screenshot)

		// 视频流控制
		agentGroup.GET("/:id/startstream", agent.StartStream)
		agentGroup.GET("/:id/stopstream", agent.StopStream)

		// 命令执行
		agentGroup.POST("/:id/execute", agent.ExecuteScript)

		// 系统控制
		agentGroup.POST("/:id/reboot", agent.RebootDevice)
		agentGroup.POST("/:id/shutdown", agent.ShutdownDevice)
		agentGroup.POST("/:id/execscript", agent.ExecuteScript)

		// WebSocket接口组 - 单独分组避免路径冲突
		wsGroup := agentGroup.Group("/ws")
		{
			// WebSocket视频流
			wsGroup.GET("/:id/stream", agent.WebSocketStream)
		}

		// 网关转发接口（放在最后，处理其他所有请求）
		// agentGroup.Any("/:id/*path", agent.ForwardToAgent)
	}

	// WebSocket状态管理（保持原有路径）
	ctx.GET("/websocket/stats", GetWebSocketStats)
	ctx.GET("/websocket/instances/:id", GetInstanceConnections)
	ctx.DELETE("/websocket/instances/:id", CloseInstanceConnections)
	ctx.DELETE("/websocket/connections/:conn_id", CloseWebSocketConnection)
}
