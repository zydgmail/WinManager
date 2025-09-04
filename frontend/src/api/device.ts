import { http } from "@/utils/http";

// 设备信息接口（匹配后端返回的字段名）
export interface DeviceInfo {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt?: string | null;
  uuid: string;
  os: string;
  arch: string;
  lan: string;
  wan: string;
  mac: string;
  cpu: string;
  cores: number;
  memory: number;
  uptime: number;
  hostname: string;
  username: string;
  status: number;
  version: string;
  agent_status?: boolean;
  watchdog_version: string;
  last_heartbeat_at?: string | null;
  bm_ip?: string;
  group_id?: number | null;
  Group?: GroupInfo;
  repair_status?: string;
  repair_time?: string | null;

  // 兼容前端使用的小写字段名
  id?: number;
  created_at?: string;
  updated_at?: string;
  LastHeartbeatAt?: string | null;
  group?: GroupInfo;

  // 前端添加的字段
  screenshot?: string;
}

// 分组信息接口（匹配后端返回的字段名）
export interface GroupInfo {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt?: string | null;
  name: string;
  total: number;

  // 兼容前端使用的小写字段名
  id?: number;
  created_at?: string;
  updated_at?: string;

  // 前端添加的字段
  device_count?: number;
}

// 设备列表查询参数
export interface DeviceListParams {
  page?: number;
  size?: number;
  group_id?: string | number;
  status?: string | number;
  search?: string;
}

// 设备列表返回结果（标准响应结构）
export interface DeviceListResult {
  devices: DeviceInfo[];  // 设备列表
  total: number;
  page: number;
  size: number;
}

// 后端API响应结构
export interface DeviceListResponse {
  data: DeviceListResult;
}

// API响应包装器
export interface ApiResponse<T> {
  data: T;
}

// 分组列表查询参数
export interface GroupListParams {
  page?: number;
  size?: number;
  search?: string;
}

// 分组列表返回结果
export interface GroupListResult {
  groups: GroupInfo[];  // 分组列表
  total: number;
  page: number;
  size: number;
}

// 创建分组参数
export interface CreateGroupParams {
  name: string;
}

// 更新分组参数
export interface UpdateGroupParams {
  name?: string;
}

// 移动设备到分组参数
export interface MoveDeviceToGroupParams {
  ids: number[];
  group_id: number;
}

// 设备管理API
export const deviceApi = {
  // 获取设备列表
  getDeviceList: (params?: DeviceListParams) => {
    return http.request<ApiResponse<DeviceListResult>>("get", "/api/instances", { params });
  },

  // 获取单个设备信息
  getDevice: (id: number) => {
    return http.request<ApiResponse<DeviceInfo>>("get", `/api/instances/${id}`);
  },

  // 更新设备信息
  updateDevice: (id: number, data: Partial<DeviceInfo>) => {
    return http.request<DeviceInfo>("patch", `/api/instances/${id}`, { data });
  },

  // 删除设备
  deleteDevice: (id: number) => {
    return http.request<void>("delete", `/api/instances/${id}`);
  },

  // 获取设备截图
  getDeviceScreenshot: (id: number) => {
    return http.request<Blob>("post", `/api/agent/${id}/screenshot`, {
      responseType: 'blob',
      params: {
        t: Date.now() // 添加时间戳避免缓存
      }
    });
  },

  // 执行设备命令
  executeDeviceCommand: (id: number, command: string) => {
    return http.request<{ output: string }>("post", `/api/instances/${id}/execute`, {
      data: { command }
    });
  },

  // 获取设备系统信息
  getDeviceSystemInfo: (id: number) => {
    return http.request<any>("get", `/api/instances/${id}/system/info`);
  },

  // 移动设备到分组
  moveDeviceToGroup: (data: MoveDeviceToGroupParams) => {
    return http.request<void>("patch", "/api/instances/move-group", { data });
  },

  // 获取设备详细信息（用于远程控制台）
  getDeviceInfo: (id: number) => {
    return http.request<ApiResponse<DeviceInfo>>("get", `/api/instances/${id}/system/info`);
  },

  // 发送键盘指令（通过代理到Agent）
  sendKeyboard: (id: number, command: string) => {
    return http.request<void>("get", `/api/proxy/${id}/api/keyboard?cmd=${command}`);
  },

  // 发送鼠标事件（通过WebSocket实现，这里暂时保留接口但不实际调用）
  sendMouseEvent: (_id: number, _event: { type: string; x: number; y: number; button: number }) => {
    // 鼠标事件通过WebSocket或Guacamole处理，不需要HTTP API
    return Promise.resolve();
  },

  // 重启设备（通过代理到Agent）
  rebootDevice: (id: number) => {
    return http.request<void>("post", `/api/proxy/${id}/api/reboot`);
  },

  // 启动视频流（使用设备ID）
  startStream: (id: number) => {
    return http.request<void>("get", `/api/stream/${id}/start`);
  },

  // 停止视频流（使用设备ID）
  stopStream: (id: number) => {
    return http.request<void>("get", `/api/stream/${id}/stop`);
  }
};

// 后端实际响应结构
export interface BackendResponse<T> {
  code: number;
  msg: string;
  data: T;
}

// 分组管理API
export const groupApi = {
  // 获取分组列表
  getGroupList: (params?: GroupListParams) => {
    return http.request<BackendResponse<GroupListResult>>("get", "/api/groups", { params });
  },

  // 获取单个分组信息
  getGroup: (id: number) => {
    return http.request<BackendResponse<GroupInfo>>("get", `/api/groups/${id}`);
  },

  // 创建分组
  createGroup: (data: CreateGroupParams) => {
    return http.request<BackendResponse<GroupInfo>>("post", "/api/groups", { data });
  },

  // 更新分组
  updateGroup: (id: number, data: UpdateGroupParams) => {
    return http.request<BackendResponse<GroupInfo>>("patch", `/api/groups/${id}`, { data });
  },

  // 删除分组
  deleteGroup: (id: number) => {
    return http.request<BackendResponse<null>>("delete", `/api/groups/${id}`);
  }
};

// WebSocket相关API
export const websocketApi = {
  // 获取WebSocket连接统计
  getWebSocketStats: () => {
    return http.request<any>("get", "/api/websocket/stats");
  },

  // 获取设备WebSocket连接
  getDeviceConnections: (id: number) => {
    return http.request<any>("get", `/api/websocket/instances/${id}`);
  },

  // 关闭设备WebSocket连接
  closeDeviceConnections: (id: number) => {
    return http.request<void>("delete", `/api/websocket/instances/${id}`);
  },

  // 关闭指定WebSocket连接
  closeWebSocketConnection: (connId: string) => {
    return http.request<void>("delete", `/api/websocket/connections/${connId}`);
  }
};

// 系统API
export const systemApi = {
  // 健康检查
  healthCheck: () => {
    return http.request<{ status: string; message: string }>("get", "/api/health");
  },

  // 版本信息
  getVersion: () => {
    return http.request<{ version: string; name: string }>("get", "/api/version");
  }
};
