package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"SREFlow/internal/models"
)

// JaegerExecutor Jaeger追踪执行器
type JaegerExecutor struct {
	DefaultNodeExecutor
}

// NewJaegerExecutor 创建Jaeger执行器
func NewJaegerExecutor() *JaegerExecutor {
	return &JaegerExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Jaeger",
			Category:    "data",
			Description: "Query distributed traces from Jaeger",
		},
	}
}

// Execute 执行Jaeger查询
func (e *JaegerExecutor) Execute(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	jaegerURL := getStr(nc.Config, "jaegerUrl", "")
	if jaegerURL == "" {
		return errorResult("jaegerUrl is required"), nil
	}

	service := getStr(nc.Config, "service", "")
	tags := getStr(nc.Config, "tags", "")
	limit := getInt(nc.Config, "limit", 100)
	startTime := getStr(nc.Config, "startTime", "")
	endTime := getStr(nc.Config, "endTime", "")
	timeout := getInt(nc.Config, "timeout", 30)

	log.Printf("[Jaeger] Querying %s for service: %s, tags: %s", jaegerURL, service, tags)

	// 构建查询参数
	params := url.Values{}
	if service != "" {
		params.Add("service", service)
	}
	if tags != "" {
		params.Add("tags", tags)
	}
	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}
	if startTime != "" {
		params.Add("start", startTime)
	}
	if endTime != "" {
		params.Add("end", endTime)
	}

	// 构建API URL
	apiURL := fmt.Sprintf("%s/api/traces?%s", strings.TrimSuffix(jaegerURL, "/"), params.Encode())

	httpCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "GET", apiURL, nil)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "SREFlow/1.0")

	// 添加数据源的 headers
	if headers, ok := nc.Config["headers"].(map[string]string); ok {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		log.Printf("[Jaeger] Added headers from config: %v", headers)
	}

	// 添加认证信息
	if authToken := getStr(nc.Config, "authToken", ""); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		log.Printf("[Jaeger] Added Bearer token authentication")
	}
	if authUsername := getStr(nc.Config, "authUsername", ""); authUsername != "" {
		authPassword := getStr(nc.Config, "authPassword", "")
		req.SetBasicAuth(authUsername, authPassword)
		log.Printf("[Jaeger] Added Basic authentication for user: %s", authUsername)
	}
	if authAPIKey := getStr(nc.Config, "authAPIKey", ""); authAPIKey != "" {
		authAPIKeyName := getStr(nc.Config, "authAPIKeyName", "X-API-Key")
		authAPIKeyIn := getStr(nc.Config, "authAPIKeyIn", "header")
		if authAPIKeyIn == "header" {
			req.Header.Set(authAPIKeyName, authAPIKey)
			log.Printf("[Jaeger] Added API Key in header: %s", authAPIKeyName)
		} else {
			// Add to query parameters
			if q := req.URL.Query(); q != nil {
				q.Add(authAPIKeyName, authAPIKey)
				req.URL.RawQuery = q.Encode()
				log.Printf("[Jaeger] Added API Key in query parameter: %s", authAPIKeyName)
			}
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to query jaeger: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("jaeger returned status %d: %s", resp.StatusCode, string(body))), nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return errorResult(fmt.Sprintf("failed to decode response: %v", err)), nil
	}

	// 处理追踪数据
	traces, ok := result["data"].([]any)
	if !ok {
		return errorResult("invalid jaeger response: no data field"), nil
	}

	log.Printf("[Jaeger] Query returned %d traces", len(traces))

	// 处理追踪结果
	var processedTraces []map[string]any
	for _, trace := range traces {
		if traceData, ok := trace.(map[string]any); ok {
			processedTraces = append(processedTraces, traceData)
		}
	}

	return newResult(map[string]any{
		"traces":       processedTraces,
		"trace_count":  len(processedTraces),
		"raw_response": result,
	}), nil
}
