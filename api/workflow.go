package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	ctxutil "SREFlow/internal/ctx"
	"SREFlow/internal/services"
	"SREFlow/internal/types"

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
	singular.POST("/executions/:executionId/cancel", cancelExecution)
	singular.GET("/executions/:executionId", getExecutionResult)
	singular.GET("/executions/:executionId/status", getExecutionStatus)
	singular.GET("/executions/:executionId/stream", streamExecutionStatus)
	singular.GET("/:id/run-history", getRunHistory)
	singular.GET("/run-history", getAllRunHistory)
	singular.DELETE("/run-history/:executionId", deleteRunHistory)

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

// executeWorkflow handles POST /api/workflow/:id/execute (async)
func executeWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		Error(c, nil, "workflow id is required")
		return
	}

	// 读取 mode query 参数，默认为 text
	mode := c.Query("mode")
	if mode == "" {
		mode = "text"
	}

	// mode=json 时读取原始 body，用于将整个 body 传递给 Start 节点
	var rawBody []byte
	if mode == "json" {
		rawBody, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(rawBody))
	}

	// 解析请求体，读取 input 参数
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

	// 验证 input 参数：input 为空且数据库中 Start 节点 text 也为空时返回 400
	ap := ctxutil.DO()
	if err := r.Valid(ap.Ctx, ap.Database, ap.Cache); err != nil {
		Error(c, nil, err.Error())
		return
	}

	Service(c, func() (interface{}, error) {
		return services.Workflow.ExecuteAsync(r)
	})
}

// getExecutionResult handles GET /api/workflow/executions/:executionId
func getExecutionResult(ctx *gin.Context) {
	executionId := ctx.Param("executionId")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetExecutionResult(executionId)
	})
}

// getExecutionStatus handles GET /api/workflow/executions/:executionId/status
func getExecutionStatus(ctx *gin.Context) {
	executionId := ctx.Param("executionId")

	Service(ctx, func() (interface{}, error) {
		return services.Workflow.GetExecutionStatus(executionId)
	})
}

// streamExecutionStatus SSE 流式推送执行状态
// 前端通过 EventSource 连接此端点，实时接收节点状态变化事件
// GET /api/workflow/executions/:executionId/stream
func streamExecutionStatus(ctx *gin.Context) {
	executionId := ctx.Param("executionId")

	if executionId == "" {
		Error(ctx, nil, "execution id is required")
		return
	}

	// 预先发送 SSE 头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("X-Accel-Buffering", "no")

	// 发送初始连接确认
	ctx.Writer.Write([]byte(": connected\n\n"))
	ctx.Writer.Flush()

	// 立即发送测试消息验证连接
	testEvent := map[string]interface{}{
		"event":   "test_connection",
		"task_id": executionId,
		"workflow": map[string]interface{}{
			"id":   executionId,
			"name": "Test Workflow",
		},
		"node": map[string]interface{}{
			"id":   "test-node",
			"type": "test",
		},
		"data": map[string]interface{}{
			"id":              executionId,
			"workflow_id":     executionId,
			"workflow_run_id": executionId,
			"status":          "testing",
			"message":         "SSE connection test successful",
		},
	}
	testJSON, _ := json.Marshal(testEvent)
	ctx.Writer.Write(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", "test_connection", string(testJSON)))
	ctx.Writer.Flush()

	// 注册流式 SSE 事件通道
	eventCh := services.Workflow.RegisterEventSSEEventChannel(executionId)

	// 清理：确保 channel 从 map 中移除（幂等操作）
	defer services.Workflow.UnregisterEventSSEEventChannel(executionId)

	// 用心跳保活连接（每 10s）
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	clientGone := ctx.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			eventJSON, err := json.Marshal(event)
			if err != nil {
				continue
			}
			ctx.Writer.Write(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", event.Event, string(eventJSON)))
			ctx.Writer.Flush()
		case <-ticker.C:
			heartbeatEvent := map[string]interface{}{
				"event":   "heartbeat",
				"task_id": executionId,
				"data": map[string]interface{}{
					"id":     executionId,
					"status": "alive",
					"time":   time.Now().Unix(),
				},
			}
			heartbeatJSON, _ := json.Marshal(heartbeatEvent)
			ctx.Writer.Write(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", "heartbeat", string(heartbeatJSON)))
			ctx.Writer.Flush()
		}
	}
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

// deleteRunHistory handles DELETE /api/workflow/run-history/:executionId
func deleteRunHistory(ctx *gin.Context) {
	executionId := ctx.Param("executionId")

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

// cancelExecution handles POST /api/workflow/executions/:executionId/cancel
func cancelExecution(ctx *gin.Context) {
	executionId := ctx.Param("executionId")

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
