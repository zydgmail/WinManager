package services

import (
	"time"
	"winmanager-backend/internal/config"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"
)

// OfflineDetector 离线检测服务
type OfflineDetector struct {
	ticker   *time.Ticker
	stopChan chan struct{}
	running  bool
}

// NewOfflineDetector 创建离线检测服务实例
func NewOfflineDetector() *OfflineDetector {
	return &OfflineDetector{
		stopChan: make(chan struct{}),
		running:  false,
	}
}

// Start 启动离线检测服务
func (od *OfflineDetector) Start() {
	if od.running {
		logger.Warn("离线检测服务已经在运行中")
		return
	}

	// 获取配置
	checkInterval := config.GetOfflineCheckInterval()
	if checkInterval <= 0 {
		logger.Warn("离线检测间隔配置无效，使用默认值60秒")
		checkInterval = 60
	}

	logger.Infof("启动离线检测服务，检测间隔: %d秒", checkInterval)

	od.ticker = time.NewTicker(time.Duration(checkInterval) * time.Second)
	od.running = true

	go func() {
		defer func() {
			od.ticker.Stop()
			od.running = false
			logger.Info("离线检测服务已停止")
		}()

		// 启动时立即执行一次检测
		od.checkOfflineDevices()

		for {
			select {
			case <-od.ticker.C:
				od.checkOfflineDevices()
			case <-od.stopChan:
				return
			}
		}
	}()

	logger.Info("离线检测服务启动成功")
}

// Stop 停止离线检测服务
func (od *OfflineDetector) Stop() {
	if !od.running {
		logger.Warn("离线检测服务未在运行")
		return
	}

	logger.Info("正在停止离线检测服务...")
	close(od.stopChan)
}

// IsRunning 检查服务是否在运行
func (od *OfflineDetector) IsRunning() bool {
	return od.running
}

// checkOfflineDevices 检查并更新离线设备状态
func (od *OfflineDetector) checkOfflineDevices() {
	// 获取心跳超时配置
	timeoutSeconds := config.GetHeartbeatTimeoutSeconds()
	if timeoutSeconds <= 0 {
		logger.Warn("心跳超时时间配置无效，使用默认值90秒")
		timeoutSeconds = 90
	}

	logger.Debugf("开始检查离线设备，心跳超时时间: %d秒", timeoutSeconds)

	// 更新超时设备状态
	affectedRows, err := models.UpdateOfflineInstances(timeoutSeconds)
	if err != nil {
		logger.Errorf("检查离线设备失败: %v", err)
		return
	}

	if affectedRows > 0 {
		logger.Infof("检测到 %d 个设备离线，已更新状态", affectedRows)
	} else {
		logger.Debugf("离线检测完成，无设备离线")
	}
}

// GetStatus 获取服务状态信息
func (od *OfflineDetector) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"running":                od.running,
		"check_interval_seconds": config.GetOfflineCheckInterval(),
		"timeout_seconds":        config.GetHeartbeatTimeoutSeconds(),
	}

	if od.running {
		status["next_check"] = time.Now().Add(time.Duration(config.GetOfflineCheckInterval()) * time.Second)
	}

	return status
}

// 全局离线检测服务实例
var globalOfflineDetector *OfflineDetector

// InitOfflineDetector 初始化全局离线检测服务
func InitOfflineDetector() {
	if globalOfflineDetector != nil {
		logger.Warn("离线检测服务已经初始化")
		return
	}

	globalOfflineDetector = NewOfflineDetector()
	globalOfflineDetector.Start()
}

// StopOfflineDetector 停止全局离线检测服务
func StopOfflineDetector() {
	if globalOfflineDetector != nil {
		globalOfflineDetector.Stop()
		globalOfflineDetector = nil
	}
}

// GetOfflineDetectorStatus 获取全局离线检测服务状态
func GetOfflineDetectorStatus() map[string]interface{} {
	if globalOfflineDetector == nil {
		return map[string]interface{}{
			"running": false,
			"error":   "service not initialized",
		}
	}
	return globalOfflineDetector.GetStatus()
}
