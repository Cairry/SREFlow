package nodes

import (
	"SREFlow/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
)

// PrometheusNodeExecutor Prometheus节点执行器包装器
type PrometheusNodeExecutor struct {
	DefaultNodeExecutor
}

func NewPrometheusExecutor() *PrometheusNodeExecutor {
	return &PrometheusNodeExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Prometheus",
			Category:    "data",
			Description: "Query Prometheus metrics using PromQL",
		},
	}
}

func (e *PrometheusNodeExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	return ExecutePrometheus(ctx, nodeCtx)
}

func ExecutePrometheus(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	prometheusUrl := getStr(nc.Config, "prometheusUrl", "")
	query := getStr(nc.Config, "query", "")
	timeout := getInt(nc.Config, "timeout", 30)

	// 验证必需配置
	if prometheusUrl == "" {
		return errorResult("prometheusUrl is required - please configure a Prometheus integration or set prometheusUrl explicitly"), nil
	}

	logc.Infof(ctx, "[Prometheus] Querying %s with: %s", prometheusUrl, query)

	// 验证查询不为空
	if query == "" {
		return errorResult("query cannot be empty"), nil
	}

	// URL编码查询参数
	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("%s/api/v1/query?query=%s", prometheusUrl, encodedQuery)

	httpCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "GET", url, nil)
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
		logc.Infof(ctx, "[Prometheus] Added headers from config: %v", headers)
	}

	// 添加认证信息
	if authToken := getStr(nc.Config, "authToken", ""); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		logc.Infof(ctx, "[Prometheus] Added Bearer token authentication")
	}
	if authUsername := getStr(nc.Config, "authUsername", ""); authUsername != "" {
		authPassword := getStr(nc.Config, "authPassword", "")
		req.SetBasicAuth(authUsername, authPassword)
		logc.Infof(ctx, "[Prometheus] Added Basic authentication for user: %s", authUsername)
	}
	if authAPIKey := getStr(nc.Config, "authAPIKey", ""); authAPIKey != "" {
		authAPIKeyName := getStr(nc.Config, "authAPIKeyName", "X-API-Key")
		authAPIKeyIn := getStr(nc.Config, "authAPIKeyIn", "header")
		if authAPIKeyIn == "header" {
			req.Header.Set(authAPIKeyName, authAPIKey)
			logc.Infof(ctx, "[Prometheus] Added API Key in header: %s", authAPIKeyName)
		} else {
			// Add to query parameters
			if q := req.URL.Query(); q != nil {
				q.Add(authAPIKeyName, authAPIKey)
				req.URL.RawQuery = q.Encode()
				logc.Infof(ctx, "[Prometheus] Added API Key in query parameter: %s", authAPIKeyName)
			}
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to query prometheus: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("prometheus returned status %d, body: %s, query: %s", resp.StatusCode, string(body), encodedQuery)), nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return errorResult(fmt.Sprintf("failed to decode response: %v", err)), nil
	}

	// 检查Prometheus响应状态
	if status, ok := result["status"].(string); !ok || status != "success" {
		errorMsg := "unknown error"
		if error, ok := result["error"].(string); ok {
			errorMsg = error
		}
		return errorResult(fmt.Sprintf("prometheus query failed: %s", errorMsg)), nil
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return errorResult("invalid prometheus response: no data field"), nil
	}

	resultType, ok := data["resultType"].(string)
	if !ok {
		return errorResult("invalid prometheus response: no resultType field"), nil
	}

	series, ok := data["result"].([]any)
	if !ok {
		return errorResult("invalid prometheus response: no result field"), nil
	}

	logc.Infof(ctx, "[Prometheus] Query returned %d series (type: %s)", len(series), resultType)

	return newResult(map[string]any{
		"promQL": query,
		"series": series,
	}), nil
}
