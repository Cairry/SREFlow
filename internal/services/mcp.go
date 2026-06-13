package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/zeromicro/go-zero/core/logc"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/internal/types"
	"SREFlow/internal/workflow/mcppool"
	"SREFlow/pkg/tools"
)

// MCPService MCP服务器服务
type MCPService struct {
	repo repos.MCPServerRepoInter
}

func (s *MCPService) Init(repo repos.MCPServerRepoInter) {
	s.repo = repo
}

// List 获取MCP服务器列表
func (s *MCPService) List(c context.Context, workspaceID string) (interface{}, error) {
	servers, err := s.repo.GetByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*types.ResponseMCPServer, 0, len(servers))
	for _, srv := range servers {
		result = append(result, s.toResponse(srv))
	}

	return map[string]interface{}{
		"servers": result,
	}, nil
}

// Get 获取单个MCP服务器
func (s *MCPService) Get(c context.Context, id string) (interface{}, error) {
	server, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"server": s.toResponse(server),
	}, nil
}

// Create 创建MCP服务器
func (s *MCPService) Create(c context.Context, req *types.RequestCreateMCPServer, workspaceID string) (interface{}, error) {
	server := &models.MCPServer{
		ID:          tools.RandId(),
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Type:        req.Type,
		URL:         req.URL,
		APIKey:      req.APIKey,
		Timeout:     req.Timeout,
		Enabled:     true,
		Description: req.Description,
	}

	if req.Enabled != nil {
		server.Enabled = *req.Enabled
	}

	if len(req.Headers) > 0 {
		headersJSON, err := json.Marshal(req.Headers)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal headers: %w", err)
		}
		server.Headers = string(headersJSON)
	}

	if err := s.repo.Create(server); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"server": s.toResponse(server),
	}, nil
}

// Update 更新MCP服务器
func (s *MCPService) Update(c context.Context, req *types.RequestUpdateMCPServer) (interface{}, error) {
	server, err := s.repo.GetByID(req.ID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		server.Name = req.Name
	}
	if req.Type != nil {
		server.Type = *req.Type
	}
	if req.URL != "" {
		server.URL = req.URL
	}
	if req.APIKey != "" {
		server.APIKey = req.APIKey
	}
	if req.Timeout != nil {
		server.Timeout = *req.Timeout
	}
	if req.Enabled != nil {
		server.Enabled = *req.Enabled
	}
	if req.Description != "" {
		server.Description = req.Description
	}
	if req.Headers != nil {
		headersJSON, err := json.Marshal(req.Headers)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal headers: %w", err)
		}
		server.Headers = string(headersJSON)
	}

	// Remove stale client from pool when config changes
	mcppool.DefaultMCPPool.RemoveClient(req.ID)

	if err := s.repo.Update(server); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"server": s.toResponse(server),
	}, nil
}

// Delete 删除MCP服务器
func (s *MCPService) Delete(c context.Context, id string) (interface{}, error) {
	// Remove client from pool
	mcppool.DefaultMCPPool.RemoveClient(id)

	if err := s.repo.Delete(id); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "MCP server deleted successfully",
	}, nil
}

// ListTools 获取MCP服务器的工具列表 — 通过MCP协议
func (s *MCPService) ListTools(c context.Context, id string) (interface{}, error) {
	server, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	logc.Infof(c, "[MCPService] ListTools: serverID=%s, name=%s, type=%s, url=%s, timeout=%d, enabled=%v",
		server.ID, server.Name, server.Type, server.URL, server.Timeout, server.Enabled)

	if !server.Enabled {
		return nil, fmt.Errorf("server is disabled")
	}

	// Quick HTTP health-check to verify the server URL is reachable before creating MCP client
	if err := s.probeEndpoint(server.URL); err != nil {
		logc.Errorf(c, "[MCPService] Health check failed for %s: %v", server.URL, err)
		return nil, fmt.Errorf("MCP server unreachable at %s: %w", server.URL, err)
	}
	logc.Infof(c, "[MCPService] Health check passed for %s", server.URL)

	headers := s.parseHeaders(server)

	client, err := mcppool.DefaultMCPPool.GetOrCreateClient(c, id, server.URL, server.Type, headers, server.APIKey, server.Timeout)
	if err != nil {
		mcppool.DefaultMCPPool.RemoveClient(id)
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	result, err := client.ListTools(c, mcp.ListToolsRequest{})
	if err != nil {
		logc.Errorf(c, "[MCPService] ListTools failed for %s (%s): %v, removing stale session and retrying...", server.Name, id, err)
		mcppool.DefaultMCPPool.RemoveClient(id)
		client, err = mcppool.DefaultMCPPool.GetOrCreateClient(c, id, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to reconnect to MCP server: %w", err)
		}
		result, err = client.ListTools(c, mcp.ListToolsRequest{})
		if err != nil {
			mcppool.DefaultMCPPool.RemoveClient(id)
			return nil, fmt.Errorf("failed to list tools after retry: %w", err)
		}
	}

	tools := make([]map[string]interface{}, 0, len(result.Tools))
	for _, tool := range result.Tools {
		t := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
		}
		if tool.InputSchema.Type != "" || len(tool.InputSchema.Properties) > 0 {
			schemaBytes, err := json.Marshal(tool.InputSchema)
			if err == nil {
				var schema map[string]interface{}
				if json.Unmarshal(schemaBytes, &schema) == nil {
					t["input_schema"] = schema
				}
			}
		} else if tool.RawInputSchema != nil {
			var schema map[string]interface{}
			if json.Unmarshal(tool.RawInputSchema, &schema) == nil {
				t["input_schema"] = schema
			}
		}
		tools = append(tools, t)
	}

	return map[string]interface{}{
		"tools": tools,
	}, nil
}

// CallTool 调用MCP服务器上的工具 — 通过MCP协议
func (s *MCPService) CallTool(c context.Context, id string, toolName string, args map[string]interface{}) (interface{}, error) {
	server, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	if !server.Enabled {
		return nil, fmt.Errorf("server is disabled")
	}

	headers := s.parseHeaders(server)

	client, err := mcppool.DefaultMCPPool.GetOrCreateClient(c, id, server.URL, server.Type, headers, server.APIKey, server.Timeout)
	if err != nil {
		mcppool.DefaultMCPPool.RemoveClient(id)
		return nil, fmt.Errorf("failed to connect to MCP server: %w", err)
	}

	logc.Infof(c, "[MCPService] Calling tool %s on server %s", toolName, server.Name)

	result, err := client.CallTool(c, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: args,
		},
	})
	if err != nil {
		logc.Errorf(c, "[MCPService] CallTool failed for %s on %s (%s): %v, removing stale session and retrying...", toolName, server.Name, id, err)
		mcppool.DefaultMCPPool.RemoveClient(id)
		client, err = mcppool.DefaultMCPPool.GetOrCreateClient(c, id, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			return nil, fmt.Errorf("failed to reconnect to MCP server: %w", err)
		}
		result, err = client.CallTool(c, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name:      toolName,
				Arguments: args,
			},
		})
		if err != nil {
			mcppool.DefaultMCPPool.RemoveClient(id)
			return nil, fmt.Errorf("failed to call tool %s after retry: %w", toolName, err)
		}
	}

	// Build response from CallToolResult
	response := map[string]interface{}{
		"isError": result.IsError,
	}

	var contents []map[string]interface{}
	for _, content := range result.Content {
		cm := map[string]interface{}{}
		switch v := content.(type) {
		case mcp.TextContent:
			cm["type"] = v.Type
			cm["text"] = v.Text
		case mcp.ImageContent:
			cm["type"] = v.Type
			cm["mimeType"] = v.MIMEType
		case mcp.AudioContent:
			cm["type"] = v.Type
			cm["mimeType"] = v.MIMEType
		default:
			cm["type"] = "unknown"
		}
		contents = append(contents, cm)
	}
	response["content"] = contents

	return response, nil
}

// GetStatus 获取MCP服务器状态 — 通过MCP协议 Ping
func (s *MCPService) GetStatus(c context.Context, id string) (interface{}, error) {
	server, err := s.repo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("server not found: %w", err)
	}

	if !server.Enabled {
		return map[string]interface{}{
			"connected": false,
			"message":   "server is disabled",
		}, nil
	}

	headers := s.parseHeaders(server)

	client, err := mcppool.DefaultMCPPool.GetOrCreateClient(c, id, server.URL, server.Type, headers, server.APIKey, server.Timeout)
	if err != nil {
		return map[string]interface{}{
			"connected": false,
			"message":   err.Error(),
		}, nil
	}

	if err := client.Ping(c); err != nil {
		logc.Errorf(c, "[MCPService] Ping failed for %s (%s): %v, removing stale session and retrying...", server.Name, id, err)
		mcppool.DefaultMCPPool.RemoveClient(id)
		client, err = mcppool.DefaultMCPPool.GetOrCreateClient(c, id, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			return map[string]interface{}{
				"connected": false,
				"message":   err.Error(),
			}, nil
		}
		if err := client.Ping(c); err != nil {
			mcppool.DefaultMCPPool.RemoveClient(id)
			return map[string]interface{}{
				"connected": false,
				"message":   err.Error(),
			}, nil
		}
	}

	return map[string]interface{}{
		"connected": true,
	}, nil
}

// parseHeaders parses the JSON headers string from the server model
func (s *MCPService) parseHeaders(server *models.MCPServer) map[string]string {
	if server.Headers == "" {
		return nil
	}
	var headers map[string]string
	if err := json.Unmarshal([]byte(server.Headers), &headers); err != nil {
		return nil
	}
	return headers
}

// toResponse 转换为响应结构
func (s *MCPService) toResponse(server *models.MCPServer) *types.ResponseMCPServer {
	resp := &types.ResponseMCPServer{
		ID:          server.ID,
		WorkspaceID: server.WorkspaceID,
		Name:        server.Name,
		Type:        server.Type,
		URL:         server.URL,
		APIKey:      server.APIKey,
		Timeout:     server.Timeout,
		Enabled:     server.Enabled,
		Description: server.Description,
		CreatedAt:   server.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   server.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if server.Headers != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(server.Headers), &headers); err == nil {
			resp.Headers = headers
		}
	}

	return resp
}

// probeEndpoint does a quick HTTP GET to verify the MCP server URL is reachable.
// This catches DNS errors, connection refused, etc. before the slow MCP handshake times out.
func (s *MCPService) probeEndpoint(url string) error {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("HTTP probe failed: %w", err)
	}
	resp.Body.Close()
	logc.Infof(context.Background(), "[MCPService] Probe endpoint %s returned HTTP %d", url, resp.StatusCode)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP probe returned status %d — the MCP SSE endpoint URL may be incorrect (SSE servers typically require a path such as /sse)", resp.StatusCode)
	}
	return nil
}
