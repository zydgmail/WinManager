package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"winmanager-agent/pkg/device"

	"github.com/shirou/gopsutil/v3/host"
	log "github.com/sirupsen/logrus"
)

// 全局变量保存注册的Agent ID
var registeredAgentID int

// RegisterResponse represents the response from agent registration
type RegisterResponse struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"` // 使用 interface{} 来处理不同的响应格式
	Message string      `json:"message"`
}

// RegisterAgent registers the agent with the server
func RegisterAgent(serverURL string) (int, error) {
	if serverURL == "" {
		return 0, fmt.Errorf("server URL is required")
	}

	log.WithField("服务器地址", serverURL).Info("正在向服务器注册 Agent")

	// Get device information
	deviceInfo, err := device.GetDeviceInfo()
	if err != nil {
		return 0, fmt.Errorf("failed to get device info: %w", err)
	}

	// Marshal device info to JSON
	jsonData, err := json.Marshal(deviceInfo)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal device info: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/api/register", serverURL)
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WinManager-Agent")

	// Send request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send registration request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if regResp.Code != 0 {
		return 0, fmt.Errorf("registration failed: %s", regResp.Message)
	}

	// 处理不同的 data 格式
	var agentID int
	switch data := regResp.Data.(type) {
	case float64:
		// 如果 data 是数字
		agentID = int(data)
	case map[string]interface{}:
		// 如果 data 是对象，尝试获取 id 字段
		if id, ok := data["id"]; ok {
			if idFloat, ok := id.(float64); ok {
				agentID = int(idFloat)
			}
		}
	default:
		log.WithField("data_type", fmt.Sprintf("%T", data)).Warn("Unknown data format in registration response")
		agentID = 1 // 使用默认值
	}

	log.WithField("Agent ID", agentID).Info("成功向服务器注册")

	// 保存Agent ID用于心跳
	registeredAgentID = agentID

	return agentID, nil
}

// StartHeartbeat starts sending periodic heartbeats to the server
func StartHeartbeat(serverURL string) {
	if serverURL == "" {
		log.Warn("Server URL not provided, heartbeat disabled")
		return
	}

	log.Info("正在启动心跳服务")

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			if err := sendHeartbeat(serverURL); err != nil {
				log.WithError(err).Error("Failed to send heartbeat")
			}
		}
	}()
}

// sendHeartbeat sends a heartbeat to the server
func sendHeartbeat(serverURL string) error {
	// Get current WAN IP
	wanIP, err := device.GetWanIP()
	if err != nil {
		log.WithError(err).Debug("Failed to get WAN IP for heartbeat")
		wanIP = "unknown"
	}

	// Get current system uptime
	uptime, err := host.Uptime()
	if err != nil {
		log.WithError(err).Debug("Failed to get uptime for heartbeat")
		uptime = 0
	}

	// Create heartbeat data
	heartbeatData := map[string]interface{}{
		"wan":       wanIP,
		"uptime":    uptime,
		"timestamp": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(heartbeatData)
	if err != nil {
		return fmt.Errorf("failed to marshal heartbeat data: %w", err)
	}

	// Send heartbeat (需要包含Agent ID)
	if registeredAgentID == 0 {
		return fmt.Errorf("agent not registered, cannot send heartbeat")
	}

	url := fmt.Sprintf("%s/api/heartbeat/%d", serverURL, registeredAgentID)
	req, err := http.NewRequest("PATCH", url, bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create heartbeat request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send heartbeat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("心跳失败，状态码: %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"Agent ID": registeredAgentID,
		"WAN IP":   wanIP,
	}).Debug("心跳发送成功")
	return nil
}

// UpdateProxyStatus updates the proxy status on the server
func UpdateProxyStatus() {
	// TODO: Implement proxy status update logic
	// This would involve checking current proxy configuration
	// and reporting it to the server
	log.Debug("Proxy status update not implemented yet")
}
