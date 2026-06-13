package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/zeromicro/go-zero/core/logx"
)

// HeaderRequestID 请求/响应头中携带 trace id 的字段名
const HeaderRequestID = "X-Request-Id"

// FieldKey 结构化日志中 trace id 的字段名
const FieldKey = "traceId"

// ctxKey 是 trace id 在 context 中的私有键，避免与其他包冲突
type ctxKey struct{}

// New 生成一个符合 W3C Trace Context 标准的 trace id：
// 16 字节随机数编码为 32 位小写十六进制字符串，与 OpenTelemetry / Jaeger 兼容。
func New() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// WithTraceID 将 trace id 写入 context：
//  1. 通过 context.Value 保存，供业务代码读取（FromContext）；
//  2. 通过 logx.ContextWithFields 注入结构化日志字段，
//     使后续 logc.Infof(ctx, ...) 等调用自动携带 traceId。
//
// 当 traceID 为空时会自动生成一个新的 trace id。
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		traceID = New()
	}
	ctx = context.WithValue(ctx, ctxKey{}, traceID)
	return logx.ContextWithFields(ctx, logx.Field(FieldKey, traceID))
}

// FromContext 从 context 中读取 trace id，不存在时返回空串
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}
