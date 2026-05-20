package api

import (
	"SREFlow/internal/services"
	"SREFlow/internal/types"

	"github.com/gin-gonic/gin"
)

type workspace struct{}

var WorkspaceController = new(workspace)

func (workspace) API(gin *gin.RouterGroup) {
	w := gin.Group("workspaces")
	{
		w.GET("", listWorkspaces)
		w.POST("", createWorkspace)
		w.GET("/:id", getWorkspace)
		w.PUT("/:id", updateWorkspace)
		w.DELETE("/:id", deleteWorkspace)
	}

}

// listWorkspaces GET /api/workspaces
func listWorkspaces(ctx *gin.Context) {
	Service(ctx, func() (interface{}, error) {
		return services.Workspace.List(ctx)
	})
}

// createWorkspace POST /api/workspaces
func createWorkspace(ctx *gin.Context) {
	r := new(types.RequestCreateWorkspace)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}
	if r.Name == "" {
		Error(ctx, nil, "workspace name is required")
		return
	}
	Service(ctx, func() (interface{}, error) {
		return services.Workspace.Create(ctx, r)
	})
}

// getWorkspace GET /api/workspaces/:id
func getWorkspace(ctx *gin.Context) {
	id := ctx.Param("id")
	Service(ctx, func() (interface{}, error) {
		return services.Workspace.GetByID(ctx, id)
	})
}

// updateWorkspace PUT /api/workspaces/:id
func updateWorkspace(ctx *gin.Context) {
	id := ctx.Param("id")
	r := new(types.RequestUpdateWorkspace)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}
	r.ID = id
	Service(ctx, func() (interface{}, error) {
		return services.Workspace.Update(ctx, r)
	})
}

// deleteWorkspace DELETE /api/workspaces/:id
func deleteWorkspace(ctx *gin.Context) {
	id := ctx.Param("id")
	Service(ctx, func() (interface{}, error) {
		return services.Workspace.Delete(ctx, id)
	})
}
