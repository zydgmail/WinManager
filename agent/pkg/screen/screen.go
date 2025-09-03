package screen

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"sync"
	"time"

	"github.com/chai2010/webp"
	"github.com/go-vgo/robotgo"
	log "github.com/sirupsen/logrus"
)

// ImageFormat represents supported image formats
type ImageFormat string

const (
	FormatPNG  ImageFormat = "png"
	FormatJPEG ImageFormat = "jpeg"
	FormatWebP ImageFormat = "webp"
)

// ScreenshotOptions contains options for screenshot capture
type ScreenshotOptions struct {
	Format  ImageFormat
	Quality int // JPEG quality (1-100)
	X       int // Capture area X coordinate
	Y       int // Capture area Y coordinate
	Width   int // Capture area width (0 = full screen)
	Height  int // Capture area height (0 = full screen)
}

// DefaultScreenshotOptions returns default screenshot options
func DefaultScreenshotOptions() ScreenshotOptions {
	return ScreenshotOptions{
		Format:  FormatWebP,
		Quality: 85,
		X:       0,
		Y:       0,
		Width:   0,
		Height:  0,
	}
}

// ScreenService implements the Service interface
type ScreenService struct {
	mutex           sync.RWMutex
	cachedScreens   []Screen
	lastScreenCheck time.Time
	screenCacheTTL  time.Duration
}

// NewScreenService creates a new screen service instance
func NewScreenService() Service {
	return &ScreenService{
		screenCacheTTL: time.Second * 5, // Cache screens for 5 seconds
	}
}

// CreateScreenGrabber creates a screen grabber for the specified screen and method
func (s *ScreenService) CreateScreenGrabber(screen Screen, method CaptureMethod) (ScreenGrabber, error) {
	if method == CaptureMethodAuto {
		method = s.selectBestMethod()
	}

	log.WithFields(log.Fields{
		"screen_index": screen.Index,
		"method":       GetCaptureMethodName(method),
		"bounds":       fmt.Sprintf("%+v", screen.Bounds),
	}).Debug("Creating screen grabber")

	switch method {
	case CaptureMethodDXGI:
		return NewDXGIGrabber(screen)
	case CaptureMethodWGC:
		return NewWGCGrabber(screen)
	case CaptureMethodRobotGo:
		// RobotGo内部在Windows使用GDI，Linux使用X11，macOS使用Quartz
		log.Info("使用RobotGo捕获方法（跨平台兼容）")
		return NewRobotGoGrabber(screen)
	default:
		// 默认使用RobotGo，兼容性最好
		log.Infof("未知捕获方法 %s，使用默认RobotGo方法", GetCaptureMethodName(method))
		return NewRobotGoGrabber(screen)
	}
}

// Screens returns all available screens
func (s *ScreenService) Screens() ([]Screen, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if we need to refresh the screen cache
	if time.Since(s.lastScreenCheck) > s.screenCacheTTL {
		screens, err := s.detectScreens()
		if err != nil {
			return nil, err
		}
		s.cachedScreens = screens
		s.lastScreenCheck = time.Now()
	}

	// Return a copy of cached screens
	result := make([]Screen, len(s.cachedScreens))
	copy(result, s.cachedScreens)
	return result, nil
}

// PrimaryScreen returns the primary screen
func (s *ScreenService) PrimaryScreen() (*Screen, error) {
	screens, err := s.Screens()
	if err != nil {
		return nil, err
	}

	for _, screen := range screens {
		if screen.Primary {
			return &screen, nil
		}
	}

	// If no primary screen found, return the first one
	if len(screens) > 0 {
		return &screens[0], nil
	}

	return nil, fmt.Errorf("no screens available")
}

// SupportedMethods returns all supported capture methods
func (s *ScreenService) SupportedMethods() []CaptureMethod {
	methods := []CaptureMethod{CaptureMethodRobotGo}

	// Check for Windows-specific methods
	if s.supportsDXGI() {
		methods = append(methods, CaptureMethodDXGI)
	}
	if s.supportsWGC() {
		methods = append(methods, CaptureMethodWGC)
	}

	return methods
}

// SupportsMethod checks if a specific capture method is supported
func (s *ScreenService) SupportsMethod(method CaptureMethod) bool {
	switch method {
	case CaptureMethodAuto, CaptureMethodRobotGo:
		return true
	case CaptureMethodDXGI:
		return s.supportsDXGI()
	case CaptureMethodWGC:
		return s.supportsWGC()
	default:
		return false
	}
}

// detectScreens detects all available screens
func (s *ScreenService) detectScreens() ([]Screen, error) {
	// For now, use robotgo to get basic screen info
	// This can be enhanced with platform-specific implementations
	width, height := robotgo.GetScreenSize()

	screen := Screen{
		Index:       0,
		Bounds:      image.Rect(0, 0, width, height),
		Primary:     true,
		Name:        "Primary Display",
		ScaleFactor: 1.0,
	}

	log.WithFields(log.Fields{
		"screen_count": 1,
		"primary_size": fmt.Sprintf("%dx%d", width, height),
	}).Debug("Detected screens")

	return []Screen{screen}, nil
}

// selectBestMethod selects the best available capture method - 优先RobotGo（兼容性最好）
func (s *ScreenService) selectBestMethod() CaptureMethod {
	// 优先使用RobotGo，兼容性最好
	return CaptureMethodRobotGo

	// 备选方案（暂时注释）
	// if s.supportsDXGI() {
	//     return CaptureMethodDXGI
	// }
	// if s.supportsWGC() {
	//     return CaptureMethodWGC
	// }
	// if s.supportsGDI() {
	//     return CaptureMethodGDI
	// }
	// return CaptureMethodRobotGo
}

// Platform-specific method support checks (placeholders for now)
func (s *ScreenService) supportsDXGI() bool {
	// 默认支持DXGI（Windows平台）
	return true
}

func (s *ScreenService) supportsWGC() bool {
	// TODO: Implement Windows Graphics Capture support check
	return false
}

// Legacy functions for backward compatibility

// CaptureScreenshot captures a screenshot and returns it as bytes
func CaptureScreenshot() ([]byte, error) {
	return CaptureScreenshotWithOptions(DefaultScreenshotOptions())
}

// CaptureScreenshotWithOptions captures a screenshot with specified options
func CaptureScreenshotWithOptions(opts ScreenshotOptions) ([]byte, error) {
	log.Infof("开始截图捕获: Format=%s, Quality=%d, X=%d, Y=%d, Width=%d, Height=%d",
		opts.Format, opts.Quality, opts.X, opts.Y, opts.Width, opts.Height)

	// 首先尝试使用更安全的方法
	if opts.Width == 0 && opts.Height == 0 {
		return captureScreenshotSafe(opts)
	}

	// Capture screenshot using robotgo
	var bitmap robotgo.CBitmap

	log.Infof("调用robotgo截图")

	// 添加错误恢复机制
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("robotgo截图过程中发生panic: %v", r)
		}
	}()

	if opts.Width > 0 && opts.Height > 0 {
		// Capture specific area
		log.Infof("区域截图: X=%d, Y=%d, Width=%d, Height=%d",
			opts.X, opts.Y, opts.Width, opts.Height)

		// 验证区域参数
		if opts.X < 0 || opts.Y < 0 || opts.Width <= 0 || opts.Height <= 0 {
			log.Errorf("无效的截图区域参数: X=%d, Y=%d, Width=%d, Height=%d",
				opts.X, opts.Y, opts.Width, opts.Height)
			return nil, fmt.Errorf("invalid screenshot region parameters")
		}

		bitmap = robotgo.CaptureScreen(opts.X, opts.Y, opts.Width, opts.Height)
	} else {
		// Capture full screen
		log.Infof("全屏截图")
		bitmap = robotgo.CaptureScreen()
	}

	log.Infof("robotgo截图完成，开始转换为图像")

	// Convert bitmap to image.Image
	img := robotgo.ToImage(bitmap)

	// 释放bitmap资源 - 防止CGO内存泄漏
	robotgo.FreeBitmap(bitmap)

	if img == nil {
		log.Errorf("robotgo转换图像失败")
		return nil, fmt.Errorf("failed to capture screenshot")
	}

	// 检查图像是否有效
	bounds := img.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		log.Errorf("截图图像无效: 尺寸=%dx%d", bounds.Dx(), bounds.Dy())
		return nil, fmt.Errorf("invalid screenshot dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}

	log.Infof("图像有效: 尺寸=%dx%d", bounds.Dx(), bounds.Dy())

	log.Infof("图像转换成功，开始编码: Format=%s", opts.Format)

	// Encode image to bytes
	var buf bytes.Buffer
	var err error
	switch opts.Format {
	case FormatPNG:
		err = png.Encode(&buf, img)
	case FormatJPEG:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: opts.Quality})
	case FormatWebP:
		// WebP编码，使用指定的质量设置
		err = webp.Encode(&buf, img, &webp.Options{
			Lossless: false,
			Quality:  float32(opts.Quality),
			Exact:    false,
		})
	default:
		log.Errorf("ERROR-[%s]-[%s]-[%s]-不支持的图像格式: %s",
			time.Now().Format("2006-01-02 15:04:05"),
			"screen.go",
			"CaptureScreenshotWithOptions",
			opts.Format)
		return nil, fmt.Errorf("unsupported image format: %s", opts.Format)
	}

	if err != nil {
		log.Errorf("图像编码失败: %v", err)
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	imageData := buf.Bytes()
	log.Infof("截图编码成功: 大小=%d bytes, 格式=%s", len(imageData), opts.Format)

	return imageData, nil
}

// GetScreenSize returns the screen dimensions
func GetScreenSize() (int, int) {
	width, height := robotgo.GetScreenSize()
	return width, height
}

// GetScreenInfo returns detailed screen information
func GetScreenInfo() map[string]interface{} {
	width, height := GetScreenSize()

	return map[string]interface{}{
		"width":  width,
		"height": height,
		"dpi":    96, // Default DPI, could be enhanced to get actual DPI
	}
}

// CaptureRegion captures a specific region of the screen
func CaptureRegion(x, y, width, height int) ([]byte, error) {
	opts := DefaultScreenshotOptions()
	opts.X = x
	opts.Y = y
	opts.Width = width
	opts.Height = height

	return CaptureScreenshotWithOptions(opts)
}

// captureScreenshotSafe 使用更安全的方法进行全屏截图
func captureScreenshotSafe(opts ScreenshotOptions) ([]byte, error) {
	log.Infof("使用安全模式进行全屏截图")

	// 获取屏幕尺寸
	width, height := robotgo.GetScreenSize()
	log.Infof("屏幕尺寸: %dx%d", width, height)

	if width <= 0 || height <= 0 {
		log.Errorf("无效的屏幕尺寸: %dx%d", width, height)
		return nil, fmt.Errorf("invalid screen size: %dx%d", width, height)
	}

	// 使用 robotgo.CaptureImg 方法
	img, err := robotgo.CaptureImg()
	if err != nil {
		log.Errorf("robotgo.CaptureImg 调用失败: %v", err)
		return nil, fmt.Errorf("failed to capture screenshot using CaptureImg: %w", err)
	}
	if img == nil {
		log.Errorf("robotgo.CaptureImg 返回空图像")
		return nil, fmt.Errorf("failed to capture screenshot using CaptureImg")
	}

	log.Infof("安全模式截图成功，开始编码: Format=%s", opts.Format)

	// Encode image to bytes
	var buf bytes.Buffer
	switch opts.Format {
	case FormatPNG:
		err = png.Encode(&buf, img)
	case FormatJPEG:
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: opts.Quality})
	case FormatWebP:
		// WebP编码，使用指定的质量设置
		err = webp.Encode(&buf, img, &webp.Options{
			Lossless: false,
			Quality:  float32(opts.Quality),
			Exact:    false,
		})
	default:
		log.Errorf("不支持的图像格式: %s", opts.Format)
		return nil, fmt.Errorf("unsupported image format: %s", opts.Format)
	}

	if err != nil {
		log.Errorf("图像编码失败: %v", err)
		return nil, fmt.Errorf("failed to encode image: %w", err)
	}

	imageData := buf.Bytes()
	log.Infof("安全模式截图编码成功: 大小=%d bytes, 格式=%s", len(imageData), opts.Format)

	return imageData, nil
}

// CaptureWindow captures a specific window (placeholder for future implementation)
func CaptureWindow(windowTitle string) ([]byte, error) {
	// TODO: Implement window-specific capture
	// This would involve finding the window by title and capturing its area
	log.WithField("window_title", windowTitle).Debug("Window capture not implemented yet")
	return CaptureScreenshot()
}
