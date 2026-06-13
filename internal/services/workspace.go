package services

import (
	"context"
	"fmt"
	"time"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/internal/types"
	"SREFlow/pkg/tools"

	"github.com/zeromicro/go-zero/core/logc"
)

// WorkspaceService 工作空间服务
type WorkspaceService struct {
	workspaceRepo      repos.WorkspaceRepoInter
	workflowRepo       repos.WorkflowRepoInter
	integrationService *IntegrationService
}

// NewWorkspaceService 创建工作空间服务
func NewWorkspaceService(workspaceRepo repos.WorkspaceRepoInter, workflowRepo repos.WorkflowRepoInter) *WorkspaceService {
	return &WorkspaceService{workspaceRepo: workspaceRepo, workflowRepo: workflowRepo}
}

// SetIntegrationService 设置集成服务（避免循环依赖）
func (s *WorkspaceService) SetIntegrationService(is *IntegrationService) {
	s.integrationService = is
}

// List 返回所有工作空间
func (s *WorkspaceService) List(ctx context.Context) (data interface{}, err error) {
	workspaces, err := s.workspaceRepo.GetAll()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"workspaces": workspaces,
	}, nil
}

// GetByID 根据ID获取工作空间
func (s *WorkspaceService) GetByID(ctx context.Context, id string) (data interface{}, err error) {
	workspace, err := s.workspaceRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %s", id)
	}
	return workspace, nil
}

// Create 创建工作空间
func (s *WorkspaceService) Create(ctx context.Context, req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestCreateWorkspace)

	workspace := &models.Workspace{
		ID:   tools.RandId(),
		Name: r.Name,
	}

	if err := s.workspaceRepo.Create(workspace); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	logc.Infof(ctx, "[WorkspaceService] Workspace created: %s (%s)", workspace.Name, workspace.ID)

	// 为新工作空间创建所有默认集成
	if s.integrationService != nil {
		s.integrationService.EnsureDefaultIntegrations(workspace.ID)
	}

	return workspace, nil
}

// Update 更新工作空间
func (s *WorkspaceService) Update(ctx context.Context, req interface{}) (data interface{}, err error) {
	r := req.(*types.RequestUpdateWorkspace)

	workspace, err := s.workspaceRepo.GetByID(r.ID)
	if err != nil {
		return nil, fmt.Errorf("workspace not found: %s", r.ID)
	}

	if r.Name != "" {
		workspace.Name = r.Name
	}
	workspace.UpdatedAt = time.Now()

	if err := s.workspaceRepo.Update(workspace); err != nil {
		return nil, fmt.Errorf("failed to update workspace: %w", err)
	}

	logc.Infof(ctx, "[WorkspaceService] Workspace updated: %s (%s)", workspace.Name, workspace.ID)
	return workspace, nil
}

// Delete 删除工作空间
func (s *WorkspaceService) Delete(ctx context.Context, id string) (data interface{}, err error) {
	// 检查工作空间下是否还有 workflow
	if s.workflowRepo != nil {
		workflows, err := s.workflowRepo.GetByWorkspaceID(id)
		if err != nil {
			return nil, fmt.Errorf("failed to check workflows for workspace: %w", err)
		}
		if len(workflows) > 0 {
			return nil, fmt.Errorf("cannot delete workspace: workspace still has %d workflow(s). Please delete all workflows first", len(workflows))
		}
	}

	if err := s.workspaceRepo.Delete(id); err != nil {
		return nil, fmt.Errorf("failed to delete workspace: %w", err)
	}
	logc.Infof(ctx, "[WorkspaceService] Workspace deleted: %s", id)
	return map[string]interface{}{"id": id}, nil
}
