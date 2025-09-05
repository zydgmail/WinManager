package controllers

import (
	"net/http"
	"time"

	"winmanager-agent/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// SetupRoutes configures all HTTP routes
func SetupRoutes(router *gin.RouterGroup) {
	// Health check
	router.GET("/health", healthHandler) // ✅ 健康检查接口，返回服务状态

	// API routes
	apiGroup := router.Group("/api")
	{
		// System information
		apiGroup.GET("/info", handlers.InfoHandler)              // ✅ 获取系统信息（设备信息、运行时状态、屏幕信息）
		apiGroup.GET("/screenshot", handlers.ScreenshotHandler)  // ✅ 获取屏幕截图（GET方式，URL参数）
		apiGroup.POST("/screenshot", handlers.ScreenshotHandler) // ✅ 获取屏幕截图（POST方式，JSON参数）

		// Encoder and capture APIs
		apiGroup.GET("/encoders", handlers.EncoderInfoHandler)                 // ✅ 获取支持的编码器和捕获方法信息
		apiGroup.GET("/encoded-screenshot", handlers.EncodedScreenshotHandler) // ✅ 获取编码后的截图（支持多种编码格式）
		apiGroup.GET("/stream", handlers.StreamingHandler)                     // ❌ HTTP视频流接口（未实现，返回501）

		// Video streaming control
		apiGroup.GET("/startstream", handlers.StartStreamHandler)   // ✅ 启动视频流服务
		apiGroup.GET("/stopstream", handlers.StopStreamHandler)     // ✅ 停止视频流服务
		apiGroup.GET("/streamstatus", handlers.StreamStatusHandler) // ✅ 获取视频流状态

		// Coordinate mapping status
		apiGroup.GET("/coordinate-mapping", handlers.CoordinateMappingStatusHandler) // ✅ 获取坐标映射状态

		// Input handling
		apiGroup.GET("/keyboard", handlers.KeyboardHandler) // ❌ 键盘输入处理（未实现）
		apiGroup.POST("/paste", handlers.PasteHandler)      // ❌ 剪贴板粘贴操作（未实现）

		// Process management
		apiGroup.GET("/process", handlers.ProcessHandler)        // ❌ 进程管理（启动/停止进程，未实现）
		apiGroup.POST("/reboot", handlers.RebootHandler)         // ✅ 系统重启（已实现）
		apiGroup.POST("/shutdown", handlers.ShutdownHandler)     // ✅ 系统关机（已实现）
		apiGroup.POST("/execscript", handlers.ExecScriptHandler) //

		// File operations
		apiGroup.GET("/download", handlers.DownloadHandler) // ❌ 文件下载（未实现）
		apiGroup.POST("/upload", handlers.UploadHandler)    // ❌ 文件上传（未实现）

		// Proxy management
		apiGroup.GET("/startip", handlers.StartProxyHandler)     // ❌ 启动代理IP（未实现）
		apiGroup.GET("/stopip", handlers.StopProxyHandler)       // ❌ 停止代理IP（未实现）
		apiGroup.GET("/checkproxy", handlers.CheckProxyHandler)  // ❌ 检查代理状态（未实现）
		apiGroup.GET("/proxylist", handlers.GetProxyListHandler) // ❌ 获取代理列表（未实现）

		// Command execution
		apiGroup.POST("/cmd", handlers.CmdHandler)              // ❌ 执行系统命令（未实现）
		apiGroup.GET("/serverconf", handlers.ServerConfHandler) // ❌ 获取服务器配置（未实现）

		// Session management
		apiGroup.Any("/session", handlers.SessionHandler) // ❌ 会话管理（未实现）
	}

	// Watchdog routes (for compatibility)
	watchdogGroup := router.Group("/watchdog")
	{
		watchdogGroup.GET("/start", handlers.WatchdogStartHandler)   // ❌ 启动看门狗服务（未实现，兼容性接口）
		watchdogGroup.GET("/stop", handlers.WatchdogStopHandler)     // ❌ 停止看门狗服务（未实现，兼容性接口）
		watchdogGroup.GET("/update", handlers.WatchdogUpdateHandler) // ❌ 更新看门狗配置（未实现，兼容性接口）
	}

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler())) // ✅ Prometheus监控指标接口

	// WebSocket streaming
	router.GET("/wsstream", handlers.WebSocketStreamHandler) // ✅ WebSocket视频流接口（H.264实时流）

	// WebSocket control interface (compatible with legacy project)
	router.GET("/wscontrol", handlers.WebSocketControlHandler) // ✅ WebSocket控制接口（鼠标键盘操控）

	// Static files (if needed)
	router.StaticFS("/static", http.Dir("web")) // ✅ 静态文件服务（web目录）
}

// healthHandler returns the health status of the agent
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": getCurrentTimestamp(),
	})
}

// getCurrentTimestamp returns the current Unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
