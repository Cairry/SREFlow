package types

import "fmt"

// RequestCreateSkill 创建Skill请求
type RequestCreateSkill struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

func (r *RequestCreateSkill) Valid() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("name is too long (max 100 characters)")
	}
	if r.Content == "" {
		return fmt.Errorf("content is required")
	}
	return nil
}

// RequestUpdateSkill 更新Skill请求
type RequestUpdateSkill struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Content     string `json:"content,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

func (r *RequestUpdateSkill) Valid() error {
	if r.ID == "" {
		return fmt.Errorf("id is required")
	}
	if r.Name != "" && len(r.Name) > 100 {
		return fmt.Errorf("name is too long (max 100 characters)")
	}
	return nil
}

// ResponseSkill Skill响应
type ResponseSkill struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspaceId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}
