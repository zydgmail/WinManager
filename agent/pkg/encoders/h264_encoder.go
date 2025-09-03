//go:build h264enc
// +build h264enc

package encoders

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"time"

	"winmanager-agent/internal/logger"

	"github.com/gen2brain/x264-go"
)

// H264Encoder h264 encoder
type H264Encoder struct {
	buffer        *bytes.Buffer
	encoder       *x264.Encoder
	realSize      image.Point
	frameCount    int       // 保留用于日志统计
	lastIDRFrame  int       // 保留用于日志统计
	frameRate     int       // 帧率，用于时间戳计算
	startTime     time.Time // 编码开始时间
	forceKeyFrame bool      // 强制下一帧为关键帧
	// 保存编码器配置，用于重置时保持一致
	encoderOptions H264Options
}

// DefaultH264Options returns default H.264 encoding options
func DefaultH264Options() H264Options {
	return H264Options{
		Preset:  "fast",
		Tune:    "zerolatency",
		Profile: "main",
		Bitrate: 20000000, // 20 Mbps
	}
}

const h264SupportedProfile = "3.1"

func newH264Encoder(size image.Point, frameRate int) (Encoder, error) {
	opts := DefaultH264Options()
	return newH264EncoderWithOptions(size, frameRate, opts)
}

// newH264EncoderWithOptions creates a new H.264 encoder with custom options
func newH264EncoderWithOptions(size image.Point, frameRate int, opts H264Options) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if frameRate <= 0 {
		frameRate = 20 // Default frame rate
	}

	buffer := bytes.NewBuffer(make([]byte, 0))
	realSize, err := findBestSizeForH264Profile(h264SupportedProfile, size)
	if err != nil {
		return nil, err
	}

	logger.WithFields(logger.Fields{
		"requested_size": fmt.Sprintf("%dx%d", size.X, size.Y),
		"actual_size":    fmt.Sprintf("%dx%d", realSize.X, realSize.Y),
		"frame_rate":     frameRate,
		"preset":         opts.Preset,
		"tune":           opts.Tune,
		"profile":        opts.Profile,
		"bitrate":        opts.Bitrate,
		"codec":          "h264",
	}).Debug("Creating H.264 encoder with options")

	// 使用传入的配置参数创建编码器
	x264Opts := x264.Options{
		Width:     realSize.X,
		Height:    realSize.Y,
		FrameRate: frameRate,
		Tune:      opts.Tune,    // 使用配置文件中的tune参数
		Preset:    opts.Preset,  // 使用配置文件中的preset参数
		Profile:   opts.Profile, // 使用配置文件中的profile参数
		LogLevel:  x264.LogWarning,
	}

	logger.Debugf("H264编码器: 使用配置参数 - Tune: %s, Preset: %s, Profile: %s",
		x264Opts.Tune, x264Opts.Preset, x264Opts.Profile)

	encoder, err := x264.NewEncoder(buffer, &x264Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create H.264 encoder: %w", err)
	}

	return &H264Encoder{
		buffer:         buffer,
		encoder:        encoder,
		realSize:       realSize,
		frameCount:     0,
		lastIDRFrame:   -1,
		frameRate:      frameRate,
		startTime:      time.Now(),
		forceKeyFrame:  false,
		encoderOptions: opts, // 保存配置用于重置
	}, nil
}

// Encode encodes a frame into a h264 payload
func (e *H264Encoder) Encode(frame *image.RGBA) ([]byte, error) {
	if frame == nil {
		return nil, fmt.Errorf("input frame is nil")
	}

	// 验证帧尺寸
	frameBounds := frame.Bounds()
	frameSize := image.Point{X: frameBounds.Dx(), Y: frameBounds.Dy()}
	if frameSize.X != e.realSize.X || frameSize.Y != e.realSize.Y {
		return nil, fmt.Errorf("frame size mismatch: expected %dx%d, got %dx%d",
			e.realSize.X, e.realSize.Y, frameSize.X, frameSize.Y)
	}

	// 验证像素数据完整性
	expectedPixelCount := frameSize.X * frameSize.Y * 4 // RGBA = 4 bytes per pixel
	if len(frame.Pix) != expectedPixelCount {
		return nil, fmt.Errorf("invalid pixel data length: expected %d, got %d",
			expectedPixelCount, len(frame.Pix))
	}

	e.frameCount++

	// 清空缓冲区确保干净的编码
	e.buffer.Reset()

	// 优化的关键帧生成策略 - 只在需要时生成
	needKeyFrame := e.frameCount == 1 || e.forceKeyFrame

	if needKeyFrame {
		if e.frameCount == 1 {
			logger.Infof("H264编码器: 生成首帧关键帧 (帧#%d)", e.frameCount)
		} else if e.forceKeyFrame {
			logger.Infof("H264编码器: 处理强制关键帧请求 (帧#%d)", e.frameCount)
			e.forceKeyFrame = false
		}

		// 通过重新创建编码器来强制生成IDR帧（包含SPS+PPS+IDR）
		logger.Debugf("H264编码器: 重新创建编码器以强制生成完整关键帧序列")

		// 保存当前编码器
		oldEncoder := e.encoder

		// 使用保存的配置创建新编码器实例，保持参数一致
		x264Opts := x264.Options{
			Width:     e.realSize.X,
			Height:    e.realSize.Y,
			FrameRate: e.frameRate,
			Tune:      e.encoderOptions.Tune,    // 使用原始配置
			Preset:    e.encoderOptions.Preset,  // 使用原始配置
			Profile:   e.encoderOptions.Profile, // 使用原始配置
			LogLevel:  x264.LogWarning,
		}

		logger.Debugf("H264编码器: 重置参数 - Tune:%s, Preset:%s, Profile:%s",
			x264Opts.Tune, x264Opts.Preset, x264Opts.Profile)

		newEncoder, err := x264.NewEncoder(e.buffer, &x264Opts)
		if err != nil {
			logger.Errorf("H264编码器: 重新创建编码器失败: %v", err)
		} else {
			// 关闭旧编码器
			oldEncoder.Close()
			e.encoder = newEncoder
			logger.Infof("H264编码器: 编码器重新创建成功，将生成SPS+PPS+IDR完整序列")
		}

		// 标记这应该是一个关键帧
		e.lastIDRFrame = e.frameCount
	}

	// 编码帧
	err := e.encoder.Encode(frame)
	if err != nil {
		return nil, fmt.Errorf("encoding failed: %w", err)
	}

	err = e.encoder.Flush()
	if err != nil {
		return nil, fmt.Errorf("flush failed: %w", err)
	}

	payload := e.buffer.Bytes()
	if len(payload) == 0 {
		return nil, fmt.Errorf("encoder produced empty payload")
	}

	// 计算时间戳信息
	currentTime := time.Now()
	elapsedTime := currentTime.Sub(e.startTime)
	expectedFrameTime := time.Duration(e.frameCount-1) * time.Second / time.Duration(e.frameRate)
	timeDrift := elapsedTime - expectedFrameTime

	// 极简日志：只在关键时刻输出
	if e.frameCount == 1 {
		logger.Infof("H264编码器启动: %dx%d@%dfps", frameSize.X, frameSize.Y, e.frameRate)
	} else if e.frameCount%1000 == 0 { // 改为每1000帧才打印一次
		actualFPS := float64(e.frameCount) / elapsedTime.Seconds()
		// 只有时间偏移过大才打印警告
		if math.Abs(timeDrift.Seconds()) > 5.0 {
			logger.Warnf("H264编码性能警告: 帧#%d FPS:%.1f 时间偏移:%.1fs", e.frameCount, actualFPS, timeDrift.Seconds())
		}
	}

	// 检测IDR帧（用于统计，不干预编码过程）
	nalUnits := e.parseNALUnits(payload)
	for _, nal := range nalUnits {
		if nal.Type == 5 { // IDR帧
			if e.lastIDRFrame != -1 {
				framesSinceLastIDR := e.frameCount - e.lastIDRFrame
				logger.Debugf("检测到IDR帧 #%d (距离上次IDR: %d帧, 大小: %d字节)",
					e.frameCount, framesSinceLastIDR, len(payload))
			} else {
				logger.Debugf("检测到首个IDR帧 #%d (大小: %d字节)", e.frameCount, len(payload))
			}
			e.lastIDRFrame = e.frameCount
			break
		}
	}

	return payload, nil
}

// parseNALUnits 简化的NAL单元解析，仅用于日志统计
func (e *H264Encoder) parseNALUnits(data []byte) []NALUnit {
	var nalUnits []NALUnit

	for i := 0; i < len(data)-4; i++ {
		// 查找NAL单元起始码 0x00000001
		if data[i] == 0x00 && data[i+1] == 0x00 && data[i+2] == 0x00 && data[i+3] == 0x01 {
			if i+4 < len(data) {
				nalType := data[i+4] & 0x1F
				nalUnits = append(nalUnits, NALUnit{
					Type:     int(nalType),
					Position: i,
				})
			}
		}
	}

	return nalUnits
}

// NALUnit represents a NAL unit in the H.264 stream
type NALUnit struct {
	Type     int // NAL unit type
	Position int // Position in the data
}

// VideoSize returns the size the other side is expecting
func (e *H264Encoder) VideoSize() (image.Point, error) {
	return e.realSize, nil
}

// GetCodec returns the codec type
func (e *H264Encoder) GetCodec() VideoCodec {
	return H264Codec
}

// Close flushes and closes the inner x264 encoder
func (e *H264Encoder) Close() error {
	return e.encoder.Close()
}

// ForceKeyFrame forces the next frame to be a key frame (IDR)
func (e *H264Encoder) ForceKeyFrame() {
	e.forceKeyFrame = true
	logger.Debugf("H264编码器: 已设置强制关键帧标志")
}

// findBestSizeForH264Profile finds the best match given the size constraint and H264 profile
func findBestSizeForH264Profile(profile string, constraints image.Point) (image.Point, error) {
	profileSizes := map[string][]image.Point{
		"3.1": []image.Point{
			image.Point{1920, 1080},
			image.Point{1280, 720},
			image.Point{720, 576},
			image.Point{720, 480},
		},
	}
	if sizes, exists := profileSizes[profile]; exists {
		minRatioDiff := math.MaxFloat64
		var minRatioSize image.Point
		for _, size := range sizes {
			if size == constraints {
				return size, nil
			}
			lowerRes := size.X < constraints.X && size.Y < constraints.Y
			hRatio := float64(constraints.X) / float64(size.X)
			vRatio := float64(constraints.Y) / float64(size.Y)
			ratioDiff := math.Abs(hRatio - vRatio)
			if lowerRes && (ratioDiff) < 0.0001 {
				return size, nil
			} else if ratioDiff < minRatioDiff {
				minRatioDiff = ratioDiff
				minRatioSize = size
			}
		}
		return minRatioSize, nil
	}
	return image.Point{}, fmt.Errorf("Profile %s not supported", profile)
}

func init() {
	RegisterEncoder(H264Codec, newH264Encoder)
}
