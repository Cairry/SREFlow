package repos

import (
	"RCAgent/internal/models"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type (
	WorkflowRepoInter interface {
		Create(workflow *models.Workflow) error
		GetByID(id string) (*models.Workflow, error)
		GetAll() ([]*models.Workflow, error)
		GetByWorkspaceID(workspaceID string) ([]*models.Workflow, error)
		Update(workflow *models.Workflow) error
		Delete(id string) error
		DeleteByWorkspaceID(workspaceID string) error
		Search(term string) ([]*models.Workflow, error)
		GetByDatasourceID(workspaceID, datasourceID string) ([]*models.Workflow, error)
	}

	WorkflowExecutionRepoInter interface {
		Create(record *models.WorkflowExecutionRecord) error
		GetByID(id string) (*models.WorkflowExecutionRecord, error)
		GetByWorkflowID(workflowID string, limit int) ([]*models.WorkflowExecutionRecord, error)
		GetAll(limit int) ([]*models.WorkflowExecutionRecord, error)
		Delete(id string) error
		DeleteOld(olderThan int64) error // 删除指定时间之前的记录
	}

	WorkflowVersionRepoInter interface {
		Create(version *models.WorkflowVersion) error
		GetByVersionNum(workflowID string, version int) (*models.WorkflowVersion, error)
		GetByWorkflowID(workflowID string, limit int) ([]*models.WorkflowVersion, error)
		GetLatestVersion(workflowID string) (*models.WorkflowVersion, error)
		GetMaxVersion(workflowID string) (int, error)
		Delete(workflowID string, version int) error
		DeleteByWorkflowID(workflowID string) error
		SetRelease(workflowID string, version int) error  // 标记为 Release（同时取消其他）
		CancelRelease(workflowID string, version int) error // 取消指定版本的 Release
	}

	workflowRepo struct {
		db *gorm.DB
	}

	workflowExecutionRepo struct {
		db *gorm.DB
	}

	workflowVersionRepo struct {
		db *gorm.DB
	}
)

func NewWorkflowRepo(db *gorm.DB) WorkflowRepoInter {
	return &workflowRepo{db: db}
}

// Create creates a new workflow
func (r *workflowRepo) Create(workflow *models.Workflow) error {
	if workflow.ID == "" {
		workflow.ID = generateID()
	}
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	return r.db.Create(workflow).Error
}

// GetByID retrieves a workflow by ID
func (r *workflowRepo) GetByID(id string) (*models.Workflow, error) {
	var workflow models.Workflow
	err := r.db.Where("id = ?", id).First(&workflow).Error
	if err != nil {
		return nil, err
	}
	return &workflow, nil
}

// GetAll retrieves all workflows
func (r *workflowRepo) GetAll() ([]*models.Workflow, error) {
	var workflows []*models.Workflow
	err := r.db.Order("created_at DESC").Find(&workflows).Error
	return workflows, err
}

// GetByWorkspaceID retrieves all workflows for a specific workspace
func (r *workflowRepo) GetByWorkspaceID(workspaceID string) ([]*models.Workflow, error) {
	var workflows []*models.Workflow
	err := r.db.Where("workspace_id = ?", workspaceID).Order("created_at DESC").Find(&workflows).Error
	return workflows, err
}

// Update updates a workflow
func (r *workflowRepo) Update(workflow *models.Workflow) error {
	workflow.UpdatedAt = time.Now()
	return r.db.Save(workflow).Error
}

// Delete soft deletes a workflow
func (r *workflowRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Workflow{}).Error
}

// DeleteByWorkspaceID soft deletes all workflows for a specific workspace
func (r *workflowRepo) DeleteByWorkspaceID(workspaceID string) error {
	return r.db.Where("workspace_id = ?", workspaceID).Delete(&models.Workflow{}).Error
}

// Search searches workflows by name, description, category, and tags
func (r *workflowRepo) Search(term string) ([]*models.Workflow, error) {
	var workflows []*models.Workflow
	searchPattern := "%" + term + "%"

	err := r.db.Where(
		"name LIKE ? OR description LIKE ? OR category LIKE ? OR tags LIKE ?",
		searchPattern, searchPattern, searchPattern, searchPattern,
	).Order("created_at DESC").Find(&workflows).Error

	return workflows, err
}

// GetByDatasourceID returns workflows that reference the given datasourceId in any node
func (r *workflowRepo) GetByDatasourceID(workspaceID, datasourceID string) ([]*models.Workflow, error) {
	workflows, err := r.GetByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	var result []*models.Workflow
	for _, wf := range workflows {
		if wf.Nodes == "" {
			continue
		}
		var nodes []map[string]any
		if err := json.Unmarshal([]byte(wf.Nodes), &nodes); err != nil {
			continue
		}
		for _, node := range nodes {
			data, ok := node["data"].(map[string]any)
			if !ok {
				continue
			}
			if id, ok := data["datasourceId"].(string); ok && id == datasourceID {
				result = append(result, wf)
				break
			}
		}
	}
	return result, nil
}

// generateID generates a unique ID for workflow
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// NewWorkflowExecutionRepo creates a new workflow execution repository
func NewWorkflowExecutionRepo(db *gorm.DB) WorkflowExecutionRepoInter {
	return &workflowExecutionRepo{db: db}
}

// Create creates a new workflow execution record
func (r *workflowExecutionRepo) Create(record *models.WorkflowExecutionRecord) error {
	if record.ID == "" {
		record.ID = generateID()
	}
	return r.db.Create(record).Error
}

// GetByID retrieves a workflow execution record by ID
func (r *workflowExecutionRepo) GetByID(id string) (*models.WorkflowExecutionRecord, error) {
	var record models.WorkflowExecutionRecord
	err := r.db.Where("id = ?", id).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByWorkflowID retrieves execution records for a specific workflow
func (r *workflowExecutionRepo) GetByWorkflowID(workflowID string, limit int) ([]*models.WorkflowExecutionRecord, error) {
	var records []*models.WorkflowExecutionRecord
	query := r.db.Where("workflow_id = ?", workflowID).Order("started_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&records).Error
	return records, err
}

// GetAll retrieves all execution records with optional limit
func (r *workflowExecutionRepo) GetAll(limit int) ([]*models.WorkflowExecutionRecord, error) {
	var records []*models.WorkflowExecutionRecord
	query := r.db.Order("started_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&records).Error
	return records, err
}

// Delete deletes a workflow execution record
func (r *workflowExecutionRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.WorkflowExecutionRecord{}).Error
}

// DeleteOld deletes execution records older than the specified timestamp
func (r *workflowExecutionRepo) DeleteOld(olderThan int64) error {
	return r.db.Where("started_at < ?", olderThan).Delete(&models.WorkflowExecutionRecord{}).Error
}

// WorkflowWithNodes represents workflow with parsed nodes and edges
type WorkflowWithNodes struct {
	*models.Workflow
	Nodes []interface{} `json:"nodes"`
	Edges []interface{} `json:"edges"`
}

// ToWorkflowWithNodes converts Workflow model to WorkflowWithNodes
func ToWorkflowWithNodes(workflow *models.Workflow) (*WorkflowWithNodes, error) {
	result := &WorkflowWithNodes{
		Workflow: workflow,
	}

	// Parse nodes JSON
	if workflow.Nodes != "" {
		if err := json.Unmarshal([]byte(workflow.Nodes), &result.Nodes); err != nil {
			return nil, fmt.Errorf("failed to parse nodes: %w", err)
		}
	}

	// Parse edges JSON
	if workflow.Edges != "" {
		if err := json.Unmarshal([]byte(workflow.Edges), &result.Edges); err != nil {
			return nil, fmt.Errorf("failed to parse edges: %w", err)
		}
	}

	return result, nil
}

// FromWorkflowWithNodes converts WorkflowWithNodes to Workflow model
func FromWorkflowWithNodes(workflow *WorkflowWithNodes) (*models.Workflow, error) {
	result := workflow.Workflow

	// Convert nodes to JSON
	if workflow.Nodes != nil {
		nodesJSON, err := json.Marshal(workflow.Nodes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal nodes: %w", err)
		}
		result.Nodes = string(nodesJSON)
	}

	// Convert edges to JSON
	if workflow.Edges != nil {
		edgesJSON, err := json.Marshal(workflow.Edges)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal edges: %w", err)
		}
		result.Edges = string(edgesJSON)
	}

	return result, nil
}

// NewWorkflowVersionRepo creates a new workflow version repository
func NewWorkflowVersionRepo(db *gorm.DB) WorkflowVersionRepoInter {
	return &workflowVersionRepo{db: db}
}

// Create creates a new workflow version record
func (r *workflowVersionRepo) Create(version *models.WorkflowVersion) error {
	version.CreatedAt = time.Now()
	return r.db.Create(version).Error
}

// GetByVersionNum retrieves a workflow version by workflow ID and version number
func (r *workflowVersionRepo) GetByVersionNum(workflowID string, version int) (*models.WorkflowVersion, error) {
	var v models.WorkflowVersion
	err := r.db.Where("workflow_id = ? AND version = ?", workflowID, version).First(&v).Error
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// GetByWorkflowID retrieves all versions for a specific workflow
func (r *workflowVersionRepo) GetByWorkflowID(workflowID string, limit int) ([]*models.WorkflowVersion, error) {
	var versions []*models.WorkflowVersion
	query := r.db.Where("workflow_id = ?", workflowID).Order("version DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&versions).Error
	return versions, err
}

// GetLatestVersion retrieves the latest version for a specific workflow
func (r *workflowVersionRepo) GetLatestVersion(workflowID string) (*models.WorkflowVersion, error) {
	var version models.WorkflowVersion
	err := r.db.Where("workflow_id = ?", workflowID).Order("version DESC").First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

// GetMaxVersion returns the highest version number for a workflow
func (r *workflowVersionRepo) GetMaxVersion(workflowID string) (int, error) {
	var maxVersion int
	err := r.db.Model(&models.WorkflowVersion{}).
		Where("workflow_id = ?", workflowID).
		Select("COALESCE(MAX(version), 0)").
		Scan(&maxVersion).Error
	return maxVersion, err
}

// Delete deletes a workflow version record by workflowID and version number
func (r *workflowVersionRepo) Delete(workflowID string, version int) error {
	return r.db.Where("workflow_id = ? AND version = ?", workflowID, version).Delete(&models.WorkflowVersion{}).Error
}

// DeleteByWorkflowID deletes all versions for a specific workflow
func (r *workflowVersionRepo) DeleteByWorkflowID(workflowID string) error {
	return r.db.Where("workflow_id = ?", workflowID).Delete(&models.WorkflowVersion{}).Error
}

// SetRelease marks a specific version as a Release version
func (r *workflowVersionRepo) SetRelease(workflowID string, version int) error {
	return r.db.Model(&models.WorkflowVersion{}).
		Where("workflow_id = ? AND version = ?", workflowID, version).
		Update("is_release", true).Error
}

// CancelRelease cancels the Release flag of a specific version
func (r *workflowVersionRepo) CancelRelease(workflowID string, version int) error {
	return r.db.Model(&models.WorkflowVersion{}).
		Where("workflow_id = ? AND version = ?", workflowID, version).
		Update("is_release", false).Error
}
