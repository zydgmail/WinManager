//go:build jpegturbo
// +build jpegturbo

package encoders

import (
	"bytes"
	"fmt"
	"image"

	jpegturbo "github.com/pixiv/go-libjpeg/jpeg"
	"winmanager-agent/internal/logger"
)

// JPEGTurboEncoder implements high-performance JPEG encoding using libjpeg-turbo
type JPEGTurboEncoder struct {
	size    image.Point
	quality int
}

// JPEGTurboOptions contains configuration for JPEG Turbo encoding
type JPEGTurboOptions struct {
	Quality int // JPEG quality (1-100)
}

// DefaultJPEGTurboOptions returns default JPEG Turbo encoding options
func DefaultJPEGTurboOptions() JPEGTurboOptions {
	return JPEGTurboOptions{
		Quality: 80,
	}
}

// newJPEGTurboEncoder creates a new JPEG Turbo encoder instance
func newJPEGTurboEncoder(size image.Point, frameRate int) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	opts := DefaultJPEGTurboOptions()
	
	logger.WithFields(logger.Fields{
		"size":    fmt.Sprintf("%dx%d", size.X, size.Y),
		"quality": opts.Quality,
		"encoder": "jpeg-turbo",
	}).Debug("Creating JPEG Turbo encoder")

	return &JPEGTurboEncoder{
		size:    size,
		quality: opts.Quality,
	}, nil
}

// newJPEGTurboEncoderWithOptions creates a new JPEG Turbo encoder with custom options
func newJPEGTurboEncoderWithOptions(size image.Point, frameRate int, opts JPEGTurboOptions) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if opts.Quality < 1 || opts.Quality > 100 {
		return nil, fmt.Errorf("invalid quality: %d (must be 1-100)", opts.Quality)
	}

	logger.WithFields(logger.Fields{
		"size":    fmt.Sprintf("%dx%d", size.X, size.Y),
		"quality": opts.Quality,
		"encoder": "jpeg-turbo",
	}).Debug("Creating JPEG Turbo encoder with custom options")

	return &JPEGTurboEncoder{
		size:    size,
		quality: opts.Quality,
	}, nil
}

// Encode encodes an RGBA image to JPEG bytes using libjpeg-turbo
func (e *JPEGTurboEncoder) Encode(frame *image.RGBA) ([]byte, error) {
	if frame == nil {
		return nil, fmt.Errorf("frame is nil")
	}

	// Check if frame size matches expected size
	frameBounds := frame.Bounds()
	frameSize := image.Point{X: frameBounds.Dx(), Y: frameBounds.Dy()}
	
	if frameSize != e.size {
		logger.WithFields(logger.Fields{
			"expected": fmt.Sprintf("%dx%d", e.size.X, e.size.Y),
			"actual":   fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		}).Warn("Frame size mismatch")
	}

	var buf bytes.Buffer
	
	// Create JPEG Turbo encoder options
	opts := &jpegturbo.EncoderOptions{
		Quality: e.quality,
	}
	
	// Encode to JPEG using libjpeg-turbo
	err := jpegturbo.Encode(&buf, frame, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG with turbo: %w", err)
	}

	encoded := buf.Bytes()
	
	logger.WithFields(logger.Fields{
		"input_size":  fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		"output_size": len(encoded),
		"quality":     e.quality,
		"encoder":     "jpeg-turbo",
	}).Debug("JPEG Turbo encoding completed")

	return encoded, nil
}

// VideoSize returns the expected video/image size
func (e *JPEGTurboEncoder) VideoSize() (image.Point, error) {
	return e.size, nil
}

// GetCodec returns the codec type
func (e *JPEGTurboEncoder) GetCodec() VideoCodec {
	return JPEGTurboCodec
}

// Close closes the encoder (no-op for JPEG Turbo)
func (e *JPEGTurboEncoder) Close() error {
	logger.Debug("Closing JPEG Turbo encoder")
	return nil
}

// SetQuality updates the JPEG quality setting
func (e *JPEGTurboEncoder) SetQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return fmt.Errorf("invalid quality: %d (must be 1-100)", quality)
	}
	e.quality = quality
	logger.WithFields(logger.Fields{
		"quality": quality,
		"encoder": "jpeg-turbo",
	}).Debug("JPEG Turbo quality updated")
	return nil
}

// GetQuality returns the current JPEG quality setting
func (e *JPEGTurboEncoder) GetQuality() int {
	return e.quality
}

// EncodeJpegTurbo is a utility function for direct JPEG Turbo encoding
func EncodeJpegTurbo(src image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	opts := &jpegturbo.EncoderOptions{Quality: quality}
	err := jpegturbo.Encode(&buf, src, opts)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// init registers the JPEG Turbo encoder factory
func init() {
	RegisterEncoder(JPEGTurboCodec, newJPEGTurboEncoder)
	logger.Debug("JPEG Turbo encoder registered")
}
