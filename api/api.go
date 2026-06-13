package api

import (
	"fmt"
	"net/http"

	"SREFlow/internal/i18n"

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
	lang := i18n.GetLanguageFromContext(ctx)

	if err != nil {
		if apiErr, ok := err.(*APIError); ok {
			Error(ctx, apiErr.Code, apiErr.Message)
		} else {
			Error(ctx, http.StatusInternalServerError, err.Error())
		}
		return
	}
	Success(ctx, data, i18n.TWithLang(lang, "common.success"))
}

func BindJson(ctx *gin.Context, req interface{}) error {
	lang := i18n.GetLanguageFromContext(ctx)

	err := ctx.ShouldBindJSON(&req)
	if err != nil {
		return fmt.Errorf(i18n.TWithLang(lang, "error.invalidParams", map[string]interface{}{"error": err.Error()}))
	}

	return nil
}

func BindQuery(ctx *gin.Context, req interface{}) error {
	lang := i18n.GetLanguageFromContext(ctx)

	err := ctx.ShouldBindQuery(req)
	if err != nil {
		return fmt.Errorf(i18n.TWithLang(lang, "error.invalidParams", map[string]interface{}{"error": err.Error()}))
	}

	return nil
}

var CodeInfo = map[int64]string{
	200: "common.success",
	400: "error.badRequest",
	401: "auth.tokenInvalid",
	403: "auth.permissionDenied",
}

func Response(c *gin.Context, httpStatus int, code int, data interface{}, msg string) {
	lang := i18n.GetLanguageFromContext(c)

	// Translate the message if it's a message ID
	translatedMsg := i18n.TWithLang(lang, msg)
	if translatedMsg == msg {
		// If translation not found, use original message
		translatedMsg = msg
	}

	c.JSON(httpStatus, gin.H{
		"code": code,
		"data": data,
		"msg":  translatedMsg,
		"lang": lang,
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
