package api

import (
	"encoding/json"
	"fmt"
	"time"

	"SREFlow/internal/services"
	wf "SREFlow/internal/workflow"

	"github.com/gin-gonic/gin"
)

// handleSSEStream 统一的 SSE 流式推送处理逻辑，供 chat 和 workflow stream 接口共用。
// 负责：设置 SSE 响应头 → 注册事件通道 → 循环转发事件（含心跳保活）→ 断开时清理。
func handleSSEStream(c *gin.Context, executionId string) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	// 连接测试事件
	c.Writer.Write([]byte(": connected\n\n"))
	c.Writer.Flush()

	eventCh := services.Workflow.RegisterStreamChannel(executionId)
	defer services.Workflow.UnregisterStreamChannel(executionId)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	clientGone := c.Request.Context().Done()

	for {
		select {
		case <-clientGone:
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			writeSSEEvent(c, event)
		case <-ticker.C:
			writeSSEHeartbeat(c, executionId)
		}
	}
}

// writeSSEEvent 将工作流事件序列化为 SSE 格式写入响应
func writeSSEEvent(c *gin.Context, event *wf.StreamStyleEvent) {
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return
	}
	c.Writer.Write(fmt.Appendf(nil, "event: %s\ndata: %s\n\n", event.Event, string(eventJSON)))
	c.Writer.Flush()
}

// writeSSEHeartbeat 写入统一格式的心跳事件
func writeSSEHeartbeat(c *gin.Context, executionId string) {
	heartbeat := map[string]interface{}{
		"event":   "heartbeat",
		"task_id": executionId,
		"data": map[string]interface{}{
			"id":     executionId,
			"status": "alive",
			"time":   time.Now().Unix(),
		},
	}
	heartbeatJSON, _ := json.Marshal(heartbeat)
	c.Writer.Write(fmt.Appendf(nil, "event: heartbeat\ndata: %s\n\n", string(heartbeatJSON)))
	c.Writer.Flush()
}
