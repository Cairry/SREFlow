package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"log"

	"io"
	"net/http"

	"RCAgent/internal/models"
	"RCAgent/internal/repos"
	"RCAgent/internal/types"
	"RCAgent/pkg/tools"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// IntegrationService 集成服务
type IntegrationService struct {
	repo         repos.IntegrationRepoInter
	workflowRepo repos.WorkflowRepoInter
}

func NewIntegrationService(repo repos.IntegrationRepoInter, workflowRepo repos.WorkflowRepoInter) *IntegrationService {
	return &IntegrationService{repo: repo, workflowRepo: workflowRepo}
}

// defaultIntegrationDefs 定义所有默认集成类型
type defaultIntegrationDef struct {
	Type models.IntegrationType
	Name string
}

var defaultIntegrationDefs = []defaultIntegrationDef{
	{models.IntegrationTypePrometheus, "Prometheus"},
	{models.IntegrationTypeJaeger, "Jaeger"},
	{models.IntegrationTypeClickhouse, "ClickHouse"},
	{models.IntegrationTypeElasticsearch, "Elasticsearch"},
	{models.IntegrationTypeLoki, "Loki"},
	{models.IntegrationTypeLLM, "LLM"},
	{models.IntegrationTypeSSHAuth, "SSH Auth"},
	{models.IntegrationTypeKubernetes, "Kubernetes"},
}

// EnsureDefaultIntegrations 确保工作空间拥有所有默认集成
func (s *IntegrationService) EnsureDefaultIntegrations(workspaceID string) {
	for _, def := range defaultIntegrationDefs {
		existing, err := s.repo.GetByWorkspaceIDAndType(workspaceID, def.Type)
		if err == nil && len(existing) > 0 {
			continue
		}

		integration := &models.Integration{
			ID:          tools.RandId(),
			WorkspaceID: workspaceID,
			Name:        def.Name,
			Type:        def.Type,
		}

		if err := s.repo.Create(integration); err != nil {
			log.Printf("[IntegrationService] Failed to create default integration %s for workspace %s: %v", def.Name, workspaceID, err)
		} else {
			log.Printf("[IntegrationService] Created default integration %s for workspace %s", def.Name, workspaceID)
		}
	}
}

// List 获取集成列表
func (s *IntegrationService) List(c context.Context, workspaceID string, integrationType string) (interface{}, error) {
	var integrations []*models.Integration
	var err error

	if integrationType != "" {
		integrations, err = s.repo.GetByWorkspaceIDAndType(workspaceID, models.IntegrationType(integrationType))
	} else {
		integrations, err = s.repo.GetByWorkspaceID(workspaceID)
	}
	if err != nil {
		return nil, err
	}

	result := make([]*types.ResponseIntegration, 0, len(integrations))
	for _, ds := range integrations {
		result = append(result, s.toResponse(ds))
	}

	return map[string]interface{}{
		"integrations": result,
	}, nil
}

// Get 获取单个集成
func (s *IntegrationService) Get(c context.Context, id string) (interface{}, error) {
	integration, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"integration": s.toResponse(integration),
	}, nil
}

// Update 更新集成
func (s *IntegrationService) Update(c context.Context, req interface{}) (interface{}, error) {
	r := req.(*types.RequestUpdateIntegration)

	integration, err := s.repo.GetByID(r.ID)
	if err != nil {
		return nil, err
	}

	if r.Name != "" {
		integration.Name = r.Name
	}

	if (integration.IsHTTPType() || integration.IsSSHAuthType()) && r.HTTPConfig != nil {
		httpConfigJSON, err := json.Marshal(r.HTTPConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal httpConfig: %w", err)
		}
		integration.HTTPConfig = string(httpConfigJSON)
	}

	if integration.IsLLMType() && r.AIConfig != nil {
		aiConfigJSON, err := json.Marshal(r.AIConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal aiConfig: %w", err)
		}
		integration.AIConfig = string(aiConfigJSON)
	}

	if integration.IsKubernetesType() && r.KubeConfig != nil {
		kubeConfigJSON, err := json.Marshal(r.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kubeConfig: %w", err)
		}
		integration.KubeConfig = string(kubeConfigJSON)
	}

	if err := s.repo.Update(integration); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"integration": s.toResponse(integration),
	}, nil
}

// toResponse 转换为响应结构
func (s *IntegrationService) toResponse(integration *models.Integration) *types.ResponseIntegration {
	resp := &types.ResponseIntegration{
		ID:          integration.ID,
		WorkspaceID: integration.WorkspaceID,
		Name:        integration.Name,
		Type:        integration.Type,
		CreatedAt:   integration.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   integration.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if integration.IsHTTPType() && integration.HTTPConfig != "" {
		var httpConfig models.HTTPConfig
		if err := json.Unmarshal([]byte(integration.HTTPConfig), &httpConfig); err == nil {
			resp.HTTPConfig = &httpConfig
		}
	}

	if integration.IsSSHAuthType() && integration.HTTPConfig != "" {
		var httpConfig models.HTTPConfig
		if err := json.Unmarshal([]byte(integration.HTTPConfig), &httpConfig); err == nil {
			resp.HTTPConfig = &httpConfig
		}
	}

	if integration.IsLLMType() && integration.AIConfig != "" {
		var aiConfig models.AIConfig
		if err := json.Unmarshal([]byte(integration.AIConfig), &aiConfig); err == nil {
			resp.AIConfig = &aiConfig
		}
	}

	if integration.IsKubernetesType() && integration.KubeConfig != "" {
		var kubeConfig models.KubeConfig
		if err := json.Unmarshal([]byte(integration.KubeConfig), &kubeConfig); err == nil {
			resp.KubeConfig = &kubeConfig
		}
	}

	return resp
}

// GetByWorkspaceID 获取 workspace 的集成列表
func (s *IntegrationService) GetByWorkspaceID(c context.Context, workspaceID string) ([]*models.Integration, error) {
	return s.repo.GetByWorkspaceID(workspaceID)
}

// GetByType 获取 workspace 指定类型的集成
func (s *IntegrationService) GetByType(c context.Context, workspaceID string, integrationType models.IntegrationType) ([]*models.Integration, error) {
	return s.repo.GetByWorkspaceIDAndType(workspaceID, integrationType)
}

// GetByID 根据ID获取集成
func (s *IntegrationService) GetByID(c context.Context, id string) (*models.Integration, error) {
	return s.repo.GetByID(id)
}

// buildK8sClient 根据 integration 构建 Kubernetes clientset
func (s *IntegrationService) buildK8sClient(integration *models.Integration) (*kubernetes.Clientset, error) {
	if integration.KubeConfig == "" {
		return nil, fmt.Errorf("kubeconfig is not configured for integration: %s", integration.Name)
	}

	var kubeConfig models.KubeConfig
	if err := json.Unmarshal([]byte(integration.KubeConfig), &kubeConfig); err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig.Kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("failed to build k8s config from kubeconfig: %w", err)
	}

	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	return clientset, nil
}

// GetNamespaces 获取 Kubernetes namespace 列表
func (s *IntegrationService) GetNamespaces(c context.Context, integrationID string) (interface{}, error) {
	integration, err := s.repo.GetByID(integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if !integration.IsKubernetesType() {
		return nil, fmt.Errorf("integration is not Kubernetes type")
	}

	clientset, err := s.buildK8sClient(integration)
	if err != nil {
		return nil, err
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(c, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	result := make([]string, 0, len(namespaces.Items))
	for _, ns := range namespaces.Items {
		result = append(result, ns.Name)
	}

	return map[string]interface{}{
		"namespaces": result,
	}, nil
}

// GetPods 获取指定 namespace 下的 Pod 列表
func (s *IntegrationService) GetPods(c context.Context, integrationID string, namespace string) (interface{}, error) {
	integration, err := s.repo.GetByID(integrationID)
	if err != nil {
		return nil, fmt.Errorf("integration not found: %w", err)
	}

	if !integration.IsKubernetesType() {
		return nil, fmt.Errorf("integration is not Kubernetes type")
	}

	clientset, err := s.buildK8sClient(integration)
	if err != nil {
		return nil, err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(c, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	result := make([]string, 0, len(pods.Items))
	for _, pod := range pods.Items {
		result = append(result, pod.Name)
	}

	return map[string]interface{}{
		"pods": result,
	}, nil
}

// TestK8sConnection 测试 Kubernetes 集成连接
func (s *IntegrationService) TestK8sConnection(c context.Context, kubeconfigContent string) (interface{}, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigContent))
	if err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}

	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s client: %w", err)
	}

	serverVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to k8s cluster: %w", err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(c, metav1.ListOptions{Limit: 1})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	return map[string]interface{}{
		"connected":      true,
		"serverVersion":  fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor),
		"namespaceCount": len(namespaces.Items),
	}, nil
}

// ListModels 获取LLM提供商的可用模型列表
func (s *IntegrationService) ListModels(c context.Context, workspaceID string) (interface{}, error) {
	integrations, err := s.repo.GetByWorkspaceIDAndType(workspaceID, models.IntegrationTypeLLM)
	if err != nil || len(integrations) == 0 {
		return nil, fmt.Errorf("no LLM integration configured for this workspace")
	}

	integration := integrations[0]
	if integration.AIConfig == "" {
		return nil, fmt.Errorf("LLM integration is not configured")
	}

	var aiConfig models.AIConfig
	if err := json.Unmarshal([]byte(integration.AIConfig), &aiConfig); err != nil {
		return nil, fmt.Errorf("failed to parse AI config: %w", err)
	}

	baseURL := aiConfig.BaseURL
	if baseURL == "" && aiConfig.API != "" {
		baseURL = aiConfig.API
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// Ensure baseURL ends without trailing slash
	if baseURL[len(baseURL)-1] == '/' {
		baseURL = baseURL[:len(baseURL)-1]
	}

	modelsURL := baseURL + "/models"

	httpReq, err := http.NewRequestWithContext(c, "GET", modelsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if aiConfig.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+aiConfig.APIKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch models (status %d): %s", resp.StatusCode, string(body))
	}

	var modelsResp struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("failed to parse models response: %w", err)
	}

	models := make([]map[string]string, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		models = append(models, map[string]string{
			"id":   m.ID,
			"name": m.ID,
		})
	}

	return map[string]interface{}{
		"models": models,
	}, nil
}
