package screen

import (
	"image"
	"time"
)

// ScreenGrabber provides continuous screen capture functionality
type ScreenGrabber interface {
	Start() error
	Frame() (*image.RGBA, error)
	Stop() error
	Screen() *Screen
	IsRunning() bool
}

// Screen represents a display screen with its properties
type Screen struct {
	Index       int             // Screen index (0-based)
	Bounds      image.Rectangle // Screen bounds
	Primary     bool            // Whether this is the primary screen
	Name        string          // Screen name/identifier
	ScaleFactor float64         // DPI scale factor
}

// Service provides screen capture services
type Service interface {
	CreateScreenGrabber(screen Screen, method CaptureMethod) (ScreenGrabber, error)
	Screens() ([]Screen, error)
	PrimaryScreen() (*Screen, error)
	SupportedMethods() []CaptureMethod
	SupportsMethod(method CaptureMethod) bool
}

// CaptureMethod represents different screen capture methods
type CaptureMethod int

const (
	// CaptureMethodAuto automatically selects the best available method
	CaptureMethodAuto CaptureMethod = iota
	// CaptureMethodDXGI uses DirectX Graphics Infrastructure (faster, Windows 8+)
	CaptureMethodDXGI
	// CaptureMethodWGC uses Windows Graphics Capture API (fastest, Windows 10 1903+)
	CaptureMethodWGC
	// CaptureMethodRobotGo uses robotgo library (cross-platform)
	// Note: RobotGo internally uses GDI on Windows, X11 on Linux, and Quartz on macOS
	CaptureMethodRobotGo
)

// CaptureMethodNames provides human-readable names for capture methods
var CaptureMethodNames = map[CaptureMethod]string{
	CaptureMethodAuto:    "auto",
	CaptureMethodDXGI:    "dxgi",
	CaptureMethodWGC:     "wgc",
	CaptureMethodRobotGo: "robotgo",
}

// GetCaptureMethodName returns the human-readable name for a capture method
func GetCaptureMethodName(method CaptureMethod) string {
	if name, exists := CaptureMethodNames[method]; exists {
		return name
	}
	return "unknown"
}

// ParseCaptureMethodName returns the capture method for a given name
func ParseCaptureMethodName(name string) CaptureMethod {
	for method, methodName := range CaptureMethodNames {
		if methodName == name {
			return method
		}
	}
	return CaptureMethodAuto
}

// CaptureOptions contains options for screen capture
type CaptureOptions struct {
	Method        CaptureMethod // Capture method to use
	FrameRate     int           // Target frame rate for continuous capture
	BufferSize    int           // Frame buffer size
	Timeout       time.Duration // Timeout for capture operations
	IncludeCursor bool          // Whether to include cursor in capture
}

// DefaultCaptureOptions returns default capture options
func DefaultCaptureOptions() CaptureOptions {
	return CaptureOptions{
		Method:        CaptureMethodRobotGo,
		FrameRate:     20,
		BufferSize:    3,
		Timeout:       time.Second * 5,
		IncludeCursor: true,
	}
}

// ScreenInfo contains detailed information about a screen
type ScreenInfo struct {
	Screen      Screen
	Width       int     // Screen width in pixels
	Height      int     // Screen height in pixels
	DPI         int     // Dots per inch
	ScaleFactor float64 // DPI scale factor
	ColorDepth  int     // Color depth in bits
	RefreshRate int     // Refresh rate in Hz
}

// CaptureStats contains statistics about screen capture performance
type CaptureStats struct {
	FramesCaptured   uint64        // Total frames captured
	FramesDropped    uint64        // Total frames dropped
	AverageFrameTime time.Duration // Average time per frame
	LastFrameTime    time.Time     // Timestamp of last captured frame
	ErrorCount       uint64        // Total number of errors
}

// FrameBuffer represents a captured frame with metadata
type FrameBuffer struct {
	Image     *image.RGBA   // The captured image
	Timestamp time.Time     // When the frame was captured
	Screen    *Screen       // Which screen was captured
	Method    CaptureMethod // Which capture method was used
}

// MultiScreenGrabber provides capture from multiple screens simultaneously
type MultiScreenGrabber interface {
	Start() error
	Frames() (map[int]*image.RGBA, error) // Returns frames indexed by screen index
	Stop() error
	Screens() []Screen
	IsRunning() bool
	GetStats() map[int]CaptureStats // Returns stats indexed by screen index
}

// RegionGrabber provides capture from a specific region of a screen
type RegionGrabber interface {
	Start() error
	Frame() (*image.RGBA, error)
	Stop() error
	Region() image.Rectangle
	Screen() *Screen
	IsRunning() bool
}

// WindowGrabber provides capture from a specific window
type WindowGrabber interface {
	Start() error
	Frame() (*image.RGBA, error)
	Stop() error
	WindowHandle() uintptr // Platform-specific window handle
	WindowTitle() string
	IsRunning() bool
}

// AdvancedService extends the basic Service with additional functionality
type AdvancedService interface {
	Service
	CreateMultiScreenGrabber(screens []Screen, method CaptureMethod) (MultiScreenGrabber, error)
	CreateRegionGrabber(screen Screen, region image.Rectangle, method CaptureMethod) (RegionGrabber, error)
	CreateWindowGrabber(windowHandle uintptr, method CaptureMethod) (WindowGrabber, error)
	GetScreenInfo(screen Screen) (*ScreenInfo, error)
	DetectScreenChanges() ([]Screen, error) // Detect if screens have changed
}
