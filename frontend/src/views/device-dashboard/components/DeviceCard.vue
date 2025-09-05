<template>
  <div
    ref="deviceCardRef"
    class="device-card"
    :class="{
      online: device.status === 1,
      offline: device.status === 0,
      selected: isSelected
    }"
    @click="handleCardClick"
    @mouseenter="showCheckbox = true"
    @mouseleave="showCheckbox = false"
  >
    <!-- 勾选框 -->
    <input
      type="checkbox"
      class="device-checkbox"
      :class="{ visible: showCheckbox || isSelected }"
      :checked="isSelected"
      @click.stop="handleCheckboxClick"
      @change="handleSelectionChange"
    />

    <!-- 状态指示器 -->
    <div class="status-indicator">
      <div
        class="status-dot"
        :class="{ online: device.status === 1, offline: device.status === 0 }"
      />
      <span class="status-text">
        {{ device.status === 1 ? '在线' : '离线' }}
      </span>
    </div>

    <!-- 设备截图内容 -->
    <div class="device-content">
      <div
        ref="screenshotContainerRef"
        class="screenshot-container"
        :style="containerStyle"
      >
        <img
          v-if="screenshotUrl"
          :src="screenshotUrl"
          :alt="`${device.hostname}的截图`"
          class="screenshot-image"
          @error="handleImageError"
          @load="handleImageLoad"
        />
        <div v-else class="no-screenshot">
          <el-icon class="screenshot-icon"><Picture /></el-icon>
          <span class="no-screenshot-text">暂无截图</span>
        </div>
      </div>
    </div>

    <!-- 底部信息栏 -->
    <div class="device-footer">
      <!-- 左下角：内网IP -->
      <div class="device-ip">
        <el-icon class="ip-icon"><Location /></el-icon>
        <span>{{ device.lan || '未知IP' }}</span>
      </div>

      <!-- 右下角：操作菜单 -->
      <div class="device-actions" @click.stop>
        <el-dropdown trigger="click" @command="handleAction">
          <el-button size="small" text class="action-btn">
            <el-icon><MoreFilled /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                :command="`console_${device.ID}`"
                :disabled="device.status !== 1"
              >
                <el-icon><Monitor /></el-icon>
                远程控制
              </el-dropdown-item>
              <el-dropdown-item :command="`detail_${device.ID}`">
                <el-icon><View /></el-icon>
                设备详情
              </el-dropdown-item>
              <el-dropdown-item :command="`execscript_${device.ID}`">
                <el-icon><Setting /></el-icon>
                命令执行
              </el-dropdown-item>
              <el-dropdown-item :command="`screenshot_${device.ID}`" divided>
                <el-icon><Camera /></el-icon>
                刷新截图
              </el-dropdown-item>
              <el-dropdown-item :command="`reboot_${device.ID}`">
                <el-icon><RefreshRight /></el-icon>
                重启设备
              </el-dropdown-item>
              <el-dropdown-item :command="`delete_${device.ID}`" divided>
                <el-icon><Delete /></el-icon>
                删除设备
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { deviceApi, type DeviceInfo } from '@/api/device'
import {
  Camera,
  Delete,
  Location,
  Monitor,
  MoreFilled,
  Picture,
  RefreshRight,
  Setting,
  View
} from '@element-plus/icons-vue'
import { computed, onMounted, onUnmounted, ref, watch, nextTick } from 'vue'

// 添加模板引用
const deviceCardRef = ref<HTMLElement>()
const screenshotContainerRef = ref<HTMLElement>()

// 动态计算的图片尺寸
const calculatedImageSize = ref({ width: 0, height: 0 })

// Props
interface Props {
  device: DeviceInfo
  selectedDevices: number[]
  refreshTimestamp?: number // 刷新时间戳，用于触发截图刷新
  resizeTimestamp?: number // 尺寸变化时间戳，用于触发尺寸重新计算
}

const props = defineProps<Props>()

// Emits
const emit = defineEmits<{
  select: [deviceId: number, selected: boolean]
  action: [command: string, device: DeviceInfo]
  click: [device: DeviceInfo]
}>()

// 响应式数据
const showCheckbox = ref(false)
const screenshotUrl = ref('')

// 计算属性
const isSelected = computed(() => {
  return props.selectedDevices.includes(props.device.ID)
})

// 计算容器样式
const containerStyle = computed(() => {
  const height = calculatedImageSize.value.height > 0
    ? calculatedImageSize.value.height + 'px'
    : '200px'
  return { height }
})

// 获取设备截图
const getScreenshot = async () => {
  // 只有在线设备才获取截图
  if (props.device.status !== 1) {
    return
  }

  try {
    const response = await deviceApi.getDeviceScreenshot(props.device.ID)

    // 创建新的URL
    const newImageUrl = URL.createObjectURL(response)

    // 预加载新图片，确保加载完成后再替换
    const img = new Image()
    img.onload = () => {
      // 清理之前的URL
      if (screenshotUrl.value) {
        URL.revokeObjectURL(screenshotUrl.value)
      }

      // 直接替换为新图片，浏览器会平滑处理
      screenshotUrl.value = newImageUrl
    }

    img.onerror = () => {
      // 图片加载失败，清理URL
      URL.revokeObjectURL(newImageUrl)
    }

    // 开始加载图片
    img.src = newImageUrl

  } catch (error) {
    console.error(`获取设备 ${props.device.ID} 截图失败:`, error)
    // 静默处理错误，不显示错误消息
    // 清理可能存在的URL
    if (screenshotUrl.value) {
      URL.revokeObjectURL(screenshotUrl.value)
      screenshotUrl.value = ''
    }
  }
}

// 方法
const handleCardClick = () => {
  emit('click', props.device)
}

const handleCheckboxClick = () => {
  // 阻止事件冒泡，避免触发卡片点击
}

const handleSelectionChange = (event: Event) => {
  const target = event.target as HTMLInputElement
  emit('select', props.device.ID, target.checked)
}

const handleAction = (command: string) => {
  // 如果是刷新截图命令，重新获取截图
  if (command === `screenshot_${props.device.ID}`) {
    getScreenshot()
  }
  emit('action', command, props.device)
}

const handleImageError = () => {
  // 图片加载失败时清理URL
  if (screenshotUrl.value) {
    URL.revokeObjectURL(screenshotUrl.value)
    screenshotUrl.value = ''
  }
}

// 动态计算图片尺寸（基于16:9比例）
const calculateImageSize = () => {
  // 使用多重nextTick确保DOM完全更新
  nextTick(() => {
    nextTick(() => {
      if (screenshotContainerRef.value) {
        const container = screenshotContainerRef.value
        const containerWidth = container.clientWidth

        // 如果容器宽度为0，说明DOM还没准备好，延迟重试
        if (containerWidth === 0) {
          setTimeout(() => calculateImageSize(), 50)
          return
        }

        // 基于16:9比例计算高度
        const aspectRatio = 16 / 9
        const calculatedHeight = containerWidth / aspectRatio

        // 更新计算的尺寸
        calculatedImageSize.value = {
          width: containerWidth,
          height: calculatedHeight
        }


      }
    })
  })
}



const handleImageLoad = (event: Event) => {
  // 图片加载成功，重新计算尺寸
  calculateImageSize()
}

// 监听设备状态变化
watch(() => props.device.status, (newStatus) => {
  if (newStatus === 1) {
    // 设备上线时获取截图
    getScreenshot()
  } else {
    // 设备离线时清理截图
    if (screenshotUrl.value) {
      URL.revokeObjectURL(screenshotUrl.value)
      screenshotUrl.value = ''
    }
  }
})

// 监听刷新时间戳变化
watch(() => props.refreshTimestamp, (newTimestamp) => {
  if (newTimestamp && props.device.status === 1) {
    // 只有在线设备才响应刷新请求
    getScreenshot()
  }
})

// 监听尺寸变化时间戳
watch(() => props.resizeTimestamp, (newTimestamp) => {
  if (newTimestamp) {
    // 重新计算图片尺寸
    calculateImageSize()
  }
})

// 窗口大小变化时重新计算尺寸
const handleResize = () => {
  calculateImageSize()
}

// 组件挂载时获取截图（仅在线设备）
onMounted(() => {
  // 计算初始图片尺寸
  calculateImageSize()

  // 监听窗口大小变化
  window.addEventListener('resize', handleResize)

  if (props.device.status === 1) {
    getScreenshot()
  }
})

// 组件卸载时清理事件监听器
onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.device-card {
  position: relative;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden; /* 保留overflow: hidden以维持圆角效果 */
  transition: all 0.3s ease;
  cursor: pointer;
  border: 2px solid transparent;
  /* 让高度完全由内容决定 */
  width: 100%;
  height: auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.device-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.12);
}

.device-card.online {
  /* 在线设备样式 */
}

.device-card.offline {
  opacity: 0.8;
}

.device-card.selected {
  border-color: #409eff;
  box-shadow: 0 4px 16px rgba(64, 158, 255, 0.3);
}

.device-checkbox {
  position: absolute;
  left: 12px;
  top: 12px;
  z-index: 10;
  opacity: 0;
  transition: opacity 0.3s ease;
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.device-checkbox.visible {
  opacity: 1;
}

.status-indicator {
  position: absolute;
  top: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  z-index: 10;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.online {
  background-color: #52c41a;
}

.status-dot.offline {
  background-color: #f5222d;
}

.status-text {
  display: none;
}

.device-content {
  /* 让高度完全由内容决定，不限制overflow */
  display: block;
  padding: 0;
  /* 移除overflow: hidden，让内容完整显示 */
}

.screenshot-container {
  width: 100%;
  /* 高度通过内联样式动态设置 */
  display: block;
  background: #f5f7fa;
  position: relative;
}

.screenshot-image {
  width: 100%;
  height: 100%;
  /* 使用fill确保图片完全填满容器 */
  object-fit: fill;
  object-position: center;
  border-radius: 0;
  display: block;
  /* 移除过渡效果，让浏览器自然处理图片替换 */
}

.no-screenshot {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  width: 100%;
  height: 100%;
  background: #f5f7fa;
  color: #909399;
  font-size: 14px;
}

.screenshot-icon {
  font-size: 48px;
  color: #c0c4cc;
}

.no-screenshot-text {
  font-size: 16px;
  color: #c0c4cc;
  font-weight: 500;
}

.device-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
  border-top: 1px solid #e4e7ed;
}

.device-ip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: #606266;
  font-weight: 500;
}

.ip-icon {
  font-size: 14px;
  color: #909399;
}

.device-actions {
  opacity: 0.7;
  transition: opacity 0.3s ease;
}

.device-card:hover .device-actions {
  opacity: 1;
}

.action-btn {
  padding: 6px;
  border-radius: 6px;
  color: #606266;
}

.action-btn:hover {
  background: rgba(64, 158, 255, 0.1);
  color: #409eff;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .screenshot-icon {
    font-size: 40px;
  }

  .no-screenshot-text {
    font-size: 14px;
  }
}

@media (max-width: 768px) {
  .device-content {
    padding: 0;
  }

  .screenshot-icon {
    font-size: 32px;
  }

  .no-screenshot-text {
    font-size: 12px;
  }

  .device-footer {
    padding: 10px 12px;
  }

  .device-ip {
    font-size: 11px;
  }
}
</style>
