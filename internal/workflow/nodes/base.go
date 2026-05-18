package nodes

// DefaultNodeExecutor 默认节点执行器基类
type DefaultNodeExecutor struct {
	NodeType    string
	Category    string
	Description string
}

// GetType 返回节点类型
func (e *DefaultNodeExecutor) GetType() string {
	return e.NodeType
}

// GetCategory 返回节点类别
func (e *DefaultNodeExecutor) GetCategory() string {
	return e.Category
}

// GetDescription 返回节点描述
func (e *DefaultNodeExecutor) GetDescription() string {
	return e.Description
}
