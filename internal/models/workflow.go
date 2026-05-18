package models

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// Workflow represents a workflow template
type Workflow struct {
	ID                string         `json:"id" gorm:"primaryKey"`
	WorkspaceID       string         `json:"workspaceId" gorm:"type:varchar(36);index"`
	Name              string         `json:"name" gorm:"not null"`
	Description       string         `json:"description"`
	Nodes             string         `json:"nodes"` // JSON array as string
	Edges             string         `json:"edges"` // JSON array as string
	CurrentVersionNum int            `json:"currentVersionNum" gorm:"default:1"`   // 当前版本号
	CurrentContentHash string        `json:"currentContentHash" gorm:"type:varchar(64)"` // 当前内容哈希
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	DeletedAt         gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName specifies the table name for Workflow model
func (Workflow) TableName() string {
	return "workflows"
}

// Port defines an input or output port for a node type
type Port struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"` // "string", "object", "number", "array", "any"
}

// FieldSchema defines a config field for a node
type FieldSchema struct {
	Key      string      `json:"key"`
	Label    string      `json:"label"`
	Type     string      `json:"type"` // "string", "number", "select", "bool"
	Default  interface{} `json:"default"`
	Required bool        `json:"required"`
	Options  []string    `json:"options,omitempty"` // for select type
}

// NodeDefinition is the static definition of a node type
type NodeDefinition struct {
	Type        string        `json:"type"`
	Label       string        `json:"label"`
	Category    string        `json:"category"` // "data", "processing", "output", "logic"
	Description string        `json:"description"`
	Inputs      []Port        `json:"inputs"`
	Outputs     []Port        `json:"outputs"`
	Config      []FieldSchema `json:"config"`
	// Runtime-only fields (not stored in DB)
	Executor NodeExecutor `json:"-"`
}

// NodeExecutor is the function signature for executing a node
type NodeExecutor func(ctx context.Context, nc *NodeContext) (*NodeResult, error)

// NodeContext is the runtime context passed to a node executor
type NodeContext struct {
	NodeID    string         `json:"nodeId"`
	NodeType  string         `json:"nodeType"`
	Config    map[string]any `json:"config"`    // node-specific config values
	Input     map[string]any `json:"input"`     // input values from connected upstream nodes
	Variables map[string]any `json:"variables"` // shared workflow variables
}

// NodeResult is the output from a node executor
type NodeResult struct {
	Outputs map[string]any `json:"outputs"` // outputs mapped to port names
	Error   string         `json:"error,omitempty"`
	Next    []string       `json:"next,omitempty"` // override next node IDs (for conditionals)
}

// WorkflowExecution represents a single workflow run
type WorkflowExecution struct {
	ID                 string                 `json:"id"`
	WorkflowID         string                 `json:"workflowId"`
	WorkflowName       string                 `json:"workflowName"`
	Status             string                 `json:"status"` // "running", "completed", "failed"
	StartedAt          int64                  `json:"startedAt"`
	FinishedAt         int64                  `json:"finishedAt"`
	NodeResults        map[string]*NodeResult `json:"nodeResults"`    // nodeId -> result (内部使用)
	RestructuredOutput map[string]map[string]string `json:"outputs"`   // nodeId -> field -> string
	Variables          map[string]any         `json:"variables"`      // final workflow variables
	RunningNodes       []string               `json:"runningNodes"`   // 当前正在运行的节点ID
	CompletedNodes     []string               `json:"completedNodes"` // 已完成的节点ID
	FailedNodes        []string               `json:"failedNodes"`    // 失败的节点ID
}

// WorkflowExecutionRecord 用于数据库存储的执行记录
type WorkflowExecutionRecord struct {
	ID           string `gorm:"primaryKey;type=varchar(36)" json:"id"`
	WorkflowID   string `gorm:"type=varchar(36);index" json:"workflowId"`
	WorkflowName string `gorm:"type=varchar(255)" json:"workflowName"`
	Status       string `gorm:"type=varchar(20)" json:"status"`
	StartedAt    int64  `gorm:"index" json:"startedAt"`
	FinishedAt   int64  `json:"finishedAt"`
	NodeResults  string `gorm:"type=text" json:"nodeResults"` // JSON string
	Variables    string `gorm:"type=text" json:"variables"`   // JSON string
	FinalOutput  string `gorm:"type=text" json:"finalOutput"`
	CreatedAt    int64  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    int64  `gorm:"autoUpdateTime" json:"updatedAt"`
}

// WorkflowVersion stores historical versions of a workflow when it is updated
type WorkflowVersion struct {
	Version     int            `gorm:"primaryKey;autoIncrement:false" json:"version"` // 版本号：1,2,3...（与 WorkflowID 组成复合主键）
	WorkflowID  string         `gorm:"type=varchar(36);primaryKey;index" json:"workflowId"`
	Name        string         `gorm:"type=varchar(255)" json:"name"`
	Description string         `gorm:"type=text" json:"description"`
	Nodes       string         `gorm:"type=text" json:"nodes"`
	Edges       string         `gorm:"type=text" json:"edges"`
	Hash        string         `gorm:"type=varchar(64);index" json:"hash"` // SHA256 of nodes+edges
	IsRelease   bool           `gorm:"default:false" json:"isRelease"`     // 是否为 Release 版本
	CreatedAt   time.Time      `json:"createdAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName specifies the table name for WorkflowVersion model
func (WorkflowVersion) TableName() string {
	return "workflow_versions"
}
