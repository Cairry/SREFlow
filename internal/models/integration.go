package models

import (
	"time"

	"gorm.io/gorm"
)

// IntegrationType 集成类型枚举
type IntegrationType string

const (
	IntegrationTypePrometheus    IntegrationType = "Prometheus"
	IntegrationTypeJaeger        IntegrationType = "Jaeger"
	IntegrationTypeClickhouse    IntegrationType = "Clickhouse"
	IntegrationTypeElasticsearch IntegrationType = "Elasticsearch"
	IntegrationTypeLoki          IntegrationType = "Loki"
	IntegrationTypeLLM           IntegrationType = "LLM"
	IntegrationTypeSSHAuth      IntegrationType = "SSHAuth"
	IntegrationTypeKubernetes   IntegrationType = "Kubernetes"
)

// AuthType 认证类型
type AuthType string

const (
	AuthTypeNone        AuthType = "none"
	AuthTypeBasic       AuthType = "basic"
	AuthTypeBearer      AuthType = "bearer"
	AuthTypeAPIKey      AuthType = "apikey"
	AuthTypeSSHPassword AuthType = "ssh_password"
	AuthTypeSSHKey      AuthType = "ssh_key"
)

// IntegrationAuth 认证配置
type IntegrationAuth struct {
	Type       AuthType `json:"type"`       // none, basic, bearer, apikey, ssh_password, ssh_key
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	Token      string   `json:"token,omitempty"`
	APIKey     string   `json:"apiKey,omitempty"`
	APIKeyIn   string   `json:"apiKeyIn,omitempty"`   // header, query
	APIKeyName string   `json:"apiKeyName,omitempty"`
	PrivateKey string   `json:"privateKey,omitempty"` // SSH私钥（仅SSHAuth类型）
}

// HTTPConfig HTTP集成配置
type HTTPConfig struct {
	Endpoint string            `json:"endpoint"`           // URL端点
	Headers  map[string]string `json:"headers,omitempty"`  // 自定义请求头
	Auth     IntegrationAuth    `json:"auth"`               // 认证配置
}

// AIConfig AI分析配置
type AIConfig struct {
	API     string `json:"api"`     // API端点
	APIKey  string `json:"apiKey"`  // API密钥
	BaseURL string `json:"baseURL,omitempty"` // 可选的base URL
}

// KubeConfig Kubernetes集成配置
type KubeConfig struct {
	Kubeconfig string `json:"kubeconfig"` // kubeconfig 文件内容
}

// Integration 集成模型
type Integration struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	WorkspaceID string         `json:"workspaceId" gorm:"type:varchar(36);index"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Type        IntegrationType `json:"type" gorm:"type:varchar(50);not null"`
	HTTPConfig  string         `json:"httpConfig,omitempty" gorm:"type:text"`
	AIConfig    string         `json:"aiConfig,omitempty" gorm:"type:text"`
	KubeConfig  string         `json:"kubeConfig,omitempty" gorm:"type:text"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

func (Integration) TableName() string {
	return "v_integration"
}

// IsValid 检查类型是否有效
func (t IntegrationType) IsValid() bool {
	switch t {
	case IntegrationTypePrometheus, IntegrationTypeJaeger,
		IntegrationTypeClickhouse, IntegrationTypeElasticsearch,
		IntegrationTypeLoki, IntegrationTypeLLM, IntegrationTypeSSHAuth,
		IntegrationTypeKubernetes:
		return true
	default:
		return false
	}
}

// IsHTTPType 判断是否为HTTP类型集成
func (t IntegrationType) IsHTTPType() bool {
	switch t {
	case IntegrationTypePrometheus, IntegrationTypeJaeger,
		IntegrationTypeClickhouse, IntegrationTypeElasticsearch,
		IntegrationTypeLoki:
		return true
	default:
		return false
	}
}

// IsSSHAuthType 判断是否为SSHAuth类型集成
func (t IntegrationType) IsSSHAuthType() bool {
	return t == IntegrationTypeSSHAuth
}

// IsKubernetesType 判断是否为Kubernetes类型集成
func (t IntegrationType) IsKubernetesType() bool {
	return t == IntegrationTypeKubernetes
}

// IsSSHAuthType 判断是否为SSHAuth类型集成
func (d *Integration) IsSSHAuthType() bool {
	return d.Type.IsSSHAuthType()
}

// IsLLMType 判断是否为LLM类型集成
func (t IntegrationType) IsLLMType() bool {
	return t == IntegrationTypeLLM
}

// IsHTTPType 判断是否为HTTP类型集成
func (d *Integration) IsHTTPType() bool {
	return d.Type.IsHTTPType()
}

// IsLLMType 判断是否为LLM类型集成
func (d *Integration) IsLLMType() bool {
	return d.Type.IsLLMType()
}

// IsKubernetesType 判断是否为Kubernetes类型集成
func (d *Integration) IsKubernetesType() bool {
	return d.Type.IsKubernetesType()
}
