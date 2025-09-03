package handlers

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"

	"winmanager-agent/pkg/device"
	"winmanager-agent/pkg/screen"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// InfoHandler returns system information
func InfoHandler(c *gin.Context) {
	log.Debug("Handling info request")

	deviceInfo, err := device.GetDeviceInfo()
	if err != nil {
		log.WithError(err).Error("Failed to get device info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to get device information",
			"error":   err.Error(),
		})
		return
	}

	// Add runtime information
	info := map[string]interface{}{
		"device": deviceInfo,
		"runtime": map[string]interface{}{
			"go_version": runtime.Version(),
			"goroutines": runtime.NumGoroutine(),
			"memory_mb":  getMemoryUsage(),
		},
		"screen": screen.GetScreenInfo(),
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": info,
	})
}

// ScreenshotRequest 截图请求结构
type ScreenshotRequest struct {
	Quality int    `json:"quality"`
	Format  string `json:"format"`
	X       int    `json:"x"`
	Y       int    `json:"y"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// ScreenshotHandler captures and returns a screenshot
func ScreenshotHandler(c *gin.Context) {
	log.Infof("收到截图请求: Method=%s", c.Request.Method)

	// 默认参数
	format := "jpeg"
	quality := 85
	x, y, width, height := 0, 0, 0, 0

	// 根据请求方法解析参数
	if c.Request.Method == "POST" {
		// POST请求：从请求体解析JSON参数
		var req ScreenshotRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			if req.Format != "" {
				format = req.Format
			}
			if req.Quality > 0 && req.Quality <= 100 {
				quality = req.Quality
			}
			x, y, width, height = req.X, req.Y, req.Width, req.Height

			log.Infof("POST参数解析: Format=%s, Quality=%d, X=%d, Y=%d, Width=%d, Height=%d",
				format, quality, x, y, width, height)
		} else {
			log.Infof("POST参数解析失败，使用默认值: %v", err)
		}
	} else {
		// GET请求：从查询参数解析
		format = c.DefaultQuery("format", "jpeg")

		// Parse quality parameter for JPEG
		if format == "jpeg" {
			if q := c.Query("quality"); q != "" {
				if parsedQuality, err := strconv.Atoi(q); err == nil && parsedQuality >= 1 && parsedQuality <= 100 {
					quality = parsedQuality
				}
			}
		}

		// Parse region parameters if provided
		if xStr := c.Query("x"); xStr != "" {
			if parsedX, err := strconv.Atoi(xStr); err == nil {
				x = parsedX
			}
		}
		if yStr := c.Query("y"); yStr != "" {
			if parsedY, err := strconv.Atoi(yStr); err == nil {
				y = parsedY
			}
		}
		if widthStr := c.Query("width"); widthStr != "" {
			if parsedWidth, err := strconv.Atoi(widthStr); err == nil {
				width = parsedWidth
			}
		}
		if heightStr := c.Query("height"); heightStr != "" {
			if parsedHeight, err := strconv.Atoi(heightStr); err == nil {
				height = parsedHeight
			}
		}

		log.Infof("GET参数解析: Format=%s, Quality=%d, X=%d, Y=%d, Width=%d, Height=%d",
			format, quality, x, y, width, height)
	}

	// Set screenshot options
	opts := screen.ScreenshotOptions{
		Format:  screen.ImageFormat(format),
		Quality: quality,
		X:       x,
		Y:       y,
		Width:   width,
		Height:  height,
	}

	if opts.Width > 0 || opts.Height > 0 {
		log.Infof("区域截图: X=%d, Y=%d, Width=%d, Height=%d",
			opts.X, opts.Y, opts.Width, opts.Height)
	} else {
		log.Infof("全屏截图")
	}

	// Capture screenshot
	log.Infof("开始捕获截图")

	imageData, err := screen.CaptureScreenshotWithOptions(opts)
	if err != nil {
		log.Errorf("截图捕获失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to capture screenshot",
			"error":   err.Error(),
		})
		return
	}

	log.Infof("截图捕获成功: 大小=%d bytes", len(imageData))

	// Set appropriate content type
	var contentType string
	switch format {
	case "png":
		contentType = "image/png"
	case "jpeg":
		contentType = "image/jpeg"
	case "webp":
		contentType = "image/webp"
	default:
		contentType = "application/octet-stream"
	}

	log.Infof("设置响应头: ContentType=%s, ContentLength=%d", contentType, len(imageData))

	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(imageData)))
	c.Data(http.StatusOK, contentType, imageData)

	log.Infof("截图响应发送完成")
}

// getMemoryUsage returns current memory usage in MB
func getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024
}
