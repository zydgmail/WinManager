package encoders

import (
	"fmt"
	"image"
	"sync"

	"winmanager-agent/internal/config"
	"winmanager-agent/internal/logger"
)

// encoderFactory is a function type for creating encoder instances
type encoderFactory func(size image.Point, frameRate int) (Encoder, error)

// registeredEncoders stores all registered encoder factories
// It's implemented this way to support conditional compilation of each encoder
var registeredEncoders = make(map[VideoCodec]encoderFactory)
var encoderMutex sync.RWMutex

// EncoderService implements the Service interface
type EncoderService struct{}

// NewEncoderService creates a new encoder service instance
func NewEncoderService() Service {
	return &EncoderService{}
}

// NewEncoder creates an instance of an encoder for the selected codec
func (s *EncoderService) NewEncoder(codec VideoCodec, size image.Point, frameRate int) (Encoder, error) {
	encoderMutex.RLock()
	factory, found := registeredEncoders[codec]
	encoderMutex.RUnlock()

	if !found {
		return nil, fmt.Errorf("codec %s (%d) not supported", GetCodecName(codec), codec)
	}

	logger.WithFields(logger.Fields{
		"codec":     GetCodecName(codec),
		"size":      fmt.Sprintf("%dx%d", size.X, size.Y),
		"frameRate": frameRate,
	}).Debug("Creating encoder")

	encoder, err := factory(size, frameRate)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s encoder: %w", GetCodecName(codec), err)
	}

	return encoder, nil
}

// NewEncoderWithConfig creates an instance of an encoder with configuration support
func (s *EncoderService) NewEncoderWithConfig(codec VideoCodec, size image.Point, frameRate int) (Encoder, error) {
	cfg := config.GetGlobalConfig()
	encoderConfig := cfg.GetEncoderConfig()

	switch codec {
	case H264Codec:
		// Check if H264 encoder with options is available (requires h264enc build tag)
		if factory, found := registeredEncoders[H264Codec]; found {
			// Try to create with configuration if the encoder supports it
			opts := H264Options{
				Preset:  encoderConfig.H264Preset,
				Tune:    encoderConfig.H264Tune,
				Profile: encoderConfig.H264Profile,
				Bitrate: encoderConfig.H264Bitrate,
			}

			// Use reflection or type assertion to check if the encoder supports options
			// For now, fall back to default factory
			logger.WithFields(logger.Fields{
				"codec": GetCodecName(codec),
				"opts":  opts,
			}).Debug("Creating H264 encoder with default factory (options not yet supported)")
			return factory(size, frameRate)
		}
		return nil, fmt.Errorf("H264 codec not available (requires h264enc build tag)")

	case JPEGCodec:
		opts := JPEGOptions{
			Quality: encoderConfig.JPEGQuality,
		}
		return newJPEGEncoderWithOptions(size, frameRate, opts)
	default:
		// 对于其他编码器，先尝试使用配置创建，如果失败则使用默认工厂函数
		logger.WithFields(logger.Fields{
			"codec": GetCodecName(codec),
		}).Debug("Using default encoder factory (config not supported)")
		return s.NewEncoder(codec, size, frameRate)
	}
}

// Supports returns whether the codec is supported
func (s *EncoderService) Supports(codec VideoCodec) bool {
	encoderMutex.RLock()
	defer encoderMutex.RUnlock()
	_, found := registeredEncoders[codec]
	return found
}

// GetSupportedCodecs returns a list of all supported codecs
func (s *EncoderService) GetSupportedCodecs() []VideoCodec {
	encoderMutex.RLock()
	defer encoderMutex.RUnlock()

	codecs := make([]VideoCodec, 0, len(registeredEncoders))
	for codec := range registeredEncoders {
		codecs = append(codecs, codec)
	}
	return codecs
}

// RegisterEncoder registers an encoder factory for a specific codec
// This function is called by individual encoder implementations in their init() functions
func RegisterEncoder(codec VideoCodec, factory encoderFactory) {
	encoderMutex.Lock()
	defer encoderMutex.Unlock()

	if factory == nil {
		logger.WithField("codec", GetCodecName(codec)).Error("Cannot register nil encoder factory")
		return
	}

	if _, exists := registeredEncoders[codec]; exists {
		logger.WithField("codec", GetCodecName(codec)).Warn("Encoder factory already registered, overwriting")
	}

	registeredEncoders[codec] = factory
	logger.WithField("codec", GetCodecName(codec)).Debug("Encoder factory registered")
}

// UnregisterEncoder removes an encoder factory (mainly for testing)
func UnregisterEncoder(codec VideoCodec) {
	encoderMutex.Lock()
	defer encoderMutex.Unlock()
	delete(registeredEncoders, codec)
	logger.WithField("codec", GetCodecName(codec)).Debug("Encoder factory unregistered")
}

// GetRegisteredEncoders returns a copy of all registered encoder codecs (for debugging)
func GetRegisteredEncoders() []VideoCodec {
	encoderMutex.RLock()
	defer encoderMutex.RUnlock()

	codecs := make([]VideoCodec, 0, len(registeredEncoders))
	for codec := range registeredEncoders {
		codecs = append(codecs, codec)
	}
	return codecs
}
