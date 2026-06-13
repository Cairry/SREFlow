package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/zeromicro/go-zero/core/logc"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/internal/workflow/mcppool"

	"github.com/eino-contrib/jsonschema"
)

// AgentExecutor Agent节点执行器 - 基于 eino ReAct 模式
type AgentExecutor struct {
	DefaultNodeExecutor
	mcpRepo   repos.MCPServerRepoInter
	skillRepo repos.SkillRepoInter
}

// NewAgentExecutor 创建Agent执行器
func NewAgentExecutor(mcpRepo repos.MCPServerRepoInter, skillRepo repos.SkillRepoInter) *AgentExecutor {
	return &AgentExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Agent",
			Category:    "ai",
			Description: "ReAct Agent with MCP tool calling and Skill support (powered by eino)",
		},
		mcpRepo:   mcpRepo,
		skillRepo: skillRepo,
	}
}

// --- MCP Tool 包装器 ---

// mcpToolWrapper 将 MCP 工具包装为 eino InvokableTool
type mcpToolWrapper struct {
	toolInfo *schema.ToolInfo
	serverID string
	toolName string // 实际 MCP 工具名（去掉 server 前缀）
	executor *AgentExecutor
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

	result, err := t.executor.callMCPTool(ctx, t.serverID, t.toolName, args)
	if err != nil {
		return "", err
	}

	resultStr := toolResultToString(result)
	if len(resultStr) > 8000 {
		resultStr = resultStr[:8000] + "...(truncated)"
	}
	return resultStr, nil
}

// toolCallRecord 工具调用记录
type toolCallRecord struct {
	ToolName  string `json:"toolName"`
	ServerID  string `json:"serverId"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
}

// --- 辅助类型 ---

// toolHint 工具过滤提示
type toolHint struct {
	ServerName string // MCP Server 名称，空表示不限
	ToolName   string // 工具名称，空表示该 server 下所有工具
}

// --- Execute ---

func (e *AgentExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	// 读取配置
	baseURL := getStr(nodeCtx.Config, "baseURL", "https://api.openai.com/v1")
	apiKey := getStr(nodeCtx.Config, "apiKey", "")
	modelID := getStr(nodeCtx.Config, "model", "gpt-4o")
	maxTokens := getInt(nodeCtx.Config, "maxTokens", 4096)
	timeoutMs := getInt(nodeCtx.Config, "timeout", 300000) // 5分钟超时
	maxIterations := getInt(nodeCtx.Config, "maxIterations", 10)
	userPrompt := getStr(nodeCtx.Config, "prompt", "")
	skillIDsStr := getSkillIDsStr(nodeCtx.Config)
	selectedTools := getSelectedTools(nodeCtx.Config)

	if apiKey == "" && baseURL == "https://api.openai.com/v1" {
		return errorResult("apiKey is required when using OpenAI API"), nil
	}

	// 解析用户输入（支持模板变量）
	resolvedUserPrompt := resolveTemplate(userPrompt, nodeCtx.Input)
	if resolvedUserPrompt == "" {
		resolvedUserPrompt = collectInputString(nodeCtx.Input)
	}

	// 加载选中的 Skills（提取工具提示 + 构建内容查找表）
	skillContents, toolHints, err := e.loadSkills(ctx, skillIDsStr)
	if err != nil {
		logc.Errorf(ctx, "[Agent] Failed to load skills: %v", err)
	}

	// 加载 MCP 工具（当选择了 Skill 或 MCP 工具时）
	var einoTools []tool.BaseTool
	if len(skillContents) > 0 || len(selectedTools) > 0 {
		var loadErr error
		einoTools, loadErr = e.loadMCPTools(ctx, nodeCtx.WorkspaceID, toolHints, selectedTools)
		if loadErr != nil {
			logc.Errorf(ctx, "[Agent] Failed to load MCP tools: %v", loadErr)
		}
	}

	logc.Infof(ctx, "[Agent] Model: %s, MaxIterations: %d, Tools: %d, Skills: %d, SelectedTools: %d",
		modelID, maxIterations, len(einoTools), len(skillContents), len(selectedTools))

	// 创建 eino OpenAI ChatModel
	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:    apiKey,
		Model:     modelID,
		BaseURL:   baseURL,
		Timeout:   time.Duration(timeoutMs) * time.Millisecond,
		MaxTokens: &maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}

	// 工具调用记录 & 中间件
	var toolCallRecords []toolCallRecord
	var mu sync.Mutex

	toolMiddleware := compose.ToolMiddleware{
		Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
			return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
				output, callErr := next(ctx, input)
				mu.Lock()
				record := toolCallRecord{
					ToolName:  input.Name,
					Arguments: input.Arguments,
				}
				if callErr != nil {
					record.Result = callErr.Error()
				} else if output != nil {
					record.Result = output.Result
				}
				toolCallRecords = append(toolCallRecords, record)
				mu.Unlock()
				return output, callErr
			}
		},
	}

	// 构建 ToolsNodeConfig
	toolsConfig := compose.ToolsNodeConfig{
		Tools:               einoTools,
		ToolCallMiddlewares: []compose.ToolMiddleware{toolMiddleware},
	}

	fmt.Println("[Agent] Tools:", einoTools)

	// 构建 Skill system prompt：将 Skill 内容注入到 system message 中
	var skillSystemPrompt string
	if len(skillContents) > 0 {
		var skillParts []string
		for name, content := range skillContents {
			skillParts = append(skillParts, fmt.Sprintf("<skill name=\"%s\">\n%s\n</skill>", name, content))
		}
		skillSystemPrompt = strings.Join(skillParts, "\n\n")
	}

	// 创建 eino ReAct Agent，通过 MessageModifier 在每轮模型调用前注入 skill system prompt
	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      toolsConfig,
		MaxStep:          maxIterations * 2, // eino MaxStep = 迭代数 * 2（每轮 ChatModel + Tools）
		MessageModifier: func(ctx context.Context, msgs []*schema.Message) []*schema.Message {
			if skillSystemPrompt == "" {
				return msgs
			}
			return append([]*schema.Message{schema.SystemMessage(skillSystemPrompt)}, msgs...)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	// 构建初始消息：Skill 内容已通过 MessageModifier 注入到 system prompt，此处只放用户 prompt
	messages := []*schema.Message{schema.UserMessage(resolvedUserPrompt)}

	// 执行 Agent
	start := time.Now()
	msg, err := reactAgent.Generate(ctx, messages)
	latency := time.Since(start).Seconds()

	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	logc.Infof(ctx, "[Agent] Final response (%.2fs): %s", latency, truncateLog(msg.Content, 300))

	result := map[string]any{
		"output":     msg.Content,
		"iterations": 0,
		"latency":    latency,
	}

	if len(toolCallRecords) > 0 {
		// 迭代次数 ≈ 工具调用次数 + 1（最后一次 ChatModel 回复）
		result["iterations"] = len(toolCallRecords) + 1
		result["toolCalls"] = toolCallRecords
	} else {
		result["iterations"] = 1
	}

	return newResult(result), nil
}

// --- Config 提取函数 ---

// getSelectedTools 从 config 中提取 selectedTools，格式: ["serverId::toolName", ...]
func getSelectedTools(cfg map[string]any) []string {
	v, ok := cfg["selectedTools"]
	if !ok || v == nil {
		return nil
	}
	switch val := v.(type) {
	case []any:
		var tools []string
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				tools = append(tools, s)
			}
		}
		return tools
	case []string:
		return val
	case string:
		if val == "" {
			return nil
		}
		return strings.Split(val, ",")
	default:
		return nil
	}
}

// getSkillIDsStr 从 config 中提取 skillIds，支持 []any / []string / string 类型
func getSkillIDsStr(cfg map[string]any) string {
	v, ok := cfg["skillIds"]
	if !ok || v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []any:
		if len(val) == 0 {
			return ""
		}
		var ids []string
		for _, item := range val {
			if s, ok := item.(string); ok && s != "" {
				ids = append(ids, s)
			}
		}
		if len(ids) == 0 {
			return ""
		}
		b, _ := json.Marshal(ids)
		return string(b)
	case []string:
		if len(val) == 0 {
			return ""
		}
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return ""
	}
}

// --- Skill 加载 ---

// loadSkills 加载选中的 Skill，返回 name->content 映射，同时提取工具提示
func (e *AgentExecutor) loadSkills(ctx context.Context, skillIDsStr string) (map[string]string, []toolHint, error) {
	if skillIDsStr == "" {
		return nil, nil, nil
	}

	var ids []string
	if err := json.Unmarshal([]byte(skillIDsStr), &ids); err != nil {
		ids = strings.Split(skillIDsStr, ",")
	}

	if len(ids) == 0 {
		return nil, nil, nil
	}

	skills, err := e.skillRepo.GetByIDs(ids)
	if err != nil {
		return nil, nil, err
	}

	contents := make(map[string]string)
	var hints []toolHint
	for _, sk := range skills {
		if !sk.Enabled || sk.Content == "" {
			continue
		}
		contents[sk.Name] = sk.Content
		h := extractToolHints(ctx, sk.Content)
		hints = append(hints, h...)
	}
	return contents, hints, nil
}

// extractToolHints 从 Skill 内容中提取工具调用提示
// 支持格式:
//
//	@server:server_name    - 指定 MCP Server（该 server 下所有工具）
//	@tool:server.tool      - 指定 server 下的工具名
func extractToolHints(ctx context.Context, content string) []toolHint {
	var hints []toolHint
	seen := make(map[string]bool)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 解析 @tool:xxx
		if strings.Contains(line, "@tool:") {
			parts := strings.Split(line, "@tool:")
			for i := 1; i < len(parts); i++ {
				token := strings.Fields(parts[i])
				if len(token) == 0 {
					continue
				}
				name := strings.TrimRight(token[0], ",;.")
				if name == "" {
					continue
				}
				key := "tool:" + name
				if seen[key] {
					continue
				}
				seen[key] = true

				if dotIdx := strings.Index(name, "."); dotIdx > 0 {
					hints = append(hints, toolHint{
						ServerName: name[:dotIdx],
						ToolName:   name[dotIdx+1:],
					})
				} else {
					hints = append(hints, toolHint{ToolName: name})
				}
			}
		}

		// 解析 @server:xxx
		if strings.Contains(line, "@server:") {
			parts := strings.Split(line, "@server:")
			for i := 1; i < len(parts); i++ {
				token := strings.Fields(parts[i])
				if len(token) == 0 {
					continue
				}
				name := strings.TrimRight(token[0], ",;.")
				if name == "" {
					continue
				}
				key := "server:" + name
				if seen[key] {
					continue
				}
				seen[key] = true
				hints = append(hints, toolHint{ServerName: name})
			}
		}
	}

	if len(hints) > 0 {
		logc.Infof(ctx, "[Agent] Extracted %d tool hints from skills", len(hints))
	}
	return hints
}

// matchToolHint 检查工具是否匹配任一提示
func matchToolHint(toolName, serverName string, hints []toolHint) bool {
	for _, h := range hints {
		// @server:xxx → 匹配该 server 下所有工具
		if h.ServerName != "" && h.ToolName == "" {
			if strings.EqualFold(h.ServerName, serverName) {
				return true
			}
		}
		// @tool:server.tool → 同时匹配 server 和 tool
		if h.ServerName != "" && h.ToolName != "" {
			if strings.EqualFold(h.ServerName, serverName) && strings.EqualFold(h.ToolName, toolName) {
				return true
			}
		}
	}
	return false
}

// --- MCP 工具加载 ---

// loadMCPTools 加载 MCP 工具，根据 selectedTools + toolHints 联合过滤，返回 eino 工具列表
func (e *AgentExecutor) loadMCPTools(ctx context.Context, workspaceID string, hints []toolHint, selectedTools []string) ([]tool.BaseTool, error) {
	servers, err := e.mcpRepo.GetByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP servers: %w", err)
	}

	var tools []tool.BaseTool
	seenTools := make(map[string]bool)
	filteredCount := 0

	for _, server := range servers {
		if !server.Enabled {
			continue
		}

		// 如果有 hints 且没有匹配该 server 的 hint，跳过整个 server
		if len(hints) > 0 && !serverMatchesHints(server.Name, hints) {
			continue
		}

		// 解析该 server 的用户选定工具
		var serverSelected map[string]bool
		if len(selectedTools) > 0 {
			serverSelected = make(map[string]bool)
			for _, st := range selectedTools {
				parts := strings.SplitN(st, "::", 2)
				if len(parts) == 2 && parts[0] == server.ID {
					serverSelected[parts[1]] = true
				}
			}
			if len(serverSelected) == 0 && !serverMatchesHints(server.Name, hints) {
				continue
			}
		}

		// 解析 headers
		var headers map[string]string
		if server.Headers != "" {
			if err := json.Unmarshal([]byte(server.Headers), &headers); err != nil {
				logc.Errorf(ctx, "[Agent] Failed to parse headers for server %s: %v", server.ID, err)
			}
		}

		// 获取 MCP 客户端
		logc.Infof(ctx, "[Agent] Connecting to MCP server: name=%s, id=%s, url=%s, type=%s", server.Name, server.ID, server.URL, server.Type)
		client, err := mcppool.DefaultMCPPool.GetOrCreateClient(ctx, server.ID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			logc.Errorf(ctx, "[Agent] Initial connect failed for server %s (%s): %v, retrying with fresh client...", server.Name, server.ID, err)
			mcppool.DefaultMCPPool.RemoveClient(server.ID)
			client, err = mcppool.DefaultMCPPool.GetOrCreateClient(ctx, server.ID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
			if err != nil {
				logc.Errorf(ctx, "[Agent] Retry connect failed for server %s (%s): %v", server.Name, server.ID, err)
				continue
			}
		}

		// 列出工具
		result, err := client.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			logc.Errorf(ctx, "[Agent] ListTools failed for server %s (%s): %v, closing stale session and retrying...", server.Name, server.ID, err)
			mcppool.DefaultMCPPool.RemoveClient(server.ID)
			client, err = mcppool.DefaultMCPPool.GetOrCreateClient(ctx, server.ID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
			if err != nil {
				logc.Errorf(ctx, "[Agent] Reconnect failed for server %s (%s): %v", server.Name, server.ID, err)
				continue
			}
			result, err = client.ListTools(ctx, mcp.ListToolsRequest{})
			if err != nil {
				logc.Errorf(ctx, "[Agent] ListTools retry failed for server %s (%s): %v", server.Name, server.ID, err)
				mcppool.DefaultMCPPool.RemoveClient(server.ID)
				continue
			}
		}
		logc.Infof(ctx, "[Agent] Server %s (%s): listed %d tools", server.Name, server.ID, len(result.Tools))

		for _, mcpTool := range result.Tools {
			// 用户选定工具过滤
			if len(selectedTools) > 0 && len(serverSelected) > 0 && !serverSelected[mcpTool.Name] {
				filteredCount++
				continue
			}

			// 如果有 hints，过滤不匹配的工具
			if len(hints) > 0 && !matchToolHint(mcpTool.Name, server.Name, hints) {
				filteredCount++
				continue
			}

			toolFullName := mcpTool.Name
			if seenTools[toolFullName] {
				toolFullName = fmt.Sprintf("%s__%s", server.Name, mcpTool.Name)
			}
			seenTools[toolFullName] = true

			// 构建 eino ParamsOneOf
			var paramsOneOf *schema.ParamsOneOf
			if mcpTool.InputSchema.Type != "" || len(mcpTool.InputSchema.Properties) > 0 {
				schemaBytes, marshalErr := json.Marshal(mcpTool.InputSchema)
				if marshalErr == nil {
					var js jsonschema.Schema
					if unmarshalErr := json.Unmarshal(schemaBytes, &js); unmarshalErr == nil {
						paramsOneOf = schema.NewParamsOneOfByJSONSchema(&js)
					}
				}
			} else if mcpTool.RawInputSchema != nil {
				var js jsonschema.Schema
				if unmarshalErr := json.Unmarshal(mcpTool.RawInputSchema, &js); unmarshalErr == nil {
					paramsOneOf = schema.NewParamsOneOfByJSONSchema(&js)
				}
			}

			desc := mcpTool.Description
			if len(desc) > 1024 {
				desc = desc[:1024]
			}

			wrapper := &mcpToolWrapper{
				toolInfo: &schema.ToolInfo{
					Name:        toolFullName,
					Desc:        desc,
					ParamsOneOf: paramsOneOf,
				},
				serverID: server.ID,
				toolName: mcpTool.Name,
				executor: e,
			}
			tools = append(tools, wrapper)
		}
	}

	if len(hints) > 0 {
		logc.Infof(ctx, "[Agent] Loaded %d tools (filtered %d by skill hints)", len(tools), filteredCount)
	} else {
		logc.Infof(ctx, "[Agent] Loaded %d tools (no hints, all available)", len(tools))
	}
	return tools, nil
}

// serverMatchesHints 检查 server 名称是否匹配任一 hint
func serverMatchesHints(serverName string, hints []toolHint) bool {
	for _, h := range hints {
		if h.ServerName != "" && strings.EqualFold(h.ServerName, serverName) {
			return true
		}
		// @tool:xxx (无 server 前缀) 表示任意 server 的该工具名
		if h.ServerName == "" && h.ToolName != "" {
			return true
		}
	}
	return false
}

// --- MCP 工具调用 ---

// callMCPTool 调用 MCP 工具
func (e *AgentExecutor) callMCPTool(ctx context.Context, serverID, toolName string, args map[string]any) (*mcp.CallToolResult, error) {
	server, err := e.mcpRepo.GetByID(serverID)
	if err != nil {
		return nil, fmt.Errorf("MCP server not found: %s", serverID)
	}

	var headers map[string]string
	if server.Headers != "" {
		json.Unmarshal([]byte(server.Headers), &headers)
	}

	// 如果是带前缀的工具名，提取实际工具名
	actualToolName := toolName
	if idx := strings.Index(toolName, "__"); idx != -1 {
		prefix := toolName[:idx]
		if prefix == server.Name {
			actualToolName = toolName[idx+2:]
		}
	}

	logc.Infof(ctx, "[Agent] Calling MCP tool: server=%s (%s), tool=%s", server.Name, serverID, actualToolName)

	client, err := mcppool.DefaultMCPPool.GetOrCreateClient(ctx, serverID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
	if err != nil {
		logc.Errorf(ctx, "[Agent] Initial connect failed for server %s: %v, retrying...", server.Name, err)
		mcppool.DefaultMCPPool.RemoveClient(serverID)
		client, err = mcppool.DefaultMCPPool.GetOrCreateClient(ctx, serverID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MCP server %s (%s): %w", server.Name, serverID, err)
		}
	}

	result, err := client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      actualToolName,
			Arguments: args,
		},
	})
	if err != nil {
		logc.Errorf(ctx, "[Agent] CallTool failed for %s on server %s: %v, closing stale session and retrying...", actualToolName, server.Name, err)
		mcppool.DefaultMCPPool.RemoveClient(serverID)
		client, err = mcppool.DefaultMCPPool.GetOrCreateClient(ctx, serverID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			return nil, fmt.Errorf("tool call %s failed: reconnect error: %w", actualToolName, err)
		}
		result, err = client.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      actualToolName,
				Arguments: args,
			},
		})
		if err != nil {
			logc.Errorf(ctx, "[Agent] CallTool retry failed for %s on server %s: %v", actualToolName, server.Name, err)
			mcppool.DefaultMCPPool.RemoveClient(serverID)
			return nil, fmt.Errorf("tool call %s failed after retry: %w", actualToolName, err)
		}
	}

	logc.Infof(ctx, "[Agent] MCP tool %s call succeeded on server %s", actualToolName, server.Name)
	return result, nil
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
