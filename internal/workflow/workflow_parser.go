package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// WorkflowParser 工作流解析器
type WorkflowParser struct{}

// NewWorkflowParser 创建工作流解析器
func NewWorkflowParser() *WorkflowParser {
	return &WorkflowParser{}
}

// ParseWorkflowFromJSON 从JSON解析工作流
func (p *WorkflowParser) ParseWorkflowFromJSON(nodesJSON, edgesJSON string) (*WorkflowTopology, error) {
	// 解析节点
	var nodes []WorkflowNode
	if err := json.Unmarshal([]byte(nodesJSON), &nodes); err != nil {
		return nil, fmt.Errorf("failed to parse nodes: %w", err)
	}

	// 解析边
	var edges []WorkflowEdge
	if err := json.Unmarshal([]byte(edgesJSON), &edges); err != nil {
		return nil, fmt.Errorf("failed to parse edges: %w", err)
	}

	return &WorkflowTopology{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// ParseWorkflowFromModel 从模型解析工作流
func (p *WorkflowParser) ParseWorkflowFromModel(workflow *models.Workflow) (*WorkflowTopology, error) {
	return p.ParseWorkflowFromJSON(workflow.Nodes, workflow.Edges)
}

// ValidateWorkflow 验证工作流
func (p *WorkflowParser) ValidateWorkflow(topology *WorkflowTopology, registry *NodeRegistry) error {
	// 验证节点
	nodeIDs := make(map[string]bool)
	for _, node := range topology.Nodes {
		if node.ID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		if node.Type == "" {
			return fmt.Errorf("node type cannot be empty for node %s", node.ID)
		}

		// 检查节点类型是否已注册
		if _, exists := registry.GetExecutor(node.Type); !exists {
			return fmt.Errorf("unknown node type: %s for node %s", node.Type, node.ID)
		}

		// 验证节点配置
		if err := registry.Validate(node.Type, node.Data); err != nil {
			return fmt.Errorf("invalid config for node %s: %w", node.ID, err)
		}

		nodeIDs[node.ID] = true
	}

	// 验证边
	for _, edge := range topology.Edges {
		if edge.Source == "" {
			return fmt.Errorf("edge source cannot be empty")
		}
		if edge.Target == "" {
			return fmt.Errorf("edge target cannot be empty")
		}

		// 检查源节点是否存在
		if !nodeIDs[edge.Source] {
			return fmt.Errorf("source node %s not found", edge.Source)
		}

		// 检查目标节点是否存在
		if !nodeIDs[edge.Target] {
			return fmt.Errorf("target node %s not found", edge.Target)
		}

		// 检查自环
		if edge.Source == edge.Target {
			return fmt.Errorf("self-loop detected for node %s", edge.Source)
		}
	}

	// 检测孤立节点（没有连接线的节点），但不阻塞工作流
	isolatedNodes := p.detectIsolatedNodes(topology)
	if len(isolatedNodes) > 0 {
		logc.Infof(context.Background(), "[Parser] Found %d isolated nodes without connections: %v (will be skipped)", len(isolatedNodes), isolatedNodes)
		// 不返回错误，允许工作流继续执行，仅跳过孤立节点
	}

	// 检查循环依赖
	if err := p.detectCycles(topology); err != nil {
		return fmt.Errorf("workflow validation failed: %w", err)
	}

	return nil
}

// detectIsolatedNodes 检测没有连接线的孤立节点，返回孤立节点ID列表
func (p *WorkflowParser) detectIsolatedNodes(topology *WorkflowTopology) []string {
	// 计算每个节点的入度和出度
	inDegree := make(map[string]int)
	outDegree := make(map[string]int)

	for _, node := range topology.Nodes {
		inDegree[node.ID] = 0
		outDegree[node.ID] = 0
	}

	for _, edge := range topology.Edges {
		inDegree[edge.Target]++
		outDegree[edge.Source]++
	}

	// 找出孤立节点（入度=0 且 出度=0）
	var isolatedNodes []string
	for _, node := range topology.Nodes {
		if inDegree[node.ID] == 0 && outDegree[node.ID] == 0 {
			isolatedNodes = append(isolatedNodes, node.ID)
		}
	}

	return isolatedNodes
}

// detectCycles 检测循环依赖
func (p *WorkflowParser) detectCycles(topology *WorkflowTopology) error {
	// 构建邻接表
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	// 初始化
	for _, node := range topology.Nodes {
		adj[node.ID] = []string{}
		inDegree[node.ID] = 0
	}

	// 构建图
	for _, edge := range topology.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// 拓扑排序检测循环
	queue := []string{}

	// 找到所有入度为0的节点
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	visitedCount := 0
	for len(queue) > 0 {
		// 取出一个节点
		current := queue[0]
		queue = queue[1:]
		visitedCount++

		// 访问所有邻接节点
		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 如果访问的节点数不等于总节点数，说明存在循环
	if visitedCount != len(topology.Nodes) {
		return fmt.Errorf("cycle detected in workflow")
	}

	return nil
}

// GetStartNodes 获取起始节点
func (p *WorkflowParser) GetStartNodes(topology *WorkflowTopology) []string {
	// 计算每个节点的入度
	inDegree := make(map[string]int)
	for _, node := range topology.Nodes {
		inDegree[node.ID] = 0
	}

	for _, edge := range topology.Edges {
		inDegree[edge.Target]++
	}

	// 返回所有入度为0的节点
	var startNodes []string
	for nodeID, degree := range inDegree {
		if degree == 0 {
			startNodes = append(startNodes, nodeID)
		}
	}

	return startNodes
}

// GetEndNodes 获取结束节点
func (p *WorkflowParser) GetEndNodes(topology *WorkflowTopology) []string {
	// 计算每个节点的出度
	outDegree := make(map[string]int)
	for _, node := range topology.Nodes {
		outDegree[node.ID] = 0
	}

	for _, edge := range topology.Edges {
		outDegree[edge.Source]++
	}

	// 返回所有出度为0的节点
	var endNodes []string
	for nodeID, degree := range outDegree {
		if degree == 0 {
			endNodes = append(endNodes, nodeID)
		}
	}

	return endNodes
}

// GetExecutionOrder 获取执行顺序
func (p *WorkflowParser) GetExecutionOrder(topology *WorkflowTopology) ([]string, error) {
	// 构建邻接表
	adj := make(map[string][]string)
	inDegree := make(map[string]int)

	// 初始化
	for _, node := range topology.Nodes {
		adj[node.ID] = []string{}
		inDegree[node.ID] = 0
	}

	// 构建图
	for _, edge := range topology.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	// 拓扑排序
	queue := []string{}
	order := []string{}

	// 找到所有入度为0的节点
	for nodeID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, nodeID)
		}
	}

	for len(queue) > 0 {
		// 取出一个节点
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		// 访问所有邻接节点
		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 检查是否存在循环
	if len(order) != len(topology.Nodes) {
		return nil, fmt.Errorf("cycle detected in workflow")
	}

	return order, nil
}
