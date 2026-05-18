package routes

import (
	"RCAgent/api"
	"RCAgent/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Register(engine *gin.Engine) {
	group := engine.Group("api/")
	group.Use(middleware.Cors())

	// Topology routes
	api.TopologyController.API(group)

	// Workflow routes
	api.WorkflowController.API(group)

	// Workspace routes
	api.WorkspaceController.API(group)

	// Datasource routes
	api.DatasourceController.API(group)

	// MCP Server routes
	api.MCPController.API(group)
}
