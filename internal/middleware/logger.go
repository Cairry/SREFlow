package middleware

import (
	"bytes"
	"context"
	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logc"
	"github.com/zeromicro/go-zero/core/logx"
	"io"
	"time"
)

// Logger returns a gin.HandlerFunc that logs requests using zap
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		/*
			打印请求的body和query params
		*/

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			logc.Errorf(context.Background(), err.Error())
			c.Abort()
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		ctxWithReq := logx.ContextWithFields(context.Background(),
			logx.Field("method", c.Request.Method),
			logx.Field("path", c.Request.URL.Path),
		)
		logc.Infof(ctxWithReq, "Body: %s", string(bodyBytes))
		logc.Infof(ctxWithReq, "Query Params: %v", c.Request.URL.Query())

		/*
			封装请求日志
		*/

		// 获取时间
		start := time.Now()
		c.Next()
		end := time.Now()

		// 计算请求耗时
		latency := end.Sub(start)

		// 获取请求的相关信息
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()
		message := c.Errors.ByType(gin.ErrorTypePrivate).String()

		ctx := logx.ContextWithFields(context.Background(),
			logx.Field("method", method),
			logx.Field("path", path),
			logx.Field("status", status),
			logx.Field("clientIP", clientIP),
			logx.Field("latency", latency),
		)
		logc.Info(ctx, message)
	}
}
