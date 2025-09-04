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

// GetSystemInfo 获取系统信息
func GetSystemInfo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("获取系统信息参数错误: %v", err)
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

	// 构建系统信息请求
	agentHTTPPort := config.GetAgentHTTPPort()
	systemInfoURL := fmt.Sprintf("http://%s:%d/api/info", instance.Lan, agentHTTPPort)

	httpReq, err := http.NewRequest("GET", systemInfoURL, nil)
	if err != nil {
		logger.Errorf("创建系统信息请求失败: %v", err)
		InternalErrorRes(c, "创建系统信息请求失败")
		return
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("系统信息请求失败: %v", err)
		InternalErrorRes(c, "系统信息请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取系统信息响应失败: %v", err)
		InternalErrorRes(c, "读取系统信息响应失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析系统信息响应失败: %v", err)
		InternalErrorRes(c, "解析系统信息响应失败")
		return
	}

	logger.Infof("获取系统信息成功: ID=%d", id)
	SuccessRes(c, result)
}

// RebootDevice 重启设备
func RebootDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("重启设备参数错误: %v", err)
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

	// 构建重启请求
	agentHTTPPort := config.GetAgentHTTPPort()
	rebootURL := fmt.Sprintf("http://%s:%d/api/reboot", instance.Lan, agentHTTPPort)

	httpReq, err := http.NewRequest("POST", rebootURL, nil)
	if err != nil {
		logger.Errorf("创建重启请求失败: %v", err)
		InternalErrorRes(c, "创建重启请求失败")
		return
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("重启请求失败: %v", err)
		InternalErrorRes(c, "重启请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取重启响应失败: %v", err)
		InternalErrorRes(c, "读取重启响应失败")
		return
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		logger.Errorf("重启请求失败: 状态码=%d, 响应=%s", resp.StatusCode, string(body))
		InternalErrorRes(c, "重启请求失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析重启响应失败: %v", err)
		InternalErrorRes(c, "解析重启响应失败")
		return
	}

	logger.Infof("重启设备成功: ID=%d", id)

	// 直接返回Agent的响应，避免双重嵌套
	c.JSON(resp.StatusCode, result)
}
