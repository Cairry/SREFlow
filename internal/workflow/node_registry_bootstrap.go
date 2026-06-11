package workflow

import (
	"SREFlow/internal/repos"
	"SREFlow/internal/workflow/nodes"
)

// BootstrapNodeRegistry 初始化节点注册器
func BootstrapNodeRegistry(mcpRepo repos.MCPServerRepoInter, skillRepo repos.SkillRepoInter) *NodeRegistry {
	registry := NewNodeRegistry()

	// 注册数据节点
	registry.Register(nodes.NewPrometheusExecutor())
	registry.Register(nodes.NewJaegerExecutor())
	registry.Register(nodes.NewElasticsearchExecutor())
	registry.Register(nodes.NewLokiExecutor())
	registry.Register(nodes.NewClickHouseExecutor())
	registry.Register(nodes.NewTopologyExecutor())
	registry.Register(nodes.NewSSHExecutor())
	registry.Register(nodes.NewKubernetesExecutorNoRepo())

	// 注册处理节点
	registry.Register(nodes.NewLLMExecutor())
	registry.Register(nodes.NewConditionExecutor())
	registry.Register(nodes.NewSmartRouterExecutor())
	registry.Register(nodes.NewGoScriptExecutor())
	registry.Register(nodes.NewMCPToolExecutor(mcpRepo))
	registry.Register(nodes.NewAgentExecutor(mcpRepo, skillRepo))

	// 注册输入输出节点
	registry.Register(nodes.NewStartExecutor())
	registry.Register(nodes.NewOutputExecutor())
	registry.Register(nodes.NewHTTPExecutor())

	return registry
}
