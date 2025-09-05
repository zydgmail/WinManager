package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
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

// ShutdownDevice 关机设备
func ShutdownDevice(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("关机设备参数错误: %v", err)
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

	// 构建关机请求
	agentHTTPPort := config.GetAgentHTTPPort()
	shutdownURL := fmt.Sprintf("http://%s:%d/api/shutdown", instance.Lan, agentHTTPPort)

	httpReq, err := http.NewRequest("POST", shutdownURL, nil)
	if err != nil {
		logger.Errorf("创建关机请求失败: %v", err)
		InternalErrorRes(c, "创建关机请求失败")
		return
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("关机请求失败: %v", err)
		InternalErrorRes(c, "关机请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取关机响应失败: %v", err)
		InternalErrorRes(c, "读取关机响应失败")
		return
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		logger.Errorf("关机请求失败: 状态码=%d, 响应=%s", resp.StatusCode, string(body))
		InternalErrorRes(c, "关机请求失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析关机响应失败: %v", err)
		InternalErrorRes(c, "解析关机响应失败")
		return
	}

	logger.Infof("关机设备成功: ID=%d", id)

	// 直接返回Agent的响应，避免双重嵌套
	c.JSON(resp.StatusCode, result)
}

// ExecuteScript 执行脚本命令
func ExecuteScript(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("执行脚本参数错误: %v", err)
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

	// 构建脚本执行请求
	agentHTTPPort := config.GetAgentHTTPPort()
	execscriptURL := fmt.Sprintf("http://%s:%d/api/execscript", instance.Lan, agentHTTPPort)

	// 转发请求体
	httpReq, err := http.NewRequest("POST", execscriptURL, c.Request.Body)
	if err != nil {
		logger.Errorf("创建脚本执行请求失败: %v", err)
		InternalErrorRes(c, "创建脚本执行请求失败")
		return
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 60 * time.Second} // 脚本执行可能需要更长时间
	resp, err := client.Do(httpReq)
	if err != nil {
		logger.Errorf("脚本执行请求失败: %v", err)
		InternalErrorRes(c, "脚本执行请求失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取脚本执行响应失败: %v", err)
		InternalErrorRes(c, "读取脚本执行响应失败")
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		logger.Errorf("解析脚本执行响应失败: %v", err)
		InternalErrorRes(c, "解析脚本执行响应失败")
		return
	}

	logger.Infof("脚本执行成功: ID=%d", id)

	// 直接返回Agent的响应，避免双重嵌套
	c.JSON(resp.StatusCode, result)
}

// DownloadFile 下载文件
func DownloadFile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("文件下载参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	// 获取实例信息
	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		NotFoundRes(c, "实例不存在")
		return
	}

	// 获取文件路径参数
	filePath := c.Query("path")
	if filePath == "" {
		BadRequestRes(c, "文件路径参数缺失")
		return
	}

	// 构建下载URL
	agentHTTPPort := config.GetAgentHTTPPort()
	downloadURL := fmt.Sprintf("http://%s:%d/api/download?path=%s", instance.Lan, agentHTTPPort, filePath)

	logger.Infof("代理文件下载请求: %s", downloadURL)

	// 创建HTTP请求
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		logger.Errorf("创建下载请求失败: %v", err)
		InternalErrorRes(c, "请求创建失败")
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
		logger.Errorf("发送下载请求失败: %v", err)
		InternalErrorRes(c, "请求发送失败")
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
		return
	}

	logger.Infof("文件下载代理完成: ID=%d, Path=%s", id, filePath)
}

// UploadFile 上传文件
func UploadFile(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("文件上传参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	// 获取实例信息
	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		NotFoundRes(c, "实例不存在")
		return
	}

	// 构建上传URL
	agentHTTPPort := config.GetAgentHTTPPort()
	uploadURL := fmt.Sprintf("http://%s:%d/api/upload", instance.Lan, agentHTTPPort)

	// 添加查询参数
	if dir := c.Query("dir"); dir != "" {
		uploadURL += "?dir=" + dir
	}

	logger.Infof("代理文件上传请求: %s", uploadURL)

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		logger.Errorf("获取上传文件失败: %v", err)
		BadRequestRes(c, "获取上传文件失败")
		return
	}
	defer file.Close()

	// 创建multipart form
	formData := &bytes.Buffer{}
	writer := multipart.NewWriter(formData)

	// 创建form字段
	part, err := writer.CreateFormFile("file", header.Filename)
	if err != nil {
		logger.Errorf("创建form字段失败: %v", err)
		InternalErrorRes(c, "创建form字段失败")
		return
	}

	// 复制文件内容
	_, err = io.Copy(part, file)
	if err != nil {
		logger.Errorf("复制文件内容失败: %v", err)
		InternalErrorRes(c, "复制文件内容失败")
		return
	}

	// 关闭writer
	writer.Close()

	// 创建HTTP请求
	req, err := http.NewRequest("POST", uploadURL, formData)
	if err != nil {
		logger.Errorf("创建上传请求失败: %v", err)
		InternalErrorRes(c, "请求创建失败")
		return
	}

	// 设置Content-Type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 发送请求
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("发送上传请求失败: %v", err)
		InternalErrorRes(c, "请求发送失败")
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("读取上传响应失败: %v", err)
		InternalErrorRes(c, "响应读取失败")
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Status(resp.StatusCode)

	// 返回响应
	c.Writer.Write(body)

	logger.Infof("文件上传代理完成: ID=%d, Filename=%s", id, header.Filename)
}
