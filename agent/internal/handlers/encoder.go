package handlers

import (
	"fmt"
	"image"
	"net/http"
	"strconv"
	"time"

	"winmanager-agent/pkg/encoders"
	"winmanager-agent/pkg/screen"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// EncoderService global instance
var encoderService encoders.Service

// InitEncoderService initializes the global encoder service
func InitEncoderService() {
	encoderService = encoders.NewEncoderService()
	log.Info("Encoder service initialized")
}

// EncoderInfoHandler returns information about available encoders
func EncoderInfoHandler(c *gin.Context) {
	log.Debug("Handling encoder info request")

	if encoderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Encoder service not initialized",
		})
		return
	}

	supportedCodecs := encoderService.GetSupportedCodecs()
	codecInfo := make(map[string]interface{})

	for _, codec := range supportedCodecs {
		codecInfo[encoders.GetCodecName(codec)] = map[string]interface{}{
			"id":        int(codec),
			"name":      encoders.GetCodecName(codec),
			"supported": encoderService.Supports(codec),
		}
	}

	// Get screen service info
	screenService := screen.NewScreenService()
	supportedMethods := screenService.SupportedMethods()
	methodInfo := make(map[string]interface{})

	for _, method := range supportedMethods {
		methodInfo[screen.GetCaptureMethodName(method)] = map[string]interface{}{
			"id":        int(method),
			"name":      screen.GetCaptureMethodName(method),
			"supported": screenService.SupportsMethod(method),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": map[string]interface{}{
			"encoders":        codecInfo,
			"capture_methods": methodInfo,
		},
	})
}

// EncodedScreenshotHandler captures and returns an encoded screenshot
func EncodedScreenshotHandler(c *gin.Context) {
	log.Debug("Handling encoded screenshot request")

	if encoderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Encoder service not initialized",
		})
		return
	}

	// Parse parameters
	codecName := c.DefaultQuery("codec", "jpeg")
	codec := encoders.ParseCodecName(codecName)
	if codec == encoders.NoCodec {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": fmt.Sprintf("Unsupported codec: %s", codecName),
		})
		return
	}

	// Check if codec is supported
	if !encoderService.Supports(codec) {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": fmt.Sprintf("Codec %s is not supported", codecName),
		})
		return
	}

	// Parse quality parameter (currently not used but may be needed for future encoder options)
	_ = 80 // quality placeholder
	if q := c.Query("quality"); q != "" {
		if parsedQuality, err := strconv.Atoi(q); err == nil && parsedQuality >= 1 && parsedQuality <= 100 {
			_ = parsedQuality // quality placeholder
		}
	}

	// Parse capture method
	methodName := c.DefaultQuery("method", "auto")
	method := screen.ParseCaptureMethodName(methodName)

	// Get screen service and primary screen
	screenService := screen.NewScreenService()
	primaryScreen, err := screenService.PrimaryScreen()
	if err != nil {
		log.WithError(err).Error("Failed to get primary screen")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to get screen information",
			"error":   err.Error(),
		})
		return
	}

	// Create screen grabber
	grabber, err := screenService.CreateScreenGrabber(*primaryScreen, method)
	if err != nil {
		log.WithError(err).Error("Failed to create screen grabber")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to create screen grabber",
			"error":   err.Error(),
		})
		return
	}

	// Capture frame
	frame, err := grabber.Frame()
	if err != nil {
		log.WithError(err).Error("Failed to capture frame")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to capture frame",
			"error":   err.Error(),
		})
		return
	}

	// Get frame size
	bounds := frame.Bounds()
	frameSize := image.Point{X: bounds.Dx(), Y: bounds.Dy()}

	// Create encoder
	encoder, err := encoderService.NewEncoderWithConfig(codec, frameSize, 1) // Single frame
	if err != nil {
		log.WithError(err).Error("Failed to create encoder")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to create encoder",
			"error":   err.Error(),
		})
		return
	}
	defer encoder.Close()

	// Encode frame
	startTime := time.Now()
	encodedData, err := encoder.Encode(frame)
	encodeTime := time.Since(startTime)

	if err != nil {
		log.WithError(err).Error("Failed to encode frame")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Failed to encode frame",
			"error":   err.Error(),
		})
		return
	}

	log.WithFields(log.Fields{
		"codec":       codecName,
		"method":      methodName,
		"frame_size":  fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		"output_size": len(encodedData),
		"encode_time": encodeTime,
	}).Debug("Frame encoded successfully")

	// Set appropriate content type
	var contentType string
	switch codec {
	case encoders.JPEGCodec, encoders.JPEGTurboCodec:
		contentType = "image/jpeg"
	case encoders.H264Codec, encoders.NVENCCodec:
		contentType = "video/h264"
	case encoders.VP8Codec:
		contentType = "video/webm"
	default:
		contentType = "application/octet-stream"
	}

	// Add custom headers
	c.Header("X-Codec", codecName)
	c.Header("X-Capture-Method", methodName)
	c.Header("X-Frame-Size", fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y))
	c.Header("X-Encode-Time", encodeTime.String())
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", fmt.Sprintf("%d", len(encodedData)))

	c.Data(http.StatusOK, contentType, encodedData)
}

// StreamingHandler handles video streaming with continuous encoding
func StreamingHandler(c *gin.Context) {
	log.Debug("Handling streaming request")

	if encoderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "Encoder service not initialized",
		})
		return
	}

	// Parse parameters
	codecName := c.DefaultQuery("codec", "h264")
	codec := encoders.ParseCodecName(codecName)
	if codec == encoders.NoCodec {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": fmt.Sprintf("Unsupported codec: %s", codecName),
		})
		return
	}

	// Only allow video codecs for streaming
	if codec == encoders.JPEGCodec || codec == encoders.JPEGTurboCodec {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "Image codecs are not supported for streaming",
		})
		return
	}

	// Parse frame rate
	frameRate := 20
	if fr := c.Query("framerate"); fr != "" {
		if parsedFR, err := strconv.Atoi(fr); err == nil && parsedFR > 0 && parsedFR <= 60 {
			frameRate = parsedFR
		}
	}

	// Parse capture method
	methodName := c.DefaultQuery("method", "auto")
	_ = screen.ParseCaptureMethodName(methodName) // method placeholder for future use

	// TODO: Implement actual streaming logic
	// This would involve:
	// 1. Creating a screen grabber with continuous capture
	// 2. Creating an encoder for video streaming
	// 3. Setting up WebSocket or HTTP streaming
	// 4. Continuously capturing and encoding frames

	c.JSON(http.StatusNotImplemented, gin.H{
		"code":    -1,
		"message": "Streaming not implemented yet",
		"params": map[string]interface{}{
			"codec":     codecName,
			"framerate": frameRate,
			"method":    methodName,
		},
	})
}
