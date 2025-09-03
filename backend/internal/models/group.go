package models

import (
	"errors"
	"winmanager-backend/internal/logger"

	"gorm.io/gorm"
)

// Group 分组模型
type Group struct {
	gorm.Model
	Name  string `json:"name" gorm:"comment:分组名称"`
	Total int    `json:"total" gorm:"comment:实例总数"`
}

// GroupWithDeviceCount 包含设备数量的分组结构
type GroupWithDeviceCount struct {
	Group
	DeviceCount int `json:"device_count" gorm:"-"` // 不存储到数据库，仅用于返回
}

// GroupListParams 分组列表查询参数
type GroupListParams struct {
	Page   int    `json:"page" form:"page"`     // 页码
	Size   int    `json:"size" form:"size"`     // 每页大小
	Search string `json:"search" form:"search"` // 分组名称搜索
}

// GroupListResult 分组列表返回结果
type GroupListResult struct {
	Groups []GroupWithDeviceCount `json:"groups"` // 分组列表（包含设备数量）
	Total  int64                  `json:"total"`  // 总数
	Page   int                    `json:"page"`   // 当前页
	Size   int                    `json:"size"`   // 每页大小
}

// ListGroups 获取分组列表
func ListGroups(ids []int) ([]Group, error) {
	var items []Group

	query := DB.Order("name")
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	if err := query.Find(&items).Error; err != nil {
		logger.Errorf("获取分组列表失败: %v", err)
		return nil, err
	}

	logger.Infof("获取分组列表成功: 数量=%d", len(items))

	return items, nil
}

// ListGroupsWithDeviceCount 获取包含设备数量的分组列表
func ListGroupsWithDeviceCount(ids []int) ([]GroupWithDeviceCount, error) {
	var baseGroups []Group
	var groups []GroupWithDeviceCount

	query := DB.Order("name")
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	if err := query.Find(&baseGroups).Error; err != nil {
		logger.Errorf("获取基础分组列表失败: %v", err)
		return nil, err
	}

	// 为每个分组计算设备数量
	for _, group := range baseGroups {
		var deviceCount int64
		if err := DB.Model(&Instance{}).Where("group_id = ?", group.ID).Count(&deviceCount).Error; err != nil {
			logger.Errorf("计算分组 %d 的设备数量失败: %v", group.ID, err)
			deviceCount = 0
		}

		groups = append(groups, GroupWithDeviceCount{
			Group:       group,
			DeviceCount: int(deviceCount),
		})
	}

	logger.Infof("获取分组列表(含设备数量)成功: 数量=%d", len(groups))

	return groups, nil
}

// GetGroupList 获取分组列表（支持分页，包含设备数量）
func GetGroupList(params *GroupListParams) (*GroupListResult, error) {
	// 设置默认值
	if params.Page <= 0 {
		params.Page = 1
	}
	if params.Size <= 0 {
		params.Size = 20
	}

	var items []GroupWithDeviceCount
	var total int64

	// 构建基础查询（用于计算总数）
	baseQuery := DB.Model(&Group{})
	if params.Search != "" {
		baseQuery = baseQuery.Where("name LIKE ?", "%"+params.Search+"%")
	}

	// 获取总数
	if err := baseQuery.Count(&total).Error; err != nil {
		logger.Errorf("获取分组总数失败: %v", err)
		return nil, err
	}

	// 先获取基础分组数据
	var baseGroups []Group
	baseQuery = DB.Model(&Group{})
	if params.Search != "" {
		baseQuery = baseQuery.Where("name LIKE ?", "%"+params.Search+"%")
	}

	// 分页查询基础分组
	offset := (params.Page - 1) * params.Size
	if err := baseQuery.Order("created_at DESC").Offset(offset).Limit(params.Size).Find(&baseGroups).Error; err != nil {
		logger.Errorf("获取基础分组列表失败: %v", err)
		return nil, err
	}

	// 为每个分组计算设备数量
	for _, group := range baseGroups {
		var deviceCount int64
		if err := DB.Model(&Instance{}).Where("group_id = ?", group.ID).Count(&deviceCount).Error; err != nil {
			logger.Errorf("计算分组 %d 的设备数量失败: %v", group.ID, err)
			deviceCount = 0
		}

		items = append(items, GroupWithDeviceCount{
			Group:       group,
			DeviceCount: int(deviceCount),
		})

		logger.Infof("分组 %s (ID: %d) 设备数量: %d", group.Name, group.ID, deviceCount)
	}

	logger.Infof("获取分组列表成功: 页码=%d, 大小=%d, 总数=%d", params.Page, len(items), total)

	return &GroupListResult{
		Groups: items,
		Total:  total,
		Page:   params.Page,
		Size:   params.Size,
	}, nil
}

// GetGroup 获取单个分组
func GetGroup(id int) (*Group, error) {
	var item Group
	if err := DB.First(&item, id).Error; err != nil {
		logger.Errorf("获取分组失败: ID=%d, 错误=%v", id, err)
		return nil, err
	}

	logger.Infof("获取分组成功: ID=%d, 名称=%s", id, item.Name)

	return &item, nil
}

// CreateGroup 创建分组
func CreateGroup(name string) (*Group, error) {
	item := Group{
		Name: name,
	}
	if err := DB.Create(&item).Error; err != nil {
		logger.Errorf("创建分组失败: 名称=%s, 错误=%v", name, err)
		return nil, err
	}

	logger.Infof("创建分组成功: ID=%d, 名称=%s", item.ID, name)

	return &item, nil
}

// PatchGroup 更新分组
func PatchGroup(id int, name string) error {
	if err := DB.Model(&Group{}).Where("id = ?", id).Update("name", name).Error; err != nil {
		logger.Errorf("更新分组失败: ID=%d, 名称=%s, 错误=%v", id, name, err)
		return err
	}

	logger.Infof("更新分组成功: ID=%d, 名称=%s", id, name)

	return nil
}

// DeleteGroup 删除分组
func DeleteGroup(id int) error {
	// 检查分组是否存在
	var group Group
	if err := DB.First(&group, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			logger.Errorf("删除分组失败: 分组不存在, ID=%d", id)
			return errors.New("分组不存在")
		}
		logger.Errorf("查询分组失败: ID=%d, 错误=%v", id, err)
		return err
	}

	// 检查分组下是否有绑定的设备
	var deviceCount int64
	if err := DB.Model(&Instance{}).Where("group_id = ?", id).Count(&deviceCount).Error; err != nil {
		logger.Errorf("检查分组设备数量失败: ID=%d, 错误=%v", id, err)
		return err
	}

	if deviceCount > 0 {
		logger.Errorf("删除分组失败: 分组下还有 %d 台设备, ID=%d, 名称=%s", deviceCount, id, group.Name)
		return errors.New("无法删除分组，该分组下还有设备，请先移除所有设备后再删除")
	}

	// 执行删除操作
	if err := DB.Delete(&Group{}, id).Error; err != nil {
		logger.Errorf("删除分组失败: ID=%d, 错误=%v", id, err)
		return err
	}

	logger.Infof("删除分组成功: ID=%d, 名称=%s", id, group.Name)

	return nil
}
