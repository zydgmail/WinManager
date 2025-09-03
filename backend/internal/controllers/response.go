package controllers

import (
	"net/http"
	"winmanager-backend/internal/logger"

	"github.com/gin-gonic/gin"
)

// 错误代码定义
const (
	ErrSuccess   = 0
	ErrParam     = 1001
	ErrBindJson  = 1002
	ErrDbReturn  = 1003
	ErrNotFound  = 1004
	ErrInternal  = 1005
)

// Response 统一响应结构
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// SuccessRes 成功响应
func SuccessRes(c *gin.Context, data interface{}) {
	response := Response{
		Code: ErrSuccess,
		Msg:  "success",
		Data: data,
	}

	logger.Infof("API响应成功: %s %s", c.Request.Method, c.Request.URL.Path)
	
	c.JSON(http.StatusOK, response)
}

// ErrorRes 错误响应
func ErrorRes(c *gin.Context, code int, msg string) {
	response := Response{
		Code: code,
		Msg:  msg,
	}
	
	logger.Errorf("API响应错误: %s %s, 错误码=%d, 错误信息=%s", c.Request.Method, c.Request.URL.Path, code, msg)
	
	c.JSON(http.StatusOK, response)
}

// NotFoundRes 404响应
func NotFoundRes(c *gin.Context, msg string) {
	if msg == "" {
		msg = "资源未找到"
	}
	ErrorRes(c, ErrNotFound, msg)
}

// BadRequestRes 400响应
func BadRequestRes(c *gin.Context, msg string) {
	if msg == "" {
		msg = "请求参数错误"
	}
	ErrorRes(c, ErrParam, msg)
}

// InternalErrorRes 500响应
func InternalErrorRes(c *gin.Context, msg string) {
	if msg == "" {
		msg = "内部服务器错误"
	}
	ErrorRes(c, ErrInternal, msg)
}
