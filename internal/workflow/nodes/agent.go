package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	einocomponentmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	einocallbacks "github.com/cloudwego/eino/utils/callbacks"
	"github.com/eino-contrib/jsonschema"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/zeromicro/go-zero/core/logc"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/internal/workflow/mcppool"
)

// AgentExecutor Agent节点执行器 - 基于 eino ReAct 模式
type AgentExecutor struct {
	DefaultNodeExecutor
	mcpRepo   repos.MCPServerRepoInter
	skillRepo repos.SkillRepoInter
}

// toolCallRecord 工具调用记录
type toolCallRecord struct {
	ToolName   string `json:"toolName"`
	ServerID   string `json:"serverId"`
	ServerName string `json:"serverName"`
	Arguments  string `json:"arguments"`
	Result     string `json:"result"`
}

// skillInfo 存储 Skill 基本信息
type skillInfo struct {
	ID          string
	Name        string
	Description string
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

	if apiKey == "" && baseURL == "https://api.openai.com/v1" {
		return errorResult("apiKey is required when using OpenAI API"), nil
	}

	// 解析用户输入（支持模板变量）
	resolvedUserPrompt := resolveTemplate(userPrompt, nodeCtx.Input)
	if resolvedUserPrompt == "" {
		resolvedUserPrompt = collectInputString(nodeCtx.Input)
	}

	// 加载 Skill 列表及完整模型（用于构建统一 skill 工具）
	skillList, skillMap, err := e.loadSkillList(ctx, skillIDsStr)
	if err != nil {
		logc.Errorf(ctx, "[Agent] Failed to load skill list: %v", err)
	}

	var einoTools []tool.BaseTool

	// 有 skill 时创建统一 skill 工具（单工具接受 name 参数，描述动态列出所有可用 skill）
	if len(skillList) > 0 {
		skillDesc := buildSkillToolDescription(skillList)
		st := &skillTool{
			toolInfo: &schema.ToolInfo{
				Name: "skill",
				Desc: skillDesc,
				ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
					"skill": {
						Type:     schema.String,
						Desc:     "The skill name to load. E.g., \"pdf\" or \"xlsx\"",
						Required: true,
					},
				}),
			},
			skills: skillMap,
		}
		einoTools = append(einoTools, st)
	}

	// 加载 workspace 下所有 MCP 工具，让 agent 自主发现并调用
	mcpTools, loadErr := e.loadMCPTools(ctx, nodeCtx.WorkspaceID)
	if loadErr != nil {
		logc.Errorf(ctx, "[Agent] Failed to load MCP tools: %v", loadErr)
	}
	einoTools = append(einoTools, mcpTools...)

	logc.Infof(ctx, "[Agent] Model: %s, MaxIterations: %d, Tools: %d, Skills: %d",
		modelID, maxIterations, len(einoTools), len(skillList))

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

	logc.Infof(ctx, "[Agent] Chat model created: %v", chatModel.GetType())

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

	// 构建 Skill 系统提示（渐进式披露：引导 agent 使用 skill 工具按需加载）
	skillListPrompt := ""
	if len(skillList) > 0 {
		skillListPrompt = skillSystemPrompt
	}

	// 创建 eino ReAct Agent，通过 MessageModifier 在每轮模型调用前注入 skill 列表
	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig:      toolsConfig,
		MaxStep:          maxIterations * 2,
		MessageModifier: func(ctx context.Context, msgs []*schema.Message) []*schema.Message {
			if skillListPrompt == "" {
				return msgs
			}
			return append([]*schema.Message{schema.SystemMessage(skillListPrompt)}, msgs...)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create react agent: %w", err)
	}

	logc.Infof(ctx, "[Agent] ReAct agent created successfully")

	// 构建初始消息
	messages := []*schema.Message{schema.UserMessage(resolvedUserPrompt)}

	// 构建 callback，打印 agent 每轮思考过程
	stepNum := 0
	var stepMu sync.Mutex
	modelHandler := &einocallbacks.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *einocomponentmodel.CallbackInput) context.Context {
			if input == nil {
				return ctx
			}
			logc.Infof(ctx, "[Agent] LLM call - Messages count: %d", len(input.Messages))
			for i, msg := range input.Messages {
				logc.Infof(ctx, "[Agent] LLM input[%d] role=%s content=%s", i, msg.Role, truncateLog(msg.Content, 2000))
				for _, tc := range msg.ToolCalls {
					logc.Infof(ctx, "[Agent] LLM input[%d] ToolCall: %s(%s)", i, tc.Function.Name, tc.Function.Arguments)
				}
				if msg.ToolCallID != "" {
					logc.Infof(ctx, "[Agent] LLM input[%d] ToolCallID: %s", i, msg.ToolCallID)
				}
			}
			return ctx
		},
		OnEnd: func(ctx context.Context, info *callbacks.RunInfo, output *einocomponentmodel.CallbackOutput) context.Context {
			stepMu.Lock()
			stepNum++
			step := stepNum
			stepMu.Unlock()
			if output.Message != nil {
				if output.Message.Content != "" {
					logc.Infof(ctx, "[Agent] Step %d - Thought: %s", step, truncateLog(output.Message.Content, 800))
				}
				for _, tc := range output.Message.ToolCalls {
					logc.Infof(ctx, "[Agent] Step - ToolCall: %v(%v)", step, tc.Function.Name, tc.Function.Arguments)
				}
			}
			return ctx
		},
	}
	agentCallback := react.BuildAgentCallback(modelHandler, nil)

	// 执行 Agent
	start := time.Now()
	msg, err := reactAgent.Generate(ctx, messages, agent.WithComposeOptions(compose.WithCallbacks(agentCallback)))
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
		result["iterations"] = len(toolCallRecords) + 1
		result["toolCalls"] = toolCallRecords
	} else {
		result["iterations"] = 1
	}

	return newResult(result), nil
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

// loadSkillList 加载选中 Skill 的基本信息（name + description）及完整模型映射。
// 返回 skillInfo 列表（用于构建工具描述/系统提示）和 name -> models.Skill 映射（用于 skillTool 运行时查找）。
func (e *AgentExecutor) loadSkillList(ctx context.Context, skillIDsStr string) ([]skillInfo, map[string]*models.Skill, error) {
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

	var list []skillInfo
	skillMap := make(map[string]*models.Skill)
	for _, sk := range skills {
		if !sk.Enabled {
			continue
		}
		list = append(list, skillInfo{
			ID:          sk.ID,
			Name:        sk.Name,
			Description: sk.Description,
		})
		skillMap[sk.Name] = sk
	}
	return list, skillMap, nil
}

// isSessionError detects MCP session errors that require reconnection.
func isSessionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Invalid session ID") ||
		strings.Contains(errStr, "session expired") ||
		strings.Contains(errStr, "session not found")
}

// loadMCPTools 并行加载 workspace 下所有启用的 MCP 工具，返回 eino 工具列表
func (e *AgentExecutor) loadMCPTools(ctx context.Context, workspaceID string) ([]tool.BaseTool, error) {
	servers, err := e.mcpRepo.GetByWorkspaceID(workspaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP servers: %w", err)
	}

	type serverResult struct {
		tools      []tool.BaseTool
		serverName string
	}

	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		results   []serverResult
		seenTools = make(map[string]bool)
	)

	for _, server := range servers {
		if !server.Enabled {
			continue
		}
		wg.Add(1)
		go func(srv *models.MCPServer) {
			defer wg.Done()
			if !server.Enabled {
				logc.Debugf(ctx, "[Agent] Skipping disabled MCP server %s (%s)", srv.Name, srv.ID)
				return
			}

			// 解析 headers
			var headers map[string]string
			if srv.Headers != "" {
				if err := json.Unmarshal([]byte(srv.Headers), &headers); err != nil {
					logc.Errorf(ctx, "[Agent] Failed to parse headers for server %s: %v", srv.ID, err)
				}
			}

			// 获取 MCP 客户端
			client, err := mcppool.DefaultMCPPool.GetOrCreateClient(ctx, srv.ID, srv.URL, srv.Type, headers, srv.APIKey, srv.Timeout)
			if err != nil {
				client, err = mcppool.DefaultMCPPool.ReconnectClient(ctx, srv.ID, srv.URL, srv.Type, headers, srv.APIKey, srv.Timeout)
				if err != nil {
					logc.Debugf(ctx, "[Agent] Connect failed for server %s (%s): %v", srv.Name, srv.ID, err)
					return
				}
			}

			// 列出工具
			result, err := client.ListTools(ctx, mcp.ListToolsRequest{})
			if err != nil {
				if isSessionError(err) {
					logc.Infof(ctx, "[Agent] Session expired for server %s, reconnecting...", srv.Name)
					client, err = mcppool.DefaultMCPPool.ReconnectClient(ctx, srv.ID, srv.URL, srv.Type, headers, srv.APIKey, srv.Timeout)
				} else {
					logc.Errorf(ctx, "[Agent] ListTools failed for server %s: %v", srv.Name, err)
					return
				}
				if err == nil {
					result, err = client.ListTools(ctx, mcp.ListToolsRequest{})
				}
				if err != nil {
					logc.Errorf(ctx, "[Agent] ListTools retry failed for server %s: %v", srv.Name, err)
					return
				}
			}
			logc.Debugf(ctx, "[Agent] Server %s (%s): listed %d tools", srv.Name, srv.ID, len(result.Tools))

			var srvTools []tool.BaseTool
			for _, mcpTool := range result.Tools {
				// 重名保护（加锁访问共享 seenTools）
				mu.Lock()
				toolFullName := mcpTool.Name
				if seenTools[toolFullName] {
					toolFullName = fmt.Sprintf("%s__%s", srv.Name, mcpTool.Name)
				}
				seenTools[toolFullName] = true
				mu.Unlock()

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
					serverID:   srv.ID,
					serverName: srv.Name,
					toolName:   mcpTool.Name,
					serverURL:  srv.URL,
					serverType: srv.Type,
					headers:    headers,
					apiKey:     srv.APIKey,
					timeout:    srv.Timeout,
					executor:   e,
				}
				srvTools = append(srvTools, wrapper)
			}

			mu.Lock()
			results = append(results, serverResult{tools: srvTools, serverName: srv.Name})
			mu.Unlock()
		}(server)
	}

	wg.Wait()

	var tools []tool.BaseTool
	for _, r := range results {
		tools = append(tools, r.tools...)
	}

	return tools, nil
}

// callMCPTool 调用 MCP 工具（使用缓存的 server 元数据，无需查询 DB）
func (e *AgentExecutor) callMCPTool(ctx context.Context, serverID, serverName, toolName string, serverURL string, serverType models.MCPServerType, headers map[string]string, apiKey string, timeout int, args map[string]any) (*mcp.CallToolResult, error) {
	// 如果是带前缀的工具名，提取实际工具名
	actualToolName := toolName
	if idx := strings.Index(toolName, "__"); idx != -1 {
		prefix := toolName[:idx]
		if prefix == serverName {
			actualToolName = toolName[idx+2:]
		}
	}

	logc.Debugf(ctx, "[Agent] Calling MCP tool: server=%s, tool=%s", serverName, actualToolName)

	client, err := mcppool.DefaultMCPPool.GetOrCreateClient(ctx, serverID, serverURL, serverType, headers, apiKey, timeout)
	if err != nil {
		client, err = mcppool.DefaultMCPPool.ReconnectClient(ctx, serverID, serverURL, serverType, headers, apiKey, timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MCP server %s: %w", serverName, err)
		}
	}

	result, err := client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      actualToolName,
			Arguments: args,
		},
	})
	if err != nil {
		// 只对 session 错误做重连，其他错误直接返回
		if !isSessionError(err) {
			return nil, fmt.Errorf("tool call %s failed on %s: %w", actualToolName, serverName, err)
		}
		logc.Debugf(ctx, "[Agent] Session expired for server %s, reconnecting...", serverName)
		client, err = mcppool.DefaultMCPPool.ReconnectClient(ctx, serverID, serverURL, serverType, headers, apiKey, timeout)
		if err != nil {
			return nil, fmt.Errorf("reconnect failed for %s: %w", serverName, err)
		}
		result, err = client.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      actualToolName,
				Arguments: args,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("tool call %s failed after retry: %w", actualToolName, err)
		}
	}

	logc.Debugf(ctx, "[Agent] MCP tool %s succeeded on server %s", actualToolName, serverName)
	return result, nil
}
