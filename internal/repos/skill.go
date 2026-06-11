package repos

import (
	"SREFlow/internal/models"

	"gorm.io/gorm"
)

type (
	SkillRepoInter interface {
		Create(skill *models.Skill) error
		GetByID(id string) (*models.Skill, error)
		GetByWorkspaceID(workspaceID string) ([]*models.Skill, error)
		GetByIDs(ids []string) ([]*models.Skill, error)
		Update(skill *models.Skill) error
		Delete(id string) error
	}

	skillRepo struct {
		db *gorm.DB
	}
)

func NewGormSkillRepo(db *gorm.DB) SkillRepoInter {
	return &skillRepo{db: db}
}

func (r *skillRepo) Create(skill *models.Skill) error {
	return r.db.Create(skill).Error
}

func (r *skillRepo) GetByID(id string) (*models.Skill, error) {
	var skill models.Skill
	err := r.db.Where("id = ?", id).First(&skill).Error
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *skillRepo) GetByWorkspaceID(workspaceID string) ([]*models.Skill, error) {
	var skills []*models.Skill
	err := r.db.Where("workspace_id = ?", workspaceID).Find(&skills).Error
	return skills, err
}

func (r *skillRepo) GetByIDs(ids []string) ([]*models.Skill, error) {
	if len(ids) == 0 {
		return []*models.Skill{}, nil
	}
	var skills []*models.Skill
	err := r.db.Where("id IN ? AND enabled = ?", ids, true).Find(&skills).Error
	return skills, err
}

func (r *skillRepo) Update(skill *models.Skill) error {
	return r.db.Save(skill).Error
}

func (r *skillRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Skill{}).Error
}
