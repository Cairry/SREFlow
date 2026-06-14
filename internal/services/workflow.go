package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/internal/types"
	"SREFlow/internal/workflow"
	"SREFlow/pkg/license"
	"SREFlow/pkg/tools"
	"SREFlow/pkg/trace"

	"github.com/zeromicro/go-zero/core/logc"
)

// injectStartNodeInput 将 input 参数注入到 Start 节点的 text 字段
// 如果数据库中节点为空且 input 也为空，返回错误
// mode=json 时将 rawBody 作为 text 注入；mode=text 时将 input 作为 text 注入
func injectStartNodeInput(workflowModel *models.Workflow, input string, mode string, rawBody string) error {
	if mode == "json" {
		if rawBody == "" {
			var nodes []map[string]interface{}
			if err := json.Unmarshal([]byte(workflowModel.Nodes), &nodes); err != nil {
				return fmt.Errorf("workflow nodes is empty")
			}
			if len(nodes) == 0 {
				return fmt.Errorf("workflow nodes is empty")
			}
			return nil
		}
	} else {
		if input == "" {
			var nodes []map[string]interface{}
			if err := json.Unmarshal([]byte(workflowModel.Nodes), &nodes); err != nil {
				return fmt.Errorf("workflow nodes is empty")
			}
			if len(nodes) == 0 {
				return fmt.Errorf("workflow nodes is empty")
			}
			return nil
		}
	}

	var nodes []map[string]interface{}
	if err := json.Unmarshal([]byte(workflowModel.Nodes), &nodes); err != nil {
		return fmt.Errorf("failed to parse workflow nodes: %w", err)
	}

	for _, node := range nodes {
		if nodeType, ok := node["type"].(string); ok && nodeType == "Start" {
			if data, ok := node["data"].(map[string]interface{}); ok {
				if mode == "json" {
					data["text"] = rawBody
				} else {
					data["text"] = input
				}
			}
		}
	}

	// 重新序列化节点
	if newNodesJSON, err := json.Marshal(nodes); err == nil {
		workflowModel.Nodes = string(newNodesJSON)
	}

	return nil
}

// NewWorkflowService 工作流服务
type NewWorkflowService struct {
	engine               *workflow.WorkflowEngine
	parser               *workflow.WorkflowParser
	monitor              *workflow.ExecutionMonitor
	stateManager         *workflow.StateManager
	workflowRepo         repos.WorkflowRepoInter
	executionRepo        repos.WorkflowExecutionRepoInter
	versionRepo          repos.WorkflowVersionRepoInter
	executionCancelFuncs sync.Map // executionID -> context.CancelFunc
}

// NewNewWorkflowService 创建工作流服务
func NewNewWorkflowService(config *workflow.EngineConfig, workflowRepo repos.WorkflowRepoInter, executionRepo repos.WorkflowExecutionRepoInter, integrationRepo repos.IntegrationRepoInter, versionRepo repos.WorkflowVersionRepoInter, mcpRepo repos.MCPServerRepoInter, skillRepo repos.SkillRepoInter) *NewWorkflowService {
	engine := workflow.NewWorkflowEngine(config, integrationRepo, mcpRepo, skillRepo)
	parser := workflow.NewWorkflowParser()
	monitor := engine.GetMonitor()
	stateManager := engine.GetStateManager()

	return &NewWorkflowService{
		engine:        engine,
		parser:        parser,
		monitor:       monitor,
		stateManager:  stateManager,
		workflowRepo:  workflowRepo,
		executionRepo: executionRepo,
		versionRepo:   versionRepo,
	}
}

// saveExecutionRecord 保存执行记录到数据库
func (s *NewWorkflowService) saveExecutionRecord(execution *models.WorkflowExecution) error {
	// 序列化节点结果
	nodeResultsJSON, err := json.Marshal(execution.NodeResults)
	if err != nil {
		return fmt.Errorf("failed to marshal node results: %w", err)
	}

	// 序列化变量
	variablesJSON, err := json.Marshal(execution.Variables)
	if err != nil {
		return fmt.Errorf("failed to marshal variables: %w", err)
	}

	// 获取最终输出
	finalOutput := ""
	if output, exists := execution.Variables["finalOutput"]; exists {
		if str, ok := output.(string); ok {
			finalOutput = str
		} else {
			finalOutputJSON, err := json.Marshal(output)
			if err == nil {
				finalOutput = string(finalOutputJSON)
			}
		}
	}

	// 创建执行记录
	record := &models.WorkflowExecutionRecord{
		ID:           execution.ID,
		WorkflowID:   execution.WorkflowID,
		WorkflowName: execution.WorkflowName,
		Status:       execution.Status,
		StartedAt:    execution.StartedAt,
		FinishedAt:   execution.FinishedAt,
		NodeResults:  string(nodeResultsJSON),
		Variables:    string(variablesJSON),
		FinalOutput:  finalOutput,
	}

	// 保存到数据库
	return s.executionRepo.Create(record)
}

// printWorkflowLogs 打印工作流执行日志
func printWorkflowLogs(ctx context.Context, execution *models.WorkflowExecution) {
	logc.Infof(ctx, "[Workflow] Execution ID: %s, Workflow: %s, Status: %s", execution.ID, execution.WorkflowName, execution.Status)

	if len(execution.NodeResults) == 0 {
		logc.Infof(ctx, "[Workflow] No node results recorded.")
		return
	}

	for nodeID, result := range execution.NodeResults {
		outputsJSON, _ := json.Marshal(result.Outputs)
		if result.Error != "" {
			logc.Errorf(ctx, "[Node: %s] Failed, outputs: %s", nodeID, string(outputsJSON))
		} else {
			logc.Infof(ctx, "[Node: %s] OK, outputs: %s", nodeID, string(outputsJSON))
		}
	}
}

// markExecutionFailed 标记执行失败
func (s *NewWorkflowService) markExecutionFailed(execution *models.WorkflowExecution, reason string) {
	execution.Status = "failed"
	execution.FinishedAt = time.Now().Unix()
	execution.Variables["error"] = reason

	// 发送工作流失败事件到SSE流
	s.monitor.EmitStreamWorkflowCompleted(execution.ID, execution.WorkflowID, execution.WorkflowName, "failed")
}

// markExecutionCompleted 标记执行完成
func (s *NewWorkflowService) markExecutionCompleted(execution *models.WorkflowExecution) {
	execution.Status = "completed"
	execution.FinishedAt = time.Now().Unix()

	// 发送工作流完成事件到SSE流
	s.monitor.EmitStreamWorkflowCompleted(execution.ID, execution.WorkflowID, execution.WorkflowName, "completed")
}

// GetWorkflowExecution 获取工作流执行结果
func (s *NewWorkflowService) GetWorkflowExecution(ctx context.Context, executionID string) (*models.WorkflowExecution, error) {
	logc.Infof(ctx, "[WorkflowService] Getting workflow execution: %s", executionID)

	// 首先从状态管理器获取节点结果
	nodeResults := s.stateManager.GetWorkflowNodeResults(executionID)

	// 尝试从监控器获取基本指标
	metrics, err := s.monitor.GetExecutionMetrics(executionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	// 如果状态管理器中没有节点结果，尝试从数据库获取
	if len(nodeResults) == 0 {
		record, dbErr := s.executionRepo.GetByID(executionID)
		if dbErr == nil && record.NodeResults != "" {
			if err := json.Unmarshal([]byte(record.NodeResults), &nodeResults); err != nil {
				logc.Errorf(ctx, "[WorkflowService] Failed to parse node results from DB: %v", err)
				nodeResults = make(map[string]*models.NodeResult)
			}
		}
	}

	execution := &models.WorkflowExecution{
		ID:                 metrics.ExecutionID,
		WorkflowID:         metrics.WorkflowID,
		WorkflowName:       metrics.WorkflowName,
		Status:             metrics.Status,
		StartedAt:          metrics.StartTime.Unix(),
		NodeResults:        nodeResults,
		RestructuredOutput: workflow.RestructureOutputs(nodeResults),
		Variables:          make(map[string]any),
	}

	if metrics.EndTime != nil {
		execution.FinishedAt = metrics.EndTime.Unix()
	}

	return execution, nil
}

// CancelWorkflowExecution 取消工作流执行
func (s *NewWorkflowService) CancelWorkflowExecution(ctx context.Context, executionID string) error {
	logc.Infof(ctx, "[WorkflowService] Cancelling workflow execution: %s", executionID)

	// 从 map 中取出 cancel func
	val, ok := s.executionCancelFuncs.Load(executionID)
	if !ok {
		return fmt.Errorf("execution %s not found or already finished", executionID)
	}
	cancel := val.(context.CancelFunc)

	// 触发 context cancel
	cancel()

	// 更新执行状态为 cancelled
	record, err := s.executionRepo.GetByID(executionID)
	if err == nil {
		record.Status = "cancelled"
		// 保存到数据库（需要转换为 WorkflowExecution）
		exec := &models.WorkflowExecution{
			ID:             record.ID,
			WorkflowID:     record.WorkflowID,
			WorkflowName:   record.WorkflowName,
			Status:         record.Status,
			StartedAt:      record.StartedAt,
			FinishedAt:     time.Now().Unix(),
			NodeResults:    make(map[string]*models.NodeResult),
			Variables:      make(map[string]any),
			RunningNodes:   []string{},
			CompletedNodes: []string{},
			FailedNodes:    []string{},
		}
		s.saveExecutionRecord(exec)
	}

	// 通过 SSE 通知前端
	event := workflow.NewStreamStyleEvent("workflow_cancelled", executionID, "", "")
	event.Data.Status = "cancelled"
	eventCh := s.monitor.GetStreamEventChannel(executionID)
	if eventCh != nil {
		eventCh <- event
	}

	return nil
}

// ListWorkflows 获取工作流列表
func (s *NewWorkflowService) ListWorkflows(workspaceId string) (data interface{}, err error) {
	// 从数据库获取工作流列表
	var workflows []*models.Workflow
	if workspaceId != "" {
		workflows, err = s.workflowRepo.GetByWorkspaceID(workspaceId)
	} else {
		workflows, err = s.workflowRepo.GetAll()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get workflows: %w", err)
	}

	// 转换为响应格式
	response := &types.ResponseWorkflowList{
		Workflows: make([]types.ResponseWorkflow, 0, len(workflows)),
		Total:     len(workflows),
	}

	for _, workflow := range workflows {
		// 转换为 WorkflowWithNodes 以便解析 nodes 和 edges
		workflowWithNodes, err := repos.ToWorkflowWithNodes(workflow)
		if err != nil {
			logc.Errorf(context.Background(), "[WorkflowService] Failed to parse workflow %s: %v", workflow.ID, err)
			continue
		}

		response.Workflows = append(response.Workflows, types.ResponseWorkflow{
			ID:          workflow.ID,
			WorkspaceID: workflow.WorkspaceID,
			Name:        workflow.Name,
			Description: workflow.Description,
			Nodes:       workflowWithNodes.Nodes,
			Edges:       workflowWithNodes.Edges,
			CreatedAt:   workflow.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   workflow.UpdatedAt.Format(time.RFC3339),
		})
	}

	return response, nil
}

// Create 创建工作流
func (s *NewWorkflowService) Create(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestCreateWorkflow)

	// Check license limit (global)
	count, err := s.workflowRepo.CountAll()
	if err != nil {
		return nil, fmt.Errorf("failed to count workflows: %w", err)
	}
	if !license.CheckWorkflowLimit(int(count)) {
		return nil, fmt.Errorf("license limit exceeded: maximum number of workflows reached")
	}

	// 转换为 WorkflowWithNodes 以便处理 nodes 和 edges
	workflowWithNodes := &repos.WorkflowWithNodes{
		Workflow: &models.Workflow{
			WorkspaceID: r.WorkspaceID,
			Name:        r.Name,
			Description: r.Description,
		},
		Nodes: r.Nodes,
		Edges: r.Edges,
	}

	// 转换为 Workflow model
	workflow, err := repos.FromWorkflowWithNodes(workflowWithNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert workflow: %w", err)
	}

	// 初始化版本号和内容哈希
	workflow.CurrentVersionNum = 1
	workflow.CurrentContentHash = computeContentHash(workflow.Nodes, workflow.Edges)

	// 创建工作流
	if err := s.workflowRepo.Create(workflow); err != nil {
		return nil, fmt.Errorf("failed to create workflow: %w", err)
	}

	// 保存初始版本（Version 1）
	initialVersion := &models.WorkflowVersion{
		WorkflowID:  workflow.ID,
		Version:     1,
		Name:        workflow.Name,
		Description: workflow.Description,
		Nodes:       workflow.Nodes,
		Edges:       workflow.Edges,
		Hash:        workflow.CurrentContentHash,
	}
	if err := s.versionRepo.Create(initialVersion); err != nil {
		logc.Errorf(context.Background(), "[WorkflowService] Failed to save initial workflow version: %v", err)
	}

	// 转换为响应格式
	response := &types.ResponseWorkflow{
		ID:          workflow.ID,
		WorkspaceID: workflow.WorkspaceID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Nodes:       r.Nodes,
		Edges:       r.Edges,
		CreatedAt:   workflow.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   workflow.UpdatedAt.Format(time.RFC3339),
	}

	return response, nil
}

// GetByID 根据ID获取工作流
func (s *NewWorkflowService) GetByID(id string) (data interface{}, err error) {
	// 从数据库获取工作流
	workflow, err := s.workflowRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", id)
	}

	// 转换为 WorkflowWithNodes 以便解析 nodes 和 edges
	workflowWithNodes, err := repos.ToWorkflowWithNodes(workflow)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// 转换为响应格式
	response := &types.ResponseWorkflow{
		ID:          workflow.ID,
		Name:        workflow.Name,
		Description: workflow.Description,
		Nodes:       workflowWithNodes.Nodes,
		Edges:       workflowWithNodes.Edges,
		CreatedAt:   workflow.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   workflow.UpdatedAt.Format(time.RFC3339),
	}

	return response, nil
}

// Update 更新工作流
func (s *NewWorkflowService) Update(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestUpdateWorkflow)

	// 获取现有工作流，用于保存版本
	existingWorkflow, err := s.workflowRepo.GetByID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", r.ID)
	}

	// 转换为 WorkflowWithNodes 以便处理 nodes 和 edges
	workflowWithNodes := &repos.WorkflowWithNodes{
		Workflow: &models.Workflow{
			ID:          r.ID,
			WorkspaceID: r.WorkspaceID,
			Name:        r.Name,
			Description: r.Description,
		},
		Nodes: r.Nodes,
		Edges: r.Edges,
	}

	// 转换为 Workflow model（得到新的 nodes/edges JSON 字符串）
	newWorkflow, err := repos.FromWorkflowWithNodes(workflowWithNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert workflow: %w", err)
	}

	// 计算新内容的 hash
	newHash := computeContentHash(newWorkflow.Nodes, newWorkflow.Edges)

	// 只有内容变化才保存新版本（保存用户正在提交的新内容）
	if existingWorkflow.CurrentContentHash != newHash {
		// 从数据库获取该 workflow 的最大版本号，确保不同 workflow 之间不冲突
		maxVer, err := s.versionRepo.GetMaxVersion(existingWorkflow.ID)
		if err != nil {
			logc.Errorf(context.Background(), "[WorkflowService] Failed to get max version: %v", err)
			maxVer = existingWorkflow.CurrentVersionNum
		}
		newVersionNum := maxVer + 1
		// 将用户提交的新内容作为新版本记录保存
		version := &models.WorkflowVersion{
			WorkflowID:  existingWorkflow.ID,
			Version:     newVersionNum,
			Name:        newWorkflow.Name,
			Description: newWorkflow.Description,
			Nodes:       newWorkflow.Nodes,
			Edges:       newWorkflow.Edges,
			Hash:        newHash,
		}
		if err := s.versionRepo.Create(version); err != nil {
			logc.Errorf(context.Background(), "[WorkflowService] Failed to save workflow version: %v", err)
		}
		// 同步递增 CurrentVersionNum
		existingWorkflow.CurrentVersionNum = newVersionNum
	}

	// 更新工作流内容及坐标
	newWorkflow.CurrentVersionNum = existingWorkflow.CurrentVersionNum
	newWorkflow.CurrentContentHash = newHash
	if err := s.workflowRepo.Update(newWorkflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow: %w", err)
	}

	// 转换为响应格式
	response := &types.ResponseWorkflow{
		ID:          newWorkflow.ID,
		WorkspaceID: r.WorkspaceID,
		Name:        newWorkflow.Name,
		Description: newWorkflow.Description,
		Nodes:       r.Nodes,
		Edges:       r.Edges,
		CreatedAt:   newWorkflow.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   newWorkflow.UpdatedAt.Format(time.RFC3339),
	}

	return response, nil
}

// computeContentHash 计算 nodes + edges 的 SHA256 哈希，用于判断内容是否变化
func computeContentHash(nodesJSON, edgesJSON string) string {
	h := sha256.New()
	h.Write([]byte(nodesJSON))
	h.Write([]byte("|"))
	h.Write([]byte(edgesJSON))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// UpdateDescription 更新工作流描述（部分更新）
func (s *NewWorkflowService) UpdateDescription(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestUpdateWorkflowDescription)

	// 获取现有工作流
	existingWorkflow, err := s.workflowRepo.GetByID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", r.ID)
	}

	// 验证 workspace ID 匹配
	if existingWorkflow.WorkspaceID != r.WorkspaceID {
		return nil, fmt.Errorf("workflow does not belong to the specified workspace")
	}

	// 只更新描述字段
	existingWorkflow.Description = r.Description

	// 更新工作流
	if err := s.workflowRepo.Update(existingWorkflow); err != nil {
		return nil, fmt.Errorf("failed to update workflow description: %w", err)
	}

	// 转换为 WorkflowWithNodes 以便解析 nodes 和 edges
	workflowWithNodes, err := repos.ToWorkflowWithNodes(existingWorkflow)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// 转换为响应格式
	response := &types.ResponseWorkflow{
		ID:          existingWorkflow.ID,
		WorkspaceID: existingWorkflow.WorkspaceID,
		Name:        existingWorkflow.Name,
		Description: existingWorkflow.Description,
		Nodes:       workflowWithNodes.Nodes,
		Edges:       workflowWithNodes.Edges,
		CreatedAt:   existingWorkflow.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   existingWorkflow.UpdatedAt.Format(time.RFC3339),
	}

	return response, nil
}

// GotoVersion 前往指定版本
// 只切换当前工作流指向目标版本的内容，不生成新版本，不影响其他版本记录
func (s *NewWorkflowService) GotoVersion(workflowId, versionNumStr string) (data interface{}, err error) {
	versionNum := 0
	if _, parseErr := fmt.Sscanf(versionNumStr, "%d", &versionNum); parseErr != nil || versionNum <= 0 {
		return nil, fmt.Errorf("invalid version number: %s", versionNumStr)
	}

	// 获取指定版本
	version, err := s.versionRepo.GetByVersionNum(workflowId, versionNum)
	if err != nil {
		return nil, fmt.Errorf("version %d not found: %w", versionNum, err)
	}

	// 获取当前工作流
	existingWorkflow, err := s.workflowRepo.GetByID(workflowId)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowId)
	}

	// 更新工作流内容及坐标：切换到目标版本，不生成新版本记录
	existingWorkflow.Nodes = version.Nodes
	existingWorkflow.Edges = version.Edges
	existingWorkflow.Name = version.Name
	existingWorkflow.Description = version.Description
	existingWorkflow.CurrentVersionNum = version.Version
	existingWorkflow.CurrentContentHash = version.Hash

	if err := s.workflowRepo.Update(existingWorkflow); err != nil {
		return nil, fmt.Errorf("failed to goto version: %w", err)
	}

	// 解析 nodes 和 edges 返回给前端
	var nodes, edges []interface{}
	if version.Nodes != "" {
		json.Unmarshal([]byte(version.Nodes), &nodes)
	}
	if version.Edges != "" {
		json.Unmarshal([]byte(version.Edges), &edges)
	}

	return map[string]interface{}{
		"id":                existingWorkflow.ID,
		"versionNum":        version.Version,
		"contentHash":       version.Hash,
		"nodes":             nodes,
		"edges":             edges,
		"name":              version.Name,
		"description":       version.Description,
		"currentVersionNum": version.Version,
	}, nil
}

// GetVersionHistory 获取工作流版本历史
func (s *NewWorkflowService) GetVersionHistory(workflowId string, limit int) (data interface{}, err error) {
	workflow, err := s.workflowRepo.GetByID(workflowId)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", workflowId)
	}

	versions, err := s.versionRepo.GetByWorkflowID(workflowId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get version history: %w", err)
	}

	history := make([]map[string]interface{}, 0, len(versions))
	for _, v := range versions {
		// 解析 nodes 和 edges
		var nodes, edges []interface{}
		if v.Nodes != "" {
			json.Unmarshal([]byte(v.Nodes), &nodes)
		}
		if v.Edges != "" {
			json.Unmarshal([]byte(v.Edges), &edges)
		}

		history = append(history, map[string]interface{}{
			"workflowId":  v.WorkflowID,
			"version":     v.Version,
			"name":        v.Name,
			"description": v.Description,
			"nodes":       nodes,
			"edges":       edges,
			"isRelease":   v.IsRelease,
			"createdAt":   v.CreatedAt.Format(time.RFC3339),
		})
	}

	return map[string]interface{}{
		"workflowId":        workflowId,
		"versions":          history,
		"total":             len(history),
		"currentVersionNum": workflow.CurrentVersionNum,
	}, nil
}

// GetVersion 获取指定版本
func (s *NewWorkflowService) GetVersion(workflowId string, versionNum int) (data interface{}, err error) {
	version, err := s.versionRepo.GetByVersionNum(workflowId, versionNum)
	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	// 解析 nodes 和 edges
	var nodes, edges []interface{}
	if version.Nodes != "" {
		json.Unmarshal([]byte(version.Nodes), &nodes)
	}
	if version.Edges != "" {
		json.Unmarshal([]byte(version.Edges), &edges)
	}

	return map[string]interface{}{
		"workflowId":  version.WorkflowID,
		"version":     version.Version,
		"name":        version.Name,
		"description": version.Description,
		"nodes":       nodes,
		"edges":       edges,
		"isRelease":   version.IsRelease,
		"createdAt":   version.CreatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteVersion 删除指定版本
func (s *NewWorkflowService) DeleteVersion(workflowId string, versionNum int) (data interface{}, err error) {
	// 获取版本确认存在
	_, err = s.versionRepo.GetByVersionNum(workflowId, versionNum)
	if err != nil {
		return nil, fmt.Errorf("version %d not found: %w", versionNum, err)
	}

	// 删除版本记录
	if err := s.versionRepo.Delete(workflowId, versionNum); err != nil {
		return nil, fmt.Errorf("failed to delete version: %w", err)
	}

	return map[string]interface{}{
		"message":    "version deleted successfully",
		"workflowId": workflowId,
		"versionNum": versionNum,
	}, nil
}

// SetRelease 标记指定版本为 Release（同时取消其他版本）
func (s *NewWorkflowService) SetRelease(workflowId string, versionNum int) (data interface{}, err error) {
	version, err := s.versionRepo.GetByVersionNum(workflowId, versionNum)
	if err != nil {
		return nil, fmt.Errorf("version %d not found: %w", versionNum, err)
	}

	if err := s.versionRepo.SetRelease(workflowId, versionNum); err != nil {
		return nil, fmt.Errorf("failed to set release: %w", err)
	}

	return map[string]interface{}{
		"message":           "version set as release successfully",
		"workflowId":        workflowId,
		"versionNum":        versionNum,
		"name":              version.Name,
		"description":       version.Description,
		"currentVersionNum": version.Version,
		"contentHash":       version.Hash,
	}, nil
}

// CancelRelease 取消指定版本的 Release 标记
func (s *NewWorkflowService) CancelRelease(workflowId string, versionNum int) (data interface{}, err error) {
	version, err := s.versionRepo.GetByVersionNum(workflowId, versionNum)
	if err != nil {
		return nil, fmt.Errorf("version %d not found: %w", versionNum, err)
	}

	if err := s.versionRepo.CancelRelease(workflowId, versionNum); err != nil {
		return nil, fmt.Errorf("failed to cancel release: %w", err)
	}

	return map[string]interface{}{
		"message":           "version release cancelled successfully",
		"workflowId":        workflowId,
		"versionNum":        versionNum,
		"currentVersionNum": version.Version,
	}, nil
}

// Delete 删除工作流
func (s *NewWorkflowService) Delete(id string) (data interface{}, err error) {
	// 删除工作流
	if err := s.workflowRepo.Delete(id); err != nil {
		return nil, fmt.Errorf("failed to delete workflow: %w", err)
	}

	// 删除该工作流的所有历史版本
	if err := s.versionRepo.DeleteByWorkflowID(id); err != nil {
		logc.Errorf(context.Background(), "[WorkflowService] Failed to delete workflow versions for %s: %v", id, err)
	}

	return map[string]interface{}{
		"message": "workflow deleted successfully",
		"id":      id,
	}, nil
}

// ExecuteAsync 异步执行工作流，立即返回执行ID
func (s *NewWorkflowService) ExecuteAsync(req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestWorkflowExecute)

	// 从数据库加载工作流以验证存在性
	wf, err := s.workflowRepo.GetByID(r.WorkflowId)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %s", r.WorkflowId)
	}

	// 将 input 参数注入到 Start 节点
	if err := injectStartNodeInput(wf, r.Input, r.Mode, r.RawBody); err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}

	// 创建执行记录
	execution := &models.WorkflowExecution{
		ID:             "exec-" + tools.RandId(),
		WorkflowID:     wf.ID,
		WorkflowName:   wf.Name,
		Status:         "pending",
		StartedAt:      time.Now().Unix(),
		NodeResults:    make(map[string]*models.NodeResult),
		Variables:      make(map[string]any),
		RunningNodes:   []string{},
		CompletedNodes: []string{},
		FailedNodes:    []string{},
	}

	// 注册 SSE 事件通道（让前端可以通过 SSE 接收节点执行事件）
	s.monitor.RegisterEventSSEEventChannel(execution.ID, make(chan *workflow.StreamStyleEvent, 100))

	// 启动异步执行
	go func() {
		// 创建可取消的 context，并注入请求级 trace id，使逻辑层日志可与接口请求串联
		baseCtx := trace.WithTraceID(context.Background(), r.TraceID)
		execCtx, cancelExec := context.WithCancel(baseCtx)
		s.executionCancelFuncs.Store(execution.ID, cancelExec)

		logc.Infof(execCtx, "[WorkflowService] start executing workflow %s, execution %s", wf.ID, execution.ID)

		defer func() {
			if r := recover(); r != nil {
				s.markExecutionFailed(execution, fmt.Sprintf("panic: %v", r))
				s.saveExecutionRecord(execution)
			}
			s.executionCancelFuncs.Delete(execution.ID)
		}()

		// 解析工作流拓扑
		topology, err := s.parser.ParseWorkflowFromModel(wf)
		if err != nil {
			logc.Errorf(execCtx, "[WorkflowService] failed to parse workflow %s: %v", wf.ID, err)
			return
		}

		// 验证工作流
		registry := s.engine.GetRegistry()
		if err := s.parser.ValidateWorkflow(topology, registry); err != nil {
			logc.Errorf(execCtx, "[WorkflowService] workflow validation failed for %s: %v", wf.ID, err)
			return
		}

		// 执行工作流（使用 execCtx，可被 CancelWorkflowExecution 取消）
		_, executionErr := s.engine.Execute(execCtx, wf, execution)
		if executionErr != nil {
			s.markExecutionFailed(execution, fmt.Sprintf("workflow execution failed: %v", executionErr))
		} else {
			s.markExecutionCompleted(execution)
		}

		// 保存执行记录到数据库
		s.saveExecutionRecord(execution)

		// 打印工作流执行日志
		printWorkflowLogs(execCtx, execution)
	}()

	// 立即返回执行ID
	return map[string]interface{}{
		"executionId": execution.ID,
		"workflowId":  wf.ID,
		"status":      "pending",
		"message":     "Workflow execution started",
	}, nil
}

// RegisterEventSSEEventChannel 注册流式 SSE 事件通道
func (s *NewWorkflowService) RegisterEventSSEEventChannel(executionId string) chan *workflow.StreamStyleEvent {
	// 直接调用 monitor 的方法，它会处理 channel 复用逻辑
	eventCh := make(chan *workflow.StreamStyleEvent, 100)
	return s.monitor.RegisterEventSSEEventChannel(executionId, eventCh)
}

// UnregisterEventSSEEventChannel 取消注册流式 SSE 事件通道
func (s *NewWorkflowService) UnregisterEventSSEEventChannel(executionId string) {
	s.monitor.UnregisterEventSSEEventChannel(executionId)
	logc.Infof(context.Background(), "[WorkflowService] Stream SSE channel UNREGISTERED for execution: %s", executionId)
}

// GetExecutionResult 获取执行结果
func (s *NewWorkflowService) GetExecutionResult(executionId string) (data interface{}, err error) {
	return s.GetWorkflowExecution(context.Background(), executionId)
}

// GetExecutionStatus 获取执行状态
func (s *NewWorkflowService) GetExecutionStatus(executionId string) (data interface{}, err error) {
	// 优先从状态管理器获取（实时状态）
	workflowState, stateErr := s.stateManager.GetWorkflowState(executionId)
	if stateErr == nil {
		// 从状态管理器获取实时节点状态
		runningNodes := []string{}
		completedNodes := []string{}
		failedNodes := []string{}
		pendingNodes := []string{}
		skippedNodes := []string{}

		for nodeID, nodeState := range workflowState.NodeStates {
			switch nodeState.State {
			case workflow.NodeStateRunning:
				runningNodes = append(runningNodes, nodeID)
			case workflow.NodeStateCompleted:
				completedNodes = append(completedNodes, nodeID)
			case workflow.NodeStateFailed:
				failedNodes = append(failedNodes, nodeID)
			case workflow.NodeStatePending, workflow.NodeStateReady, workflow.NodeStateIdle:
				pendingNodes = append(pendingNodes, nodeID)
			case workflow.NodeStateSkipped:
				skippedNodes = append(skippedNodes, nodeID)
			}
		}

		// 如果工作流已完成但节点状态为空，从 NodeResults 推断
		if workflowState.State == "completed" && len(completedNodes) == 0 && len(failedNodes) == 0 && len(workflowState.NodeResults) > 0 {
			for nodeID := range workflowState.NodeResults {
				completedNodes = append(completedNodes, nodeID)
			}
		}

		response := map[string]interface{}{
			"executionId":    executionId,
			"workflowId":     workflowState.WorkflowID,
			"workflowName":   workflowState.WorkflowName,
			"status":         workflowState.State,
			"startedAt":      workflowState.StartTime.Unix(),
			"totalNodes":     workflowState.TotalNodes,
			"runningNodes":   runningNodes,
			"completedNodes": completedNodes,
			"failedNodes":    failedNodes,
			"pendingNodes":   pendingNodes,
			"skippedNodes":   skippedNodes,
		}
		if workflowState.EndTime != nil {
			response["finishedAt"] = workflowState.EndTime.Unix()
		}
		return response, nil
	}

	// 状态管理器没有，从监控器获取（监控器在 engine 和 executor 间共享）
	metrics, monErr := s.monitor.GetExecutionMetrics(executionId)
	if monErr == nil {
		runningNodes := []string{}
		completedNodes := []string{}
		failedNodes := []string{}
		skippedNodes := []string{}

		for nodeID, status := range metrics.NodeStatuses {
			switch status {
			case workflow.NodeStatusRunning:
				runningNodes = append(runningNodes, nodeID)
			case workflow.NodeStatusCompleted:
				completedNodes = append(completedNodes, nodeID)
			case workflow.NodeStatusFailed:
				failedNodes = append(failedNodes, nodeID)
			case workflow.NodeStatusSkipped:
				skippedNodes = append(skippedNodes, nodeID)
			}
		}

		response := map[string]interface{}{
			"executionId":    executionId,
			"workflowId":     metrics.WorkflowID,
			"workflowName":   metrics.WorkflowName,
			"status":         metrics.Status,
			"startedAt":      metrics.StartTime.Unix(),
			"totalNodes":     metrics.TotalNodes,
			"runningNodes":   runningNodes,
			"completedNodes": completedNodes,
			"failedNodes":    failedNodes,
			"pendingNodes":   []string{},
			"skippedNodes":   skippedNodes,
		}
		if metrics.EndTime != nil {
			response["finishedAt"] = metrics.EndTime.Unix()
		}
		return response, nil
	}

	// 都没有，从数据库获取（最终状态）
	record, dbErr := s.executionRepo.GetByID(executionId)
	if dbErr != nil {
		return map[string]interface{}{
			"executionId":    executionId,
			"status":         "completed",
			"message":        "Execution completed or not found",
			"runningNodes":   []string{},
			"completedNodes": []string{},
			"failedNodes":    []string{},
			"pendingNodes":   []string{},
		}, nil
	}

	var nodeResults map[string]*models.NodeResult
	if record.NodeResults != "" {
		json.Unmarshal([]byte(record.NodeResults), &nodeResults)
	}
	if nodeResults == nil {
		nodeResults = make(map[string]*models.NodeResult)
	}

	completedNodes := []string{}
	failedNodes := []string{}
	for nodeID, result := range nodeResults {
		if result.Error != "" {
			failedNodes = append(failedNodes, nodeID)
		} else {
			completedNodes = append(completedNodes, nodeID)
		}
	}

	return map[string]interface{}{
		"executionId":    executionId,
		"status":         record.Status,
		"workflowId":     record.WorkflowID,
		"workflowName":   record.WorkflowName,
		"startedAt":      record.StartedAt,
		"finishedAt":     record.FinishedAt,
		"totalNodes":     len(nodeResults),
		"runningNodes":   []string{},
		"completedNodes": completedNodes,
		"failedNodes":    failedNodes,
		"pendingNodes":   []string{},
	}, nil
}

// GetRunHistory 获取运行历史
func (s *NewWorkflowService) GetRunHistory(workflowId string, limit int) (data interface{}, err error) {
	// 从数据库获取执行记录
	records, err := s.executionRepo.GetByWorkflowID(workflowId, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get run history: %w", err)
	}

	// 转换为响应格式
	history := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		// 解析节点结果
		var nodeResults map[string]*models.NodeResult
		if record.NodeResults != "" {
			if err := json.Unmarshal([]byte(record.NodeResults), &nodeResults); err != nil {
				logc.Errorf(context.Background(), "[WorkflowService] Failed to parse node results for %s: %v", record.ID, err)
				nodeResults = make(map[string]*models.NodeResult)
			}
		} else {
			nodeResults = make(map[string]*models.NodeResult)
		}

		// 解析变量
		var variables map[string]any
		if record.Variables != "" {
			if err := json.Unmarshal([]byte(record.Variables), &variables); err != nil {
				logc.Errorf(context.Background(), "[WorkflowService] Failed to parse variables for %s: %v", record.ID, err)
				variables = make(map[string]any)
			}
		} else {
			variables = make(map[string]any)
		}

		history = append(history, map[string]interface{}{
			"id":           record.ID,
			"workflowId":   record.WorkflowID,
			"workflowName": record.WorkflowName,
			"status":       record.Status,
			"startedAt":    record.StartedAt,
			"finishedAt":   record.FinishedAt,
			"outputs":      workflow.RestructureOutputs(nodeResults),
			"variables":    variables,
			"finalOutput":  record.FinalOutput,
			"createdAt":    record.CreatedAt,
			"updatedAt":    record.UpdatedAt,
		})
	}

	return map[string]interface{}{
		"workflowId": workflowId,
		"history":    history,
		"total":      len(history),
		"limit":      limit,
	}, nil
}

// GetAllRunHistory 获取所有运行历史
func (s *NewWorkflowService) GetAllRunHistory(limit int) (data interface{}, err error) {
	// 从数据库获取所有执行记录
	records, err := s.executionRepo.GetAll(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get all run history: %w", err)
	}

	// 转换为响应格式
	history := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		// 解析节点结果
		var nodeResults map[string]*models.NodeResult
		if record.NodeResults != "" {
			if err := json.Unmarshal([]byte(record.NodeResults), &nodeResults); err != nil {
				logc.Errorf(context.Background(), "[WorkflowService] Failed to parse node results for %s: %v", record.ID, err)
				nodeResults = make(map[string]*models.NodeResult)
			}
		} else {
			nodeResults = make(map[string]*models.NodeResult)
		}

		// 解析变量
		var variables map[string]any
		if record.Variables != "" {
			if err := json.Unmarshal([]byte(record.Variables), &variables); err != nil {
				logc.Errorf(context.Background(), "[WorkflowService] Failed to parse variables for %s: %v", record.ID, err)
				variables = make(map[string]any)
			}
		} else {
			variables = make(map[string]any)
		}

		history = append(history, map[string]interface{}{
			"id":           record.ID,
			"workflowId":   record.WorkflowID,
			"workflowName": record.WorkflowName,
			"status":       record.Status,
			"startedAt":    record.StartedAt,
			"finishedAt":   record.FinishedAt,
			"outputs":      workflow.RestructureOutputs(nodeResults),
			"variables":    variables,
			"finalOutput":  record.FinalOutput,
			"createdAt":    record.CreatedAt,
			"updatedAt":    record.UpdatedAt,
		})
	}

	return map[string]interface{}{
		"history": history,
		"total":   len(history),
		"limit":   limit,
	}, nil
}

// DeleteRunHistory 删除运行历史
func (s *NewWorkflowService) DeleteRunHistory(executionId string) (data interface{}, err error) {
	// 从数据库删除执行记录
	if err := s.executionRepo.Delete(executionId); err != nil {
		return nil, fmt.Errorf("failed to delete run history: %w", err)
	}

	return map[string]interface{}{
		"message":     "Run history deleted successfully",
		"executionId": executionId,
	}, nil
}
