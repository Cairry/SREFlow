package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// ExecutionMonitor 执行监控器
type ExecutionMonitor struct {
	mu          sync.RWMutex
	executions  map[string]*ExecutionMetrics
	nodeMetrics map[string]*NodeMetrics

	// SSE 事件通道管理
	channelMu           sync.Mutex
	streamEventChannels map[string]*streamChannelState
	eventHandler        EventHandler
}

// streamChannelState 封装 channel 及其状态，防止向已关闭 channel 发送
type streamChannelState struct {
	ch     chan *StreamStyleEvent
	closed bool
	mu     sync.RWMutex
}

func newStreamChannelState(bufferSize int) *streamChannelState {
	return &streamChannelState{
		ch:     make(chan *StreamStyleEvent, bufferSize),
		closed: false,
	}
}

// send 安全发送事件，返回是否成功
func (s *streamChannelState) send(event *StreamStyleEvent) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return false
	}

	select {
	case s.ch <- event:
		return true
	default:
		logc.Errorf(context.Background(), "[Monitor] channel full, event %s dropped", event.Event)
		return false
	}
}

// close 安全关闭 channel
func (s *streamChannelState) close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.ch)
	}
}

// StreamStyleEvent 流式SSE事件
type StreamStyleEvent struct {
	ID       string       `json:"id"`
	Event    string       `json:"event"`
	TaskID   string       `json:"task_id"`
	Workflow WorkflowInfo `json:"workflow"`
	Node     *NodeInfo    `json:"node,omitempty"`
	Data     EventData    `json:"data"`
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
	Outputs   map[string]string `json:"outputs,omitempty"`
	Error     string            `json:"error,omitempty"`
	ExeTime   string            `json:"elapsed_time,omitempty"`
}

// EventData 事件数据
type EventData struct {
	ID              string            `json:"id"`
	WorkflowID      string            `json:"workflow_id"`
	WorkflowRunID   string            `json:"workflow_run_id"`
	Status          string            `json:"status"`
	Error           string            `json:"error,omitempty"`
	NodeExecutionID string            `json:"node_execution_id,omitempty"`
	NodeID          string            `json:"node_id,omitempty"`
	NodeType        string            `json:"node_type,omitempty"`
	NodeTitle       string            `json:"node_title,omitempty"`
	Outputs         map[string]string `json:"outputs,omitempty"`
	NodeStates      map[string]string `json:"node_states,omitempty"`
	TotalNodes      int               `json:"total_nodes,omitempty"`
	CompletedNodes  int               `json:"completed_nodes,omitempty"`
	Duration        float64           `json:"duration,omitempty"`
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

func (h *DefaultEventHandler) OnExecutionStarted(executionID string)                              {}
func (h *DefaultEventHandler) OnExecutionCompleted(executionID string, metrics *ExecutionMetrics) {}
func (h *DefaultEventHandler) OnExecutionFailed(executionID string, metrics *ExecutionMetrics, err error) {
}
func (h *DefaultEventHandler) OnNodeStarted(executionID, nodeID, nodeType string) {}
func (h *DefaultEventHandler) OnNodeCompleted(executionID, nodeID string, result *models.NodeResult) {
}
func (h *DefaultEventHandler) OnNodeFailed(executionID, nodeID string, err error)             {}
func (h *DefaultEventHandler) OnNodeRetry(executionID, nodeID string, attempt int, err error) {}

// NewExecutionMonitor 创建新的执行监控器
func NewExecutionMonitor() *ExecutionMonitor {
	return &ExecutionMonitor{
		executions:          make(map[string]*ExecutionMetrics),
		nodeMetrics:         make(map[string]*NodeMetrics),
		streamEventChannels: make(map[string]*streamChannelState),
		eventHandler:        &DefaultEventHandler{},
	}
}

// RegisterStreamChannel 注册流式 SSE 事件通道
func (m *ExecutionMonitor) RegisterStreamChannel(executionID string) chan *StreamStyleEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existingState, exists := m.streamEventChannels[executionID]; exists && existingState != nil {
		return existingState.ch
	}

	state := newStreamChannelState(100)
	m.streamEventChannels[executionID] = state
	return state.ch
}

// GetStreamChannel 获取流式事件通道
func (m *ExecutionMonitor) GetStreamChannel(executionID string) chan *StreamStyleEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if state, exists := m.streamEventChannels[executionID]; exists {
		return state.ch
	}
	return nil
}

// UnregisterStreamChannel 取消注册流式 SSE 事件通道
func (m *ExecutionMonitor) UnregisterStreamChannel(executionID string) {
	m.mu.Lock()
	state, exists := m.streamEventChannels[executionID]
	if !exists {
		m.mu.Unlock()
		return
	}
	delete(m.streamEventChannels, executionID)
	m.mu.Unlock()

	if state != nil {
		state.close()
	}
}

// emitStreamEvent 统一的流式事件发送方法
func (m *ExecutionMonitor) emitStreamEvent(event *StreamStyleEvent) {
	m.mu.RLock()
	state, exists := m.streamEventChannels[event.TaskID]
	m.mu.RUnlock()

	if !exists || state == nil {
		return
	}

	if !state.send(event) {
		logc.Errorf(context.Background(), "[Monitor] failed to send event %s for execution %s", event.Event, event.TaskID)
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

// getNodeStatesSnapshotRL 读取节点状态快照（调用方须持有 mu 锁）
func (m *ExecutionMonitor) getNodeStatesSnapshotRL(executionID string) map[string]string {
	states := make(map[string]string)
	if metrics, exists := m.executions[executionID]; exists {
		for nodeID, status := range metrics.NodeStatuses {
			states[nodeID] = string(status)
		}
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
func (m *ExecutionMonitor) StartExecution(executionID, workflowID, workflowName string, isolatedNodes []string) {
	m.mu.Lock()

	metrics := &ExecutionMetrics{
		ExecutionID:  executionID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		StartTime:    time.Now(),
		Status:       "running",
		NodeStatuses: make(map[string]NodeStatus),
	}

	for _, nodeID := range isolatedNodes {
		metrics.NodeStatuses[nodeID] = NodeStatusSkipped
	}

	m.executions[executionID] = metrics
	nodeStates := m.getNodeStatesSnapshotRL(executionID)
	totalNodes := len(metrics.NodeStatuses)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnExecutionStarted(executionID)
	}

	event := NewStreamStyleEvent("running", executionID, workflowID, workflowName)
	event.Data.NodeStates = nodeStates
	event.Data.TotalNodes = totalNodes
	m.emitStreamEvent(event)
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
	nodeStates := m.getNodeStatesSnapshotRL(executionID)
	duration := metrics.Duration.Seconds()
	totalNodes := metrics.TotalNodes
	completedNodes := metrics.CompletedNodes

	var streamState *streamChannelState
	if ch, ok := m.streamEventChannels[executionID]; ok {
		streamState = ch
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

	event := NewStreamStyleEvent(eventType, executionID, workflowID, workflowName)
	event.Data.NodeStates = nodeStates
	event.Data.TotalNodes = totalNodes
	event.Data.CompletedNodes = completedNodes
	event.Data.Duration = duration
	m.emitStreamEvent(event)

	if streamState != nil {
		time.Sleep(10 * time.Millisecond)
		streamState.close()

		m.mu.Lock()
		delete(m.streamEventChannels, executionID)
		m.mu.Unlock()
	}

	go func() {
		time.Sleep(30 * time.Second)
		m.mu.Lock()
		defer m.mu.Unlock()

		delete(m.executions, executionID)

		for key := range m.nodeMetrics {
			if strings.HasPrefix(key, executionID+":") {
				delete(m.nodeMetrics, key)
			}
		}
	}()
}

// NodeStarted 节点开始执行
func (m *ExecutionMonitor) NodeStarted(executionID, nodeID, nodeType string) {
	m.mu.Lock()

	if execMetrics, exists := m.executions[executionID]; exists {
		execMetrics.Mutex.Lock()
		execMetrics.RunningNodes++
		execMetrics.NodeStatuses[nodeID] = NodeStatusRunning
		execMetrics.Mutex.Unlock()
	}

	m.nodeMetrics[executionID+":"+nodeID] = &NodeMetrics{
		NodeID:      nodeID,
		NodeType:    nodeType,
		ExecutionID: executionID,
		Status:      NodeStatusRunning,
		StartTime:   time.Now(),
	}

	var workflowID, workflowName string
	if metrics, exists := m.executions[executionID]; exists {
		workflowID = metrics.WorkflowID
		workflowName = metrics.WorkflowName
	}

	nodeStates := m.getNodeStatesSnapshotRL(executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnNodeStarted(executionID, nodeID, nodeType)
	}

	event := NewStreamNodeEvent("node_started", executionID, workflowID, workflowName, nodeID, nodeType, nil, "", 0)
	event.Data.NodeStates = nodeStates
	m.emitStreamEvent(event)
}

// NodeCompleted 节点执行完成
func (m *ExecutionMonitor) NodeCompleted(executionID, nodeID string, result *models.NodeResult) {
	m.mu.Lock()

	var duration time.Duration

	if execMetrics, exists := m.executions[executionID]; exists {
		execMetrics.Mutex.Lock()
		execMetrics.RunningNodes--
		execMetrics.CompletedNodes++
		execMetrics.NodeStatuses[nodeID] = NodeStatusCompleted
		execMetrics.Mutex.Unlock()
	}

	key := executionID + ":" + nodeID
	if nodeMetrics, exists := m.nodeMetrics[key]; exists {
		nodeMetrics.Mutex.Lock()
		now := time.Now()
		nodeMetrics.EndTime = &now
		nodeMetrics.Duration = now.Sub(nodeMetrics.StartTime)
		nodeMetrics.Status = NodeStatusCompleted
		duration = nodeMetrics.Duration
		nodeMetrics.Mutex.Unlock()
	}

	var workflowID, workflowName, nodeType string
	if metrics, exists := m.executions[executionID]; exists {
		workflowID = metrics.WorkflowID
		workflowName = metrics.WorkflowName
	}
	if nm, exists := m.nodeMetrics[key]; exists {
		nodeType = nm.NodeType
	}

	nodeStates := m.getNodeStatesSnapshotRL(executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnNodeCompleted(executionID, nodeID, result)
	}

	outputs := make(map[string]any)
	if result != nil {
		outputs = result.Outputs
	}

	stringifiedOutputs := make(map[string]string)
	for k, v := range outputs {
		stringifiedOutputs[k] = stringifyValue(v)
	}

	event := NewStreamNodeEvent("node_completed", executionID, workflowID, workflowName, nodeID, nodeType, nil, "", duration.Seconds())
	event.Node.Outputs = stringifiedOutputs
	event.Data.NodeStates = nodeStates

	m.mu.RLock()
	if metrics, exists := m.executions[executionID]; exists {
		event.Data.TotalNodes = metrics.TotalNodes
		event.Data.CompletedNodes = metrics.CompletedNodes
	}
	m.mu.RUnlock()

	m.emitStreamEvent(event)
}

// NodeFailed 节点执行失败
func (m *ExecutionMonitor) NodeFailed(executionID, nodeID string, err error) {
	m.mu.Lock()

	errMsg := "unknown error"
	if err != nil {
		errMsg = err.Error()
	}

	var duration time.Duration

	if execMetrics, exists := m.executions[executionID]; exists {
		execMetrics.Mutex.Lock()
		execMetrics.RunningNodes--
		execMetrics.FailedNodes++
		execMetrics.NodeStatuses[nodeID] = NodeStatusFailed
		execMetrics.Mutex.Unlock()
	}

	key := executionID + ":" + nodeID
	if nodeMetrics, exists := m.nodeMetrics[key]; exists {
		nodeMetrics.Mutex.Lock()
		now := time.Now()
		nodeMetrics.EndTime = &now
		nodeMetrics.Duration = now.Sub(nodeMetrics.StartTime)
		nodeMetrics.Status = NodeStatusFailed
		nodeMetrics.Error = errMsg
		duration = nodeMetrics.Duration
		nodeMetrics.Mutex.Unlock()
	}

	var workflowID, workflowName, nodeType string
	if metrics, exists := m.executions[executionID]; exists {
		workflowID = metrics.WorkflowID
		workflowName = metrics.WorkflowName
	}
	if nm, exists := m.nodeMetrics[key]; exists {
		nodeType = nm.NodeType
	}

	nodeStates := m.getNodeStatesSnapshotRL(executionID)

	m.mu.Unlock()

	if m.eventHandler != nil {
		m.eventHandler.OnNodeFailed(executionID, nodeID, err)
	}

	event := NewStreamNodeEvent("node_failed", executionID, workflowID, workflowName, nodeID, nodeType, nil, errMsg, duration.Seconds())
	event.Data.Error = errMsg
	event.Data.NodeStates = nodeStates
	m.emitStreamEvent(event)
}

// NodeRetry 节点重试
func (m *ExecutionMonitor) NodeRetry(executionID, nodeID string, attempt int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := executionID + ":" + nodeID
	if nodeMetrics, exists := m.nodeMetrics[key]; exists {
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

// ========== 流式事件工具方法 ==========

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
		return "running"
	case "node_failed":
		return "failed"
	case "workflow_completed":
		return "succeeded"
	case "workflow_failed":
		return "failed"
	default:
		return "running"
	}
}

// EmitStreamWorkflowStarted 发送工作流开始事件
func (m *ExecutionMonitor) EmitStreamWorkflowStarted(executionID, workflowID, workflowName string) {
	event := NewStreamStyleEvent("running", executionID, workflowID, workflowName)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamNodeStarted 发送节点开始事件
func (m *ExecutionMonitor) EmitStreamNodeStarted(executionID, workflowID, workflowName, nodeID, nodeType string) {
	event := NewStreamNodeEvent("node_started", executionID, workflowID, workflowName, nodeID, nodeType, nil, "", 0)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamNodeCompleted 发送节点完成事件
func (m *ExecutionMonitor) EmitStreamNodeCompleted(executionID, workflowID, workflowName, nodeID, nodeType string, outputs map[string]any, duration float64) {
	stringifiedOutputs := make(map[string]string)
	for k, v := range outputs {
		stringifiedOutputs[k] = stringifyValue(v)
	}

	event := NewStreamNodeEvent("node_completed", executionID, workflowID, workflowName, nodeID, nodeType, nil, "", duration)
	event.Node.Outputs = stringifiedOutputs
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)

	m.mu.RLock()
	if metrics, exists := m.executions[executionID]; exists {
		event.Data.TotalNodes = metrics.TotalNodes
		event.Data.CompletedNodes = metrics.CompletedNodes
	}
	m.mu.RUnlock()

	m.emitStreamEvent(event)
}

// EmitStreamNodeFailed 发送节点失败事件
func (m *ExecutionMonitor) EmitStreamNodeFailed(executionID, workflowID, workflowName, nodeID, nodeType string, errMsg string, duration float64) {
	event := NewStreamNodeEvent("node_failed", executionID, workflowID, workflowName, nodeID, nodeType, nil, errMsg, duration)
	event.Data.Error = errMsg
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)
	m.emitStreamEvent(event)
}

// EmitStreamNodeSkipped 发送节点跳过事件
func (m *ExecutionMonitor) EmitStreamNodeSkipped(executionID, workflowID, workflowName, nodeID, nodeType string) {
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

// EmitStreamWorkflowCompleted 发送工作流完成事件
func (m *ExecutionMonitor) EmitStreamWorkflowCompleted(executionID, workflowID, workflowName string, status string) {
	eventType := "workflow_completed"
	if status == "failed" {
		eventType = "workflow_failed"
	}

	event := NewStreamStyleEvent(eventType, executionID, workflowID, workflowName)
	event.Data.NodeStates = m.getNodeStatesSnapshot(executionID)

	m.mu.RLock()
	if metrics, exists := m.executions[executionID]; exists {
		event.Data.TotalNodes = metrics.TotalNodes
		event.Data.CompletedNodes = metrics.CompletedNodes
		event.Data.Duration = metrics.Duration.Seconds()
	}
	m.mu.RUnlock()

	m.emitStreamEvent(event)
}
