package agent

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码常量
const (
	ErrDbReturn  = "DB_RETURN_ERROR"
	ErrBindJson  = "BIND_JSON_ERROR"
)

// SuccessRes 成功响应
func SuccessRes(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    data,
	})
}

// ErrorRes 错误响应
func ErrorRes(c *gin.Context, errCode string, message string) {
	c.JSON(http.StatusOK, gin.H{
		"code":    -1,
		"message": message,
		"error":   errCode,
	})
}

// BadRequestRes 请求参数错误响应
func BadRequestRes(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"code":    -1,
		"message": message,
	})
}

// InternalErrorRes 内部错误响应
func InternalErrorRes(c *gin.Context, message string) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    -1,
		"message": message,
	})
}
