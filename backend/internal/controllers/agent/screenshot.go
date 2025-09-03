package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"winmanager-backend/internal/config"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// ScreenshotRequest 截图请求结构
type ScreenshotRequest struct {
	Quality int    `json:"quality"`
	Format  string `json:"format"`
}

// Screenshot 获取实例截图
func Screenshot(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("截图请求参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	logger.Infof("开始处理截图请求: ID=%d", id)

	// 获取实例信息
	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("获取实例信息成功: ID=%d, LAN=%s, Status=%d", id, instance.Lan, instance.Status)

	// 解析请求参数
	var req ScreenshotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 使用默认值
		req.Quality = 85
		req.Format = "jpeg"
		logger.Infof("使用默认截图参数: Quality=%d, Format=%s", req.Quality, req.Format)
	} else {
		logger.Infof("解析截图参数成功: Quality=%d, Format=%s", req.Quality, req.Format)
	}

	// 构建截图请求 - 使用配置中的Agent HTTP端口
	agentHTTPPort := config.GetAgentHTTPPort()
	screenshotURL := fmt.Sprintf("http://%s:%d/api/screenshot", instance.Lan, agentHTTPPort)

	logger.Infof("构建截图请求URL: %s", screenshotURL)

	requestBody, _ := json.Marshal(req)
	httpReq, err := http.NewRequest("POST", screenshotURL, bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Errorf("创建截图请求失败: %v", err)
		InternalErrorRes(c, "创建截图请求失败")
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	logger.Infof("发送截图请求到Agent: %s", screenshotURL)

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("截图请求失败: %v", err)
		InternalErrorRes(c, "截图请求失败")
		return
	}
	defer resp.Body.Close()

	logger.Infof("收到Agent响应: StatusCode=%d", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		// 读取错误响应内容
		errorBody, _ := io.ReadAll(resp.Body)
		logger.Errorf("截图请求失败: 状态码=%d, 响应内容=%s", resp.StatusCode, string(errorBody))
		InternalErrorRes(c, "截图请求失败")
		return
	}

	// 获取Agent返回的Content-Type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg" // 默认为WebP
	}

	logger.Infof("Agent响应Content-Type: %s", contentType)

	// 设置响应头
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	// 直接将Agent的响应流式传输给客户端
	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		logger.Errorf("传输截图数据失败: %v", err)
		return
	}

	logger.Infof("截图请求成功: ID=%d", id)
}
