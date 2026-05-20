package types

import (
	"SREFlow/internal/ctx"
	"SREFlow/internal/models"
	"fmt"
)

// RequestUpdateIntegration 更新集成请求
type RequestUpdateIntegration struct {
	ID         string             `json:"id"`
	Name       string             `json:"name,omitempty"`
	HTTPConfig *models.HTTPConfig `json:"httpConfig,omitempty"`
	AIConfig   *models.AIConfig   `json:"aiConfig,omitempty"`
	KubeConfig *models.KubeConfig `json:"kubeConfig,omitempty"`
}

func (r *RequestUpdateIntegration) Valid(c *ctx.Context) error {
	if r.ID == "" {
		return fmt.Errorf("integration ID is required")
	}
	if r.Name != "" && len(r.Name) > 100 {
		return fmt.Errorf("integration name is too long (max 100 characters)")
	}
	return nil
}

// ResponseIntegration 集成响应
type ResponseIntegration struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspaceId"`
	Name        string                 `json:"name"`
	Type        models.IntegrationType `json:"type"`
	HTTPConfig  *models.HTTPConfig     `json:"httpConfig,omitempty"`
	AIConfig    *models.AIConfig       `json:"aiConfig,omitempty"`
	KubeConfig  *models.KubeConfig     `json:"kubeConfig,omitempty"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// RequestListIntegration 列出集成请求
type RequestListIntegration struct {
	Type string `form:"type"`
}

// RequestTestK8sConnection 测试K8s连接请求
type RequestTestK8sConnection struct {
	Kubeconfig string `json:"kubeconfig"`
}
