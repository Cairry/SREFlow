package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"RCAgent/internal/models"
	"RCAgent/internal/repos"
	"RCAgent/internal/types"
	"RCAgent/pkg/tools"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// DatasourceService 数据源服务
type DatasourceService struct {
	repo         repos.DatasourceRepoInter
	workflowRepo repos.WorkflowRepoInter
}

func NewDatasourceService(repo repos.DatasourceRepoInter, workflowRepo repos.WorkflowRepoInter) *DatasourceService {
	return &DatasourceService{repo: repo, workflowRepo: workflowRepo}
}

// List 获取数据源列表
func (s *DatasourceService) List(c context.Context, workspaceID string, datasourceType string) (interface{}, error) {
	var datasources []*models.Datasource
	var err error

	if datasourceType != "" {
		datasources, err = s.repo.GetByWorkspaceIDAndType(workspaceID, models.DatasourceType(datasourceType))
	} else {
		datasources, err = s.repo.GetByWorkspaceID(workspaceID)
	}
	if err != nil {
		return nil, err
	}

	result := make([]*types.ResponseDatasource, 0, len(datasources))
	for _, ds := range datasources {
		result = append(result, s.toResponse(ds))
	}

	return map[string]interface{}{
		"datasources": result,
	}, nil
}

// Get 获取单个数据源
func (s *DatasourceService) Get(c context.Context, id string) (interface{}, error) {
	datasource, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"datasource": s.toResponse(datasource),
	}, nil
}

// Create 创建数据源
func (s *DatasourceService) Create(c context.Context, req interface{}, workspaceID string) (interface{}, error) {
	r := req.(*types.RequestCreateDatasource)

	datasource := &models.Datasource{
		ID:          tools.RandId(),
		WorkspaceID: workspaceID,
		Name:        r.Name,
		Description: r.Description,
		Type:        r.Type,
	}

	if (r.Type.IsHTTPType() || r.Type.IsSSHAuthType()) && r.HTTPConfig != nil {
		httpConfigJSON, err := json.Marshal(r.HTTPConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal httpConfig: %w", err)
		}
		datasource.HTTPConfig = string(httpConfigJSON)
	}

	if r.Type.IsLLMType() && r.AIConfig != nil {
		aiConfigJSON, err := json.Marshal(r.AIConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal aiConfig: %w", err)
		}
		datasource.AIConfig = string(aiConfigJSON)
	}

	if r.Type.IsKubernetesType() && r.KubeConfig != nil {
		kubeConfigJSON, err := json.Marshal(r.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kubeConfig: %w", err)
		}
		datasource.KubeConfig = string(kubeConfigJSON)
	}

	if err := s.repo.Create(datasource); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"datasource": s.toResponse(datasource),
	}, nil
}

// Update 更新数据源
func (s *DatasourceService) Update(c context.Context, req interface{}) (interface{}, error) {
	r := req.(*types.RequestUpdateDatasource)

	datasource, err := s.repo.GetByID(r.ID)
	if err != nil {
		return nil, err
	}

	if r.Name != "" {
		datasource.Name = r.Name
	}
	if r.Description != "" {
		datasource.Description = r.Description
	}

	if (datasource.IsHTTPType() || datasource.IsSSHAuthType()) && r.HTTPConfig != nil {
		httpConfigJSON, err := json.Marshal(r.HTTPConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal httpConfig: %w", err)
		}
		datasource.HTTPConfig = string(httpConfigJSON)
	}

	if datasource.IsLLMType() && r.AIConfig != nil {
		aiConfigJSON, err := json.Marshal(r.AIConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal aiConfig: %w", err)
		}
		datasource.AIConfig = string(aiConfigJSON)
	}

	if datasource.IsKubernetesType() && r.KubeConfig != nil {
		kubeConfigJSON, err := json.Marshal(r.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal kubeConfig: %w", err)
		}
		datasource.KubeConfig = string(kubeConfigJSON)
	}

	if err := s.repo.Update(datasource); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"datasource": s.toResponse(datasource),
	}, nil
}

// Delete 删除数据源
func (s *DatasourceService) Delete(c context.Context, id string, workspaceID string) (interface{}, error) {
	// 检查是否有 workflow 使用该 datasource
	usedBy, err := s.workflowRepo.GetByDatasourceID(workspaceID, id)
	if err != nil {
		return nil, err
	}
	if len(usedBy) > 0 {
		names := make([]string, len(usedBy))
		for i, wf := range usedBy {
			names[i] = wf.Name
		}
		return nil, fmt.Errorf("cannot delete datasource: it is used by workflow(s): %v", names)
	}

	if err := s.repo.Delete(id); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"message": "datasource deleted successfully",
	}, nil
}

// toResponse 转换为响应结构
func (s *DatasourceService) toResponse(datasource *models.Datasource) *types.ResponseDatasource {
	resp := &types.ResponseDatasource{
		ID:          datasource.ID,
		WorkspaceID: datasource.WorkspaceID,
		Name:        datasource.Name,
		Description: datasource.Description,
		Type:        datasource.Type,
		CreatedAt:   datasource.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   datasource.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	if datasource.IsHTTPType() && datasource.HTTPConfig != "" {
		var httpConfig models.HTTPConfig
		if err := json.Unmarshal([]byte(datasource.HTTPConfig), &httpConfig); err == nil {
			resp.HTTPConfig = &httpConfig
		}
	}

	if datasource.IsSSHAuthType() && datasource.HTTPConfig != "" {
		var httpConfig models.HTTPConfig
		if err := json.Unmarshal([]byte(datasource.HTTPConfig), &httpConfig); err == nil {
			resp.HTTPConfig = &httpConfig
		}
	}

	if datasource.IsLLMType() && datasource.AIConfig != "" {
		var aiConfig models.AIConfig
		if err := json.Unmarshal([]byte(datasource.AIConfig), &aiConfig); err == nil {
			resp.AIConfig = &aiConfig
		}
	}

	if datasource.IsKubernetesType() && datasource.KubeConfig != "" {
		var kubeConfig models.KubeConfig
		if err := json.Unmarshal([]byte(datasource.KubeConfig), &kubeConfig); err == nil {
			resp.KubeConfig = &kubeConfig
		}
	}

	return resp
}

// GetByWorkspaceID 获取 workspace 的数据源列表
func (s *DatasourceService) GetByWorkspaceID(c context.Context, workspaceID string) ([]*models.Datasource, error) {
	return s.repo.GetByWorkspaceID(workspaceID)
}

// GetByType 获取 workspace 指定类型的数据源
func (s *DatasourceService) GetByType(c context.Context, workspaceID string, datasourceType models.DatasourceType) ([]*models.Datasource, error) {
	return s.repo.GetByWorkspaceIDAndType(workspaceID, datasourceType)
}

// GetByID 根据ID获取数据源
func (s *DatasourceService) GetByID(c context.Context, id string) (*models.Datasource, error) {
	return s.repo.GetByID(id)
}

// buildK8sClient 根据 datasource 构建 Kubernetes clientset
func (s *DatasourceService) buildK8sClient(datasource *models.Datasource) (*kubernetes.Clientset, error) {
	if datasource.KubeConfig == "" {
		return nil, fmt.Errorf("kubeconfig is not configured for datasource: %s", datasource.Name)
	}

	var kubeConfig models.KubeConfig
	if err := json.Unmarshal([]byte(datasource.KubeConfig), &kubeConfig); err != nil {
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
func (s *DatasourceService) GetNamespaces(c context.Context, datasourceID string) (interface{}, error) {
	datasource, err := s.repo.GetByID(datasourceID)
	if err != nil {
		return nil, fmt.Errorf("datasource not found: %w", err)
	}

	if !datasource.IsKubernetesType() {
		return nil, fmt.Errorf("datasource is not Kubernetes type")
	}

	clientset, err := s.buildK8sClient(datasource)
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
func (s *DatasourceService) GetPods(c context.Context, datasourceID string, namespace string) (interface{}, error) {
	datasource, err := s.repo.GetByID(datasourceID)
	if err != nil {
		return nil, fmt.Errorf("datasource not found: %w", err)
	}

	if !datasource.IsKubernetesType() {
		return nil, fmt.Errorf("datasource is not Kubernetes type")
	}

	clientset, err := s.buildK8sClient(datasource)
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

// TestK8sConnection 测试 Kubernetes 数据源连接
func (s *DatasourceService) TestK8sConnection(c context.Context, kubeconfigContent string) (interface{}, error) {
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
		"connected":    true,
		"serverVersion": fmt.Sprintf("%s.%s", serverVersion.Major, serverVersion.Minor),
		"namespaceCount": len(namespaces.Items),
	}, nil
}
