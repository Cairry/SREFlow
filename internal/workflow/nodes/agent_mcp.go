package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/zeromicro/go-zero/core/logc"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// --- MCP Tool 包装器 ---

// mcpToolWrapper 将 MCP 工具包装为 eino InvokableTool
type mcpToolWrapper struct {
	toolInfo   *schema.ToolInfo
	serverID   string
	serverName string               // 缓存，避免 callMCPTool 查 DB
	toolName   string               // 实际 MCP 工具名（去掉 server 前缀）
	serverURL  string               // 缓存 server 连接信息
	serverType models.MCPServerType // 缓存
	headers    map[string]string    // 缓存
	apiKey     string               // 缓存
	timeout    int                  // 缓存
	executor   *AgentExecutor
}

// Info 返回工具信息（eino BaseTool 接口）
func (t *mcpToolWrapper) Info(_ context.Context) (*schema.ToolInfo, error) {
	return t.toolInfo, nil
}

// InvokableRun 执行 MCP 工具调用（eino InvokableTool 接口）
func (t *mcpToolWrapper) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var args map[string]any
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &args); err != nil {
			args = map[string]any{}
		}
	}

	// 使用缓存的 server 元数据调用，无需查询 DB
	result, err := t.executor.callMCPTool(ctx, t.serverID, t.serverName, t.toolName,
		t.serverURL, t.serverType, t.headers, t.apiKey, t.timeout, args)
	if err != nil {
		return "", err
	}

	return toolResultToString(result), nil
}

// --- Skill Tool 包装器 ---

// skillToolWrapper 将 Skill 加载功能包装为 eino InvokableTool
type skillToolWrapper struct {
	toolInfo  *schema.ToolInfo
	skillID   string
	skillName string
	skillRepo repos.SkillRepoInter
}

// Info 返回工具信息（eino BaseTool 接口）
func (t *skillToolWrapper) Info(_ context.Context) (*schema.ToolInfo, error) {
	return t.toolInfo, nil
}

// InvokableRun 执行 Skill 加载（eino InvokableTool 接口）
func (t *skillToolWrapper) InvokableRun(ctx context.Context, _ string, _ ...tool.Option) (string, error) {
	skill, err := t.skillRepo.GetByID(t.skillID)
	if err != nil {
		return "", fmt.Errorf("failed to load skill %s: %w", t.skillName, err)
	}
	if !skill.Enabled {
		return "", fmt.Errorf("skill %s is disabled", t.skillName)
	}

	logc.Infof(ctx, "[Agent] Skill loaded: %s", t.skillName)
	return skill.Content, nil
}

// toolResultToString 将 MCP 工具结果转为字符串
func toolResultToString(result *mcp.CallToolResult) string {
	if result.IsError {
		var parts []string
		for _, c := range result.Content {
			if tc, ok := c.(mcp.TextContent); ok {
				parts = append(parts, tc.Text)
			}
		}
		return "Error: " + strings.Join(parts, "\n")
	}

	var parts []string
	for _, c := range result.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			parts = append(parts, v.Text)
		default:
			parts = append(parts, fmt.Sprintf("[%T content]", c))
		}
	}
	return strings.Join(parts, "\n")
}
