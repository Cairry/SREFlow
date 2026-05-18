package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError 自定义API错误类型
type APIError struct {
	Code    int
	Message string
}

// Error 实现error接口
func (e *APIError) Error() string {
	return e.Message
}

func Service(ctx *gin.Context, fu func() (interface{}, error)) {
	data, err := fu()
	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			Error(ctx, apiErr.Code, apiErr.Message)
		} else {
			Error(ctx, http.StatusInternalServerError, err.Error())
		}
		return
	}
	Success(ctx, data, "success")
}

func BindJson(ctx *gin.Context, req interface{}) error {
	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		return fmt.Errorf("[Body] 请求参数错误, %s", err.Error())
	}

	return nil
}

func BindQuery(ctx *gin.Context, req interface{}) error {
	err := ctx.ShouldBindQuery(req)
	if err != nil {
		return fmt.Errorf("[Query] 请求参数错误, %s", err.Error())
	}

	return nil
}

var CodeInfo = map[int64]string{
	200: "OK",
	400: "请求失败",
	401: "Token鉴权失败",
	403: "权限不足",
}

func Response(c *gin.Context, httpStatus int, code int, data interface{}, msg string) {
	c.JSON(httpStatus, gin.H{
		"code": code,
		"data": data,
		"msg":  msg,
	})
}

func Success(ctx *gin.Context, data interface{}, msg string) {
	Response(ctx, http.StatusOK, 200, data, msg)
}

func Error(ctx *gin.Context, data interface{}, msg string) {
	code := 400
	Response(ctx, code, code, data, msg)
}

func TokenFail(ctx *gin.Context) {
	code := 401
	Response(ctx, code, code, nil, CodeInfo[int64(code)])
}

func PermissionFail(ctx *gin.Context) {
	code := 403
	Response(ctx, code, code, nil, CodeInfo[int64(code)])
}
