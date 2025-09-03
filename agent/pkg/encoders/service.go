package encoders

import (
	"image"
	"io"
)

// Service creates encoder instances
type Service interface {
	NewEncoder(codec VideoCodec, size image.Point, frameRate int) (Encoder, error)
	NewEncoderWithConfig(codec VideoCodec, size image.Point, frameRate int) (Encoder, error)
	Supports(codec VideoCodec) bool
	GetSupportedCodecs() []VideoCodec
}

// Encoder takes an image/frame and encodes it
type Encoder interface {
	io.Closer
	Encode(*image.RGBA) ([]byte, error)
	VideoSize() (image.Point, error)
	GetCodec() VideoCodec
}

// VideoCodec represents different video/image encoding formats
type VideoCodec int

const (
	// NoCodec represents no codec (zero-value)
	NoCodec VideoCodec = iota
	// JPEGCodec for JPEG image encoding
	JPEGCodec
	// JPEGTurboCodec for high-performance JPEG encoding
	JPEGTurboCodec
	// H264Codec for H.264 video encoding
	H264Codec
	// VP8Codec for VP8 video encoding
	VP8Codec
	// NVENCCodec for NVIDIA hardware-accelerated H.264 encoding
	NVENCCodec
)

// CodecNames provides human-readable names for codecs
var CodecNames = map[VideoCodec]string{
	NoCodec:        "none",
	JPEGCodec:      "jpeg",
	JPEGTurboCodec: "jpeg-turbo",
	H264Codec:      "h264",
	VP8Codec:       "vp8",
	NVENCCodec:     "nvenc",
}

// GetCodecName returns the human-readable name for a codec
func GetCodecName(codec VideoCodec) string {
	if name, exists := CodecNames[codec]; exists {
		return name
	}
	return "unknown"
}

// ParseCodecName returns the codec for a given name
func ParseCodecName(name string) VideoCodec {
	for codec, codecName := range CodecNames {
		if codecName == name {
			return codec
		}
	}
	return NoCodec
}

// H264Options contains configuration for H.264 encoding
type H264Options struct {
	Preset  string // H.264 preset (ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow)
	Tune    string // H.264 tune (film, animation, grain, stillimage, psnr, ssim, fastdecode, zerolatency)
	Profile string // H.264 profile (baseline, main, high, high10, high422, high444)
	Bitrate int64  // Target bitrate in bits per second (note: x264-go doesn't support bitrate directly)
}

// JPEGOptions contains configuration for JPEG encoding
type JPEGOptions struct {
	Quality int // JPEG quality (1-100)
}
