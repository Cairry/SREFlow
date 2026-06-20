package api

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	ctxutil "SREFlow/internal/ctx"
	"SREFlow/internal/services"
	"SREFlow/internal/types"
	"SREFlow/pkg/trace"

	"github.com/gin-gonic/gin"
)

type workflow struct{}

var WorkflowController = new(workflow)

func (workflow) API(gin *gin.RouterGroup) {
	// Support both singular and plural forms for compatibility
	singular := gin.Group("workflow")
	plural := gin.Group("workflows")

	// Workflow CRUD operations
	singular.GET("/:id", getWorkflow)
	singular.PUT("/:id", updateWorkflow)
	singular.PATCH("/:id", updateWorkflowDescription)
	singular.DELETE("/:id", deleteWorkflow)

	plural.GET("", listWorkflows)
	plural.POST("", createWorkflow)

	// Workflow execution operations
	singular.POST("/:id/execute", executeWorkflow)
	singular.POST("/:id/cancel", cancelExecution)
	singular.GET("/:id/result", getExecutionResult)
	singular.GET("/:id/status", getExecutionStatus)
	singular.GET("/:id/stream", streamExecutionStatus)
	singular.GET("/:id/run-history", getRunHistory)
	singular.GET("/run-history", getAllRunHistory)
	singular.DELETE("/run-history/:id", deleteRunHistory)

	// Workflow version history
	singular.GET("/:id/versions", getVersionHistory)
	singular.GET("/:id/versions/:version", getVersion)
	singular.DELETE("/:id/versions/:version", deleteVersion)
	singular.POST("/:id/goto/:version", gotoVersion)
	singular.POST("/:id/release/:version", setRelease)
	singular.DELETE("/:id/release/:version", cancelRelease)
}

func listWorkflows(ctx *gin.Context) {
	workspaceId := ctx.GetHeader("X-Workspace-Id")
	if workspaceId == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.ListWorkflows(workspaceId)
	})
}

// createWorkflow handles POST /api/workflows
func createWorkflow(ctx *gin.Context) {
	workspaceId := ctx.GetHeader("X-Workspace-Id")
	if workspaceId == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	r := new(types.RequestCreateWorkflow)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	// 验证必填字段
	if r.Name == "" {
		Error(ctx, nil, "workflow name is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.Create(r)
	})
}

// getWorkflow handles GET /api/workflow/:id
func getWorkflow(ctx *gin.Context) {
	workspaceId := ctx.GetHeader("X-Workspace-Id")
	if workspaceId == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	id := ctx.Param("id")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetByID(id)
	})
}

// updateWorkflow handles PUT /api/workflows/:id
func updateWorkflow(ctx *gin.Context) {

	id := ctx.Param("id")
	r := new(types.RequestUpdateWorkflow)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	r.WorkspaceID = ctx.GetHeader("X-Workspace-Id")
	if r.WorkspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	r.ID = id
	if r.Name == "" {
		Error(ctx, nil, "workflow name is required")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.Update(r)
	})
}

// updateWorkflowDescription handles PATCH /api/workflows/:id
func updateWorkflowDescription(ctx *gin.Context) {
	id := ctx.Param("id")
	r := new(types.RequestUpdateWorkflowDescription)
	if err := BindJson(ctx, r); err != nil {
		Error(ctx, nil, err.Error())
		return
	}

	r.WorkspaceID = ctx.GetHeader("X-Workspace-Id")
	if r.WorkspaceID == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	r.ID = id

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.UpdateDescription(r)
	})
}

// deleteWorkflow handles DELETE /api/workflow/:id
func deleteWorkflow(ctx *gin.Context) {
	workspaceId := ctx.GetHeader("X-Workspace-Id")
	if workspaceId == "" {
		Error(ctx, nil, "X-Workspace-Id header is required")
		return
	}

	id := ctx.Param("id")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.Delete(id)
	})
}

// executeWorkflow handles POST /api/workflow/:id/execute
// Returns executionId, then client subscribes to SSE via /api/workflow/:id/stream or /api/chat/:id/stream
func executeWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		Error(c, nil, "workflow id is required")
		return
	}

	// Read query params
	mode := c.Query("mode")
	if mode == "" {
		mode = "text"
	}

	// mode=json: read raw body for passing entire body to Start node
	var rawBody []byte
	if mode == "json" {
		rawBody, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
	}

	// Parse request body
	r := new(types.RequestWorkflowExecute)
	if err := BindJson(c, r); err != nil {
		Error(c, nil, err.Error())
		return
	}

	r.WorkflowId = id
	r.Mode = mode
	if mode == "json" {
		r.RawBody = string(rawBody)
	}
	r.TraceID = trace.FromContext(c.Request.Context())

	// Validate
	ap := ctxutil.DO()
	if err := r.Valid(ap.Ctx, ap.Database, ap.Cache); err != nil {
		Error(c, nil, err.Error())
		return
	}

	// Execute synchronously with streaming support
	_, executionID, err := services.Workflow.ExecuteSyncWithStream(r)
	if err != nil {
		Error(c, nil, err.Error())
		return
	}

	// Return executionId for SSE subscription
	Success(c, map[string]string{
		"executionId": executionID,
	}, "")
}

// getExecutionResult handles GET /api/workflow/:executionId
func getExecutionResult(ctx *gin.Context) {
	executionId := ctx.Param("id")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetExecutionResult(executionId)
	})
}

// getExecutionStatus handles GET /api/workflow/:executionId/status
func getExecutionStatus(ctx *gin.Context) {
	executionId := ctx.Param("id")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetExecutionStatus(executionId)
	})
}

// streamExecutionStatus SSE 流式推送执行状态
func streamExecutionStatus(ctx *gin.Context) {
	executionId := ctx.Param("id")

	if executionId == "" {
		Error(ctx, nil, "execution id is required")
		return
	}

	handleSSEStream(ctx, executionId)
}

// getRunHistory handles GET /api/workflows/:id/run-history
func getRunHistory(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	limit := 50 // default limit
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetRunHistory(workflowId, limit)
	})
}

// getAllRunHistory handles GET /api/workflow/run-history
func getAllRunHistory(ctx *gin.Context) {
	limit := 100 // default limit
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetAllRunHistory(limit)
	})
}

// deleteRunHistory handles DELETE /api/workflow/run-history/:id
func deleteRunHistory(ctx *gin.Context) {
	executionId := ctx.Param("id")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.DeleteRunHistory(executionId)
	})
}

// getVersionHistory handles GET /api/workflow/:id/versions
func getVersionHistory(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	limit := 50
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetVersionHistory(workflowId, limit)
	})
}

// getVersion handles GET /api/workflow/versions/:versionId
func getVersion(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	version := 0
	if _, err := fmt.Sscanf(ctx.Param("version"), "%d", &version); err != nil || version <= 0 {
		Error(ctx, nil, "invalid version number")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetVersion(workflowId, version)
	})
}

// deleteVersion handles DELETE /api/workflow/:id/versions/:version
func deleteVersion(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	version := 0
	if _, err := fmt.Sscanf(ctx.Param("version"), "%d", &version); err != nil || version <= 0 {
		Error(ctx, nil, "invalid version number")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.DeleteVersion(workflowId, version)
	})
}

// setRelease handles POST /api/workflow/:id/release/:version
func setRelease(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	version := 0
	if _, err := fmt.Sscanf(ctx.Param("version"), "%d", &version); err != nil || version <= 0 {
		Error(ctx, nil, "invalid version number")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.SetRelease(workflowId, version)
	})
}

// cancelRelease handles DELETE /api/workflow/:id/release/:version
func cancelRelease(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	version := 0
	if _, err := fmt.Sscanf(ctx.Param("version"), "%d", &version); err != nil || version <= 0 {
		Error(ctx, nil, "invalid version number")
		return
	}

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.CancelRelease(workflowId, version)
	})
}

// gotoVersion handles POST /api/workflow/:id/goto/:version
func gotoVersion(ctx *gin.Context) {
	workflowId := ctx.Param("id")
	versionNumStr := ctx.Param("version")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GotoVersion(workflowId, versionNumStr)
	})
}

// cancelExecution handles POST /api/workflow/:id/cancel
func cancelExecution(ctx *gin.Context) {
	executionId := ctx.Param("id")

	Service(ctx, func() (interface{}, error) {
		err := services.Workflow.CancelWorkflowExecution(ctx, executionId)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"id":      executionId,
			"status":  "cancelled",
			"message": "Workflow execution cancelled",
		}, nil
	})
}
