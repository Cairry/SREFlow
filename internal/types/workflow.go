package types

import (
	"context"
	"encoding/json"
	"fmt"

	"SREFlow/internal/cache"
	"SREFlow/internal/ctx"
	"SREFlow/internal/models"
	"SREFlow/internal/repos"
)

// RequestWorkflowExecute 请求执行工作流
type RequestWorkflowExecute struct {
	TenantId   string `json:"tenantId"`
	WorkflowId string `json:"workflowId"`
	Input      string `json:"input"` // Start 节点 text 参数
}

func (execute *RequestWorkflowExecute) Valid(ctx context.Context, database repos.EntryInter, cache cache.EntryInter) error {
	if execute.WorkflowId == "" {
		return fmt.Errorf("workflow ID is required")
	}

	// 验证工作流是否存在，并检查 Start 节点的 text 参数
	var workflow models.Workflow
	db := database.MySQL()
	if err := db.Where("id = ?", execute.WorkflowId).First(&workflow).Error; err != nil {
		return fmt.Errorf("workflow not found")
	}

	// 解析工作流节点，检查 Start 节点的 text 是否为空
	if execute.Input == "" {
		// input 参数未传递，检查数据库中 Start 节点的 text 是否配置
		var nodes []map[string]interface{}
		if nodesStr := workflow.Nodes; nodesStr != "" {
			if err := json.Unmarshal([]byte(nodesStr), &nodes); err == nil {
				for _, node := range nodes {
					if nodeType, ok := node["type"].(string); ok && nodeType == "Start" {
						if data, ok := node["data"].(map[string]interface{}); ok {
							if text, ok := data["text"].(string); ok && text != "" {
								// 数据库中 Start 节点已配置 text，可以使用
								return nil
							}
						}
					}
				}
			}
		}
		// input 未传递且数据库中 Start 节点 text 也为空
		return fmt.Errorf("input parameter is required: Start node text is empty, please provide input parameter")
	}

	return nil
}

// RequestWorkflowValidate 请求验证工作流
type RequestWorkflowValidate struct {
	TenantId string        `json:"tenantId"`
	Nodes    []models.Node `json:"nodes"`
	Edges    []models.Edge `json:"edges"`
}

// ResponseWorkflowExecute 响应工作流执行结果
type ResponseWorkflowExecute struct {
	models.WorkflowExecution
	// Nodes 工作流节点定义（前端用于渲染节点状态）
	Nodes []interface{} `json:"nodes"`
	// Edges 工作流边定义（前端用于渲染连线）
	Edges []interface{} `json:"edges"`
	// NodeStates 节点当前状态（来自 StateManager，供前端直接刷新 UI，无需轮询）
	NodeStates map[string]string `json:"nodeStates"` // nodeId -> state: "pending"/"running"/"completed"/"failed"
}

// RequestCreateWorkflow 创建工作流请求
type RequestCreateWorkflow struct {
	WorkspaceID string        `json:"workspaceId"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Nodes       []interface{} `json:"nodes"`
	Edges       []interface{} `json:"edges"`
}

func (create *RequestCreateWorkflow) Valid(ctx *ctx.Context) error {
	if create.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if create.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(create.Name) > 200 {
		return fmt.Errorf("workflow name is too long (max 200 characters)")
	}
	if len(create.Description) > 1000 {
		return fmt.Errorf("workflow description is too long (max 1000 characters)")
	}

	// 验证工作空间是否存在
	var count int64
	db := ctx.Database.MySQL()
	db.Model(&models.Workspace{}).Where("id = ?", create.WorkspaceID).Count(&count)
	if count <= 0 {
		return fmt.Errorf("workspace not found")
	}

	return nil
}

// RequestUpdateWorkflow 更新工作流请求
type RequestUpdateWorkflow struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspaceId"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Nodes       []interface{} `json:"nodes"`
	Edges       []interface{} `json:"edges"`
}

func (update *RequestUpdateWorkflow) Valid(ctx *ctx.Context) error {
	if update.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}
	if update.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if update.Name == "" {
		return fmt.Errorf("workflow name is required")
	}
	if len(update.Name) > 200 {
		return fmt.Errorf("workflow name is too long (max 200 characters)")
	}
	if len(update.Description) > 1000 {
		return fmt.Errorf("workflow description is too long (max 1000 characters)")
	}

	// 验证工作流是否存在
	var workflowCount int64
	db := ctx.Database.MySQL()
	db.Model(&models.Workflow{}).Where("id = ?", update.ID).Count(&workflowCount)
	if workflowCount <= 0 {
		return fmt.Errorf("workflow not found")
	}

	// 验证工作空间是否存在
	var workspaceCount int64
	db.Model(&models.Workspace{}).Where("id = ?", update.WorkspaceID).Count(&workspaceCount)
	if workspaceCount <= 0 {
		return fmt.Errorf("workspace not found")
	}

	return nil
}

// RequestUpdateWorkflowDescription 更新工作流描述请求（部分更新）
type RequestUpdateWorkflowDescription struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Description string `json:"description"`
}

func (update *RequestUpdateWorkflowDescription) Valid(ctx *ctx.Context) error {
	if update.ID == "" {
		return fmt.Errorf("workflow ID is required")
	}
	if update.WorkspaceID == "" {
		return fmt.Errorf("workspace ID is required")
	}
	if len(update.Description) > 1000 {
		return fmt.Errorf("workflow description is too long (max 1000 characters)")
	}

	// 验证工作流是否存在
	var workflowCount int64
	db := ctx.Database.MySQL()
	db.Model(&models.Workflow{}).Where("id = ?", update.ID).Count(&workflowCount)
	if workflowCount <= 0 {
		return fmt.Errorf("workflow not found")
	}

	// 验证工作空间是否存在
	var workspaceCount int64
	db.Model(&models.Workspace{}).Where("id = ?", update.WorkspaceID).Count(&workspaceCount)
	if workspaceCount <= 0 {
		return fmt.Errorf("workspace not found")
	}

	return nil
}

// ResponseWorkflow 工作流响应
type ResponseWorkflow struct {
	ID          string        `json:"id"`
	WorkspaceID string        `json:"workspaceId"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Nodes       []interface{} `json:"nodes"`
	Edges       []interface{} `json:"edges"`
	CreatedAt   string        `json:"createdAt"`
	UpdatedAt   string        `json:"updatedAt"`
}

// ResponseWorkflowList 工作流列表响应
type ResponseWorkflowList struct {
	Workflows []ResponseWorkflow `json:"workflows"`
	Total     int                `json:"total"`
}

// ResponseWorkflowExecutionList 工作流执行列表响应
type ResponseWorkflowExecutionList struct {
	WorkflowId string                     `json:"workflowId"`
	Executions []models.WorkflowExecution `json:"executions"`
	Total      int                        `json:"total"`
}

// StreamTaskResponse 流式任务响应
type StreamTaskResponse struct {
	TaskID     string             `json:"task_id"`
	WorkflowID string             `json:"workflow_id"`
	Workflow   StreamWorkflowInfo `json:"workflow"`
	Data       StreamEventData    `json:"data"`
}

// StreamWorkflowInfo 流式工作流信息
type StreamWorkflowInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	VariableNum int    `json:"variable_num,omitempty"`
}

// StreamEventData 流式事件数据
type StreamEventData struct {
	ID             string            `json:"id"`
	WorkflowID     string            `json:"workflow_id"`
	WorkflowRunID  string            `json:"workflow_run_id"`
	Status         string            `json:"status"`
	Error          string            `json:"error,omitempty"`
	NodeID         string            `json:"node_id,omitempty"`
	NodeType       string            `json:"node_type,omitempty"`
	Outputs        map[string]any    `json:"outputs,omitempty"`
	NodeStates     map[string]string `json:"node_states,omitempty"`
	TotalNodes     int               `json:"total_nodes,omitempty"`
	CompletedNodes int               `json:"completed_nodes,omitempty"`
	Duration       float64           `json:"duration,omitempty"`
}

// StreamEvent 流式SSE事件
type StreamEvent struct {
	ID       string             `json:"id"`
	Event    string             `json:"event"`
	TaskID   string             `json:"task_id"`
	Workflow StreamWorkflowInfo `json:"workflow"`
	Node     *StreamNodeInfo    `json:"node,omitempty"`
	Data     StreamEventData    `json:"data"`
}

// StreamNodeInfo 流式节点信息
type StreamNodeInfo struct {
	ID      string         `json:"id"`
	Type    string         `json:"type"`
	Title   string         `json:"title,omitempty"`
	Index   int            `json:"index,omitempty"`
	Inputs  map[string]any `json:"inputs,omitempty"`
	Outputs map[string]any `json:"outputs,omitempty"`
	Error   string         `json:"error,omitempty"`
	ExeTime float64        `json:"elapsed_time,omitempty"`
}
