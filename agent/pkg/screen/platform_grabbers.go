//go:build windows
// +build windows

package screen

import (
	"fmt"
	"image"

	log "github.com/sirupsen/logrus"
)

// 删除重复的NewDXGIGrabber声明，使用dxgi_grabber.go中的实现

// WGCGrabber implements ScreenGrabber using Windows Graphics Capture API
type WGCGrabber struct {
	screen Screen
	running bool
}

// NewWGCGrabber creates a new Windows Graphics Capture screen grabber
func NewWGCGrabber(screen Screen) (ScreenGrabber, error) {
	log.WithFields(log.Fields{
		"screen_index": screen.Index,
		"method":       "wgc",
	}).Debug("Creating WGC screen grabber")

	// TODO: Implement Windows Graphics Capture API
	return nil, fmt.Errorf("Windows Graphics Capture not implemented yet")
}



func (g *WGCGrabber) Start() error {
	return fmt.Errorf("WGC screen capture not implemented")
}

func (g *WGCGrabber) Frame() (*image.RGBA, error) {
	return nil, fmt.Errorf("WGC screen capture not implemented")
}

func (g *WGCGrabber) Stop() error {
	return fmt.Errorf("WGC screen capture not implemented")
}

func (g *WGCGrabber) Screen() *Screen {
	return &g.screen
}

func (g *WGCGrabber) IsRunning() bool {
	return g.running
}
