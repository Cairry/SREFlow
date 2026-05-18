package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"

	"RCAgent/internal/models"
	"RCAgent/internal/repos"
	"RCAgent/internal/workflow/mcppool"
)

// MCPToolExecutor MCP工具执行器
type MCPToolExecutor struct {
	DefaultNodeExecutor
	mcpRepo repos.MCPServerRepoInter
}

// NewMCPToolExecutor 创建MCP工具执行器
func NewMCPToolExecutor(mcpRepo repos.MCPServerRepoInter) *MCPToolExecutor {
	return &MCPToolExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "MCP",
			Category:    "processing",
			Description: "Execute MCP (Model Context Protocol) tools",
		},
		mcpRepo: mcpRepo,
	}
}

func (e *MCPToolExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	serverID := getStr(nodeCtx.Config, "serverId", "")
	toolName := getStr(nodeCtx.Config, "toolName", "")
	arguments := getMap(nodeCtx.Config, "arguments")

	if serverID == "" {
		return errorResult("serverId is required"), nil
	}
	if toolName == "" {
		return errorResult("toolName is required"), nil
	}

	// Look up MCP server from database
	server, err := e.mcpRepo.GetByID(serverID)
	if err != nil {
		return errorResult(fmt.Sprintf("MCP server not found: %s", serverID)), nil
	}

	if !server.Enabled {
		return errorResult(fmt.Sprintf("MCP server %s is disabled", server.Name)), nil
	}

	// Parse headers
	var headers map[string]string
	if server.Headers != "" {
		if err := json.Unmarshal([]byte(server.Headers), &headers); err != nil {
			log.Printf("[MCPTool] Failed to parse headers for server %s: %v", serverID, err)
		}
	}

	// Get or create MCP client from pool
	client, err := mcppool.DefaultMCPPool.GetOrCreateClient(ctx, serverID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
	if err != nil {
		// Remove stale client and retry once
		mcppool.DefaultMCPPool.RemoveClient(serverID)
		client, err = mcppool.DefaultMCPPool.GetOrCreateClient(ctx, serverID, server.URL, server.Type, headers, server.APIKey, server.Timeout)
		if err != nil {
			return errorResult(fmt.Sprintf("failed to connect to MCP server %s: %v", server.Name, err)), nil
		}
	}

	log.Printf("[MCPTool] Calling %s/%s with args: %v", server.Name, toolName, arguments)

	// Call tool via MCP protocol
	result, err := client.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      toolName,
			Arguments: arguments,
		},
	})
	if err != nil {
		// Remove potentially stale client
		mcppool.DefaultMCPPool.RemoveClient(serverID)
		return errorResult(fmt.Sprintf("MCP tool call failed: %s/%s: %v", server.Name, toolName, err)), nil
	}

	// Extract content and try to parse text as JSON
	var outputsList []any
	for _, c := range result.Content {
		switch v := c.(type) {
		case mcp.TextContent:
			// Try to parse text content as JSON
			var parsedJSON map[string]any
			if err := json.Unmarshal([]byte(v.Text), &parsedJSON); err == nil {
				outputsList = append(outputsList, parsedJSON)
			} else {
				outputsList = append(outputsList, v.Text)
			}
		case mcp.ImageContent, mcp.AudioContent:
			// Non-text content, skip for output
		default:
			// Unknown content type, skip
		}
	}

	// Extract content from result
	outputs := map[string]any{
		"isError": result.IsError,
		"output":  outputsList,
	}

	return newResult(outputs), nil
}

// getMap extracts a map[string]any from config
func getMap(cfg map[string]any, key string) map[string]any {
	if v, ok := cfg[key]; ok {
		switch val := v.(type) {
		case map[string]any:
			return val
		case string:
			var m map[string]any
			if err := json.Unmarshal([]byte(val), &m); err == nil {
				return m
			}
		}
	}
	return map[string]any{}
}
