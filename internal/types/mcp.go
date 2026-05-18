package types

import (
	"fmt"
	"RCAgent/internal/models"
)

// RequestCreateMCPServer 创建MCP服务器请求
type RequestCreateMCPServer struct {
	Name        string               `json:"name"`
	Type        models.MCPServerType `json:"type"`
	URL         string               `json:"url"`
	Headers     map[string]string    `json:"headers,omitempty"`
	APIKey      string               `json:"apiKey,omitempty"`
	Timeout     int                  `json:"timeout,omitempty"`
	Enabled     *bool                `json:"enabled,omitempty"`
	Description string               `json:"description,omitempty"`
}

func (r *RequestCreateMCPServer) Valid() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("name is too long (max 100 characters)")
	}
	if r.URL == "" {
		return fmt.Errorf("url is required")
	}
	if r.Type != "" && !r.Type.IsValid() {
		return fmt.Errorf("invalid server type: %s, valid types are: http, streamableHttp, sse", r.Type)
	}
	if r.Type == "" {
		r.Type = models.MCPServerTypeHTTP
	}
	if r.Timeout <= 0 {
		r.Timeout = 30
	}
	return nil
}

// RequestUpdateMCPServer 更新MCP服务器请求
type RequestUpdateMCPServer struct {
	ID          string                `json:"id"`
	Name        string                `json:"name,omitempty"`
	Type        *models.MCPServerType `json:"type,omitempty"`
	URL         string                `json:"url,omitempty"`
	Headers     map[string]string     `json:"headers,omitempty"`
	APIKey      string                `json:"apiKey,omitempty"`
	Timeout     *int                  `json:"timeout,omitempty"`
	Enabled     *bool                 `json:"enabled,omitempty"`
	Description string                `json:"description,omitempty"`
}

func (r *RequestUpdateMCPServer) Valid() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Name != "" && len(r.Name) > 100 {
		return fmt.Errorf("name is too long (max 100 characters)")
	}
	if r.Type != nil && !r.Type.IsValid() {
		return fmt.Errorf("invalid server type: %s", *r.Type)
	}
	return nil
}

// ResponseMCPServer MCP服务器响应
type ResponseMCPServer struct {
	ID          string               `json:"id"`
	WorkspaceID string               `json:"workspaceId"`
	Name        string               `json:"name"`
	Type        models.MCPServerType `json:"type"`
	URL         string               `json:"url"`
	Headers     map[string]string    `json:"headers,omitempty"`
	APIKey      string               `json:"apiKey,omitempty"`
	Timeout     int                  `json:"timeout"`
	Enabled     bool                 `json:"enabled"`
	Description string               `json:"description"`
	CreatedAt   string               `json:"createdAt"`
	UpdatedAt   string               `json:"updatedAt"`
}
