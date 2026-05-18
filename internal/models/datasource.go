package models

import (
	"time"

	"gorm.io/gorm"
)

// DatasourceType 数据源类型枚举
type DatasourceType string

const (
	DatasourceTypePrometheus    DatasourceType = "Prometheus"
	DatasourceTypeJaeger        DatasourceType = "Jaeger"
	DatasourceTypeClickhouse    DatasourceType = "Clickhouse"
	DatasourceTypeElasticsearch DatasourceType = "Elasticsearch"
	DatasourceTypeLoki          DatasourceType = "Loki"
	DatasourceTypeLLM           DatasourceType = "LLM"
	DatasourceTypeSSHAuth      DatasourceType = "SSHAuth"
	DatasourceTypeKubernetes   DatasourceType = "Kubernetes"
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

// DatasourceAuth 认证配置
type DatasourceAuth struct {
	Type       AuthType `json:"type"`       // none, basic, bearer, apikey, ssh_password, ssh_key
	Username   string   `json:"username,omitempty"`
	Password   string   `json:"password,omitempty"`
	Token      string   `json:"token,omitempty"`
	APIKey     string   `json:"apiKey,omitempty"`
	APIKeyIn   string   `json:"apiKeyIn,omitempty"`   // header, query
	APIKeyName string   `json:"apiKeyName,omitempty"`
	PrivateKey string   `json:"privateKey,omitempty"` // SSH私钥（仅SSHAuth类型）
}

// HTTPConfig HTTP数据源配置
type HTTPConfig struct {
	Endpoint string            `json:"endpoint"`           // URL端点
	Headers  map[string]string `json:"headers,omitempty"`  // 自定义请求头
	Auth     DatasourceAuth    `json:"auth"`               // 认证配置
}

// AIConfig AI分析配置
type AIConfig struct {
	API     string `json:"api"`     // API端点
	APIKey  string `json:"apiKey"`  // API密钥
	Model   string `json:"model"`   // 模型名称
	BaseURL string `json:"baseURL,omitempty"` // 可选的base URL
}

// KubeConfig Kubernetes数据源配置
type KubeConfig struct {
	Kubeconfig string `json:"kubeconfig"` // kubeconfig 文件内容
}

// Datasource 数据源模型
type Datasource struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	WorkspaceID string         `json:"workspaceId" gorm:"type:varchar(36);index"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	Description string         `json:"description" gorm:"type:text"`
	Type        DatasourceType `json:"type" gorm:"type:varchar(50);not null"`
	HTTPConfig  string         `json:"httpConfig,omitempty" gorm:"type:text"`
	AIConfig    string         `json:"aiConfig,omitempty" gorm:"type:text"`
	KubeConfig  string         `json:"kubeConfig,omitempty" gorm:"type:text"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

func (Datasource) TableName() string {
	return "v_datasource"
}

// IsValid 检查类型是否有效
func (t DatasourceType) IsValid() bool {
	switch t {
	case DatasourceTypePrometheus, DatasourceTypeJaeger,
		DatasourceTypeClickhouse, DatasourceTypeElasticsearch,
		DatasourceTypeLoki, DatasourceTypeLLM, DatasourceTypeSSHAuth,
		DatasourceTypeKubernetes:
		return true
	default:
		return false
	}
}

// IsHTTPType 判断是否为HTTP类型数据源
func (t DatasourceType) IsHTTPType() bool {
	switch t {
	case DatasourceTypePrometheus, DatasourceTypeJaeger,
		DatasourceTypeClickhouse, DatasourceTypeElasticsearch,
		DatasourceTypeLoki:
		return true
	default:
		return false
	}
}

// IsSSHAuthType 判断是否为SSHAuth类型数据源
func (t DatasourceType) IsSSHAuthType() bool {
	return t == DatasourceTypeSSHAuth
}

// IsKubernetesType 判断是否为Kubernetes类型数据源
func (t DatasourceType) IsKubernetesType() bool {
	return t == DatasourceTypeKubernetes
}

// IsSSHAuthType 判断是否为SSHAuth类型数据源
func (d *Datasource) IsSSHAuthType() bool {
	return d.Type.IsSSHAuthType()
}

// IsLLMType 判断是否为LLM类型数据源
func (t DatasourceType) IsLLMType() bool {
	return t == DatasourceTypeLLM
}

// IsHTTPType 判断是否为HTTP类型数据源
func (d *Datasource) IsHTTPType() bool {
	return d.Type.IsHTTPType()
}

// IsLLMType 判断是否为LLM类型数据源
func (d *Datasource) IsLLMType() bool {
	return d.Type.IsLLMType()
}

// IsKubernetesType 判断是否为Kubernetes类型数据源
func (d *Datasource) IsKubernetesType() bool {
	return d.Type.IsKubernetesType()
}
