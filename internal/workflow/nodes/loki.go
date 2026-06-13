package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// LokiExecutor Loki日志查询执行器
type LokiExecutor struct {
	DefaultNodeExecutor
}

// NewLokiExecutor 创建Loki执行器
func NewLokiExecutor() *LokiExecutor {
	return &LokiExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Loki",
			Category:    "data",
			Description: "Query Loki logs using LogQL",
		},
	}
}

// Execute 执行Loki查询
func (e *LokiExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	lokiURL := getStr(nodeCtx.Config, "lokiUrl", "")
	if lokiURL == "" {
		return errorResult("lokiUrl is required"), nil
	}

	query := getStr(nodeCtx.Config, "query", "")
	limit := getInt(nodeCtx.Config, "limit", 1000)
	startTime := getStr(nodeCtx.Config, "startTime", "")
	endTime := getStr(nodeCtx.Config, "endTime", "")
	direction := getStr(nodeCtx.Config, "direction", "backward")
	timeout := getInt(nodeCtx.Config, "timeout", 30)

	if query == "" {
		return errorResult("query is required"), nil
	}

	logc.Infof(ctx, "[Loki] Querying %s: %s", lokiURL, query)

	// 构建查询参数
	params := url.Values{}
	params.Add("query", query)
	if limit > 0 {
		params.Add("limit", strconv.Itoa(limit))
	}
	if startTime != "" {
		params.Add("start", startTime)
	}
	if endTime != "" {
		params.Add("end", endTime)
	}
	if direction != "" {
		params.Add("direction", direction)
	}

	// 构建Loki API URL
	apiURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", strings.TrimSuffix(lokiURL, "/"), params.Encode())

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
	if headers, ok := nodeCtx.Config["headers"].(map[string]string); ok {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		logc.Infof(ctx, "[Loki] Added headers from config: %v", headers)
	}

	// 添加认证信息 - 优先使用从数据源合并的认证配置
	username := getStr(nodeCtx.Config, "authUsername", "")
	password := getStr(nodeCtx.Config, "authPassword", "")
	if username != "" {
		req.SetBasicAuth(username, password)
		logc.Infof(ctx, "[Loki] Added Basic authentication for user: %s", username)
	}

	// 添加Bearer token认证
	if authToken := getStr(nodeCtx.Config, "authToken", ""); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		logc.Infof(ctx, "[Loki] Added Bearer token authentication")
	}

	// 添加API Key认证
	if authAPIKey := getStr(nodeCtx.Config, "authAPIKey", ""); authAPIKey != "" {
		authAPIKeyName := getStr(nodeCtx.Config, "authAPIKeyName", "X-API-Key")
		authAPIKeyIn := getStr(nodeCtx.Config, "authAPIKeyIn", "header")
		if authAPIKeyIn == "header" {
			req.Header.Set(authAPIKeyName, authAPIKey)
			logc.Infof(ctx, "[Loki] Added API Key in header: %s", authAPIKeyName)
		} else {
			// Add to query parameters
			if q := req.URL.Query(); q != nil {
				q.Add(authAPIKeyName, authAPIKey)
				req.URL.RawQuery = q.Encode()
				logc.Infof(ctx, "[Loki] Added API Key in query parameter: %s", authAPIKeyName)
			}
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to query loki: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("loki returned status %d: %s", resp.StatusCode, string(body))), nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return errorResult(fmt.Sprintf("failed to decode response: %v", err)), nil
	}

	// 检查Loki响应状态
	if status, ok := result["status"].(string); !ok || status != "success" {
		errorMsg := "unknown error"
		if error, ok := result["error"].(string); ok {
			errorMsg = error
		}
		return errorResult(fmt.Sprintf("loki query failed: %s", errorMsg)), nil
	}

	data, ok := result["data"].(map[string]any)
	if !ok {
		return errorResult("invalid loki response: no data field"), nil
	}

	resultType, ok := data["resultType"].(string)
	if !ok {
		return errorResult("invalid loki response: no resultType field"), nil
	}

	series, ok := data["result"].([]any)
	if !ok {
		return errorResult("invalid loki response: no result field"), nil
	}

	logc.Infof(ctx, "[Loki] Query returned %d series (type: %s)", len(series), resultType)

	// 处理查询结果
	var processedSeries []map[string]any
	for _, s := range series {
		if seriesData, ok := s.(map[string]any); ok {
			processedSeries = append(processedSeries, seriesData)
		}
	}

	return newResult(map[string]any{
		"data": map[string]any{
			"result_type": resultType,
			"series":      processedSeries,
			"data":        result,
		},
		"query": query,
	}), nil
}
