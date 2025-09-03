package screen

import (
	"fmt"
	"image"
	"image/draw"
	"sync"
	"time"

	"github.com/go-vgo/robotgo"
	log "github.com/sirupsen/logrus"
)

// RobotGoGrabber implements ScreenGrabber using robotgo library
type RobotGoGrabber struct {
	screen    Screen
	running   bool
	mutex     sync.RWMutex
	stopChan  chan struct{}
	frameChan chan *image.RGBA
	stats     CaptureStats
}

// NewRobotGoGrabber creates a new RobotGo screen grabber
func NewRobotGoGrabber(screen Screen) (ScreenGrabber, error) {
	log.WithFields(log.Fields{
		"screen_index": screen.Index,
		"bounds":       fmt.Sprintf("%+v", screen.Bounds),
		"method":       "robotgo",
	}).Debug("Creating RobotGo screen grabber")

	return &RobotGoGrabber{
		screen:    screen,
		stopChan:  make(chan struct{}),
		frameChan: make(chan *image.RGBA, 3), // Buffer up to 3 frames
	}, nil
}

// Start begins continuous screen capture
func (g *RobotGoGrabber) Start() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.running {
		return fmt.Errorf("grabber is already running")
	}

	log.WithField("screen_index", g.screen.Index).Debug("Starting RobotGo screen grabber")

	g.running = true
	g.stats = CaptureStats{} // Reset stats

	// Start capture goroutine
	go g.captureLoop()

	return nil
}

// Frame returns the latest captured frame
func (g *RobotGoGrabber) Frame() (*image.RGBA, error) {
	if !g.IsRunning() {
		// If not running, capture a single frame
		return g.captureFrame()
	}

	// Try to get the latest frame from the buffer
	select {
	case frame := <-g.frameChan:
		return frame, nil
	case <-time.After(time.Second):
		return nil, fmt.Errorf("timeout waiting for frame")
	}
}

// Stop stops the continuous screen capture
func (g *RobotGoGrabber) Stop() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.running {
		return nil
	}

	log.WithField("screen_index", g.screen.Index).Debug("Stopping RobotGo screen grabber")

	close(g.stopChan)
	g.running = false

	// Drain the frame channel
	for {
		select {
		case <-g.frameChan:
		default:
			goto done
		}
	}
done:

	log.WithFields(log.Fields{
		"screen_index":    g.screen.Index,
		"frames_captured": g.stats.FramesCaptured,
		"frames_dropped":  g.stats.FramesDropped,
		"error_count":     g.stats.ErrorCount,
	}).Debug("RobotGo screen grabber stopped")

	return nil
}

// Screen returns the screen being captured
func (g *RobotGoGrabber) Screen() *Screen {
	return &g.screen
}

// IsRunning returns whether the grabber is currently running
func (g *RobotGoGrabber) IsRunning() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.running
}

// GetStats returns capture statistics
func (g *RobotGoGrabber) GetStats() CaptureStats {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.stats
}

// captureLoop runs the continuous capture loop
func (g *RobotGoGrabber) captureLoop() {
	ticker := time.NewTicker(time.Second / 15) // 降低到15 FPS，提高稳定性
	defer ticker.Stop()

	for {
		select {
		case <-g.stopChan:
			return
		case <-ticker.C:
			startTime := time.Now()
			frame, err := g.captureFrame()
			captureTime := time.Since(startTime)

			g.mutex.Lock()
			if err != nil {
				g.stats.ErrorCount++
				log.WithError(err).Warn("Failed to capture frame")
			} else if frame != nil {
				g.stats.FramesCaptured++
				g.stats.LastFrameTime = time.Now()
				g.stats.AverageFrameTime = (g.stats.AverageFrameTime + captureTime) / 2

				// 改进的帧发送逻辑：清空旧帧，确保最新帧
				select {
				case g.frameChan <- frame:
					// 成功发送新帧
				default:
					// 通道满了，清空旧帧并发送新帧
					select {
					case <-g.frameChan:
						// 丢弃一个旧帧
						g.stats.FramesDropped++
					default:
					}

					// 尝试再次发送新帧
					select {
					case g.frameChan <- frame:
						// 成功发送
					default:
						// 仍然无法发送，记录丢帧
						g.stats.FramesDropped++
					}
				}
			}
			g.mutex.Unlock()
		}
	}
}

// captureFrame captures a single frame using robotgo
func (g *RobotGoGrabber) captureFrame() (*image.RGBA, error) {
	var bitmap robotgo.CBitmap

	bounds := g.screen.Bounds
	if bounds.Dx() > 0 && bounds.Dy() > 0 {
		// Capture specific area
		bitmap = robotgo.CaptureScreen(bounds.Min.X, bounds.Min.Y, bounds.Dx(), bounds.Dy())
	} else {
		// Capture full screen
		bitmap = robotgo.CaptureScreen()
	}

	// Check if bitmap is nil before converting
	if bitmap == nil {
		return nil, fmt.Errorf("robotgo.CaptureScreen returned nil bitmap")
	}

	// Convert bitmap to image.Image
	img := robotgo.ToImage(bitmap)
	if img == nil {
		return nil, fmt.Errorf("failed to capture screen using robotgo")
	}

	// 释放bitmap资源 - 这是关键！防止CGO内存泄漏
	robotgo.FreeBitmap(bitmap)

	// 改进的RGBA转换，确保像素格式正确
	var rgba *image.RGBA
	if rgbaImg, ok := img.(*image.RGBA); ok {
		// 验证RGBA图像的完整性
		imgBounds := rgbaImg.Bounds()
		if imgBounds.Dx() <= 0 || imgBounds.Dy() <= 0 {
			return nil, fmt.Errorf("invalid RGBA image dimensions: %dx%d", imgBounds.Dx(), imgBounds.Dy())
		}

		// 创建新的RGBA图像以避免共享内存问题
		rgba = image.NewRGBA(imgBounds)
		copy(rgba.Pix, rgbaImg.Pix)
	} else {
		// 更安全的像素转换
		bounds := img.Bounds()
		if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			return nil, fmt.Errorf("invalid image dimensions: %dx%d", bounds.Dx(), bounds.Dy())
		}

		rgba = image.NewRGBA(bounds)

		// 使用更高效的像素复制方法，避免逐像素转换的内存分配
		switch srcImg := img.(type) {
		case *image.NRGBA:
			// 直接复制NRGBA到RGBA
			copy(rgba.Pix, srcImg.Pix)
		case *image.RGBA:
			// 直接复制RGBA到RGBA
			copy(rgba.Pix, srcImg.Pix)
		default:
			// 对于其他格式，使用draw包进行高效转换，避免逐像素分配
			// 这比逐像素调用At()和Set()方法快得多，且不会产生大量内存分配
			src := img
			dst := rgba
			draw.Draw(dst, dst.Bounds(), src, bounds.Min, draw.Src)
		}
	}

	// 验证最终图像的完整性
	finalBounds := rgba.Bounds()
	if finalBounds.Dx() <= 0 || finalBounds.Dy() <= 0 {
		return nil, fmt.Errorf("final RGBA image has invalid dimensions: %dx%d", finalBounds.Dx(), finalBounds.Dy())
	}

	// 检查像素数据是否有效
	if len(rgba.Pix) == 0 {
		return nil, fmt.Errorf("RGBA image has no pixel data")
	}

	return rgba, nil
}

// CaptureScreenToRGBA captures the screen and returns an RGBA image
func CaptureScreenToRGBA() (*image.RGBA, error) {
	service := NewScreenService()
	screen, err := service.PrimaryScreen()
	if err != nil {
		return nil, err
	}

	grabber, err := NewRobotGoGrabber(*screen)
	if err != nil {
		return nil, err
	}

	return grabber.(*RobotGoGrabber).captureFrame()
}

// CaptureRegionToRGBA captures a specific region and returns an RGBA image
func CaptureRegionToRGBA(x, y, width, height int) (*image.RGBA, error) {
	screen := Screen{
		Index:  0,
		Bounds: image.Rect(x, y, x+width, y+height),
	}

	grabber, err := NewRobotGoGrabber(screen)
	if err != nil {
		return nil, err
	}

	return grabber.(*RobotGoGrabber).captureFrame()
}
