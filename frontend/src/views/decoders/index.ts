// 视频解码器导出
export { default as JMuxerDecoder } from './JMuxerDecoder.vue'
export { default as WebCodecsDecoder } from './WebCodecsDecoder.vue'
export { default as MSEDecoder } from './MSEDecoder.vue'

// 解码器类型枚举
export enum DecoderType {
  JMUXER = 'jmuxer',
  WEBCODECS = 'webcodecs',
  MSE = 'mse'
}

// 解码器配置接口
export interface DecoderConfig {
  deviceId: string | number
  deviceIp?: string
  autoStart?: boolean
  width?: number
  height?: number
}

// 解码器事件接口
export interface DecoderEvents {
  connected: []
  disconnected: []
  error: [message: string]
}
