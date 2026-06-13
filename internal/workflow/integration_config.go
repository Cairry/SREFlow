package workflow

import (
	"encoding/json"

	"SREFlow/internal/models"
)

// mergeIntegrationConfig 从 integration 合并配置到节点配置
func (e *WorkflowExecutor) mergeIntegrationConfig(ctx *ExecutionContext, node *ExecutableNode, config map[string]any) map[string]any {
	// 如果没有 integration repo，直接返回原配置
	if e.integrationRepo == nil {
		return config
	}

	// 自动按类型解析 integration（同一 workspace 中同类型 integration 唯一）
	integrationType := e.nodeTypeToIntegrationType(node.Type)
	if integrationType == "" {
		return config
	}
	integrations, err := e.integrationRepo.GetByWorkspaceIDAndType(ctx.WorkspaceID, integrationType)
	if err != nil || len(integrations) == 0 {
		return config
	}
	integration := integrations[0]

	// 根据 integration 类型合并配置
	switch integration.Type {
	case models.IntegrationTypeLLM:
		config = e.mergeAIConfig(config, integration)
	case models.IntegrationTypePrometheus:
		config = e.mergePrometheusConfig(config, integration)
	case models.IntegrationTypeSSHAuth:
		config = e.mergeSSHAuth(config, integration)
	case models.IntegrationTypeKubernetes:
		config = e.mergeKubeConfig(config, integration)
	default:
		// 其他 HTTP 类型的数据源
		if integration.Type.IsHTTPType() {
			config = e.mergeHTTPConfig(config, integration, node.Type)
		}
	}

	return config
}

// mergeAIConfig 合并 AI integration 配置
func (e *WorkflowExecutor) mergeAIConfig(config map[string]any, integration *models.Integration) map[string]any {
	if integration.AIConfig == "" {
		return config
	}

	var aiConfig models.AIConfig
	if err := json.Unmarshal([]byte(integration.AIConfig), &aiConfig); err != nil {
		return config
	}

	// 数据源配置优先于节点配置
	if aiConfig.APIKey != "" {
		config["apiKey"] = aiConfig.APIKey
	}
	if aiConfig.BaseURL != "" {
		config["baseURL"] = aiConfig.BaseURL
	} else if aiConfig.API != "" {
		config["baseURL"] = aiConfig.API
	}

	return config
}

// mergePrometheusConfig 合并 Prometheus integration 配置
func (e *WorkflowExecutor) mergePrometheusConfig(config map[string]any, integration *models.Integration) map[string]any {
	if integration.HTTPConfig == "" {
		return config
	}

	var httpConfig models.HTTPConfig
	if err := json.Unmarshal([]byte(integration.HTTPConfig), &httpConfig); err != nil {
		return config
	}

	if httpConfig.Endpoint != "" {
		config["prometheusUrl"] = httpConfig.Endpoint
	}

	// 合并认证配置
	config = e.mergeHTTPAuth(config, httpConfig.Auth)

	return config
}

// mergeHTTPConfig 合并通用 HTTP integration 配置
func (e *WorkflowExecutor) mergeHTTPConfig(config map[string]any, integration *models.Integration, nodeType string) map[string]any {
	if integration.HTTPConfig == "" {
		return config
	}

	var httpConfig models.HTTPConfig
	if err := json.Unmarshal([]byte(integration.HTTPConfig), &httpConfig); err != nil {
		return config
	}

	// 根据节点类型设置 URL 配置
	switch nodeType {
	case "prometheus":
		if httpConfig.Endpoint != "" {
			config["prometheusUrl"] = httpConfig.Endpoint
		}
	case "jaeger":
		if httpConfig.Endpoint != "" {
			config["jaegerUrl"] = httpConfig.Endpoint
		}
	case "clickhouse":
		if httpConfig.Endpoint != "" {
			config["clickhouseUrl"] = httpConfig.Endpoint
		}
	case "elasticsearch":
		if httpConfig.Endpoint != "" {
			config["elasticsearchUrl"] = httpConfig.Endpoint
		}
	case "loki":
		if httpConfig.Endpoint != "" {
			config["lokiUrl"] = httpConfig.Endpoint
		}
	}

	// 合并认证配置
	config = e.mergeHTTPAuth(config, httpConfig.Auth)

	return config
}

// mergeHTTPAuth 合并 HTTP 认证配置
func (e *WorkflowExecutor) mergeHTTPAuth(config map[string]any, auth models.IntegrationAuth) map[string]any {
	switch auth.Type {
	case models.AuthTypeBearer:
		if auth.Token != "" {
			config["authToken"] = auth.Token
		}
	case models.AuthTypeBasic:
		if auth.Username != "" {
			config["authUsername"] = auth.Username
		}
		if auth.Password != "" {
			config["authPassword"] = auth.Password
		}
	case models.AuthTypeAPIKey:
		if auth.APIKey != "" {
			config["authAPIKey"] = auth.APIKey
			config["authAPIKeyName"] = auth.APIKeyName
			config["authAPIKeyIn"] = auth.APIKeyIn
		}
	}
	return config
}

func (e *WorkflowExecutor) mergeSSHAuth(config map[string]any, integration *models.Integration) map[string]any {
	if integration.HTTPConfig == "" {
		return config
	}

	var httpConfig models.HTTPConfig
	if err := json.Unmarshal([]byte(integration.HTTPConfig), &httpConfig); err != nil {
		return config
	}

	auth := httpConfig.Auth

	// 合并认证类型
	if auth.Type != "" {
		config["authType"] = string(auth.Type)
	}

	// 合并用户名
	if auth.Username != "" {
		config["authUsername"] = auth.Username
	}

	// 合并密码
	if auth.Password != "" {
		config["authPassword"] = auth.Password
	}
	// 合并私钥
	if auth.PrivateKey != "" {
		config["authPrivateKey"] = auth.PrivateKey
	}

	return config
}

// mergeKubeConfig 合并 Kubernetes integration 配置
func (e *WorkflowExecutor) mergeKubeConfig(config map[string]any, integration *models.Integration) map[string]any {
	if integration.KubeConfig == "" {
		return config
	}

	var kubeConfig models.KubeConfig
	if err := json.Unmarshal([]byte(integration.KubeConfig), &kubeConfig); err != nil {
		return config
	}

	if kubeConfig.Kubeconfig != "" {
		config["kubeconfig"] = kubeConfig.Kubeconfig
	}

	return config
}

// nodeTypeToIntegrationType 将工作流节点类型映射到 Integration 类型
func (e *WorkflowExecutor) nodeTypeToIntegrationType(nodeType string) models.IntegrationType {
	switch nodeType {
	case "LLM":
		return models.IntegrationTypeLLM
	case "Prometheus":
		return models.IntegrationTypePrometheus
	case "Jaeger":
		return models.IntegrationTypeJaeger
	case "Elasticsearch":
		return models.IntegrationTypeElasticsearch
	case "Loki":
		return models.IntegrationTypeLoki
	case "ClickHouse":
		return models.IntegrationTypeClickhouse
	case "SmartRouter":
		return models.IntegrationTypeLLM
	case "Agent":
		return models.IntegrationTypeLLM
	case "SSH":
		return models.IntegrationTypeSSHAuth
	case "Kubernetes":
		return models.IntegrationTypeKubernetes
	default:
		return ""
	}
}
