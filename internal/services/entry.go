package services

import (
	"RCAgent/internal/ctx"
	"RCAgent/internal/repos"
	"RCAgent/internal/workflow"
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
)

var (
	Topology   TopologyService
	Workflow   *NewWorkflowService
	Workspace  *WorkspaceService
	Integration *IntegrationService
	MCP        *MCPService
)

func Init(ctx *ctx.Context) {
	Topology = newTopologyService(ctx)

	// 使用默认配置创建工作流服务
	workflowConfig := &workflow.EngineConfig{
		MaxConcurrentNodes: 10,
		DefaultTimeout:     300 * time.Second,
		EnableRetry:        true,
		MaxRetries:         3,
		RetryDelay:         5 * time.Second,
	}
	Workflow = NewNewWorkflowService(workflowConfig, ctx.Database.Workflow(), repos.NewWorkflowExecutionRepo(ctx.Database.MySQL()), ctx.Database.Integration(), repos.NewWorkflowVersionRepo(ctx.Database.MySQL()), ctx.Database.MCPServer())
	Workspace = NewWorkspaceService(ctx.Database.Workspace(), ctx.Database.Workflow())
	Integration = NewIntegrationService(ctx.Database.Integration(), ctx.Database.Workflow())
	Workspace.SetIntegrationService(Integration)

	// 确保所有现有工作空间都拥有默认集成
	workspaces, err := ctx.Database.Workspace().GetAll()
	if err == nil {
		for _, ws := range workspaces {
			Integration.EnsureDefaultIntegrations(ws.ID)
		}
	}
	MCP = &MCPService{}
	MCP.Init(ctx.Database.MCPServer())

	logc.Info(context.Background(), "服务逻辑层初始化完成!")
}
