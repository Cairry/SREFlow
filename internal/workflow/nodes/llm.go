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

// OpenAI API 请求结构
type OpenAIRequest struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// OpenAI API 响应结构
type OpenAIResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// OpenAI API 错误响应
type OpenAIErrorResponse struct {
	Error OpenAIErrorDetail `json:"error"`
}

type OpenAIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    any    `json:"code"`
}

// LLMExecutor LLM执行器
type LLMExecutor struct {
	DefaultNodeExecutor
}

// NewLLMExecutor 创建LLM执行器
func NewLLMExecutor() *LLMExecutor {
	return &LLMExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "LLM",
			Category:    "processing",
			Description: "Call LLM models for text generation",
		},
	}
}

// ExecuteLLM 执行LLM调用
func ExecuteLLM(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	llmExecutor := NewLLMExecutor()
	return llmExecutor.Execute(ctx, nc)
}

func (e *LLMExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	prompt := getStr(nodeCtx.Config, "prompt", "")
	if prompt == "" {
		return errorResult("prompt not found in config"), nil
	}

	// 验证其他必需配置 - 优先使用从数据源合并的配置
	modelID := getStr(nodeCtx.Config, "model", "gpt-4o")
	baseURL := getStr(nodeCtx.Config, "baseURL", "https://api.openai.com/v1")
	apiKey := getStr(nodeCtx.Config, "apiKey", "")

	// 如果使用默认OpenAI但没有提供API key，返回错误
	if baseURL == "https://api.openai.com/v1" && apiKey == "" {
		return errorResult("apiKey is required when using OpenAI API"), nil
	}

	maxTokens := getInt(nodeCtx.Config, "maxTokens", 4096)
	timeoutMs := getInt(nodeCtx.Config, "timeout", 60000)
	systemPrompt := getStr(nodeCtx.Config, "systemPrompt", "")

	// 变量替换：{{varName}} -> input value
	resolvedPrompt := resolveTemplate(prompt, nodeCtx.Input)
	resolvedSystem := resolveTemplate(systemPrompt, nodeCtx.Input)

	// 构建消息列表
	var messages []Message
	if resolvedSystem != "" {
		messages = append(messages, Message{Role: "system", Content: resolvedSystem})
	}
	messages = append(messages, Message{Role: "user", Content: resolvedPrompt})

	log.Printf("[LLM] Model: %s, BaseURL: %s", modelID, baseURL)
	log.Printf("[LLM] Prompt: %s", truncateLog(resolvedPrompt, 200))
	log.Printf("[LLM] Input variables: %d", len(nodeCtx.Input))

	// 构建请求体
	request := OpenAIRequest{
		Model:     modelID,
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		log.Printf("[LLM] Failed to marshal request: %v", err)
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	log.Printf("[LLM] Sending %d messages to AI API", len(messages))
	for i, msg := range messages {
		log.Printf("[LLM] Message %d: Role=%s, Content=%s", i, msg.Role, truncateLog(msg.Content, 100))
	}

	// 创建HTTP请求
	httpCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "POST", baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Printf("[LLM] Failed to create request: %v", err)
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "SREFlow/1.0")

	log.Printf("[LLM] API Key: %s...", apiKey[:8])

	// 发送请求
	start := time.Now()
	client := &http.Client{Timeout: time.Duration(timeoutMs) * time.Millisecond}
	resp, err := client.Do(req)
	latency := time.Since(start).Seconds()

	if err != nil {
		log.Printf("[LLM] HTTP request error: %v", err)
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[LLM] API returned status %d: %s", resp.StatusCode, string(body))

		var errResp OpenAIErrorResponse
		if json.Unmarshal(body, &errResp) == nil {
			return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, errResp.Error.Message)
		}
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var openAIResp OpenAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&openAIResp); err != nil {
		log.Printf("[LLM] Failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 检查有效响应
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no valid response content")
	}

	aiResponse := openAIResp.Choices[0].Message.Content
	log.Printf("[LLM] AI response (%s, %.2fs): %s", modelID, latency, truncateLog(aiResponse, 200))

	result := map[string]any{
		"output":  aiResponse,
		"latency": latency,
	}

	// 提取 token 用量
	if openAIResp.Usage != nil {
		result["usage"] = map[string]any{
			"maxTokens":    maxTokens,
			"inputTokens":  openAIResp.Usage.PromptTokens,
			"outputTokens": openAIResp.Usage.CompletionTokens,
			"totalTokens":  openAIResp.Usage.TotalTokens,
		}
	}

	// 尝试解析 AI 返回中的 JSON 结构化数据
	if parsed := tryParseJSON(aiResponse); parsed != nil {
		result["parsed"] = parsed
	}

	log.Printf("[LLM] Final result keys: %v", getMapKeys(result))
	log.Printf("[LLM] Final result: %+v", result)

	return &models.NodeResult{Outputs: result}, nil
}

// getMapKeys 获取map的所有键
func getMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// resolveTemplate 将模板中的 {{varName}} 替换为 input map 中的值
func resolveTemplate(template string, inputs map[string]any) string {
	result := template
	for key, value := range inputs {
		placeholder := fmt.Sprintf("{{%s}}", key)
		valStr := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, valStr)
	}
	return result
}

// tryParseJSON 尝试从 AI 返回内容中解析 JSON 结构化数据
// 支持处理 markdown 代码块包裹的 JSON
func tryParseJSON(content string) map[string]any {
	trimmed := strings.TrimSpace(content)

	// 去掉 markdown code block
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
	trimmed = strings.TrimSpace(trimmed)

	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return nil
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(trimmed), &result); err != nil {
		// 不是有效 JSON，返回原始内容
		return map[string]any{"raw": trimmed}
	}
	return result
}

// truncateLog 截断日志输出
func truncateLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func float32Ptr(v float32) *float32 { return &v }

func intPtr(v int) *int { return &v }
