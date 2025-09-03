package controllers

import (
	"sort"
	"strconv"
	"strings"
	"time"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// DeviceInfo 设备信息结构（用于注册）
type DeviceInfo struct {
	Uuid            string `json:"uuid"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	LAN             string `json:"lan"`
	WAN             string `json:"wan"`
	MAC             string `json:"mac"`
	CPU             string `json:"cpu"`
	Cores           int    `json:"cores"`
	RAM             uint64 `json:"ram"`
	Uptime          uint64 `json:"uptime"`
	Hostname        string `json:"hostname"`
	Username        string `json:"username"`
	Version         string `json:"version"`
	WatchdogVersion string `json:"watchdog_version"`
}

// HeartbeatRequest 心跳请求结构
type HeartbeatRequest struct {
	Wan    string `json:"wan"`    // 外网IP
	Uptime uint64 `json:"uptime"` // 系统运行时间
}

// Heartbeat 心跳接口
func Heartbeat(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("心跳参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	// 解析心跳数据
	var heartbeatData HeartbeatRequest
	if err := c.ShouldBindJSON(&heartbeatData); err != nil {
		logger.Errorf("心跳数据绑定失败: %v", err)
		// 兼容旧版本，如果解析失败就只更新状态和心跳时间
		heartbeatData = HeartbeatRequest{}
	}

	// 准备更新数据
	now := time.Now()
	updateData := map[string]interface{}{
		"status":            1,    // 设置为在线状态
		"last_heartbeat_at": &now, // 更新心跳时间
	}

	// 如果有WAN IP，更新WAN IP
	if heartbeatData.Wan != "" {
		updateData["wan"] = heartbeatData.Wan
	}

	// 如果有uptime，更新uptime
	if heartbeatData.Uptime > 0 {
		updateData["uptime"] = heartbeatData.Uptime
	}

	// 更新实例
	err = models.PatchInstance(id, updateData)
	if err != nil {
		logger.Errorf("更新心跳状态失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("心跳更新成功: ID=%d, WAN=%s, Uptime=%d", id, heartbeatData.Wan, heartbeatData.Uptime)

	SuccessRes(c, nil)
}

// Register 注册实例
func Register(c *gin.Context) {
	var info DeviceInfo
	if err := c.ShouldBindJSON(&info); err != nil {
		logger.Errorf("注册参数绑定失败: %v", err)
		ErrorRes(c, ErrBindJson, err.Error())
		return
	}

	newInstance := models.Instance{
		Uuid:            info.Uuid,
		OS:              info.OS,
		Arch:            info.Arch,
		Lan:             info.LAN,
		Wan:             info.WAN,
		Mac:             info.MAC,
		Cpu:             info.CPU,
		Cores:           info.Cores,
		Memory:          info.RAM,
		Uptime:          info.Uptime,
		Hostname:        info.Hostname,
		Username:        info.Username,
		Version:         info.Version,
		WatchdogVersion: info.WatchdogVersion,
		Status:          1, // 设置为在线状态
	}

	logger.Infof("注册设备: %+v", newInstance)

	id, err := models.CreateOrUpdateInstance(newInstance)
	if err != nil {
		logger.Errorf("注册设备失败: %v", err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("设备注册成功: ID=%d", id)

	SuccessRes(c, id)
}

// ListInstances 获取实例列表
func ListInstances(c *gin.Context) {
	// 检查是否使用新的查询参数
	if c.Query("page") != "" || c.Query("size") != "" || c.Query("search") != "" || c.Query("status") != "" || c.Query("group_id") != "" {
		// 使用新的分页和搜索功能
		var params models.InstanceListParams
		if err := c.ShouldBindQuery(&params); err != nil {
			logger.Errorf("参数绑定失败: %v", err)
			ErrorRes(c, ErrBindJson, err.Error())
			return
		}

		result, err := models.ListInstancesWithParams(params)
		if err != nil {
			logger.Errorf("获取实例列表失败: %v", err)
			ErrorRes(c, ErrDbReturn, err.Error())
			return
		}

		logger.Infof("获取实例列表成功: 总数=%d, 当前页=%d", result.Total, result.Page)

		SuccessRes(c, result)
		return
	}

	// 兼容原有的ids查询方式
	_ids := strings.Split(c.Query("ids"), ",")

	ids := []int{}
	for _, id := range _ids {
		tmp, err := strconv.Atoi(id)
		if err == nil {
			ids = append(ids, tmp)
		}
	}

	instances, err := models.ListInstances(ids)
	if err != nil {
		logger.Errorf("获取实例列表失败: %v", err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	// 按主机名排序
	sort.SliceStable(instances, func(i, j int) bool {
		i_nums := strings.Split(instances[i].Hostname, "-")
		j_nums := strings.Split(instances[j].Hostname, "-")
		if len(i_nums) != 4 || len(j_nums) != 4 {
			return instances[i].Hostname < instances[j].Hostname
		}
		if i_nums[0] == j_nums[0] && i_nums[1] == j_nums[1] {
			i3, _ := strconv.Atoi(i_nums[2])
			j3, _ := strconv.Atoi(j_nums[2])
			if i3 == j3 {
				i4, _ := strconv.Atoi(i_nums[3])
				j4, _ := strconv.Atoi(j_nums[3])
				return i4 < j4
			}
			return i3 < j3
		}
		return instances[i].Hostname < instances[j].Hostname
	})

	logger.Infof("获取实例列表成功: 数量=%d", len(instances))

	SuccessRes(c, instances)
}

// GetInstance 获取单个实例
func GetInstance(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("获取实例参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	instance, err := models.GetInstance(id)
	if err != nil {
		logger.Errorf("获取实例失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("获取实例成功: ID=%d", id)

	SuccessRes(c, instance)
}

// PatchInstance 更新实例
func PatchInstance(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("更新实例参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	var item map[string]interface{}
	if err := c.ShouldBindJSON(&item); err != nil {
		logger.Errorf("更新实例参数绑定失败: %v", err)
		ErrorRes(c, ErrBindJson, err.Error())
		return
	}

	err = models.PatchInstance(id, item)
	if err != nil {
		logger.Errorf("更新实例失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("更新实例成功: ID=%d", id)

	SuccessRes(c, nil)
}

// DeleteInstance 删除实例
func DeleteInstance(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("删除实例参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	err = models.DeleteInstance(id)
	if err != nil {
		logger.Errorf("删除实例失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("删除实例成功: ID=%d", id)

	SuccessRes(c, nil)
}

// MoveGroupRequest 移动分组请求结构
type MoveGroupRequest struct {
	Ids     []int `json:"ids"`
	GroupId int   `json:"group_id"`
}

// MoveGroupInstance 移动实例到分组
func MoveGroupInstance(c *gin.Context) {
	var req MoveGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Errorf("移动分组参数绑定失败: %v", err)
		ErrorRes(c, ErrBindJson, err.Error())
		return
	}

	var group_id *int
	if req.GroupId == 0 {
		group_id = nil
	} else {
		group_id = &req.GroupId
	}

	err := models.DB.Model(models.Instance{}).Where("id IN ?", req.Ids).Updates(
		map[string]interface{}{
			"group_id": group_id,
		},
	).Error

	if err != nil {
		logger.Errorf("移动实例到分组失败: IDs=%v, GroupID=%v, 错误=%v", req.Ids, group_id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("移动实例到分组成功: IDs=%v, GroupID=%v", req.Ids, group_id)

	SuccessRes(c, nil)
}
