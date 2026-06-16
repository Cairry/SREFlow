package nodes

import (
	"context"
	"fmt"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// OutputExecutor 输出执行器
type OutputExecutor struct {
	DefaultNodeExecutor
}

// NewOutputExecutor 创建输出执行器
func NewOutputExecutor() *OutputExecutor {
	return &OutputExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Output",
			Category:    "output",
			Description: "Output workflow results",
		},
	}
}

// ExecuteOutput 执行输出
func ExecuteOutput(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	outputExecutor := NewOutputExecutor()
	return outputExecutor.Execute(ctx, nc)
}

func (e *OutputExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	customFilename := getStr(nodeCtx.Config, "customFilename", "output")
	contentTemplate := getStr(nodeCtx.Config, "contentTemplate", "")

	logc.Infof(ctx, "[Output] Processing markdown output, filename: %s", customFilename)
	logc.Infof(ctx, "[Output] Input variables: %d", len(nodeCtx.Input))

	// 构建输出内容：优先使用 contentTemplate，否则兜底拼接所有 input
	var output string
	if contentTemplate != "" {
		output = contentTemplate
		logc.Infof(ctx, "[Output] Using content template (length: %d)", len(output))
	} else {
		for key, value := range nodeCtx.Input {
			if output != "" {
				output += "\n"
			}
			output += fmt.Sprintf("[%s]: %v", key, value)
		}
	}

	// 确保文件名以 .md 结尾
	filename := customFilename
	if !endsWith(filename, ".md") {
		filename += ".md"
	}

	result := map[string]any{
		"output":   output,
		"filename": filename,
	}

	return &models.NodeResult{
		Outputs: result,
	}, nil
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
