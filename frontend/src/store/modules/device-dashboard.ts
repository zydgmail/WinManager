import { defineStore } from "pinia";
import { storageLocal, responsiveStorageNameSpace } from "../utils";

export interface DeviceDashboardState {
  selectedGroupId: number | null;
  refreshInterval: number;
  columnCount: number;
}

export const useDeviceDashboardStore = defineStore("device-dashboard", {
  state: (): DeviceDashboardState => ({
    // 当前选择的分组ID，从localStorage恢复
    selectedGroupId: storageLocal().getItem<DeviceDashboardState>(
      `${responsiveStorageNameSpace()}device-dashboard`
    )?.selectedGroupId ?? null,
    
    // 刷新间隔（秒），从localStorage恢复
    refreshInterval: storageLocal().getItem<DeviceDashboardState>(
      `${responsiveStorageNameSpace()}device-dashboard`
    )?.refreshInterval ?? 0,
    
    // 列数设置，从localStorage恢复
    columnCount: storageLocal().getItem<DeviceDashboardState>(
      `${responsiveStorageNameSpace()}device-dashboard`
    )?.columnCount ?? 4
  }),

  getters: {
    getSelectedGroupId(state) {
      return state.selectedGroupId;
    },
    getRefreshInterval(state) {
      return state.refreshInterval;
    },
    getColumnCount(state) {
      return state.columnCount;
    }
  },

  actions: {
    // 设置当前选择的分组
    setSelectedGroupId(groupId: number | null) {
      this.selectedGroupId = groupId;
      this.saveToStorage();
    },

    // 设置刷新间隔
    setRefreshInterval(interval: number) {
      this.refreshInterval = interval;
      this.saveToStorage();
    },

    // 设置列数
    setColumnCount(count: number) {
      this.columnCount = count;
      this.saveToStorage();
    },

    // 保存到localStorage
    saveToStorage() {
      storageLocal().setItem(
        `${responsiveStorageNameSpace()}device-dashboard`,
        {
          selectedGroupId: this.selectedGroupId,
          refreshInterval: this.refreshInterval,
          columnCount: this.columnCount
        }
      );
    },

    // 清除存储的状态
    clearStorage() {
      storageLocal().removeItem(`${responsiveStorageNameSpace()}device-dashboard`);
      this.selectedGroupId = null;
      this.refreshInterval = 0;
      this.columnCount = 4;
    }
  }
});

// Hook函数，方便在组件中使用
export function useDeviceDashboardStoreHook() {
  return useDeviceDashboardStore();
}
