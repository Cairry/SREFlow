package models

import (
	"time"

	"gorm.io/gorm"
)

// Workspace 工作空间模型
type Workspace struct {
	ID        string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name      string         `json:"name" gorm:"type:varchar(100);not null"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName 指定表名
func (Workspace) TableName() string {
	return "v_workspaces"
}
