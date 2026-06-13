package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// ClickHouseExecutor ClickHouse SQL执行器
type ClickHouseExecutor struct {
	DefaultNodeExecutor
}

// NewClickHouseExecutor 创建ClickHouse执行器
func NewClickHouseExecutor() *ClickHouseExecutor {
	return &ClickHouseExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "ClickHouse",
			Category:    "data",
			Description: "Execute ClickHouse SQL queries",
		},
	}
}

// Execute 执行ClickHouse查询
func (e *ClickHouseExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	clickhouseURL := getStr(nodeCtx.Config, "clickhouseUrl", "")
	if clickhouseURL == "" {
		return errorResult("clickhouseUrl is required"), nil
	}

	database := getStr(nodeCtx.Config, "database", "default")
	username := getStr(nodeCtx.Config, "username", "default")
	password := getStr(nodeCtx.Config, "password", "")
	sqlQuery := getStr(nodeCtx.Config, "query", "")
	timeout := getInt(nodeCtx.Config, "timeout", 30)

	if sqlQuery == "" {
		return errorResult("query is required"), nil
	}

	logc.Infof(ctx, "[ClickHouse] Querying %s database %s: %s", clickhouseURL, database, sqlQuery)

	// 构建ClickHouse HTTP API URL
	apiURL := fmt.Sprintf("%s", strings.TrimSuffix(clickhouseURL, "/"))

	// 准备查询参数
	params := url.Values{}
	if database != "" {
		params.Add("database", database)
	}
	params.Add("query", sqlQuery)

	// 构建完整URL
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())

	httpCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, "POST", fullURL, nil)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create request: %v", err)), nil
	}

	// 设置认证 - 优先使用从数据源合并的认证配置
	authUsername := getStr(nodeCtx.Config, "authUsername", username)
	authPassword := getStr(nodeCtx.Config, "authPassword", password)

	if authUsername != "" {
		if authPassword != "" {
			req.SetBasicAuth(authUsername, authPassword)
		} else {
			req.SetBasicAuth(authUsername, "")
		}
		logc.Infof(ctx, "[ClickHouse] Using authentication for user: %s", authUsername)
	}

	// 设置请求头
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SREFlow/1.0")

	// 添加数据源的 headers
	if headers, ok := nodeCtx.Config["headers"].(map[string]string); ok {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		logc.Infof(ctx, "[ClickHouse] Added headers from config: %v", headers)
	}

	// 添加Bearer token认证
	if authToken := getStr(nodeCtx.Config, "authToken", ""); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		logc.Infof(ctx, "[ClickHouse] Added Bearer token authentication")
	}

	// 添加API Key认证
	if authAPIKey := getStr(nodeCtx.Config, "authAPIKey", ""); authAPIKey != "" {
		authAPIKeyName := getStr(nodeCtx.Config, "authAPIKeyName", "X-API-Key")
		authAPIKeyIn := getStr(nodeCtx.Config, "authAPIKeyIn", "header")
		if authAPIKeyIn == "header" {
			req.Header.Set(authAPIKeyName, authAPIKey)
			logc.Infof(ctx, "[ClickHouse] Added API Key in header: %s", authAPIKeyName)
		} else {
			// Add to query parameters
			if q := req.URL.Query(); q != nil {
				q.Add(authAPIKeyName, authAPIKey)
				req.URL.RawQuery = q.Encode()
				logc.Infof(ctx, "[ClickHouse] Added API Key in query parameter: %s", authAPIKeyName)
			}
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to query clickhouse: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("clickhouse returned status %d: %s", resp.StatusCode, string(body))), nil
	}

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to read response: %v", err)), nil
	}

	// 尝试解析JSON响应
	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// 如果不是JSON，返回原始文本
		result = string(body)
	}

	logc.Infof(ctx, "[ClickHouse] Query executed successfully, response size: %d bytes", len(body))

	return newResult(map[string]any{
		"data": result,
	}), nil
}
