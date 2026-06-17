package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/zeromicro/go-zero/core/logc"

	"SREFlow/internal/models"

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

// --- Skill Tool（统一单工具，参考 eino adk/middlewares/skill 设计）---

// skillInputArguments skill 工具入参
type skillInputArguments struct {
	Skill string `json:"skill"`
}

// skillTool 统一 Skill 加载工具：单一工具接受 skill 名称参数，
// 工具描述动态列出所有可用 skill（渐进式披露）。
type skillTool struct {
	toolInfo *schema.ToolInfo
	skills   map[string]*models.Skill // name -> full skill model
}

// Info 返回工具信息，描述动态包含可用 skill 列表
func (t *skillTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return t.toolInfo, nil
}

// InvokableRun 按 skill 名称加载并返回完整内容
func (t *skillTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	args := &skillInputArguments{}
	if argumentsInJSON != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), args); err != nil {
			return "", fmt.Errorf("failed to unmarshal skill arguments: %w", err)
		}
	}
	if args.Skill == "" {
		return "", fmt.Errorf("skill name is required")
	}

	sk, ok := t.skills[args.Skill]
	if !ok {
		available := make([]string, 0, len(t.skills))
		for name := range t.skills {
			available = append(available, name)
		}
		return "", fmt.Errorf("skill not found: %s, available skills: %v", args.Skill, available)
	}
	if !sk.Enabled {
		return "", fmt.Errorf("skill %s is disabled", args.Skill)
	}

	logc.Infof(ctx, "[Agent] Skill loaded: %s", args.Skill)
	return fmt.Sprintf("Launching skill: %s\n\n%s", sk.Name, sk.Content), nil
}

// buildSkillToolDescription 动态生成 skill 工具描述，列出所有可用 skill
func buildSkillToolDescription(skills []skillInfo) string {
	var b strings.Builder
	b.WriteString(skillToolDescriptionBase)
	b.WriteString("\n<available_skills>\n")
	for _, sk := range skills {
		b.WriteString("<skill>\n")
		b.WriteString(fmt.Sprintf("<name>\n%s\n</name>\n", sk.Name))
		b.WriteString(fmt.Sprintf("<description>\n%s\n</description>\n", sk.Description))
		b.WriteString("</skill>\n")
	}
	b.WriteString("</available_skills>\n")
	return b.String()
}

const skillToolDescriptionBase = `Execute a skill within the main conversation

<skills_instructions>
When users ask you to perform tasks, check if any of the available skills below can help complete the task more effectively. Skills provide specialized capabilities and domain knowledge.

How to invoke:
- Use this tool with the skill name as the "skill" argument
- Examples:
  - skill: "pdf" - invoke the pdf skill
  - skill: "xlsx" - invoke the xlsx skill

Important:
- When a skill is relevant, you must invoke this tool IMMEDIATELY as your first action
- NEVER just announce or mention a skill in your text response without actually calling this tool
- This is a BLOCKING REQUIREMENT: invoke the relevant Skill tool BEFORE generating any other response about the task
- Only use skills listed in <available_skills> below
- Do not invoke a skill that is already running
</skills_instructions>
`

const skillSystemPrompt = `
# Skills System

**How to Use Skills (Progressive Disclosure):**

Skills follow a **progressive disclosure** pattern - you see their name and description in the skill tool, but only read full instructions when needed:

1. **Recognize when a skill applies**: Check if the user's task matches a skill's description
2. **Read the skill's full instructions**: Use the 'skill' tool to load skill by name
3. **Follow the skill's instructions**: Tool result contains step-by-step workflows, best practices, and examples
4. **Execute skill workflows**: Follow the loaded instructions precisely

**When to Use Skills:**
- User's request matches a skill's domain (e.g., "research X" -> web-research skill)
- You need specialized knowledge or structured workflows
- A skill provides proven patterns for complex tasks

**Example Workflow:**

User: "Can you research the latest developments in quantum computing?"

1. Check available skills in skill tool -> See "web-research" skill
2. Call 'skill' tool with name "web-research" to read full instructions
3. Follow the skill's research workflow (search -> organize -> synthesize)

Remember: Skills make you more capable and consistent. When in doubt, check if a skill exists for the task!
`

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
