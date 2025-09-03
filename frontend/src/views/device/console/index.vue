<!-- eslint-disable vue/attributes-order -->
<!-- eslint-disable prettier/prettier -->
<template>
  <div class="console-container">
    <!-- 控制台头部 -->
    <div class="console-header">
      <div class="device-info">
        <h3>远程控制台 - {{ deviceInfo.hostname || deviceId }}</h3>
        <div class="connection-status">
          <span :class="['status-indicator', connectionStatus]" />
          {{ connectionStatusText }}
        </div>
      </div>

      <div class="control-buttons">
        <el-button-group>
          <el-button @click="toggleFullscreen" :icon="isFullscreen ? 'Compress' : 'FullScreen'">
            {{ isFullscreen ? '退出全屏' : '全屏' }}
          </el-button>
          <el-button @click="showDesktop" icon="Monitor">桌面</el-button>
          <el-button @click="openTaskManager" icon="Grid">任务管理器</el-button>
          <el-button @click="rebootDevice" icon="RefreshRight" type="danger">重启</el-button>
          <el-button @click="disconnect" icon="SwitchButton">断开连接</el-button>
        </el-button-group>

        <el-select v-model="streamType" @change="switchStreamType" style="margin-left: 16px; width: 150px">
          <el-option label="WebSocket流" value="websocket" />
          <el-option label="Guacamole" value="guacamole" />
          <el-option label="WebRTC" value="webrtc" />
        </el-select>
      </div>
    </div>

    <!-- 主显示区域 -->
    <div class="console-main" ref="consoleMain">
      <div v-loading="isLoading" class="stream-container">
        <!-- WebSocket 视频流 -->
        <div v-if="streamType === 'websocket'" class="websocket-stream">
          <JMuxerDecoder
            v-if="deviceInfo.lan"
            :device-id="deviceId"
            :device-ip="deviceInfo.lan"
            :auto-start="true"
            @connected="handleStreamConnected"
            @disconnected="handleStreamDisconnected"
            @error="handleStreamError"
          />
          <div v-else class="loading-device-info">
            <el-icon class="is-loading"><Loading /></el-icon>
            正在获取设备信息...
          </div>
        </div>

        <!-- Guacamole 远程桌面 -->
        <div v-else-if="streamType === 'guacamole'" class="guacamole-stream">
          <div
            ref="guacDisplay"
            :id="guacDisplayId"
            class="guac-display"
            tabindex="0"
            @click="focusGuacDisplay"
          />
        </div>

        <!-- WebRTC 流 -->
        <div v-else-if="streamType === 'webrtc'" class="webrtc-stream">
          <video
            ref="webrtcVideo"
            autoplay
            muted
            playsinline
            class="video-display"
          />
          <div class="webrtc-info">WebRTC 功能开发中...</div>
        </div>

        <!-- 连接错误提示 -->
        <div v-if="connectionError" class="error-message">
          <el-alert
            :title="connectionError"
            type="error"
            :closable="false"
            show-icon
          />
          <el-button @click="reconnect" type="primary" style="margin-top: 16px">
            重新连接
          </el-button>
        </div>
      </div>
    </div>

    <!-- 底部状态栏 -->
    <div class="console-footer">
      <div class="status-info">
        <span>设备: {{ deviceInfo.lan || deviceId }}</span>
        <span v-if="deviceInfo.wan">外网: {{ deviceInfo.wan }}</span>
        <span>分辨率: {{ resolution.width }}x{{ resolution.height }}</span>
        <span v-if="streamType === 'websocket'">FPS: {{ fps }}</span>
      </div>

      <div class="performance-info">
        <span v-if="latency > 0">延迟: {{ latency }}ms</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { deviceApi, type DeviceInfo } from '@/api/device'
import JMuxerDecoder from '@/views/decoders/JMuxerDecoder.vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import { computed, nextTick, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// 路由和设备信息
const route = useRoute()
const router = useRouter()
const deviceId = ref(route.params.id as string)
const deviceInfo = ref<Partial<DeviceInfo>>({})

// 连接状态
const connectionStatus = ref<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected')
const connectionError = ref('')
const isLoading = ref(false)
const isFullscreen = ref(false)

// 流类型和配置
const streamType = ref('websocket')
const fps = ref(20)
const resolution = reactive({ width: 1920, height: 1080 })
const latency = ref(0)

// 视频显示尺寸（动态计算）
const viewWidth = ref(960)
const viewHeight = ref(540)
const displayRect = ref<DOMRect | null>(null)

// DOM 引用
const consoleMain = ref<HTMLElement>()
const viewport = ref<HTMLElement>()
const display = ref<HTMLElement>()
const videoElement = ref<HTMLVideoElement>()
const guacDisplay = ref<HTMLElement>()
const webrtcVideo = ref<HTMLVideoElement>()

// 唯一ID
const videoElementId = computed(() => `video-${deviceId.value}`)
const viewportId = computed(() => `viewport-${deviceId.value}`)
const displayId = computed(() => `display-${deviceId.value}`)
const guacDisplayId = computed(() => `guac-${deviceId.value}`)

// 连接状态文本
const connectionStatusText = computed(() => {
  const statusMap = {
    disconnected: '未连接',
    connecting: '连接中...',
    connected: '已连接',
    error: '连接错误'
  }
  return statusMap[connectionStatus.value]
})

// WebSocket 和其他连接对象
let wsVideo: WebSocket | null = null
let jmuxer: any = null
let guacClient: any = null
let guacTunnel: any = null

// 获取设备信息
const getDeviceInfo = async () => {
  try {
    console.log('开始获取设备信息，设备ID:', deviceId.value)

    // 先获取设备基本信息（通过ID）
    const response = await deviceApi.getDevice(Number(deviceId.value))
    deviceInfo.value = response.data
    console.log('设备基本信息:', deviceInfo.value)

    // 然后获取系统详细信息（通过ID，后端会转换为LAN查询Agent）
    try {
      const systemResponse = await deviceApi.getDeviceInfo(Number(deviceId.value))
      // 合并系统信息到设备信息中
      if (systemResponse.data) {
        deviceInfo.value = { ...deviceInfo.value, ...systemResponse.data }
      }
      console.log('合并后的设备信息:', deviceInfo.value)
    } catch (systemError) {
      console.warn('获取系统详细信息失败:', systemError)
      // 不影响基本功能，继续执行
    }

    // 确保设备IP可用
    if (deviceInfo.value.lan) {
      console.log('设备IP地址可用:', deviceInfo.value.lan)
    } else {
      console.warn('设备IP地址不可用，设备信息:', deviceInfo.value)
    }

  } catch (error) {
    ElMessage.error('获取设备信息失败')
    console.error('获取设备信息错误:', error)
  }
}

// 初始化连接
const initConnection = async () => {
  await getDeviceInfo()

  // 检查设备信息是否正确加载
  console.log('设备信息加载完成，检查lan字段:', {
    deviceInfo: deviceInfo.value,
    lan: deviceInfo.value.lan,
    hasLan: !!deviceInfo.value.lan
  })

  if (!deviceInfo.value.lan) {
    ElMessage.error('设备IP地址不可用')
    return
  }

  switch (streamType.value) {
    case 'websocket':
      console.log('使用WebSocket流模式，设备IP:', deviceInfo.value.lan)
      // WebSocket流由JMuxerDecoder组件自动处理，不需要在这里初始化
      break
    case 'guacamole':
      await initGuacamoleStream()
      break
    case 'webrtc':
      await initWebRTCStream()
      break
  }
}

// 切换流类型
const switchStreamType = async (type: string) => {
  await disconnect()
  streamType.value = type
  await nextTick()
  await initConnection()
}

// 断开连接
const disconnect = async () => {
  connectionStatus.value = 'disconnected'
  connectionError.value = ''

  // 停止视频流
  if (streamType.value === 'websocket') {
    try {
      await deviceApi.stopStream(Number(deviceId.value))
    } catch (error) {
      console.warn('停止视频流失败:', error)
    }
  }

  // 清理 WebSocket
  if (wsVideo) {
    wsVideo.close()
    wsVideo = null
  }

  // 清理 JMuxer
  if (jmuxer) {
    jmuxer.destroy()
    jmuxer = null
  }

  // 清理 Guacamole
  if (guacClient) {
    guacClient.disconnect()
    guacClient = null
  }

  if (guacTunnel) {
    guacTunnel.disconnect()
    guacTunnel = null
  }
}

// 重新连接
const reconnect = async () => {
  await disconnect()
  await initConnection()
}

// 处理连接错误
const handleConnectionError = (error: string) => {
  connectionStatus.value = 'error'
  connectionError.value = error
  isLoading.value = false
  ElMessage.error(error)
}

// 计算视频显示尺寸（参考重构前项目）
const getDimensions = () => {
  if (!viewport.value) {
    console.warn('viewport元素不存在')
    return
  }

  const containerWidth = viewport.value.offsetWidth
  if (containerWidth <= 0) {
    console.warn('容器宽度为0')
    return
  }

  // 确保宽度是16的倍数（重构前项目的逻辑）
  viewWidth.value = containerWidth - (containerWidth % 16)
  // 16:9比例
  viewHeight.value = viewWidth.value * 0.5625

  // 延迟获取display的位置信息，用于鼠标坐标转换
  setTimeout(() => {
    if (display.value) {
      displayRect.value = display.value.getBoundingClientRect()
    }
  }, 500)

  console.log(`视频尺寸计算: ${viewWidth.value}x${viewHeight.value}`)
}

// 页面初始化
onMounted(async () => {
  // 添加全屏状态监听
  document.addEventListener('fullscreenchange', handleFullscreenChange)

  // 添加窗口大小变化监听
  window.addEventListener('resize', getDimensions)

  await initConnection()
})

// 页面销毁
onUnmounted(async () => {
  // 移除全屏状态监听
  document.removeEventListener('fullscreenchange', handleFullscreenChange)

  // 移除窗口大小变化监听
  window.removeEventListener('resize', getDimensions)

  await disconnect()
})

// WebSocket 流媒体方法
const initWebSocketStream = async () => {
  try {
    isLoading.value = true
    connectionStatus.value = 'connecting'

    await nextTick()

    // 先启动设备的视频流
    try {
      await deviceApi.startStream(Number(deviceId.value))
    } catch (error) {
      console.warn('启动视频流失败，继续尝试连接:', error)
    }

    // 动态导入 JMuxer
    const JMuxer = (await import('jmuxer')).default

    // 初始化 JMuxer - 使用更兼容的配置
    jmuxer = new JMuxer({
      node: videoElementId.value,
      mode: 'video',
      flushingTime: 1000, // 增加缓冲时间
      clearBuffer: false, // 不清除缓冲区
      fps: 15, // 与Agent实际使用的15fps保持一致
      debug: true, // 开启调试模式查看问题
      onReady: () => {
        console.log('JMuxer is ready')
      },
      onError: (error) => {
        console.error('JMuxer error:', error)
        ElMessage.error('JMuxer解码错误: ' + error.message)
      }
    })

    // 构建 WebSocket URL - 根据重构前项目的配置
    let wsUrl: string

    // 如果是本地调试环境，直接连接到agent
    if (window.location.host.startsWith('127.0.0.1') || window.location.host.startsWith('localhost')) {
      // 直接连接到设备的agent
      wsUrl = `ws://${deviceInfo.value.lan}:50052/wsstream`
    } else {
      // 生产环境通过后端代理
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      wsUrl = `${protocol}//${window.location.host}/api/ws/${deviceId.value}/stream`
    }

    console.log('WebSocket URL:', wsUrl)

    wsVideo = new WebSocket(wsUrl)
    wsVideo.binaryType = 'arraybuffer'

    wsVideo.onopen = () => {
      connectionStatus.value = 'connected'
      isLoading.value = false
      ElMessage.success('WebSocket 连接成功')

      // 连接成功后计算尺寸
      nextTick(() => {
        getDimensions()
      })
    }

    wsVideo.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        // 添加调试信息
        const dataSize = event.data.byteLength
        console.log(`Received data: ${dataSize} bytes`)

        const uint8Array = new Uint8Array(event.data)

        // 检查数据格式
        if (uint8Array.length >= 4) {
          const header = Array.from(uint8Array.slice(0, 4)).map(b => b.toString(16).padStart(2, '0')).join(' ')
          console.log(`Data header: ${header}`)

          // 检查是否为JPEG格式 (FF D8 FF)
          if (uint8Array[0] === 0xFF && uint8Array[1] === 0xD8 && uint8Array[2] === 0xFF) {
            console.log('Detected JPEG data')
            // 创建blob URL并显示图像
            const blob = new Blob([uint8Array], { type: 'image/jpeg' })
            const url = URL.createObjectURL(blob)

            // 获取video容器并显示图像
            const videoContainer = document.getElementById(videoElementId.value)
            if (videoContainer) {
              // 查找或创建img元素
              let img = videoContainer.querySelector('img') as HTMLImageElement
              if (!img) {
                img = document.createElement('img')
                img.style.width = '100%'
                img.style.height = '100%'
                img.style.objectFit = 'contain'
                img.style.display = 'block'
                videoContainer.appendChild(img)
                console.log('Created new img element')
              }

              // 释放之前的URL
              if (img.src && img.src.startsWith('blob:')) {
                try {
                  URL.revokeObjectURL(img.src)
                } catch (e) {
                  console.warn('Failed to revoke URL:', e)
                }
              }

              // 设置新的图像源
              img.src = url

              img.onload = () => {
                console.log('Image updated successfully, dimensions:', img.naturalWidth, 'x', img.naturalHeight)
              }
              img.onerror = (error) => {
                console.error('Image load error:', error)
                try {
                  URL.revokeObjectURL(url)
                } catch (e) {
                  console.warn('Failed to revoke URL on error:', e)
                }
              }
            }
          } else {
            // H.264数据，使用JMuxer
            if (jmuxer) {
              jmuxer.feed({
                video: uint8Array
              })
            }
          }
        }
      }
    }

    wsVideo.onerror = (error) => {
      console.error('WebSocket 错误:', error)
      handleConnectionError('WebSocket 连接失败')
    }

    wsVideo.onclose = (event) => {
      connectionStatus.value = 'disconnected'
      if (event.code !== 1000) {
        handleConnectionError(`WebSocket 连接断开: ${event.reason || '未知原因'}`)
      } else {
        ElMessage.warning('WebSocket 连接已断开')
      }
    }

  } catch (error) {
    console.error('初始化 WebSocket 流失败:', error)
    handleConnectionError('初始化失败: ' + (error as Error).message)
  }
}

// Guacamole 远程桌面方法
const initGuacamoleStream = async () => {
  try {
    isLoading.value = true
    connectionStatus.value = 'connecting'

    await nextTick()

    // 动态导入 Guacamole
    const Guacamole = (await import('guacamole-common-js')).default

    // 构建 WebSocket URL
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/ws/${deviceId.value}`

    // 创建 Guacamole 隧道
    guacTunnel = new Guacamole.WebSocketTunnel(wsUrl)

    // 创建 Guacamole 客户端
    guacClient = new Guacamole.Client(guacTunnel)

    // 获取显示元素
    const display = guacClient.getDisplay()

    // 将显示元素添加到容器中
    if (guacDisplay.value) {
      guacDisplay.value.innerHTML = ''
      guacDisplay.value.appendChild(display.getElement())
    }

    // 设置鼠标
    const mouse = new Guacamole.Mouse(display.getElement())
    mouse.onmousedown = mouse.onmouseup = mouse.onmousemove = (mouseState) => {
      guacClient.sendMouseState(mouseState)
    }

    // 设置键盘
    const keyboard = new Guacamole.Keyboard(guacDisplay.value)
    keyboard.onkeydown = (keysym) => {
      guacClient.sendKeyEvent(1, keysym)
    }
    keyboard.onkeyup = (keysym) => {
      guacClient.sendKeyEvent(0, keysym)
    }

    // 连接事件处理
    guacClient.onstatechange = (state) => {
      switch (state) {
        case Guacamole.Client.IDLE:
          connectionStatus.value = 'disconnected'
          break
        case Guacamole.Client.CONNECTING:
          connectionStatus.value = 'connecting'
          break
        case Guacamole.Client.CONNECTED:
          connectionStatus.value = 'connected'
          isLoading.value = false
          ElMessage.success('Guacamole 连接成功')
          break
        case Guacamole.Client.DISCONNECTING:
          connectionStatus.value = 'disconnected'
          break
        case Guacamole.Client.DISCONNECTED:
          connectionStatus.value = 'disconnected'
          ElMessage.warning('Guacamole 连接已断开')
          break
      }
    }

    guacClient.onerror = (error) => {
      console.error('Guacamole 错误:', error)
      handleConnectionError('Guacamole 连接失败: ' + error.message)
    }

    // 开始连接
    guacClient.connect()

  } catch (error) {
    console.error('初始化 Guacamole 失败:', error)
    handleConnectionError('初始化失败: ' + (error as Error).message)
  }
}

// WebRTC 流媒体方法（暂未实现）
const initWebRTCStream = async () => {
  ElMessage.info('WebRTC 流功能开发中...')
  connectionStatus.value = 'error'
  connectionError.value = 'WebRTC 功能尚未实现'
}

// 全屏切换（参考重构前项目，全屏viewport而不是整个控制台）
const toggleFullscreen = () => {
  if (!document.fullscreenElement) {
    viewport.value?.requestFullscreen().then(() => {
      isFullscreen.value = true
      // 全屏后重新计算尺寸
      setTimeout(() => {
        getDimensions()
      }, 100)
    }).catch(err => {
      ElMessage.error('进入全屏失败: ' + err.message)
    })
  } else {
    document.exitFullscreen().then(() => {
      isFullscreen.value = false
      // 退出全屏后重新计算尺寸
      setTimeout(() => {
        getDimensions()
      }, 100)
    }).catch(err => {
      ElMessage.error('退出全屏失败: ' + err.message)
    })
  }
}

// 显示桌面
const showDesktop = async () => {
  try {
    await deviceApi.sendKeyboard(Number(deviceId.value), 'win_d')
    ElMessage.success('已发送显示桌面指令')
  } catch (error) {
    ElMessage.error('发送指令失败')
  }
}

// 打开任务管理器（Agent暂不支持ctrl+shift+esc，使用其他方式）
const openTaskManager = async () => {
  try {
    // Agent的KeyboardHandler只支持win_d和win，暂时提示用户手动操作
    ElMessage.info('请手动按 Ctrl+Shift+Esc 打开任务管理器')
  } catch (error) {
    ElMessage.error('发送指令失败')
  }
}

// 重启设备
const rebootDevice = async () => {
  try {
    await ElMessageBox.confirm('确定要重启设备吗？', '确认重启', {
      type: 'warning'
    })

    await deviceApi.rebootDevice(Number(deviceId.value))
    ElMessage.success('重启指令已发送')

    // 断开连接
    await disconnect()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('重启失败')
    }
  }
}

// 聚焦显示区域
const focusDisplay = () => {
  if (display.value) {
    display.value.focus()
  }
}

// 鼠标进入事件
const handleMouseEnter = () => {
  if (display.value) {
    display.value.focus()
    // 重新计算尺寸
    getDimensions()
  }
}

// 鼠标离开事件
const handleMouseLeave = () => {
  // 可以在这里添加鼠标离开的处理逻辑
}

// 视频点击事件处理（使用正确的坐标转换）
const handleVideoClick = (event: MouseEvent) => {
  if (streamType.value === 'websocket' && displayRect.value) {
    // 使用固定的分辨率进行坐标转换（参考重构前项目）
    const w = 1920
    const h = 1080
    const x = Math.round((w / displayRect.value.width) * (event.clientX - displayRect.value.left))
    const y = Math.round((h / displayRect.value.height) * (event.clientY - displayRect.value.top))

    console.log(`鼠标点击: 屏幕坐标(${event.clientX}, ${event.clientY}) -> 设备坐标(${x}, ${y})`)

    // 发送鼠标点击事件到设备
    sendMouseEvent('click', x, y, 1)
  }
}

// 视频右键点击事件处理
const handleVideoRightClick = (event: MouseEvent) => {
  event.preventDefault()

  if (streamType.value === 'websocket' && displayRect.value) {
    // 使用固定的分辨率进行坐标转换（参考重构前项目）
    const w = 1920
    const h = 1080
    const x = Math.round((w / displayRect.value.width) * (event.clientX - displayRect.value.left))
    const y = Math.round((h / displayRect.value.height) * (event.clientY - displayRect.value.top))

    console.log(`鼠标右键: 屏幕坐标(${event.clientX}, ${event.clientY}) -> 设备坐标(${x}, ${y})`)

    // 发送鼠标右键点击事件到设备
    sendMouseEvent('rightclick', x, y, 2)
  }
}

// 发送鼠标事件
const sendMouseEvent = async (type: string, x: number, y: number, button: number) => {
  try {
    await deviceApi.sendMouseEvent(Number(deviceId.value), {
      type,
      x,
      y,
      button
    })
  } catch (error) {
    console.error('发送鼠标事件失败:', error)
  }
}

// 聚焦 Guacamole 显示区域
const focusGuacDisplay = () => {
  if (guacDisplay.value) {
    guacDisplay.value.focus()
  }
}

// 监听全屏状态变化
const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
}

// 流事件处理方法
const handleStreamConnected = () => {
  connectionStatus.value = 'connected'
  isLoading.value = false
  connectionError.value = ''
  ElMessage.success('视频流连接成功')
}

const handleStreamDisconnected = () => {
  connectionStatus.value = 'disconnected'
  ElMessage.warning('视频流连接断开')
}

const handleStreamError = (message: string) => {
  connectionStatus.value = 'error'
  connectionError.value = message
  isLoading.value = false
  ElMessage.error(`视频流错误: ${message}`)
}
</script>

<style scoped>
.console-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #1a1a1a;
  color: #fff;
}

.console-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: #2d2d2d;
  border-bottom: 1px solid #404040;
}

.device-info h3 {
  margin: 0 0 4px 0;
  font-size: 16px;
  font-weight: 500;
}

.connection-status {
  display: flex;
  align-items: center;
  font-size: 12px;
  color: #999;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 6px;
}

.status-indicator.disconnected {
  background: #666;
}

.status-indicator.connecting {
  background: #f39c12;
  animation: pulse 1.5s infinite;
}

.status-indicator.connected {
  background: #27ae60;
}

.status-indicator.error {
  background: #e74c3c;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.control-buttons {
  display: flex;
  align-items: center;
}

.console-main {
  flex: 1;
  background: #000;
  position: relative;
  /* 移除居中对齐，让视频占满空余区域 */
}

.stream-container {
  width: 100%;
  height: 100%;
  /* 移除居中对齐，让视频直接占满容器 */
}

.websocket-stream {
  width: 100%;
  height: 100%;
}

.loading-device-info {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #fff;
  font-size: 16px;
  gap: 8px;
}

/* 视频流容器样式（参考重构前项目） */
.viewport {
  background-color: rgba(251, 152, 116, 0.1);
  position: relative;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
}

.display {
  position: relative;
  background: #000;
  border: 1px solid #333;
  outline: none;
}

.video-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
}

.video-container {
  width: 100%;
  height: 100%;
  position: relative;
  background: #000;
}

/* JMuxer创建的video元素样式 */
.video-container video {
  width: 100%;
  height: 100%;
  object-fit: contain; /* 保持比例，不裁剪 */
  background: #000;
  border: none;
  outline: none;
}

.video-display {
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  bottom: 0;
  right: 0;
  object-fit: cover; /* 关键：覆盖模式，防止变形 */
  background: #000;
  border: none;
  outline: none;
}

.guac-display {
  max-width: 100%;
  max-height: 100%;
  border: none;
  outline: none;
  background: #fff;
}

.guac-display {
  background: #fff;
  cursor: crosshair;
}

.webrtc-info {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #999;
  font-size: 14px;
}

.error-message {
  text-align: center;
  padding: 40px;
}

.console-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 20px;
  background: #2d2d2d;
  border-top: 1px solid #404040;
  font-size: 12px;
  color: #999;
}

.status-info,
.performance-info {
  display: flex;
  gap: 16px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .console-header {
    flex-direction: column;
    gap: 12px;
    padding: 12px;
  }

  .control-buttons {
    width: 100%;
    justify-content: space-between;
  }

  .console-footer {
    flex-direction: column;
    gap: 8px;
    text-align: center;
  }
}
</style>
