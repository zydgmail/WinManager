package screen

import (
	"image"
	"testing"
	"time"
)

func TestScreenService(t *testing.T) {
	service := NewScreenService()

	// Test service creation
	if service == nil {
		t.Fatal("NewScreenService returned nil")
	}

	// Test getting screens
	screens, err := service.Screens()
	if err != nil {
		t.Fatalf("Failed to get screens: %v", err)
	}

	if len(screens) == 0 {
		t.Error("No screens found")
	}

	// Test primary screen
	primaryScreen, err := service.PrimaryScreen()
	if err != nil {
		t.Fatalf("Failed to get primary screen: %v", err)
	}

	if primaryScreen == nil {
		t.Error("Primary screen is nil")
	}

	// Test supported methods
	supportedMethods := service.SupportedMethods()
	if len(supportedMethods) == 0 {
		t.Error("No supported capture methods found")
	}

	// RobotGo should always be supported
	if !service.SupportsMethod(CaptureMethodRobotGo) {
		t.Error("RobotGo capture method should always be supported")
	}

	// Auto method should always be supported
	if !service.SupportsMethod(CaptureMethodAuto) {
		t.Error("Auto capture method should always be supported")
	}
}

func TestCaptureMethodNames(t *testing.T) {
	testCases := []struct {
		method CaptureMethod
		name   string
	}{
		{CaptureMethodAuto, "auto"},
		{CaptureMethodDXGI, "dxgi"},
		{CaptureMethodWGC, "wgc"},
		{CaptureMethodRobotGo, "robotgo"},
	}

	for _, tc := range testCases {
		name := GetCaptureMethodName(tc.method)
		if name != tc.name {
			t.Errorf("GetCaptureMethodName(%d) = %s, want %s", tc.method, name, tc.name)
		}

		method := ParseCaptureMethodName(tc.name)
		if method != tc.method {
			t.Errorf("ParseCaptureMethodName(%s) = %d, want %d", tc.name, method, tc.method)
		}
	}

	// Test unknown method
	unknownName := GetCaptureMethodName(CaptureMethod(999))
	if unknownName != "unknown" {
		t.Errorf("GetCaptureMethodName(999) = %s, want unknown", unknownName)
	}

	unknownMethod := ParseCaptureMethodName("invalid")
	if unknownMethod != CaptureMethodAuto {
		t.Errorf("ParseCaptureMethodName(invalid) = %d, want %d", unknownMethod, CaptureMethodAuto)
	}
}

func TestDefaultCaptureOptions(t *testing.T) {
	opts := DefaultCaptureOptions()

	if opts.Method != CaptureMethodAuto {
		t.Errorf("Default method mismatch: expected %d, got %d", CaptureMethodAuto, opts.Method)
	}

	if opts.FrameRate != 20 {
		t.Errorf("Default frame rate mismatch: expected 20, got %d", opts.FrameRate)
	}

	if opts.BufferSize != 3 {
		t.Errorf("Default buffer size mismatch: expected 3, got %d", opts.BufferSize)
	}

	if opts.Timeout != time.Second*5 {
		t.Errorf("Default timeout mismatch: expected 5s, got %v", opts.Timeout)
	}

	if !opts.IncludeCursor {
		t.Error("Default include cursor should be true")
	}
}

func TestRobotGoGrabber(t *testing.T) {
	// Create a test screen
	screen := Screen{
		Index:   0,
		Bounds:  image.Rect(0, 0, 640, 480),
		Primary: true,
		Name:    "Test Screen",
	}

	grabber, err := NewRobotGoGrabber(screen)
	if err != nil {
		t.Fatalf("Failed to create RobotGo grabber: %v", err)
	}

	// Test initial state
	if grabber.IsRunning() {
		t.Error("Grabber should not be running initially")
	}

	if grabber.Screen().Index != screen.Index {
		t.Error("Screen index mismatch")
	}

	// Test single frame capture
	frame, err := grabber.Frame()
	if err != nil {
		t.Fatalf("Failed to capture frame: %v", err)
	}

	if frame == nil {
		t.Error("Captured frame is nil")
	}

	// Test frame properties
	bounds := frame.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Errorf("Invalid frame size: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestRobotGoGrabberContinuous(t *testing.T) {
	screen := Screen{
		Index:   0,
		Bounds:  image.Rect(0, 0, 320, 240),
		Primary: true,
		Name:    "Test Screen",
	}

	grabber, err := NewRobotGoGrabber(screen)
	if err != nil {
		t.Fatalf("Failed to create RobotGo grabber: %v", err)
	}

	// Start continuous capture
	err = grabber.Start()
	if err != nil {
		t.Fatalf("Failed to start grabber: %v", err)
	}

	if !grabber.IsRunning() {
		t.Error("Grabber should be running after start")
	}

	// Wait a bit for frames to be captured
	time.Sleep(100 * time.Millisecond)

	// Get a frame
	frame, err := grabber.Frame()
	if err != nil {
		t.Fatalf("Failed to get frame from running grabber: %v", err)
	}

	if frame == nil {
		t.Error("Frame from running grabber is nil")
	}

	// Stop the grabber
	err = grabber.Stop()
	if err != nil {
		t.Fatalf("Failed to stop grabber: %v", err)
	}

	if grabber.IsRunning() {
		t.Error("Grabber should not be running after stop")
	}

	// Test double start/stop
	err = grabber.Start()
	if err != nil {
		t.Fatalf("Failed to restart grabber: %v", err)
	}

	err = grabber.Start() // Should return error
	if err == nil {
		t.Error("Expected error for double start")
	}

	err = grabber.Stop()
	if err != nil {
		t.Fatalf("Failed to stop grabber: %v", err)
	}

	err = grabber.Stop() // Should not return error
	if err != nil {
		t.Errorf("Unexpected error for double stop: %v", err)
	}
}

func TestLegacyScreenshotFunctions(t *testing.T) {
	// Test basic screenshot
	data, err := CaptureScreenshot()
	if err != nil {
		t.Fatalf("Failed to capture screenshot: %v", err)
	}

	if len(data) == 0 {
		t.Error("Screenshot data is empty")
	}

	// Test screenshot with options
	opts := DefaultScreenshotOptions()
	opts.Format = FormatPNG
	opts.Width = 100
	opts.Height = 100

	data, err = CaptureScreenshotWithOptions(opts)
	if err != nil {
		t.Fatalf("Failed to capture screenshot with options: %v", err)
	}

	if len(data) == 0 {
		t.Error("Screenshot data with options is empty")
	}

	// Test region capture
	data, err = CaptureRegion(0, 0, 50, 50)
	if err != nil {
		t.Fatalf("Failed to capture region: %v", err)
	}

	if len(data) == 0 {
		t.Error("Region capture data is empty")
	}
}

func TestGetScreenSize(t *testing.T) {
	width, height := GetScreenSize()
	if width <= 0 || height <= 0 {
		t.Errorf("Invalid screen size: %dx%d", width, height)
	}
}

func TestGetScreenInfo(t *testing.T) {
	info := GetScreenInfo()
	if info == nil {
		t.Error("Screen info is nil")
	}

	width, ok := info["width"].(int)
	if !ok || width <= 0 {
		t.Error("Invalid width in screen info")
	}

	height, ok := info["height"].(int)
	if !ok || height <= 0 {
		t.Error("Invalid height in screen info")
	}

	dpi, ok := info["dpi"].(int)
	if !ok || dpi <= 0 {
		t.Error("Invalid DPI in screen info")
	}
}

func TestCaptureScreenToRGBA(t *testing.T) {
	frame, err := CaptureScreenToRGBA()
	if err != nil {
		t.Fatalf("Failed to capture screen to RGBA: %v", err)
	}

	if frame == nil {
		t.Error("RGBA frame is nil")
	}

	bounds := frame.Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		t.Errorf("Invalid RGBA frame size: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestCaptureRegionToRGBA(t *testing.T) {
	frame, err := CaptureRegionToRGBA(0, 0, 100, 100)
	if err != nil {
		t.Fatalf("Failed to capture region to RGBA: %v", err)
	}

	if frame == nil {
		t.Error("RGBA region frame is nil")
	}

	bounds := frame.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Errorf("RGBA region frame size mismatch: expected 100x100, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func BenchmarkScreenCapture(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CaptureScreenshot()
		if err != nil {
			b.Fatalf("Failed to capture screenshot: %v", err)
		}
	}
}

func BenchmarkRGBACapture(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := CaptureScreenToRGBA()
		if err != nil {
			b.Fatalf("Failed to capture RGBA: %v", err)
		}
	}
}
