<template>
  <div class="group-panel">
    <div class="panel-header">
      <div class="header-left">
        <h3>分组</h3>
        <el-button
          size="small"
          text
          @click="handleRefresh"
          :loading="loading"
          class="refresh-btn"
        >
          <el-icon><Refresh /></el-icon>
        </el-button>
      </div>
      <el-button size="small" @click="handleAddGroup">添加分组</el-button>
    </div>

    <div class="group-list" v-loading="loading">
      <!-- 待分组设备 -->
      <div
        class="group-item"
        :class="{ active: selectedGroupId === null }"
        @click="handleSelectGroup(null)"
      >
        <div class="group-main">
          <div class="group-info">
            <el-icon class="group-icon"><Monitor /></el-icon>
            <span class="group-name">待分组设备</span>
          </div>
          <span class="device-count">{{ ungroupedDevices }}</span>
        </div>
      </div>

      <!-- 分组列表 -->
      <div
        v-for="group in groupList"
        :key="group.ID"
        class="group-item"
        :class="{ active: selectedGroupId === group.ID }"
      >
        <div class="group-main" @click="handleSelectGroup(group.ID)">
          <div class="group-info">
            <el-icon class="group-icon"><FolderOpened /></el-icon>
            <span class="group-name">{{ group.name }}</span>
          </div>
          <span class="device-count">{{ group.device_count || 0 }}</span>
        </div>

        <!-- 分组操作按钮 -->
        <div class="group-actions" @click.stop>
          <el-dropdown trigger="click" @command="handleGroupAction">
            <el-button size="small" text class="action-btn">
              <el-icon><MoreFilled /></el-icon>
            </el-button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item :command="`edit_${group.ID}`">
                  <el-icon><Edit /></el-icon>
                  编辑
                </el-dropdown-item>
                <el-dropdown-item :command="`delete_${group.ID}`" divided>
                  <el-icon><Delete /></el-icon>
                  删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </div>

    <!-- 添加分组对话框 -->
    <el-dialog v-model="showAddDialog" title="添加分组" width="400px">
      <el-form :model="addForm" label-width="80px">
        <el-form-item label="分组名称" required>
          <el-input v-model="addForm.name" placeholder="请输入分组名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showAddDialog = false">取消</el-button>
          <el-button type="primary" @click="handleConfirmAdd">确定</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 编辑分组对话框 -->
    <el-dialog v-model="showEditDialog" title="编辑分组" width="400px">
      <el-form :model="editForm" label-width="80px">
        <el-form-item label="分组名称" required>
          <el-input v-model="editForm.name" placeholder="请输入分组名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="showEditDialog = false">取消</el-button>
          <el-button type="primary" @click="handleConfirmEdit">确定</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { groupApi, type GroupInfo } from '@/api/device'
import {
  Delete,
  Edit,
  FolderOpened,
  Monitor,
  MoreFilled,
  Refresh
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { reactive, ref } from 'vue'

// Props
interface Props {
  groupList: GroupInfo[]
  selectedGroupId: number | null
  ungroupedDevices: number
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false
})

// Emits
const emit = defineEmits<{
  selectGroup: [groupId: number | null]
  refresh: []
}>()

// 响应式数据
const showAddDialog = ref(false)
const showEditDialog = ref(false)

const addForm = reactive({
  name: ''
})

const editForm = reactive({
  id: 0,
  name: ''
})

// 方法
const handleSelectGroup = (groupId: number | null) => {
  emit('selectGroup', groupId)
}

const handleRefresh = () => {
  emit('refresh')
}

const handleAddGroup = () => {
  addForm.name = ''
  showAddDialog.value = true
}

const handleConfirmAdd = async () => {
  if (!addForm.name.trim()) {
    ElMessage.warning('请输入分组名称')
    return
  }

  try {
    await groupApi.createGroup({ name: addForm.name })

    // 添加成功后，通知父组件刷新数据而不是直接操作本地缓存
    emit('refresh')
    showAddDialog.value = false
    ElMessage.success('分组添加成功')
  } catch (error: any) {
    // 显示后端返回的具体错误信息
    const errorMessage = error?.message || '分组添加失败'
    ElMessage.error(errorMessage)
    console.error('添加分组错误:', error)
  }
}

const handleGroupAction = async (command: string) => {
  const [action, groupIdStr] = command.split('_')
  const groupId = parseInt(groupIdStr)

  switch (action) {
    case 'edit':
      await editGroup(groupId)
      break
    case 'delete':
      await deleteGroup(groupId)
      break
  }
}

const editGroup = async (groupId: number) => {
  const group = props.groupList.find(g => g.ID === groupId)
  if (!group) {
    ElMessage.error('分组不存在')
    return
  }

  editForm.id = groupId
  editForm.name = group.name
  showEditDialog.value = true
}

const handleConfirmEdit = async () => {
  if (!editForm.name.trim()) {
    ElMessage.warning('请输入分组名称')
    return
  }

  try {
    await groupApi.updateGroup(editForm.id, { name: editForm.name })

    // 更新成功后，通知父组件刷新数据而不是直接操作本地缓存
    emit('refresh')
    showEditDialog.value = false
    ElMessage.success('分组更新成功')
  } catch (error: any) {
    // 显示后端返回的具体错误信息
    const errorMessage = error?.message || '分组更新失败'
    ElMessage.error(errorMessage)
    console.error('更新分组错误:', error)
  }
}

const deleteGroup = async (groupId: number) => {
  const group = props.groupList.find(g => g.ID === groupId)
  if (!group) {
    ElMessage.error('分组不存在')
    return
  }

  try {
    await ElMessageBox.confirm(`确定要删除分组"${group.name}"吗？`, '确认删除', {
      type: 'warning'
    })

    await groupApi.deleteGroup(groupId)

    // 删除成功后，通知父组件刷新数据而不是直接操作本地缓存
    emit('refresh')
    ElMessage.success('分组删除成功')
  } catch (error: any) {
    if (error !== 'cancel') {
      // 显示后端返回的具体错误信息
      const errorMessage = error?.message || '分组删除失败'
      ElMessage.error(errorMessage)
      console.error('删除分组错误:', error)
    }
  }
}
</script>

<style scoped>
.group-panel {
  width: 10vw;
  min-width: 200px;
  max-width: 280px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  box-shadow: 2px 0 8px rgba(0, 0, 0, 0.06);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px;
  border-bottom: 1px solid #e4e7ed;
  background: linear-gradient(135deg, #f8f9fa 0%, #ffffff 100%);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.refresh-btn {
  padding: 4px;
  border-radius: 4px;
  color: #606266;
  transition: all 0.3s ease;
}

.refresh-btn:hover {
  background: rgba(64, 158, 255, 0.1);
  color: #409eff;
}

.group-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.group-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  transition: all 0.3s ease;
  border-left: 3px solid transparent;
  position: relative;
}

.group-main {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex: 1;
  cursor: pointer;
}

.group-item:hover {
  background: linear-gradient(135deg, #f5f7fa 0%, #ecf5ff 100%);
  transform: translateX(2px);
}

.group-item.active {
  background: linear-gradient(135deg, #ecf5ff 0%, #e1f3ff 100%);
  border-left-color: #409eff;
  box-shadow: inset 0 0 0 1px rgba(64, 158, 255, 0.1);
}

.group-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.group-icon {
  color: #909399;
  font-size: 18px;
  transition: color 0.3s ease;
}

.group-item.active .group-icon {
  color: #409eff;
}

.group-name {
  font-size: 14px;
  color: #303133;
  font-weight: 500;
}

.device-count {
  font-size: 12px;
  color: #909399;
  background: #f0f2f5;
  padding: 4px 8px;
  border-radius: 12px;
  min-width: 24px;
  text-align: center;
  font-weight: 500;
  transition: all 0.3s ease;
}

.group-item.active .device-count {
  background: #409eff;
  color: #fff;
}

.group-actions {
  opacity: 0;
  transition: opacity 0.3s ease;
  margin-left: 12px;
}

.group-item:hover .group-actions {
  opacity: 1;
}

.action-btn {
  padding: 4px;
  border-radius: 6px;
}

.action-btn:hover {
  background: rgba(64, 158, 255, 0.1);
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .group-panel {
    width: 18vw;
    min-width: 180px;
    max-width: 240px;
  }
}

@media (max-width: 768px) {
  .group-panel {
    width: 100%;
    min-width: unset;
    max-width: unset;
    height: 200px;
  }

  .panel-header {
    padding: 12px 16px;
  }

  .panel-header h3 {
    font-size: 14px;
  }
}

/* 滚动条样式 */
.group-list::-webkit-scrollbar {
  width: 6px;
}

.group-list::-webkit-scrollbar-track {
  background: #f1f1f1;
  border-radius: 3px;
}

.group-list::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 3px;
}

.group-list::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}
</style>
