package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"RCAgent/internal/models"
	"RCAgent/internal/repos"
)

// WorkflowExecutor 工作流执行器 - 基于树结构的层级执行模型
// 同级的多个节点必须全部完成执行后，才处理下一级节点
type WorkflowExecutor struct {
	registry       *NodeRegistry
	config         *EngineConfig
	retryPolicy    *RetryPolicy
	monitor        *ExecutionMonitor
	stateManager   *StateManager
	datasourceRepo repos.DatasourceRepoInter
	executionMutex sync.Mutex // 保护执行上下文的并发访问
}

// NewWorkflowExecutor 创建新的工作流执行器
func NewWorkflowExecutor(registry *NodeRegistry, config *EngineConfig, stateManager *StateManager, monitor *ExecutionMonitor, datasourceRepo repos.DatasourceRepoInter) *WorkflowExecutor {
	return &WorkflowExecutor{
		registry:       registry,
		config:         config,
		retryPolicy:    NewRetryPolicy(config),
		monitor:        monitor,
		stateManager:   stateManager,
		datasourceRepo: datasourceRepo,
	}
}

// Execute 执行工作流
func (e *WorkflowExecutor) Execute(ctx context.Context, execution *models.WorkflowExecution, topology *WorkflowTopology, workspaceID string) error {
	// 构建执行图
	execGraph, err := e.buildExecutionGraph(topology)
	if err != nil {
		return err
	}

	// 计算所有节点的层级
	levels := e.computeLevels(execGraph)

	// 创建执行上下文
	execCtx := &ExecutionContext{
		Context:        ctx,
		Execution:      execution,
		Graph:          execGraph,
		Registry:       e.registry,
		Config:         e.config,
		RetryPolicy:    e.retryPolicy,
		Monitor:        e.monitor,
		StateManager:   e.stateManager,
		WorkspaceID:    workspaceID,
		DatasourceRepo: e.datasourceRepo,
	}

	// 执行工作流（按层级）
	return e.executeByLevels(execCtx, levels)
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	Context        context.Context
	Execution      *models.WorkflowExecution
	Graph          *ExecutionGraph
	Registry       *NodeRegistry
	Config         *EngineConfig
	RetryPolicy    *RetryPolicy
	Monitor        *ExecutionMonitor
	StateManager   *StateManager
	WorkspaceID    string
	DatasourceRepo repos.DatasourceRepoInter
}

// ExecutionGraph 执行图
type ExecutionGraph struct {
	Nodes    map[string]*ExecutableNode
	Edges    map[string][]string // source -> targets
	Incoming map[string][]string // target -> sources
	Mutex    sync.RWMutex        // 保护图的并发访问
}

// ExecutableNode 可执行节点
type ExecutableNode struct {
	ID      string
	Type    string
	Config  map[string]any
	Status  NodeStatus
	Level   int // 节点层级，同级节点需全部完成后才执行下一级
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

// LevelNodes 存储每层级的节点
type LevelNodes map[int][]string

// buildExecutionGraph 构建执行图
func (e *WorkflowExecutor) buildExecutionGraph(topology *WorkflowTopology) (*ExecutionGraph, error) {
	graph := &ExecutionGraph{
		Nodes:    make(map[string]*ExecutableNode),
		Edges:    make(map[string][]string),
		Incoming: make(map[string][]string),
	}

	// 创建边
	for _, edge := range topology.Edges {
		graph.Edges[edge.Source] = append(graph.Edges[edge.Source], edge.Target)
		graph.Incoming[edge.Target] = append(graph.Incoming[edge.Target], edge.Source)
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
			Level:  0,
		}
	}

	return graph, nil
}

// computeLevels 计算所有节点的层级
func (e *WorkflowExecutor) computeLevels(graph *ExecutionGraph) LevelNodes {
	// 计算每个节点的入度
	inDegree := make(map[string]int)
	for nodeID := range graph.Nodes {
		inDegree[nodeID] = 0
	}
	for target, sources := range graph.Incoming {
		inDegree[target] = len(sources)
	}

	// 初始化层级：入度为0的节点在第0层
	levels := make(LevelNodes)
	queue := []string{}

	// 找出所有根节点（入度为0）
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
			graph.Nodes[nodeID].Level = 0
			levels[0] = append(levels[0], nodeID)
		}
	}

	// BFS计算层级
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		currentNode := graph.Nodes[current]
		currentLevel := currentNode.Level

		// 遍历当前节点的所有出边
		for _, targetID := range graph.Edges[current] {
			targetNode := graph.Nodes[targetID]

			// 更新目标节点的层级（取所有前置节点的最大层级+1）
			newLevel := currentLevel + 1
			if newLevel > targetNode.Level {
				targetNode.Level = newLevel
			}

			// 更新入度并检查是否就绪
			inDegree[targetID]--
			if inDegree[targetID] == 0 {
				queue = append(queue, targetID)
				// 节点就绪，将其加入对应层级
				levels[targetNode.Level] = append(levels[targetNode.Level], targetID)
			}
		}
	}

	return levels
}

// executeByLevels 按层级执行节点
// 每一层内的所有节点并发执行，等待全部完成后才执行下一层
func (e *WorkflowExecutor) executeByLevels(ctx *ExecutionContext, levels LevelNodes) error {
	maxLevel := 0
	for level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	// 按层级顺序执行
	for level := 0; level <= maxLevel; level++ {
		nodeIDs, exists := levels[level]
		if !exists || len(nodeIDs) == 0 {
			continue
		}

		log.Printf("[Executor] Level %d: executing nodes %v\n", level, nodeIDs)

		// 执行当前层级的所有节点（并发）
		if err := e.executeLevel(ctx, nodeIDs); err != nil {
			return err
		}

		// 等待当前层级所有节点完成
		if err := e.waitLevelComplete(ctx, nodeIDs); err != nil {
			// 如果是超时错误，停止执行
			if strings.Contains(err.Error(), "timeout") {
				return err
			}
		}

	}

	return nil
}

// executeLevel 并发执行某一层级的所有节点
func (e *WorkflowExecutor) executeLevel(ctx *ExecutionContext, nodeIDs []string) error {
	semaphore := make(chan struct{}, ctx.Config.MaxConcurrentNodes)
	var wg sync.WaitGroup
	errorChan := make(chan error, len(nodeIDs))

	for _, nodeID := range nodeIDs {
		select {
		case <-ctx.Context.Done():
			return ctx.Context.Err()
		case <-errorChan:
			// 遇到错误继续执行其他节点
		case semaphore <- struct{}{}:
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

	// 检查是否有失败（但不返回错误，让工作流继续）
	select {
	case err := <-errorChan:
		log.Printf("[Execute] err: %s", err)
	default:
	}
	return nil
}

// waitLevelComplete 等待某一层级的所有节点执行完成
func (e *WorkflowExecutor) waitLevelComplete(ctx *ExecutionContext, nodeIDs []string) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(10 * time.Minute) // 10分钟超时

	for {
		select {
		case <-ctx.Context.Done():
			return ctx.Context.Err()
		case <-timeout:
			return fmt.Errorf("level execution timeout")
		case <-ticker.C:
			allComplete := true
			hasFailure := false
			for _, nodeID := range nodeIDs {
				ctx.Graph.Mutex.RLock()
				node := ctx.Graph.Nodes[nodeID]
				ctx.Graph.Mutex.RUnlock()

				if node != nil {
					node.Mutex.RLock()
					status := node.Status
					node.Mutex.RUnlock()

					if status != NodeStatusCompleted && status != NodeStatusFailed {
						allComplete = false
						break
					}
					if status == NodeStatusFailed {
						hasFailure = true
					}
				}
			}

			if allComplete {
				if hasFailure {
					return fmt.Errorf("level has failed nodes")
				}
				return nil
			}
		}
	}
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
		resultModel = &models.NodeResult{
			Outputs: map[string]any{},
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

	// 从前置节点收集输出
	ctx.Graph.Mutex.RLock()
	incomingNodes := ctx.Graph.Incoming[nodeID]
	ctx.Graph.Mutex.RUnlock()

	for _, sourceID := range incomingNodes {
		ctx.Graph.Mutex.RLock()
		sourceNode := ctx.Graph.Nodes[sourceID]
		ctx.Graph.Mutex.RUnlock()

		if sourceNode != nil {
			sourceNode.Mutex.RLock()
			if sourceNode.Result != nil {
				for portName, value := range sourceNode.Result.Outputs {
					// 使用节点ID作为前缀，避免不同节点的相同输出键冲突
					inputKey := sourceID + "." + portName
					input[inputKey] = value
					// 同时保留原始键名（如果不存在的话）
					if _, exists := input[portName]; !exists {
						input[portName] = value
					}
				}
			}
			sourceNode.Mutex.RUnlock()
		}
	}

	// 创建变量解析器
	resolver := NewVariableResolver(ctx.Execution.NodeResults, ctx.Execution.Variables)

	// 解析输入中的变量引用
	resolvedInput, err := resolver.ResolveVariablesInMap(input)
	if err != nil {
		resolvedInput = input
	}

	// 解析配置中的变量引用
	resolvedConfig, err := resolver.ResolveVariablesInMap(node.Config)
	if err != nil {
		resolvedConfig = node.Config
	}

	// 从 datasource 合并配置
	resolvedConfig = e.mergeDatasourceConfig(ctx, node, resolvedConfig)

	return &models.NodeContext{
		NodeID:    nodeID,
		NodeType:  node.Type,
		Config:    resolvedConfig,
		Input:     resolvedInput,
		Variables: ctx.Execution.Variables,
	}
}

// mergeDatasourceConfig 从 datasource 合并配置到节点配置
func (e *WorkflowExecutor) mergeDatasourceConfig(ctx *ExecutionContext, node *ExecutableNode, config map[string]any) map[string]any {
	// 检查节点配置中是否指定了 datasourceId
	datasourceID, ok := config["datasourceId"].(string)
	if !ok || datasourceID == "" {
		return config
	}

	// 如果没有 datasource repo，直接返回原配置
	if e.datasourceRepo == nil {
		return config
	}

	// 查找 datasource
	datasource, err := e.datasourceRepo.GetByID(datasourceID)
	if err != nil {
		return config
	}

	// 确保 datasource 属于同一个 workspace
	if datasource.WorkspaceID != ctx.WorkspaceID {
		return config
	}

	// 根据 datasource 类型合并配置
	switch datasource.Type {
	case models.DatasourceTypeLLM:
		config = e.mergeAIConfig(config, datasource)
	case models.DatasourceTypePrometheus:
		config = e.mergePrometheusConfig(config, datasource)
	case models.DatasourceTypeSSHAuth:
		config = e.mergeSSHAuth(config, datasource)
	case models.DatasourceTypeKubernetes:
		config = e.mergeKubeConfig(config, datasource)
	default:
		// 其他 HTTP 类型的数据源
		if datasource.Type.IsHTTPType() {
			config = e.mergeHTTPConfig(config, datasource, node.Type)
		}
	}

	return config
}

// mergeAIConfig 合并 AI datasource 配置
func (e *WorkflowExecutor) mergeAIConfig(config map[string]any, datasource *models.Datasource) map[string]any {
	if datasource.AIConfig == "" {
		return config
	}

	var aiConfig models.AIConfig
	if err := json.Unmarshal([]byte(datasource.AIConfig), &aiConfig); err != nil {
		return config
	}

	// 数据源配置优先于节点配置
	if aiConfig.APIKey != "" {
		config["apiKey"] = aiConfig.APIKey
	}
	if aiConfig.BaseURL != "" {
		config["baseURL"] = aiConfig.BaseURL
	} else if aiConfig.API != "" {
		config["baseURL"] = aiConfig.API
	}
	if aiConfig.Model != "" {
		config["model"] = aiConfig.Model
	}

	return config
}

// mergePrometheusConfig 合并 Prometheus datasource 配置
func (e *WorkflowExecutor) mergePrometheusConfig(config map[string]any, datasource *models.Datasource) map[string]any {
	if datasource.HTTPConfig == "" {
		return config
	}

	var httpConfig models.HTTPConfig
	if err := json.Unmarshal([]byte(datasource.HTTPConfig), &httpConfig); err != nil {
		return config
	}

	if httpConfig.Endpoint != "" {
		config["prometheusUrl"] = httpConfig.Endpoint
	}

	// 合并认证配置
	config = e.mergeHTTPAuth(config, httpConfig.Auth)

	return config
}

// mergeHTTPConfig 合并通用 HTTP datasource 配置
func (e *WorkflowExecutor) mergeHTTPConfig(config map[string]any, datasource *models.Datasource, nodeType string) map[string]any {
	if datasource.HTTPConfig == "" {
		return config
	}

	var httpConfig models.HTTPConfig
	if err := json.Unmarshal([]byte(datasource.HTTPConfig), &httpConfig); err != nil {
		return config
	}

	// 根据节点类型设置 URL 配置
	switch nodeType {
	case "prometheus":
		if httpConfig.Endpoint != "" {
			config["prometheusUrl"] = httpConfig.Endpoint
		}
	case "jaeger":
		if httpConfig.Endpoint != "" {
			config["jaegerUrl"] = httpConfig.Endpoint
		}
	case "clickhouse":
		if httpConfig.Endpoint != "" {
			config["clickhouseUrl"] = httpConfig.Endpoint
		}
	case "elasticsearch":
		if httpConfig.Endpoint != "" {
			config["elasticsearchUrl"] = httpConfig.Endpoint
		}
	case "loki":
		if httpConfig.Endpoint != "" {
			config["lokiUrl"] = httpConfig.Endpoint
		}
	}

	// 合并认证配置
	config = e.mergeHTTPAuth(config, httpConfig.Auth)

	return config
}

// mergeHTTPAuth 合并 HTTP 认证配置
func (e *WorkflowExecutor) mergeHTTPAuth(config map[string]any, auth models.DatasourceAuth) map[string]any {
	switch auth.Type {
	case models.AuthTypeBearer:
		if auth.Token != "" {
			config["authToken"] = auth.Token
		}
	case models.AuthTypeBasic:
		if auth.Username != "" {
			config["authUsername"] = auth.Username
		}
		if auth.Password != "" {
			config["authPassword"] = auth.Password
		}
	case models.AuthTypeAPIKey:
		if auth.APIKey != "" {
			config["authAPIKey"] = auth.APIKey
			config["authAPIKeyName"] = auth.APIKeyName
			config["authAPIKeyIn"] = auth.APIKeyIn
		}
	}
	return config
}

func (e *WorkflowExecutor) mergeSSHAuth(config map[string]any, datasource *models.Datasource) map[string]any {
	if datasource.HTTPConfig == "" {
		return config
	}

	var httpConfig models.HTTPConfig
	if err := json.Unmarshal([]byte(datasource.HTTPConfig), &httpConfig); err != nil {
		return config
	}

	auth := httpConfig.Auth

	// 合并认证类型
	if auth.Type != "" {
		config["authType"] = string(auth.Type)
	}

	// 合并用户名
	if auth.Username != "" {
		config["authUsername"] = auth.Username
	}

	// 合并密码
	if auth.Password != "" {
		config["authPassword"] = auth.Password
	}
	// 合并私钥
	if auth.PrivateKey != "" {
		config["authPrivateKey"] = auth.PrivateKey
	}

	return config
}

// mergeKubeConfig 合并 Kubernetes datasource 配置
func (e *WorkflowExecutor) mergeKubeConfig(config map[string]any, datasource *models.Datasource) map[string]any {
	if datasource.KubeConfig == "" {
		return config
	}

	var kubeConfig models.KubeConfig
	if err := json.Unmarshal([]byte(datasource.KubeConfig), &kubeConfig); err != nil {
		return config
	}

	if kubeConfig.Kubeconfig != "" {
		config["kubeconfig"] = kubeConfig.Kubeconfig
	}

	return config
}

// executeSingleNode 执行单个节点
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
