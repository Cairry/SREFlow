package nodes

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"SREFlow/internal/models"
)

// HTTPExecutor HTTP请求执行器
type HTTPExecutor struct {
	DefaultNodeExecutor
}

// NewHTTPExecutor 创建HTTP执行器
func NewHTTPExecutor() *HTTPExecutor {
	return &HTTPExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "HTTP",
			Category:    "output",
			Description: "Send HTTP requests to external endpoints",
		},
	}
}

// ExecuteHTTP 执行HTTP请求
func ExecuteHTTP(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	httpExecutor := NewHTTPExecutor()
	return httpExecutor.Execute(ctx, nc)
}

func (e *HTTPExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	url := getStr(nodeCtx.Config, "url", "")
	method := getStr(nodeCtx.Config, "method", "POST")
	body := getStr(nodeCtx.Config, "body", "{}")
	timeout := getInt(nodeCtx.Config, "timeout", 30)

	if url == "" {
		return errorResult("url is required"), nil
	}

	log.Printf("[HTTP] Sending %s request to %s", method, url)

	// 替换模板变量
	resolvedBody := resolveTemplateVars(body, nodeCtx.Input)

	// 构建请求
	var req *http.Request
	var err error

	normalizedMethod := strings.ToUpper(method)
	if normalizedMethod == "GET" {
		sep := "?"
		if strings.Contains(url, "?") {
			sep = "&"
		}
		req, err = http.NewRequest("GET", url+sep+"body="+resolvedBody, nil)
	} else {
		req, err = http.NewRequest(normalizedMethod, url, strings.NewReader(resolvedBody))
	}

	if err != nil {
		return errorResult(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// 设置请求头
	headersRaw, ok := nodeCtx.Config["headers"].(map[string]any)
	if ok {
		for k, v := range headersRaw {
			if s, ok := v.(string); ok {
				req.Header.Set(k, s)
			}
		}
	}
	// 默认 Content-Type
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 设置超时
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return errorResult(fmt.Sprintf("request failed: %v", err)), nil
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read response body: %v", err)), nil
	}

	// 收集响应头
	responseHeaders := make(map[string]string)
	for k, values := range resp.Header {
		if len(values) > 0 {
			responseHeaders[k] = values[0]
		}
	}

	log.Printf("[HTTP] Response status: %d, body length: %d", resp.StatusCode, len(bodyBytes))

	return newResult(map[string]any{
		"url":             url,
		"method":          method,
		"requestBody":     resolvedBody,
		"statusCode":      resp.StatusCode,
		"responseBody":    string(bodyBytes),
		"responseHeaders": responseHeaders,
	}), nil
}

// resolveTemplateVars 替换模板变量，支持 {{key}} 语法
func resolveTemplateVars(template string, input map[string]any) string {
	result := template
	seen := make(map[string]bool)

	var placeholders []string
	for key := range input {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if !seen[placeholder] {
			placeholders = append(placeholders, placeholder)
			seen[placeholder] = true
		}
	}

	// 按长度降序替换，避免部分匹配
	for _, placeholder := range placeholders {
		key := placeholder[2 : len(placeholder)-2]
		if value, ok := input[key]; ok {
			result = replaceString(result, placeholder, fmt.Sprintf("%v", value))
		}
	}

	// 清理未替换的占位符末尾逗号
	for {
		trimmed := strings.TrimRight(result, ",")
		if trimmed == result {
			break
		}
		result = strings.TrimSpace(trimmed)
	}

	return result
}
