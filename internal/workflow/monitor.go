package workflow

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"SREFlow/internal/models"
)

// ExecutionMonitor 执行监控器
type ExecutionMonitor struct {
	mu           sync.RWMutex
	executions   map[string]*ExecutionMetrics
	nodeMetrics  map[string]*NodeMetrics
	eventHandler EventHandler
	// SSE 事件通道 - executionId -> 事件通道 (向后兼容)
	eventChannels map[string]chan *ExecutionEvent
	// 流式事件通道 - executionId -> 事件通道
	streamEventChannels map[string]chan *StreamStyleEvent
	// 保护通道关闭的锁（防止 emitEvent 和 SSE handler 之间竞争）
	channelMu sync.Mutex
}

// StreamStyleEvent 流式SSE事件
type StreamStyleEvent struct {
	ID       string       `json:"id"`             // 事件ID，用于追踪
	Event    string       `json:"event"`          // 事件类型: running, succeeded, failed, node_started, node_completed, node_failed
	TaskID   string       `json:"task_id"`        // 任务ID (execution ID)
	Workflow WorkflowInfo `json:"workflow"`       // 工作流信息
	Node     *NodeInfo    `json:"node,omitempty"` // 节点信息（节点事件时）
	Data     EventData    `json:"data"`           // 事件数据
}

// WorkflowInfo 工作流基本信息
type WorkflowInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	VariableNum int    `json:"variable_num,omitempty"`
}

// NodeInfo 节点信息
type NodeInfo struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Title     string            `json:"title,omitempty"`
	Index     int               `json:"index,omitempty"`
	PreNodeID string            `json:"pre_node_id,omitempty"`
	Inputs    map[string]any    `json:"inputs,omitempty"`
	Outputs   map[string]string `json:"outputs,omitempty"` // 字段统一为 string
	Error     string            `json:"error,omitempty"`
	ExeTime   string            `json:"elapsed_time,omitempty"` // 执行耗时(格式化字符串)
}

// EventData 事件数据
type EventData struct {
	ID              string            `json:"id"` // execution ID
	WorkflowID      string            `json:"workflow_id"`
	WorkflowRunID   string            `json:"workflow_run_id"`
	Status          string            `json:"status"` // running, succeeded, failed
	Error           string            `json:"error,omitempty"`
	NodeExecutionID string            `json:"node_execution_id,omitempty"`
	NodeID          string            `json:"node_id,omitempty"`
	NodeType        string            `json:"node_type,omitempty"`
	NodeTitle       string            `json:"node_title,omitempty"`
	Outputs         map[string]string `json:"outputs,omitempty"`     // 字段统一为 string
	NodeStates      map[string]string `json:"node_states,omitempty"` // 节点状态快照
	TotalNodes      int               `json:"total_nodes,omitempty"`
	CompletedNodes  int               `json:"completed_nodes,omitempty"`
	Duration        float64           `json:"duration,omitempty"` // 总耗时(秒)
}

// ExecutionEvent 执行事件（SSE推送用）- 保留向后兼容
type ExecutionEvent struct {
	Type        string         `json:"type"` // "node_started" | "node_completed" | "node_failed" | "workflow_started" | "workflow_completed" | "workflow_failed"
	ExecutionID string         `json:"executionId"`
	NodeID      string         `json:"nodeId,omitempty"`
	NodeType    string         `json:"nodeType,omitempty"`
	Status      string         `json:"status,omitempty"` // 当前工作流状态
	Error       string         `json:"error,omitempty"`
	Outputs     map[string]any `json:"outputs,omitempty"`
	Timestamp   int64          `json:"timestamp"`
	// 快照：每次事件推送完整的节点状态快照，供前端直接刷新
	NodeStates map[string]string `json:"nodeStates"` // nodeId -> state
}

// ExecutionMetrics 执行指标
type ExecutionMetrics struct {
	ExecutionID    string
	WorkflowID     string
	WorkflowName   string
	Status         string
	StartTime      time.Time
	EndTime        *time.Time
	Duration       time.Duration
	TotalNodes     int
	CompletedNodes int
	FailedNodes    int
	RunningNodes   int
	NodeStatuses   map[string]NodeStatus
	Mutex          sync.RWMutex
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	NodeID      string
	NodeType    string
	ExecutionID string
	Status      NodeStatus
	StartTime   time.Time
	EndTime     *time.Time
	Duration    time.Duration
	Retries     int
	Error       string
	Mutex       sync.RWMutex
}

// EventHandler 事件处理器接口
type EventHandler interface {
	OnExecutionStarted(executionID string)
	OnExecutionCompleted(executionID string, metrics *ExecutionMetrics)
	OnExecutionFailed(executionID string, metrics *ExecutionMetrics, err error)
	OnNodeStarted(executionID, nodeID, nodeType string)
	OnNodeCompleted(executionID, nodeID string, result *models.NodeResult)
	OnNodeFailed(executionID, nodeID string, err error)
	OnNodeRetry(executionID, nodeID string, attempt int, err error)
}

// DefaultEventHandler 默认事件处理器
type DefaultEventHandler struct{}

// NewStreamStyleEvent 创建流式风格的事件
func NewStreamStyleEvent(eventType, executionID, workflowID, workflowName string) *StreamStyleEvent {
	return &StreamStyleEvent{
		ID:     executionID,
		Event:  eventType,
		TaskID: executionID,
		Workflow: WorkflowInfo{
			ID:   workflowID,
			Name: workflowName,
		},
		Data: EventData{
			ID:            executionID,
			WorkflowID:    workflowID,
			WorkflowRunID: executionID,
			Status:        getStatusFromEvent(eventType),
		},
	}
}

// NewStreamNodeEvent 创建流式风格的节点事件
func NewStreamNodeEvent(eventType, executionID, workflowID, workflowName, nodeID, nodeType string, outputs map[string]any, errMsg string, duration float64) *StreamStyleEvent {
	event := NewStreamStyleEvent(eventType, executionID, workflowID, workflowName)
	event.Node = &NodeInfo{
		ID:      nodeID,
		Type:    nodeType,
		ExeTime: formatDuration(duration),
	}
	if errMsg != "" {
		event.Node.Error = errMsg
	}
	if outputs != nil {
		stringified := make(map[string]string)
		for k, v := range outputs {
			stringified[k] = stringifyValue(v)
		}
		event.Node.Outputs = stringified
	}
	return event
}

// formatDuration 格式化耗时为用户友好的字符串
func formatDuration(seconds float64) string {
	if seconds < 0.001 {
		return "<1ms"
	} else if seconds < 1 {
		return fmt.Sprintf("%.1fms", seconds*1000)
	} else if seconds < 60 {
		return fmt.Sprintf("%.2fs", seconds)
	} else {
		minutes := int(seconds) / 60
		remainingSeconds := seconds - float64(minutes*60)
		return fmt.Sprintf("%dm%.1fs", minutes, remainingSeconds)
	}
}

// getStatusFromEvent 根据事件类型获取工作流状态
func getStatusFromEvent(event string) string {
	switch event {
	case "running", "node_started":
		return "running"
	case "succeeded", "node_completed":
		return "running" // 节点完成时工作流仍在运行，直到所有节点完成
	case "node_failed":
		return "failed" // 节点失败时工作流失败
	case "workflow_completed":
		return "succeeded"
	case "workflow_failed":
		return "failed"
	default:
		return "running"
	}
}

// NewExecutionMonitor 创建新的执行监控器
func NewExecutionMonitor() *ExecutionMonitor {
	return &ExecutionMonitor{
		executions:          make(map[string]*ExecutionMetrics),
		nodeMetrics:         make(map[string]*NodeMetrics),
		eventHandler:        &DefaultEventHandler{},
		eventChannels:       make(map[string]chan *ExecutionEvent),
		streamEventChannels: make(map[string]chan *StreamStyleEvent),
	}
}

// RegisterEventChannel 注册 SSE 事件通道
func (m *ExecutionMonitor) RegisterEventChannel(executionID string, ch chan *ExecutionEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventChannels[executionID] = ch
}

// RegisterEventSSEEventChannel 注册流式 SSE 事件通道
// 如果 channel 已存在，返回原有 channel（用于前端 SSE 连接时复用已创建的 channel）
func (m *ExecutionMonitor) RegisterEventSSEEventChannel(executionID string, ch chan *StreamStyleEvent) chan *StreamStyleEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 如果已有 channel 存在，关闭新传入的 channel 并返回旧的
	if existingCh, exists := m.streamEventChannels[executionID]; exists && existingCh != nil {
		go func() { close(ch) }()
		return existingCh
	}

	m.streamEventChannels[executionID] = ch
	return ch
}

// GetStreamEventChannel 获取流式事件通道（已注册的）
func (m *ExecutionMonitor) GetStreamEventChannel(executionID string) chan *StreamStyleEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.streamEventChannels[executionID]
}

// UnregisterEventChannel 取消注册 SSE 事件通道
func (m *ExecutionMonitor) UnregisterEventChannel(executionID string) {
	m.mu.Lock()
	ch, exists := m.eventChannels[executionID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.eventChannels, executionID)
	m.mu.Unlock()

	// 用 channelMu 保护 close 操作，防止与 emitEvent 竞争
	m.channelMu.Lock()
	defer m.channelMu.Unlock()
	close(ch)
}

// UnregisterEventSSEEventChannel 取消注册流式 SSE 事件通道（兼容services调用）
func (m *ExecutionMonitor) UnregisterEventSSEEventChannel(executionID string) {
	m.mu.Lock()
	ch, exists := m.streamEventChannels[executionID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.streamEventChannels, executionID)
	m.mu.Unlock()

	m.channelMu.Lock()
	defer m.channelMu.Unlock()
	close(ch)
}

// emitEvent 发送事件到 SSE 通道
// 使用 channelMu 锁保护通道访问，确保在检查和发送之间通道不会被关闭
func (m *ExecutionMonitor) emitEvent(event *ExecutionEvent) {
	m.mu.RLock()
	ch, exists := m.eventChannels[event.ExecutionID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	m.channelMu.Lock()
	defer m.channelMu.Unlock()

	select {
	case ch <- event:
	default:
	}
}

// EmitStreamWorkflowStarted 发送流式风格的工作流开始事件
func (m *ExecutionMonitor) EmitStreamWorkflowStarted(executionID, workflowID, workflowName string) {
	event := NewStreamStyleEvent("running", executionID, workflowID, workflowName)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamNodeStarted 发送流式风格的节点开始事件
func (m *ExecutionMonitor) EmitStreamNodeStarted(executionID, workflowID, workflowName, nodeID, nodeType string) {
	event := NewStreamNodeEvent("node_started", executionID, workflowID, workflowName, nodeID, nodeType, nil, "", 0)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamNodeCompleted 发送流式风格的节点完成事件
func (m *ExecutionMonitor) EmitStreamNodeCompleted(executionID, workflowID, workflowName, nodeID, nodeType string, outputs map[string]any, duration float64) {
	// 将 outputs 转换为 string 格式
	stringifiedOutputs := make(map[string]string)
	for k, v := range outputs {
		stringifiedOutputs[k] = stringifyValue(v)
	}

	event := NewStreamNodeEvent("node_completed", executionID, workflowID, workflowName, nodeID, nodeType, nil, "", duration)
	event.Node.Outputs = stringifiedOutputs
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)

	// 更新完成节点计数
	m.mu.RLock()
	if metrics, exists := m.executions[executionID]; exists {
		event.Data.TotalNodes = metrics.TotalNodes
		event.Data.CompletedNodes = metrics.CompletedNodes
	}
	m.mu.RUnlock()

	m.emitStreamEvent(event)
}

// EmitStreamNodeFailed 发送流式风格的节点失败事件
func (m *ExecutionMonitor) EmitStreamNodeFailed(executionID, workflowID, workflowName, nodeID, nodeType string, errMsg string, duration float64) {
	event := NewStreamNodeEvent("node_failed", executionID, workflowID, workflowName, nodeID, nodeType, nil, errMsg, duration)
	event.Data.Error = errMsg
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamNodeSkipped 发送流式风格的节点跳过事件（孤立节点）
func (m *ExecutionMonitor) EmitStreamNodeSkipped(executionID, workflowID, workflowName, nodeID, nodeType string) {
	// 更新监控指标中的节点状态
	m.mu.RLock()
	metrics, exists := m.executions[executionID]
	m.mu.RUnlock()
	if exists {
		metrics.Mutex.Lock()
		metrics.NodeStatuses[nodeID] = NodeStatusSkipped
		metrics.Mutex.Unlock()
	}

	event := NewStreamNodeEvent("node_skipped", executionID, workflowID, workflowName, nodeID, nodeType, nil, "no connections, skipped", 0)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamWorkflowCompleted 发送流式风格的工作流完成事件
func (m *ExecutionMonitor) EmitStreamWorkflowCompleted(executionID, workflowID, workflowName string, status string) {
	eventType := "workflow_completed"
	if status == "failed" {
		eventType = "workflow_failed"
	}

	event := NewStreamStyleEvent(eventType, executionID, workflowID, workflowName)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)

	// 计算总耗时
	m.mu.RLock()
	if metrics, exists := m.executions[executionID]; exists {
		event.Data.TotalNodes = metrics.TotalNodes
		event.Data.CompletedNodes = metrics.CompletedNodes
		event.Data.Duration = metrics.Duration.Seconds()
	}
	m.mu.RUnlock()

	m.emitStreamEvent(event)
}

// emitStreamEvent 发送流式风格事件到 SSE 通道
func (m *ExecutionMonitor) emitStreamEvent(event *StreamStyleEvent) {
	m.mu.RLock()
	ch, exists := m.streamEventChannels[event.TaskID]
	m.mu.RUnlock()

	if !exists {
		return
	}

	m.channelMu.Lock()
	defer m.channelMu.Unlock()

	select {
	case ch <- event:
	default:
	}
}

// getNodeStatesSnapshot 获取当前所有节点状态快照
func (m *ExecutionMonitor) getNodeStatesSnapshot(executionID string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	states := make(map[string]string)
	if metrics, exists := m.executions[executionID]; exists {
		metrics.Mutex.RLock()
		for nodeID, status := range metrics.NodeStatuses {
			states[nodeID] = string(status)
		}
		metrics.Mutex.RUnlock()
	}
	return states
}

// SetEventHandler 设置事件处理器
func (m *ExecutionMonitor) SetEventHandler(handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHandler = handler
}

// StartExecution 开始执行监控
// isolatedNodes: 孤立节点ID列表，这些节点会被预标记为 skipped
func (m *ExecutionMonitor) StartExecution(executionID, workflowID, workflowName string, isolatedNodes []string) {
	m.mu.Lock()

	metrics := &ExecutionMetrics{
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		StartTime:    time.Now(),
		NodeStatuses: make(map[string]NodeStatus),
	}

	// 预标记孤立节点为 skipped
	for _, nodeID := range isolatedNodes {
		metrics.NodeStatuses[nodeID] = NodeStatusSkipped
	}

	m.executions[executionID] = metrics

	// 获取节点状态快照（在释放锁之前）
	nodeStates := m.getNodeStatesSnapshotRL(m.executions, executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnExecutionStarted(executionID)
	}

	// 推送 workflow_started 事件 (向后兼容)
	m.emitEvent(&ExecutionEvent{
		Type:        "workflow_started",
		ExecutionID: executionID,
		Status:      "running",
		Timestamp:   time.Now().Unix(),
		NodeStates:  nodeStates,
	})

	// 推送流式风格事件
	m.EmitStreamWorkflowStarted(executionID, workflowID, workflowName)
}

// getNodeStatesSnapshotRL 读取节点状态快照（调用方须持有 mu.RLock）
func (m *ExecutionMonitor) getNodeStatesSnapshotRL(executions map[string]*ExecutionMetrics, executionID string) map[string]string {
	states := make(map[string]string)
	if metrics, exists := executions[executionID]; exists {
		for nodeID, status := range metrics.NodeStatuses {
			states[nodeID] = string(status)
		}
	}
	return states
}

// EndExecution 结束执行监控
func (m *ExecutionMonitor) EndExecution(executionID, status string) {
	m.mu.Lock()

	metrics, exists := m.executions[executionID]
	if !exists {
		m.mu.Unlock()
		return
	}

	metrics.Mutex.Lock()
	now := time.Now()
	metrics.EndTime = &now
	metrics.Duration = now.Sub(metrics.StartTime)
	metrics.Status = status

	if status == "completed" {
		for nodeID, nodeStatus := range metrics.NodeStatuses {
			if nodeStatus == NodeStatusRunning || nodeStatus == NodeStatusPending {
				metrics.NodeStatuses[nodeID] = NodeStatusCompleted
			}
		}
	}

	metrics.Mutex.Unlock()

	workflowID := metrics.WorkflowID
	workflowName := metrics.WorkflowName
	nodeStates := m.getNodeStatesSnapshotRL(m.executions, executionID)

	var streamCh chan *StreamStyleEvent
	if ch, ok := m.streamEventChannels[executionID]; ok {
		streamCh = ch
		delete(m.streamEventChannels, executionID)
	}

	m.mu.Unlock()

	if m.eventHandler != nil {
		if status == "completed" {
			m.eventHandler.OnExecutionCompleted(executionID, metrics)
		} else if status == "failed" {
			m.eventHandler.OnExecutionFailed(executionID, metrics, fmt.Errorf("execution failed"))
		}
	}

	eventType := "workflow_completed"
	if status == "failed" {
		eventType = "workflow_failed"
	}
	m.emitEvent(&ExecutionEvent{
		Type:        eventType,
		ExecutionID: executionID,
		Status:      status,
		Timestamp:   time.Now().Unix(),
		NodeStates:  nodeStates,
	})

	m.EmitStreamWorkflowCompleted(executionID, workflowID, workflowName, status)

	if streamCh != nil {
		m.channelMu.Lock()
		close(streamCh)
		m.channelMu.Unlock()
	}

	// 延迟清理执行数据
	go func() {
		time.Sleep(30 * time.Second)
		m.mu.Lock()
		defer m.mu.Unlock()

		if _, exists := m.executions[executionID]; exists {
			delete(m.executions, executionID)
		}

		keysToDelete := []string{}
		for key := range m.nodeMetrics {
			if strings.HasPrefix(key, executionID+":") {
				keysToDelete = append(keysToDelete, key)
			}
		}
		for _, key := range keysToDelete {
			delete(m.nodeMetrics, key)
		}
	}()
}

// NodeStarted 节点开始执行
func (m *ExecutionMonitor) NodeStarted(executionID, nodeID, nodeType string) {
	m.mu.Lock()

	// 更新执行指标
	execMetrics, exists := m.executions[executionID]
	if exists {
		execMetrics.Mutex.Lock()
		execMetrics.RunningNodes++
		execMetrics.NodeStatuses[nodeID] = NodeStatusRunning
		execMetrics.Mutex.Unlock()
	}

	// 创建节点指标
	nodeMetrics := &NodeMetrics{
		NodeID:      nodeID,
		NodeType:    nodeType,
		ExecutionID: executionID,
		Status:      NodeStatusRunning,
		StartTime:   time.Now(),
	}

	m.nodeMetrics[executionID+":"+nodeID] = nodeMetrics

	// 获取工作流信息（在释放锁之前）
	var workflowID, workflowName string
	if metrics, exists := m.executions[executionID]; exists {
		workflowID = metrics.WorkflowID
		workflowName = metrics.WorkflowName
	}

	// 获取节点状态快照（在释放锁之前）
	nodeStates := m.getNodeStatesSnapshotRL(m.executions, executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnNodeStarted(executionID, nodeID, nodeType)
	}

	// 推送 node_started 事件 (向后兼容)
	m.emitEvent(&ExecutionEvent{
		Type:        "node_started",
		ExecutionID: executionID,
		NodeID:      nodeID,
		NodeType:    nodeType,
		Status:      "running",
		Timestamp:   time.Now().Unix(),
		NodeStates:  nodeStates,
	})

	// 推送流式风格事件
	m.EmitStreamNodeStarted(executionID, workflowID, workflowName, nodeID, nodeType)
}

// NodeCompleted 节点执行完成
func (m *ExecutionMonitor) NodeCompleted(executionID, nodeID string, result *models.NodeResult) {
	m.mu.Lock()

	var duration time.Duration

	// 更新执行指标
	execMetrics, exists := m.executions[executionID]
	if exists {
		execMetrics.Mutex.Lock()
		execMetrics.RunningNodes--
		execMetrics.CompletedNodes++
		execMetrics.NodeStatuses[nodeID] = NodeStatusCompleted
		execMetrics.Mutex.Unlock()
	}

	// 更新节点指标
	key := executionID + ":" + nodeID
	nodeMetrics, exists := m.nodeMetrics[key]
	if exists {
		nodeMetrics.Mutex.Lock()
		now := time.Now()
		nodeMetrics.EndTime = &now
		nodeMetrics.Duration = now.Sub(nodeMetrics.StartTime)
		nodeMetrics.Status = NodeStatusCompleted
		duration = nodeMetrics.Duration
		nodeMetrics.Mutex.Unlock()
	}

	// 获取工作流信息（在释放锁之前）
	var workflowID, workflowName, nodeType string
	if metrics, exists := m.executions[executionID]; exists {
		workflowID = metrics.WorkflowID
		workflowName = metrics.WorkflowName
	}
	if nodeMetrics, exists := m.nodeMetrics[executionID+":"+nodeID]; exists {
		nodeType = nodeMetrics.NodeType
	}

	// 获取节点状态快照（在释放锁之前）
	nodeStates := m.getNodeStatesSnapshotRL(m.executions, executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnNodeCompleted(executionID, nodeID, result)
	}

	// 推送 node_completed 事件 (向后兼容)
	eventOutputs := make(map[string]any)
	if result != nil {
		eventOutputs = result.Outputs
	}
	m.emitEvent(&ExecutionEvent{
		Type:        "node_completed",
		ExecutionID: executionID,
		NodeID:      nodeID,
		Status:      "running",
		Outputs:     eventOutputs,
		Timestamp:   time.Now().Unix(),
		NodeStates:  nodeStates,
	})

	// 推送流式风格事件
	m.EmitStreamNodeCompleted(executionID, workflowID, workflowName, nodeID, nodeType, eventOutputs, duration.Seconds())
}

// NodeFailed 节点执行失败
func (m *ExecutionMonitor) NodeFailed(executionID, nodeID string, err error) {
	m.mu.Lock()

	// 更新执行指标
	execMetrics, exists := m.executions[executionID]
	if exists {
		execMetrics.Mutex.Lock()
		execMetrics.RunningNodes--
		execMetrics.FailedNodes++
		execMetrics.NodeStatuses[nodeID] = NodeStatusFailed
		execMetrics.Mutex.Unlock()
	}

	// 更新节点指标
	key := executionID + ":" + nodeID
	nodeMetrics, exists := m.nodeMetrics[key]
	if exists {
		nodeMetrics.Mutex.Lock()
		now := time.Now()
		nodeMetrics.EndTime = &now
		nodeMetrics.Duration = now.Sub(nodeMetrics.StartTime)
		nodeMetrics.Status = NodeStatusFailed
		nodeMetrics.Error = err.Error()
		nodeMetrics.Mutex.Unlock()
	}

	// 获取工作流信息（在释放锁之前）
	var workflowID, workflowName, nodeType string
	var duration float64
	if metrics, exists := m.executions[executionID]; exists {
		workflowID = metrics.WorkflowID
		workflowName = metrics.WorkflowName
	}
	if nodeMetrics, exists := m.nodeMetrics[executionID+":"+nodeID]; exists {
		nodeType = nodeMetrics.NodeType
		duration = nodeMetrics.Duration.Seconds()
	}

	// 获取节点状态快照（在释放锁之前）
	nodeStates := m.getNodeStatesSnapshotRL(m.executions, executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnNodeFailed(executionID, nodeID, err)
	}

	// 推送 node_failed 事件 (向后兼容)
	m.emitEvent(&ExecutionEvent{
		Type:        "node_failed",
		ExecutionID: executionID,
		NodeID:      nodeID,
		Status:      "running",
		Error:       err.Error(),
		Timestamp:   time.Now().Unix(),
		NodeStates:  nodeStates,
	})

	// 推送流式风格事件
	m.EmitStreamNodeFailed(executionID, workflowID, workflowName, nodeID, nodeType, err.Error(), duration)
}

// NodeRetry 节点重试
func (m *ExecutionMonitor) NodeRetry(executionID, nodeID string, attempt int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 更新节点指标
	key := executionID + ":" + nodeID
	nodeMetrics, exists := m.nodeMetrics[key]
	if exists {
		nodeMetrics.Mutex.Lock()
		nodeMetrics.Retries = attempt
		nodeMetrics.Mutex.Unlock()
	}

	if m.eventHandler != nil {
		m.eventHandler.OnNodeRetry(executionID, nodeID, attempt, err)
	}
}

// GetExecutionMetrics 获取执行指标
func (m *ExecutionMonitor) GetExecutionMetrics(executionID string) (*ExecutionMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metrics, exists := m.executions[executionID]
	if !exists {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	// 返回副本
	metrics.Mutex.RLock()
	defer metrics.Mutex.RUnlock()

	copy := &ExecutionMetrics{
		ExecutionID:    metrics.ExecutionID,
		WorkflowID:     metrics.WorkflowID,
		WorkflowName:   metrics.WorkflowName,
		Status:         metrics.Status,
		StartTime:      metrics.StartTime,
		EndTime:        metrics.EndTime,
		Duration:       metrics.Duration,
		TotalNodes:     metrics.TotalNodes,
		CompletedNodes: metrics.CompletedNodes,
		FailedNodes:    metrics.FailedNodes,
		RunningNodes:   metrics.RunningNodes,
		NodeStatuses:   make(map[string]NodeStatus),
	}

	for k, v := range metrics.NodeStatuses {
		copy.NodeStatuses[k] = v
	}

	return copy, nil
}

// GetNodeMetrics 获取节点指标
func (m *ExecutionMonitor) GetNodeMetrics(executionID, nodeID string) (*NodeMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := executionID + ":" + nodeID
	metrics, exists := m.nodeMetrics[key]
	if !exists {
		return nil, fmt.Errorf("node metrics not found: %s", key)
	}

	// 返回副本
	metrics.Mutex.RLock()
	defer metrics.Mutex.RUnlock()

	copy := &NodeMetrics{
		NodeID:      metrics.NodeID,
		NodeType:    metrics.NodeType,
		ExecutionID: metrics.ExecutionID,
		Status:      metrics.Status,
		StartTime:   metrics.StartTime,
		EndTime:     metrics.EndTime,
		Duration:    metrics.Duration,
		Retries:     metrics.Retries,
		Error:       metrics.Error,
	}

	return copy, nil
}

// GetAllExecutions 获取所有执行指标
func (m *ExecutionMonitor) GetAllExecutions() map[string]*ExecutionMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*ExecutionMetrics)
	for id, metrics := range m.executions {
		metrics.Mutex.RLock()
		copy := &ExecutionMetrics{
			ExecutionID:    metrics.ExecutionID,
			WorkflowID:     metrics.WorkflowID,
			WorkflowName:   metrics.WorkflowName,
			Status:         metrics.Status,
			StartTime:      metrics.StartTime,
			EndTime:        metrics.EndTime,
			Duration:       metrics.Duration,
			TotalNodes:     metrics.TotalNodes,
			CompletedNodes: metrics.CompletedNodes,
			FailedNodes:    metrics.FailedNodes,
			RunningNodes:   metrics.RunningNodes,
			NodeStatuses:   make(map[string]NodeStatus),
		}
		for k, v := range metrics.NodeStatuses {
			copy.NodeStatuses[k] = v
		}
		metrics.Mutex.RUnlock()
		result[id] = copy
	}

	return result
}

// Clear 清空监控数据
func (m *ExecutionMonitor) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executions = make(map[string]*ExecutionMetrics)
	m.nodeMetrics = make(map[string]*NodeMetrics)
}

// DefaultEventHandler 实现

func (h *DefaultEventHandler) OnExecutionStarted(executionID string)                              {}
func (h *DefaultEventHandler) OnExecutionCompleted(executionID string, metrics *ExecutionMetrics) {}
func (h *DefaultEventHandler) OnExecutionFailed(executionID string, metrics *ExecutionMetrics, err error) {
}
func (h *DefaultEventHandler) OnNodeStarted(executionID, nodeID, nodeType string) {}
func (h *DefaultEventHandler) OnNodeCompleted(executionID, nodeID string, result *models.NodeResult) {
}
func (h *DefaultEventHandler) OnNodeFailed(executionID, nodeID string, err error)             {}
func (h *DefaultEventHandler) OnNodeRetry(executionID, nodeID string, attempt int, err error) {}
