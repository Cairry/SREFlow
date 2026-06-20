package api

import (
	"time"

	ctxutil "SREFlow/internal/ctx"
	"SREFlow/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type chat struct{}

var ChatController = new(chat)

func (chat) API(gin *gin.RouterGroup) {
	chatGroup := gin.Group("chat")
	chatGroup.GET("/:id/stream", streamChatExecution)
	chatGroup.GET("/:id/sessions", listChatSessions)
	chatGroup.POST("/:id/sessions", createChatSession)
	chatGroup.GET("/:id/sessions/:sessionId", getChatSession)
	chatGroup.DELETE("/:id/sessions/:sessionId", deleteChatSession)
	chatGroup.GET("/:id/sessions/:sessionId/messages", getChatMessages)
}

// streamChatExecution handles GET /api/chat/:id/stream
func streamChatExecution(c *gin.Context) {
	executionId := c.Param("id")
	if executionId == "" {
		Error(c, nil, "execution id is required")
		return
	}
	handleSSEStream(c, executionId)
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
