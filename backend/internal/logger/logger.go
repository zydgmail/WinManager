package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"winmanager-backend/internal/config"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 统一的日志接口，供项目其他包使用
var (
	// Info 信息级别日志
	Info = log.Info
	Infof = log.Infof

	// Debug 调试级别日志
	Debug = log.Debug
	Debugf = log.Debugf

	// Warn 警告级别日志
	Warn = log.Warn
	Warnf = log.Warnf

	// Error 错误级别日志
	Error = log.Error
	Errorf = log.Errorf

	// Fatal 致命错误日志
	Fatal = log.Fatal
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

// ChineseFormatter 中文日志格式化器
type ChineseFormatter struct{}

// Format 格式化日志输出
// 格式：日志等级-[日期时间]-[文件路径]-[函数名]-日志消息
func (f *ChineseFormatter) Format(entry *log.Entry) ([]byte, error) {
	// 获取调用者信息
	var file string
	var function string
	if entry.HasCaller() {
		file = fmt.Sprintf("%s:%d", filepath.Base(entry.Caller.File), entry.Caller.Line)
		ss := strings.Split(entry.Caller.Function, ".")
		function = ss[len(ss)-1]
	} else {
		file = "unknown"
		function = "unknown"
	}

	// 格式化时间
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// 格式化日志等级
	level := strings.ToUpper(entry.Level.String())

	// 构建日志消息
	message := fmt.Sprintf("%s-[%s]-[%s]-[%s]-%s\n",
		level,
		timestamp,
		file,
		function,
		entry.Message)

	return []byte(message), nil
}

// Init 初始化日志系统
func Init() {
	// 设置自定义格式化器
	log.SetFormatter(&ChineseFormatter{})

	// 启用调用者信息
	log.SetReportCaller(true)

	// 获取日志配置
	logConfig := config.GetLogConfig()

	// 设置日志级别
	level, err := log.ParseLevel(logConfig.Level)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	// 确保日志目录存在
	logDir := filepath.Dir(logConfig.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("创建日志目录失败: %v", err)
	}

	// 设置日志输出
	if logConfig.File != "" {
		// 使用 lumberjack 进行日志轮转
		logger := &lumberjack.Logger{
			Filename:   logConfig.File,
			MaxSize:    logConfig.MaxSize,    // MB
			MaxBackups: logConfig.MaxBackups, // 保留的旧文件数量
			Compress:   logConfig.Compress,   // 是否压缩
		}

		// 同时输出到文件和控制台
		multiWriter := io.MultiWriter(os.Stdout, logger)
		log.SetOutput(multiWriter)
	} else {
		// 只输出到控制台
		log.SetOutput(os.Stdout)
	}

	log.Infof("日志系统初始化完成: 级别=%s, 文件=%s", level.String(), logConfig.File)
}

// GetLogger 获取日志实例
func GetLogger() *log.Logger {
	return log.StandardLogger()
}

// SetLevel 设置日志级别
func SetLevel(level string) error {
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		return fmt.Errorf("无效的日志级别: %s", level)
	}
	log.SetLevel(logLevel)

	log.Infof("日志级别已更新: %s", level)

	return nil
}
