package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"

	"github.com/zeromicro/go-zero/core/logc"
)

// WorkflowExecutor 工作流执行器 - 基于"就绪"判定的逐步推进执行模型。
// 从 Start 节点出发，按节点的前驱终态情况逐步推进：每个节点的下游不预先计算，
// 而是在该节点执行完成后，依据其真实结果动态决定；未被任何前驱选中的节点标记为 skipped。
type WorkflowExecutor struct {
	registry        *NodeRegistry
	config          *EngineConfig
	retryPolicy     *RetryPolicy
	monitor         *ExecutionMonitor
	stateManager    *StateManager
	integrationRepo repos.IntegrationRepoInter
	executionMutex  sync.Mutex // 保护执行上下文的并发访问
}

// NewWorkflowExecutor 创建新的工作流执行器
func NewWorkflowExecutor(registry *NodeRegistry, config *EngineConfig, stateManager *StateManager, monitor *ExecutionMonitor, integrationRepo repos.IntegrationRepoInter) *WorkflowExecutor {
	return &WorkflowExecutor{
		registry:        registry,
		config:          config,
		retryPolicy:     NewRetryPolicy(config),
		monitor:         monitor,
		stateManager:    stateManager,
		integrationRepo: integrationRepo,
	}
}

// Execute 执行工作流
func (e *WorkflowExecutor) Execute(ctx context.Context, execution *models.WorkflowExecution, topology *WorkflowTopology, workspaceID string) error {
	// 构建执行图
	execGraph, err := e.buildExecutionGraph(topology)
	if err != nil {
		return err
	}

	// 创建执行上下文
	execCtx := &ExecutionContext{
		Context:         ctx,
		Execution:       execution,
		Graph:           execGraph,
		Registry:        e.registry,
		Config:          e.config,
		RetryPolicy:     e.retryPolicy,
		Monitor:         e.monitor,
		StateManager:    e.stateManager,
		WorkspaceID:     workspaceID,
		IntegrationRepo: e.integrationRepo,
	}

	// 执行工作流（从 Start 节点开始逐步推进）
	return e.execute(execCtx)
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	Context         context.Context
	Execution       *models.WorkflowExecution
	Graph           *ExecutionGraph
	Registry        *NodeRegistry
	Config          *EngineConfig
	RetryPolicy     *RetryPolicy
	Monitor         *ExecutionMonitor
	StateManager    *StateManager
	WorkspaceID     string
	IntegrationRepo repos.IntegrationRepoInter
}

// ExecutionGraph 执行图
type ExecutionGraph struct {
	Nodes       map[string]*ExecutableNode
	Edges       map[string][]string          // source -> targets
	HandleEdges map[string]map[string]string // source -> sourceHandle -> target (for SmartRouter branch routing)
	Incoming    map[string][]string          // target -> sources
	Mutex       sync.RWMutex                 // 保护图的并发访问
}

// ExecutableNode 可执行节点
type ExecutableNode struct {
	ID      string
	Type    string
	Config  map[string]any
	Status  NodeStatus
	Result  *models.NodeResult
	Error   error
	Retries int
	Mutex   sync.RWMutex
}

// NodeStatus 节点状态
type NodeStatus string

const (
	NodeStatusPending   NodeStatus = "pending"
	NodeStatusRunning   NodeStatus = "running"
	NodeStatusCompleted NodeStatus = "completed"
	NodeStatusFailed    NodeStatus = "failed"
	NodeStatusSkipped   NodeStatus = "skipped"
	NodeStatusWarning   NodeStatus = "warning" // 完成但有警告
)

// buildExecutionGraph 构建执行图
func (e *WorkflowExecutor) buildExecutionGraph(topology *WorkflowTopology) (*ExecutionGraph, error) {
	graph := &ExecutionGraph{
		Nodes:       make(map[string]*ExecutableNode),
		Edges:       make(map[string][]string),
		HandleEdges: make(map[string]map[string]string),
		Incoming:    make(map[string][]string),
	}

	// 创建边
	for _, edge := range topology.Edges {
		graph.Edges[edge.Source] = append(graph.Edges[edge.Source], edge.Target)
		graph.Incoming[edge.Target] = append(graph.Incoming[edge.Target], edge.Source)
		// 记录 sourceHandle 映射（用于 SmartRouter 等多输出节点）
		if edge.SourceHandle != "" {
			if graph.HandleEdges[edge.Source] == nil {
				graph.HandleEdges[edge.Source] = make(map[string]string)
			}
			graph.HandleEdges[edge.Source][edge.SourceHandle] = edge.Target
		}
	}

	// 创建节点：只包含有连接线的节点
	var isolatedNodes []string
	for _, node := range topology.Nodes {
		// 跳过没有连接线的节点（既没有入边也没有出边）
		hasIncoming := len(graph.Incoming[node.ID]) > 0
		hasOutgoing := len(graph.Edges[node.ID]) > 0
		if !hasIncoming && !hasOutgoing {
			isolatedNodes = append(isolatedNodes, node.ID)
			continue
		}

		graph.Nodes[node.ID] = &ExecutableNode{
			ID:     node.ID,
			Type:   node.Type,
			Config: node.Data,
			Status: NodeStatusPending,
		}
	}

	return graph, nil
}

// execute 从 Start 节点开始逐步推进执行工作流
func (e *WorkflowExecutor) execute(ctx *ExecutionContext) error {
	// 查找 Start 节点
	startNodeID := e.findStartNode(ctx.Graph)
	if startNodeID == "" {
		return fmt.Errorf("no start node found in workflow")
	}

	// 初始化执行状态
	e.executionMutex.Lock()
	ctx.Execution.Status = "running"
	ctx.Execution.StartedAt = time.Now().Unix()
	ctx.Execution.RunningNodes = []string{}
	ctx.Execution.CompletedNodes = []string{}
	ctx.Execution.FailedNodes = []string{}
	ctx.Execution.NodeResults = make(map[string]*models.NodeResult)
	e.executionMutex.Unlock()

	// 记录工作流开始
	ctx.Monitor.EmitStreamWorkflowStarted(ctx.Execution.ID, ctx.Execution.WorkflowID, ctx.Execution.WorkflowName)

	// 从 Start 节点开始逐步遍历执行
	selected := map[string]bool{startNodeID: true}
	return e.traverse(ctx, selected)
}

// findStartNode 查找工作流的 Start 节点（没有入边的节点）
func (e *WorkflowExecutor) findStartNode(graph *ExecutionGraph) string {
	for nodeID := range graph.Nodes {
		// 没有入边的节点是 Start 节点
		if len(graph.Incoming[nodeID]) == 0 {
			return nodeID
		}
	}
	return ""
}

// traverse 从已选中的起始节点出发，基于"就绪"判定逐步推进工作流，直至无可推进节点。
//
//  1. 收集所有"就绪"节点（其全部前驱均已进入终态：completed/failed/skipped）；
//  2. 就绪节点中，被上游选中的执行，未被任何前驱选中的标记为 skipped；
//  3. 执行完成后，按节点真实结果计算其选中的下游节点并记入 selected，供后续轮次使用。
//
// 下游路径不预先计算，而是随执行结果逐步展开，因此能正确处理分支与汇聚：
// 分支节点（Condition/SmartRouter）只把选中分支的下游记入 selected；
// 汇聚节点（多前驱 join）必须等全部前驱终态后才就绪，且只要有任一前驱选中它即会执行，
// 从而避免"未选中分支误伤共享下游"与"join 节点提前/重复执行"。
func (e *WorkflowExecutor) traverse(ctx *ExecutionContext, selected map[string]bool) error {
	for {
		select {
		case <-ctx.Context.Done():
			return ctx.Context.Err()
		default:
		}

		// 收集当前所有"就绪"的待执行节点
		ready := e.collectReadyNodes(ctx)
		if len(ready) == 0 {
			return nil
		}

		// 将就绪节点划分为"应执行"与"应跳过"
		var toExecute []string
		for _, nodeID := range ready {
			if selected[nodeID] {
				toExecute = append(toExecute, nodeID)
			} else {
				e.markNodeSkipped(ctx, nodeID)
			}
		}

		// 本轮仅产生跳过（无可执行节点）：跳过节点已进入终态，
		// 继续下一轮以推进它们的下游。
		if len(toExecute) == 0 {
			continue
		}

		// 并发执行本轮选中的就绪节点
		if err := e.executeBatch(ctx, toExecute); err != nil {
			return err
		}

		// 执行完成后，依据各节点真实结果计算其选中的下游节点
		for _, nodeID := range toExecute {
			for _, target := range e.resolveSelectedTargets(ctx, nodeID) {
				selected[target] = true
			}
		}
	}
}

// collectReadyNodes 收集所有"就绪"的待执行（Pending）节点：
// 一个节点就绪当且仅当其全部前驱节点均已进入终态。
func (e *WorkflowExecutor) collectReadyNodes(ctx *ExecutionContext) []string {
	graph := ctx.Graph

	graph.Mutex.RLock()
	defer graph.Mutex.RUnlock()

	var ready []string
	for nodeID, node := range graph.Nodes {
		node.Mutex.RLock()
		status := node.Status
		node.Mutex.RUnlock()

		if status != NodeStatusPending {
			continue
		}

		if e.allPredecessorsTerminal(graph, nodeID) {
			ready = append(ready, nodeID)
		}
	}
	return ready
}

// allPredecessorsTerminal 判断节点的所有前驱是否都已进入终态
// （completed/failed/skipped），即不再处于 pending 或 running。
// 调用方须持有 graph.Mutex 的读锁。
func (e *WorkflowExecutor) allPredecessorsTerminal(graph *ExecutionGraph, nodeID string) bool {
	for _, predID := range graph.Incoming[nodeID] {
		pred := graph.Nodes[predID]
		if pred == nil {
			continue
		}
		pred.Mutex.RLock()
		status := pred.Status
		pred.Mutex.RUnlock()
		if status == NodeStatusPending || status == NodeStatusRunning {
			return false
		}
	}
	return true
}

// resolveSelectedTargets 根据节点的真实执行结果，计算它选中的下游节点：
//   - 节点执行成功时，以 result.Next 为准。注意 executeNode 已为普通节点
//     自动填充 result.Next 为全部出边；而带分支路由的节点（Condition/SmartRouter）
//     会自行设置 result.Next。因此空的 result.Next 表示"不路由到任何下游"
//     （例如 Condition 命中的分支没有连接下游节点），不能再回退到全部出边。
//   - 仅当节点没有任何结果（如执行失败，result == nil）时，才默认走全部出边，
//     以便下游节点可以处理上游错误。
func (e *WorkflowExecutor) resolveSelectedTargets(ctx *ExecutionContext, nodeID string) []string {
	graph := ctx.Graph

	graph.Mutex.RLock()
	node := graph.Nodes[nodeID]
	allOutgoing := make([]string, len(graph.Edges[nodeID]))
	copy(allOutgoing, graph.Edges[nodeID])
	graph.Mutex.RUnlock()

	if node == nil || len(allOutgoing) == 0 {
		return nil
	}

	node.Mutex.RLock()
	result := node.Result
	node.Mutex.RUnlock()

	if result != nil {
		// 成功执行：result.Next 已是权威路由结果（普通节点已自动填充全部出边，
		// 分支节点已设置具体分支；为空则表示不路由到任何下游）。
		return result.Next
	}

	// 无结果（执行失败）：默认走全部出边，让下游有机会处理错误。
	return allOutgoing
}

// markNodeSkipped 将单个节点标记为 Skipped（仅处理 Pending 状态的节点）。
// 注意：此处只跳过该节点本身，不递归跳过其后代——后代是否跳过将在它们
// 自身就绪时根据"是否被选中"独立判定，从而正确处理共享下游的 join 节点。
func (e *WorkflowExecutor) markNodeSkipped(ctx *ExecutionContext, nodeID string) {
	graph := ctx.Graph

	graph.Mutex.RLock()
	node := graph.Nodes[nodeID]
	graph.Mutex.RUnlock()

	if node == nil {
		return
	}

	node.Mutex.Lock()
	if node.Status != NodeStatusPending {
		node.Mutex.Unlock()
		return // 已经在执行中或已完成，不覆盖状态
	}
	node.Status = NodeStatusSkipped
	nodeType := node.Type
	node.Mutex.Unlock()

	e.executionMutex.Lock()
	ctx.Execution.CompletedNodes = append(ctx.Execution.CompletedNodes, nodeID)
	e.executionMutex.Unlock()

	logc.Infof(ctx.Context, "[Workflow] Node %s skipped (branch not selected)", nodeID)

	if ctx.StateManager != nil {
		ctx.StateManager.SetNodeState(ctx.Execution.ID, nodeID, nodeType, NodeStateSkipped, nil, "branch not selected")
	}
	ctx.Monitor.EmitStreamNodeSkipped(ctx.Execution.ID, ctx.Execution.WorkflowID, ctx.Execution.WorkflowName, nodeID, nodeType)
}

// executeBatch 并发执行一批节点
func (e *WorkflowExecutor) executeBatch(ctx *ExecutionContext, nodeIDs []string) error {
	semaphore := make(chan struct{}, ctx.Config.MaxConcurrentNodes)
	var wg sync.WaitGroup
	errorChan := make(chan error, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		select {
		case <-ctx.Context.Done():
			return ctx.Context.Err()
		default:
			semaphore <- struct{}{}
			wg.Add(1)
			go func(nid string) {
				defer func() {
					<-semaphore
					wg.Done()
				}()

				if err := e.executeNode(ctx, nid); err != nil {
					select {
					case errorChan <- err:
					default:
					}
				}
			}(nodeID)
		}
	}

	wg.Wait()

	// 检查是否有错误（但不中断工作流）
	select {
	case err := <-errorChan:
		logc.Errorf(ctx.Context, "[Execute] batch error: %s", err)
	default:
	}

	return nil
}

// executeNode 执行单个节点
func (e *WorkflowExecutor) executeNode(ctx *ExecutionContext, nodeID string) error {
	// 获取节点引用（需要读锁保护）
	ctx.Graph.Mutex.RLock()
	node := ctx.Graph.Nodes[nodeID]
	ctx.Graph.Mutex.RUnlock()

	if node == nil {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	// 更新节点状态为运行中
	node.Mutex.Lock()
	if node.Status != NodeStatusPending {
		node.Mutex.Unlock()
		return nil // 已经被执行过
	}
	node.Status = NodeStatusRunning

	// 需要同步更新执行上下文
	e.executionMutex.Lock()
	ctx.Execution.RunningNodes = append(ctx.Execution.RunningNodes, nodeID)
	e.executionMutex.Unlock()

	node.Mutex.Unlock()

	// 记录开始执行时间
	startTime := time.Now()
	ctx.Monitor.EmitStreamNodeStarted(ctx.Execution.ID, ctx.Execution.WorkflowID, ctx.Execution.WorkflowName, nodeID, node.Type)

	// 更新状态管理器中的节点状态为 running
	if ctx.StateManager != nil {
		ctx.StateManager.SetNodeState(ctx.Execution.ID, nodeID, node.Type, NodeStateRunning, nil, "")
	}

	// 执行节点
	result, err := e.executeNodeWithRetry(ctx, nodeID)

	// 更新执行结果
	node.Mutex.Lock()
	node.Result = result
	node.Error = err

	// 节点执行完成后，根据连接线自动填充 Next 字段（仅当节点未设置 Next 且没有 HandleEdges 时）
	graph := ctx.Graph
	graph.Mutex.RLock()
	hasHandleEdges := len(graph.HandleEdges[nodeID]) > 0
	outgoingEdges := graph.Edges[nodeID]
	graph.Mutex.RUnlock()

	// 只有当节点没有 HandleEdges（自定义路由逻辑）且没有设置 Next 时，才自动填充
	if !hasHandleEdges && len(outgoingEdges) > 0 && result != nil && len(result.Next) == 0 {
		result.Next = outgoingEdges
		node.Result = result
	}

	var resultModel *models.NodeResult
	var hasError bool
	if err != nil {
		hasError = true
	} else if result != nil && result.Error != "" {
		// 节点返回了错误结果但没有抛出异常
		hasError = true
		err = fmt.Errorf(result.Error)
	}

	// 需要在节点锁内更新执行状态，避免并发写入问题
	if hasError {
		node.Status = NodeStatusFailed
		// 将错误信息写入 Outputs["error"]，让下游节点可通过 {{nodeID.error}} 引用
		outputs := map[string]any{"error": err.Error()}
		if result != nil && result.Outputs != nil {
			for k, v := range result.Outputs {
				outputs[k] = v
			}
			outputs["error"] = err.Error() // 确保 error 不被覆盖
		}
		resultModel = &models.NodeResult{
			Outputs: outputs,
			Error:   err.Error(),
		}
	} else {
		node.Status = NodeStatusCompleted
		resultModel = result
	}
	node.Mutex.Unlock()

	// 在节点锁外更新执行上下文（需要单独的同步）
	e.executionMutex.Lock()
	if hasError {
		ctx.Execution.FailedNodes = append(ctx.Execution.FailedNodes, nodeID)
	} else {
		ctx.Execution.CompletedNodes = append(ctx.Execution.CompletedNodes, nodeID)
	}
	ctx.Execution.NodeResults[nodeID] = resultModel
	ctx.Execution.RunningNodes = removeFromStringSlice(ctx.Execution.RunningNodes, nodeID)
	e.executionMutex.Unlock()

	// 记录执行完成
	duration := time.Since(startTime)
	if err != nil {
		ctx.Monitor.EmitStreamNodeFailed(ctx.Execution.ID, ctx.Execution.WorkflowID, ctx.Execution.WorkflowName, nodeID, node.Type, err.Error(), duration.Seconds())
		if ctx.StateManager != nil {
			ctx.StateManager.SetNodeState(ctx.Execution.ID, nodeID, node.Type, NodeStateFailed, resultModel, err.Error())
		}
	} else {
		ctx.Monitor.EmitStreamNodeCompleted(ctx.Execution.ID, ctx.Execution.WorkflowID, ctx.Execution.WorkflowName, nodeID, node.Type, result.Outputs, duration.Seconds())
		if ctx.StateManager != nil {
			ctx.StateManager.SetNodeState(ctx.Execution.ID, nodeID, node.Type, NodeStateCompleted, resultModel, "")
		}
	}

	return err
}

// executeNodeWithRetry 带重试的节点执行
func (e *WorkflowExecutor) executeNodeWithRetry(ctx *ExecutionContext, nodeID string) (*models.NodeResult, error) {
	for attempt := 0; attempt <= ctx.Config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Context.Done():
				return nil, ctx.Context.Err()
			case <-time.After(ctx.RetryPolicy.GetDelay(attempt)):
			}
		}

		nodeCtx := e.prepareNodeContext(ctx, nodeID)
		result, err := e.executeSingleNode(ctx, nodeCtx)

		if err == nil {
			return result, nil
		}

		ctx.Graph.Mutex.RLock()
		node := ctx.Graph.Nodes[nodeID]
		ctx.Graph.Mutex.RUnlock()

		if node != nil {
			node.Mutex.Lock()
			node.Retries = attempt
			node.Mutex.Unlock()
		}

		ctx.Monitor.NodeRetry(ctx.Execution.ID, nodeID, attempt, err)

		if attempt == ctx.Config.MaxRetries {
			return nil, err
		}
	}

	return nil, fmt.Errorf("node execution failed after %d retries", ctx.Config.MaxRetries)
}

// prepareNodeContext 准备节点上下文
func (e *WorkflowExecutor) prepareNodeContext(ctx *ExecutionContext, nodeID string) *models.NodeContext {
	// 获取节点引用（需要读锁保护）
	ctx.Graph.Mutex.RLock()
	node := ctx.Graph.Nodes[nodeID]
	ctx.Graph.Mutex.RUnlock()

	if node == nil {
		return &models.NodeContext{
			NodeID:    nodeID,
			NodeType:  "unknown",
			Config:    make(map[string]any),
			Input:     make(map[string]any),
			Variables: make(map[string]any),
		}
	}

	input := make(map[string]any)

	// 从 ctx.Execution.NodeResults 收集所有已执行节点的输出
	// NodeResults 包含所有已执行节点的结果，不仅限于直接前驱
	for ancestorID, nodeResult := range ctx.Execution.NodeResults {
		if ancestorID == nodeID || nodeResult == nil {
			continue
		}
		for portName, value := range nodeResult.Outputs {
			// 使用节点ID作为前缀，避免不同节点的相同输出键冲突
			inputKey := ancestorID + "." + portName
			input[inputKey] = value
			// 同时保留原始键名（如果不存在的话）
			if _, exists := input[portName]; !exists {
				input[portName] = value
			}
		}
	}

	// 创建变量解析器
	resolver := NewVariableResolver(ctx.Context, ctx.Execution.NodeResults, ctx.Execution.Variables)

	// 解析输入中的变量引用（单个字段解析失败不影响其他字段）
	resolvedInput, _ := resolver.ResolveVariablesInMap(input)

	// 解析配置中的变量引用（单个字段解析失败不影响其他字段）
	resolvedConfig, _ := resolver.ResolveVariablesInMap(node.Config)

	// 从 integration 合并配置
	resolvedConfig = e.mergeIntegrationConfig(ctx, node, resolvedConfig)

	// 获取该节点的 handle->target 映射（用于 SmartRouter 等多输出节点）
	var handleEdges map[string]string
	ctx.Graph.Mutex.RLock()
	if he, ok := ctx.Graph.HandleEdges[nodeID]; ok {
		handleEdges = he
	}
	ctx.Graph.Mutex.RUnlock()

	return &models.NodeContext{
		NodeID:      nodeID,
		NodeType:    node.Type,
		Config:      resolvedConfig,
		Input:       resolvedInput,
		Variables:   ctx.Execution.Variables,
		HandleEdges: handleEdges,
		WorkspaceID: ctx.WorkspaceID,
	}
}

// executeSingleNode 通过节点注册表查找对应执行器并在带超时的上下文中执行该节点
func (e *WorkflowExecutor) executeSingleNode(ctx *ExecutionContext, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	executor, exists := ctx.Registry.GetExecutor(nodeCtx.NodeType)
	if !exists {
		return nil, fmt.Errorf("unknown node type: %s", nodeCtx.NodeType)
	}

	timeout := ctx.Config.DefaultTimeout
	if timeoutVal, ok := nodeCtx.Config["timeout"].(float64); ok {
		timeout = time.Duration(timeoutVal) * time.Second
	}

	execCtx, cancel := context.WithTimeout(ctx.Context, timeout)
	defer cancel()

	return executor.Execute(execCtx, nodeCtx)
}

// removeFromStringSlice 从字符串切片中移除元素
func removeFromStringSlice(slice []string, item string) []string {
	for i, v := range slice {
		if v == item {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}
