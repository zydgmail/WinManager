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

// GatewayRequest 网关请求结构
type GatewayRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// GatewayResponse 网关响应结构
type GatewayResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

// ForwardToAgent 转发请求到Agent
func ForwardToAgent(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("转发请求参数错误: %v", err)
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

	// 创建转发请求
	req, err := http.NewRequest(c.Request.Method, agentURL, c.Request.Body)
	if err != nil {
		logger.Errorf("创建转发请求失败: %v", err)
		InternalErrorRes(c, "创建转发请求失败")
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
		logger.Errorf("转发请求失败: %v", err)
		InternalErrorRes(c, "转发请求失败")
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

	logger.Infof("转发请求成功: ID=%d, URL=%s, 状态=%d", id, agentURL, resp.StatusCode)
}

// HTTPGateway 通用HTTP网关控制器
func HTTPGateway() gin.HandlerFunc {
	return func(c *gin.Context) {
		// /api/gateway/:id/*path
		instanceParam := c.Param("id")
		gatewayPath := c.Param("path")

		logger.Infof("HTTP网关请求: %s %s%s", c.Request.Method, instanceParam, gatewayPath)

		var instance *models.Instance
		var err error

		// 尝试将参数解析为ID（数字）
		if id, parseErr := strconv.Atoi(instanceParam); parseErr == nil {
			// 参数是数字，按ID查询
			instance, err = models.GetInstance(id)
			if err != nil {
				logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
				ErrorRes(c, ErrDbReturn, "实例未找到")
				return
			}
		} else {
			// 参数不是数字，按LAN地址查询（兼容旧版本）
			instance, err = models.GetInstanceByLan(instanceParam)
			if err != nil {
				logger.Errorf("获取实例失败: LAN=%s, 错误=%v", instanceParam, err)
				ErrorRes(c, ErrDbReturn, "实例未找到")
				return
			}
		}

		// 构建目标URL
		agentHTTPPort := config.GetAgentHTTPPort()
		targetURL := fmt.Sprintf("http://%s:%d%s", instance.Lan, agentHTTPPort, gatewayPath)

		// 解析查询参数
		if c.Request.URL.RawQuery != "" {
			targetURL += "?" + c.Request.URL.RawQuery
		}

		// 创建网关请求
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			logger.Errorf("创建网关请求失败: %v", err)
			InternalErrorRes(c, "创建网关请求失败")
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
			logger.Errorf("网关请求失败: %s, 错误=%v", targetURL, err)
			InternalErrorRes(c, "网关请求失败")
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

		logger.Infof("HTTP网关成功: %s %s → %d", c.Request.Method, targetURL, resp.StatusCode)
	}
}
