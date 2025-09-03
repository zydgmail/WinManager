<template>
  <div class="jmuxer-decoder">
    <div class="video-wrapper">
      <video
        :id="videoId"
        ref="videoElement"
        autoplay
        muted
        playsinline
        class="video-player"
      />
      <div v-if="!connected && !error" class="connection-status">
        JMuxer连接中...
      </div>
      <div v-if="error" class="error-message">
        {{ error }}
      </div>
      <!-- Stalled/Reconnect Overlay -->
      <div v-if="showStalledOverlay" class="stalled-overlay">
        <div class="stalled-content">
          <div class="stalled-message">
            {{ stalled ? '画面已停止' : '无画面显示' }}
          </div>
          <button class="reconnect-btn" @click="reconnectStream">
            重新连接
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import JMuxer from 'jmuxer'
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'

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
const stalled = ref(false)
const hasVideo = ref(false)
const videoElement = ref<HTMLVideoElement>()
const startupGrace = ref(false)

// Debug控制
const DEBUG = false  // 设为true启用详细日志

// 日志工具
const log = (message: string, ...args: any[]) => {
  if (DEBUG) console.log(`[JMuxer-${props.deviceId}] ${message}`, ...args)
}

const warn = (message: string, ...args: any[]) => {
  console.warn(`[JMuxer-${props.deviceId}] ${message}`, ...args)
}

const error_log = (message: string, ...args: any[]) => {
  console.error(`[JMuxer-${props.deviceId}] ${message}`, ...args)
}

// JMuxer 实例
let jmuxer: any = null
let wsVideo: WebSocket | null = null

// 帧计数和性能监控
let frameCount = 0
let lastFrameTime = 0
let bufferHealthCheck: NodeJS.Timeout | null = null
let videoElementCheck: NodeJS.Timeout | null = null
let lastVideoCurrentTime = 0
let stuckDetectionCount = 0

// 动态统计数据
let frameRateHistory: number[] = [] // 最近的帧率历史
let lastFrameRateCheck = 0
let avgFrameInterval = 50 // 统一使用20fps = 50ms间隔，与agent保持一致
let frameIntervalHistory: number[] = [] // 帧间隔历史
let statisticsCounter = 0 // 统计计数器，减少计算频率
let jmuxerHealthCheck: NodeJS.Timeout | null = null // JMuxer健康检查
let lastBufferedEnd = 0 // 上次缓冲区结束时间
let preventiveRestart: NodeJS.Timeout | null = null // 预防性重启定时器

// 页面可见性检测
let visibilityCheck: NodeJS.Timeout | null = null
let isPageVisible = true
let wasHiddenLongTime = false

// 卡住检测
let stallDetectionTimer: NodeJS.Timeout | null = null
let lastVideoProgressTime = Date.now()
let stallRecoveryTimer: NodeJS.Timeout | null = null
let stallRecoveryAttempts = 0

// 启动遮罩宽限期
let startupGraceTimer: NodeJS.Timeout | null = null

// 计算属性
const videoId = computed(() => `jmuxer-video-${props.deviceId}`)
const showStalledOverlay = computed(() => {
  return connected.value && !error.value && !startupGrace.value && (stalled.value || !hasVideo.value)
})

const wsUrl = computed(() => {
  // 检查设备IP是否可用
  if (!props.deviceIp) {
    warn('Device IP not available yet')
    return ''
  }

  // 根据重构前项目的配置构建WebSocket URL
  if (window.location.host.startsWith('192.168') || window.location.host.startsWith('localhost')) {
    // 本地调试环境，直接连接到agent（与重构前项目保持一致）
    return `ws://${props.deviceIp}:50052/wsstream`
  }

  // 生产环境通过后端代理
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/api/devices/${props.deviceId}/stream`
})

// 页面可见性处理
const handleVisibilityChange = () => {
  isPageVisible = !document.hidden
  
  if (!isPageVisible) {
    log('页面隐藏，开始计时')
    wasHiddenLongTime = false
    // 页面隐藏时，停止视频检查减少资源消耗
    if (visibilityCheck) clearTimeout(visibilityCheck)
    visibilityCheck = setTimeout(() => {
      wasHiddenLongTime = true
      log('页面长时间隐藏，标记需要重连')
    }, 30000) // 30秒后标记为长时间隐藏
  } else {
    log('页面显示')
    if (visibilityCheck) clearTimeout(visibilityCheck)
    // 如果长时间隐藏后返回，直接重连以丢弃积压帧
    if (wasHiddenLongTime && connected.value) {
      warn('页面长时间隐藏后返回，重新连接流媒体')
      setTimeout(() => reconnectStream(), 300)
    }
    wasHiddenLongTime = false
  }
}

// 卡住检测
const startStallDetection = () => {
  if (stallDetectionTimer) clearInterval(stallDetectionTimer)
  
  stallDetectionTimer = setInterval(() => {
    const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
    if (!videoEl || !connected.value) return
    
    const now = Date.now()
    const currentTime = videoEl.currentTime
    const hasProgressed = Math.abs(currentTime - lastVideoCurrentTime) > 0.01
    
    if (hasProgressed) {
      lastVideoProgressTime = now
      lastVideoCurrentTime = currentTime
      if (stalled.value) {
        log('视频恢复播放')
        stalled.value = false
      }
      hasVideo.value = true
      // 恢复后清理恢复定时器
      if (stallRecoveryTimer) {
        clearTimeout(stallRecoveryTimer)
        stallRecoveryTimer = null
      }
      stallRecoveryAttempts = 0
    } else {
      // 检查是否真的卡住了
      const stallTime = now - lastVideoProgressTime
      if (stallTime > 10000 && frameCount > 100) { // 10秒无进度且有帧数据
        if (!stalled.value) {
          warn('检测到视频卡住，显示重连按钮')
          stalled.value = true
          hasVideo.value = false
          // 启动分级自恢复：先重建解码器，仍不恢复则重连
          if (!stallRecoveryTimer) {
            const scheduleNext = () => {
              if (!stalled.value) {
                if (stallRecoveryTimer) clearTimeout(stallRecoveryTimer)
                stallRecoveryTimer = null
                return
              }
              if (stallRecoveryAttempts === 0) {
                warn('卡住后自动重建解码器')
                reinitializeJMuxer()
              } else {
                warn('卡住持续，自动重连流媒体')
                reconnectStream()
              }
              stallRecoveryAttempts++
              if (stallRecoveryAttempts < 2) {
                stallRecoveryTimer = setTimeout(scheduleNext, 10000)
              } else {
                // 达到最大尝试次数后停止自动重试，保留手动按钮
                stallRecoveryTimer = null
              }
            }
            stallRecoveryTimer = setTimeout(scheduleNext, 1000)
          }
        }
      }
    }
  }, 2000)
}

// 重连流媒体
const reconnectStream = async () => {
  try {
    log('用户手动重连流媒体')
    stalled.value = false
    hasVideo.value = false
    
    // 先停止当前流
    stopStream()
    
    // 等待一下再重新连接
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // 重新开始流
    await startStream()
  } catch (err) {
    error_log('重连失败:', err)
    error.value = '重连失败，请稍后再试'
  }
}

// Video元素播放状态检查
const startVideoElementCheck = () => {
  if (videoElementCheck) {
    clearInterval(videoElementCheck)
  }

  videoElementCheck = setInterval(() => {
    const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
    if (!videoEl) return

    const currentTime = videoEl.currentTime
    const timeDiff = currentTime - lastVideoCurrentTime

    // 检查Video元素是否需要启动播放
    if (frameCount > 100 && videoEl.readyState === 0 && videoEl.paused) {
      warn(`Video元素未开始播放，尝试启动: readyState=${videoEl.readyState}`)

      // 尝试启动播放
      videoEl.play().catch(err => {
        warn(`自动播放失败:`, err)
      })
    }

    // 检查video元素是否卡住（currentTime不再增长）
    if (frameCount > 2000 && videoEl.readyState >= 2 && Math.abs(timeDiff) < 0.1) {
      stuckDetectionCount++
      log(`Video元素可能卡住 #${stuckDetectionCount}: currentTime=${currentTime}, 帧数=${frameCount}`)

      // 连续8次检测到卡住，尝试恢复（进一步增加容错性，减少误判）
      if (stuckDetectionCount >= 8) {
        warn(`检测到Video元素卡住，尝试恢复`)
        handleVideoStuck()
      }
    } else {
      stuckDetectionCount = 0 // 重置计数
    }

    lastVideoCurrentTime = currentTime

    // 减少状态日志频率
    if (frameCount % 2000 === 0 && frameCount > 0) {
      log(`Video状态检查: readyState=${videoEl.readyState}, currentTime=${currentTime.toFixed(2)}, paused=${videoEl.paused}`)
    }

  }, 5000) // 减少到每5秒检查一次，降低CPU负担
}

// 处理Video卡住问题 - 更保守的恢复策略
const handleVideoStuck = () => {
  const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
  if (!videoEl) return

  log(`尝试恢复卡住的Video元素`)

  try {
    // 方法1: 尝试轻微调整currentTime来触发播放
    const currentTime = videoEl.currentTime
    videoEl.currentTime = currentTime + 0.01

    // 方法2: 确保视频在播放状态
    if (videoEl.paused) {
      videoEl.play().catch(err => {
        warn(`重新播放失败:`, err)
      })
    }

    // 重置卡住检测计数，给系统时间恢复
    stuckDetectionCount = 0

    // 方法3: 只有在极端情况下才重新初始化（延长等待时间）
    setTimeout(() => {
      // 再次检查是否真的卡住了
      const newCurrentTime = videoEl.currentTime
      if (Math.abs(newCurrentTime - currentTime) < 0.01 && stuckDetectionCount >= 5) {
        warn(`Video确实卡住，重新初始化解码器`)
        reinitializeJMuxer()
      }
    }, 10000) // 延长到10秒后再检查

  } catch (err) {
    error_log(`恢复Video元素时出错:`, err)
    // 即使出错也不立即重新初始化，给系统更多时间
    setTimeout(() => reinitializeJMuxer(), 5000)
  }
}

// 分析帧率和性能统计
const analyzeFramePerformance = () => {
  const now = Date.now()

  // 计算当前帧率（每5秒统计一次）
  if (now - lastFrameRateCheck >= 5000) {
    const timeDiff = now - lastFrameRateCheck
    const frameDiff = frameCount - (frameRateHistory.length > 0 ? frameRateHistory[frameRateHistory.length - 1] : 0)
    const currentFPS = (frameDiff / timeDiff) * 1000

    frameRateHistory.push(frameCount)
    if (frameRateHistory.length > 10) {
      frameRateHistory.shift() // 只保留最近10次记录
    }

    lastFrameRateCheck = now

    // 检测帧率异常（基于20fps调整阈值）
    if (currentFPS < 15 && frameCount > 2000) { // 帧率低于15fps才认为异常
      warn(`帧率异常: ${currentFPS.toFixed(1)}fps (期望20fps)，总帧数: ${frameCount}`)

      // 如果连续多次帧率异常，可能需要重新初始化
      if (frameRateHistory.length >= 3) {
        const recentFrameRates = frameRateHistory.slice(-3).map((count, index, arr) => {
          if (index === 0) return 20 // 调整为实际的20fps
          const prevCount = arr[index - 1]
          return ((count - prevCount) / 5000) * 1000
        })

        const avgRecentFPS = recentFrameRates.reduce((a, b) => a + b, 0) / recentFrameRates.length
        if (avgRecentFPS < 10) { // 平均帧率低于10fps时重新初始化
          error_log(`持续帧率过低: ${avgRecentFPS.toFixed(1)}fps，尝试重新初始化`)
          reinitializeJMuxer()
          return
        }
      }
    }

    log(`帧率统计: ${currentFPS.toFixed(1)}fps，总帧数: ${frameCount}`)
  }
}

// 缓冲区健康检查
const startBufferHealthCheck = () => {
  if (bufferHealthCheck) {
    clearInterval(bufferHealthCheck)
  }

  bufferHealthCheck = setInterval(() => {
    const now = Date.now()
    const timeSinceLastFrame = now - lastFrameTime

    // 动态检测：如果帧间隔异常长，可能卡住了
    const expectedMaxInterval = avgFrameInterval * 20 // 基于50ms调整，约1000ms
    if (timeSinceLastFrame > expectedMaxInterval && frameCount > 2000) {
      warn(`检测到帧间隔异常: ${timeSinceLastFrame}ms (期望<${expectedMaxInterval}ms)，总帧数: ${frameCount}`)

      // 分析最近的帧间隔趋势
      if (frameIntervalHistory.length >= 10) {
        const avgRecentInterval = frameIntervalHistory.slice(-10).reduce((a, b) => a + b, 0) / 10
        if (avgRecentInterval > avgFrameInterval * 4) { // 基于50ms，约200ms阈值
          error_log(`帧间隔持续异常，平均间隔: ${avgRecentInterval.toFixed(1)}ms (期望50ms)，重新初始化解码器`)
          reinitializeJMuxer()
          return
        }
      }

      // 只有在真正严重的情况下才重新初始化
      if (timeSinceLastFrame > expectedMaxInterval * 2) {
        reinitializeJMuxer()
      }
    }

    // 执行帧率分析
    analyzeFramePerformance()

  }, 10000) // 减少到每10秒检查一次，降低CPU负担
}

// JMuxer健康检查 - 检测JMuxer是否停止工作
const startJMuxerHealthCheck = () => {
  if (jmuxerHealthCheck) {
    clearInterval(jmuxerHealthCheck)
  }

  jmuxerHealthCheck = setInterval(() => {
    const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
    if (!videoEl) return

    const currentBufferedEnd = videoEl.buffered.length > 0 ? videoEl.buffered.end(0) : 0
    const bufferedGrowth = currentBufferedEnd - lastBufferedEnd

    // 检查JMuxer是否停止工作的关键指标
    if (frameCount > 1000) {
      // 1. 缓冲区完全空了但还在接收数据
      if (videoEl.buffered.length === 0 && frameCount > lastFrameRateCheck + 100) {
        error_log(`JMuxer异常: 缓冲区空但仍接收数据，帧数=${frameCount}`)
        reinitializeJMuxer()
        return
      }

      // 2. 缓冲区长时间不增长但还在接收数据
      if (bufferedGrowth < 0.1 && frameCount > lastFrameRateCheck + 150) {
        error_log(`JMuxer异常: 缓冲区停止增长，增长=${bufferedGrowth.toFixed(2)}s，帧数=${frameCount}`)
        reinitializeJMuxer()
        return
      }

      // 3. Video元素readyState异常
      if (videoEl.readyState < 2 && frameCount > 2000) {
        error_log(`JMuxer异常: Video readyState=${videoEl.readyState}，帧数=${frameCount}`)
        reinitializeJMuxer()
        return
      }
    }

    lastBufferedEnd = currentBufferedEnd

    // 记录健康状态
    if (frameCount % 2000 === 0 && frameCount > 0) {
      log(`JMuxer健康检查: 缓冲区=${currentBufferedEnd.toFixed(2)}s, 增长=${bufferedGrowth.toFixed(2)}s, readyState=${videoEl.readyState}`)
    }

  }, 3000) // 每3秒检查一次JMuxer健康状态
}

// 预防性重启机制 - 更保守的策略
const startPreventiveRestart = () => {
  if (preventiveRestart) {
    clearTimeout(preventiveRestart)
  }

  // 每30分钟预防性重新初始化JMuxer，减少不必要的重启
  preventiveRestart = setTimeout(() => {
    // 只有在运行足够长时间且没有其他问题时才重启
    if (frameCount > 20000 && stuckDetectionCount === 0) {
      log(`预防性重新初始化JMuxer，当前帧数: ${frameCount}`)
      reinitializeJMuxer()
    }

    // 递归设置下一次重启
    startPreventiveRestart()
  }, 30 * 60 * 1000) // 30分钟
}

// 播放状态检查 - 确保video元素正常播放
let playbackCheck: NodeJS.Timeout | null = null
const startPlaybackCheck = () => {
  if (playbackCheck) {
    clearInterval(playbackCheck)
  }

  playbackCheck = setInterval(() => {
    const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
    if (!videoEl) return

    // 检查播放状态
    if (videoEl.paused && videoEl.readyState >= 2) {
      warn(`Video元素暂停，尝试恢复播放`)
      videoEl.play().catch(err => {
        warn(`恢复播放失败:`, err)
      })
    }

    // 检查是否有数据但没有播放
    if (videoEl.readyState >= 2 && videoEl.buffered.length > 0 && videoEl.currentTime === 0) {
      warn(`Video有数据但未播放，强制开始播放`)
      videoEl.currentTime = 0.01 // 轻微调整时间触发播放
      videoEl.play().catch(err => {
        warn(`强制播放失败:`, err)
      })
    }
  }, 2000) // 每2秒检查一次
}

// 重新初始化JMuxer - 改进版本，减少黑屏问题
let isReinitializing = false
const reinitializeJMuxer = () => {
  try {
    // 防止重复重新初始化
    if (isReinitializing) {
      log(`重新初始化已在进行中，跳过`)
      return
    }

    isReinitializing = true
    warn(`开始重新初始化解码器`)

    // 停止所有检查定时器，避免在重新初始化过程中触发更多问题
    if (bufferHealthCheck) {
      clearInterval(bufferHealthCheck)
      bufferHealthCheck = null
    }
    if (videoElementCheck) {
      clearInterval(videoElementCheck)
      videoElementCheck = null
    }
    if (jmuxerHealthCheck) {
      clearInterval(jmuxerHealthCheck)
      jmuxerHealthCheck = null
    }

    // 保存当前video元素状态
    const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
    const wasPlaying = videoEl && !videoEl.paused

    // 销毁现有的JMuxer
    if (jmuxer) {
      try {
        jmuxer.destroy()
      } catch (err) {
        warn(`销毁旧解码器时出现警告:`, err)
      }
      jmuxer = null
    }

    // 重置统计数据
    frameCount = 0
    lastFrameTime = Date.now()
    lastVideoCurrentTime = 0
    stuckDetectionCount = 0
    frameRateHistory = []
    frameIntervalHistory = []
    statisticsCounter = 0
    lastBufferedEnd = 0

    // 延迟重新初始化，给系统时间清理
    setTimeout(() => {
      try {
        initJMuxer()

        // 如果之前在播放，尝试恢复播放状态
        if (wasPlaying) {
          setTimeout(() => {
            const newVideoEl = document.getElementById(videoId.value) as HTMLVideoElement
            if (newVideoEl && newVideoEl.paused) {
              newVideoEl.play().catch(err => {
                warn(`恢复播放失败:`, err)
              })
            }
          }, 500)
        }

        isReinitializing = false
      } catch (err) {
        error_log(`重新初始化过程中出错:`, err)
        isReinitializing = false
      }
    }, 200)

  } catch (err) {
    error_log(`重新初始化失败:`, err)
    isReinitializing = false
  }
}

// 初始化JMuxer
const initJMuxer = () => {
  try {
    // 重置帧计数
    frameCount = 0
    lastFrameTime = Date.now()

    // 使用针对性优化的JMuxer配置解决缓冲区问题
    jmuxer = new JMuxer({
      node: videoId.value,
      mode: 'video',
      flushingTime: 80, // 降低缓冲时长以减少延迟
      clearBuffer: false, // 暂时禁用自动清理，让缓冲区积累更多数据
      maxDelay: 400, // 降低最大延迟，避免过度累计导致糊化
      fps: 20, // 统一使用20fps，与agent配置保持一致
      debug: false,
      onReady: () => {
        log(`解码器就绪 (配置20fps)`)

        // 解码器就绪后，确保Video元素开始播放
        setTimeout(() => {
          const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
          if (videoEl) {
            // 设置video元素属性确保自动播放
            videoEl.autoplay = true
            videoEl.muted = true // 静音以允许自动播放
            videoEl.playsInline = true

            if (videoEl.paused) {
              log(`启动播放`)
              videoEl.play().catch(err => {
                warn(`启动播放失败:`, err)
              })
            }
            // 附加视频事件监听以更可靠地判断是否有画面
            try { videoEl.removeEventListener('timeupdate', onVideoProgress) } catch {}
            videoEl.addEventListener('timeupdate', onVideoProgress)
            try { videoEl.removeEventListener('loadeddata', onVideoLoaded) } catch {}
            videoEl.addEventListener('loadeddata', onVideoLoaded)
          }
        }, 100)
      },
      onError: (err: any) => {
        error_log(`JMuxer内部错误:`, err)
        // 立即重新初始化，避免状态异常持续
        setTimeout(() => reinitializeJMuxer(), 200)
      },
      onMissingVideoFrames: (frames: any) => {
        warn(`丢失视频帧:`, frames, '可能导致帧率从20fps降到15fps')
      }
    })

    // 添加JMuxer事件监听来诊断问题
    if (jmuxer.on) {
      jmuxer.on('ready', () => {
        log(`解码器就绪`)
      })

      jmuxer.on('error', (err: any) => {
        error_log(`解码错误:`, err)
        // 尝试重新初始化解码器
        setTimeout(() => {
          warn(`尝试重新初始化解码器`)
          reinitializeJMuxer()
        }, 1000)
      })
    }

    log(`解码器初始化完成`)

    // 启动缓冲区健康检查
    startBufferHealthCheck()

    // 启动Video元素状态检查
    startVideoElementCheck()

    // 启动JMuxer健康检查
    startJMuxerHealthCheck()

    // 启动预防性重启机制（每10分钟重新初始化一次，避免长时间运行后的状态异常）
    startPreventiveRestart()

    // 启动播放状态检查
    startPlaybackCheck()

    // 启动卡住检测
    startStallDetection()

  } catch (err) {
    error_log(`初始化失败:`, err)
    error.value = 'JMuxer初始化失败'
    emit('error', error.value)
  }
}

// 连接WebSocket
const connectWebSocket = () => {
  try {
    // 检查WebSocket URL是否有效
    if (!wsUrl.value) {
      error.value = '设备IP地址不可用，无法建立连接'
      emit('error', error.value)
      return
    }

    warn('连接到:', wsUrl.value)

    wsVideo = new WebSocket(wsUrl.value)
    wsVideo.binaryType = 'arraybuffer'

    wsVideo.addEventListener('open', () => {
      warn(`WebSocket连接成功`)
      connected.value = true
      error.value = ''
      stalled.value = false
      hasVideo.value = false
      // 启动无画面提示宽限期（避免刚连上短暂黑屏误报）
      startupGrace.value = true
      if (startupGraceTimer) clearTimeout(startupGraceTimer)
      startupGraceTimer = setTimeout(() => {
        startupGrace.value = false
      }, 3000)
      emit('connected')
    })

    wsVideo.addEventListener('message', (event) => {
      if (jmuxer && event.data) {
        try {
          const now = Date.now()
          const frameInterval = lastFrameTime > 0 ? now - lastFrameTime : avgFrameInterval

          frameCount++
          lastFrameTime = now
          statisticsCounter++

          // 只每10帧计算一次统计数据，减少CPU负担
          if (statisticsCounter % 10 === 0) {
            // 记录帧间隔历史
            frameIntervalHistory.push(frameInterval)
            if (frameIntervalHistory.length > 20) {
              frameIntervalHistory.shift() // 只保留最近20个间隔
            }

            // 更新平均帧间隔，但保持在合理范围内
            if (frameIntervalHistory.length >= 10) {
              const newAvgInterval = frameIntervalHistory.slice(-10).reduce((a, b) => a + b, 0) / 10
              // 限制平均间隔在40-80ms之间，对应12.5-25fps，以20fps为中心
              avgFrameInterval = Math.max(40, Math.min(80, newAvgInterval))
            }
          }

          const uint8Array = new Uint8Array(event.data)

          // 基于统计的智能日志记录（减少日志频率）
          const shouldLogDetail = frameCount % 5000 === 0 || // 每5000帧记录一次
                                 frameInterval > avgFrameInterval * 10 || // 只有严重异常才记录
                                 event.data.byteLength > 300000 // 进一步提高阈值，减少噪音

          if (shouldLogDetail) {
            // 识别帧类型并记录
            const nalType = uint8Array[4] & 0x1F
            let frameType = 'Unknown'
            switch (nalType) {
              case 1: frameType = 'P帧'; break
              case 5: frameType = 'IDR帧'; break
              case 7: frameType = 'SPS'; break
              case 8: frameType = 'PPS'; break
              case 9: frameType = 'AUD'; break
              default: frameType = `NAL${nalType}`
            }

            log(`帧 #${frameCount}: ${frameType}, ${event.data.byteLength} 字节, 间隔: ${frameInterval.toFixed(1)}ms`)

            // 修正异常帧检测逻辑 - 只有真正异常的情况才记录
            const isReallyAbnormal = frameInterval > avgFrameInterval * 20 || // 帧间隔超过1秒才算异常
                                   event.data.byteLength > 1000000 || // 超过1MB才算异常大小
                                   event.data.byteLength < 100 // 小于100字节才算异常小

            if (isReallyAbnormal) {
              const header = Array.from(uint8Array.slice(0, 8)).map(b => b.toString(16).padStart(2, '0')).join(' ')
              warn(`真正异常帧 #${frameCount} 头部: ${header}`)

              // 检查video元素状态
              const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
              if (videoEl) {
                const bufferedInfo = videoEl.buffered.length > 0 ?
                  `${videoEl.buffered.start(0).toFixed(2)}-${videoEl.buffered.end(0).toFixed(2)}` : 'empty'

                warn(`真正异常帧 #${frameCount} Video状态:`, {
                  readyState: videoEl.readyState,
                  currentTime: videoEl.currentTime.toFixed(2),
                  buffered: bufferedInfo,
                  paused: videoEl.paused,
                  networkState: videoEl.networkState,
                  error: videoEl.error ? videoEl.error.message : null,
                  frameInterval: frameInterval.toFixed(1) + 'ms',
                  avgInterval: avgFrameInterval.toFixed(1) + 'ms'
                })

                // 检查是否有缓冲区问题
                if (videoEl.buffered.length > 0) {
                  const bufferEnd = videoEl.buffered.end(0)
                  const currentTime = videoEl.currentTime
                  const bufferHealth = bufferEnd - currentTime

                  // 调整缓冲区健康度阈值，减少误报
                  if (bufferHealth < 0.2) { // 只有缓冲区少于200ms才警告
                    warn(`帧 #${frameCount} 缓冲区健康度低: ${bufferHealth.toFixed(2)}s`)
                  }

                  // 只有在极端情况下才重新初始化
                  if (bufferHealth < 0.1 && frameInterval > avgFrameInterval * 10) {
                    error_log(`检测到严重的缓冲区和帧间隔问题，尝试重新初始化`)
                    setTimeout(() => reinitializeJMuxer(), 100)
                    return
                  }
                }
              }
            }
          }

          // 喂给JMuxer解码
          jmuxer.feed({
            video: uint8Array
          })

          // 根据video缓冲快速标记有画面
          const videoElQuick = document.getElementById(videoId.value) as HTMLVideoElement
          if (videoElQuick && videoElQuick.readyState >= 2 && videoElQuick.buffered.length > 0) {
            hasVideo.value = true
          }

        } catch (err) {
          error_log(`帧 #${frameCount} 处理失败:`, err)

          // 基于错误频率决定是否重新初始化
          const errorRate = frameCount > 0 ? (1 / frameCount) : 0
          if (errorRate > 0.001 || frameCount > 5000) { // 错误率过高或运行时间较长时更容易重新初始化
            warn(`帧处理错误，错误率: ${(errorRate * 100).toFixed(3)}%，尝试重新初始化`)
            setTimeout(() => reinitializeJMuxer(), 500)
          }
        }
      }
    })

    wsVideo.addEventListener('error', (err) => {
      error_log(`WebSocket错误:`, err)
      error.value = 'WebSocket连接错误'
      connected.value = false
      stalled.value = false
      hasVideo.value = false
      emit('error', error.value)
    })

    wsVideo.addEventListener('close', (event) => {
      warn(`WebSocket关闭: ${event.code} ${event.reason}`)
      connected.value = false
      stalled.value = false
      hasVideo.value = false
      emit('disconnected')

      if (event.code !== 1000) {
        error.value = `WebSocket连接断开: ${event.reason || '未知原因'}`
        emit('error', error.value)
      }
    })

  } catch (err) {
    error_log('连接WebSocket失败:', err)
    error.value = (err as Error).message
    emit('error', error.value)
  }
}

// 开始流
const startStream = async () => {
  try {
    error.value = ''

    // 等待DOM更新
    await nextTick()

    // 初始化JMuxer
    initJMuxer()

    // 建立WebSocket连接
    connectWebSocket()

  } catch (err) {
    error_log('启动流失败:', err)
    error.value = (err as Error).message
    emit('error', error.value)
  }
}

// 停止流
const stopStream = () => {
  try {
    warn(`停止流媒体，总计处理 ${frameCount} 帧`)

    // 停止缓冲区健康检查
    if (bufferHealthCheck) {
      clearInterval(bufferHealthCheck)
      bufferHealthCheck = null
    }

    // 停止Video元素检查
    if (videoElementCheck) {
      clearInterval(videoElementCheck)
      videoElementCheck = null
    }

    // 停止JMuxer健康检查
    if (jmuxerHealthCheck) {
      clearInterval(jmuxerHealthCheck)
      jmuxerHealthCheck = null
    }

    // 停止预防性重启
    if (preventiveRestart) {
      clearTimeout(preventiveRestart)
      preventiveRestart = null
    }

    // 停止播放状态检查
    if (playbackCheck) {
      clearInterval(playbackCheck)
      playbackCheck = null
    }

    // 停止卡住检测
    if (stallDetectionTimer) {
      clearInterval(stallDetectionTimer)
      stallDetectionTimer = null
    }

    // 停止页面可见性检查
    if (visibilityCheck) {
      clearTimeout(visibilityCheck)
      visibilityCheck = null
    }

    // 关闭WebSocket
    if (wsVideo) {
      wsVideo.close()
      wsVideo = null
    }

    // 销毁JMuxer
    if (jmuxer) {
      try {
        jmuxer.destroy()
      } catch (err) {
        warn(`销毁解码器时出现警告:`, err)
      }
      jmuxer = null
    }

    // 重置状态
    connected.value = false
    error.value = ''
    stalled.value = false
    hasVideo.value = false
    frameCount = 0
    lastFrameTime = 0
    lastVideoCurrentTime = 0
    stuckDetectionCount = 0

    // 重置统计数据
    frameRateHistory = []
    lastFrameRateCheck = 0
    avgFrameInterval = 50 // 重置为20fps对应的间隔
    frameIntervalHistory = []
    statisticsCounter = 0
    lastBufferedEnd = 0

    emit('disconnected')

  } catch (err) {
    error_log(`停止流媒体失败:`, err)
    error.value = (err as Error).message
    emit('error', error.value)
  }
}

// 清理资源
const cleanup = () => {
  stopStream()
}

// 视频事件处理
const onVideoProgress = () => {
  hasVideo.value = true
  lastVideoProgressTime = Date.now()
}

const onVideoLoaded = () => {
  // 初次有数据可播放
  hasVideo.value = true
}

// 暴露方法给父组件
defineExpose({
  startStream,
  stopStream,
  reconnectStream,
  connected,
  error,
  stalled,
  hasVideo
})

// 生命周期
onMounted(() => {
  // 添加页面可见性监听
  document.addEventListener('visibilitychange', handleVisibilityChange)
  
  if (props.autoStart) {
    startStream()
  }
})

onBeforeUnmount(() => {
  // 移除页面可见性监听
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  const videoEl = document.getElementById(videoId.value) as HTMLVideoElement
  if (videoEl) {
    try { videoEl.removeEventListener('timeupdate', onVideoProgress) } catch {}
    try { videoEl.removeEventListener('loadeddata', onVideoLoaded) } catch {}
  }
  cleanup()
})
</script>

<style scoped>
.jmuxer-decoder {
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

.stalled-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.3);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 15;
}

.stalled-content {
  text-align: center;
  color: #fff;
  background: rgba(0, 0, 0, 0.3);
  border-radius: 8px;
  padding: 24px 28px;
  box-shadow: 0 2px 6px rgba(0,0,0,0.2);
}

.stalled-message {
  font-size: 16px;
  margin-bottom: 16px;
  text-shadow: 0 2px 4px rgba(0,0,0,0.6);
}


.reconnect-btn {
  background: rgba(255,255,255,0.1);
  border: 1px solid rgba(255,255,255,0.3);
  color: #fff;
  padding: 8px 16px;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.reconnect-btn:hover {
  background: rgba(255,255,255,0.2);
  border-color: rgba(255,255,255,0.5);
}

.reconnect-btn:active {
  background: rgba(255,255,255,0.25);
  border-color: rgba(255,255,255,0.6);
}
</style>
