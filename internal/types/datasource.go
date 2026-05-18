package types

import (
	"fmt"
	"RCAgent/internal/ctx"
	"RCAgent/internal/models"
)

// RequestCreateDatasource 创建数据源请求
type RequestCreateDatasource struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Type        models.DatasourceType `json:"type"`
	HTTPConfig  *models.HTTPConfig    `json:"httpConfig,omitempty"`
	AIConfig    *models.AIConfig      `json:"aiConfig,omitempty"`
	KubeConfig  *models.KubeConfig    `json:"kubeConfig,omitempty"`
}

func (r *RequestCreateDatasource) Valid(c *ctx.Context) error {
	if r.Name == "" {
		return fmt.Errorf("datasource name is required")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("datasource name is too long (max 100 characters)")
	}
	if !r.Type.IsValid() {
		return fmt.Errorf("invalid datasource type: %s, valid types are: prometheus, jaeger, clickhouse, elasticsearch, loki, llm, sshauth, kubernetes", r.Type)
	}
	if r.Type.IsHTTPType() && r.HTTPConfig == nil {
		return fmt.Errorf("httpConfig is required for type: %s", r.Type)
	}
	if r.Type.IsHTTPType() && r.HTTPConfig != nil {
		if r.HTTPConfig.Endpoint == "" {
			return fmt.Errorf("httpConfig.endpoint is required")
		}
	}
	if r.Type.IsLLMType() && r.AIConfig == nil {
		return fmt.Errorf("aiConfig is required for type: llm")
	}
	if r.Type.IsLLMType() && r.AIConfig != nil {
		if r.AIConfig.API == "" {
			return fmt.Errorf("aiConfig.api is required")
		}
		if r.AIConfig.APIKey == "" {
			return fmt.Errorf("aiConfig.apiKey is required")
		}
		if r.AIConfig.Model == "" {
			return fmt.Errorf("aiConfig.model is required")
		}
	}
	if r.Type.IsSSHAuthType() && r.HTTPConfig == nil {
		return fmt.Errorf("httpConfig is required for type: sshauth")
	}
	if r.Type.IsSSHAuthType() && r.HTTPConfig != nil {
		if r.HTTPConfig.Auth.Username == "" {
			return fmt.Errorf("httpConfig.auth.username is required for sshauth type")
		}
		if r.HTTPConfig.Auth.Type != models.AuthTypeSSHPassword && r.HTTPConfig.Auth.Type != models.AuthTypeSSHKey {
			return fmt.Errorf("httpConfig.auth.type must be 'ssh_password' or 'ssh_key' for sshauth type")
		}
		if r.HTTPConfig.Auth.Type == models.AuthTypeSSHPassword && r.HTTPConfig.Auth.Password == "" {
			return fmt.Errorf("password is required when auth type is 'ssh_password'")
		}
		if r.HTTPConfig.Auth.Type == models.AuthTypeSSHKey && r.HTTPConfig.Auth.PrivateKey == "" {
			return fmt.Errorf("privateKey is required when auth type is 'ssh_key'")
		}
	}
	if r.Type.IsKubernetesType() && r.KubeConfig == nil {
		return fmt.Errorf("kubeConfig is required for type: kubernetes")
	}
	if r.Type.IsKubernetesType() && r.KubeConfig != nil {
		if r.KubeConfig.Kubeconfig == "" {
			return fmt.Errorf("kubeConfig.kubeconfig is required")
		}
	}
	return nil
}

// RequestUpdateDatasource 更新数据源请求
type RequestUpdateDatasource struct {
	ID          string             `json:"id"`
	Name        string             `json:"name,omitempty"`
	Description string             `json:"description,omitempty"`
	HTTPConfig  *models.HTTPConfig `json:"httpConfig,omitempty"`
	AIConfig    *models.AIConfig   `json:"aiConfig,omitempty"`
	KubeConfig  *models.KubeConfig `json:"kubeConfig,omitempty"`
}

func (r *RequestUpdateDatasource) Valid(c *ctx.Context) error {
	if r.ID == "" {
		return fmt.Errorf("datasource ID is required")
	}
	if r.Name != "" && len(r.Name) > 100 {
		return fmt.Errorf("datasource name is too long (max 100 characters)")
	}
	return nil
}

// ResponseDatasource 数据源响应
type ResponseDatasource struct {
	ID          string                `json:"id"`
	WorkspaceID string                `json:"workspaceId"`
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Type        models.DatasourceType `json:"type"`
	HTTPConfig  *models.HTTPConfig    `json:"httpConfig,omitempty"`
	AIConfig    *models.AIConfig      `json:"aiConfig,omitempty"`
	KubeConfig  *models.KubeConfig    `json:"kubeConfig,omitempty"`
	CreatedAt   string                `json:"createdAt"`
	UpdatedAt   string                `json:"updatedAt"`
}

// RequestListDatasource 列出数据源请求
type RequestListDatasource struct {
	Type string `form:"type"`
}

// RequestTestK8sConnection 测试K8s连接请求
type RequestTestK8sConnection struct {
	Kubeconfig string `json:"kubeconfig"`
}
