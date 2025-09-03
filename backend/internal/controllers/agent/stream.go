package agent

import (
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

// StartStream 启动视频流
func StartStream(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("启动视频流参数错误: %v", err)
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

	// 构建启动视频流请求
	agentHTTPPort := config.GetAgentHTTPPort()
	startStreamURL := fmt.Sprintf("http://%s:%d/api/startstream", instance.Lan, agentHTTPPort)

	httpReq, err := http.NewRequest("GET", startStreamURL, nil)
	if err != nil {
		logger.Errorf("创建启动视频流请求失败: %v", err)
		InternalErrorRes(c, "创建启动视频流请求失败")
		return
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("启动视频流请求失败: %v", err)
		InternalErrorRes(c, "启动视频流请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取启动视频流响应失败: %v", err)
		InternalErrorRes(c, "读取启动视频流响应失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析启动视频流响应失败: %v", err)
		InternalErrorRes(c, "解析启动视频流响应失败")
		return
	}

	logger.Infof("启动视频流成功: ID=%d", id)
	SuccessRes(c, result)
}

// StopStream 停止视频流
func StopStream(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("停止视频流参数错误: %v", err)
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

	// 构建停止视频流请求
	agentHTTPPort := config.GetAgentHTTPPort()
	stopStreamURL := fmt.Sprintf("http://%s:%d/api/stopstream", instance.Lan, agentHTTPPort)

	httpReq, err := http.NewRequest("GET", stopStreamURL, nil)
	if err != nil {
		logger.Errorf("创建停止视频流请求失败: %v", err)
		InternalErrorRes(c, "创建停止视频流请求失败")
		return
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("停止视频流请求失败: %v", err)
		InternalErrorRes(c, "停止视频流请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取停止视频流响应失败: %v", err)
		InternalErrorRes(c, "读取停止视频流响应失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析停止视频流响应失败: %v", err)
		InternalErrorRes(c, "解析停止视频流响应失败")
		return
	}

	logger.Infof("停止视频流成功: ID=%d", id)
	SuccessRes(c, result)
}
