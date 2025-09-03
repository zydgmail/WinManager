package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// VideoSaver 用于保存调试视频
type VideoSaver struct {
	savePath      string
	duration      time.Duration // 保存时长
	isRecording   bool
	startTime     time.Time
	videoFile     *os.File
	currentSession string
	mutex         sync.Mutex
}

// NewVideoSaver 创建新的视频保存器
func NewVideoSaver(savePath string, durationSeconds int) *VideoSaver {
	return &VideoSaver{
		savePath:    savePath,
		duration:    time.Duration(durationSeconds) * time.Second,
		isRecording: false,
	}
}

// StartRecording 开始录制视频
func (vs *VideoSaver) StartRecording() error {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	if vs.duration <= 0 {
		return nil // 不需要录制
	}

	if vs.isRecording {
		return nil // 已在录制中
	}

	// 确保目录存在
	if err := os.MkdirAll(vs.savePath, 0755); err != nil {
		return fmt.Errorf("创建调试目录失败: %w", err)
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	vs.currentSession = fmt.Sprintf("video_%s.h264", timestamp)
	filePath := filepath.Join(vs.savePath, vs.currentSession)

	// 创建文件
	var err error
	vs.videoFile, err = os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建视频文件失败: %w", err)
	}

	vs.isRecording = true
	vs.startTime = time.Now()

	log.Infof("开始录制视频: %s (时长: %.0f秒)", vs.currentSession, vs.duration.Seconds())

	// 启动定时器，到时间后自动停止录制
	go func() {
		time.Sleep(vs.duration)
		vs.StopRecording()
	}()

	return nil
}

// StopRecording 停止录制视频
func (vs *VideoSaver) StopRecording() error {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	if !vs.isRecording {
		return nil
	}

	vs.isRecording = false

	if vs.videoFile != nil {
		vs.videoFile.Close()
		vs.videoFile = nil
	}

	duration := time.Since(vs.startTime)
	log.Infof("录制完成: %s (实际时长: %.1f秒)", vs.currentSession, duration.Seconds())

	// 转换为MP4格式
	go vs.convertToMP4()

	return nil
}

// WriteFrame 写入H.264帧数据
func (vs *VideoSaver) WriteFrame(data []byte) error {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	if !vs.isRecording || vs.videoFile == nil || data == nil {
		return nil
	}

	// 写入H.264数据
	if _, err := vs.videoFile.Write(data); err != nil {
		return fmt.Errorf("写入视频数据失败: %w", err)
	}

	return nil
}

// convertToMP4 将H.264文件转换为MP4格式
func (vs *VideoSaver) convertToMP4() {
	if vs.currentSession == "" {
		return
	}

	h264Path := filepath.Join(vs.savePath, vs.currentSession)
	mp4Path := filepath.Join(vs.savePath, vs.currentSession[:len(vs.currentSession)-5]+".mp4")

	// 使用FFmpeg转换（如果可用）
	// 这里只是记录日志，实际转换需要系统安装FFmpeg
	log.Infof("H.264文件已保存: %s", h264Path)
	log.Infof("可使用以下命令转换为MP4: ffmpeg -i %s -c copy %s", h264Path, mp4Path)
}
// IsRecording 检查是否正在录制
func (vs *VideoSaver) IsRecording() bool {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()
	return vs.isRecording
}

// GetStats 获取统计信息
func (vs *VideoSaver) GetStats() map[string]interface{} {
	vs.mutex.Lock()
	defer vs.mutex.Unlock()

	stats := map[string]interface{}{
		"save_path":      vs.savePath,
		"duration":       vs.duration.Seconds(),
		"is_recording":   vs.isRecording,
		"current_session": vs.currentSession,
	}

	if vs.isRecording {
		stats["recording_time"] = time.Since(vs.startTime).Seconds()
	}

	return stats
}

