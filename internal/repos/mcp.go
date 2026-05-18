package repos

import (
	"RCAgent/internal/models"
	"gorm.io/gorm"
)

type (
	MCPServerRepoInter interface {
		Create(server *models.MCPServer) error
		GetByID(id string) (*models.MCPServer, error)
		GetByWorkspaceID(workspaceID string) ([]*models.MCPServer, error)
		Update(server *models.MCPServer) error
		Delete(id string) error
	}

	mcpServerRepo struct {
		db *gorm.DB
	}
)

func NewGormMCPServerRepo(db *gorm.DB) MCPServerRepoInter {
	return &mcpServerRepo{db: db}
}

func (r *mcpServerRepo) Create(server *models.MCPServer) error {
	return r.db.Create(server).Error
}

func (r *mcpServerRepo) GetByID(id string) (*models.MCPServer, error) {
	var server models.MCPServer
	err := r.db.Where("id = ?", id).First(&server).Error
	if err != nil {
		return nil, err
	}
	return &server, nil
}

func (r *mcpServerRepo) GetByWorkspaceID(workspaceID string) ([]*models.MCPServer, error) {
	var servers []*models.MCPServer
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&servers).Error
	return servers, err
}

func (r *mcpServerRepo) Update(server *models.MCPServer) error {
	return r.db.Save(server).Error
}

func (r *mcpServerRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.MCPServer{}).Error
}
