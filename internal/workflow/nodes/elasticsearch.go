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

// ElasticsearchExecutor Elasticsearch查询执行器
type ElasticsearchExecutor struct {
	DefaultNodeExecutor
}

// NewElasticsearchExecutor 创建Elasticsearch执行器
func NewElasticsearchExecutor() *ElasticsearchExecutor {
	return &ElasticsearchExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Elasticsearch",
			Category:    "data",
			Description: "Query Elasticsearch indices and documents",
		},
	}
}

// Execute 执行Elasticsearch查询
func (e *ElasticsearchExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	elasticsearchURL := getStr(nodeCtx.Config, "elasticsearchUrl", "")
	if elasticsearchURL == "" {
		return errorResult("elasticsearchUrl is required"), nil
	}

	index := getStr(nodeCtx.Config, "index", "")
	queryType := getStr(nodeCtx.Config, "queryType", "search")
	query := getStr(nodeCtx.Config, "query", "")
	timeout := getInt(nodeCtx.Config, "timeout", 30)

	if index == "" {
		return errorResult("index is required"), nil
	}

	log.Printf("[Elasticsearch] Querying %s index: %s, type: %s", elasticsearchURL, index, queryType)

	// 构建Elasticsearch API URL
	var apiURL string
	var requestBody *bytes.Buffer
	var method string

	switch queryType {
	case "search":
		method = "POST"
		apiURL = fmt.Sprintf("%s/%s/_search", strings.TrimSuffix(elasticsearchURL, "/"), index)
		if query == "" {
			query = `{"query": {"match_all": {}}}`
		}
		requestBody = bytes.NewBufferString(query)
	case "count":
		method = "POST"
		apiURL = fmt.Sprintf("%s/%s/_count", strings.TrimSuffix(elasticsearchURL, "/"), index)
		if query == "" {
			query = `{"query": {"match_all": {}}}`
		}
		requestBody = bytes.NewBufferString(query)
	case "get":
		method = "GET"
		docID := getStr(nodeCtx.Config, "docId", "")
		if docID == "" {
			return errorResult("docId is required for get operation"), nil
		}
		apiURL = fmt.Sprintf("%s/%s/_doc/%s", strings.TrimSuffix(elasticsearchURL, "/"), index, docID)
		requestBody = nil
	default:
		return errorResult(fmt.Sprintf("unsupported queryType: %s", queryType)), nil
	}

	httpCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var req *http.Request
	var err error
	if requestBody != nil {
		req, err = http.NewRequestWithContext(httpCtx, method, apiURL, requestBody)
	} else {
		req, err = http.NewRequestWithContext(httpCtx, method, apiURL, nil)
	}

	if err != nil {
		return errorResult(fmt.Sprintf("failed to create request: %v", err)), nil
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
		log.Printf("[Elasticsearch] Added headers from config: %v", headers)
	}

	// 添加认证信息 - 优先使用从数据源合并的认证配置
	username := getStr(nodeCtx.Config, "authUsername", "")
	password := getStr(nodeCtx.Config, "authPassword", "")
	if username != "" {
		req.SetBasicAuth(username, password)
		log.Printf("[Elasticsearch] Added Basic authentication for user: %s", username)
	}

	// 添加Bearer token认证
	if authToken := getStr(nodeCtx.Config, "authToken", ""); authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
		log.Printf("[Elasticsearch] Added Bearer token authentication")
	}

	// 添加API Key认证
	if authAPIKey := getStr(nodeCtx.Config, "authAPIKey", ""); authAPIKey != "" {
		authAPIKeyName := getStr(nodeCtx.Config, "authAPIKeyName", "X-API-Key")
		authAPIKeyIn := getStr(nodeCtx.Config, "authAPIKeyIn", "header")
		if authAPIKeyIn == "header" {
			req.Header.Set(authAPIKeyName, authAPIKey)
			log.Printf("[Elasticsearch] Added API Key in header: %s", authAPIKeyName)
		} else {
			// Add to query parameters
			if q := req.URL.Query(); q != nil {
				q.Add(authAPIKeyName, authAPIKey)
				req.URL.RawQuery = q.Encode()
				log.Printf("[Elasticsearch] Added API Key in query parameter: %s", authAPIKeyName)
			}
		}
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to query elasticsearch: %v", err)), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return errorResult(fmt.Sprintf("elasticsearch returned status %d: %s", resp.StatusCode, string(body))), nil
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return errorResult(fmt.Sprintf("failed to decode response: %v", err)), nil
	}

	log.Printf("[Elasticsearch] Query executed successfully for index: %s", index)

	return newResult(map[string]any{
		"data": result,
	}), nil
}
