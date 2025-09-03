<template>
  <div class="device-detail">
    <!-- 页面头部 -->
    <div class="page-header">
      <el-row :gutter="20">
        <el-col :span="18">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item :to="{ path: '/device/list' }">设备列表</el-breadcrumb-item>
            <el-breadcrumb-item>设备详情</el-breadcrumb-item>
          </el-breadcrumb>
          <h2 v-if="deviceInfo">{{ deviceInfo.hostname || '设备详情' }}</h2>
        </el-col>
        <el-col :span="6" class="text-right">
          <el-button @click="handleBack">
            <el-icon><ArrowLeft /></el-icon>
            返回
          </el-button>
          <el-button type="primary" @click="handleRefresh">
            <el-icon><Refresh /></el-icon>
            刷新
          </el-button>
        </el-col>
      </el-row>
    </div>

    <div v-loading="loading">
      <!-- 基本信息卡片 -->
      <el-card class="info-card" header="设备信息">
        <el-row v-if="deviceInfo" :gutter="20">
          <el-col :span="8">
            <div class="info-item">
              <label>设备名称：</label>
              <span>{{ deviceInfo.hostname || '-' }}</span>
            </div>
            <div class="info-item">
              <label>操作系统：</label>
              <span>{{ deviceInfo.os || '-' }}</span>
            </div>
            <div class="info-item">
              <label>内网IP：</label>
              <span>{{ deviceInfo.lan || '-' }}</span>
            </div>
            <div class="info-item">
              <label>外网IP：</label>
              <span>{{ deviceInfo.wan || '未知' }}</span>
            </div>
          </el-col>
          <el-col :span="8">
            <div class="info-item">
              <label>MAC地址：</label>
              <span>{{ deviceInfo.mac || '-' }}</span>
            </div>
            <div class="info-item">
              <label>CPU：</label>
              <span>{{ deviceInfo.cpu || '-' }}</span>
            </div>
            <div class="info-item">
              <label>CPU核心数：</label>
              <span>{{ deviceInfo.cores || '-' }}</span>
            </div>
            <div class="info-item">
              <label>内存：</label>
              <span>{{ formatMemory(deviceInfo.memory) }}</span>
            </div>
          </el-col>
          <el-col :span="8">
            <div class="info-item">
              <label>运行时间：</label>
              <span>{{ formatUptime(deviceInfo.uptime) }}</span>
            </div>
            <div class="info-item">
              <label>状态：</label>
              <el-tag :type="deviceInfo.status === 1 ? 'success' : 'danger'">
                {{ deviceInfo.status === 1 ? '在线' : '离线' }}
              </el-tag>
            </div>
            <div class="info-item">
              <label>最后心跳：</label>
              <span>{{ formatTime(deviceInfo.last_heartbeat_at || deviceInfo.LastHeartbeatAt || deviceInfo.UpdatedAt || deviceInfo.updated_at) }}</span>
            </div>
            <div class="info-item">
              <label>所属分组：</label>
              <span>{{ getGroupName(deviceInfo) }}</span>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 设备操作 -->
      <el-card class="info-card" header="设备操作">
        <div class="action-buttons">
          <el-button
            type="primary"
            @click="handleScreenshot"
            :disabled="deviceInfo?.status !== 1"
          >
            <el-icon><Camera /></el-icon>
            获取截图
          </el-button>
          <el-text v-if="deviceInfo?.status !== 1" type="info" size="small">
            设备离线时无法获取截图
          </el-text>
        </div>
      </el-card>

      <!-- 调试信息 (开发环境) -->
      <el-card v-if="isDev" class="info-card" header="调试信息">
        <el-collapse>
          <el-collapse-item title="原始数据" name="raw">
            <pre class="debug-info">{{ JSON.stringify(deviceInfo, null, 2) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </el-card>
    </div>

    <!-- 截图预览对话框 -->
    <el-dialog
      v-model="showScreenshotDialog"
      title="设备截图"
      :width="screenshotDialogWidth"
      :fullscreen="isFullscreen"
      class="screenshot-dialog"
    >
      <template #header="{ titleId, titleClass }">
        <div class="screenshot-dialog-header">
          <span :id="titleId" :class="titleClass">设备截图</span>
          <div class="screenshot-dialog-actions">
            <el-button
              link
              @click="toggleFullscreen"
              :icon="isFullscreen ? 'el-icon-copy-document' : 'el-icon-full-screen'"
            >
              {{ isFullscreen ? '退出全屏' : '全屏' }}
            </el-button>
            <el-button
              link
              @click="downloadScreenshot"
              icon="el-icon-download"
            >
              下载
            </el-button>
          </div>
        </div>
      </template>

      <div v-loading="screenshotLoading" class="screenshot-container">
        <img
          v-if="screenshotData"
          :src="screenshotData"
          alt="设备截图"
          class="screenshot-image"
          @load="onImageLoad"
        />
        <div v-else-if="!screenshotLoading" class="no-screenshot">
          暂无截图数据
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage } from "element-plus";
import { ArrowLeft, Refresh, Camera } from "@element-plus/icons-vue";
import { deviceApi, type DeviceInfo } from "@/api/device";
import { formatTime, formatMemory, formatUptime } from "@/utils/format";

const route = useRoute();
const router = useRouter();

// 响应式数据
const loading = ref(false);
const deviceInfo = ref<DeviceInfo | null>(null);
const showScreenshotDialog = ref(false);
const screenshotLoading = ref(false);
const screenshotData = ref("");
const isFullscreen = ref(false);
const screenshotDialogWidth = ref("90%");
const isDev = ref(import.meta.env.DEV);

// 获取设备ID
const deviceId = Number(route.params.id);

// 获取设备详情
const getDeviceDetail = async () => {
  loading.value = true;
  try {
    const response = await deviceApi.getDevice(deviceId);
    console.log("设备详情数据:", response); // 调试信息
    deviceInfo.value = response.data; // 使用标准的响应格式
  } catch (error) {
    console.error("获取设备详情失败:", error);
    ElMessage.error("获取设备详情失败");
  } finally {
    loading.value = false;
  }
};

// 返回
const handleBack = () => {
  router.back();
};

// 刷新
const handleRefresh = () => {
  getDeviceDetail();
};

// 获取分组名称
const getGroupName = (device: any) => {
  if (!device) return '未分组';

  // 尝试不同的字段名
  const group = device.Group || device.group;
  if (group && group.name) {
    return group.name;
  }

  // 如果没有分组信息但有group_id，显示ID
  const groupId = device.group_id || device.GroupId;
  if (groupId) {
    return `分组${groupId}`;
  }

  return '未分组';
};

// 获取截图
const handleScreenshot = async () => {
  const status = deviceInfo.value?.status;
  if (status !== 1) {
    ElMessage.warning("设备离线，无法获取截图");
    return;
  }

  showScreenshotDialog.value = true;
  screenshotLoading.value = true;
  isFullscreen.value = false;

  // 清空之前的截图URL
  if (screenshotData.value) {
    URL.revokeObjectURL(screenshotData.value);
    screenshotData.value = "";
  }

  try {
    const response = await deviceApi.getDeviceScreenshot(deviceId);
    console.log("截图响应:", response); // 调试信息

    // 将Blob转换为URL用于显示
    screenshotData.value = URL.createObjectURL(response);
  } catch (error) {
    console.error("获取截图失败:", error);
    ElMessage.error("获取截图失败");
  } finally {
    screenshotLoading.value = false;
  }
};

// 切换全屏
const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value;
};

// 下载截图
const downloadScreenshot = () => {
  if (screenshotData.value) {
    const link = document.createElement('a');
    link.href = screenshotData.value;
    link.download = `device-${deviceId}-screenshot-${Date.now()}.jpg`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  }
};

// 图片加载完成，调整对话框大小
const onImageLoad = (event: Event) => {
  const img = event.target as HTMLImageElement;
  const screenWidth = window.innerWidth;
  const screenHeight = window.innerHeight;

  // 根据图片尺寸和屏幕尺寸计算合适的对话框宽度
  const imgRatio = img.naturalWidth / img.naturalHeight;
  const maxWidth = Math.min(screenWidth * 0.9, 1400);
  const maxHeight = screenHeight * 0.8;

  let dialogWidth;
  if (img.naturalWidth > maxWidth) {
    dialogWidth = maxWidth;
  } else if (img.naturalHeight > maxHeight) {
    dialogWidth = maxHeight * imgRatio;
  } else {
    dialogWidth = Math.max(img.naturalWidth, 800);
  }

  screenshotDialogWidth.value = `${Math.min(dialogWidth, maxWidth)}px`;
};

// 监听截图对话框关闭，清理URL避免内存泄漏
watch(showScreenshotDialog, (newVal) => {
  if (!newVal && screenshotData.value) {
    URL.revokeObjectURL(screenshotData.value);
    screenshotData.value = "";
  }
});

// 初始化
onMounted(() => {
  if (deviceId) {
    getDeviceDetail();
  } else {
    ElMessage.error("设备ID无效");
    router.push("/device/list");
  }
});
</script>

<style scoped>
.device-detail {
  padding: 20px;
}

.page-header {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #ebeef5;
}

.page-header h2 {
  margin: 10px 0 0 0;
  color: #303133;
  font-size: 24px;
  font-weight: 500;
}

.text-right {
  text-align: right;
}

.info-card {
  margin-bottom: 20px;
}

.info-item {
  margin-bottom: 15px;
  display: flex;
  align-items: center;
}

.info-item label {
  font-weight: 500;
  color: #606266;
  min-width: 100px;
  margin-right: 10px;
}

.info-item span {
  color: #303133;
  word-break: break-all;
}

.action-buttons {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

/* 截图对话框样式 */
.screenshot-dialog :deep(.el-dialog__body) {
  padding: 0;
}

.screenshot-dialog-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.screenshot-dialog-actions {
  display: flex;
  gap: 8px;
}

.screenshot-container {
  text-align: center;
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
  border-radius: 4px;
}

.screenshot-image {
  max-width: 100%;
  max-height: 80vh;
  height: auto;
  border-radius: 4px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.no-screenshot {
  color: #909399;
  font-size: 14px;
  padding: 40px;
}

/* 全屏模式下的样式 */
.screenshot-dialog.is-fullscreen .screenshot-container {
  height: calc(100vh - 120px);
}

.screenshot-dialog.is-fullscreen .screenshot-image {
  max-height: calc(100vh - 120px);
}

.debug-info {
  background: #f5f7fa;
  padding: 15px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
  max-height: 400px;
  overflow-y: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .device-detail {
    padding: 10px;
  }

  .page-header {
    margin-bottom: 15px;
    padding-bottom: 15px;
  }

  .info-card {
    margin-bottom: 15px;
  }

  .info-item {
    flex-direction: column;
    align-items: flex-start;
    margin-bottom: 10px;
  }

  .info-item label {
    min-width: auto;
    margin-right: 0;
    margin-bottom: 5px;
  }

  .action-buttons {
    justify-content: center;
  }
}
</style>
