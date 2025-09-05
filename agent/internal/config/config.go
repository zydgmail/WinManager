package config

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"winmanager-agent/pkg/device"

	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
)

// FileConfig represents the JSON configuration file structure
type FileConfig struct {
	Version    string           `json:"version"`
	Server     ServerConfig     `json:"server"`
	Agent      AgentConfig      `json:"agent"`
	Screen     ScreenConfig     `json:"screen"`
	Encoder    EncoderConfig    `json:"encoder"`
	Input      InputConfig      `json:"input"`
	Proxy      ProxyConfig      `json:"proxy"`
	Monitoring MonitoringConfig `json:"monitoring"`
	System     SystemConfig     `json:"system"`
}

type ServerConfig struct {
	URL           string `json:"url"`
	Timeout       int    `json:"timeout"`
	RetryInterval int    `json:"retry_interval"`
}

type AgentConfig struct {
	HTTPPort int    `json:"http_port"`
	GRPCPort int    `json:"grpc_port"`
	Debug    bool   `json:"debug"`
	LogLevel string `json:"log_level"`
}

type ScreenConfig struct {
	JPEGQuality   int    `json:"jpeg_quality"`
	CaptureMethod string `json:"capture_method"`
}

type EncoderConfig struct {
	DefaultCodec   string            `json:"default_codec"`
	JPEGQuality    int               `json:"jpeg_quality"`
	H264Preset     string            `json:"h264_preset"`
	H264Tune       string            `json:"h264_tune"`
	H264Profile    string            `json:"h264_profile"`
	H264Bitrate    int64             `json:"h264_bitrate"`
	VP8Bitrate     int               `json:"vp8_bitrate"`
	NVENCBitrate   int64             `json:"nvenc_bitrate"`
	NVENCPreset    string            `json:"nvenc_preset"`
	FrameRate      int               `json:"frame_rate"`
	EnabledCodecs  []string          `json:"enabled_codecs"`
	CodecPriority  []string          `json:"codec_priority"`
	CustomSettings map[string]string `json:"custom_settings"`
	Debug          DebugConfig       `json:"debug"`
}

type DebugConfig struct {
	SavePath          string `json:"save_path"`           // 保存路径
	SaveVideoDuration int    `json:"save_video_duration"` // 保存视频时长（秒），0表示不保存
}

type InputConfig struct {
	MouseEnabled    bool `json:"mouse_enabled"`
	KeyboardEnabled bool `json:"keyboard_enabled"`
	PasteEnabled    bool `json:"paste_enabled"`
}

type ProxyConfig struct {
	Enabled    bool   `json:"enabled"`
	AutoDetect bool   `json:"auto_detect"`
	URL        string `json:"url"`
	Username   string `json:"username"`
	Password   string `json:"password"`
}

type MonitoringConfig struct {
	MetricsEnabled     bool `json:"metrics_enabled"`
	MetricsInterval    int  `json:"metrics_interval"`
	HealthCheckEnabled bool `json:"health_check_enabled"`
}

type SystemConfig struct {
	RebootEnabled   bool `json:"reboot_enabled"`
	RebootDelay     int  `json:"reboot_delay"` // 重启延迟秒数
	ShutdownEnabled bool `json:"shutdown_enabled"`
	CommandsEnabled bool `json:"commands_enabled"` // 是否启用系统命令执行
}

// Config holds the application configuration
type Config struct {
	fileConfig    *FileConfig
	serverURL     string
	autoMonitor   bool
	localIP       string
	mutex         sync.RWMutex
	cache         *cache.Cache
	cronScheduler *cron.Cron
}

// NewConfig creates a new configuration instance
func NewConfig() *Config {
	return &Config{
		cache:         cache.New(3*time.Second, 10*time.Second),
		cronScheduler: cron.New(),
	}
}

// LoadConfigFile loads configuration from a JSON file
func (c *Config) LoadConfigFile(configPath string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.WithField("config_path", configPath).Info("Config file not found, using defaults")
		c.fileConfig = getDefaultConfig()
		return nil
	}

	// Read config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var fileConfig FileConfig
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	c.fileConfig = &fileConfig

	log.WithFields(log.Fields{
		"config_path":          configPath,
		"version":              fileConfig.Version,
		"debug_save_path":      fileConfig.Encoder.Debug.SavePath,
		"debug_video_duration": fileConfig.Encoder.Debug.SaveVideoDuration,
	}).Info("Configuration loaded from file")

	return nil
}

// getDefaultConfig returns default configuration
func getDefaultConfig() *FileConfig {
	return &FileConfig{
		Version: "1.0.0",
		Server: ServerConfig{
			URL:           "http://172.17.1.242:9090",
			Timeout:       30,
			RetryInterval: 5,
		},
		Agent: AgentConfig{
			HTTPPort: 50052,
			GRPCPort: 50051,
			Debug:    false,
			LogLevel: "info",
		},
		Screen: ScreenConfig{
			JPEGQuality:   80,
			CaptureMethod: "robotgo",
		},
		Encoder: EncoderConfig{
			DefaultCodec:   "h264",
			JPEGQuality:    80,
			H264Preset:     "medium",
			H264Tune:       "zerolatency",
			H264Profile:    "baseline",
			H264Bitrate:    20000000, // 20 Mbps
			VP8Bitrate:     8192,
			NVENCBitrate:   50000000,
			NVENCPreset:    "fast",
			FrameRate:      20,
			EnabledCodecs:  []string{"h264", "jpeg", "jpeg-turbo", "vp8"},
			CodecPriority:  []string{"h264", "vp8", "jpeg-turbo", "jpeg"},
			CustomSettings: make(map[string]string),
			Debug: DebugConfig{
				SavePath:          "./debug",
				SaveVideoDuration: 0, // 默认保存30秒视频，设为0则不保存
			},
		},
		Input: InputConfig{
			MouseEnabled:    true,
			KeyboardEnabled: true,
			PasteEnabled:    true,
		},
		Proxy: ProxyConfig{
			Enabled:    false,
			AutoDetect: true,
			URL:        "",
			Username:   "",
			Password:   "",
		},
		Monitoring: MonitoringConfig{
			MetricsEnabled:     true,
			MetricsInterval:    15,
			HealthCheckEnabled: true,
		},
		System: SystemConfig{
			RebootEnabled:   true, // 默认禁用重启功能，需要手动启用
			RebootDelay:     3,    // 默认5秒延迟
			ShutdownEnabled: true,
			CommandsEnabled: true,
		},
	}
}

// Initialize sets up the configuration with the provided server URL
func (c *Config) Initialize(serverURL string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Load default config if not already loaded
	if c.fileConfig == nil {
		c.fileConfig = getDefaultConfig()
	}

	// Get local IP address
	localIP, err := device.GetLocalIP()
	if err != nil {
		log.WithError(err).Warn("Failed to get local IP, using 'unknown'")
		localIP = "unknown"
	}
	c.localIP = localIP

	// Set server URL (command line overrides config file)
	if serverURL != "" {
		c.serverURL = serverURL
	} else if c.fileConfig.Server.URL != "" {
		c.serverURL = c.fileConfig.Server.URL
	} else {
		// Try to get server config from remote or use default
		c.serverURL = c.getServerConfig(localIP)
	}

	if c.serverURL == "" {
		return fmt.Errorf("server URL not configured")
	}

	// Start cron scheduler
	c.cronScheduler.Start()

	log.WithFields(log.Fields{
		"本地IP":  localIP,
		"服务器地址": c.serverURL,
		"版本":    c.fileConfig.Version,
	}).Info("配置初始化完成")

	return nil
}

// GetServerURL returns the configured server URL
func (c *Config) GetServerURL() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.serverURL
}

// SetServerURL updates the server URL
func (c *Config) SetServerURL(url string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.serverURL = url
}

// GetLocalIP returns the local IP address
func (c *Config) GetLocalIP() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.localIP
}

// IsAutoMonitorEnabled returns whether auto monitoring is enabled
func (c *Config) IsAutoMonitorEnabled() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.autoMonitor
}

// SetAutoMonitor enables or disables auto monitoring
func (c *Config) SetAutoMonitor(enabled bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.autoMonitor = enabled
}

// GetCache returns the cache instance
func (c *Config) GetCache() *cache.Cache {
	return c.cache
}

// GetCronScheduler returns the cron scheduler instance
func (c *Config) GetCronScheduler() *cron.Cron {
	return c.cronScheduler
}

// GetVersion returns the version from config
func (c *Config) GetVersion() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return c.fileConfig.Version
	}
	return "1.0.0"
}

// GetHTTPPort returns the HTTP port from config
func (c *Config) GetHTTPPort() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return c.fileConfig.Agent.HTTPPort
	}
	return 8080
}

// GetGRPCPort returns the gRPC port from config
func (c *Config) GetGRPCPort() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return c.fileConfig.Agent.GRPCPort
	}
	return 50051
}

// IsDebugMode returns whether debug mode is enabled in config
func (c *Config) IsDebugMode() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return c.fileConfig.Agent.Debug
	}
	return false
}

// GetFileConfig returns the entire file configuration
func (c *Config) GetFileConfig() *FileConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.fileConfig
}

// GetEncoderConfig returns the encoder configuration
func (c *Config) GetEncoderConfig() *EncoderConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return &c.fileConfig.Encoder
	}
	// 如果配置文件不存在，使用默认配置中的编码器配置
	defaultConfig := getDefaultConfig()
	return &defaultConfig.Encoder
}

// GetScreenConfig returns the screen configuration
func (c *Config) GetScreenConfig() *ScreenConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return &c.fileConfig.Screen
	}
	// 如果配置文件不存在，使用默认配置中的屏幕配置
	defaultConfig := getDefaultConfig()
	return &defaultConfig.Screen
}

// UpdateEncoderConfig updates encoder configuration
func (c *Config) UpdateEncoderConfig(config EncoderConfig) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.fileConfig != nil {
		c.fileConfig.Encoder = config
	}
}

// GetDefaultCodec returns the default encoder codec
func (c *Config) GetDefaultCodec() string {
	encoderConfig := c.GetEncoderConfig()
	return encoderConfig.DefaultCodec
}

// GetFrameRate returns the configured frame rate
func (c *Config) GetFrameRate() int {
	encoderConfig := c.GetEncoderConfig()
	return encoderConfig.FrameRate
}

// GetJPEGQuality returns the JPEG quality setting
func (c *Config) GetJPEGQuality() int {
	encoderConfig := c.GetEncoderConfig()
	return encoderConfig.JPEGQuality
}

// IsCodecEnabled checks if a codec is enabled
func (c *Config) IsCodecEnabled(codec string) bool {
	encoderConfig := c.GetEncoderConfig()
	for _, enabledCodec := range encoderConfig.EnabledCodecs {
		if enabledCodec == codec {
			return true
		}
	}
	return false
}

// GetCodecPriority returns the codec priority list
func (c *Config) GetCodecPriority() []string {
	encoderConfig := c.GetEncoderConfig()
	return encoderConfig.CodecPriority
}

// GetBestAvailableCodec returns the best available codec based on priority
// This method can be used to implement automatic codec selection
func (c *Config) GetBestAvailableCodec() string {
	priority := c.GetCodecPriority()
	enabledCodecs := c.GetEncoderConfig().EnabledCodecs

	// Create a map for quick lookup of enabled codecs
	enabledMap := make(map[string]bool)
	for _, codec := range enabledCodecs {
		enabledMap[codec] = true
	}

	// Return the first codec in priority list that is also enabled
	for _, codec := range priority {
		if enabledMap[codec] {
			return codec
		}
	}

	// Fallback to default codec if none in priority list are enabled
	return c.GetDefaultCodec()
}

// GetCaptureMethod returns the screen capture method
func (c *Config) GetCaptureMethod() string {
	screenConfig := c.GetScreenConfig()
	return screenConfig.CaptureMethod
}

// GetDebugConfig returns the debug configuration
func (c *Config) GetDebugConfig() *DebugConfig {
	encoderConfig := c.GetEncoderConfig()
	return &encoderConfig.Debug
}

// GetDebugSavePath returns the debug save path
func (c *Config) GetDebugSavePath() string {
	debugConfig := c.GetDebugConfig()
	return debugConfig.SavePath
}

// GetSystemConfig returns the system configuration
func (c *Config) GetSystemConfig() *SystemConfig {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if c.fileConfig != nil {
		return &c.fileConfig.System
	}
	// 如果配置文件不存在，使用默认配置中的系统配置
	defaultConfig := getDefaultConfig()
	return &defaultConfig.System
}

// IsRebootEnabled returns whether reboot is enabled
func (c *Config) IsRebootEnabled() bool {
	systemConfig := c.GetSystemConfig()
	return systemConfig.RebootEnabled
}

// GetRebootDelay returns the reboot delay in seconds
func (c *Config) GetRebootDelay() int {
	systemConfig := c.GetSystemConfig()
	return systemConfig.RebootDelay
}

// IsShutdownEnabled returns whether shutdown is enabled
func (c *Config) IsShutdownEnabled() bool {
	systemConfig := c.GetSystemConfig()
	return systemConfig.ShutdownEnabled
}

// IsCommandsEnabled returns whether commands execution is enabled
func (c *Config) IsCommandsEnabled() bool {
	systemConfig := c.GetSystemConfig()
	return systemConfig.CommandsEnabled
}

// Shutdown gracefully shuts down the configuration
func (c *Config) Shutdown() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.cronScheduler != nil {
		c.cronScheduler.Stop()
	}

	log.Info("Configuration shutdown completed")
}

// getServerConfig attempts to determine the server configuration
// This is a placeholder for the original logic that might involve
// remote configuration discovery
func (c *Config) getServerConfig(localIP string) string {
	// TODO: Implement server discovery logic based on local IP
	// For now, return a default or empty string

	// This could involve:
	// - Checking local configuration files
	// - DNS-based service discovery
	// - Multicast discovery
	// - Default server addresses based on network segments

	log.WithField("local_ip", localIP).Debug("Attempting to discover server configuration")

	// Return empty string to indicate no server found
	// The caller should handle this case appropriately
	return ""
}

// Global configuration instance (for backward compatibility)
var globalConfig *Config
var globalConfigOnce sync.Once

// GetGlobalConfig returns the global configuration instance
func GetGlobalConfig() *Config {
	globalConfigOnce.Do(func() {
		globalConfig = NewConfig()
	})
	return globalConfig
}

// Legacy global variables for backward compatibility
var (
	GlobalCron  *cron.Cron
	GlobalCache *cache.Cache
)

// InitGlobalConfig initializes the global configuration (for backward compatibility)
func InitGlobalConfig(serverURL string) error {
	config := GetGlobalConfig()
	err := config.Initialize(serverURL)
	if err != nil {
		return err
	}

	// Set global variables for backward compatibility
	GlobalCron = config.GetCronScheduler()
	GlobalCache = config.GetCache()

	return nil
}

// GetDebugSaveVideoDuration returns the video save duration in seconds
func (c *Config) GetDebugSaveVideoDuration() int {
	debugConfig := c.GetDebugConfig()
	return debugConfig.SaveVideoDuration
}

// IsDebugVideoSaveEnabled returns whether video saving is enabled
func (c *Config) IsDebugVideoSaveEnabled() bool {
	return c.GetDebugSaveVideoDuration() > 0
}
