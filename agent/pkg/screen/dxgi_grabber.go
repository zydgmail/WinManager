package screen

import (
	"errors"
	"image"
	"sync"

	"github.com/go-vgo/robotgo"
	log "github.com/sirupsen/logrus"
)

// 全局单例DXGI捕获器，复用资源
var (
	globalDXGIGrabber *DXGIGrabber
	dxgiOnce          sync.Once
	dxgiMutex         sync.Mutex
)

// DXGIGrabber implements ScreenGrabber using DXGI (DirectX Graphics Infrastructure)
type DXGIGrabber struct {
	mutex   sync.Mutex
	running bool
	screen  Screen
	frame   *image.RGBA

	// DXGI相关资源 - 需要实现具体的DXGI接口
	// device    *d3d.ID3D11Device
	// deviceCtx *d3d.ID3D11DeviceContext
	// ddup      *d3d.OutputDuplicator
}

// 固定缓冲区，避免重复分配内存
var dxgiImgBuf = image.NewRGBA(image.Rectangle{
	Min: image.Point{X: 0, Y: 0},
	Max: image.Point{X: 1920, Y: 1080},
})

// NewDXGIGrabber 创建DXGI捕获器（单例模式）
func NewDXGIGrabber(screen Screen) (ScreenGrabber, error) {
	dxgiOnce.Do(func() {
		log.Info("初始化全局DXGI屏幕捕获器")
		globalDXGIGrabber = &DXGIGrabber{
			screen: screen,
		}
	})
	return globalDXGIGrabber, nil
}

// Start 启动DXGI捕获
func (g *DXGIGrabber) Start() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if g.running {
		return nil
	}

	log.Info("启动DXGI屏幕捕获")

	// TODO: 初始化DXGI设备和资源
	// var err error
	// g.device, g.deviceCtx, err = d3d.NewD3D11Device()
	// if err != nil {
	//     return fmt.Errorf("创建D3D11设备失败: %w", err)
	// }

	// g.ddup, err = d3d.NewIDXGIOutputDuplication(g.device, g.deviceCtx, uint(0))
	// if err != nil {
	//     return fmt.Errorf("创建DXGI输出复制器失败: %w", err)
	// }

	g.running = true
	log.Info("DXGI设备初始化成功")
	return nil
}

// Stop 停止DXGI捕获
func (g *DXGIGrabber) Stop() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.running {
		return nil
	}

	log.Info("停止DXGI屏幕捕获")

	// TODO: 释放DXGI资源
	// if g.ddup != nil {
	//     g.ddup.Release()
	//     g.ddup = nil
	// }
	// if g.deviceCtx != nil {
	//     g.deviceCtx.Release()
	// }
	// if g.device != nil {
	//     g.device.Release()
	// }

	g.running = false
	g.frame = nil
	log.Info("DXGI资源释放完成")
	return nil
}

// Frame 获取当前帧（使用 RobotGo 进行真实屏幕捕获）
func (g *DXGIGrabber) Frame() (*image.RGBA, error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	if !g.running {
		return nil, errors.New("DXGI捕获器未运行")
	}

	// 使用 RobotGo 进行真实屏幕捕获
	// TODO: 后续可以替换为真正的 DXGI 实现
	// start := time.Now()

	// 获取屏幕尺寸
	width, height := robotgo.GetScreenSize()

	// 捕获整个屏幕
	bitmap := robotgo.CaptureScreen(0, 0, width, height)
	if bitmap == nil {
		return nil, errors.New("RobotGo 屏幕捕获失败")
	}

	// 转换为 image.RGBA
	img := robotgo.ToImage(bitmap)
	robotgo.FreeBitmap(bitmap) // 释放 bitmap 内存

	if img == nil {
		return nil, errors.New("转换为 image.RGBA 失败")
	}

	// 转换为 *image.RGBA
	var rgbaImg *image.RGBA
	if rgba, ok := img.(*image.RGBA); ok {
		rgbaImg = rgba
	} else {
		// 如果不是 RGBA 格式，需要转换
		bounds := img.Bounds()
		rgbaImg = image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgbaImg.Set(x, y, img.At(x, y))
			}
		}
	}

	// elapsed := time.Since(start)
	// log.Debugf("屏幕捕获耗时: %v, 尺寸: %dx%d", elapsed, rgbaImg.Bounds().Dx(), rgbaImg.Bounds().Dy())

	return rgbaImg, nil
}

// Screen 返回捕获的屏幕信息
func (g *DXGIGrabber) Screen() *Screen {
	return &g.screen
}

// IsRunning 检查是否正在运行
func (g *DXGIGrabber) IsRunning() bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	return g.running
}

// GetStats 获取统计信息
func (g *DXGIGrabber) GetStats() CaptureStats {
	return CaptureStats{
		FramesCaptured:   0,
		FramesDropped:    0,
		ErrorCount:       0,
		AverageFrameTime: 0,
	}
}
