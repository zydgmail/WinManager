package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// Config 应用配置结构
type Config struct {
	Database DatabaseConfig `json:"database"`
	Server   ServerConfig   `json:"server"`
	Agent    AgentConfig    `json:"agent"`
	Log      LogConfig      `json:"log"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Path string `json:"path"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port string `json:"port"`
}

// AgentConfig Agent配置
type AgentConfig struct {
	HTTPPort                int `json:"http_port"`                  // Agent HTTP端口
	GRPCPort                int `json:"grpc_port"`                  // Agent gRPC端口
	HeartbeatTimeoutSeconds int `json:"heartbeat_timeout_seconds"`  // 心跳超时时间(秒)，超过此时间判断设备离线
	OfflineCheckInterval    int `json:"offline_check_interval"`     // 离线检测间隔(秒)
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `json:"level"`
	File       string `json:"file"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	Compress   bool   `json:"compress"`
}

// GlobalConfig 全局配置实例
var GlobalConfig Config

// Init 初始化配置
func Init() {
	// 默认配置
	GlobalConfig = Config{
		Database: DatabaseConfig{
			Path: "./data.db",
		},
		Server: ServerConfig{
			Port: ":8080",
		},
		Agent: AgentConfig{
			HTTPPort:                50052, // Agent默认HTTP端口
			GRPCPort:                50051, // Agent默认gRPC端口
			HeartbeatTimeoutSeconds: 90,    // 默认90秒超时
			OfflineCheckInterval:    60,    // 默认60秒检查一次
		},
		Log: LogConfig{
			Level:      "debug",
			File:       "./logs/backend.log",
			MaxSize:    10,
			MaxBackups: 5,
			Compress:   false,
		},
	}

	// 尝试从配置文件加载
	if err := loadConfigFromFile("config.json"); err != nil {
		// 这里还不能使用logger，因为logger依赖config初始化
		log.Warnf("无法加载配置文件，使用默认配置: %v", err)
	}

	// 验证配置
	if err := validateConfig(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 这里还不能使用logger，因为logger依赖config初始化
	log.Infof("配置初始化完成")
}

// loadConfigFromFile 从文件加载配置
func loadConfigFromFile(filename string) error {
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("配置文件 %s 不存在", filename)
	}

	// 读取文件内容
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析JSON
	if err := json.Unmarshal(data, &GlobalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %v", err)
	}

	log.Infof("从文件 %s 加载配置成功", filename)

	return nil
}

// validateConfig 验证配置
func validateConfig() error {
	// 验证数据库路径
	if GlobalConfig.Database.Path == "" {
		return fmt.Errorf("数据库路径不能为空")
	}

	// 确保数据库目录存在
	dbDir := filepath.Dir(GlobalConfig.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("创建数据库目录失败: %v", err)
	}

	// 验证服务器端口
	if GlobalConfig.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}

	// 验证日志文件路径
	if GlobalConfig.Log.File == "" {
		return fmt.Errorf("日志文件路径不能为空")
	}

	// 确保日志目录存在
	logDir := filepath.Dir(GlobalConfig.Log.File)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	return nil
}

// GetDatabasePath 获取数据库路径
func GetDatabasePath() string {
	return GlobalConfig.Database.Path
}

// GetServerPort 获取服务器端口
func GetServerPort() string {
	return GlobalConfig.Server.Port
}

// GetLogConfig 获取日志配置
func GetLogConfig() LogConfig {
	return GlobalConfig.Log
}

// GetAgentHTTPPort 获取Agent HTTP端口
func GetAgentHTTPPort() int {
	return GlobalConfig.Agent.HTTPPort
}

// GetAgentGRPCPort 获取Agent gRPC端口
func GetAgentGRPCPort() int {
	return GlobalConfig.Agent.GRPCPort
}

// GetHeartbeatTimeoutSeconds 获取心跳超时时间(秒)
func GetHeartbeatTimeoutSeconds() int {
	return GlobalConfig.Agent.HeartbeatTimeoutSeconds
}

// GetOfflineCheckInterval 获取离线检测间隔(秒)
func GetOfflineCheckInterval() int {
	return GlobalConfig.Agent.OfflineCheckInterval
}

// SaveConfig 保存配置到文件
func SaveConfig(filename string) error {
	data, err := json.MarshalIndent(GlobalConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	// 这里还不能使用logger，因为可能在logger初始化之前调用
	log.Infof("配置保存到文件 %s 成功", filename)

	return nil
}
