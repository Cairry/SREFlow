package nodes

import (
	"context"
	"fmt"
	"strings"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// ConditionExecutor 条件判断执行器
type ConditionExecutor struct {
	DefaultNodeExecutor
}

// NewConditionExecutor 创建条件执行器
func NewConditionExecutor() *ConditionExecutor {
	return &ConditionExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Condition",
			Category:    "logic",
			Description: "Evaluate conditions and control workflow flow",
		},
	}
}

// Execute 执行条件判断
func ExecuteCondition(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	condExecutor := NewConditionExecutor()
	return condExecutor.Execute(ctx, nc)
}

func (e *ConditionExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	inputStr := getStr(nodeCtx.Config, "condition", "{{input}}")
	operator := getStr(nodeCtx.Config, "operator", "contains")
	value := getStr(nodeCtx.Config, "value", "")

	// 计算条件结果
	result := evaluateCondition(inputStr, operator, value)
	// 确定走哪个分支：true 或 false
	branch := "false"
	if result {
		branch = "true"
	}

	nodeResult := &models.NodeResult{
		Outputs: map[string]any{
			"branch": branch,
		},
	}

	// 根据 HandleEdges 设置 Next，只路由到匹配分支的下游节点
	if nodeCtx.HandleEdges != nil {
		if targetNodeID, ok := nodeCtx.HandleEdges[branch]; ok {
			nodeResult.Next = []string{targetNodeID}
			logc.Infof(ctx, "[Condition] Routing to branch '%s' -> node %s", branch, targetNodeID)
		} else {
			logc.Infof(ctx, "[Condition] Branch '%s' has no connected downstream node", branch)
		}
	}

	return nodeResult, nil
}

func evaluateCondition(input, operator, value string) bool {
	switch operator {
	case "contains":
		return strings.Contains(strings.ToLower(input), strings.ToLower(value))
	case "equals":
		return input == value
	case "notEquals":
		return input != value
	case "startsWith":
		return strings.HasPrefix(strings.ToLower(input), strings.ToLower(value))
	case "endsWith":
		return strings.HasSuffix(strings.ToLower(input), strings.ToLower(value))
	case "greaterThan":
		return parseFloat(input) > parseFloat(value)
	case "lessThan":
		return parseFloat(input) < parseFloat(value)
	case "greaterOrEqual":
		return parseFloat(input) >= parseFloat(value)
	case "lessOrEqual":
		return parseFloat(input) <= parseFloat(value)
	case "isEmpty":
		return strings.TrimSpace(input) == ""
	case "isNotEmpty":
		return strings.TrimSpace(input) != ""
	case "matches":
		// 使用简单的通配符匹配
		return matchWildcard(input, value)
	default:
		return false
	}
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func matchWildcard(s, pattern string) bool {
	// 简单的通配符匹配：* 匹配任意字符
	parts := strings.Split(pattern, "*")
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !strings.Contains(s, part) {
			return false
		}
	}
	return true
}
