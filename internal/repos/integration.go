package repos

import (
	"RCAgent/internal/models"
	"gorm.io/gorm"
)

type (
	IntegrationRepoInter interface {
		Create(integration *models.Integration) error
		GetByID(id string) (*models.Integration, error)
		GetByWorkspaceID(workspaceID string) ([]*models.Integration, error)
		GetByWorkspaceIDAndType(workspaceID string, integrationType models.IntegrationType) ([]*models.Integration, error)
		Update(integration *models.Integration) error
		Delete(id string) error
		DeleteByWorkspaceID(workspaceID string) error
	}

	integrationRepo struct {
		db *gorm.DB
	}
)

func NewGormIntegrationRepo(db *gorm.DB) IntegrationRepoInter {
	return &integrationRepo{db: db}
}

func (r *integrationRepo) Create(integration *models.Integration) error {
	return r.db.Create(integration).Error
}

func (r *integrationRepo) GetByID(id string) (*models.Integration, error) {
	var integration models.Integration
	err := r.db.Where("id = ?", id).First(&integration).Error
	if err != nil {
		return nil, err
	}
	return &integration, nil
}

func (r *integrationRepo) GetByWorkspaceID(workspaceID string) ([]*models.Integration, error) {
	var integrations []*models.Integration
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&integrations).Error
	return integrations, err
}

func (r *integrationRepo) GetByWorkspaceIDAndType(workspaceID string, integrationType models.IntegrationType) ([]*models.Integration, error) {
	var integrations []*models.Integration
	err := r.db.Where("workspace_id = ? AND type = ?", workspaceID, integrationType).Find(&integrations).Error
	return integrations, err
}

func (r *integrationRepo) Update(integration *models.Integration) error {
	return r.db.Save(integration).Error
}

func (r *integrationRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Integration{}).Error
}

func (r *integrationRepo) DeleteByWorkspaceID(workspaceID string) error {
	return r.db.Where("workspace_id = ?", workspaceID).Delete(&models.Integration{}).Error
}
