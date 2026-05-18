package workflow

import (
	"context"
	"fmt"
	"sync"

	"RCAgent/internal/models"
)

// NodeRegistry 节点注册器
type NodeRegistry struct {
	mu        sync.RWMutex
	executors map[string]NodeExecutor
	definitions map[string]models.NodeDefinition
}

// NodeExecutor 节点执行器接口
type NodeExecutor interface {
	Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error)
	GetType() string
	GetCategory() string
	GetDescription() string
}

// NewNodeRegistry 创建新的节点注册器
func NewNodeRegistry() *NodeRegistry {
	return &NodeRegistry{
		executors: make(map[string]NodeExecutor),
		definitions: make(map[string]models.NodeDefinition),
	}
}

// Register 注册节点执行器
func (r *NodeRegistry) Register(executor NodeExecutor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	nodeType := executor.GetType()
	r.executors[nodeType] = executor
	
	// 同时创建节点定义
	r.definitions[nodeType] = models.NodeDefinition{
		Type:        executor.GetType(),
		Category:    executor.GetCategory(),
		Description: executor.GetDescription(),
		Executor: func(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
			return executor.Execute(ctx, nc)
		},
	}
}

// RegisterDefinition 注册节点定义
func (r *NodeRegistry) RegisterDefinition(def models.NodeDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.definitions[def.Type] = def
}

// GetExecutor 获取节点执行器
func (r *NodeRegistry) GetExecutor(nodeType string) (NodeExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	executor, exists := r.executors[nodeType]
	return executor, exists
}

// GetDefinition 获取节点定义
func (r *NodeRegistry) GetDefinition(nodeType string) (models.NodeDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	def, exists := r.definitions[nodeType]
	return def, exists
}

// GetAllTypes 获取所有注册的节点类型
func (r *NodeRegistry) GetAllTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	types := make([]string, 0, len(r.executors))
	for nodeType := range r.executors {
		types = append(types, nodeType)
	}
	return types
}

// GetAllDefinitions 获取所有节点定义
func (r *NodeRegistry) GetAllDefinitions() map[string]models.NodeDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	definitions := make(map[string]models.NodeDefinition)
	for nodeType, def := range r.definitions {
		definitions[nodeType] = def
	}
	return definitions
}

// GetByCategory 按类别获取节点类型
func (r *NodeRegistry) GetByCategory(category string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var types []string
	for nodeType, executor := range r.executors {
		if executor.GetCategory() == category {
			types = append(types, nodeType)
		}
	}
	return types
}

// Unregister 注销节点
func (r *NodeRegistry) Unregister(nodeType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	delete(r.executors, nodeType)
	delete(r.definitions, nodeType)
}

// Clear 清空所有注册的节点
func (r *NodeRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.executors = make(map[string]NodeExecutor)
	r.definitions = make(map[string]models.NodeDefinition)
}

// Validate 验证节点配置
func (r *NodeRegistry) Validate(nodeType string, config map[string]any) error {
	def, exists := r.GetDefinition(nodeType)
	if !exists {
		return fmt.Errorf("unknown node type: %s", nodeType)
	}
	
	// 验证必需字段
	for _, field := range def.Config {
		if field.Required {
			if _, exists := config[field.Key]; !exists {
				return fmt.Errorf("required field '%s' is missing", field.Key)
			}
		}
	}
	
	return nil
}

// GetSchema 获取节点配置模式
func (r *NodeRegistry) GetSchema(nodeType string) ([]models.FieldSchema, error) {
	def, exists := r.GetDefinition(nodeType)
	if !exists {
		return nil, fmt.Errorf("unknown node type: %s", nodeType)
	}
	
	return def.Config, nil
}
