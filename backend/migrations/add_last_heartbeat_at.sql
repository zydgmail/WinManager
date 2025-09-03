-- 添加最后心跳时间字段
-- 执行时间: 2025-06-19

-- 为instances表添加last_heartbeat_at字段
ALTER TABLE instances ADD COLUMN last_heartbeat_at DATETIME DEFAULT NULL COMMENT '最后心跳时间';

-- 创建索引以提高查询性能
CREATE INDEX idx_instances_last_heartbeat_at ON instances(last_heartbeat_at);
CREATE INDEX idx_instances_status_heartbeat ON instances(status, last_heartbeat_at);

-- 将现有的updated_at值复制到last_heartbeat_at（仅对在线设备）
UPDATE instances 
SET last_heartbeat_at = updated_at 
WHERE status = 1;

-- 添加注释说明
ALTER TABLE instances MODIFY COLUMN last_heartbeat_at DATETIME DEFAULT NULL COMMENT '最后心跳时间，专门记录Agent心跳，与updated_at分离';
