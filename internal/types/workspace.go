package types

import (
	"SREFlow/internal/ctx"
	"SREFlow/internal/models"
	"fmt"
)

// RequestCreateWorkspace 创建工作空间请求
type RequestCreateWorkspace struct {
	Name string `json:"name"`
}

func (create *RequestCreateWorkspace) Valid(ctx *ctx.Context) error {
	if create.Name == "" {
		return fmt.Errorf("workspace name is required")
	}
	if len(create.Name) > 100 {
		return fmt.Errorf("workspace name is too long (max 100 characters)")
	}

	return nil
}

// RequestUpdateWorkspace 更新工作空间请求
type RequestUpdateWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (update *RequestUpdateWorkspace) Valid(ctx *ctx.Context) error {
	if update.ID == "" {
		return fmt.Errorf("workspace ID is required")
	}

	// 验证工作空间是否存在
	var count int64
	db := ctx.Database.MySQL()
	db.Model(&models.Workspace{}).Where("id = ?", update.ID).Count(&count)
	if count <= 0 {
		return fmt.Errorf("workspace not found")
	}

	// 如果要更新名称，验证名称不为空
	if update.Name != "" && len(update.Name) > 100 {
		return fmt.Errorf("workspace name is too long (max 100 characters)")
	}

	return nil
}
