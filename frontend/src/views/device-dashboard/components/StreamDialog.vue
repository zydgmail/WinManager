<template>
  <!-- 自定义弹窗遮罩 -->
  <div v-if="dialogVisible" class="stream-dialog-overlay" @click="handleOverlayClick">
    <div class="stream-dialog-container" @click.stop>
      <!-- 自定义头部 -->
      <div class="dialog-header">
        <span class="dialog-title">
          {{ `${device?.hostname || '未知设备'}(${device?.lan || '未知IP'})` }}
        </span>

        <!-- 控制按钮区域（移到头部） -->
        <div v-if="isStreamActive" class="header-control-buttons">
          <el-button size="small" title="显示桌面" @click="showDesktop">
            <el-icon><Monitor /></el-icon>
            桌面
          </el-button>
          <el-button size="small" title="打开任务管理器" @click="openTaskManager">
            <el-icon><Setting /></el-icon>
            任务管理器
          </el-button>
          <el-button size="small" type="warning" title="重启设备" @click="rebootDevice">
            <el-icon><RefreshRight /></el-icon>
            重启
          </el-button>
        </div>

        <div class="header-controls">
          <div class="fullscreen-btn" title="全屏" @click="toggleFullscreen">
            <el-icon :size="18">
              <FullScreen />
            </el-icon>
          </div>
          <button class="close-btn" title="关闭" @click="handleClose">
            <el-icon :size="18">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024">
                <path fill="currentColor" d="M764.288 214.592 512 466.88 259.712 214.592a31.936 31.936 0 0 0-45.12 45.12L466.752 512 214.528 764.224a31.936 31.936 0 1 0 45.12 45.184L512 557.184l252.288 252.288a31.936 31.936 0 0 0 45.12-45.12L557.12 512.064l252.288-252.352a31.936 31.936 0 1 0-45.12-45.184z"/>
              </svg>
            </el-icon>
          </button>
        </div>
      </div>

      <!-- 弹窗内容 -->
      <div class="dialog-body">
        <div class="stream-container">
          <!-- 视频流显示区域 -->
          <div ref="videoAreaRef" class="video-area">
            <div v-loading="isLoading" class="stream-wrapper">
              <!-- 可交互的视频显示区域 -->
              <div
                v-if="device?.lan && isStreamActive"
                ref="interactiveAreaRef"
                class="interactive-video-container"
                tabindex="0"
                @contextmenu="handleVideoRightClick"
                @mousedown="handleMouseDown"
                @mouseup="handleMouseUp"
                @mousemove="handleMouseMove"
                @wheel="handleWheel"
                @keydown="handleKeyDown"
                @keyup="handleKeyUp"
                @paste="handlePaste"
                @mouseenter="handleMouseEnter"
                @mouseleave="handleMouseLeave"
              >
                <JMuxerDecoder
                  :device-id="device.ID"
                  :device-ip="device.lan"
                  :auto-start="true"
                  @connected="handleStreamConnected"
                  @disconnected="handleStreamDisconnected"
                  @error="handleStreamError"
                />
              </div>
              <div v-else-if="!isStreamActive && device?.lan" class="stream-placeholder">
                <el-icon class="stream-icon"><VideoCamera /></el-icon>
                <span>正在启动视频流...</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { deviceApi, type DeviceInfo } from '@/api/device'
import JMuxerDecoder from '@/views/decoders/JMuxerDecoder.vue'
import { FullScreen, Monitor, RefreshRight, Setting, VideoCamera } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, nextTick, ref, watch } from 'vue'

// Props
interface Props {
  visible: boolean
  device: DeviceInfo | null
}

const props = defineProps<Props>()

// Emits
const emit = defineEmits<{
  'update:visible': [value: boolean]
  close: []
}>()

// 响应式数据
const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value)
})

const videoAreaRef = ref<HTMLElement>()
const interactiveAreaRef = ref<HTMLElement>()
const isLoading = ref(false)
const isStarting = ref(false)
const isStopping = ref(false)
const isStreamActive = ref(false)
const isFullscreen = ref(false)

// 连接状态
const connectionStatus = ref<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected')
const connectionError = ref('')

// 鼠标和键盘控制相关
const displayRect = ref<DOMRect | null>(null)
const wsControl = ref<WebSocket | null>(null)
const mousePressed = ref(0)
const isControlEnabled = ref(false)

// Debug控制
const DEBUG = false  // 设为true启用详细日志
const DEBUG_MOUSE = false  // 设为true启用鼠标移动日志

// 日志工具
const log = (message: string, ...args: any[]) => {
  if (DEBUG) console.log(`[StreamDialog-${props.device?.ID}] ${message}`, ...args)
}

const warn = (message: string, ...args: any[]) => {
  console.warn(`[StreamDialog-${props.device?.ID}] ${message}`, ...args)
}

const error_log = (message: string, ...args: any[]) => {
  console.error(`[StreamDialog-${props.device?.ID}] ${message}`, ...args)
}

const debug = (message: string, ...args: any[]) => {
  if (DEBUG) console.debug(`[StreamDialog-${props.device?.ID}] ${message}`, ...args)
}

// 方法
const startStream = async () => {
  if (!props.device) return

  try {
    isStarting.value = true
    connectionStatus.value = 'connecting'

    // 调用agent的startstream接口
    await deviceApi.startStream(props.device.ID)

    // 启动成功后激活流
    isStreamActive.value = true

    // 启动控制WebSocket连接
    await startControlConnection()

    ElMessage.success('视频流启动成功')
  } catch (error) {
    error_log('启动视频流失败:', error)
    ElMessage.error('启动视频流失败')
    connectionStatus.value = 'error'
  } finally {
    isStarting.value = false
  }
}

// 启动控制WebSocket连接（直连agent）
const startControlConnection = async () => {
  if (!props.device?.lan) return

  try {
    // 直连agent的WebSocket控制接口
    const wsUrl = `ws://${props.device.lan}:50052/wscontrol`
    debug('🔗 启动控制WebSocket连接:', wsUrl)
    wsControl.value = new WebSocket(wsUrl)

    wsControl.value.onopen = () => {
      warn('✅ 控制WebSocket连接成功:', wsUrl)
      isControlEnabled.value = true
    }

    wsControl.value.onclose = (event) => {
      warn('🔌 控制WebSocket连接断开:', {
        code: event.code,
        reason: event.reason,
        wasClean: event.wasClean
      })
      isControlEnabled.value = false
    }

    wsControl.value.onerror = (error) => {
      error_log('❌ 控制WebSocket连接错误:', error)
      isControlEnabled.value = false
    }

    wsControl.value.onmessage = (event) => {
      debug('📨 收到控制响应消息:', event.data)
      try {
        const response = JSON.parse(event.data)
        debug('📨 解析控制响应:', {
          type: response.type,
          data: response.data,
          timestamp: response.timestamp ? new Date(response.timestamp).toISOString() : 'N/A'
        })
      } catch (e) {
        debug('📨 收到非JSON控制响应:', event.data)
      }
    }

    // 处理心跳消息
    wsControl.value.addEventListener('ping', () => {
      debug('💓 收到服务器心跳')
    })

    wsControl.value.addEventListener('pong', () => {
      debug('💓 心跳响应已发送')
    })
  } catch (error) {
    error_log('❌ 启动控制连接失败:', error)
  }
}

const stopStream = async () => {
  if (!props.device) return

  try {
    isStopping.value = true

    // 先停止前端的流
    isStreamActive.value = false
    connectionStatus.value = 'disconnected'

    // 关闭控制WebSocket连接
    stopControlConnection()

    // 调用agent的stopstream接口
    await deviceApi.stopStream(props.device.ID)

    ElMessage.success('视频流已停止')
  } catch (error) {
    error_log('停止视频流失败:', error)
    ElMessage.error('停止视频流失败')
  } finally {
    isStopping.value = false
  }
}

// 停止控制WebSocket连接
const stopControlConnection = () => {
  if (wsControl.value) {
    wsControl.value.close()
    wsControl.value = null
  }
  isControlEnabled.value = false
}

const handleClose = async () => {
  // 保存当前流状态
  const wasStreamActive = isStreamActive.value

  // 立即重置状态，避免显示启动提示
  isStreamActive.value = false
  connectionStatus.value = 'disconnected'

  // 立即关闭弹窗
  emit('close')

  // 如果之前正在串流，异步停止（不阻塞关闭）
  if (wasStreamActive) {
    stopStream().catch(error => {
      error_log('停止视频流失败:', error)
    })
  }
}

const handleOverlayClick = () => {
  // 点击遮罩层不关闭弹窗，保持原有行为
  // 如果需要点击遮罩关闭，可以调用 handleClose()
}

const toggleFullscreen = () => {
  if (!videoAreaRef.value) return

  if (!isFullscreen.value) {
    if (videoAreaRef.value.requestFullscreen) {
      videoAreaRef.value.requestFullscreen()
    }
  } else {
    if (document.exitFullscreen) {
      document.exitFullscreen()
    }
  }
}

// 获取显示区域尺寸
const getDimensions = () => {
  if (interactiveAreaRef.value) {
    displayRect.value = interactiveAreaRef.value.getBoundingClientRect()
  }
}

// 控制消息类型常量
const MSG_TYPES = {
  // 鼠标控制消息类型
  MOUSE_MOVE: 'MOUSE_MOVE',
  MOUSE_LEFT_CLICK: 'MOUSE_LEFT_CLICK',
  MOUSE_RIGHT_CLICK: 'MOUSE_RIGHT_CLICK',
  MOUSE_MIDDLE_CLICK: 'MOUSE_MIDDLE_CLICK',
  MOUSE_LEFT_DOWN: 'MOUSE_LEFT_DOWN',
  MOUSE_LEFT_UP: 'MOUSE_LEFT_UP',
  MOUSE_RIGHT_DOWN: 'MOUSE_RIGHT_DOWN',
  MOUSE_RIGHT_UP: 'MOUSE_RIGHT_UP',
  MOUSE_MIDDLE_DOWN: 'MOUSE_MIDDLE_DOWN',
  MOUSE_MIDDLE_UP: 'MOUSE_MIDDLE_UP',
  MOUSE_WHEEL_UP: 'MOUSE_WHEEL_UP',
  MOUSE_WHEEL_DOWN: 'MOUSE_WHEEL_DOWN',
  MOUSE_RESET: 'MOUSE_RESET',

  // 键盘控制消息类型
  KEY_DOWN: 'KEY_DOWN',
  KEY_UP: 'KEY_UP',
  KEY_PRESS: 'KEY_PRESS',
  KEY_COMBO: 'KEY_COMBO',

  // 剪贴板消息类型
  CLIPBOARD_PASTE: 'CLIPBOARD_PASTE',

  // 系统控制消息类型
  SYSTEM_DESKTOP: 'SYSTEM_DESKTOP',
  SYSTEM_TASKMANAGER: 'SYSTEM_TASKMANAGER',
  SYSTEM_REBOOT: 'SYSTEM_REBOOT'
}

// 发送WebSocket控制消息（新格式）
const sendControlMessage = (type: string, data: any = {}) => {
  if (wsControl.value && wsControl.value.readyState === WebSocket.OPEN) {
    const message = {
      type,
      data,
      timestamp: Date.now(),
      id: `msg_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    }
    debug('🎮 发送控制消息:', {
      type,
      data,
      messageId: message.id,
      timestamp: new Date(message.timestamp).toISOString()
    })
    wsControl.value.send(JSON.stringify(message))
  } else {
    warn('❌ 控制WebSocket未连接，无法发送消息:', type, data)
  }
}

// 发送旧格式控制消息（兼容性）
const sendLegacyControlMessage = (message: string) => {
  if (wsControl.value && wsControl.value.readyState === WebSocket.OPEN) {
    debug('🎮 发送旧格式控制消息:', message)
    wsControl.value.send(message)
  } else {
    warn('❌ 控制WebSocket未连接，无法发送消息:', message)
  }
}

// 鼠标事件处理
const handleVideoRightClick = (event: MouseEvent) => {
  event.preventDefault()
  if (!isControlEnabled.value || !displayRect.value) return

  const { x, y } = getDeviceCoordinates(event)
  debug('🖱️ 鼠标右键点击:', { x, y, clientX: event.clientX, clientY: event.clientY })
  sendControlMessage(MSG_TYPES.MOUSE_RIGHT_CLICK, { x, y })
}

const handleMouseDown = (event: MouseEvent) => {
  event.preventDefault()
  if (!isControlEnabled.value || !displayRect.value) return

  const { x, y } = getDeviceCoordinates(event)

  switch (event.button) {
    case 0: // 左键
      mousePressed.value = 1
      debug('🖱️ 鼠标左键按下:', { x, y, button: event.button })
      sendControlMessage(MSG_TYPES.MOUSE_LEFT_DOWN, { x, y })
      break
    case 1: // 中键
      mousePressed.value = 2
      debug('🖱️ 鼠标中键按下:', { x, y, button: event.button })
      sendControlMessage(MSG_TYPES.MOUSE_MIDDLE_DOWN, { x, y })
      break
    case 2: // 右键
      mousePressed.value = 4
      debug('🖱️ 鼠标右键按下:', { x, y, button: event.button })
      sendControlMessage(MSG_TYPES.MOUSE_RIGHT_DOWN, { x, y })
      break
  }
}

const handleMouseUp = (event: MouseEvent) => {
  event.preventDefault()
  if (!isControlEnabled.value || !displayRect.value) return

  const { x, y } = getDeviceCoordinates(event)

  switch (mousePressed.value) {
    case 1: // 左键
      debug('🖱️ 鼠标左键释放:', { x, y, previousPressed: mousePressed.value })
      sendControlMessage(MSG_TYPES.MOUSE_LEFT_UP, { x, y })
      break
    case 2: // 中键
      debug('🖱️ 鼠标中键释放:', { x, y, previousPressed: mousePressed.value })
      sendControlMessage(MSG_TYPES.MOUSE_MIDDLE_UP, { x, y })
      break
    case 4: // 右键
      debug('🖱️ 鼠标右键释放:', { x, y, previousPressed: mousePressed.value })
      sendControlMessage(MSG_TYPES.MOUSE_RIGHT_UP, { x, y })
      break
  }

  mousePressed.value = 0
}

const handleMouseMove = (event: MouseEvent) => {
  if (!isControlEnabled.value || !displayRect.value) return

  const { x, y } = getDeviceCoordinates(event)
  // 鼠标移动事件频率较高，仅在DEBUG_MOUSE为true时记录
  if (DEBUG_MOUSE) {
    debug('🖱️ 鼠标移动:', { x, y, clientX: event.clientX, clientY: event.clientY })
  }
  sendControlMessage(MSG_TYPES.MOUSE_MOVE, { x, y })
}

const handleWheel = (event: WheelEvent) => {
  event.preventDefault()
  if (!isControlEnabled.value || !displayRect.value) return

  const { x, y } = getDeviceCoordinates(event)
  const wheelType = event.deltaY > 0 ? MSG_TYPES.MOUSE_WHEEL_DOWN : MSG_TYPES.MOUSE_WHEEL_UP
  debug('🖱️ 鼠标滚轮:', { x, y, deltaY: event.deltaY, wheelType })

  sendControlMessage(wheelType, { x, y })
}

// 坐标转换：将浏览器坐标转换为设备坐标（考虑object-fit: contain产生的黑边）
const getDeviceCoordinates = (event: MouseEvent) => {
  if (!interactiveAreaRef.value) {
    warn('📐 交互容器未初始化，返回默认坐标')
    return { x: 0, y: 0 }
  }

  // 设备逻辑分辨率
  const deviceWidth = 1920
  const deviceHeight = 1080

  // 容器尺寸（交互区域尺寸）
  const containerRect = (displayRect.value ?? interactiveAreaRef.value.getBoundingClientRect())
  const containerWidth = containerRect.width
  const containerHeight = containerRect.height

  // 获取实际video元素及其内在分辨率
  const videoEl = interactiveAreaRef.value.querySelector('video') as HTMLVideoElement | null
  const sourceWidth = videoEl?.videoWidth || deviceWidth
  const sourceHeight = videoEl?.videoHeight || deviceHeight

  // 计算在object-fit: contain下，实际渲染的视频区域大小与偏移
  const sourceAspect = sourceWidth / sourceHeight
  const containerAspect = containerWidth / containerHeight

  let renderedWidth = containerWidth
  let renderedHeight = containerHeight
  let offsetX = 0
  let offsetY = 0

  if (containerAspect > sourceAspect) {
    // 容器更宽，按高度铺满，左右留黑边
    renderedHeight = containerHeight
    renderedWidth = renderedHeight * sourceAspect
    offsetX = (containerWidth - renderedWidth) / 2
    offsetY = 0
  } else {
    // 容器更窄，按宽度铺满，上下留黑边
    renderedWidth = containerWidth
    renderedHeight = renderedWidth / sourceAspect
    offsetX = 0
    offsetY = (containerHeight - renderedHeight) / 2
  }

  // 将事件坐标转换为相对渲染视频区域的坐标
  const localX = event.clientX - containerRect.left - offsetX
  const localY = event.clientY - containerRect.top - offsetY

  // 限制在渲染区域内（防止落在黑边上）
  const clampedLocalX = Math.max(0, Math.min(localX, renderedWidth))
  const clampedLocalY = Math.max(0, Math.min(localY, renderedHeight))

  // 映射到设备逻辑分辨率
  const x = Math.round((deviceWidth / renderedWidth) * clampedLocalX)
  const y = Math.round((deviceHeight / renderedHeight) * clampedLocalY)

  // 坐标转换详细信息，仅在DEBUG_MOUSE模式下记录
  if (DEBUG_MOUSE) {
    debug('📐 坐标转换(含letterbox):', {
      client: { x: event.clientX, y: event.clientY },
      container: {
        left: containerRect.left,
        top: containerRect.top,
        width: containerWidth,
        height: containerHeight
      },
      source: { width: sourceWidth, height: sourceHeight, aspect: sourceAspect.toFixed(4) },
      rendered: { width: renderedWidth, height: renderedHeight, offsetX, offsetY, aspect: (renderedWidth / renderedHeight).toFixed(4) },
      local: { x: localX, y: localY },
      clampedLocal: { x: clampedLocalX, y: clampedLocalY },
      device: { x, y },
      scale: {
        x: deviceWidth / renderedWidth,
        y: deviceHeight / renderedHeight
      }
    })
  }

  return { x, y }
}



// 键盘事件处理
const handleKeyDown = (event: KeyboardEvent) => {
  if (!isControlEnabled.value) return

  event.preventDefault()
  const keysym = convertKeyToGuacamole(event)
  if (keysym) {
    const keyStr = convertKeysymToString(keysym)
    debug('⌨️ 按键按下:', {
      key: event.key,
      code: event.code,
      keysym,
      keyStr,
      ctrlKey: event.ctrlKey,
      shiftKey: event.shiftKey,
      altKey: event.altKey
    })
    sendControlMessage(MSG_TYPES.KEY_DOWN, {
      key: keysym,
      keyStr: keyStr
    })
  } else {
    warn('⌨️ 无法转换按键:', { key: event.key, code: event.code })
  }
}

const handleKeyUp = (event: KeyboardEvent) => {
  if (!isControlEnabled.value) return

  event.preventDefault()
  const keysym = convertKeyToGuacamole(event)
  if (keysym) {
    const keyStr = convertKeysymToString(keysym)
    debug('⌨️ 按键释放:', {
      key: event.key,
      code: event.code,
      keysym,
      keyStr,
      ctrlKey: event.ctrlKey,
      shiftKey: event.shiftKey,
      altKey: event.altKey
    })
    sendControlMessage(MSG_TYPES.KEY_UP, {
      key: keysym,
      keyStr: keyStr
    })
  } else {
    warn('⌨️ 无法转换按键:', { key: event.key, code: event.code })
  }
}

const handlePaste = (event: ClipboardEvent) => {
  if (!isControlEnabled.value) return

  event.preventDefault()
  const pastedData = event.clipboardData?.getData('Text')
  if (pastedData && pastedData.length > 0) {
    debug('📋 粘贴操作:', {
      textLength: pastedData.length,
      preview: pastedData.substring(0, 50) + (pastedData.length > 50 ? '...' : '')
    })
    // 先发送Ctrl键释放
    sendControlMessage(MSG_TYPES.KEY_UP, { key: 65507, keyStr: 'ctrl' })
    // 发送粘贴内容
    sendControlMessage(MSG_TYPES.CLIPBOARD_PASTE, { text: pastedData })
  } else {
    warn('📋 粘贴操作失败: 无有效文本数据')
  }
}

const handleMouseEnter = () => {
  if (interactiveAreaRef.value) {
    interactiveAreaRef.value.focus()
    getDimensions()
  }
}

const handleMouseLeave = () => {
  // 重置鼠标状态
  if (isControlEnabled.value) {
    sendControlMessage(MSG_TYPES.MOUSE_RESET)
  }
}

// 简单的键盘码转换（基于重构前项目的实现）
const convertKeyToGuacamole = (event: KeyboardEvent): number | null => {
  // 这里实现基本的键盘码转换，可以根据需要扩展
  const keyMap: Record<string, number> = {
    'Backspace': 65288,
    'Tab': 65289,
    'Enter': 65293,
    'Shift': 65505,
    'Control': 65507,
    'Alt': 65513,
    'Escape': 65307,
    'Space': 32,
    'ArrowLeft': 65361,
    'ArrowUp': 65362,
    'ArrowRight': 65363,
    'ArrowDown': 65364,
    'Delete': 65535,
    'Home': 65360,
    'End': 65367,
    'PageUp': 65365,
    'PageDown': 65366,
  }

  // 特殊键
  if (keyMap[event.key]) {
    return keyMap[event.key]
  }

  // 字母和数字
  if (event.key.length === 1) {
    return event.key.charCodeAt(0)
  }

  // 功能键 F1-F12
  if (event.key.startsWith('F') && event.key.length <= 3) {
    const fNum = parseInt(event.key.substring(1))
    if (fNum >= 1 && fNum <= 12) {
      return 65469 + fNum
    }
  }

  return null
}

// 键码转字符串转换函数
const convertKeysymToString = (keysym: number): string => {
  // 基本字符（ASCII）
  if (keysym >= 32 && keysym <= 126) {
    return String.fromCharCode(keysym).toLowerCase()
  }

  // 特殊按键映射
  const keyMap: Record<number, string> = {
    65288: 'backspace',
    65289: 'tab',
    65293: 'enter',
    65505: 'shift',
    65507: 'ctrl',
    65513: 'alt',
    65307: 'esc',
    32: 'space',
    65361: 'left',
    65362: 'up',
    65363: 'right',
    65364: 'down',
    65535: 'delete',
    65360: 'home',
    65367: 'end',
    65365: 'pageup',
    65366: 'pagedown',
  }

  if (keyMap[keysym]) {
    return keyMap[keysym]
  }

  // 功能键 F1-F12
  if (keysym >= 65470 && keysym <= 65481) {
    return `f${keysym - 65469}`
  }

  return ''
}

// 设备操控方法
const showDesktop = () => {
  if (!isControlEnabled.value) {
    warn('🖥️ 控制未启用，无法执行显示桌面操作')
    return
  }
  debug('🖥️ 执行显示桌面操作')
  sendControlMessage(MSG_TYPES.SYSTEM_DESKTOP)
}

const openTaskManager = () => {
  if (!isControlEnabled.value) {
    warn('📊 控制未启用，无法执行打开任务管理器操作')
    return
  }
  debug('📊 执行打开任务管理器操作')
  sendControlMessage(MSG_TYPES.SYSTEM_TASKMANAGER)
}

const rebootDevice = async () => {
  try {
    debug('🔄 用户请求重启设备')
    await ElMessageBox.confirm('确定要重启设备吗？', '确认重启', {
      type: 'warning'
    })

    if (isControlEnabled.value) {
      debug('🔄 发送系统重启命令')
      sendControlMessage(MSG_TYPES.SYSTEM_REBOOT)
    } else {
      warn('🔄 控制未启用，无法执行重启操作')
    }

    ElMessage.success('重启指令已发送')

    // 断开连接
    await handleClose()
  } catch (error) {
    if (error !== 'cancel') {
      error_log('🔄 重启操作失败:', error)
      ElMessage.error('重启失败')
    } else {
      debug('🔄 用户取消重启操作')
    }
  }
}

// 流事件处理
const handleStreamConnected = () => {
  connectionStatus.value = 'connected'
  isLoading.value = false
  connectionError.value = ''

  // 获取显示区域尺寸
  nextTick(() => {
    getDimensions()
  })

  ElMessage.success('视频流连接成功')
}

const handleStreamDisconnected = () => {
  connectionStatus.value = 'disconnected'
  ElMessage.warning('视频流连接断开')
}

const handleStreamError = (message: string) => {
  connectionStatus.value = 'error'
  connectionError.value = message
  ElMessage.error(`视频流错误: ${message}`)
}

// 监听全屏状态变化
const handleFullscreenChange = () => {
  isFullscreen.value = !!document.fullscreenElement
}

// 监听弹窗显示状态
watch(() => props.visible, async (visible) => {
  if (visible) {
    // 弹窗打开时重置状态
    connectionStatus.value = 'disconnected'
    connectionError.value = ''
    isStreamActive.value = false
    isControlEnabled.value = false

    // 监听全屏事件
    document.addEventListener('fullscreenchange', handleFullscreenChange)

    // 自动开始串流（只要有设备信息就启动）
    if (props.device?.lan) {
      await startStream()
    }
  } else {
    // 弹窗关闭时清理
    document.removeEventListener('fullscreenchange', handleFullscreenChange)
    stopControlConnection()
    isFullscreen.value = false
    isStreamActive.value = false
    isControlEnabled.value = false
    displayRect.value = null
    mousePressed.value = 0
  }
})
</script>

<style scoped>
/* 自定义弹窗遮罩层 */
.stream-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  padding: 20px;
  box-sizing: border-box;
}

/* 自定义弹窗容器 */
.stream-dialog-container {
  /* 计算合适的尺寸：考虑头部高度，视频区域按16:9比例 */
  --header-height: 40px;
  --video-height: min(85vh, 1000px);
  --total-height: calc(var(--video-height) + var(--header-height));

  height: var(--total-height);
  width: calc(var(--video-height) * 16 / 9);
  max-width: 95vw;
  background: transparent;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  display: flex;
  flex-direction: column;
}

/* 对话框头部 */
.dialog-header {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  background: rgba(0, 0, 0, 0.7);
  backdrop-filter: blur(4px);
  color: white;
  z-index: 10;
  min-height: 40px;
  box-sizing: border-box;
  flex-wrap: wrap;
  gap: 8px;
}

.dialog-title {
  font-size: 14px;
  font-weight: 500;
  color: white;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.5);
  flex-shrink: 0;
}

/* 头部控制按钮区域 */
.header-control-buttons {
  display: flex;
  gap: 6px;
  align-items: center;
  flex: 1;
  justify-content: center;
}

.header-control-buttons .el-button {
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  color: white;
  transition: all 0.2s ease;
  font-size: 12px;
  padding: 4px 8px;
  height: 28px;
}

.header-control-buttons .el-button:hover {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.4);
}

.header-control-buttons .el-button.is-type-warning {
  background: rgba(245, 108, 108, 0.2);
  border-color: rgba(245, 108, 108, 0.4);
}

.header-control-buttons .el-button.is-type-warning:hover {
  background: rgba(245, 108, 108, 0.3);
  border-color: rgba(245, 108, 108, 0.6);
}

/* 头部控制按钮 */
.header-controls {
  display: flex;
  align-items: center;
  gap: 6px;
}

.fullscreen-btn,
.close-btn {
  background: rgba(255, 255, 255, 0.1);
  border-radius: 4px;
  padding: 4px;
  cursor: pointer;
  color: white;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(255, 255, 255, 0.2);
  width: 28px;
  height: 28px;
}

.fullscreen-btn:hover,
.close-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.4);
}

/* 对话框主体 */
.dialog-body {
  flex: 1;
  padding: 0;
  overflow: hidden;
  position: relative;
}

.stream-container {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.video-area {
  width: 100%;
  height: 100%;
  position: relative;
  background: transparent;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border-radius: 0 0 8px 8px;
}



.stream-wrapper {
  width: 100%;
  height: 100%;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
}

/* 可交互的视频容器 */
.interactive-video-container {
  width: 100%;
  height: 100%;
  position: relative;
  outline: none;
  cursor: crosshair;
}

.interactive-video-container:focus {
  outline: 2px solid rgba(64, 158, 255, 0.5);
  outline-offset: -2px;
}

/* 确保JMuxerDecoder组件适应容器并设置透明背景 */
.stream-wrapper :deep(.jmuxer-decoder) {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  background: transparent !important;
}

.stream-wrapper :deep(.video-wrapper) {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  background: transparent !important;
  border-radius: 0 !important;
}

.stream-wrapper :deep(.video-player) {
  width: 100%;
  height: 100%;
  max-width: 100%;
  max-height: 100%;
  object-fit: contain; /* 保持宽高比，适应容器 */
  background: transparent !important;
}

/* 覆盖JMuxerDecoder的连接状态样式 */
.stream-wrapper :deep(.connection-status) {
  color: white !important;
  font-size: 16px !important;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.8) !important;
  background: rgba(0, 0, 0, 0.3) !important;
  backdrop-filter: blur(4px) !important;
  border-radius: 8px !important;
  padding: 16px 24px !important;
  z-index: 10 !important;
}

.stream-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  color: white;
  font-size: 16px;
  text-shadow: 0 2px 4px rgba(0, 0, 0, 0.8);
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(4px);
  border-radius: 8px;
  padding: 32px;
  margin: 20px;
}

.stream-icon {
  font-size: 64px;
  color: white;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.8));
}





/* 响应式设计 */
@media (max-width: 768px) {
  .stream-dialog-container {
    /* 平板端：考虑头部高度的视频区域尺寸 */
    --video-height: min(70vh, 600px);
    --total-height: calc(var(--video-height) + var(--header-height));
    height: var(--total-height);
    width: calc(var(--video-height) * 16 / 9);
    max-width: 95vw;
  }
}

@media (max-width: 480px) {
  .stream-dialog-container {
    /* 手机端：进一步缩小但保持比例 */
    --video-height: min(60vh, 400px);
    --total-height: calc(var(--video-height) + var(--header-height));
    height: var(--total-height);
    width: calc(var(--video-height) * 16 / 9);
    max-width: 95vw;
  }

  .dialog-header {
    padding: 6px 12px;
    height: 36px;
  }

  .dialog-title {
    font-size: 12px;
  }

  .stream-dialog-container {
    --header-height: 36px;
  }
}
</style>
