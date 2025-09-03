//go:build nvenc
// +build nvenc

package encoders

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"unsafe"

	"winmanager-agent/internal/logger"

	"github.com/asticode/go-astiav"
)

/*
 * NVENC H.264 encoder using FFmpeg wrapper
 */

/*
void rgba2yuv(void *destination, void *source, int width, int height, int stride) {
	const int image_size = width * height;
	unsigned char *rgba = source;
  unsigned char *dst_y = destination;
  unsigned char *dst_u = destination + image_size;
  unsigned char *dst_v = destination + image_size + image_size/4;
	// Y plane
	for( int y=0; y<height; ++y ) {
    for( int x=0; x<width; ++x ) {
      const int i = y*(width+stride) + x;
			*dst_y++ = ( ( 66*rgba[4*i] + 129*rgba[4*i+1] + 25*rgba[4*i+2] ) >> 8 ) + 16;
		}
  }
  // U plane
  for( int y=0; y<height; y+=2 ) {
    for( int x=0; x<width; x+=2 ) {
      const int i = y*(width+stride) + x;
			*dst_u++ = ( ( -38*rgba[4*i] + -74*rgba[4*i+1] + 112*rgba[4*i+2] ) >> 8 ) + 128;
		}
  }
  // V plane
  for( int y=0; y<height; y+=2 ) {
    for( int x=0; x<width; x+=2 ) {
      const int i = y*(width+stride) + x;
			*dst_v++ = ( ( 112*rgba[4*i] + -94*rgba[4*i+1] + -18*rgba[4*i+2] ) >> 8 ) + 128;
		}
  }
}
*/
import "C"

// NVENCEncoder implements NVIDIA hardware-accelerated H.264 encoding
type NVENCEncoder struct {
	buffer    *bytes.Buffer
	realSize  image.Point
	c         *astiav.CodecContext
	frame     *astiav.Frame
	pkt       *astiav.Packet
	count     int64
	frameRate int
}

// NVENCOptions contains configuration for NVENC encoding
type NVENCOptions struct {
	Bitrate int64  // Target bitrate in bits per second
	Profile string // H.264 profile (baseline, main, high)
	Preset  string // NVENC preset (default, slow, medium, fast, hp, hq, bd, ll, llhq, llhp, lossless, losslesshp)
}

// DefaultNVENCOptions returns default NVENC encoding options
func DefaultNVENCOptions() NVENCOptions {
	return NVENCOptions{
		Bitrate: 50000000, // 50 Mbps
		Profile: "high",
		Preset:  "hq",
	}
}

// newNVENCEncoder creates a new NVENC encoder instance
func newNVENCEncoder(size image.Point, frameRate int) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if frameRate <= 0 {
		frameRate = 20 // Default frame rate
	}

	// Fixed size for NVENC (can be made configurable)
	realSize := image.Point{X: 1920, Y: 1080}
	opts := DefaultNVENCOptions()

	logger.WithFields(logger.Fields{
		"requested_size": fmt.Sprintf("%dx%d", size.X, size.Y),
		"actual_size":    fmt.Sprintf("%dx%d", realSize.X, realSize.Y),
		"frame_rate":     frameRate,
		"bitrate":        opts.Bitrate,
		"profile":        opts.Profile,
		"preset":         opts.Preset,
		"codec":          "nvenc",
	}).Debug("Creating NVENC encoder")

	codec := astiav.FindEncoderByName("h264_nvenc")
	if codec == nil {
		return nil, errors.New("h264_nvenc encoder not supported (NVIDIA GPU or drivers not available)")
	}

	c := astiav.AllocCodecContext(codec)
	c.SetBitRate(opts.Bitrate)
	c.SetWidth(realSize.X)
	c.SetHeight(realSize.Y)
	c.SetTimeBase(astiav.NewRational(1, frameRate))
	c.SetPixelFormat(astiav.PixelFormatYuv420P)

	d := astiav.Dictionary{}
	d.Set("profile", opts.Profile, 0)
	d.Set("preset", opts.Preset, 0)

	err := c.Open(codec, &d)
	if err != nil {
		logger.WithError(err).Error("Failed to open NVENC codec")
		return nil, fmt.Errorf("failed to open NVENC codec: %w", err)
	}

	frame := astiav.AllocFrame()
	frame.SetPixelFormat(astiav.PixelFormatYuv420P)
	frame.SetWidth(realSize.X)
	frame.SetHeight(realSize.Y)
	frame.AllocBuffer(20)

	return &NVENCEncoder{
		buffer:    bytes.NewBuffer(make([]byte, 0)),
		realSize:  realSize,
		c:         c,
		frame:     frame,
		pkt:       astiav.AllocPacket(),
		frameRate: frameRate,
	}, nil
}

// newNVENCEncoderWithOptions creates a new NVENC encoder with custom options
func newNVENCEncoderWithOptions(size image.Point, frameRate int, opts NVENCOptions) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if frameRate <= 0 {
		frameRate = 20 // Default frame rate
	}

	// Fixed size for NVENC (can be made configurable)
	realSize := image.Point{X: 1920, Y: 1080}

	logger.WithFields(logger.Fields{
		"requested_size": fmt.Sprintf("%dx%d", size.X, size.Y),
		"actual_size":    fmt.Sprintf("%dx%d", realSize.X, realSize.Y),
		"frame_rate":     frameRate,
		"bitrate":        opts.Bitrate,
		"profile":        opts.Profile,
		"preset":         opts.Preset,
		"codec":          "nvenc",
	}).Debug("Creating NVENC encoder with custom options")

	codec := astiav.FindEncoderByName("h264_nvenc")
	if codec == nil {
		return nil, errors.New("h264_nvenc encoder not supported (NVIDIA GPU or drivers not available)")
	}

	c := astiav.AllocCodecContext(codec)
	c.SetBitRate(opts.Bitrate)
	c.SetWidth(realSize.X)
	c.SetHeight(realSize.Y)
	c.SetTimeBase(astiav.NewRational(1, frameRate))
	c.SetPixelFormat(astiav.PixelFormatYuv420P)

	d := astiav.Dictionary{}
	d.Set("profile", opts.Profile, 0)
	d.Set("preset", opts.Preset, 0)

	err := c.Open(codec, &d)
	if err != nil {
		logger.WithError(err).Error("Failed to open NVENC codec")
		return nil, fmt.Errorf("failed to open NVENC codec: %w", err)
	}

	frame := astiav.AllocFrame()
	frame.SetPixelFormat(astiav.PixelFormatYuv420P)
	frame.SetWidth(realSize.X)
	frame.SetHeight(realSize.Y)
	frame.AllocBuffer(20)

	return &NVENCEncoder{
		buffer:    bytes.NewBuffer(make([]byte, 0)),
		realSize:  realSize,
		c:         c,
		frame:     frame,
		pkt:       astiav.AllocPacket(),
		frameRate: frameRate,
	}, nil
}

// Encode encodes a frame using NVENC
func (e *NVENCEncoder) Encode(f *image.RGBA) ([]byte, error) {
	if f == nil {
		return nil, fmt.Errorf("frame is nil")
	}

	// Check if frame size matches expected size
	frameBounds := f.Bounds()
	frameSize := image.Point{X: frameBounds.Dx(), Y: frameBounds.Dy()}

	if frameSize != e.realSize {
		logger.WithFields(logger.Fields{
			"expected": fmt.Sprintf("%dx%d", e.realSize.X, e.realSize.Y),
			"actual":   fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		}).Warn("Frame size mismatch for NVENC encoding")
	}

	err := e.frame.SetDataBytes(RgbaToYuv(f))
	if err != nil {
		logger.WithError(err).Error("Failed to set frame data")
		return nil, fmt.Errorf("failed to set frame data: %w", err)
	}

	err = e.c.SendFrame(e.frame)
	if err != nil {
		return nil, fmt.Errorf("failed to send frame to encoder: %w", err)
	}

	for {
		if err = e.c.ReceivePacket(e.pkt); err != nil {
			break
		}
		e.buffer.Write(e.pkt.Data())
	}

	payload := e.buffer.Bytes()
	e.buffer.Reset()
	e.count++

	logger.WithFields(logger.Fields{
		"input_size":  fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		"output_size": len(payload),
		"frame_count": e.count,
		"codec":       "nvenc",
	}).Debug("NVENC encoding completed")

	return payload, nil
}

// VideoSize returns the expected video size
func (e *NVENCEncoder) VideoSize() (image.Point, error) {
	return e.realSize, nil
}

// GetCodec returns the codec type
func (e *NVENCEncoder) GetCodec() VideoCodec {
	return NVENCCodec
}

// Close frees NVENC encoder resources
func (e *NVENCEncoder) Close() error {
	logger.Debug("Closing NVENC encoder")
	if e.pkt != nil {
		e.pkt.Free()
	}
	if e.frame != nil {
		e.frame.Free()
	}
	if e.c != nil {
		e.c.Free()
	}
	e.count = 0
	return nil
}

// GetFrameRate returns the current frame rate
func (e *NVENCEncoder) GetFrameRate() int {
	return e.frameRate
}

// GetFrameCount returns the current frame count
func (e *NVENCEncoder) GetFrameCount() int64 {
	return e.count
}

// RgbaToYuv converts RGBA image to YUV420P format
func RgbaToYuv(rgba *image.RGBA) []byte {
	w := rgba.Rect.Max.X
	h := rgba.Rect.Max.Y
	size := int(float32(w*h) * 1.5)
	stride := rgba.Stride - w*4
	yuv := make([]byte, size, size)
	C.rgba2yuv(unsafe.Pointer(&yuv[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(w), C.int(h), C.int(stride))
	return yuv
}

// init registers the NVENC encoder factory
func init() {
	RegisterEncoder(NVENCCodec, newNVENCEncoder)
	logger.Debug("NVENC encoder registered")
}
