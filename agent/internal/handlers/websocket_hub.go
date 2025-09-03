package handlers

import (
	"fmt"
	"image"
	"sync"
	"time"

	"winmanager-agent/internal/config"
	"winmanager-agent/pkg/debug"
	"winmanager-agent/pkg/encoders"
	"winmanager-agent/pkg/screen"
	"winmanager-agent/pkg/utils"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"golang.org/x/image/draw"
)

// 全局单例Hub和编码器，复用资源
var (
	globalHub        *Hub
	globalEncoder    encoders.Encoder
	globalGrabber    screen.ScreenGrabber
	globalVideoSaver *debug.VideoSaver
	hubOnce          sync.Once
	hubMutex         sync.Mutex
	running          bool
	stop             chan struct{}
)

// Hub 管理所有WebSocket连接
type Hub struct {
	clients     map[*Connection]bool
	broadcast   chan []byte
	register    chan *Connection
	unregister  chan *Connection
	clientsName map[*Connection]string
	mutex       sync.RWMutex

	// H.264关键帧缓存
	keyFrameCache []byte       // 最新的关键帧数据（包含SPS/PPS）
	keyFrameMutex sync.RWMutex // 关键帧缓存的读写锁
}

// Connection WebSocket连接
type Connection struct {
	ws   *websocket.Conn
	send chan []byte
	hub  *Hub
	run  bool
	key  string
	stop chan struct{}
}

// 初始化全局Hub
func initGlobalHub() {
	hubOnce.Do(func() {
		log.Info("初始化全局WebSocket Hub")
		globalHub = &Hub{
			broadcast:   make(chan []byte),
			register:    make(chan *Connection),
			unregister:  make(chan *Connection),
			clients:     make(map[*Connection]bool),
			clientsName: make(map[*Connection]string),
		}
		stop = make(chan struct{})
		go globalHub.run()
	})
}

// Hub运行循环
func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()
			log.Infof("WebSocket客户端注册: %s", client.key)

			// 立即发送缓存的关键帧给新客户端
			h.sendKeyFrameToClient(client)

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				delete(h.clientsName, client)
				close(client.send)
			}
			h.mutex.Unlock()
			log.Infof("WebSocket客户端注销: %s", client.key)

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clientsName, client)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

// 设置连接名称
func (h *Hub) setHubConnName(conn *Connection) string {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	u1 := uuid.NewV4()
	h.clientsName[conn] = u1.String()
	return u1.String()
}

// 停止所有连接
func (h *Hub) stopAllHubConn() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	for client := range h.clients {
		h.unregister <- client
	}
}

// 获取客户端数量
func (h *Hub) getClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// 发送关键帧给指定客户端
func (h *Hub) sendKeyFrameToClient(client *Connection) {
	h.keyFrameMutex.RLock()
	keyFrame := h.keyFrameCache
	h.keyFrameMutex.RUnlock()

	if keyFrame != nil && len(keyFrame) > 0 {
		// 分析关键帧内容
		h.analyzeKeyFrameContent(keyFrame)

		select {
		case client.send <- keyFrame:
			log.Infof("关键帧已发送给新客户端: %s (%d字节)", client.key, len(keyFrame))
		default:
			log.Warnf("无法发送关键帧给客户端: %s (发送通道满)", client.key)
		}
	} else {
		log.Infof("暂无关键帧缓存，为新客户端请求生成关键帧: %s", client.key)
		// 没有缓存的关键帧，请求编码器生成一个新的关键帧
		h.requestKeyFrame()
	}
}

// 分析关键帧内容（用于调试）
func (h *Hub) analyzeKeyFrameContent(payload []byte) {
	if len(payload) < 10 {
		return
	}

	nalUnits := []string{}

	// 查找所有NAL单元
	for i := 0; i < len(payload)-4; i++ {
		if payload[i] == 0x00 && payload[i+1] == 0x00 && payload[i+2] == 0x00 && payload[i+3] == 0x01 {
			if i+4 < len(payload) {
				nalType := payload[i+4] & 0x1F
				switch nalType {
				case 1:
					nalUnits = append(nalUnits, "P帧")
				case 5:
					nalUnits = append(nalUnits, "IDR帧")
				case 7:
					nalUnits = append(nalUnits, "SPS")
				case 8:
					nalUnits = append(nalUnits, "PPS")
				case 9:
					nalUnits = append(nalUnits, "AUD")
				default:
					nalUnits = append(nalUnits, fmt.Sprintf("NAL%d", nalType))
				}
			}
		}
	}

	log.Debugf("关键帧内容分析: %v, 头部: %02x %02x %02x %02x %02x %02x %02x %02x",
		nalUnits, payload[0], payload[1], payload[2], payload[3], payload[4], payload[5], payload[6], payload[7])
}

// 分析帧内容（用于调试所有帧）- 优化版本
var frameAnalysisCounter int
var lastAnalysisTime time.Time

func (h *Hub) analyzeFrameContent(payload []byte) {
	frameAnalysisCounter++

	// 每10秒分析一次，而不是每100帧，避免日志过多
	now := time.Now()
	if now.Sub(lastAnalysisTime) < 10*time.Second {
		return
	}
	lastAnalysisTime = now

	if len(payload) < 10 {
		return
	}

	nalTypes := []int{}

	// 查找所有NAL单元类型
	for i := 0; i < len(payload)-4; i++ {
		if payload[i] == 0x00 && payload[i+1] == 0x00 && payload[i+2] == 0x00 && payload[i+3] == 0x01 {
			if i+4 < len(payload) {
				nalType := int(payload[i+4] & 0x1F)
				nalTypes = append(nalTypes, nalType)
			}
		}
	}

	log.Infof("流媒体状态: 总帧数=%d, 当前帧NAL类型=%v, 大小=%d字节",
		frameAnalysisCounter, nalTypes, len(payload))
}

// 更新关键帧缓存
func (h *Hub) updateKeyFrameCache(payload []byte) {
	if h.isKeyFrame(payload) {
		h.keyFrameMutex.Lock()

		// 检查是否需要添加AUD（Access Unit Delimiter）
		needsAUD := true
		if len(payload) >= 6 {
			// 检查是否已经有AUD (NAL type 9)
			for i := 0; i < len(payload)-5; i++ {
				if payload[i] == 0x00 && payload[i+1] == 0x00 && payload[i+2] == 0x00 && payload[i+3] == 0x01 {
					nalType := payload[i+4] & 0x1F
					if nalType == 9 { // AUD
						needsAUD = false
						break
					}
				}
			}
		}

		if needsAUD {
			// 添加AUD到关键帧前面
			aud := []byte{0x00, 0x00, 0x00, 0x01, 0x09, 0x10} // AUD with primary_pic_type = 0
			h.keyFrameCache = make([]byte, len(aud)+len(payload))
			copy(h.keyFrameCache, aud)
			copy(h.keyFrameCache[len(aud):], payload)
			log.Infof("关键帧缓存已更新(含AUD): %d字节", len(h.keyFrameCache))
		} else {
			h.keyFrameCache = make([]byte, len(payload))
			copy(h.keyFrameCache, payload)
			log.Infof("关键帧缓存已更新: %d字节", len(payload))
		}

		h.keyFrameMutex.Unlock()
	}
}

// 检查是否为关键帧（包含SPS/PPS或IDR帧）
func (h *Hub) isKeyFrame(payload []byte) bool {
	if len(payload) < 5 {
		return false
	}

	hasSPS := false
	hasPPS := false
	hasIDR := false

	// 查找H.264 NAL单元
	for i := 0; i < len(payload)-4; i++ {
		// 查找NAL单元起始码 0x00000001
		if payload[i] == 0x00 && payload[i+1] == 0x00 && payload[i+2] == 0x00 && payload[i+3] == 0x01 {
			if i+4 < len(payload) {
				nalType := payload[i+4] & 0x1F
				switch nalType {
				case 5: // IDR帧
					hasIDR = true
				case 7: // SPS
					hasSPS = true
				case 8: // PPS
					hasPPS = true
				}
			}
		}
	}

	// 只有包含IDR帧的才认为是真正的关键帧，仅SPS+PPS不足以让新客户端开始解码
	isKeyFrame := hasIDR

	// 记录检测到的帧类型
	if hasIDR {
		log.Infof("检测到IDR关键帧: SPS=%v, PPS=%v, IDR=%v, 大小=%d字节", hasSPS, hasPPS, hasIDR, len(payload))
	} else if hasSPS && hasPPS {
		log.Infof("检测到参数集: SPS=%v, PPS=%v, IDR=%v, 大小=%d字节", hasSPS, hasPPS, hasIDR, len(payload))
	}

	return isKeyFrame
}

// 请求生成关键帧（用于新客户端连接时）
func (h *Hub) requestKeyFrame() {
	hubMutex.Lock()
	encoder := globalEncoder
	hubMutex.Unlock()

	if encoder != nil {
		// 检查编码器是否支持强制关键帧
		if forceKeyFramer, ok := encoder.(interface{ ForceKeyFrame() }); ok {
			forceKeyFramer.ForceKeyFrame()
			log.Infof("已请求编码器生成关键帧")
		}
	} else {
		log.Info("编码器未初始化，关键帧将在编码器创建时自动生成")
	}
}

// 初始化全局屏幕捕获器（单例模式）
func initGlobalGrabber() error {
	log.Info("检查全局捕获器状态...")

	if globalGrabber != nil {
		log.Info("全局捕获器已存在，跳过初始化")
		return nil // 已经初始化
	}

	log.Info("开始初始化全局捕获器")

	// 先创建屏幕捕获器获取实际分辨率
	log.Info("创建屏幕捕获器...")
	screenService := screen.NewScreenService()
	primaryScreen, err := screenService.PrimaryScreen()
	if err != nil {
		log.WithError(err).Error("获取主屏幕失败")
		return err
	}
	log.Infof("主屏幕信息: %+v", primaryScreen)

	// 获取配置
	cfg := config.GetGlobalConfig()
	screenConfig := cfg.GetScreenConfig()

	// 根据配置选择屏幕捕获方法
	captureMethod := screen.ParseCaptureMethodName(screenConfig.CaptureMethod)
	log.Infof("根据配置使用屏幕捕获方法: %s", screenConfig.CaptureMethod)

	screenService = screen.NewScreenService()
	globalGrabber, err = screenService.CreateScreenGrabber(*primaryScreen, captureMethod)
	if err != nil {
		log.WithError(err).Errorf("创建%s捕获器失败，尝试降级", screenConfig.CaptureMethod)
		// 降级到RobotGo
		globalGrabber, err = screen.NewRobotGoGrabber(*primaryScreen)
		if err != nil {
			log.WithError(err).Error("RobotGo捕获器创建也失败")
			return err
		}
		log.Info("RobotGo捕获器创建成功（降级）")
	} else {
		log.Infof("%s捕获器创建成功", screenConfig.CaptureMethod)
	}

	log.Info("全局捕获器初始化完成")
	return nil
}

// 初始化全局编码器（单例模式）
func initGlobalEncoder() error {
	log.Info("检查全局编码器状态...")

	if globalEncoder != nil {
		log.Info("全局编码器已存在，跳过初始化")
		return nil // 已经初始化
	}

	log.Info("开始初始化全局编码器")

	// 初始化视频保存器
	cfg := config.GetGlobalConfig()
	debugConfig := cfg.GetDebugConfig()
	log.Infof("调试配置检查: SavePath=%s, SaveVideoDuration=%d",
		debugConfig.SavePath, debugConfig.SaveVideoDuration)

	if cfg.IsDebugVideoSaveEnabled() {
		globalVideoSaver = debug.NewVideoSaver(
			debugConfig.SavePath,
			debugConfig.SaveVideoDuration,
		)
		log.Infof("视频保存器已启用: 路径=%s, 时长=%d秒",
			debugConfig.SavePath, debugConfig.SaveVideoDuration)
	} else {
		log.Info("视频保存器未启用")
	}

	// 获取屏幕分辨率（从已初始化的捕获器）
	if globalGrabber == nil {
		log.Error("全局捕获器未初始化，无法获取屏幕分辨率")
		return fmt.Errorf("全局捕获器未初始化")
	}

	// 从捕获器获取实际分辨率
	screenService := screen.NewScreenService()
	primaryScreen, err := screenService.PrimaryScreen()
	if err != nil {
		log.WithError(err).Error("获取主屏幕失败")
		return err
	}

	actualSize := image.Point{
		X: primaryScreen.Bounds.Dx(),
		Y: primaryScreen.Bounds.Dy(),
	}
	log.Infof("实际屏幕分辨率: %dx%d", actualSize.X, actualSize.Y)

	// 获取编码器配置
	encoderConfig := cfg.GetEncoderConfig()

	// 使用配置中的编码器和帧率设置
	log.Infof("根据配置创建编码器: codec=%s, frame_rate=%d", encoderConfig.DefaultCodec, encoderConfig.FrameRate)
	encoderService := encoders.NewEncoderService()

	// 根据配置选择编码器
	var codec encoders.VideoCodec
	switch encoderConfig.DefaultCodec {
	case "h264":
		codec = encoders.H264Codec
	case "jpeg":
		codec = encoders.JPEGCodec
	case "jpeg-turbo":
		codec = encoders.JPEGTurboCodec
	case "vp8":
		codec = encoders.VP8Codec
	default:
		log.Warnf("未知编码器类型: %s，使用默认H.264", encoderConfig.DefaultCodec)
		codec = encoders.H264Codec
	}

	globalEncoder, err = encoderService.NewEncoderWithConfig(codec, actualSize, encoderConfig.FrameRate)
	if err != nil {
		log.WithError(err).Errorf("创建%s编码器失败", encoderConfig.DefaultCodec)
		// 降级到JPEG编码器
		log.Info("降级到JPEG编码器...")
		globalEncoder, err = encoderService.NewEncoderWithConfig(encoders.JPEGCodec, actualSize, encoderConfig.FrameRate)
		if err != nil {
			log.WithError(err).Error("创建JPEG编码器也失败")
			return err
		}
		log.Info("JPEG编码器创建成功（降级）")
	} else {
		log.Infof("%s编码器创建成功", encoderConfig.DefaultCodec)
	}

	// 获取并记录编码器期望的尺寸
	if globalEncoder != nil {
		expectedSize, err := globalEncoder.VideoSize()
		if err == nil {
			log.Infof("编码器期望尺寸: %dx%d", expectedSize.X, expectedSize.Y)

			// 更新坐标映射
			if err := UpdateCoordinateMapping(expectedSize.X, expectedSize.Y); err != nil {
				log.WithError(err).Warn("更新坐标映射失败")
			} else {
				log.Info("坐标映射已根据编码器尺寸更新")
			}
		}
	}

	log.Info("全局编码器初始化完成")
	return nil
}

// StartGlobalStreaming 启动全局视频流
func StartGlobalStreaming() error {
	log.Info("开始启动全局视频流...")
	hubMutex.Lock()
	defer hubMutex.Unlock()

	if running {
		log.Info("全局视频流已在运行")
		return nil // 已经在运行
	}

	// 确保Hub已初始化
	initGlobalHub()

	// 初始化屏幕捕获器（必需），编码器延迟初始化
	log.Info("初始化屏幕捕获器...")
	if err := initGlobalGrabber(); err != nil {
		log.WithError(err).Error("初始化屏幕捕获器失败")
		return err
	}

	// 初始化坐标映射（基于实际屏幕分辨率）
	log.Info("初始化坐标映射...")
	if err := UpdateCoordinateMapping(1920, 1080); err != nil {
		log.WithError(err).Warn("初始化坐标映射失败，将使用默认值")
	} else {
		log.Info("坐标映射初始化完成")
	}

	log.Info("编码器将在首个客户端连接时创建")

	running = true
	log.Info("启动全局视频流成功")

	// 启动编码和广播循环
	go globalStreamingLoop()
	log.Info("全局视频流循环已启动")
	return nil
}

// StopGlobalStreaming 停止全局视频流
func StopGlobalStreaming() error {
	hubMutex.Lock()
	defer hubMutex.Unlock()

	if !running {
		log.Info("全局视频流未在运行，无需停止")
		return nil
	}

	log.Info("停止全局视频流")

	// 1. 首先设置停止标志
	running = false

	// 2. 发送停止信号并等待循环结束
	select {
	case stop <- struct{}{}:
		log.Debug("停止信号已发送")
	default:
		log.Warn("停止信号发送失败，通道可能已满")
	}

	// 3. 给循环一些时间来检查停止信号
	time.Sleep(100 * time.Millisecond)

	// 4. 停止视频录制
	if globalVideoSaver != nil {
		log.Debug("停止视频录制")
		globalVideoSaver.StopRecording()
	}

	// 5. 停止捕获器
	if globalGrabber != nil {
		log.Debug("停止屏幕捕获器")
		if err := globalGrabber.Stop(); err != nil {
			log.WithError(err).Warn("停止捕获器时出现警告")
		}
		globalGrabber = nil
	}

	// 6. 最后关闭编码器（确保循环已经停止）
	if globalEncoder != nil {
		log.Debug("关闭编码器")
		if err := globalEncoder.Close(); err != nil {
			log.WithError(err).Warn("关闭编码器时出现警告")
		}
		globalEncoder = nil
	}

	log.Info("全局视频流已停止")
	return nil
}

// 全局流媒体循环
func globalStreamingLoop() {
	// 从配置获取帧率
	cfg := config.GetGlobalConfig()
	encoderConfig := cfg.GetEncoderConfig()
	targetFPS := encoderConfig.FrameRate
	log.Infof("使用配置的帧率: %d FPS", targetFPS)
	limiter := utils.NewFrameLimiter(targetFPS)
	var oldFrame *image.RGBA
	var frameCount int

	log.Info("开始全局视频流循环")
	for running {
		select {
		case <-stop:
			log.Info("收到停止信号，退出流媒体循环")
			return
		default:
			limiter.Wait()
		}

		// 检查是否有客户端连接
		if globalHub == nil || globalHub.getClientCount() == 0 {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 检查grabber是否可用（线程安全）
		hubMutex.Lock()
		grabber := globalGrabber
		hubMutex.Unlock()

		if grabber == nil {
			log.Warn("全局grabber为空，跳过此帧")
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 捕获帧
		frame, err := grabber.Frame()
		if err != nil {
			log.WithError(err).Warn("捕获帧失败")
			if oldFrame != nil {
				frame = oldFrame
			} else {
				continue
			}
		}

		if frame == nil {
			log.Warn("捕获的帧为空，跳过")
			continue
		}

		// 验证帧的完整性
		bounds := frame.Bounds()
		frameWidth := bounds.Dx()
		frameHeight := bounds.Dy()

		// 检查帧尺寸是否有效
		if frameWidth <= 0 || frameHeight <= 0 {
			log.Warnf("捕获的帧尺寸无效: %dx%d，跳过", frameWidth, frameHeight)
			continue
		}

		// 检查像素数据是否完整
		expectedPixelCount := frameWidth * frameHeight * 4 // RGBA
		if len(frame.Pix) != expectedPixelCount {
			log.Warnf("帧像素数据不完整: 期望 %d 字节，实际 %d 字节，跳过",
				expectedPixelCount, len(frame.Pix))
			continue
		}

		// 采样检查帧是否为纯黑色
		isBlackFrame := true
		samplePoints := 20 // 增加采样点数量以更好地检测问题

		for i := 0; i < samplePoints && isBlackFrame; i++ {
			x := (frameWidth * i) / samplePoints
			y := (frameHeight * i) / samplePoints
			if x < frameWidth && y < frameHeight {
				pixel := frame.RGBAAt(x, y)

				// 检查是否为黑色像素
				if pixel.R > 10 || pixel.G > 10 || pixel.B > 10 {
					isBlackFrame = false
				}
			}
		}

		// 移除过于严格的损坏帧检测，让编码器处理所有帧

		oldFrame = frame
		frameCount++

		// 每500帧记录一次帧信息（约25秒@20fps）
		if frameCount%500 == 0 {
			log.Infof("流媒体运行状态: 帧#%d, 分辨率%dx%d, 黑帧检测:%v, 数据量:%d字节",
				frameCount, frameWidth, frameHeight, isBlackFrame, len(frame.Pix))
		}

		// 检查encoder是否可用，如果没有则初始化（线程安全）
		hubMutex.Lock()
		encoder := globalEncoder
		encoderJustCreated := false
		if encoder == nil {
			log.Info("编码器未初始化，现在创建编码器...")
			// 在有客户端连接时才初始化编码器
			if err := initGlobalEncoder(); err != nil {
				log.WithError(err).Error("延迟初始化编码器失败")
				hubMutex.Unlock()
				continue
			}
			encoder = globalEncoder
			encoderJustCreated = true
			log.Info("编码器延迟初始化成功")
		}
		hubMutex.Unlock()

		if encoder == nil {
			log.Warn("编码器初始化后仍为空，跳过此帧")
			continue
		}

		// 如果编码器刚刚创建，强制生成关键帧给等待的客户端
		if encoderJustCreated {
			if forceKeyFramer, ok := encoder.(interface{ ForceKeyFrame() }); ok {
				forceKeyFramer.ForceKeyFrame()
				log.Info("编码器初始化后强制生成关键帧")
			}
		}

		// 开始录制视频（如果是第一帧且启用了视频保存）
		if frameCount == 1 && globalVideoSaver != nil {
			if err := globalVideoSaver.StartRecording(); err != nil {
				log.WithError(err).Warn("启动视频录制失败")
			}
		}

		// 获取编码器期望尺寸
		expectedSize, sizeErr := encoder.VideoSize()
		if sizeErr != nil {
			log.WithError(sizeErr).Warn("无法获取编码器期望尺寸")
			continue
		}

		// 如果帧尺寸与编码器期望不匹配，进行高质量缩放（不改变目标分辨率/映射）
		var encodingFrame *image.RGBA = frame
		if frameWidth != expectedSize.X || frameHeight != expectedSize.Y {
			if frameCount == 1 {
				log.Infof("帧尺寸不匹配，需要缩放: 捕获 %dx%d -> 编码器期望 %dx%d",
					frameWidth, frameHeight, expectedSize.X, expectedSize.Y)
			}

			scaledFrame := image.NewRGBA(image.Rect(0, 0, expectedSize.X, expectedSize.Y))
			draw.CatmullRom.Scale(scaledFrame, scaledFrame.Bounds(), frame, frame.Bounds(), draw.Over, nil)
			encodingFrame = scaledFrame
		}

		// 编码帧（使用本地变量避免竞态条件）
		payload, err := encoder.Encode(encodingFrame)
		if err != nil {
			log.WithError(err).Warnf("编码帧失败: 原始尺寸 %dx%d, 编码尺寸 %dx%d, 像素数据 %d字节",
				frameWidth, frameHeight, expectedSize.X, expectedSize.Y, len(encodingFrame.Pix))
			continue
		}

		if payload == nil {
			log.Warn("编码后的payload为空，跳过")
			continue
		}

		// 写入视频数据（如果正在录制）
		if globalVideoSaver != nil && encoder.GetCodec() == encoders.H264Codec {
			if err := globalVideoSaver.WriteFrame(payload); err != nil {
				log.WithError(err).Warn("写入视频帧失败")
			}
		}

		// 每500帧记录一次编码信息（约25秒@20fps）
		if frameCount%500 == 0 {
			log.Infof("编码状态: 帧#%d, 输入%dx%d, 输出%d字节, 压缩比%.1f:1",
				frameCount, frameWidth, frameHeight, len(payload),
				float64(len(frame.Pix))/float64(len(payload)))
		}

		// 广播到所有客户端
		if globalHub != nil {
			// 分析每一帧的内容（用于调试）
			globalHub.analyzeFrameContent(payload)

			// 如果是关键帧，更新缓存
			globalHub.updateKeyFrameCache(payload)

			select {
			case globalHub.broadcast <- payload:
			default:
				// 广播通道满，跳过这一帧
			}
		}
	}
	log.Info("全局视频流循环结束")
}

// 获取全局Hub实例
func GetGlobalHub() *Hub {
	initGlobalHub()
	return globalHub
}

// 检查是否正在运行
func IsGlobalStreamingRunning() bool {
	hubMutex.Lock()
	defer hubMutex.Unlock()
	return running
}

// 获取连接统计
func GetStreamingStats() map[string]interface{} {
	hubMutex.Lock()
	defer hubMutex.Unlock()

	clientCount := 0
	if globalHub != nil {
		clientCount = globalHub.getClientCount()
	}

	return map[string]interface{}{
		"running":      running,
		"client_count": clientCount,
		"encoder":      globalEncoder != nil,
		"grabber":      globalGrabber != nil,
	}
}
