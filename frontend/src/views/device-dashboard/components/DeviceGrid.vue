<template>
  <div class="device-grid-container">
    <!-- 工具栏 -->
    <div class="grid-toolbar">
      <div class="toolbar-left">
        <el-checkbox
          v-model="selectAll"
          :indeterminate="isIndeterminate"
          @change="handleSelectAll"
        >
          全选
        </el-checkbox>
        <span class="selected-count" v-if="selectedDevices.length > 0">
          已选择 {{ selectedDevices.length }} 台设备
        </span>
      </div>

      <div class="toolbar-right">
        <div class="operation-buttons">
          <el-dropdown
            :disabled="selectedDevices.length === 0"
            @command="handleBatchOperation"
          >
            <el-button :disabled="selectedDevices.length === 0">
              <span>批量操作</span>
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="reboot">批量重启</el-dropdown-item>
                <el-dropdown-item command="shutdown">批量关机</el-dropdown-item>
                <el-dropdown-item command="screenshot">刷新截图</el-dropdown-item>
                <el-dropdown-item divided command="delete">批量删除</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>

          <el-dropdown
            :disabled="selectedDevices.length === 0"
            @command="handleMoveToGroup"
          >
            <el-button :disabled="selectedDevices.length === 0">
              <span>移动分组</span>
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item
                  v-for="group in groupList"
                  :key="group.ID"
                  :command="group.ID"
                >
                  {{ group.name }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>

        <el-select
          v-model="columnCount"
          @change="handleColumnCountChange"
          style="width: 100px;"
        >
          <el-option label="1列" :value="1" />
          <el-option label="2列" :value="2" />
          <el-option label="3列" :value="3" />
          <el-option label="4列" :value="4" />
        </el-select>

        <el-select
          v-model="autoRefreshInterval"
          @change="handleRefreshIntervalChange"
          style="width: 140px;"
        >
          <el-option label="不刷新屏幕" :value="0" />
          <el-option label="1秒刷新屏幕" :value="1" />
          <el-option label="3秒刷新屏幕" :value="3" />
          <el-option label="5秒刷新屏幕" :value="5" />
          <el-option label="10秒刷新屏幕" :value="10" />
          <el-option label="30秒刷新屏幕" :value="30" />
        </el-select>
      </div>
    </div>

    <!-- 设备网格 -->
    <div class="device-grid" v-loading="loading">
      <DeviceCard
        v-for="device in deviceList"
        :key="device.ID"
        :device="device"
        :selected-devices="selectedDevices"
        :refresh-timestamp="props.refreshTimestamp"
        :resize-timestamp="resizeTimestamp"
        @select="handleDeviceSelect"
        @action="handleDeviceAction"
        @click="handleDeviceClick"
      />

      <!-- 空状态 -->
      <div v-if="deviceList.length === 0 && !loading" class="empty-state">
        <el-empty description="暂无设备数据">
          <el-button type="primary" @click="handleRefresh">刷新数据</el-button>
        </el-empty>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { DeviceInfo, GroupInfo } from '@/api/device'
import { useDeviceDashboardStore } from '@/store/modules/device-dashboard'
import { ArrowDown } from '@element-plus/icons-vue'
import { computed, nextTick, onMounted, ref } from 'vue'
import DeviceCard from './DeviceCard.vue'

// Props
interface Props {
  deviceList: DeviceInfo[]
  groupList: GroupInfo[]
  loading?: boolean
  refreshLoading?: boolean
  refreshTimestamp?: number // 刷新时间戳
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  refreshLoading: false,
  refreshTimestamp: 0
})

// Emits
const emit = defineEmits<{
  deviceSelect: [deviceId: number, selected: boolean]
  deviceAction: [command: string, device: DeviceInfo]
  deviceClick: [device: DeviceInfo]
  batchOperation: [command: string, deviceIds: number[]]
  moveToGroup: [groupId: number, deviceIds: number[]]
  refreshIntervalChange: [interval: number]
  refresh: []
}>()

// 使用设备仪表板store
const dashboardStore = useDeviceDashboardStore()

// 响应式数据
const selectedDevices = ref<number[]>([])
const selectAll = ref(false)
const resizeTimestamp = ref(0) // 用于触发子组件重新计算尺寸

// 从store获取状态
const autoRefreshInterval = computed({
  get: () => dashboardStore.getRefreshInterval,
  set: (value: number) => dashboardStore.setRefreshInterval(value)
})

const columnCount = computed({
  get: () => dashboardStore.getColumnCount,
  set: (value: number) => dashboardStore.setColumnCount(value)
})

// 计算属性
const isIndeterminate = computed(() => {
  const selectedCount = selectedDevices.value.length
  const totalCount = props.deviceList.length
  return selectedCount > 0 && selectedCount < totalCount
})

// 方法
const handleDeviceSelect = (deviceId: number, selected: boolean) => {
  if (selected) {
    if (!selectedDevices.value.includes(deviceId)) {
      selectedDevices.value.push(deviceId)
    }
  } else {
    const index = selectedDevices.value.indexOf(deviceId)
    if (index > -1) {
      selectedDevices.value.splice(index, 1)
    }
  }

  // 更新全选状态
  const totalCount = props.deviceList.length
  selectAll.value = selectedDevices.value.length === totalCount

  emit('deviceSelect', deviceId, selected)
}

const handleSelectAll = (checked: boolean) => {
  if (checked) {
    selectedDevices.value = props.deviceList.map(device => device.ID)
  } else {
    selectedDevices.value = []
  }
  selectAll.value = checked
}

const handleDeviceAction = (command: string, device: DeviceInfo) => {
  emit('deviceAction', command, device)
}

const handleDeviceClick = (device: DeviceInfo) => {
  emit('deviceClick', device)
}

const handleBatchOperation = (command: string) => {
  emit('batchOperation', command, selectedDevices.value)
  selectedDevices.value = []
  selectAll.value = false
}

const handleMoveToGroup = (groupId: number) => {
  emit('moveToGroup', groupId, selectedDevices.value)
  selectedDevices.value = []
  selectAll.value = false
}

const handleColumnCountChange = (count: number) => {
  columnCount.value = count
  console.log(`列数已更改为: ${count}列，已保存到store`)

  // 延迟触发重新计算，确保CSS Grid布局完全更新
  nextTick(() => {
    setTimeout(() => {
      resizeTimestamp.value = Date.now()
      console.log('触发设备卡片尺寸重新计算')
    }, 200) // 增加延迟时间确保布局完成
  })
}

const handleRefreshIntervalChange = (interval: number) => {
  autoRefreshInterval.value = interval
  emit('refreshIntervalChange', interval)
  console.log(`刷新间隔已更改为: ${interval}秒，已保存到store`)
}

const handleRefresh = () => {
  emit('refresh')
}

// 组件挂载时初始化设置
onMounted(() => {
  // 从store恢复设置
  console.log(`DeviceGrid从store恢复设置:`)
  console.log(`- 列数: ${columnCount.value}列`)
  console.log(`- 刷新间隔: ${autoRefreshInterval.value}秒`)

  // 如果有保存的刷新间隔设置，通知父组件
  if (autoRefreshInterval.value > 0) {
    emit('refreshIntervalChange', autoRefreshInterval.value)
  }
})

// 暴露方法给父组件
defineExpose({
  clearSelection: () => {
    selectedDevices.value = []
    selectAll.value = false
  }
})
</script>

<style scoped>
.device-grid-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #f8f9fa;
}

.grid-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.selected-count {
  font-size: 14px;
  color: #409eff;
  font-weight: 600;
  background: linear-gradient(135deg, #ecf5ff 0%, #e1f3ff 100%);
  padding: 6px 12px;
  border-radius: 16px;
  border: 1px solid rgba(64, 158, 255, 0.2);
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.operation-buttons {
  display: flex;
  align-items: center;
  gap: 12px;
}

.operation-buttons .el-button {
  display: flex;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.operation-buttons .el-button span {
  display: inline-block;
  text-align: center;
}

.device-grid {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  display: grid;
  grid-template-columns: v-bind('`repeat(${columnCount}, 1fr)`');
  gap: 8px;
  align-content: start;
  /* 确保网格项目高度由内容决定 */
  grid-auto-rows: max-content;
  align-items: start;
}

.empty-state {
  grid-column: 1 / -1;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
}

/* 响应式设计 - 在小屏幕上覆盖用户选择的列数 */
@media (max-width: 1600px) {
  .device-grid {
    gap: 14px;
    padding: 14px;
  }
}

@media (max-width: 1200px) {
  .device-grid {
    gap: 12px;
    padding: 12px;
  }

  .grid-toolbar {
    padding: 16px 20px;
  }
}

@media (max-width: 768px) {
  .device-grid {
    grid-template-columns: repeat(2, 1fr) !important;
    gap: 10px;
    padding: 10px;
  }

  .grid-toolbar {
    flex-direction: column;
    gap: 16px;
    padding: 16px;
  }

  .toolbar-left,
  .toolbar-right {
    width: 100%;
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .device-grid {
    grid-template-columns: 1fr !important;
    gap: 8px;
    padding: 8px;
  }
}

/* 滚动条样式 */
.device-grid::-webkit-scrollbar {
  width: 8px;
}

.device-grid::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 4px;
}

.device-grid::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;
}

.device-grid::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>
