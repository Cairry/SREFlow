package repos

import (
	"RCAgent/internal/models"

	"gorm.io/gorm"
)

// WorkspaceRepoInter 工作空间仓储接口
type WorkspaceRepoInter interface {
	Create(workspace *models.Workspace) error
	GetByID(id string) (*models.Workspace, error)
	GetAll() ([]*models.Workspace, error)
	Update(workspace *models.Workspace) error
	Delete(id string) error
}

// GormWorkspaceRepo 基于 GORM 的工作空间仓储实现
type GormWorkspaceRepo struct {
	db *gorm.DB
}

// NewGormWorkspaceRepo 创建工作空间仓储
func NewGormWorkspaceRepo(db *gorm.DB) *GormWorkspaceRepo {
	return &GormWorkspaceRepo{db: db}
}

func (r *GormWorkspaceRepo) Create(workspace *models.Workspace) error {
	return r.db.Create(workspace).Error
}

func (r *GormWorkspaceRepo) GetByID(id string) (*models.Workspace, error) {
	var workspace models.Workspace
	if err := r.db.First(&workspace, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

func (r *GormWorkspaceRepo) GetAll() ([]*models.Workspace, error) {
	var workspaces []*models.Workspace
	if err := r.db.Find(&workspaces).Error; err != nil {
		return nil, err
	}
	return workspaces, nil
}

func (r *GormWorkspaceRepo) Update(workspace *models.Workspace) error {
	return r.db.Save(workspace).Error
}

func (r *GormWorkspaceRepo) Delete(id string) error {
	return r.db.Delete(&models.Workspace{}, "id = ?", id).Error
}
