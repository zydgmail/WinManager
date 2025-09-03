//go:build !windows
// +build !windows

package screen

import (
	"fmt"
	"image"
)

// Fallback implementations for non-Windows platforms

// NewDXGIGrabber creates a new DXGI screen grabber (not available on non-Windows)
func NewDXGIGrabber(screen Screen) (ScreenGrabber, error) {
	return nil, fmt.Errorf("DXGI screen capture is only available on Windows")
}

// NewWGCGrabber creates a new WGC screen grabber (not available on non-Windows)
func NewWGCGrabber(screen Screen) (ScreenGrabber, error) {
	return nil, fmt.Errorf("Windows Graphics Capture is only available on Windows")
}

// Dummy types for interface compliance
type DXGIGrabber struct{}
type WGCGrabber struct{}

func (g *DXGIGrabber) Start() error                { return fmt.Errorf("not supported") }
func (g *DXGIGrabber) Frame() (*image.RGBA, error) { return nil, fmt.Errorf("not supported") }
func (g *DXGIGrabber) Stop() error                 { return fmt.Errorf("not supported") }
func (g *DXGIGrabber) Screen() *Screen              { return nil }
func (g *DXGIGrabber) IsRunning() bool              { return false }

func (g *WGCGrabber) Start() error                { return fmt.Errorf("not supported") }
func (g *WGCGrabber) Frame() (*image.RGBA, error) { return nil, fmt.Errorf("not supported") }
func (g *WGCGrabber) Stop() error                 { return fmt.Errorf("not supported") }
func (g *WGCGrabber) Screen() *Screen              { return nil }
func (g *WGCGrabber) IsRunning() bool              { return false }
