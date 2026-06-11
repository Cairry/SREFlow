package models

import (
	"time"

	"gorm.io/gorm"
)

// Skill represents a reusable skill/prompt template for Agent nodes
type Skill struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	WorkspaceID string         `json:"workspaceId" gorm:"type:varchar(36);index"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Content     string         `json:"content" gorm:"type:text"` // Skill system prompt content
	Enabled     bool           `json:"enabled" gorm:"type:bool;default:true"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

func (Skill) TableName() string {
	return "v_skills"
}
