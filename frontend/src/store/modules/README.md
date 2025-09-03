# 设备仪表板状态管理

## 概述

`device-dashboard.ts` 是专门为设备控制台页面创建的状态管理模块，用于持久化存储用户的设置和选择状态。

## 功能特性

### 1. 分组选择持久化
- 自动保存用户当前选择的设备分组
- 页面刷新或重新访问时自动恢复之前的选择
- 支持"未分组设备"（null值）的状态保存

### 2. 刷新间隔设置
- 保存用户设置的截图自动刷新间隔
- 支持0秒（不刷新）到30秒的各种间隔设置
- 页面重新加载时自动恢复设置

### 3. 列数布局设置
- 保存设备卡片的列数显示设置（1-4列）
- 响应式布局配置的持久化存储

## 使用方法

### 在组件中使用

```typescript
import { useDeviceDashboardStore } from '@/store/modules/device-dashboard'

// 在setup函数中
const dashboardStore = useDeviceDashboardStore()

// 获取当前选择的分组ID
const selectedGroupId = computed({
  get: () => dashboardStore.getSelectedGroupId,
  set: (value: number | null) => dashboardStore.setSelectedGroupId(value)
})

// 获取刷新间隔
const refreshInterval = computed({
  get: () => dashboardStore.getRefreshInterval,
  set: (value: number) => dashboardStore.setRefreshInterval(value)
})
```

### 使用Hook函数

```typescript
import { useDeviceDashboardStoreHook } from '@/store/modules/device-dashboard'

const dashboardStore = useDeviceDashboardStoreHook()
```

## 存储结构

数据存储在 localStorage 中，键名为：`${responsiveStorageNameSpace()}device-dashboard`

存储的数据结构：
```typescript
{
  selectedGroupId: number | null,  // 当前选择的分组ID
  refreshInterval: number,         // 刷新间隔（秒）
  columnCount: number             // 列数设置
}
```

## API 方法

### Getters
- `getSelectedGroupId`: 获取当前选择的分组ID
- `getRefreshInterval`: 获取刷新间隔设置
- `getColumnCount`: 获取列数设置

### Actions
- `setSelectedGroupId(groupId)`: 设置当前选择的分组
- `setRefreshInterval(interval)`: 设置刷新间隔
- `setColumnCount(count)`: 设置列数
- `saveToStorage()`: 手动保存到localStorage
- `clearStorage()`: 清除所有存储的状态

## 实现细节

1. **自动保存**: 每次状态变更时自动保存到localStorage
2. **自动恢复**: 页面加载时自动从localStorage恢复状态
3. **类型安全**: 使用TypeScript提供完整的类型支持
4. **响应式**: 基于Pinia的响应式状态管理

## 注意事项

- 状态变更会立即保存到localStorage，无需手动调用保存方法
- 如果localStorage中没有数据，会使用默认值
- 清除存储后会重置为默认状态
