package models

import (
	"time"

	"gorm.io/gorm"
)

// ChatSession represents a chat conversation session
type ChatSession struct {
	ID          string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	WorkflowID  string         `json:"workflowId" gorm:"type:varchar(36);index"`
	Title       string         `json:"title" gorm:"type:varchar(255)"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"deletedAt,omitempty" gorm:"index"`
}

// TableName specifies the table name for ChatSession model
func (ChatSession) TableName() string {
	return "chat_sessions"
}

// ChatMessage represents a single message in a chat session
type ChatMessage struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	SessionID string    `json:"sessionId" gorm:"type:varchar(36);index"`
	Role      string    `json:"role" gorm:"type:varchar(20)"` // "user" or "assistant"
	Content   string    `json:"content" gorm:"type:text"`
	Thinking  string    `json:"thinking" gorm:"type:text"` // JSON array of thinking steps
	CreatedAt time.Time `json:"createdAt"`
}

// TableName specifies the table name for ChatMessage model
func (ChatMessage) TableName() string {
	return "chat_messages"
}
