package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"winmanager-agent/internal/config"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Placeholder handlers for routes that need to be implemented

// 剪贴板HTTP接口已废弃，使用WebSocket控制接口进行剪贴板同步与粘贴

func ProcessHandler(c *gin.Context) {
	log.Debug("Process handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func ShutdownHandler(c *gin.Context) {
	log.Info("收到关机设备请求")

	// 获取配置
	config := config.GetGlobalConfig()

	// 检查关机功能是否启用
	if !config.IsShutdownEnabled() {
		log.WithFields(log.Fields{
			"event_type":     "SHUTDOWN_REQUEST",
			"action":         "shutdown_request",
			"security_level": "disabled",
			"execution":      "blocked",
		}).Warn("🔌 关机功能被禁用")

		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": "关机功能被禁用，请在配置文件中启用",
			"data": gin.H{
				"action":         "shutdown_disabled",
				"config_enabled": false,
				"status":         "blocked",
			},
		})
		return
	}

	// 获取延迟配置（使用重启延迟配置）
	delay := config.GetRebootDelay()

	log.WithFields(log.Fields{
		"event_type":     "SHUTDOWN_REQUEST",
		"action":         "shutdown_request",
		"security_level": "enabled",
		"execution":      "scheduled",
		"delay_seconds":  delay,
	}).Info("🔌 准备执行系统关机")

	// 在后台执行关机，避免阻塞响应
	go func() {
		log.WithFields(log.Fields{
			"delay_seconds": delay,
		}).Info("⏰ 关机倒计时开始")

		// 延迟指定秒数后执行关机
		time.Sleep(time.Duration(delay) * time.Second)

		log.Info("🔌 正在执行系统关机...")

		// 执行Windows关机命令
		cmd := exec.Command("shutdown", "/s", "/t", "0")
		err := cmd.Run()

		if err != nil {
			log.WithError(err).Error("❌ 关机命令执行失败")
		} else {
			log.Info("✅ 关机命令执行成功")
		}
	}()

	// 立即返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("关机指令已发送，系统将在%d秒后关机", delay),
		"data": gin.H{
			"action":         "shutdown_scheduled",
			"delay_seconds":  delay,
			"config_enabled": true,
			"status":         "scheduled",
		},
	})
}

func RebootHandler(c *gin.Context) {
	log.Info("收到重启设备请求")

	// 获取配置
	config := config.GetGlobalConfig()

	// 检查重启功能是否启用
	if !config.IsRebootEnabled() {
		log.WithFields(log.Fields{
			"event_type":     "REBOOT_REQUEST",
			"action":         "reboot_request",
			"security_level": "disabled",
			"execution":      "blocked",
		}).Warn("🔄 重启功能被禁用")

		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": "重启功能被禁用，请在配置文件中启用",
			"data": gin.H{
				"action":         "reboot_disabled",
				"config_enabled": false,
				"status":         "blocked",
			},
		})
		return
	}

	// 获取重启延迟
	delay := config.GetRebootDelay()

	log.WithFields(log.Fields{
		"event_type":     "REBOOT_REQUEST",
		"action":         "reboot_request",
		"security_level": "enabled",
		"execution":      "scheduled",
		"delay_seconds":  delay,
	}).Info("🔄 准备执行系统重启")

	// 在后台执行重启，避免阻塞响应
	go func() {
		log.WithFields(log.Fields{
			"delay_seconds": delay,
		}).Info("⏰ 重启倒计时开始")

		// 延迟指定秒数后执行重启
		time.Sleep(time.Duration(delay) * time.Second)

		log.Info("🔄 正在执行系统重启...")

		// 执行Windows重启命令
		cmd := exec.Command("shutdown", "/r", "/t", "0")
		err := cmd.Run()

		if err != nil {
			log.WithError(err).Error("❌ 重启命令执行失败")
		} else {
			log.Info("✅ 重启命令执行成功")
		}
	}()

	// 立即返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": fmt.Sprintf("重启指令已发送，系统将在%d秒后重启", delay),
		"data": gin.H{
			"action":         "reboot_scheduled",
			"delay_seconds":  delay,
			"config_enabled": true,
			"status":         "scheduled",
		},
	})
}

func ExecScriptHandler(c *gin.Context) {
	log.Info("收到脚本执行请求")

	// 获取配置
	config := config.GetGlobalConfig()

	// 检查命令执行功能是否启用
	if !config.IsCommandsEnabled() {
		log.WithFields(log.Fields{
			"event_type":     "COMMAND_REQUEST",
			"action":         "command_execution",
			"security_level": "disabled",
			"execution":      "blocked",
		}).Warn("⚡ 命令执行功能被禁用")

		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": "命令执行功能被禁用，请在配置文件中启用",
			"data": gin.H{
				"action":         "command_disabled",
				"config_enabled": false,
				"status":         "blocked",
			},
		})
		return
	}

	// 获取命令参数
	var cmdReq struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
		Timeout int      `json:"timeout"`
	}

	if err := c.ShouldBindJSON(&cmdReq); err != nil {
		log.WithError(err).Error("解析命令参数失败")
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    1,
			"message": "命令参数格式错误",
			"data": gin.H{
				"error": err.Error(),
			},
		})
		return
	}

	log.WithFields(log.Fields{
		"command": cmdReq.Command,
		"args":    cmdReq.Args,
		"timeout": cmdReq.Timeout,
	}).Info("⚡ 准备执行命令")

	// 在后台执行命令
	go func() {
		log.Info("⚡ 正在执行命令...")

		// 构建命令
		var cmd *exec.Cmd
		if len(cmdReq.Args) > 0 {
			cmd = exec.Command(cmdReq.Command, cmdReq.Args...)
		} else {
			cmd = exec.Command(cmdReq.Command)
		}

		// 设置超时
		timeout := time.Duration(30) * time.Second
		if cmdReq.Timeout > 0 {
			timeout = time.Duration(cmdReq.Timeout) * time.Second
		}

		// 使用context控制超时
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)

		// 执行命令并获取输出
		output, err := cmd.CombinedOutput()

		if err != nil {
			log.WithError(err).WithField("output", string(output)).Error("❌ 命令执行失败")
		} else {
			log.WithField("output", string(output)).Info("✅ 命令执行成功")
		}
	}()

	// 立即返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "命令执行请求已接收",
		"data": gin.H{
			"action":         "command_scheduled",
			"command":        cmdReq.Command,
			"args":           cmdReq.Args,
			"config_enabled": true,
			"status":         "scheduled",
		},
	})
}

func DownloadHandler(c *gin.Context) {
	// 获取文件路径参数
	filePath := c.Query("path")
	if filePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "文件路径参数缺失",
		})
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "文件不存在",
		})
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"file_path": filePath,
			"error":     err.Error(),
		}).Error("获取文件信息失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取文件信息失败",
		})
		return
	}

	// 设置响应头
	filename := filepath.Base(filePath)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 发送文件
	c.File(filePath)

	log.WithFields(log.Fields{
		"file_path": filePath,
		"file_size": fileInfo.Size(),
		"filename":  filename,
	}).Info("文件下载成功")
}

func UploadHandler(c *gin.Context) {
	// 获取上传目录参数
	uploadDir := c.Query("dir")
	if uploadDir == "" {
		uploadDir = "./uploads" // 默认上传目录
	}

	// 确保上传目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.WithFields(log.Fields{
			"upload_dir": uploadDir,
			"error":      err.Error(),
		}).Error("创建上传目录失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "创建上传目录失败",
		})
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("获取上传文件失败")
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "获取上传文件失败",
		})
		return
	}

	// 生成安全的文件名
	filename := file.Filename
	// 移除路径分隔符，防止目录遍历攻击
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	// 如果文件名为空，使用时间戳
	if filename == "" {
		filename = fmt.Sprintf("upload_%d", time.Now().Unix())
	}

	// 构建完整文件路径
	filePath := filepath.Join(uploadDir, filename)

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		log.WithFields(log.Fields{
			"file_path": filePath,
			"error":     err.Error(),
		}).Error("保存上传文件失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "保存上传文件失败",
		})
		return
	}

	// 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.WithFields(log.Fields{
			"file_path": filePath,
			"error":     err.Error(),
		}).Error("获取上传文件信息失败")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取文件信息失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "文件上传成功",
		"data": gin.H{
			"filename":   filename,
			"file_path":  filePath,
			"file_size":  fileInfo.Size(),
			"upload_dir": uploadDir,
		},
	})

	log.WithFields(log.Fields{
		"filename":   filename,
		"file_path":  filePath,
		"file_size":  fileInfo.Size(),
		"upload_dir": uploadDir,
	}).Info("文件上传成功")
}

func StartProxyHandler(c *gin.Context) {
	log.Debug("StartProxy handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func StopProxyHandler(c *gin.Context) {
	log.Debug("StopProxy handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func CheckProxyHandler(c *gin.Context) {
	log.Debug("CheckProxy handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func GetProxyListHandler(c *gin.Context) {
	log.Debug("GetProxyList handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func GetAccountHandler(c *gin.Context) {
	log.Debug("GetAccount handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func SaveGameAccountHandler(c *gin.Context) {
	log.Debug("SaveGameAccount handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func CmdHandler(c *gin.Context) {
	log.Debug("Cmd handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func ServerConfHandler(c *gin.Context) {
	log.Debug("ServerConf handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func SessionHandler(c *gin.Context) {
	log.Debug("Session handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func WatchdogStartHandler(c *gin.Context) {
	log.Debug("WatchdogStart handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func WatchdogStopHandler(c *gin.Context) {
	log.Debug("WatchdogStop handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}

func WatchdogUpdateHandler(c *gin.Context) {
	log.Debug("WatchdogUpdate handler called")
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Not implemented yet"})
}
