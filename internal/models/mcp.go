package models

import (
	"time"

	"gorm.io/gorm"
)

// MCPServerType MCP服务器类型
type MCPServerType string

const (
	MCPServerTypeHTTP           MCPServerType = "http"
	MCPServerTypeStreamableHTTP MCPServerType = "streamableHttp"
	MCPServerTypeSSE            MCPServerType = "sse"
)

// MCPServer MCP服务器配置模型
type MCPServer struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	WorkspaceID string         `json:"workspaceId" gorm:"type:varchar(36);index"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Type        MCPServerType  `json:"type" gorm:"type:varchar(50);not null;default:'http'"`
	URL         string         `json:"url" gorm:"type:text;not null"`
	Headers     string         `json:"headers,omitempty" gorm:"type:text"`
	APIKey      string         `json:"apiKey,omitempty" gorm:"type:varchar(500)"`
	Timeout     int            `json:"timeout" gorm:"type:int;default:30"`
	Enabled     bool           `json:"enabled" gorm:"type:bool;default:true"`
	Description string         `json:"description" gorm:"type:text"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

func (MCPServer) TableName() string {
	return "v_mcp_server"
}

func (t MCPServerType) IsValid() bool {
	switch t {
	case MCPServerTypeHTTP, MCPServerTypeStreamableHTTP, MCPServerTypeSSE:
		return true
	default:
		return false
	}
}
