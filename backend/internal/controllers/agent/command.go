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

// ExecuteCommand 执行命令
func ExecuteCommand(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("执行命令参数错误: %v", err)
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

	// 获取命令参数
	var cmdReq struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Timeout int      `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&cmdReq); err != nil {
		logger.Errorf("绑定命令参数失败: %v", err)
		ErrorRes(c, ErrBindJson, err.Error())
		return
	}

	// 构建命令执行请求
	agentHTTPPort := config.GetAgentHTTPPort()
	commandURL := fmt.Sprintf("http://%s:%d/api/execute", instance.Lan, agentHTTPPort)

	requestBody, _ := json.Marshal(cmdReq)
	httpReq, err := http.NewRequest("POST", commandURL, bytes.NewBuffer(requestBody))
	if err != nil {
		logger.Errorf("创建命令执行请求失败: %v", err)
		InternalErrorRes(c, "创建命令执行请求失败")
		return
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// 设置超时
	timeout := 30 * time.Second
	if cmdReq.Timeout > 0 {
		timeout = time.Duration(cmdReq.Timeout) * time.Second
	}

	// 发送请求
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("命令执行请求失败: %v", err)
		InternalErrorRes(c, "命令执行请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取命令执行响应失败: %v", err)
		InternalErrorRes(c, "读取命令执行响应失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析命令执行响应失败: %v", err)
		InternalErrorRes(c, "解析命令执行响应失败")
		return
	}

	logger.Infof("命令执行成功: ID=%d, 命令=%s", id, cmdReq.Command)
	SuccessRes(c, result)
}
