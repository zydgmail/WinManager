package models

import (
	"time"
	"winmanager-backend/internal/logger"

	"gorm.io/gorm"
)

// Instance 实例模型
type Instance struct {
	gorm.Model
	Uuid            string     `json:"uuid" gorm:"comment:设备唯一标识"`
	OS              string     `json:"os" gorm:"comment:操作系统"`
	Arch            string     `json:"arch" gorm:"comment:架构"`
	Lan             string     `gorm:"unique" json:"lan" gorm:"comment:内网IP"`
	Wan             string     `json:"wan" gorm:"comment:外网IP"`
	Mac             string     `json:"mac" gorm:"comment:MAC地址"`
	Cpu             string     `json:"cpu" gorm:"comment:CPU信息"`
	Cores           int        `json:"cores" gorm:"comment:CPU核心数"`
	Memory          uint64     `json:"memory" gorm:"comment:内存大小"`
	Uptime          uint64     `json:"uptime" gorm:"comment:运行时间"`
	Hostname        string     `json:"hostname" gorm:"comment:主机名"`
	Username        string     `json:"username" gorm:"comment:用户名"`
	Status          int        `json:"status" gorm:"comment:状态"`
	Version         string     `json:"version" gorm:"comment:Agent版本"`
	WatchdogVersion string     `json:"watchdog_version" gorm:"comment:Watchdog版本"`
	LastHeartbeatAt *time.Time `json:"last_heartbeat_at" gorm:"comment:最后心跳时间"`

	// 物理机地址
	BmIp string `json:"bm_ip" gorm:"comment:物理机地址"`

	GroupID      *int `json:"group_id" gorm:"comment:分组ID"` // 允许group_id为空
	Group        Group
	RepairStatus string     `json:"repair_status" gorm:"comment:修复状态"`
	RepairTime   *time.Time `json:"repair_time" gorm:"comment:修复时间"`
}

// CreateOrUpdateInstance 创建或更新实例
func CreateOrUpdateInstance(instance Instance) (uint, error) {
	result := DB.Where(Instance{Lan: instance.Lan}).Assign(instance).FirstOrCreate(&instance)
	if result.Error != nil {
		logger.Errorf("创建或更新实例失败: %v", result.Error)
		return 0, result.Error
	}

	logger.Infof("实例创建或更新成功: ID=%d, LAN=%s", instance.ID, instance.Lan)

	return instance.ID, nil
}

// InstanceListParams 实例列表查询参数
type InstanceListParams struct {
	Page    int    `json:"page" form:"page"`         // 页码
	Size    int    `json:"size" form:"size"`         // 每页大小
	Search  string `json:"search" form:"search"`     // 设备名称搜索
	Status  *int   `json:"status" form:"status"`     // 设备状态
	GroupID *int   `json:"group_id" form:"group_id"` // 分组ID
}

// InstanceListResult 实例列表返回结果
type InstanceListResult struct {
	Devices []Instance `json:"devices"` // 改为 devices 字段
	Total   int64      `json:"total"`
	Page    int        `json:"page"`
	Size    int        `json:"size"`
}

// ListInstances 获取实例列表
func ListInstances(ids []int) ([]Instance, error) {
	var items []Instance

	query := DB.Order("hostname").Preload("Group")
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	if err := query.Find(&items).Error; err != nil {
		logger.Errorf("获取实例列表失败: %v", err)
		return nil, err
	}

	logger.Infof("获取实例列表成功: 数量=%d", len(items))

	return items, nil
}

// ListInstancesWithParams 根据参数获取实例列表（支持搜索和分页）
func ListInstancesWithParams(params InstanceListParams) (*InstanceListResult, error) {
	var items []Instance
	var total int64

	// 构建查询
	query := DB.Model(&Instance{}).Preload("Group")

	// 设备名称搜索
	if params.Search != "" {
		query = query.Where("hostname LIKE ?", "%"+params.Search+"%")
	}

	// 设备状态筛选
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	// 分组筛选
	if params.GroupID != nil {
		if *params.GroupID == 0 {
			// 查询未分组的设备
			query = query.Where("group_id IS NULL")
		} else {
			query = query.Where("group_id = ?", *params.GroupID)
		}
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		logger.Errorf("获取实例总数失败: %v", err)
		return nil, err
	}

	// 设置默认分页参数
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}

	// 分页查询
	offset := (params.Page - 1) * params.Size
	if err := query.Order("hostname").Offset(offset).Limit(params.Size).Find(&items).Error; err != nil {
		logger.Errorf("获取实例列表失败: %v", err)
		return nil, err
	}

	logger.Infof("获取实例列表成功: 总数=%d, 当前页=%d, 每页=%d", total, params.Page, params.Size)

	return &InstanceListResult{
		Devices: items, // 改为 Devices 字段
		Total:   total,
		Page:    params.Page,
		Size:    params.Size,
	}, nil
}

// GetInstance 获取单个实例
func GetInstance(id int) (*Instance, error) {
	var item Instance
	if err := DB.Preload("Group").First(&item, id).Error; err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		return nil, err
	}

	logger.Infof("获取实例成功: ID=%d", id)

	return &item, nil
}

// GetInstances 获取多个实例
func GetInstances(ids []int) ([]Instance, error) {
	var items []Instance
	if err := DB.Preload("Group").Find(&items, ids).Error; err != nil {
		logger.Errorf("获取多个实例失败: %v", err)
		return nil, err
	}

	logger.Infof("获取多个实例成功: 数量=%d", len(items))

	return items, nil
}

// GetInstanceByLan 根据LAN IP获取实例
func GetInstanceByLan(lan string) (*Instance, error) {
	var item Instance
	if err := DB.Where("lan = ?", lan).Preload("Group").First(&item).Error; err != nil {
		logger.Errorf("根据LAN获取实例失败: LAN=%s, 错误=%v", lan, err)
		return nil, err
	}

	logger.Infof("根据LAN获取实例成功: LAN=%s, ID=%d", lan, item.ID)

	return &item, nil
}

// PatchInstance 更新实例
func PatchInstance(id int, data map[string]interface{}) error {
	var item Instance
	if err := DB.First(&item, id).Error; err != nil {
		logger.Errorf("查找实例失败: ID=%d, 错误=%v", id, err)
		return err
	}

	if err := DB.Model(&item).Updates(data).Error; err != nil {
		logger.Errorf("更新实例失败: ID=%d, 错误=%v", id, err)
		return err
	}

	logger.Infof("更新实例成功: ID=%d", id)

	return nil
}

// DeleteInstance 删除实例
func DeleteInstance(id int) error {
	if err := DB.Unscoped().Delete(&Instance{}, id).Error; err != nil {
		logger.Errorf("删除实例失败: ID=%d, 错误=%v", id, err)
		return err
	}

	logger.Infof("删除实例成功: ID=%d", id)

	return nil
}

// ListInstancesByGroupId 根据分组ID获取实例列表
func ListInstancesByGroupId(gid int) ([]Instance, error) {
	var items []Instance

	if err := DB.Where("group_id = ?", gid).Find(&items).Error; err != nil {
		logger.Errorf("根据分组获取实例失败: GroupID=%d, 错误=%v", gid, err)
		return nil, err
	}

	logger.Infof("根据分组获取实例成功: GroupID=%d, 数量=%d", gid, len(items))

	return items, nil
}

// CountInstanceByGroup 统计分组中的实例数量
func CountInstanceByGroup(id int) (int, error) {
	var count int64
	if err := DB.Model(&Instance{}).Where("group_id = ? AND (repair_status IS NULL OR repair_status = '')", id).Count(&count).Error; err != nil {
		logger.Errorf("统计分组实例数量失败: GroupID=%d, 错误=%v", id, err)
		return 0, err
	}

	logger.Infof("统计分组实例数量成功: GroupID=%d, 数量=%d", id, int(count))

	return int(count), nil
}

// UpdateOfflineInstances 更新超时离线的设备状态
func UpdateOfflineInstances(timeoutSeconds int) (int, error) {
	// 计算超时时间点
	timeoutTime := time.Now().Add(-time.Duration(timeoutSeconds) * time.Second)

	// 更新状态为离线(0)的设备
	result := DB.Model(&Instance{}).
		Where("status = ? AND (last_heartbeat_at IS NULL OR last_heartbeat_at < ?)", 1, timeoutTime).
		Update("status", 0)

	if result.Error != nil {
		logger.Errorf("更新离线设备状态失败: 超时时间=%d秒, 错误=%v", timeoutSeconds, result.Error)
		return 0, result.Error
	}

	affectedRows := int(result.RowsAffected)
	if affectedRows > 0 {
		logger.Infof("更新离线设备状态成功: 超时时间=%d秒, 影响行数=%d", timeoutSeconds, affectedRows)
	}

	return affectedRows, nil
}
