package nodes

import (
	"RCAgent/internal/models"
	"context"
)

// StartExecutor 开始节点执行器
type StartExecutor struct {
	DefaultNodeExecutor
}

// NewStartExecutor 创建Start执行器
func NewStartExecutor() *StartExecutor {
	return &StartExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Start",
			Category:    "logic",
			Description: "Workflow starting point with initial data",
		},
	}
}

// Execute 执行Start节点
func (e *StartExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	// 获取文本数据，如果有
	text, hasText := nodeCtx.Config["text"].(string)
	if !hasText {
		text = ""
	}

	// 返回文本数据作为输出
	return newResult(map[string]any{
		"text": text,
	}), nil
}
