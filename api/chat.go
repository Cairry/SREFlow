package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"SREFlow/internal/models"
	ctxutil "SREFlow/internal/ctx"
	"SREFlow/internal/services"
	"SREFlow/internal/types"
	"SREFlow/pkg/trace"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type chat struct{}

var ChatController = new(chat)

func (chat) API(gin *gin.RouterGroup) {
	singular := gin.Group("workflow")
	singular.POST("/:id/chat", chatWorkflow)
	singular.GET("/:id/sessions", listChatSessions)
	singular.POST("/:id/sessions", createChatSession)
	singular.GET("/:id/sessions/:sessionId", getChatSession)
	singular.DELETE("/:id/sessions/:sessionId", deleteChatSession)
	singular.GET("/:id/sessions/:sessionId/messages", getChatMessages)
}

// chatWorkflow handles POST /api/workflow/:id/chat (sync with streaming)
func chatWorkflow(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		Error(c, nil, "workflow id is required")
		return
	}

	// Get sessionId from query for saving messages
	sessionId := c.Query("sessionId")

	// Read raw body for mode=json
	mode := c.Query("mode")
	if mode == "" {
		mode = "text"
	}

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

	// Save user message if sessionId provided
	ap := ctxutil.DO()
	if sessionId != "" && r.Input != "" {
		userMsg := &models.ChatMessage{
			ID:        uuid.NewString(),
			SessionID: sessionId,
			Role:      "user",
			Content:   r.Input,
			CreatedAt: time.Now(),
		}
		ap.Database.ChatMessage().Create(userMsg)
	}

	r.WorkflowId = id
	r.Mode = mode
	if mode == "json" {
		r.RawBody = string(rawBody)
	}
	r.TraceID = trace.FromContext(c.Request.Context())

	// Validate
	if err := r.Valid(ap.Ctx, ap.Database, ap.Cache); err != nil {
		Error(c, nil, err.Error())
		return
	}

	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// Send connected event
	c.Writer.Write([]byte(": connected\n\n"))
	c.Writer.Flush()

	// Execute synchronously with streaming
	eventCh, executionID, err := services.Workflow.ExecuteSyncWithStream(r)
	if err != nil {
		c.Writer.Write(fmt.Appendf(nil, "event: error\ndata: %s\n\n", err.Error()))
		c.Writer.Flush()
		return
	}

	// Send execution ID to frontend for cancel support
	startPayload := map[string]string{"executionId": executionID}
	startJSON, _ := json.Marshal(startPayload)
	c.Writer.Write(fmt.Appendf(nil, "event: execution_started\ndata: %s\n\n", string(startJSON)))
	c.Writer.Flush()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	clientGone := c.Request.Context().Done()
	var assistantContent string
	var thinkingSteps []map[string]interface{}

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-eventCh:
			if !ok {
				// Stream ended, save assistant message
				if sessionId != "" && assistantContent != "" {
					thinkingJSON := ""
					if len(thinkingSteps) > 0 {
						if b, err := json.Marshal(thinkingSteps); err == nil {
							thinkingJSON = string(b)
						}
					}
					
					assistantMsg := &models.ChatMessage{
						ID:        uuid.NewString(),
						SessionID: sessionId,
						Role:      "assistant",
						Content:   assistantContent,
						Thinking:  thinkingJSON,
						CreatedAt: time.Now(),
					}
					ap.Database.ChatMessage().Create(assistantMsg)

					// Update session timestamp
					if session, err := ap.Database.ChatSession().GetByID(sessionId); err == nil {
						session.UpdatedAt = time.Now()
						ap.Database.ChatSession().Update(session)
					}
				}
				return
			}
			eventJSON, err := json.Marshal(event)
			if err != nil {
				continue
			}

			// Accumulate assistant content and thinking steps
			if sessionId != "" && event.Node != nil {
				if event.Event == "reply" {
					if reply, ok := event.Node.Outputs["reply"]; ok {
						assistantContent += reply
					}
				} else if event.Event == "thinking" {
					// Add thinking step
					step := map[string]interface{}{
						"nodeId":      event.Node.ID,
						"nodeType":    event.Node.Type,
						"outputs":     event.Node.Outputs,
						"elapsedTime": event.Node.ExeTime,
					}
					thinkingSteps = append(thinkingSteps, step)
				}
			}

			c.Writer.Write(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", event.Event, string(eventJSON)))
			c.Writer.Flush()
		case <-ticker.C:
			heartbeatEvent := map[string]interface{}{
				"event": "heartbeat",
				"time":  time.Now().Unix(),
			}
			heartbeatJSON, _ := json.Marshal(heartbeatEvent)
			c.Writer.Write(fmt.Appendf(nil, "event: heartbeat\ndata: %s\n\n", string(heartbeatJSON)))
			c.Writer.Flush()
		}
	}
}

// listChatSessions handles GET /api/workflow/:id/sessions
func listChatSessions(c *gin.Context) {
	workflowId := c.Param("id")
	if workflowId == "" {
		Error(c, nil, "workflow id is required")
		return
	}

	ap := ctxutil.DO()
	sessions, err := ap.Database.ChatSession().GetByWorkflowID(workflowId, 50)
	if err != nil {
		Error(c, nil, "failed to get sessions")
		return
	}

	Success(c, sessions, "")
}

type createSessionRequest struct {
	Title string `json:"title"`
}

// createChatSession handles POST /api/workflow/:id/sessions
func createChatSession(c *gin.Context) {
	workflowId := c.Param("id")
	if workflowId == "" {
		Error(c, nil, "workflow id is required")
		return
	}

	var req createSessionRequest
	if err := BindJson(c, &req); err != nil {
		Error(c, nil, err.Error())
		return
	}

	session := &models.ChatSession{
		ID:         uuid.NewString(),
		WorkflowID: workflowId,
		Title:      req.Title,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	ap := ctxutil.DO()
	if err := ap.Database.ChatSession().Create(session); err != nil {
		Error(c, nil, "failed to create session")
		return
	}

	Success(c, session, "")
}

// getChatSession handles GET /api/workflow/:id/sessions/:sessionId
func getChatSession(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		Error(c, nil, "session id is required")
		return
	}

	ap := ctxutil.DO()
	session, err := ap.Database.ChatSession().GetByID(sessionId)
	if err != nil {
		Error(c, nil, "session not found")
		return
	}

	Success(c, session, "")
}

// deleteChatSession handles DELETE /api/workflow/:id/sessions/:sessionId
func deleteChatSession(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		Error(c, nil, "session id is required")
		return
	}

	ap := ctxutil.DO()
	// Delete messages first
	if err := ap.Database.ChatMessage().DeleteBySessionID(sessionId); err != nil {
		Error(c, nil, "failed to delete session messages")
		return
	}

	// Delete session
	if err := ap.Database.ChatSession().Delete(sessionId); err != nil {
		Error(c, nil, "failed to delete session")
		return
	}

	Success(c, nil, "")
}

// getChatMessages handles GET /api/workflow/:id/sessions/:sessionId/messages
func getChatMessages(c *gin.Context) {
	sessionId := c.Param("sessionId")
	if sessionId == "" {
		Error(c, nil, "session id is required")
		return
	}

	ap := ctxutil.DO()
	messages, err := ap.Database.ChatMessage().GetBySessionID(sessionId)
	if err != nil {
		Error(c, nil, "failed to get messages")
		return
	}

	Success(c, messages, "")
}
