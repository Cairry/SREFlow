package repos

import (
	"RCAgent/internal/models"
	"gorm.io/gorm"
)

type (
	DatasourceRepoInter interface {
		Create(datasource *models.Datasource) error
		GetByID(id string) (*models.Datasource, error)
		GetByWorkspaceID(workspaceID string) ([]*models.Datasource, error)
		GetByWorkspaceIDAndType(workspaceID string, datasourceType models.DatasourceType) ([]*models.Datasource, error)
		Update(datasource *models.Datasource) error
		Delete(id string) error
		DeleteByWorkspaceID(workspaceID string) error
	}

	datasourceRepo struct {
		db *gorm.DB
	}
)

func NewGormDatasourceRepo(db *gorm.DB) DatasourceRepoInter {
	return &datasourceRepo{db: db}
}

func (r *datasourceRepo) Create(datasource *models.Datasource) error {
	return r.db.Create(datasource).Error
}

func (r *datasourceRepo) GetByID(id string) (*models.Datasource, error) {
	var datasource models.Datasource
	err := r.db.Where("id = ?", id).First(&datasource).Error
	if err != nil {
		return nil, err
	}
	return &datasource, nil
}

func (r *datasourceRepo) GetByWorkspaceID(workspaceID string) ([]*models.Datasource, error) {
	var datasources []*models.Datasource
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&datasources).Error
	return datasources, err
}

func (r *datasourceRepo) GetByWorkspaceIDAndType(workspaceID string, datasourceType models.DatasourceType) ([]*models.Datasource, error) {
	var datasources []*models.Datasource
	err := r.db.Where("workspace_id = ? AND type = ?", workspaceID, datasourceType).Find(&datasources).Error
	return datasources, err
}

func (r *datasourceRepo) Update(datasource *models.Datasource) error {
	return r.db.Save(datasource).Error
}

func (r *datasourceRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Datasource{}).Error
}

func (r *datasourceRepo) DeleteByWorkspaceID(workspaceID string) error {
	return r.db.Where("workspace_id = ?", workspaceID).Delete(&models.Datasource{}).Error
}
