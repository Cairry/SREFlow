package nodes

import (
	"context"
	"fmt"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// ReplyExecutor 直接回复执行器
type ReplyExecutor struct {
	DefaultNodeExecutor
}

// NewReplyExecutor 创建直接回复执行器
func NewReplyExecutor() *ReplyExecutor {
	return &ReplyExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Reply",
			Category:    "output",
			Description: "Reply workflow result directly to caller",
		},
	}
}

// ExecuteReply 执行直接回复
func ExecuteReply(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	replyExecutor := NewReplyExecutor()
	return replyExecutor.Execute(ctx, nc)
}

func (e *ReplyExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	contentTemplate := getStr(nodeCtx.Config, "contentTemplate", "")

	logc.Infof(ctx, "[Reply] Processing reply content")
	logc.Infof(ctx, "[Reply] Input variables: %d", len(nodeCtx.Input))

	// 构建回复内容：优先使用 contentTemplate，否则兜底拼接所有 input
	var reply string
	if contentTemplate != "" {
		reply = contentTemplate
		logc.Infof(ctx, "[Reply] Using content template (length: %d)", len(reply))
	} else {
		for key, value := range nodeCtx.Input {
			if reply != "" {
				reply += "\n"
			}
			reply += fmt.Sprintf("[%s]: %v", key, value)
		}
	}

	result := map[string]any{
		"reply": reply,
	}

	return &models.NodeResult{
		Outputs: result,
	}, nil
}
