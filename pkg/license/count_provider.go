package license

import (
	"SREFlow/internal/repos"
)

// RepoCountProvider implements CountProvider using repos
type RepoCountProvider struct {
	workspaceRepo repos.WorkspaceRepoInter
	workflowRepo  repos.WorkflowRepoInter
	mcpRepo       repos.MCPServerRepoInter
	skillRepo     repos.SkillRepoInter
}

// NewRepoCountProvider creates a new RepoCountProvider
func NewRepoCountProvider(
	workspaceRepo repos.WorkspaceRepoInter,
	workflowRepo repos.WorkflowRepoInter,
	mcpRepo repos.MCPServerRepoInter,
	skillRepo repos.SkillRepoInter,
) *RepoCountProvider {
	return &RepoCountProvider{
		workspaceRepo: workspaceRepo,
		workflowRepo:  workflowRepo,
		mcpRepo:       mcpRepo,
		skillRepo:     skillRepo,
	}
}

func (p *RepoCountProvider) CountWorkspaces() (int64, error) {
	return p.workspaceRepo.Count()
}

func (p *RepoCountProvider) CountAllWorkflows() (int64, error) {
	return p.workflowRepo.CountAll()
}

func (p *RepoCountProvider) CountAllMCP() (int64, error) {
	return p.mcpRepo.CountAll()
}

func (p *RepoCountProvider) CountAllSkills() (int64, error) {
	return p.skillRepo.CountAll()
}
