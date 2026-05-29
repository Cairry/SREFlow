package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"SREFlow/internal/models"
)

// SmartRouterExecutor 智能路由执行器
type SmartRouterExecutor struct {
	DefaultNodeExecutor
}

// NewSmartRouterExecutor 创建智能路由执行器
func NewSmartRouterExecutor() *SmartRouterExecutor {
	return &SmartRouterExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "SmartRouter",
			Category:    "logic",
			Description: "AI-powered routing: LLM analyzes input and selects the best branch",
		},
	}
}

// smartRouterRequest LLM 请求结构
type smartRouterRequest struct {
	Model          string               `json:"model"`
	Messages       []smartRouterMessage `json:"messages"`
	MaxTokens      int                  `json:"max_tokens,omitempty"`
	ResponseFormat *smartRouterRespFmt  `json:"response_format,omitempty"`
}

type smartRouterMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type smartRouterRespFmt struct {
	Type       string      `json:"type"`
	JSONSchema interface{} `json:"json_schema,omitempty"`
}

// smartRouterResponse LLM 响应结构
type smartRouterResponse struct {
	ID      string              `json:"id"`
	Choices []smartRouterChoice `json:"choices"`
	Usage   *smartRouterUsage   `json:"usage,omitempty"`
}

type smartRouterChoice struct {
	Index        int                `json:"index"`
	Message      smartRouterMessage `json:"message"`
	FinishReason string             `json:"finish_reason"`
}

type smartRouterUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SmartRouterResult 结构化路由结果
type SmartRouterResult struct {
	SelectedBranch string  `json:"selectedBranch"`
	Confidence     float64 `json:"confidence"`
	Reasoning      string  `json:"reasoning"`
}

// smartRouterErrorDetail LLM 错误响应
type smartRouterErrorDetail struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (e *SmartRouterExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	// 读取配置
	baseURL := getStr(nodeCtx.Config, "baseURL", "https://api.openai.com/v1")
	apiKey := getStr(nodeCtx.Config, "apiKey", "")
	modelID := getStr(nodeCtx.Config, "model", "gpt-4o")
	maxTokens := getInt(nodeCtx.Config, "maxTokens", 1024)
	timeoutMs := getInt(nodeCtx.Config, "timeout", 60000)

	// branches: JSON 格式的分支定义
	// 格式: [{"id": "branch1", "label": "下游服务异常", "description": "当日志中显示下游调用超时或 5xx"}, ...]
	branchesStr := getStr(nodeCtx.Config, "branches", "")
	if branchesStr == "" {
		return errorResult("branches is required - please define at least 2 routing branches"), nil
	}

	var branches []map[string]string
	if err := json.Unmarshal([]byte(branchesStr), &branches); err != nil {
		return errorResult(fmt.Sprintf("invalid branches JSON: %v", err)), nil
	}

	if len(branches) < 2 {
		return errorResult("at least 2 branches are required for routing"), nil
	}

	// 收集上游输入
	inputStr := collectInputString(nodeCtx.Input)

	// 构建 system prompt
	systemPrompt := buildRouterSystemPrompt(branches)

	// 构建 user prompt
	userPrompt := fmt.Sprintf("Based on the following data, determine which branch to route to:\n\n%s", inputStr)

	// 构建消息
	messages := []smartRouterMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	// 构建请求（强制 JSON 输出）
	req := smartRouterRequest{
		Model:     modelID,
		Messages:  messages,
		MaxTokens: maxTokens,
		ResponseFormat: &smartRouterRespFmt{
			Type: "json_object",
		},
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("[SmartRouter] Model: %s, Branches: %d", modelID, len(branches))

	// 发送请求
	httpCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(httpCtx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("User-Agent", "SREFlow/1.0")

	start := time.Now()
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	resp, err := client.Do(httpReq)
	latency := time.Since(start).Seconds()

	if err != nil {
		return errorResult(fmt.Sprintf("LLM request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("LLM API returned %d: %s", resp.StatusCode, string(body))), nil
	}

	var llmResp smartRouterResponse
	if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {
		return errorResult(fmt.Sprintf("failed to decode LLM response: %v", err)), nil
	}

	if len(llmResp.Choices) == 0 {
		return errorResult("LLM returned no choices"), nil
	}

	// 解析 LLM 返回的 JSON
	content := llmResp.Choices[0].Message.Content
	log.Printf("[SmartRouter] LLM response (%.2fs): %s", latency, truncateLog(content, 300))

	var routerResult SmartRouterResult
	if err := json.Unmarshal([]byte(content), &routerResult); err != nil {
		// 尝试从 markdown 代码块中提取 JSON
		extracted := extractJSONFromMarkdown(content)
		if err2 := json.Unmarshal([]byte(extracted), &routerResult); err2 != nil {
			return errorResult(fmt.Sprintf("failed to parse LLM routing result: %v, raw: %s", err, truncateLog(content, 200))), nil
		}
	}

	// 验证 selectedBranch 在分支列表中
	validBranch := false
	for _, b := range branches {
		if b["id"] == routerResult.SelectedBranch {
			validBranch = true
			break
		}
	}
	if !validBranch && len(branches) > 0 {
		// 尝试模糊匹配：LLM 可能返回 label 而不是 id
		for _, b := range branches {
			if strings.EqualFold(b["label"], routerResult.SelectedBranch) {
				routerResult.SelectedBranch = b["id"]
				validBranch = true
				break
			}
		}
	}
	if !validBranch {
		routerResult.SelectedBranch = branches[0]["id"]
		routerResult.Confidence = 0.0
		routerResult.Reasoning = fmt.Sprintf("LLM selected invalid branch '%s', fell back to first branch. %s", routerResult.SelectedBranch, routerResult.Reasoning)
	}

	log.Printf("[SmartRouter] Selected: %s, Confidence: %.2f", routerResult.SelectedBranch, routerResult.Confidence)

	result := map[string]any{
		"output":         routerResult.SelectedBranch,
		"selectedBranch": routerResult.SelectedBranch,
		"confidence":     routerResult.Confidence,
		"reasoning":      routerResult.Reasoning,
		"latency":        latency,
	}

	if llmResp.Usage != nil {
		result["usage"] = map[string]any{
			"inputTokens":  llmResp.Usage.PromptTokens,
			"outputTokens": llmResp.Usage.CompletionTokens,
			"totalTokens":  llmResp.Usage.TotalTokens,
		}
	}

	nodeResult := newResult(result)

	// 根据 selectedBranch 和 HandleEdges 确定下游节点（只执行选中分支的下游）
	if nodeCtx.HandleEdges != nil {
		if targetNodeID, ok := nodeCtx.HandleEdges[routerResult.SelectedBranch]; ok {
			nodeResult.Next = []string{targetNodeID}
			log.Printf("[SmartRouter] Routing to branch '%s' -> node %s", routerResult.SelectedBranch, targetNodeID)
		}
	}

	return nodeResult, nil
}

// buildRouterSystemPrompt 构建路由 system prompt
func buildRouterSystemPrompt(branches []map[string]string) string {
	var sb strings.Builder

	sb.WriteString("You are an intelligent routing assistant for SRE incident triage. ")
	sb.WriteString("Your job is to analyze the input data and determine which investigation branch is most appropriate.\n\n")

	sb.WriteString("Available branches:\n")
	for _, b := range branches {
		id := b["id"]
		label := b["label"]
		desc := b["description"]
		sb.WriteString(fmt.Sprintf("- **%s**: %s", id, label))
		if desc != "" {
			sb.WriteString(fmt.Sprintf(" — %s", desc))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("\nYou MUST respond with a JSON object in exactly this format:\n")
	sb.WriteString("```json\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"selectedBranch\": \"<branch_id>\",\n")
	sb.WriteString("  \"confidence\": <0.0-1.0>,\n")
	sb.WriteString("  \"reasoning\": \"<brief explanation of why this branch was selected>\"\n")
	sb.WriteString("}\n")
	sb.WriteString("```\n\n")

	sb.WriteString("Rules:\n")
	sb.WriteString("1. `selectedBranch` MUST be one of the branch IDs listed above\n")
	sb.WriteString("2. `confidence` reflects how certain you are (0.0 = guess, 1.0 = certain)\n")
	sb.WriteString("3. `reasoning` should be a concise one-sentence explanation\n")
	sb.WriteString("4. If multiple branches could apply, pick the one most likely to find the root cause\n")
	sb.WriteString("5. If data is insufficient to determine, pick the broadest/most general branch and set confidence below 0.5\n")

	return sb.String()
}

// collectInputString 将上游输入收集为字符串
func collectInputString(input map[string]any) string {
	if len(input) == 0 {
		return "(no input data)"
	}

	var parts []string
	for key, value := range input {
		valStr := fmt.Sprintf("%v", value)
		// 如果值太长，截断
		if len(valStr) > 2000 {
			valStr = valStr[:2000] + "...(truncated)"
		}
		parts = append(parts, fmt.Sprintf("**%s**:\n%s", key, valStr))
	}
	return strings.Join(parts, "\n\n")
}

// extractJSONFromMarkdown 从 markdown 代码块中提取 JSON
func extractJSONFromMarkdown(content string) string {
	trimmed := strings.TrimSpace(content)

	if strings.HasPrefix(trimmed, "```json") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimSuffix(trimmed, "```")
	} else if strings.HasPrefix(trimmed, "```") {
		parts := strings.SplitN(trimmed, "\n", 2)
		if len(parts) > 1 {
			rest := parts[1]
			if idx := strings.Index(rest, "```"); idx != -1 {
				trimmed = rest[:idx]
			}
		}
	}

	return strings.TrimSpace(trimmed)
}
