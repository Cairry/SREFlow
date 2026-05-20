package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Config struct {
	Name    string
	Version string
}

const (
	baseURL = "http://localhost:9001/api"
)

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func main() {
	transport := flag.String("transport", "streamable-http", "Transport type: sse or streamable-http")
	addr := flag.String("addr", ":8088", "Listen address")
	flag.Parse()

	mcpSrv := NewServer(Config{
		Name:    "SREFlow",
		Version: "1.0.0",
	})

	switch *transport {
	case "sse":
		sseServer := server.NewSSEServer(mcpSrv)
		log.Printf("[MCP] SSE server listening on %s", *addr)
		log.Fatal(http.ListenAndServe(*addr, sseServer))
	default:
		httpServer := server.NewStreamableHTTPServer(mcpSrv)
		log.Printf("[MCP] StreamableHTTP server listening on %s", *addr)
		log.Fatal(http.ListenAndServe(*addr, httpServer))
	}
}

func NewServer(cfg Config) *server.MCPServer {
	s := server.NewMCPServer(cfg.Name, cfg.Version,
		server.WithRecovery(),
	)

	s.AddTool(mcp.NewTool("workflow_list",
		mcp.WithDescription("List all workflows in a workspace"),
		mcp.WithString("workspaceId", mcp.Required(), mcp.Description("Workspace ID")),
	), listWorkflowsHandler())

	s.AddTool(mcp.NewTool("workflow_execute",
		mcp.WithDescription("Execute a workflow by ID"),
		mcp.WithString("workflowId", mcp.Required(), mcp.Description("Workflow ID to execute")),
		mcp.WithString("input", mcp.Description("Input text for the Start node")),
	), executeWorkflowHandler())

	s.AddTool(mcp.NewTool("workflow_status",
		mcp.WithDescription("Get the execution status of a workflow run"),
		mcp.WithString("executionId", mcp.Required(), mcp.Description("Execution ID")),
	), executionStatusHandler())

	s.AddTool(mcp.NewTool("workflow_result",
		mcp.WithDescription("Get the result of a completed workflow execution"),
		mcp.WithString("executionId", mcp.Required(), mcp.Description("Execution ID")),
	), executionResultHandler())

	return s
}

func listWorkflowsHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workspaceId, err := req.RequireString("workspaceId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		httpReq, err := http.NewRequest("GET", baseURL+"/workflows", nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
		}
		httpReq.Header.Set("X-Workspace-Id", workspaceId)

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to call API", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return mcp.NewToolResultError(string(body)), nil
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read response", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func executeWorkflowHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		workflowId, err := req.RequireString("workflowId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		input := req.GetString("input", "")

		body := map[string]interface{}{
			"workflowId": workflowId,
			"input":      input,
		}
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to marshal request", err), nil
		}

		httpReq, err := http.NewRequest("POST", baseURL+"/workflow/"+workflowId+"/execute", bytes.NewReader(bodyBytes))
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
		}
		httpReq.Header.Set("Content-Type", "application/json")

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to call API", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return mcp.NewToolResultError(string(respBody)), nil
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read response", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func executionStatusHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionId, err := req.RequireString("executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		httpReq, err := http.NewRequest("GET", baseURL+"/workflow/executions/"+executionId+"/status", nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
		}

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to call API", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return mcp.NewToolResultError(string(body)), nil
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read response", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}

func executionResultHandler() server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		executionId, err := req.RequireString("executionId")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		httpReq, err := http.NewRequest("GET", baseURL+"/workflow/executions/"+executionId, nil)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to create request", err), nil
		}

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to call API", err), nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return mcp.NewToolResultError(string(body)), nil
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return mcp.NewToolResultErrorFromErr("failed to read response", err), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	}
}
