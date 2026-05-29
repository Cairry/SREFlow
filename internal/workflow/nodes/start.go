package nodes

import (
	"SREFlow/internal/models"
	"context"
	"encoding/json"
	"fmt"
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

	outputs := map[string]any{}

	// 如果text是JSON字符串，解析为map并展开到output中
	// 纯字符串时通过 start.text 引用，JSON时通过 start.xxx 引用各字段
	if text != "" {
		var jsonMap map[string]any
		if err := json.Unmarshal([]byte(text), &jsonMap); err == nil {
			// JSON模式：将JSON对象的各字段展开到output中，如 start.key1, start.key2
			for k, v := range jsonMap {
				outputs[k] = v
			}
		} else {
			// 纯字符串模式：通过 start.text 引用完整内容
			outputs["text"] = text
		}
	}

	fmt.Println("[Start] outputs:", outputs)

	return newResult(outputs), nil
}
