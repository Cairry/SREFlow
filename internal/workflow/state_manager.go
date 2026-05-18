package workflow

import (
	"fmt"
	"sync"
	"time"

	"RCAgent/internal/models"
)

// NodeState 节点状态枚举
type NodeState string

const (
	NodeStateIdle      NodeState = "idle"
	NodeStatePending   NodeState = "pending"
	NodeStateReady     NodeState = "ready"
	NodeStateRunning   NodeState = "running"
	NodeStateCompleted NodeState = "completed"
	NodeStateFailed    NodeState = "failed"
	NodeStateSkipped   NodeState = "skipped"
	NodeStateCancelled NodeState = "cancelled"
)

// NodeStateTransition 状态转换规则
var stateTransitions = map[NodeState][]NodeState{
	NodeStateIdle:      {NodeStatePending},
	NodeStatePending:   {NodeStateRunning, NodeStateCancelled, NodeStateSkipped},
	NodeStateRunning:   {NodeStateCompleted, NodeStateFailed, NodeStateCancelled},
	NodeStateCompleted: {},                 // 终态
	NodeStateFailed:    {NodeStatePending}, // 可以重试
	NodeStateSkipped:   {},                 // 终态
	NodeStateCancelled: {},                 // 终态
}

// NodeStateInfo 节点状态信息
type NodeStateInfo struct {
	NodeID    string                 `json:"nodeId"`
	NodeType  string                 `json:"nodeType"`
	State     NodeState              `json:"state"`
	StartTime *time.Time             `json:"startTime,omitempty"`
	EndTime   *time.Time             `json:"endTime,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Result    *models.NodeResult     `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Retries   int                    `json:"retries"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Mutex     sync.RWMutex           `json:"-"`
}

// WorkflowState 工作流状态
type WorkflowState struct {
	ExecutionID    string                        `json:"executionId"`
	WorkflowID     string                        `json:"workflowId"`
	WorkflowName   string                        `json:"workflowName"`
	State          string                        `json:"state"` // "pending", "running", "completed", "failed", "cancelled"
	StartTime      time.Time                     `json:"startTime"`
	EndTime        *time.Time                    `json:"endTime,omitempty"`
	Duration       time.Duration                 `json:"duration"`
	TotalNodes     int                           `json:"totalNodes"`
	NodeStates     map[string]*NodeStateInfo     `json:"nodeStates"`
	NodeResults    map[string]*models.NodeResult `json:"nodeResults"` // 节点执行结果
	GlobalMetadata map[string]interface{}        `json:"globalMetadata,omitempty"`
	Mutex          sync.RWMutex                  `json:"-"`
}

// StateManager 状态管理器
type StateManager struct {
	workflows map[string]*WorkflowState
	mu        sync.RWMutex
}

// NewStateManager 创建状态管理器
func NewStateManager() *StateManager {
	return &StateManager{
		workflows: make(map[string]*WorkflowState),
	}
}

// CreateWorkflowState 创建工作流状态
func (sm *StateManager) CreateWorkflowState(executionID, workflowID, workflowName string, totalNodes int) *WorkflowState {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state := &WorkflowState{
		ExecutionID:    executionID,
		WorkflowID:     workflowID,
		WorkflowName:   workflowName,
		State:          "pending",
		StartTime:      time.Now(),
		TotalNodes:     totalNodes,
		NodeStates:     make(map[string]*NodeStateInfo),
		NodeResults:    make(map[string]*models.NodeResult),
		GlobalMetadata: make(map[string]interface{}),
	}

	sm.workflows[executionID] = state
	return state
}

// SetWorkflowNodeResults 设置工作流的节点执行结果
func (sm *StateManager) SetWorkflowNodeResults(executionID string, results map[string]*models.NodeResult) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.workflows[executionID]
	if !exists {
		return
	}

	state.Mutex.Lock()
	defer state.Mutex.Unlock()

	state.NodeResults = results
}

// GetWorkflowNodeResults 获取工作流的节点执行结果
func (sm *StateManager) GetWorkflowNodeResults(executionID string) map[string]*models.NodeResult {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.workflows[executionID]
	if !exists {
		return nil
	}

	state.Mutex.RLock()
	defer state.Mutex.RUnlock()

	return state.NodeResults
}

// GetWorkflowState 获取工作流状态
func (sm *StateManager) GetWorkflowState(executionID string) (*WorkflowState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.workflows[executionID]
	if !exists {
		return nil, fmt.Errorf("workflow state not found: %s", executionID)
	}

	// 返回副本
	return sm.cloneWorkflowState(state), nil
}

// CheckWorkflowState 检查工作流状态是否存在（不返回副本，仅检查存在性）
func (sm *StateManager) CheckWorkflowState(executionID string) (*WorkflowState, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.workflows[executionID]
	return state, exists
}

// UpdateWorkflowState 更新工作流状态
func (sm *StateManager) UpdateWorkflowState(executionID, newState string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.workflows[executionID]
	if !exists {
		return fmt.Errorf("workflow state not found: %s", executionID)
	}

	state.Mutex.Lock()
	defer state.Mutex.Unlock()

	// 工作流状态转换：pending -> running -> completed/failed
	state.State = newState
	if newState == "completed" || newState == "failed" || newState == "cancelled" {
		now := time.Now()
		state.EndTime = &now
		state.Duration = now.Sub(state.StartTime)

		// 工作流完成时，将所有未明确标记的节点状态设为 completed
		for _, nodeState := range state.NodeStates {
			nodeState.Mutex.Lock()
			if nodeState.State == NodeStatePending || nodeState.State == NodeStateReady {
				nodeState.State = NodeStateCompleted
				nodeState.EndTime = &now
				nodeState.Duration = now.Sub(state.StartTime)
			}
			nodeState.Mutex.Unlock()
		}
	}

	return nil
}

// SetNodeState 设置节点状态
func (sm *StateManager) SetNodeState(executionID, nodeID, nodeType string, newState NodeState, result *models.NodeResult, errorMsg string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	workflowState, exists := sm.workflows[executionID]
	if !exists {
		return fmt.Errorf("workflow state not found: %s", executionID)
	}

	workflowState.Mutex.Lock()
	defer workflowState.Mutex.Unlock()

	nodeState, nodeExists := workflowState.NodeStates[nodeID]
	if !nodeExists {
		nodeState = &NodeStateInfo{
			NodeID:   nodeID,
			NodeType: nodeType,
			State:    NodeStateIdle,
			Metadata: make(map[string]interface{}),
		}
		workflowState.NodeStates[nodeID] = nodeState
	}

	if err := sm.validateNodeStateTransition(nodeState.State, newState); err != nil {
		return err
	}

	nodeState.Mutex.Lock()
	defer nodeState.Mutex.Unlock()

	now := time.Now()

	switch newState {
	case NodeStateRunning:
		if nodeState.StartTime == nil {
			nodeState.StartTime = &now
		}
	case NodeStateCompleted, NodeStateFailed, NodeStateSkipped, NodeStateCancelled:
		if nodeState.StartTime != nil && nodeState.EndTime == nil {
			nodeState.EndTime = &now
			nodeState.Duration = now.Sub(*nodeState.StartTime)
		}
	}

	nodeState.State = newState
	if result != nil {
		nodeState.Result = result
	}
	if errorMsg != "" {
		nodeState.Error = errorMsg
	}

	return nil
}

// validateNodeStateTransition 验证节点状态转换
func (sm *StateManager) validateNodeStateTransition(from, to NodeState) error {
	allowedStates, valid := stateTransitions[from]
	if !valid {
		return fmt.Errorf("invalid current state: %s", from)
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return nil
		}
	}

	return fmt.Errorf("invalid state transition from %s to %s", from, to)
}

// GetNodeState 获取节点状态
func (sm *StateManager) GetNodeState(executionID, nodeID string) (*NodeStateInfo, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	workflowState, exists := sm.workflows[executionID]
	if !exists {
		return nil, fmt.Errorf("workflow state not found: %s", executionID)
	}

	workflowState.Mutex.RLock()
	defer workflowState.Mutex.RUnlock()

	nodeState, exists := workflowState.NodeStates[nodeID]
	if !exists {
		return nil, fmt.Errorf("node state not found: %s/%s", executionID, nodeID)
	}

	// 返回副本
	return sm.cloneNodeState(nodeState), nil
}

// GetWorkflowSnapshot 获取工作流状态快照
func (sm *StateManager) GetWorkflowSnapshot(executionID string) map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	workflowState, exists := sm.workflows[executionID]
	if !exists {
		return map[string]interface{}{
			"executionId":    executionID,
			"status":         "not_found",
			"runningNodes":   []string{},
			"completedNodes": []string{},
			"failedNodes":    []string{},
			"pendingNodes":   []string{},
		}
	}

	workflowState.Mutex.RLock()
	defer workflowState.Mutex.RUnlock()

	runningNodes := []string{}
	completedNodes := []string{}
	failedNodes := []string{}
	pendingNodes := []string{}

	for nodeID, nodeState := range workflowState.NodeStates {
		nodeState.Mutex.RLock()
		switch nodeState.State {
		case NodeStateRunning:
			runningNodes = append(runningNodes, nodeID)
		case NodeStateCompleted:
			completedNodes = append(completedNodes, nodeID)
		case NodeStateFailed:
			failedNodes = append(failedNodes, nodeID)
		case NodeStatePending, NodeStateReady:
			pendingNodes = append(pendingNodes, nodeID)
		}
		nodeState.Mutex.RUnlock()
	}

	// 工作流完成但节点状态为空时，使用 workflowState.NodeStates 的键推断节点
	// （处理边界情况：某些快速完成的执行可能没有逐个更新节点状态）
	if workflowState.State == "completed" && len(completedNodes) == 0 && len(failedNodes) == 0 {
		for nodeID := range workflowState.NodeStates {
			completedNodes = append(completedNodes, nodeID)
		}
	}

	snapshot := map[string]interface{}{
		"executionId":    workflowState.ExecutionID,
		"workflowId":     workflowState.WorkflowID,
		"workflowName":   workflowState.WorkflowName,
		"status":         workflowState.State,
		"startedAt":      workflowState.StartTime.Unix(),
		"runningNodes":   runningNodes,
		"completedNodes": completedNodes,
		"failedNodes":    failedNodes,
		"pendingNodes":   pendingNodes,
		"totalNodes":     workflowState.TotalNodes,
	}

	if workflowState.EndTime != nil {
		snapshot["finishedAt"] = workflowState.EndTime.Unix()
	}

	return snapshot
}

// CleanupWorkflowState 清理工作流状态
func (sm *StateManager) CleanupWorkflowState(executionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if _, exists := sm.workflows[executionID]; exists {
		delete(sm.workflows, executionID)
	}
}

// CleanupExpiredStates 清理过期状态
func (sm *StateManager) CleanupExpiredStates(maxAge time.Duration) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expiredIDs := []string{}

	for executionID, workflowState := range sm.workflows {
		workflowState.Mutex.RLock()
		if workflowState.EndTime != nil && now.Sub(*workflowState.EndTime) > maxAge {
			expiredIDs = append(expiredIDs, executionID)
		}
		workflowState.Mutex.RUnlock()
	}

	for _, executionID := range expiredIDs {
		delete(sm.workflows, executionID)
	}
}

// cloneWorkflowState 克隆工作流状态
func (sm *StateManager) cloneWorkflowState(original *WorkflowState) *WorkflowState {
	original.Mutex.RLock()
	defer original.Mutex.RUnlock()

	clone := &WorkflowState{
		ExecutionID:    original.ExecutionID,
		WorkflowID:     original.WorkflowID,
		WorkflowName:   original.WorkflowName,
		State:          original.State,
		StartTime:      original.StartTime,
		EndTime:        original.EndTime,
		Duration:       original.Duration,
		TotalNodes:     original.TotalNodes,
		NodeStates:     make(map[string]*NodeStateInfo),
		NodeResults:    make(map[string]*models.NodeResult),
		GlobalMetadata: make(map[string]interface{}),
	}

	// 克隆节点状态
	for nodeID, nodeState := range original.NodeStates {
		clone.NodeStates[nodeID] = sm.cloneNodeState(nodeState)
	}

	// 克隆节点结果
	for nodeID, result := range original.NodeResults {
		clone.NodeResults[nodeID] = result
	}

	// 克隆元数据
	for k, v := range original.GlobalMetadata {
		clone.GlobalMetadata[k] = v
	}

	return clone
}

// cloneNodeState 克隆节点状态
func (sm *StateManager) cloneNodeState(original *NodeStateInfo) *NodeStateInfo {
	original.Mutex.RLock()
	defer original.Mutex.RUnlock()

	clone := &NodeStateInfo{
		NodeID:    original.NodeID,
		NodeType:  original.NodeType,
		State:     original.State,
		StartTime: original.StartTime,
		EndTime:   original.EndTime,
		Duration:  original.Duration,
		Result:    original.Result,
		Error:     original.Error,
		Retries:   original.Retries,
		Metadata:  make(map[string]interface{}),
	}

	// 克隆元数据
	for k, v := range original.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}

// GetStatistics 获取统计信息
func (sm *StateManager) GetStatistics(executionID string) map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	workflowState, exists := sm.workflows[executionID]
	if !exists {
		return map[string]interface{}{
			"exists": false,
		}
	}

	workflowState.Mutex.RLock()
	defer workflowState.Mutex.RUnlock()

	stats := map[string]interface{}{
		"exists":      true,
		"executionId": workflowState.ExecutionID,
		"workflowId":  workflowState.WorkflowID,
		"state":       workflowState.State,
		"startTime":   workflowState.StartTime,
		"totalNodes":  workflowState.TotalNodes,
		"nodeCount":   len(workflowState.NodeStates),
		"nodeStates":  make(map[string]int),
	}

	// 统计节点状态
	stateCounts := map[NodeState]int{}
	for _, nodeState := range workflowState.NodeStates {
		nodeState.Mutex.RLock()
		stateCounts[nodeState.State]++
		nodeState.Mutex.RUnlock()
	}

	for state, count := range stateCounts {
		stats["nodeStates"].(map[string]int)[string(state)] = count
	}

	if workflowState.EndTime != nil {
		stats["endTime"] = workflowState.EndTime
		stats["duration"] = workflowState.Duration
	}

	return stats
}
