package api

import (
	"SREFlow/internal/services"
	"SREFlow/internal/types"

	"github.com/gin-gonic/gin"
)

type mcpServer struct{}

var MCPController = new(mcpServer)

func (mcpServer) API(gin *gin.RouterGroup) {
	g := gin.Group("mcp-servers")
	{
		g.GET("", listMCPServers)
		g.POST("", createMCPServer)
		g.GET("/:id", getMCPServer)
		g.PUT("/:id", updateMCPServer)
		g.DELETE("/:id", deleteMCPServer)
		g.GET("/:id/tools", listMCPTools)
		g.GET("/:id/status", getMCPServerStatus)
	}
}

func listMCPServers(ctx *gin.Context) {
	workspaceID := ctx.GetHeader("X-Workspace-Id")
	if workspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.List(ctx, workspaceID)
	})
}

func getMCPServer(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.Get(ctx, id)
	})
}

func createMCPServer(ctx *gin.Context) {
	workspaceID := ctx.GetHeader("X-Workspace-Id")
	if workspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	r := new(types.RequestCreateMCPServer)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	if err := r.Valid(); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.Create(ctx, r, workspaceID)
	})
}

func updateMCPServer(ctx *gin.Context) {
	r := new(types.RequestUpdateMCPServer)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	r.ID = ctx.Param("id")
	if r.ID == "" {
		Error(ctx, nil, "id is required")
		return
	}

	if err := r.Valid(); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.Update(ctx, r)
	})
}

func deleteMCPServer(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.Delete(ctx, id)
	})
}

func listMCPTools(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.ListTools(ctx, id)
	})
}

func getMCPServerStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		Error(ctx, nil, "id is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.MCP.GetStatus(ctx, id)
	})
}
