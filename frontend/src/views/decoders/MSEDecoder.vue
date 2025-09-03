<template>
  <div class="mse-decoder">
    <div class="video-wrapper">
      <video 
        ref="videoElement"
        autoplay 
        muted 
        playsinline
        class="video-player"
      />
      <div v-if="!connected" class="connection-status">
        连接中...
      </div>
      <div v-if="error" class="error-message">
        {{ error }}
      </div>
      <div v-if="!isSupported" class="error-message">
        当前浏览器不支持Media Source Extensions
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
const videoElement = ref<HTMLVideoElement>()

// MSE 支持检测
const isSupported = computed(() => {
  return 'MediaSource' in window && MediaSource.isTypeSupported('video/mp4; codecs="avc1.42E01E"')
})

// MSE 相关
let wsVideo: WebSocket | null = null
let mediaSource: MediaSource | null = null
let sourceBuffer: SourceBuffer | null = null
let queue: Uint8Array[] = []

const wsUrl = computed(() => {
  if (window.location.host.startsWith('192.168') || window.location.host.startsWith('localhost')) {
    return `ws://${props.deviceIp}:50052/wsstream`
  }
  
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/api/devices/${props.deviceId}/stream`
})

// 初始化MSE
const initMSE = () => {
  if (!isSupported.value) {
    error.value = '当前浏览器不支持Media Source Extensions'
    emit('error', error.value)
    return
  }

  try {
    mediaSource = new MediaSource()
    
    mediaSource.addEventListener('sourceopen', () => {
      console.log('MediaSource opened')
      
      try {
        sourceBuffer = mediaSource!.addSourceBuffer('video/mp4; codecs="avc1.42E01E"')
        
        sourceBuffer.addEventListener('updateend', () => {
          // 处理队列中的下一个数据块
          if (queue.length > 0 && !sourceBuffer!.updating) {
            const nextChunk = queue.shift()
            if (nextChunk) {
              sourceBuffer!.appendBuffer(nextChunk)
            }
          }
        })
        
        sourceBuffer.addEventListener('error', (err) => {
          console.error('SourceBuffer error:', err)
          error.value = 'SourceBuffer错误'
          emit('error', error.value)
        })
        
      } catch (err) {
        console.error('Failed to add source buffer:', err)
        error.value = 'SourceBuffer创建失败'
        emit('error', error.value)
      }
    })
    
    mediaSource.addEventListener('sourceended', () => {
      console.log('MediaSource ended')
    })
    
    mediaSource.addEventListener('error', (err) => {
      console.error('MediaSource error:', err)
      error.value = 'MediaSource错误'
      emit('error', error.value)
    })
    
    // 设置video元素的src
    if (videoElement.value) {
      videoElement.value.src = URL.createObjectURL(mediaSource)
    }
    
    console.log('MSE initialized')
  } catch (err) {
    console.error('Failed to initialize MSE:', err)
    error.value = 'MSE初始化失败'
    emit('error', error.value)
  }
}

// 添加数据到SourceBuffer
const appendBuffer = (data: Uint8Array) => {
  if (!sourceBuffer) {
    console.warn('SourceBuffer not ready')
    return
  }
  
  if (sourceBuffer.updating) {
    // 如果正在更新，加入队列
    queue.push(data)
  } else {
    try {
      sourceBuffer.appendBuffer(data)
    } catch (err) {
      console.error('Failed to append buffer:', err)
      // 加入队列重试
      queue.push(data)
    }
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
      if (event.data) {
        try {
          const uint8Array = new Uint8Array(event.data)
          
          // 添加调试信息
          const dataSize = event.data.byteLength
          console.log(`Received data: ${dataSize} bytes`)
          
          // 将数据添加到SourceBuffer
          appendBuffer(uint8Array)
          
        } catch (err) {
          console.error('Failed to process video data:', err)
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
    error.value = '当前浏览器不支持Media Source Extensions'
    emit('error', error.value)
    return
  }

  try {
    error.value = ''
    
    await nextTick()
    
    // 初始化MSE
    initMSE()
    
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
    
    if (mediaSource && mediaSource.readyState === 'open') {
      try {
        if (sourceBuffer) {
          mediaSource.removeSourceBuffer(sourceBuffer)
          sourceBuffer = null
        }
        mediaSource.endOfStream()
      } catch (err) {
        console.error('Failed to close MediaSource:', err)
      }
    }
    
    if (videoElement.value) {
      videoElement.value.src = ''
    }
    
    queue = []
    mediaSource = null
    connected.value = false
    error.value = ''
    
    console.log('MSE stream stopped')
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
.mse-decoder {
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

.video-player {
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
