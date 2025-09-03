//go:build vp8enc
// +build vp8enc

package encoders

import (
	"bytes"
	"fmt"
	"image"
	"unsafe"

	"winmanager-agent/internal/logger"
)

/*
#cgo pkg-config: vpx
#include <stdlib.h>
#include <string.h>
#include <vpx/vpx_encoder.h>
#include <vpx/vp8cx.h>

void rgba_to_yuv(uint8_t *destination, uint8_t *rgba, size_t width, size_t height) {
	size_t image_size = width * height;
	size_t upos = image_size;
	size_t vpos = upos + upos / 4;
	size_t i = 0;

	for( size_t line = 0; line < height; ++line ) {
		if( !(line % 2) ) {
			for( size_t x = 0; x < width; x += 2 ) {
				uint8_t r = rgba[4 * i];
				uint8_t g = rgba[4 * i + 1];
				uint8_t b = rgba[4 * i + 2];

				destination[i++] = ((66*r + 129*g + 25*b) >> 8) + 16;

				destination[upos++] = ((-38*r + -74*g + 112*b) >> 8) + 128;
				destination[vpos++] = ((112*r + -94*g + -18*b) >> 8) + 128;

				r = rgba[4 * i];
				g = rgba[4 * i + 1];
				b = rgba[4 * i + 2];

				destination[i++] = ((66*r + 129*g + 25*b) >> 8) + 16;
			}
		} else {
			for( size_t x = 0; x < width; x += 1 ) {
					uint8_t r = rgba[4 * i];
					uint8_t g = rgba[4 * i + 1];
					uint8_t b = rgba[4 * i + 2];

					destination[i++] = ((66*r + 129*g + 25*b) >> 8) + 16;
			}
		}
	}
}

int vpx_img_plane_width(const vpx_image_t *img, int plane) {
  if (plane > 0 && img->x_chroma_shift > 0)
    return (img->d_w + 1) >> img->x_chroma_shift;
  else
    return img->d_w;
}

int vpx_img_plane_height(const vpx_image_t *img, int plane) {
  if (plane > 0 && img->y_chroma_shift > 0)
    return (img->d_h + 1) >> img->y_chroma_shift;
  else
    return img->d_h;
}

int vpx_img_read(vpx_image_t *img, void *bs) {
  int plane;
  for (plane = 0; plane < 3; ++plane) {
    unsigned char *buf = img->planes[plane];
    const int stride = img->stride[plane];
    const int w = vpx_img_plane_width(img, plane) *
                  ((img->fmt & VPX_IMG_FMT_HIGHBITDEPTH) ? 2 : 1);
    const int h = vpx_img_plane_height(img, plane);
    int y;
    for (y = 0; y < h; ++y) {
      memcpy(buf, bs, w);
      buf += stride;
      bs += w;
    }
  }
  return 1;
}

int32_t encode_frame(vpx_codec_ctx_t *ctx, vpx_image_t *img, int32_t framec, int32_t flags,
										 void *rgba, void *yuv_buf, int32_t w, int32_t h, void **encoded_frame) {
	rgba_to_yuv(yuv_buf, rgba, w, h);
	vpx_img_read(img, yuv_buf);
	if (vpx_codec_encode(ctx, img, (vpx_codec_pts_t)framec, 1, flags, VPX_DL_REALTIME) != 0) {
		return 0;
	}
	const vpx_codec_cx_pkt_t *pkt = NULL;
	vpx_codec_iter_t it = NULL;
	while ((pkt = vpx_codec_get_cx_data(ctx, &it)) != NULL) {
		if (pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
			*encoded_frame = pkt->data.frame.buf;
			return pkt->data.frame.sz;
		}
	}
	*encoded_frame = (void *)0xDEADBEEF;
	return 0;
}

vpx_codec_err_t codec_enc_config_default(vpx_codec_enc_cfg_t *cfg) {
	return vpx_codec_enc_config_default(vpx_codec_vp8_cx(), cfg, 0);
}

vpx_codec_err_t codec_enc_init(vpx_codec_ctx_t *codec, vpx_codec_enc_cfg_t *cfg) {
	return vpx_codec_enc_init(codec, vpx_codec_vp8_cx(), cfg, 0);
}

*/
import "C"

const keyFrameInterval = 15

// VP8Encoder implements VP8 video encoding using libvpx
type VP8Encoder struct {
	buffer     *bytes.Buffer
	realSize   image.Point
	codecCtx   C.vpx_codec_ctx_t
	vpxImage   C.vpx_image_t
	yuvBuffer  []byte
	frameCount uint
	frameRate  int
}

// VP8Options contains configuration for VP8 encoding
type VP8Options struct {
	Bitrate          uint // Target bitrate in kbps
	KeyFrameInterval uint // Interval between key frames
	ErrorResilient   bool // Enable error resilient mode
}

// DefaultVP8Options returns default VP8 encoding options
func DefaultVP8Options() VP8Options {
	return VP8Options{
		Bitrate:          8192, // 8 Mbps
		KeyFrameInterval: keyFrameInterval,
		ErrorResilient:   true,
	}
}

// newVP8Encoder creates a new VP8 encoder instance
func newVP8Encoder(size image.Point, frameRate int) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if frameRate <= 0 {
		frameRate = 20 // Default frame rate
	}

	buffer := bytes.NewBuffer(make([]byte, 0))
	opts := DefaultVP8Options()

	logger.WithFields(logger.Fields{
		"size":       fmt.Sprintf("%dx%d", size.X, size.Y),
		"frame_rate": frameRate,
		"bitrate":    opts.Bitrate,
		"codec":      "vp8",
	}).Debug("Creating VP8 encoder")

	var cfg C.vpx_codec_enc_cfg_t
	if C.codec_enc_config_default(&cfg) != 0 {
		return nil, fmt.Errorf("failed to initialize default VP8 encoder config")
	}

	cfg.g_w = C.uint(size.X)
	cfg.g_h = C.uint(size.Y)
	cfg.g_timebase.num = 1
	cfg.g_timebase.den = C.int(frameRate)
	cfg.rc_target_bitrate = C.uint(opts.Bitrate)
	if opts.ErrorResilient {
		cfg.g_error_resilient = 1
	}

	var vpxCodecCtx C.vpx_codec_ctx_t
	if C.codec_enc_init(&vpxCodecCtx, &cfg) != 0 {
		return nil, fmt.Errorf("failed to initialize VP8 encoder context")
	}

	var vpxImage C.vpx_image_t
	if C.vpx_img_alloc(&vpxImage, C.VPX_IMG_FMT_I420, C.uint(size.X), C.uint(size.Y), 0) == nil {
		return nil, fmt.Errorf("failed to allocate VP8 image buffer")
	}

	return &VP8Encoder{
		buffer:     buffer,
		realSize:   size,
		codecCtx:   vpxCodecCtx,
		vpxImage:   vpxImage,
		yuvBuffer:  make([]byte, size.X*size.Y*2),
		frameCount: 0,
		frameRate:  frameRate,
	}, nil
}

// newVP8EncoderWithOptions creates a new VP8 encoder with custom options
func newVP8EncoderWithOptions(size image.Point, frameRate int, opts VP8Options) (Encoder, error) {
	if size.X <= 0 || size.Y <= 0 {
		return nil, fmt.Errorf("invalid size: %dx%d", size.X, size.Y)
	}

	if frameRate <= 0 {
		frameRate = 20 // Default frame rate
	}

	buffer := bytes.NewBuffer(make([]byte, 0))

	logger.WithFields(logger.Fields{
		"size":       fmt.Sprintf("%dx%d", size.X, size.Y),
		"frame_rate": frameRate,
		"bitrate":    opts.Bitrate,
		"codec":      "vp8",
	}).Debug("Creating VP8 encoder with custom options")

	var cfg C.vpx_codec_enc_cfg_t
	if C.codec_enc_config_default(&cfg) != 0 {
		return nil, fmt.Errorf("failed to initialize default VP8 encoder config")
	}

	cfg.g_w = C.uint(size.X)
	cfg.g_h = C.uint(size.Y)
	cfg.g_timebase.num = 1
	cfg.g_timebase.den = C.int(frameRate)
	cfg.rc_target_bitrate = C.uint(opts.Bitrate)
	if opts.ErrorResilient {
		cfg.g_error_resilient = 1
	}

	var vpxCodecCtx C.vpx_codec_ctx_t
	if C.codec_enc_init(&vpxCodecCtx, &cfg) != 0 {
		return nil, fmt.Errorf("failed to initialize VP8 encoder context")
	}

	var vpxImage C.vpx_image_t
	if C.vpx_img_alloc(&vpxImage, C.VPX_IMG_FMT_I420, C.uint(size.X), C.uint(size.Y), 0) == nil {
		return nil, fmt.Errorf("failed to allocate VP8 image buffer")
	}

	return &VP8Encoder{
		buffer:     buffer,
		realSize:   size,
		codecCtx:   vpxCodecCtx,
		vpxImage:   vpxImage,
		yuvBuffer:  make([]byte, size.X*size.Y*2),
		frameCount: 0,
		frameRate:  frameRate,
	}, nil
}

// Encode encodes a frame into VP8 payload
func (e *VP8Encoder) Encode(frame *image.RGBA) ([]byte, error) {
	if frame == nil {
		return nil, fmt.Errorf("frame is nil")
	}

	// Check if frame size matches expected size
	frameBounds := frame.Bounds()
	frameSize := image.Point{X: frameBounds.Dx(), Y: frameBounds.Dy()}

	if frameSize != e.realSize {
		logger.WithFields(logger.Fields{
			"expected": fmt.Sprintf("%dx%d", e.realSize.X, e.realSize.Y),
			"actual":   fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
		}).Warn("Frame size mismatch for VP8 encoding")
	}

	encodedData := unsafe.Pointer(nil)
	var flags C.int
	if e.frameCount%keyFrameInterval == 0 {
		flags |= C.VPX_EFLAG_FORCE_KF
	}

	frameSize32 := C.encode_frame(
		&e.codecCtx,
		&e.vpxImage,
		C.int(e.frameCount),
		flags,
		unsafe.Pointer(&frame.Pix[0]),
		unsafe.Pointer(&e.yuvBuffer[0]),
		C.int(e.realSize.X),
		C.int(e.realSize.Y),
		&encodedData,
	)

	e.frameCount++

	if int(frameSize32) > 0 {
		encoded := C.GoBytes(encodedData, frameSize32)

		logger.WithFields(logger.Fields{
			"input_size":  fmt.Sprintf("%dx%d", frameSize.X, frameSize.Y),
			"output_size": len(encoded),
			"frame_count": e.frameCount,
			"codec":       "vp8",
		}).Debug("VP8 encoding completed")

		return encoded, nil
	}

	return nil, nil
}

// VideoSize returns the expected video size
func (e *VP8Encoder) VideoSize() (image.Point, error) {
	return e.realSize, nil
}

// GetCodec returns the codec type
func (e *VP8Encoder) GetCodec() VideoCodec {
	return VP8Codec
}

// Close frees VP8 encoder resources
func (e *VP8Encoder) Close() error {
	logger.Debug("Closing VP8 encoder")
	C.vpx_img_free(&e.vpxImage)
	C.vpx_codec_destroy(&e.codecCtx)
	e.frameCount = 0
	return nil
}

// GetFrameRate returns the current frame rate
func (e *VP8Encoder) GetFrameRate() int {
	return e.frameRate
}

// GetFrameCount returns the current frame count
func (e *VP8Encoder) GetFrameCount() uint {
	return e.frameCount
}

// init registers the VP8 encoder factory
func init() {
	RegisterEncoder(VP8Codec, newVP8Encoder)
	logger.Debug("VP8 encoder registered")
}
