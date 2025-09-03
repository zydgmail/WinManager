import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import "dayjs/locale/zh-cn";

dayjs.extend(relativeTime);
dayjs.locale("zh-cn");

/**
 * 格式化时间
 * @param time 时间字符串或时间戳
 * @param format 格式化模板，默认为 'YYYY-MM-DD HH:mm:ss'
 * @returns 格式化后的时间字符串
 */
export function formatTime(time: string | number | Date, format = "YYYY-MM-DD HH:mm:ss"): string {
  if (!time) return "-";
  return dayjs(time).format(format);
}

/**
 * 格式化相对时间
 * @param time 时间字符串或时间戳
 * @returns 相对时间字符串，如 "2小时前"
 */
export function formatRelativeTime(time: string | number | Date): string {
  if (!time) return "-";
  return dayjs(time).fromNow();
}

/**
 * 格式化内存大小
 * @param bytes 字节数
 * @returns 格式化后的内存大小字符串
 */
export function formatMemory(bytes: number): string {
  if (!bytes || bytes === 0) return "0 B";
  
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i];
}

/**
 * 格式化文件大小
 * @param bytes 字节数
 * @returns 格式化后的文件大小字符串
 */
export function formatFileSize(bytes: number): string {
  return formatMemory(bytes);
}

/**
 * 格式化运行时间
 * @param seconds 秒数
 * @returns 格式化后的运行时间字符串
 */
export function formatUptime(seconds: number): string {
  if (!seconds || seconds === 0) return "0秒";
  
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = seconds % 60;
  
  const parts = [];
  if (days > 0) parts.push(`${days}天`);
  if (hours > 0) parts.push(`${hours}小时`);
  if (minutes > 0) parts.push(`${minutes}分钟`);
  if (secs > 0 || parts.length === 0) parts.push(`${secs}秒`);
  
  return parts.join("");
}

/**
 * 格式化百分比
 * @param value 数值
 * @param total 总数
 * @param decimals 小数位数，默认为2
 * @returns 格式化后的百分比字符串
 */
export function formatPercentage(value: number, total: number, decimals = 2): string {
  if (!total || total === 0) return "0%";
  const percentage = (value / total) * 100;
  return `${percentage.toFixed(decimals)}%`;
}

/**
 * 格式化数字，添加千分位分隔符
 * @param num 数字
 * @returns 格式化后的数字字符串
 */
export function formatNumber(num: number): string {
  if (num === null || num === undefined) return "-";
  return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ",");
}

/**
 * 格式化IP地址
 * @param ip IP地址字符串
 * @returns 格式化后的IP地址
 */
export function formatIP(ip: string): string {
  if (!ip) return "-";
  return ip;
}

/**
 * 格式化MAC地址
 * @param mac MAC地址字符串
 * @returns 格式化后的MAC地址
 */
export function formatMAC(mac: string): string {
  if (!mac) return "-";
  // 将MAC地址格式化为 XX:XX:XX:XX:XX:XX 格式
  return mac.replace(/(.{2})/g, "$1:").slice(0, -1).toUpperCase();
}

/**
 * 格式化状态文本
 * @param status 状态值
 * @param statusMap 状态映射表
 * @returns 状态文本
 */
export function formatStatus(status: number | string, statusMap: Record<string | number, string>): string {
  return statusMap[status] || "未知";
}

/**
 * 格式化设备状态
 * @param status 设备状态
 * @returns 状态文本
 */
export function formatDeviceStatus(status: number): string {
  const statusMap = {
    0: "离线",
    1: "在线",
    2: "维护中",
    3: "故障"
  };
  return formatStatus(status, statusMap);
}

/**
 * 格式化Agent状态
 * @param status Agent状态
 * @returns 状态文本
 */
export function formatAgentStatus(status: boolean): string {
  return status ? "正常" : "异常";
}

/**
 * 截断文本
 * @param text 文本
 * @param maxLength 最大长度
 * @param suffix 后缀，默认为 "..."
 * @returns 截断后的文本
 */
export function truncateText(text: string, maxLength: number, suffix = "..."): string {
  if (!text) return "";
  if (text.length <= maxLength) return text;
  return text.substring(0, maxLength) + suffix;
}

/**
 * 格式化版本号
 * @param version 版本号
 * @returns 格式化后的版本号
 */
export function formatVersion(version: string): string {
  if (!version) return "未知";
  return version;
}

/**
 * 格式化CPU信息
 * @param cpu CPU信息
 * @param cores CPU核心数
 * @returns 格式化后的CPU信息
 */
export function formatCPU(cpu: string, cores?: number): string {
  if (!cpu) return "-";
  if (cores) {
    return `${cpu} (${cores}核)`;
  }
  return cpu;
}
