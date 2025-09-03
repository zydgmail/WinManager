<!-- eslint-disable vue/attributes-order -->
<!-- eslint-disable prettier/prettier -->
<template>
  <div class="device-list">
    <!-- 搜索和操作栏 -->
    <div class="search-bar">
      <el-row :gutter="16" align="middle">
        <el-col :span="4">
          <el-input
            v-model="searchForm.search"
            placeholder="请输入设备名称"
            clearable
            @keyup.enter="handleSearch"
          />
        </el-col>
        <el-col :span="3">
          <el-select
            v-model="searchForm.status"
            placeholder="设备状态"
            clearable
            @change="handleSearch"
          >
            <el-option label="全部状态" value="" />
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
          </el-select>
        </el-col>
        <el-col :span="3">
          <el-select
            v-model="searchForm.group_id"
            placeholder="设备分组"
            clearable
            @change="handleSearch"
          >
            <el-option label="全部分组" value="" />
            <el-option
              v-for="group in groupList"
              :key="group.id || group.ID"
              :label="group.name"
              :value="group.id || group.ID"
            />
          </el-select>
        </el-col>
        <el-col :span="4">
          <div class="search-buttons">
            <el-button type="primary" @click="handleSearch">查询</el-button>
            <el-button @click="handleReset">重置</el-button>
          </div>
        </el-col>
        <el-col :span="8" />
        <el-col :span="2">
          <el-dropdown
            :disabled="selectedDevices.length === 0"
            @command="handleBatchCommand"
          >
            <el-button type="warning" :disabled="selectedDevices.length === 0">
              批量操作<el-icon class="el-icon--right"><arrow-down /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="changeGroup"
                  >修改分组</el-dropdown-item
                >
                <el-dropdown-item command="changeStatus"
                  >修改状态</el-dropdown-item
                >
                <el-dropdown-item divided command="batchDelete"
                  >批量删除</el-dropdown-item
                >
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </el-col>
      </el-row>
    </div>

    <!-- 设备表格 -->
    <div class="table-container">
      <el-table
        v-loading="loading"
        :data="deviceList"
        style="width: 100%"
        :header-cell-style="{ background: '#f5f7fa', color: '#606266' }"
        table-layout="auto"
        :fit="true"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="50" />
        <el-table-column label="ID" width="60">
          <template #default="{ row }">
            {{ row.ID || row.id }}
          </template>
        </el-table-column>
        <el-table-column
          prop="hostname"
          label="设备名称"
          min-width="120"
          :show-overflow-tooltip="true"
        >
          <template #default="{ row }">
            <el-link type="primary" @click="handleViewDetail(row.ID || row.id)">
              {{ row.hostname }}
            </el-link>
          </template>
        </el-table-column>
        <el-table-column prop="lan" label="内网IP" min-width="110" />
        <el-table-column prop="wan" label="外网IP" min-width="110">
          <template #default="{ row }">
            {{ row.wan || "-" }}
          </template>
        </el-table-column>
        <el-table-column
          prop="mac"
          label="MAC地址"
          min-width="130"
          :show-overflow-tooltip="true"
        >
          <template #default="{ row }">
            {{ row.mac || "-" }}
          </template>
        </el-table-column>
        <el-table-column prop="os" label="系统" min-width="80" />
        <el-table-column
          prop="cpu"
          label="CPU信息"
          min-width="200"
          :show-overflow-tooltip="true"
        />
        <el-table-column prop="memory" label="内存" min-width="80">
          <template #default="{ row }">
            {{ formatMemory(row.memory) }}
          </template>
        </el-table-column>
        <el-table-column prop="cores" label="核心" min-width="60" />
        <el-table-column
          prop="uptime"
          label="运行时间"
          min-width="100"
          :show-overflow-tooltip="true"
        >
          <template #default="{ row }">
            {{ formatUptime(row.uptime) }}
          </template>
        </el-table-column>
        <el-table-column prop="group" label="分组" min-width="90">
          <template #default="{ row }">
            {{
              (row.Group && row.Group.name) ||
              (row.group && row.group.name) ||
              "未分组"
            }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" min-width="70">
          <template #default="{ row }">
            <el-tag
              :type="row.status === 1 ? 'success' : 'danger'"
              size="small"
            >
              {{ row.status === 1 ? "在线" : "离线" }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后心跳" min-width="140">
          <template #default="{ row }">
            {{
              formatTime(
                row.last_heartbeat_at ||
                  row.LastHeartbeatAt ||
                  row.UpdatedAt ||
                  row.updated_at,
                "YYYY-MM-DD HH:mm:ss"
              )
            }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template #default="{ row }">
            <el-dropdown @command="command => handleRowCommand(command, row)">
              <el-button type="primary" size="small">
                操作<el-icon class="el-icon--right"><arrow-down /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="detail">查看详情</el-dropdown-item>
                  <el-dropdown-item command="screenshot"
                    >获取截图</el-dropdown-item
                  >
                  <el-dropdown-item command="remote">远程连接</el-dropdown-item>
                  <el-dropdown-item divided command="delete"
                    >删除设备</el-dropdown-item
                  >
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- 分页 -->
    <div class="pagination-container">
      <div class="pagination-info">
        <span>共 {{ pagination.total }} 条</span>
        <span v-if="selectedDevices.length > 0" class="selected-info">
          已选择 {{ selectedDevices.length }} 项
        </span>
      </div>
      <el-pagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.size"
        :page-sizes="[10, 20, 50, 100]"
        :total="pagination.total"
        layout="sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>

    <!-- 修改分组对话框 -->
    <el-dialog v-model="showChangeGroupDialog" title="修改分组" width="400px">
      <el-form>
        <el-form-item label="目标分组">
          <el-select
            v-model="targetGroupId"
            placeholder="请选择分组"
            style="width: 100%"
          >
            <el-option label="未分组" value="" />
            <el-option
              v-for="group in groupList"
              :key="group.id || group.ID"
              :label="group.name"
              :value="group.id || group.ID"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showChangeGroupDialog = false">取消</el-button>
        <el-button type="primary" @click="handleChangeGroup">确定</el-button>
      </template>
    </el-dialog>

    <!-- 修改状态对话框 -->
    <el-dialog v-model="showChangeStatusDialog" title="修改状态" width="400px">
      <el-form>
        <el-form-item label="目标状态">
          <el-select
            v-model="targetStatus"
            placeholder="请选择状态"
            style="width: 100%"
          >
            <el-option label="在线" :value="1" />
            <el-option label="离线" :value="0" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showChangeStatusDialog = false">取消</el-button>
        <el-button type="primary" @click="handleChangeStatus">确定</el-button>
      </template>
    </el-dialog>

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
import {
  deviceApi,
  groupApi,
  type DeviceInfo,
  type GroupInfo
} from "@/api/device";
import { formatMemory, formatTime, formatUptime } from "@/utils/format";
import { ArrowDown } from "@element-plus/icons-vue";
import { ElMessage, ElMessageBox } from "element-plus";
import { onMounted, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";

const router = useRouter();

// 响应式数据
const loading = ref(false);
const deviceList = ref<DeviceInfo[]>([]);
const groupList = ref<GroupInfo[]>([]);
const selectedDevices = ref<DeviceInfo[]>([]);

// 搜索表单
const searchForm = reactive({
  search: "",
  group_id: "" as string | number,
  status: "" as string | number
});

// 分页
const pagination = reactive({
  page: 1,
  size: 20,
  total: 0
});

// 对话框状态
const showChangeGroupDialog = ref(false);
const showChangeStatusDialog = ref(false);
const targetGroupId = ref<string | number>("");
const targetStatus = ref<string | number>("");
const showScreenshotDialog = ref(false);
const screenshotLoading = ref(false);
const screenshotData = ref("");
const isFullscreen = ref(false);
const screenshotDialogWidth = ref("90%");

// 获取设备列表
const getDeviceList = async () => {
  loading.value = true;
  try {
    const params = {
      page: pagination.page,
      size: pagination.size,
      search: searchForm.search || undefined,
      status: searchForm.status,
      group_id: searchForm.group_id
    };

    // 过滤掉空值
    Object.keys(params).forEach(key => {
      if (params[key] === undefined || params[key] === null || params[key] === "") {
        delete params[key];
      }
    });

    const response = await deviceApi.getDeviceList(params);
    // 使用标准的响应格式
    deviceList.value = response.data.devices || [];
    pagination.total = response.data.total || 0;
    pagination.page = response.data.page || 1;
    pagination.size = response.data.size || 20;
  } catch (error) {
    ElMessage.error("获取设备列表失败");
    console.error("获取设备列表错误:", error);
  } finally {
    loading.value = false;
  }
};

// 获取分组列表
const getGroupList = async () => {
  try {
    const response = await groupApi.getGroupList();
    groupList.value = response.data.groups;
  } catch (error) {
    ElMessage.error("获取分组列表失败");
  }
};

// 搜索
const handleSearch = () => {
  pagination.page = 1;
  getDeviceList();
};

// 重置
const handleReset = () => {
  // 重置搜索表单
  Object.assign(searchForm, {
    search: "",
    group_id: "",
    status: ""
  });
  // 重置分页
  pagination.page = 1;
  // 重新获取数据
  getDeviceList();
};

// 查看详情
const handleViewDetail = (id: number) => {
  router.push(`/device/detail/${id}`);
};

// 截图
const handleScreenshot = async (id: number) => {
  showScreenshotDialog.value = true;
  screenshotLoading.value = true;
  isFullscreen.value = false;

  try {
    const response = await deviceApi.getDeviceScreenshot(id);
    // 将Blob转换为URL用于显示
    screenshotData.value = URL.createObjectURL(response);
  } catch (error) {
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
    link.download = `screenshot-${Date.now()}.jpg`;
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

// 删除设备
const handleDelete = async (id: number) => {
  try {
    await ElMessageBox.confirm("确定要删除这个设备吗？", "确认删除", {
      type: "warning"
    });
    await deviceApi.deleteDevice(id);
    ElMessage.success("删除成功");
    getDeviceList();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("删除失败");
    }
  }
};

// 选择变化
const handleSelectionChange = (selection: DeviceInfo[]) => {
  selectedDevices.value = selection;
};

// 批量操作命令处理
const handleBatchCommand = (command: string) => {
  if (selectedDevices.value.length === 0) {
    ElMessage.warning("请先选择设备");
    return;
  }

  switch (command) {
    case "changeGroup":
      showChangeGroupDialog.value = true;
      break;
    case "changeStatus":
      showChangeStatusDialog.value = true;
      break;
    case "batchDelete":
      handleBatchDelete();
      break;
  }
};

// 行操作命令处理
const handleRowCommand = (command: string, row: DeviceInfo) => {
  const deviceId = row.ID || row.id;
  switch (command) {
    case "detail":
      handleViewDetail(deviceId);
      break;
    case "screenshot":
      handleScreenshot(deviceId);
      break;
    case "remote":
      handleRemoteConnect(deviceId);
      break;
    case "delete":
      handleDelete(deviceId);
      break;
  }
};

// 修改分组
const handleChangeGroup = async () => {
  try {
    await deviceApi.moveDeviceToGroup({
      ids: selectedDevices.value.map(d => d.ID || d.id),
      group_id: targetGroupId.value === "" ? 0 : Number(targetGroupId.value)
    });
    ElMessage.success("修改分组成功");
    showChangeGroupDialog.value = false;
    targetGroupId.value = "";
    selectedDevices.value = [];
    getDeviceList();
  } catch (error) {
    ElMessage.error("修改分组失败");
  }
};

// 修改状态
const handleChangeStatus = async () => {
  if (targetStatus.value === "" || targetStatus.value === null || targetStatus.value === undefined) {
    ElMessage.warning("请选择目标状态");
    return;
  }

  try {
    // 批量更新设备状态
    for (const device of selectedDevices.value) {
      await deviceApi.updateDevice(device.ID || device.id, {
        status: targetStatus.value
      });
    }
    ElMessage.success("修改状态成功");
    showChangeStatusDialog.value = false;
    targetStatus.value = "";
    selectedDevices.value = [];
    getDeviceList();
  } catch (error) {
    ElMessage.error("修改状态失败");
  }
};

// 远程连接
const handleRemoteConnect = (id: number) => {
  // 跳转到远程控制台页面
  router.push(`/device/console/${id}`);
};

// 批量删除
const handleBatchDelete = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要删除选中的 ${selectedDevices.value.length} 个设备吗？`,
      "确认删除",
      { type: "warning" }
    );

    for (const device of selectedDevices.value) {
      await deviceApi.deleteDevice(device.ID || device.id);
    }

    ElMessage.success("批量删除成功");
    selectedDevices.value = [];
    getDeviceList();
  } catch (error) {
    if (error !== "cancel") {
      ElMessage.error("批量删除失败");
    }
  }
};

// 分页变化
const handleSizeChange = (size: number) => {
  pagination.size = size;
  pagination.page = 1;
  getDeviceList();
};

const handleCurrentChange = (page: number) => {
  pagination.page = page;
  getDeviceList();
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
  getDeviceList();
  getGroupList();
});
</script>

<style scoped>
.device-list {
  padding: 16px;
  background: #f5f7fa;
  min-height: 100vh;
}

.search-bar {
  margin-bottom: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.search-buttons {
  display: flex;
  gap: 8px;
}

.table-container {
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.pagination-container {
  margin-top: 16px;
  padding: 16px;
  background: #fff;
  border-radius: 4px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pagination-info {
  display: flex;
  align-items: center;
  gap: 16px;
  color: #606266;
  font-size: 14px;
}

.selected-info {
  color: #409eff;
  font-weight: 500;
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

:deep(.el-table) {
  border: none;
  table-layout: auto;
}

:deep(.el-table__header) {
  background: #f5f7fa;
}

:deep(.el-table th) {
  background: #f5f7fa !important;
  border-bottom: 1px solid #ebeef5;
  padding: 8px 12px;
  font-size: 13px;
  white-space: nowrap;
}

:deep(.el-table td) {
  border-bottom: 1px solid #ebeef5;
  padding: 8px 12px;
  font-size: 13px;
}

:deep(.el-table__row:hover > td) {
  background-color: #f5f7fa !important;
}

:deep(.el-pagination) {
  --el-pagination-font-size: 14px;
}

/* 响应式表格样式 */
@media (max-width: 1200px) {
  :deep(.el-table th),
  :deep(.el-table td) {
    padding: 6px 8px;
    font-size: 12px;
  }
}

@media (max-width: 768px) {
  .table-container {
    overflow-x: auto;
  }

  :deep(.el-table) {
    min-width: 1200px;
  }

  :deep(.el-table th),
  :deep(.el-table td) {
    padding: 4px 6px;
    font-size: 11px;
  }
}
</style>
