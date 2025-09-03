<template>
  <div class="webcodecs-decoder">
    <div class="video-wrapper">
      <canvas 
        ref="canvasElement"
        class="video-canvas"
      />
      <div v-if="!connected" class="connection-status">
        连接中...
      </div>
      <div v-if="error" class="error-message">
        {{ error }}
      </div>
      <div v-if="!isSupported" class="error-message">
        当前浏览器不支持WebCodecs API
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'

interface Props {
  deviceId: string | number
  deviceIp?: string
  autoStart?: boolean
  width?: number
  height?: number
}

const props = withDefaults(defineProps<Props>(), {
  autoStart: true,
  width: 960,
  height: 540
})

const emit = defineEmits<{
  connected: []
  disconnected: []
  error: [message: string]
}>()

// 响应式数据
const connected = ref(false)
const error = ref('')
const canvasElement = ref<HTMLCanvasElement>()

// WebCodecs 支持检测
const isSupported = computed(() => {
  return 'VideoDecoder' in window && 'VideoFrame' in window
})

// WebSocket 和解码器
let wsVideo: WebSocket | null = null
let decoder: any = null
let ctx: CanvasRenderingContext2D | null = null

const wsUrl = computed(() => {
  if (window.location.host.startsWith('192.168') || window.location.host.startsWith('localhost')) {
    return `ws://${props.deviceIp}:50052/wsstream`
  }
  
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/api/devices/${props.deviceId}/stream`
})

// 初始化WebCodecs解码器
const initDecoder = () => {
  if (!isSupported.value) {
    error.value = '当前浏览器不支持WebCodecs API'
    emit('error', error.value)
    return
  }

  try {
    // WebCodecs解码器配置
    const config = {
      codec: 'avc1.42E01E', // H.264 Baseline Profile
      codedWidth: props.width,
      codedHeight: props.height,
    }

    decoder = new (window as any).VideoDecoder({
      output: (frame: any) => {
        // 渲染帧到canvas
        if (ctx && canvasElement.value) {
          ctx.drawImage(frame, 0, 0, canvasElement.value.width, canvasElement.value.height)
          frame.close()
        }
      },
      error: (err: Error) => {
        console.error('WebCodecs decoder error:', err)
        error.value = `解码错误: ${err.message}`
        emit('error', error.value)
      }
    })

    decoder.configure(config)
    console.log('WebCodecs decoder initialized')
  } catch (err) {
    console.error('Failed to initialize WebCodecs decoder:', err)
    error.value = 'WebCodecs解码器初始化失败'
    emit('error', error.value)
  }
}

// 连接WebSocket
const connectWebSocket = () => {
  try {
    console.log('Connecting to:', wsUrl.value)
    
    wsVideo = new WebSocket(wsUrl.value)
    wsVideo.binaryType = 'arraybuffer'
    
    wsVideo.addEventListener('open', () => {
      console.log('WebSocket connected')
      connected.value = true
      error.value = ''
      emit('connected')
    })
    
    wsVideo.addEventListener('message', (event) => {
      if (decoder && event.data) {
        try {
          // 解析H.264数据并送入解码器
          const chunk = new (window as any).EncodedVideoChunk({
            type: 'key', // 简化处理，实际需要检测帧类型
            timestamp: performance.now() * 1000,
            data: event.data
          })
          
          decoder.decode(chunk)
        } catch (err) {
          console.error('Failed to decode video data:', err)
        }
      }
    })
    
    wsVideo.addEventListener('error', (err) => {
      console.error('WebSocket error:', err)
      error.value = 'WebSocket连接错误'
      connected.value = false
      emit('error', error.value)
    })
    
    wsVideo.addEventListener('close', (event) => {
      console.log('WebSocket closed:', event.code, event.reason)
      connected.value = false
      emit('disconnected')
      
      if (event.code !== 1000) {
        error.value = `WebSocket连接断开: ${event.reason || '未知原因'}`
        emit('error', error.value)
      }
    })
    
  } catch (err) {
    console.error('Failed to connect WebSocket:', err)
    error.value = (err as Error).message
    emit('error', error.value)
  }
}

// 开始流
const startStream = async () => {
  if (!isSupported.value) {
    error.value = '当前浏览器不支持WebCodecs API'
    emit('error', error.value)
    return
  }

  try {
    error.value = ''
    
    await nextTick()
    
    // 初始化canvas
    if (canvasElement.value) {
      canvasElement.value.width = props.width
      canvasElement.value.height = props.height
      ctx = canvasElement.value.getContext('2d')
    }
    
    // 初始化解码器
    initDecoder()
    
    // 建立WebSocket连接
    connectWebSocket()
    
  } catch (err) {
    console.error('Failed to start stream:', err)
    error.value = (err as Error).message
    emit('error', error.value)
  }
}

// 停止流
const stopStream = () => {
  try {
    if (wsVideo) {
      wsVideo.close()
      wsVideo = null
    }
    
    if (decoder) {
      try {
        decoder.close()
      } catch (err) {
        console.error('Failed to close decoder:', err)
      }
      decoder = null
    }
    
    connected.value = false
    error.value = ''
    
    console.log('WebCodecs stream stopped')
    emit('disconnected')
    
  } catch (err) {
    console.error('Failed to stop stream:', err)
    error.value = (err as Error).message
    emit('error', error.value)
  }
}

// 清理资源
const cleanup = () => {
  stopStream()
}

// 暴露方法给父组件
defineExpose({
  startStream,
  stopStream,
  connected,
  error,
  isSupported
})

// 生命周期
onMounted(() => {
  if (props.autoStart && isSupported.value) {
    startStream()
  }
})

onBeforeUnmount(() => {
  cleanup()
})
</script>

<style scoped>
.webcodecs-decoder {
  width: 100%;
  height: 100%;
}

.video-wrapper {
  position: relative;
  width: 100%;
  height: 100%;
  background: #000;
  border-radius: 4px;
  overflow: hidden;
}

.video-canvas {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.connection-status {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #fff;
  font-size: 16px;
  z-index: 10;
}

.error-message {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  color: #f56c6c;
  font-size: 14px;
  text-align: center;
  z-index: 10;
  background: rgba(0, 0, 0, 0.8);
  padding: 8px 16px;
  border-radius: 4px;
}
</style>
