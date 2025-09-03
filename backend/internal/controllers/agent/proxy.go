package agent

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
)

// ProxyRequest 代理请求结构
type ProxyRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// ProxyResponse 代理响应结构
type ProxyResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// ProxyToAgent 代理请求到Agent
func ProxyToAgent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("代理请求参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	// 获取实例信息
	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	// 构建Agent URL - 提取路径中的agent部分
	// 从 /agent/:id/*path 中提取 *path 部分
	path := c.Param("path")
	if path == "" {
		path = "/"
	}

	agentHTTPPort := config.GetAgentHTTPPort()
	agentURL := fmt.Sprintf("http://%s:%d%s", instance.Lan, agentHTTPPort, path)

	// 创建代理请求
	req, err := http.NewRequest(c.Request.Method, agentURL, c.Request.Body)
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
		logger.Errorf("代理请求失败: %v", err)
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

	logger.Infof("代理请求成功: ID=%d, URL=%s, 状态=%d", id, agentURL, resp.StatusCode)
}
