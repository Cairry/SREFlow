package nodes

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// WebhookConfig webhook配置
type WebhookConfig struct {
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
	Timeout int               `json:"timeout"`
}

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

// Execute 执行输出
func ExecuteOutput(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	outputExecutor := NewOutputExecutor()
	return outputExecutor.Execute(ctx, nc)
}

func (e *OutputExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	outputType := getStr(nodeCtx.Config, "outputType", "webhook")
	customFilename := getStr(nodeCtx.Config, "customFilename", "")

	logc.Infof(ctx, "[Output] Processing output (type: %s)", outputType)
	logc.Infof(ctx, "[Output] Input variables: %d", len(nodeCtx.Input))

	// 构建输入内容
	var inputContent string
	for key, value := range nodeCtx.Input {
		if inputContent != "" {
			inputContent += "\n"
		}
		inputContent += fmt.Sprintf("[%s]: %v", key, value)
	}

	// 根据输出类型生成输出
	var output string
	switch outputType {
	case "webhook":
		webhookResult, err := executeWebhook(ctx, nodeCtx.Config, nodeCtx.Input)
		if err != nil {
			return &models.NodeResult{
				Outputs: map[string]any{
					"output":     "",
					"outputType": outputType,
					"error":      err.Error(),
				},
			}, nil
		}
		result := map[string]any{
			"url":             webhookResult.URL,
			"request":         map[string]any{"body": webhookResult.RequestBody},
			"output":          webhookResult.ResponseBody,
			"outputType":      outputType,
			"statusCode":      webhookResult.StatusCode,
			"responseHeaders": webhookResult.ResponseHeaders,
		}
		return &models.NodeResult{
			Outputs: result,
		}, nil
	default:
		output = inputContent
	}

	// 如果有自定义文件名，将文件名作为元数据包含
	if customFilename != "" {
		logc.Infof(ctx, "[Output] Output file: %s", customFilename)
	}

	result := map[string]any{
		"output":     output,
		"outputType": outputType,
		"filename":   customFilename,
	}

	return &models.NodeResult{
		Outputs: result,
	}, nil
}

// WebhookResult webhook执行结果
type WebhookResult struct {
	URL             string
	RequestBody     string
	StatusCode      int
	ResponseBody    string
	ResponseHeaders map[string]string
}

// executeWebhook 执行webhook请求
func executeWebhook(ctx context.Context, config map[string]any, input map[string]any) (*WebhookResult, error) {
	// 获取webhook配置
	webhookConfig := getWebhookConfig(config)

	if webhookConfig.URL == "" {
		return nil, fmt.Errorf("webhook URL is required")
	}

	logc.Infof(ctx, "[Output/Webhook] Sending %s request to %s", webhookConfig.Method, webhookConfig.URL)

	// 替换模板变量
	body := replaceTemplateVars(webhookConfig.Body, input)

	// 创建HTTP请求
	var req *http.Request
	var err error

	method := webhookConfig.Method
	if method == "" {
		method = "POST"
	}

	if method == "GET" {
		// GET请求将body作为query参数处理
		url := webhookConfig.URL
		if strings.Contains(url, "?") {
			url += "&body=" + body
		} else {
			url += "?body=" + body
		}
		req, err = http.NewRequest(method, url, nil)
	} else {
		req, err = http.NewRequest(method, webhookConfig.URL, strings.NewReader(body))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	for key, value := range webhookConfig.Headers {
		req.Header.Set(key, value)
	}

	// 设置超时
	timeout := webhookConfig.Timeout
	if timeout <= 0 {
		timeout = 30
	}
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 提取响应头
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[key] = values[0]
		}
	}

	logc.Infof(ctx, "[Output/Webhook] Response status: %d", resp.StatusCode)

	return &WebhookResult{
		URL:             webhookConfig.URL,
		RequestBody:     body,
		StatusCode:      resp.StatusCode,
		ResponseBody:    string(bodyBytes),
		ResponseHeaders: responseHeaders,
	}, nil
}

// getWebhookConfig 从配置中获取webhook配置
func getWebhookConfig(config map[string]any) WebhookConfig {
	// 优先从 webhook 嵌套对象中获取
	webhookMap, ok := config["webhook"].(map[string]any)
	if ok {
		return parseWebhookFromMap(webhookMap)
	}

	// 兼容直接配置在顶层的情况
	return parseWebhookFromMap(config)
}

// parseWebhookFromMap 从map中解析webhook配置
func parseWebhookFromMap(webhookMap map[string]any) WebhookConfig {
	// 解析headers
	headers := make(map[string]string)
	if headersMap, ok := webhookMap["headers"].(map[string]any); ok {
		for k, v := range headersMap {
			if str, ok := v.(string); ok {
				headers[k] = str
			}
		}
	}
	if len(headers) == 0 {
		headers["Content-Type"] = "application/json"
	}

	method := "POST"
	if m, ok := webhookMap["method"].(string); ok && m != "" {
		method = m
	}

	timeout := 30
	if t, ok := webhookMap["timeout"].(float64); ok {
		timeout = int(t)
	}

	// 获取URL并清理可能的尾随字符
	url := getStr(webhookMap, "url", "")
	url = strings.TrimRight(url, ",. \t\n")

	// 获取body
	body := "{}"
	if b, ok := webhookMap["body"].(string); ok && b != "" {
		body = b
	} else if b, ok := webhookMap["data"].(string); ok && b != "" {
		// 兼容 data 字段作为 body 的别名
		body = b
	}

	return WebhookConfig{
		URL:     url,
		Method:  method,
		Headers: headers,
		Body:    body,
		Timeout: timeout,
	}
}

func replaceTemplateVars(template string, input map[string]any) string {
	result := template

	// 收集所有可能的占位符（避免在迭代过程中修改map）
	placeholders := []string{}
	seen := make(map[string]bool)

	for key := range input {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if !seen[placeholder] {
			placeholders = append(placeholders, placeholder)
			seen[placeholder] = true
		}
	}

	// 按占位符长度降序替换（避免部分匹配问题）
	for _, placeholder := range placeholders {
		// 提取key（去掉{{和}}）
		key := placeholder[2 : len(placeholder)-2]
		if value, ok := input[key]; ok {
			replacement := fmt.Sprintf("%v", value)
			result = replaceString(result, placeholder, replacement)
		}
	}

	// 清理未替换的占位符（可选：避免模板语法残留）
	result = cleanUnreplacedPlaceholders(result)

	return result
}

// cleanUnreplacedPlaceholders 清理未替换的模板占位符
func cleanUnreplacedPlaceholders(s string) string {
	// 移除孤立的 {{ 但保留格式
	result := s
	// 移除末尾的逗号（如果是由未替换的占位符导致的）
	for {
		trimmed := strings.TrimRight(result, ",")
		if trimmed == result {
			break
		}
		result = strings.TrimSpace(trimmed)
	}
	return result
}
