package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"RCAgent/internal/models"
	"RCAgent/internal/repos"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesExecutor Kubernetes节点执行器
type KubernetesExecutor struct {
	DefaultNodeExecutor
	integrationRepo repos.IntegrationRepoInter
}

// NewKubernetesExecutor 创建Kubernetes执行器
func NewKubernetesExecutor(repo repos.IntegrationRepoInter) *KubernetesExecutor {
	return &KubernetesExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Kubernetes",
			Category:    "data",
			Description: "Query Kubernetes pod status and events",
		},
		integrationRepo: repo,
	}
}

// NewKubernetesExecutorNoRepo 创建不带repo的Kubernetes执行器（用于注册）
func NewKubernetesExecutorNoRepo() *KubernetesExecutor {
	return &KubernetesExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "Kubernetes",
			Category:    "data",
			Description: "Query Kubernetes pod status and events",
		},
	}
}

func (e *KubernetesExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	return ExecuteKubernetes(ctx, nodeCtx, e.integrationRepo)
}

// ExecuteKubernetes 执行Kubernetes查询
func ExecuteKubernetes(ctx context.Context, nc *models.NodeContext, integrationRepo repos.IntegrationRepoInter) (*models.NodeResult, error) {
	namespace := getStr(nc.Config, "namespace", "")
	pod := getStr(nc.Config, "pod", "all")
	resourceType := getStr(nc.Config, "resourceType", "status")

	if namespace == "" {
		return errorResult("namespace is required - please select a namespace"), nil
	}

	// 获取 kubeconfig
	kubeconfigStr := getStr(nc.Config, "kubeconfig", "")
	if kubeconfigStr == "" {
		return errorResult("kubeconfig is not configured - please check integration configuration"), nil
	}

	// 构建 k8s client
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigStr))
	if err != nil {
		return errorResult(fmt.Sprintf("failed to build k8s config: %v", err)), nil
	}
	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errorResult(fmt.Sprintf("failed to create k8s client: %v", err)), nil
	}

	switch resourceType {
	case "status":
		return executePodStatus(ctx, clientset, namespace, pod)
	case "event":
		return executePodEvents(ctx, clientset, namespace, pod)
	default:
		return errorResult(fmt.Sprintf("unknown resourceType: %s, must be 'status' or 'event'", resourceType)), nil
	}
}

// executePodStatus 获取 Pod 状态
func executePodStatus(ctx context.Context, clientset *kubernetes.Clientset, namespace, pod string) (*models.NodeResult, error) {
	if pod == "all" {
		pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return errorResult(fmt.Sprintf("failed to list pods: %v", err)), nil
		}

		podStatuses := make([]map[string]interface{}, 0, len(pods.Items))
		for _, p := range pods.Items {
			podStatuses = append(podStatuses, formatPodStatus(&p))
		}

		log.Printf("[Kubernetes] Listed %d pods in namespace %s", len(podStatuses), namespace)

		return newResult(map[string]any{
			"pods":   podStatuses,
			"count":  len(podStatuses),
			"namespace": namespace,
		}), nil
	}

	// 获取单个 Pod
	p, err := clientset.CoreV1().Pods(namespace).Get(ctx, pod, metav1.GetOptions{})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to get pod %s/%s: %v", namespace, pod, err)), nil
	}

	log.Printf("[Kubernetes] Got pod %s/%s status", namespace, pod)

	return newResult(map[string]any{
		"pod":      formatPodStatus(p),
		"namespace": namespace,
	}), nil
}

// executePodEvents 获取 Pod 相关事件
func executePodEvents(ctx context.Context, clientset *kubernetes.Clientset, namespace, pod string) (*models.NodeResult, error) {
	if pod == "all" {
		events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			return errorResult(fmt.Sprintf("failed to list events: %v", err)), nil
		}

		eventList := make([]map[string]interface{}, 0, len(events.Items))
		for _, e := range events.Items {
			eventList = append(eventList, formatEvent(&e))
		}

		log.Printf("[Kubernetes] Listed %d events in namespace %s", len(eventList), namespace)

		return newResult(map[string]any{
			"events":    eventList,
			"count":     len(eventList),
			"namespace": namespace,
		}), nil
	}

	// 获取特定 Pod 的事件（通过 field selector 过滤）
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", pod)
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return errorResult(fmt.Sprintf("failed to list events for pod %s/%s: %v", namespace, pod, err)), nil
	}

	eventList := make([]map[string]interface{}, 0, len(events.Items))
	for _, e := range events.Items {
		eventList = append(eventList, formatEvent(&e))
	}

	log.Printf("[Kubernetes] Listed %d events for pod %s/%s", len(eventList), namespace, pod)

	return newResult(map[string]any{
		"events":    eventList,
		"count":     len(eventList),
		"namespace": namespace,
		"pod":       pod,
	}), nil
}

// formatPodStatus 格式化 Pod 状态信息
func formatPodStatus(pod interface{}) map[string]interface{} {
	type podInfo interface {
		GetName() string
		GetNamespace() string
		GetCreationTimestamp() metav1.Time
	}

	// 使用 JSON 序列化/反序列化来提取字段
	data, err := json.Marshal(pod)
	if err != nil {
		return map[string]interface{}{"error": "failed to marshal pod"}
	}

	var podMap map[string]interface{}
	if err := json.Unmarshal(data, &podMap); err != nil {
		return map[string]interface{}{"error": "failed to unmarshal pod"}
	}

	// 提取关键字段
	result := map[string]interface{}{
		"name":              podMap["metadata"].(map[string]interface{})["name"],
		"namespace":         podMap["metadata"].(map[string]interface{})["namespace"],
		"creationTimestamp": podMap["metadata"].(map[string]interface{})["creationTimestamp"],
	}

	if status, ok := podMap["status"].(map[string]interface{}); ok {
		result["phase"] = status["phase"]
		result["podIP"] = status["podIP"]
		result["hostIP"] = status["hostIP"]
		result["startTime"] = status["startTime"]
		result["message"] = status["message"]
		result["reason"] = status["reason"]

		// 容器状态
		if containerStatuses, ok := status["containerStatuses"].([]interface{}); ok {
			containers := make([]map[string]interface{}, 0, len(containerStatuses))
			for _, cs := range containerStatuses {
				if csMap, ok := cs.(map[string]interface{}); ok {
					container := map[string]interface{}{
						"name":  csMap["name"],
						"ready": csMap["ready"],
						"image": csMap["image"],
					}
					if state, ok := csMap["state"].(map[string]interface{}); ok {
						if waiting, ok := state["waiting"].(map[string]interface{}); ok {
							container["state"] = "waiting"
							container["reason"] = waiting["reason"]
							container["message"] = waiting["message"]
						} else if running, ok := state["running"].(map[string]interface{}); ok {
							container["state"] = "running"
							container["startedAt"] = running["startedAt"]
						} else if terminated, ok := state["terminated"].(map[string]interface{}); ok {
							container["state"] = "terminated"
							container["reason"] = terminated["reason"]
							container["exitCode"] = terminated["exitCode"]
						}
					}
					containers = append(containers, container)
				}
			}
			result["containers"] = containers
		}
	}

	// 标签
	if metadata, ok := podMap["metadata"].(map[string]interface{}); ok {
		if labels, ok := metadata["labels"].(map[string]interface{}); ok {
			result["labels"] = labels
		}
	}

	return result
}

// formatEvent 格式化事件信息
func formatEvent(event interface{}) map[string]interface{} {
	data, err := json.Marshal(event)
	if err != nil {
		return map[string]interface{}{"error": "failed to marshal event"}
	}

	var eventMap map[string]interface{}
	if err := json.Unmarshal(data, &eventMap); err != nil {
		return map[string]interface{}{"error": "failed to unmarshal event"}
	}

	result := map[string]interface{}{
		"message":   eventMap["message"],
		"reason":    eventMap["reason"],
		"type":      eventMap["type"],
		"count":     eventMap["count"],
		"eventTime": eventMap["lastTimestamp"],
	}

	if involved, ok := eventMap["involvedObject"].(map[string]interface{}); ok {
		result["involvedObject"] = map[string]interface{}{
			"kind":      involved["kind"],
			"name":      involved["name"],
			"namespace": involved["namespace"],
		}
	}

	if source, ok := eventMap["source"].(map[string]interface{}); ok {
		result["source"] = source["component"]
	}

	return result
}
