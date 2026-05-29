package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/pkg/tools"
)

// WorkflowEngine 核心工作流引擎
type WorkflowEngine struct {
	registry        *NodeRegistry
	executor        *WorkflowExecutor
	monitor         *ExecutionMonitor
	stateManager    *StateManager
	retryPolicy     *RetryPolicy
	integrationRepo repos.IntegrationRepoInter
	mcpRepo         repos.MCPServerRepoInter
}

// EngineConfig 引擎配置
type EngineConfig struct {
	MaxConcurrentNodes int           `json:"maxConcurrentNodes"`
	DefaultTimeout     time.Duration `json:"defaultTimeout"`
	EnableRetry        bool          `json:"enableRetry"`
	MaxRetries         int           `json:"maxRetries"`
	RetryDelay         time.Duration `json:"retryDelay"`
}

// DefaultEngineConfig 默认引擎配置
func DefaultEngineConfig() *EngineConfig {
	return &EngineConfig{
		MaxConcurrentNodes: 10,
		DefaultTimeout:     30 * time.Second,
		EnableRetry:        true,
		MaxRetries:         3,
		RetryDelay:         1 * time.Second,
	}
}

// NewWorkflowEngine 创建新的工作流引擎
func NewWorkflowEngine(config *EngineConfig, integrationRepo repos.IntegrationRepoInter, mcpRepo repos.MCPServerRepoInter) *WorkflowEngine {
	if config == nil {
		config = DefaultEngineConfig()
	}

	registry := BootstrapNodeRegistry(mcpRepo)
	stateManager := NewStateManager()
	monitor := NewExecutionMonitor()
	executor := NewWorkflowExecutor(registry, config, stateManager, monitor, integrationRepo)
	retryPolicy := NewRetryPolicy(config)

	return &WorkflowEngine{
		registry:        registry,
		executor:        executor,
		monitor:         monitor,
		stateManager:    stateManager,
		retryPolicy:     retryPolicy,
		integrationRepo: integrationRepo,
		mcpRepo:         mcpRepo,
	}
}

// Execute 执行工作流
// 如果传入了 execution 对象，则使用该对象（复用已有的 ID 和元数据）
// 否则创建新的执行上下文
// 执行节点（executeNodes=true）默认开启
func (e *WorkflowEngine) Execute(ctx context.Context, workflow *models.Workflow, execution *models.WorkflowExecution) (*models.WorkflowExecution, error) {
	return e.ExecuteWithNodes(ctx, workflow, execution, true)
}

// ExecuteInitOnly 仅初始化节点状态（同步，不执行节点）
// 用于在启动 goroutine 之前预初始化，让 SSE 连接时 StateManager 已有数据
func (e *WorkflowEngine) ExecuteInitOnly(ctx context.Context, workflow *models.Workflow, execution *models.WorkflowExecution) (*models.WorkflowExecution, error) {
	return e.ExecuteWithNodes(ctx, workflow, execution, false)
}

// ExecuteWithNodes 内部实现
func (e *WorkflowEngine) ExecuteWithNodes(ctx context.Context, workflow *models.Workflow, execution *models.WorkflowExecution, executeNodes bool) (*models.WorkflowExecution, error) {
	// 解析工作流拓扑
	topology, err := e.parseWorkflowTopology(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow topology: %w", err)
	}

	// 检测孤立节点（没有连接线的节点）
	isolatedNodes := findIsolatedNodes(topology)

	// 如果没有传入 execution，则创建新的
	if execution == nil {
		execution = &models.WorkflowExecution{
			ID:             "exec-" + tools.RandId(),
			WorkflowID:     workflow.ID,
			WorkflowName:   workflow.Name,
			Status:         "running",
			StartedAt:      time.Now().Unix(),
			NodeResults:    make(map[string]*models.NodeResult),
			Variables:      make(map[string]any),
			RunningNodes:   []string{},
			CompletedNodes: []string{},
			FailedNodes:    []string{},
		}
	}

	// 检查是否已初始化（ExecuteInitOnly 已初始化过，跳过）
	_, stateExists := e.stateManager.CheckWorkflowState(execution.ID)

	// 仅在未初始化时创建状态
	if !stateExists {
		// 创建工作流状态
		e.stateManager.CreateWorkflowState(execution.ID, workflow.ID, workflow.Name, len(topology.Nodes))
		// 开始监控
		e.monitor.StartExecution(execution.ID, workflow.ID, workflow.Name, isolatedNodes)
		// 设置工作流为运行状态
		e.stateManager.UpdateWorkflowState(execution.ID, "running")
		// 初始化所有节点状态
		for _, node := range topology.Nodes {
			e.stateManager.SetNodeState(execution.ID, node.ID, node.Type, NodeStatePending, nil, "")
		}
		// 标记孤立节点为 skipped 状态，并发送 SSE 事件通知前端
		for _, nodeID := range isolatedNodes {
			// 获取节点类型
			nodeType := ""
			for _, n := range topology.Nodes {
				if n.ID == nodeID {
					nodeType = n.Type
					break
				}
			}
			e.stateManager.SetNodeState(execution.ID, nodeID, nodeType, NodeStateSkipped, nil, "no connections, skipped")
			e.monitor.EmitStreamNodeSkipped(execution.ID, workflow.ID, workflow.Name, nodeID, nodeType)
		}
	}

	// 执行工作流节点（可选）
	if executeNodes {
		err = e.executor.Execute(ctx, execution, topology, workflow.WorkspaceID)
		if err != nil {
			e.markExecutionFailed(execution, fmt.Sprintf("workflow execution failed: %v", err))
			e.stateManager.UpdateWorkflowState(execution.ID, "failed")
		} else {
			e.markExecutionCompleted(execution)
			e.stateManager.UpdateWorkflowState(execution.ID, "completed")
		}

		// 将节点执行结果存储到状态管理器，供前端查询
		e.stateManager.SetWorkflowNodeResults(execution.ID, execution.NodeResults)

		// 结束监控
		e.monitor.EndExecution(execution.ID, execution.Status)
	}

	return execution, nil
}

// parseWorkflowTopology 解析工作流拓扑
func (e *WorkflowEngine) parseWorkflowTopology(workflow *models.Workflow) (*WorkflowTopology, error) {
	// 解析节点
	nodes, err := parseNodes(workflow.Nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nodes: %w", err)
	}

	// 解析边
	edges, err := parseEdges(workflow.Edges)
	if err != nil {
		return nil, fmt.Errorf("failed to parse edges: %w", err)
	}

	return &WorkflowTopology{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// markExecutionFailed 标记执行失败
func (e *WorkflowEngine) markExecutionFailed(execution *models.WorkflowExecution, reason string) {
	execution.Status = "failed"
	execution.FinishedAt = time.Now().Unix()
	execution.Variables["error"] = reason
}

// markExecutionCompleted 标记执行完成
func (e *WorkflowEngine) markExecutionCompleted(execution *models.WorkflowExecution) {
	execution.Status = "completed"
	execution.FinishedAt = time.Now().Unix()
}

// GetRegistry 获取节点注册器
func (e *WorkflowEngine) GetRegistry() *NodeRegistry {
	return e.registry
}

// GetMonitor 获取执行监控器
func (e *WorkflowEngine) GetMonitor() *ExecutionMonitor {
	return e.monitor
}

// GetStateManager 获取状态管理器
func (e *WorkflowEngine) GetStateManager() *StateManager {
	return e.stateManager
}

// WorkflowTopology 工作流拓扑
type WorkflowTopology struct {
	Nodes []WorkflowNode `json:"nodes"`
	Edges []WorkflowEdge `json:"edges"`
}

// WorkflowNode 工作流节点
type WorkflowNode struct {
	ID   string                 `json:"id"`
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

// WorkflowEdge 工作流边
type WorkflowEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	SourceHandle string `json:"sourceHandle,omitempty"`
}

// parseNodes 解析节点
func parseNodes(nodesStr string) ([]WorkflowNode, error) {
	if nodesStr == "" {
		return []WorkflowNode{}, nil
	}

	var nodes []WorkflowNode
	err := json.Unmarshal([]byte(nodesStr), &nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nodes JSON: %w", err)
	}

	return nodes, nil
}

// findIsolatedNodes 找出没有连接线的孤立节点
func findIsolatedNodes(topology *WorkflowTopology) []string {
	inDegree := make(map[string]int)
	outDegree := make(map[string]int)

	for _, node := range topology.Nodes {
		inDegree[node.ID] = 0
		outDegree[node.ID] = 0
	}

	for _, edge := range topology.Edges {
		inDegree[edge.Target]++
		outDegree[edge.Source]++
	}

	var isolated []string
	for _, node := range topology.Nodes {
		if inDegree[node.ID] == 0 && outDegree[node.ID] == 0 {
			isolated = append(isolated, node.ID)
		}
	}
	return isolated
}

// parseEdges 解析边
func parseEdges(edgesStr string) ([]WorkflowEdge, error) {
	if edgesStr == "" {
		return []WorkflowEdge{}, nil
	}

	var edges []WorkflowEdge
	err := json.Unmarshal([]byte(edgesStr), &edges)
	if err != nil {
		return nil, fmt.Errorf("failed to parse edges JSON: %w", err)
	}

	return edges, nil
}
