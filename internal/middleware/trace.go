package middleware

import (
	"SREFlow/pkg/trace"

	"github.com/gin-gonic/gin"
)

// GinTraceKey 是 trace id 在 gin.Context 中保存的键
const GinTraceKey = "traceID"

// TraceID 为每个请求生成或透传 trace id：
//   - 优先复用请求头 X-Request-Id（便于跨服务串联）；
//   - 缺省时自动生成；
//   - 写入请求 context（业务/逻辑层及日志可读取）与 gin.Context；
//   - 通过响应头 X-Request-Id 回传调用方。
func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(trace.HeaderRequestID)
		if traceID == "" {
			traceID = trace.New()
		}

		ctx := trace.WithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)
		c.Set(GinTraceKey, traceID)
		c.Header(trace.HeaderRequestID, traceID)

		c.Next()
	}
}
