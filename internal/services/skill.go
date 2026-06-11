package services

import (
	"context"

	"SREFlow/internal/models"
	"SREFlow/internal/repos"
	"SREFlow/internal/types"
	"SREFlow/pkg/tools"
)

// SkillService Skill服务
type SkillService struct {
	repo repos.SkillRepoInter
}

func (s *SkillService) Init(repo repos.SkillRepoInter) {
	s.repo = repo
}

// List 获取Skill列表
func (s *SkillService) List(c context.Context, workspaceID string) (interface{}, error) {
	skills, err := s.repo.GetByWorkspaceID(workspaceID)
	if err != nil {
		return nil, err
	}

	result := make([]*types.ResponseSkill, 0, len(skills))
	for _, sk := range skills {
		result = append(result, s.toResponse(sk))
	}

	return map[string]interface{}{
		"skills": result,
	}, nil
}

// Get 获取单个Skill
func (s *SkillService) Get(c context.Context, id string) (interface{}, error) {
	skill, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"skill": s.toResponse(skill),
	}, nil
}

// Create 创建Skill
func (s *SkillService) Create(c context.Context, req *types.RequestCreateSkill, workspaceID string) (interface{}, error) {
	skill := &models.Skill{
		ID:          tools.RandId(),
		WorkspaceID: workspaceID,
		Name:        req.Name,
		Description: req.Description,
		Content:     req.Content,
		Enabled:     true,
	}

	if req.Enabled != nil {
		skill.Enabled = *req.Enabled
	}

	if err := s.repo.Create(skill); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"skill": s.toResponse(skill),
	}, nil
}

// Update 更新Skill
func (s *SkillService) Update(c context.Context, req *types.RequestUpdateSkill) (interface{}, error) {
	skill, err := s.repo.GetByID(req.ID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		skill.Name = req.Name
	}
	if req.Description != "" {
		skill.Description = req.Description
	}
	if req.Content != "" {
		skill.Content = req.Content
	}
	if req.Enabled != nil {
		skill.Enabled = *req.Enabled
	}

	if err := s.repo.Update(skill); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"skill": s.toResponse(skill),
	}, nil
}

// Delete 删除Skill
func (s *SkillService) Delete(c context.Context, id string) (interface{}, error) {
	if err := s.repo.Delete(id); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "Skill deleted successfully",
	}, nil
}

// toResponse 转换为响应结构
func (s *SkillService) toResponse(skill *models.Skill) *types.ResponseSkill {
	return &types.ResponseSkill{
		ID:          skill.ID,
		WorkspaceID: skill.WorkspaceID,
		Name:        skill.Name,
		Description: skill.Description,
		Content:     skill.Content,
		Enabled:     skill.Enabled,
		CreatedAt:   skill.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   skill.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}
