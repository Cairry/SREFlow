package nodes

import (
	"SREFlow/internal/models"
	"context"
	"log"
)

// TopologyExecutor 拓扑查询执行器
type TopologyExecutor struct {
	DefaultNodeExecutor
}

// NewTopologyExecutor 创建拓扑执行器
func NewTopologyExecutor() *TopologyExecutor {
	return &TopologyExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "topology",
			Category:    "data",
			Description: "Query service topology and dependencies",
		},
	}
}

func ExecuteTopology(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	topologyExecutor := NewTopologyExecutor()
	return topologyExecutor.Execute(ctx, nc)
}

func (e *TopologyExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	service := getStr(nodeCtx.Config, "service", "")
	env := getStr(nodeCtx.Config, "environment", "prod")

	log.Printf("[Topology] Querying topology for service: %s (%s)", service, env)

	return newResult(map[string]any{
		"nodes": map[string]any{
			"service":     service,
			"environment": env,
			"count":       0,
		},
	}), nil
}
