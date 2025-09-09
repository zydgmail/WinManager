package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 统一的日志接口，供项目其他包使用
var (
	// Info 信息级别日志
	Info  = log.Info
	Infof = log.Infof

	// Debug 调试级别日志
	Debug  = log.Debug
	Debugf = log.Debugf

	// Warn 警告级别日志
	Warn  = log.Warn
	Warnf = log.Warnf

	// Error 错误级别日志
	Error  = log.Error
	Errorf = log.Errorf

	// Fatal 致命错误日志
	Fatal  = log.Fatal
	Fatalf = log.Fatalf

	// WithError 带错误的日志
	WithError = log.WithError

	// WithField 带单个字段的日志
	WithField = log.WithField

	// WithFields 带多个字段的日志
	WithFields = log.WithFields
)

// Fields 日志字段类型别名
type Fields = log.Fields

// CustomFormatter 自定义日志格式器
type CustomFormatter struct{}

// Format 实现自定义格式: 【日志等级】【日期时间】【文件路径】【函数名】日志消息
func (f *CustomFormatter) Format(entry *log.Entry) ([]byte, error) {
	// 获取文件路径和行号
	var fileInfo string
	if entry.HasCaller() {
		// 只保留文件名和行号，去掉完整路径
		fileName := filepath.Base(entry.Caller.File)
		fileInfo = fmt.Sprintf("%s:%d", fileName, entry.Caller.Line)
	} else {
		fileInfo = "unknown"
	}

	// 获取函数名
	var funcName string
	if entry.HasCaller() {
		// 只保留函数名，去掉包路径
		fullFunc := entry.Caller.Function
		parts := strings.Split(fullFunc, "/")
		if len(parts) > 0 {
			lastPart := parts[len(parts)-1]
			funcName = lastPart
		} else {
			funcName = fullFunc
		}
	} else {
		funcName = "unknown"
	}

	// 构建日志消息，包含错误信息
	var message string
	if len(entry.Data) > 0 {
		// 如果有额外数据（如错误信息），将其添加到消息中
		var dataParts []string
		for key, value := range entry.Data {
			dataParts = append(dataParts, fmt.Sprintf("%s=%v", key, value))
		}
		message = fmt.Sprintf("%s [%s]", entry.Message, strings.Join(dataParts, ", "))
	} else {
		message = entry.Message
	}

	// 格式: 【日志等级】【日期时间】【文件路径】【函数名】日志消息
	logLine := fmt.Sprintf("【%s】【%s】【%s】【%s】%s\n",
		strings.ToUpper(entry.Level.String()),
		entry.Time.Format("2006-01-02 15:04:05"),
		fileInfo,
		funcName,
		message,
	)

	return []byte(logLine), nil
}

// SetupDebugLogger configures logging for debug mode
func SetupDebugLogger() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&CustomFormatter{})

	// 启用调用者信息 (文件路径和行号)
	log.SetReportCaller(true)

	log.Info("调试模式日志已启用")
}

// SetupProductionLogger configures logging for production mode
func SetupProductionLogger() {
	// Ensure logs directory exists
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		log.WithError(err).Warn("Failed to create logs directory, using stdout")
		SetupDebugLogger()
		return
	}

	logFile := filepath.Join(logsDir, "agent.log")

	logger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    10,   // 每个日志文件最大10MB
		MaxBackups: 5,    // 保留5个备份文件
		MaxAge:     7,    // 保留7天
		Compress:   true, // 压缩旧日志
		LocalTime:  true, // 使用本地时间
	}

	log.SetOutput(logger)
	log.SetLevel(log.InfoLevel)
	log.SetFormatter(&CustomFormatter{})

	// 启用调用者信息 (文件路径和行号)
	log.SetReportCaller(true)

	log.WithFields(log.Fields{
		"log_file":    logFile,
		"max_size":    "10MB",
		"max_backups": 5,
		"max_age":     "7天",
	}).Info("生产环境日志已启用")
}

// SetupTestLogger configures logging for testing
func SetupTestLogger() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.WarnLevel)
	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}
