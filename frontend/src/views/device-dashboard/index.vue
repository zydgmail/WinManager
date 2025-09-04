<template>
  <div class="device-dashboard-container">
    <!-- 主内容区域 -->
    <div class="dashboard-main">
      <!-- 左侧分组列表 -->
      <GroupList
        :group-list="groupList"
        :selected-group-id="selectedGroupId"
        :ungrouped-devices="ungroupedDevices"
        :loading="groupLoading"
        @select-group="handleSelectGroup"
        @refresh="refreshListsOnly"
      />

      <!-- 右侧设备网格 -->
      <DeviceGrid
        :device-list="filteredDevices"
        :group-list="groupList"
        :loading="deviceLoading"
        :refresh-loading="syncLoading"
        :refresh-timestamp="screenshotRefreshTimestamp"
        @device-select="handleDeviceSelect"
        @device-action="handleDeviceAction"
        @device-click="handleDeviceClick"
        @batch-operation="handleBatchOperation"
        @move-to-group="handleMoveToGroup"
        @refresh-interval-change="handleRefreshIntervalChange"
        @refresh="refreshListsOnly"
      />
    </div>

    <!-- 串流弹窗 -->
    <StreamDialog
      v-model:visible="showStreamDialog"
      :device="selectedDevice"
      @close="handleStreamDialogClose"
    />
  </div>
</template>

<script setup lang="ts">
import { deviceApi, groupApi, type DeviceInfo, type GroupInfo } from '@/api/device'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDeviceDashboardStore } from '@/store/modules/device-dashboard'
import DeviceGrid from './components/DeviceGrid.vue'
import GroupList from './components/GroupList.vue'
import StreamDialog from './components/StreamDialog.vue'

// 路由
const router = useRouter()

// 使用设备仪表板store
const dashboardStore = useDeviceDashboardStore()

// 响应式数据
const groupLoading = ref(false)
const deviceLoading = ref(false)
const syncLoading = ref(false)
const groupList = ref<GroupInfo[]>([])
const deviceList = ref<DeviceInfo[]>([])

// 串流弹窗相关
const showStreamDialog = ref(false)
const selectedDevice = ref<DeviceInfo | null>(null)

// 从store获取当前选择的分组ID
const selectedGroupId = computed({
  get: () => dashboardStore.getSelectedGroupId,
  set: (value: number | null) => dashboardStore.setSelectedGroupId(value)
})

// 从store获取刷新间隔设置
const autoRefreshInterval = computed({
  get: () => dashboardStore.getRefreshInterval,
  set: (value: number) => dashboardStore.setRefreshInterval(value)
})

const screenshotRefreshTimestamp = ref(0) // 截图刷新时间戳

// 自动刷新定时器
let refreshTimer: NodeJS.Timeout | null = null

// 计算属性
const totalDevices = computed(() => deviceList.value.length)
const onlineDevices = computed(() =>
  deviceList.value.filter(device => device.status === 1).length
)

// 计算待分组设备数量（group_id为0、null或undefined的设备）
const ungroupedDevices = computed(() =>
  deviceList.value.filter(device =>
    !device.group_id || device.group_id === 0
  ).length
)

const filteredDevices = computed(() => {
  if (selectedGroupId.value === null) {
    // 显示待分组设备（group_id为0、null或undefined的设备）
    return deviceList.value.filter(device =>
      !device.group_id || device.group_id === 0
    )
  }
  return deviceList.value.filter(device =>
    device.group_id === selectedGroupId.value ||
    (device.Group && device.Group.ID === selectedGroupId.value)
  )
})

// 方法
const loadGroupList = async () => {
  try {
    groupLoading.value = true
    const response = await groupApi.getGroupList({ page: 1, size: 100 })
    // 处理可能的嵌套响应结构，后端应该返回包含device_count的分组数据
    groupList.value = response.data.groups || response.data.groups || []
  } catch (error) {
    ElMessage.error('加载分组列表失败')
    console.error('加载分组列表错误:', error)
    groupList.value = []
  } finally {
    groupLoading.value = false
  }
}

const loadDeviceList = async () => {
  try {
    deviceLoading.value = true
    const response = await deviceApi.getDeviceList({ page: 1, size: 100 })
    deviceList.value = response.data.devices || []
    // 不在这里加载截图，让DeviceCard组件自己处理
  } catch (error) {
    ElMessage.error('加载设备列表失败')
    console.error('加载设备列表错误:', error)
  } finally {
    deviceLoading.value = false
  }
}

const refreshData = async () => {
  try {
    await Promise.all([
      loadGroupList(),
      loadDeviceList()
    ])
  } catch (error) {
    ElMessage.error('数据刷新失败')
  }
}

// 只刷新设备和分组列表，不刷新截图
const refreshListsOnly = async () => {
  try {
    await Promise.all([
      loadGroupList(),
      loadDeviceList()
    ])
    ElMessage.success('列表刷新成功')
  } catch (error) {
    ElMessage.error('列表刷新失败')
  }
}

// 分组相关事件处理
const handleSelectGroup = (groupId: number | null) => {
  selectedGroupId.value = groupId
  console.log('选择分组:', groupId, '已保存到store')
}

// 设备相关事件处理
const handleDeviceSelect = (deviceId: number, selected: boolean) => {
  // 这里可以处理设备选择逻辑，如果需要的话
  console.log('Device select:', deviceId, selected)
}

const handleDeviceClick = (device: DeviceInfo) => {
  if (device.status !== 1) {
    ElMessage.warning('设备离线，无法打开控制台')
    return
  }
  // 显示串流弹窗而不是跳转页面
  showStreamDialog.value = true
  selectedDevice.value = device

  // 暂停自动刷新以节省资源
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
    console.log('串流弹窗打开，已暂停后台图片刷新')
  }
}

const handleDeviceAction = (command: string, device: DeviceInfo) => {
  const [action, deviceIdStr] = command.split('_')
  const deviceId = parseInt(deviceIdStr)

  switch (action) {
    case 'console':
      handleDeviceClick(device)
      break
    case 'detail':
      router.push(`/device/detail/${deviceId}`)
      break
    case 'screenshot':
      // 通过更新时间戳触发该设备的截图刷新
      screenshotRefreshTimestamp.value = Date.now()
      ElMessage.success('正在刷新截图...')
      break
    case 'reboot':
      rebootDevice(deviceId)
      break
    case 'delete':
      deleteDevice(deviceId)
      break
  }
}

const handleBatchOperation = (command: string, deviceIds: number[]) => {
  switch (command) {
    case 'reboot':
      batchRebootDevices(deviceIds)
      break
    case 'screenshot':
      batchRefreshScreenshots(deviceIds)
      break
    case 'delete':
      batchDeleteDevices(deviceIds)
      break
  }
}

const handleMoveToGroup = async (groupId: number, deviceIds: number[]) => {
  try {
    await deviceApi.moveDeviceToGroup({
      ids: deviceIds,
      group_id: groupId
    })
    ElMessage.success('设备移动成功')

    // 重新加载分组列表和设备列表以获取最新的设备数量
    await refreshListsOnly()
  } catch (error) {
    ElMessage.error('设备移动失败')
    console.error('移动设备错误:', error)
  }
}

const handleRefreshIntervalChange = (interval: number) => {
  autoRefreshInterval.value = interval
  setupAutoRefresh()
  console.log('刷新间隔已更新:', interval, '秒，已保存到store')
}





const rebootDevice = async (deviceId: number) => {
  try {
    await ElMessageBox.confirm('确定要重启该设备吗？', '确认重启', {
      type: 'warning'
    })

    await deviceApi.rebootDevice(deviceId)
    ElMessage.success('重启指令已发送')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('重启失败')
    }
  }
}

const deleteDevice = async (deviceId: number) => {
  try {
    await ElMessageBox.confirm('确定要删除该设备吗？此操作不可恢复！', '确认删除', {
      type: 'error'
    })

    await deviceApi.deleteDevice(deviceId)

    // 重新加载分组列表和设备列表以获取最新的设备数量
    await refreshListsOnly()

    ElMessage.success('设备删除成功')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('删除失败')
      console.error('删除设备错误:', error)
    }
  }
}

// 辅助方法

const setupAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }

  if (autoRefreshInterval.value > 0) {
    refreshTimer = setInterval(() => {
      // 通过更新时间戳触发所有DeviceCard组件刷新截图
      screenshotRefreshTimestamp.value = Date.now()
      // console.log(`自动刷新截图，时间戳: ${screenshotRefreshTimestamp.value}`)
    }, autoRefreshInterval.value * 1000)
  }
}

const batchRebootDevices = async (deviceIds: number[]) => {
  try {
    await ElMessageBox.confirm(`确定要重启选中的 ${deviceIds.length} 台设备吗？`, '批量重启', {
      type: 'warning'
    })

    const promises = deviceIds.map(deviceId =>
      deviceApi.rebootDevice(deviceId).catch(error => {
        console.error(`重启设备 ${deviceId} 失败:`, error)
        return null
      })
    )

    await Promise.all(promises)
    ElMessage.success('批量重启指令已发送')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量重启失败')
    }
  }
}

const batchRefreshScreenshots = (deviceIds: number[]) => {
  try {
    // 获取选中的在线设备
    const onlineDevices = deviceList.value.filter(device => 
      deviceIds.includes(device.ID) && device.status === 1
    )
    
    if (onlineDevices.length === 0) {
      ElMessage.warning('没有选中的在线设备可以刷新截图')
      return
    }

    // 通过更新时间戳触发选中设备的截图刷新
    screenshotRefreshTimestamp.value = Date.now()
    
    ElMessage.success(`正在刷新 ${onlineDevices.length} 台设备的截图...`)
    console.log(`批量刷新截图，设备IDs: ${deviceIds.join(', ')}，在线设备数: ${onlineDevices.length}`)
  } catch (error) {
    ElMessage.error('批量刷新截图失败')
    console.error('批量刷新截图错误:', error)
  }
}

const batchDeleteDevices = async (deviceIds: number[]) => {
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${deviceIds.length} 台设备吗？此操作不可恢复！`, '批量删除', {
      type: 'error'
    })

    const promises = deviceIds.map(deviceId =>
      deviceApi.deleteDevice(deviceId).catch(error => {
        console.error(`删除设备 ${deviceId} 失败:`, error)
        return null
      })
    )

    await Promise.all(promises)

    // 重新加载分组列表和设备列表以获取最新的设备数量
    await refreshListsOnly()

    ElMessage.success('批量删除完成')
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('批量删除失败')
    }
  }
}

// 串流弹窗相关方法
const handleStreamDialogClose = () => {
  showStreamDialog.value = false
  selectedDevice.value = null

  // 恢复自动刷新
  setupAutoRefresh()
  console.log('串流弹窗关闭，已恢复后台图片刷新')
}

// 生命周期
onMounted(async () => {
  await refreshData()

  // 从store恢复状态，输出日志
  console.log(`从store恢复状态:`)
  console.log(`- 选择的分组ID: ${selectedGroupId.value}`)
  console.log(`- 刷新间隔: ${autoRefreshInterval.value}秒`)

  setupAutoRefresh()
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
})
</script>

<style scoped>
.device-dashboard-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.dashboard-main {
  flex: 1;
  display: flex;
  overflow: hidden;
  gap: 0;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .dashboard-main {
    flex-direction: column;
  }
}
</style>
