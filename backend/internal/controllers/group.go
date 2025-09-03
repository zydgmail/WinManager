package controllers

import (
	"strconv"
	"strings"
	"winmanager-backend/internal/logger"
	"winmanager-backend/internal/models"

	"github.com/gin-gonic/gin"
)

// CreateGroupRequest 创建分组请求结构
type CreateGroupRequest struct {
	Name string `json:"name"`
}

// ListGroups 获取分组列表（支持分页）
func ListGroups(c *gin.Context) {
	// 解析查询参数
	var params models.GroupListParams
	if err := c.ShouldBindQuery(&params); err != nil {
		logger.Errorf("分组列表参数绑定失败: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	// 获取分组列表
	result, err := models.GetGroupList(&params)
	if err != nil {
		logger.Errorf("获取分组列表失败: %v", err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("获取分组列表成功: 页码=%d, 数量=%d, 总数=%d", result.Page, len(result.Groups), result.Total)

	SuccessRes(c, result)
}

// ListGroupsSimple 获取简单分组列表（兼容旧接口）
func ListGroupsSimple(c *gin.Context) {
	_ids := strings.Split(c.Query("ids"), ",")

	ids := []int{}
	for _, id := range _ids {
		tmp, err := strconv.Atoi(id)
		if err == nil {
			ids = append(ids, tmp)
		}
	}

	groups, err := models.ListGroups(ids)
	if err != nil {
		logger.Errorf("获取分组列表失败: %v", err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("获取分组列表成功: 数量=%d", len(groups))

	SuccessRes(c, groups)
}

// CreateGroup 创建分组
func CreateGroup(c *gin.Context) {
	var item CreateGroupRequest
	if err := c.ShouldBindJSON(&item); err != nil {
		logger.Errorf("创建分组参数绑定失败: %v", err)
		ErrorRes(c, ErrBindJson, err.Error())
		return
	}

	if item.Name == "" {
		logger.Errorf("创建分组名称为空")
		BadRequestRes(c, "分组名称不能为空")
		return
	}

	group, err := models.CreateGroup(item.Name)
	if err != nil {
		logger.Errorf("创建分组失败: 名称=%s, 错误=%v", item.Name, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("创建分组成功: ID=%d, 名称=%s", group.ID, item.Name)

	SuccessRes(c, group)
}

// PatchGroup 更新分组
func PatchGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("更新分组参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	var item CreateGroupRequest
	if err := c.ShouldBindJSON(&item); err != nil {
		logger.Errorf("更新分组参数绑定失败: %v", err)
		ErrorRes(c, ErrBindJson, err.Error())
		return
	}

	if item.Name == "" {
		logger.Errorf("更新分组名称为空: ID=%d", id)
		BadRequestRes(c, "分组名称不能为空")
		return
	}

	err = models.PatchGroup(id, item.Name)
	if err != nil {
		logger.Errorf("更新分组失败: ID=%d, 名称=%s, 错误=%v", id, item.Name, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("更新分组成功: ID=%d, 名称=%s", id, item.Name)

	SuccessRes(c, item)
}

// DeleteGroup 删除分组
func DeleteGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("删除分组参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	err = models.DeleteGroup(id)
	if err != nil {
		logger.Errorf("删除分组失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("删除分组成功: ID=%d", id)

	SuccessRes(c, nil)
}

// GetGroup 获取单个分组
func GetGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Errorf("获取分组参数错误: %v", err)
		BadRequestRes(c, "参数错误")
		return
	}

	group, err := models.GetGroup(id)
	if err != nil {
		logger.Errorf("获取分组失败: ID=%d, 错误=%v", id, err)
		ErrorRes(c, ErrDbReturn, err.Error())
		return
	}

	logger.Infof("获取分组成功: ID=%d, 名称=%s", id, group.Name)

	SuccessRes(c, group)
}
