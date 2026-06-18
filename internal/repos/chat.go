package repos

import (
	"SREFlow/internal/models"

	"gorm.io/gorm"
)

type (
	ChatSessionRepoInter interface {
		Create(session *models.ChatSession) error
		GetByID(id string) (*models.ChatSession, error)
		GetByWorkflowID(workflowID string, limit int) ([]*models.ChatSession, error)
		Update(session *models.ChatSession) error
		Delete(id string) error
	}

	ChatMessageRepoInter interface {
		Create(message *models.ChatMessage) error
		GetBySessionID(sessionID string) ([]*models.ChatMessage, error)
		DeleteBySessionID(sessionID string) error
	}
)

type chatSessionRepo struct {
	db *gorm.DB
}

type chatMessageRepo struct {
	db *gorm.DB
}

func NewChatSessionRepo(db *gorm.DB) ChatSessionRepoInter {
	return &chatSessionRepo{db: db}
}

func NewChatMessageRepo(db *gorm.DB) ChatMessageRepoInter {
	return &chatMessageRepo{db: db}
}

func (r *chatSessionRepo) Create(session *models.ChatSession) error {
	return r.db.Create(session).Error
}

func (r *chatSessionRepo) GetByID(id string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.db.Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *chatSessionRepo) GetByWorkflowID(workflowID string, limit int) ([]*models.ChatSession, error) {
	var sessions []*models.ChatSession
	query := r.db.Where("workflow_id = ?", workflowID).Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&sessions).Error
	return sessions, err
}

func (r *chatSessionRepo) Update(session *models.ChatSession) error {
	return r.db.Save(session).Error
}

func (r *chatSessionRepo) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.ChatSession{}).Error
}

func (r *chatMessageRepo) Create(message *models.ChatMessage) error {
	return r.db.Create(message).Error
}

func (r *chatMessageRepo) GetBySessionID(sessionID string) ([]*models.ChatMessage, error) {
	var messages []*models.ChatMessage
	err := r.db.Where("session_id = ?", sessionID).Order("created_at ASC").Find(&messages).Error
	return messages, err
}

func (r *chatMessageRepo) DeleteBySessionID(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&models.ChatMessage{}).Error
}
