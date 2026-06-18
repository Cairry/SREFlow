package repos

import (
	"gorm.io/gorm"
)

type (
	entry struct {
		mysql       *gorm.DB
		gorm        InterGormClient
		topo        InterTopologyRepo
		workflow    WorkflowRepoInter
		workspace   WorkspaceRepoInter
		integration IntegrationRepoInter
		mcpServer   MCPServerRepoInter
		skill       SkillRepoInter
		chatSession ChatSessionRepoInter
		chatMessage ChatMessageRepoInter
	}

	EntryInter interface {
		MySQL() *gorm.DB
		Gorm() InterGormClient
		Topology() InterTopologyRepo
		Workflow() WorkflowRepoInter
		Workspace() WorkspaceRepoInter
		Integration() IntegrationRepoInter
		MCPServer() MCPServerRepoInter
		Skill() SkillRepoInter
		ChatSession() ChatSessionRepoInter
		ChatMessage() ChatMessageRepoInter
	}
)

func NewEntryRepo(db *gorm.DB) EntryInter {
	return &entry{
		mysql:       db,
		gorm:        NewGormClient(db),
		workflow:    NewWorkflowRepo(db),
		workspace:   NewGormWorkspaceRepo(db),
		integration: NewGormIntegrationRepo(db),
		mcpServer:   NewGormMCPServerRepo(db),
		skill:       NewGormSkillRepo(db),
		chatSession: NewChatSessionRepo(db),
		chatMessage: NewChatMessageRepo(db),
	}
}

// MySQL returns the database connection
func (e *entry) MySQL() *gorm.DB {
	return e.mysql
}

// Gorm returns the GORM client
func (e *entry) Gorm() InterGormClient {
	return e.gorm
}

// Topology returns the topology repository
func (e *entry) Topology() InterTopologyRepo {
	return e.topo
}

// Workflow returns the workflow repository
func (e *entry) Workflow() WorkflowRepoInter {
	return e.workflow
}

// Workspace returns the workspace repository
func (e *entry) Workspace() WorkspaceRepoInter {
	return e.workspace
}

// Integration returns the integration repository
func (e *entry) Integration() IntegrationRepoInter {
	return e.integration
}

// MCPServer returns the MCP server repository
func (e *entry) MCPServer() MCPServerRepoInter {
	return e.mcpServer
}

// Skill returns the Skill repository
func (e *entry) Skill() SkillRepoInter {
	return e.skill
}

// ChatSession returns the chat session repository
func (e *entry) ChatSession() ChatSessionRepoInter {
	return e.chatSession
}

// ChatMessage returns the chat message repository
func (e *entry) ChatMessage() ChatMessageRepoInter {
	return e.chatMessage
}
