package encoders

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"winmanager-agent/internal/logger"
)

// JPEGEncoder implements JPEG image encoding
type JPEGEncoder struct {
	size    image.Point
	quality int
}



// DefaultJPEGOptions returns default JPEG encoding options
func DefaultJPEGOptions() JPEGOptions {
	return JPEGOptions{
		Quality: 80,
	}
}

// newJPEGEncoder creates a new JPEG encoder instance
func newJPEGEncoder(size image.Point, frameRate int) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	opts := DefaultJPEGOptions()
	
	logger.WithFields(logger.Fields{
		"size":    fmt.Sprintf("%dx%d", size.X, size.Y),
		"quality": opts.Quality,
	}).Debug("Creating JPEG encoder")

	return &JPEGEncoder{
		size:    size,
		quality: opts.Quality,
	}, nil
}

// newJPEGEncoderWithOptions creates a new JPEG encoder with custom options
func newJPEGEncoderWithOptions(size image.Point, frameRate int, opts JPEGOptions) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if opts.Quality < 1 || opts.Quality > 100 {
		return nil, fmt.Errorf("invalid quality: %d (must be 1-100)", opts.Quality)
	}

	logger.WithFields(logger.Fields{
		"size":    fmt.Sprintf("%dx%d", size.X, size.Y),
		"quality": opts.Quality,
	}).Debug("Creating JPEG encoder with custom options")

	return &JPEGEncoder{
		size:    size,
		quality: opts.Quality,
	}, nil
}

// Encode encodes an RGBA image to JPEG bytes
func (e *JPEGEncoder) Encode(frame *image.RGBA) ([]byte, error) {
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
	
	// Encode to JPEG
	err := jpeg.Encode(&buf, frame, &jpeg.Options{Quality: e.quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}

	encoded := buf.Bytes()
	
	logger.WithFields(logger.Fields{
		"input_size":  fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		"output_size": len(encoded),
		"quality":     e.quality,
	}).Debug("JPEG encoding completed")

	return encoded, nil
}

// VideoSize returns the expected video/image size
func (e *JPEGEncoder) VideoSize() (image.Point, error) {
	return e.size, nil
}

// GetCodec returns the codec type
func (e *JPEGEncoder) GetCodec() VideoCodec {
	return JPEGCodec
}

// Close closes the encoder (no-op for JPEG)
func (e *JPEGEncoder) Close() error {
	logger.Debug("Closing JPEG encoder")
	return nil
}

// SetQuality updates the JPEG quality setting
func (e *JPEGEncoder) SetQuality(quality int) error {
	if quality < 1 || quality > 100 {
		return fmt.Errorf("invalid quality: %d (must be 1-100)", quality)
	}
	e.quality = quality
	logger.WithField("quality", quality).Debug("JPEG quality updated")
	return nil
}

// GetQuality returns the current JPEG quality setting
func (e *JPEGEncoder) GetQuality() int {
	return e.quality
}

// init registers the JPEG encoder factory
func init() {
	RegisterEncoder(JPEGCodec, newJPEGEncoder)
}
