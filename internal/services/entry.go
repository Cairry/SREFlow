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
	Datasource *DatasourceService
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
	Workflow = NewNewWorkflowService(workflowConfig, ctx.Database.Workflow(), repos.NewWorkflowExecutionRepo(ctx.Database.MySQL()), ctx.Database.Datasource(), repos.NewWorkflowVersionRepo(ctx.Database.MySQL()), ctx.Database.MCPServer())
	Workspace = NewWorkspaceService(ctx.Database.Workspace(), ctx.Database.Workflow())
	Datasource = NewDatasourceService(ctx.Database.Datasource(), ctx.Database.Workflow())
	MCP = &MCPService{}
	MCP.Init(ctx.Database.MCPServer())

	logc.Info(context.Background(), "服务逻辑层初始化完成!")
}
