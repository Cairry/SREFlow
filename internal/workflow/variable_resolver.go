package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"SREFlow/internal/models"

	"github.com/zeromicro/go-zero/core/logc"
)

// VariableResolver 变量解析器
type VariableResolver struct {
	// 请求上下文，用于日志携带 traceId
	ctx context.Context
	// 存储所有节点的执行结果
	nodeResults map[string]*models.NodeResult
	// 存储全局变量
	variables map[string]any
	// 模板函数映射
	funcMap template.FuncMap
}

// NewVariableResolver 创建新的变量解析器
func NewVariableResolver(ctx context.Context, nodeResults map[string]*models.NodeResult, variables map[string]any) *VariableResolver {
	return &VariableResolver{
		ctx:         ctx,
		nodeResults: nodeResults,
		variables:   variables,
		funcMap:     createTemplateFuncMap(nodeResults, variables),
	}
}

// createTemplateFuncMap 创建模板函数映射
func createTemplateFuncMap(nodeResults map[string]*models.NodeResult, variables map[string]any) template.FuncMap {
	return template.FuncMap{
		// 节点输出访问函数
		"node": func(nodeId string) map[string]any {
			if result, exists := nodeResults[nodeId]; exists && result.Outputs != nil {
				return result.Outputs
			}
			return make(map[string]any)
		},
		// 全局变量访问函数
		"var": func(varName string) any {
			if variables != nil {
				if value, exists := variables[varName]; exists {
					return value
				}
			}
			return nil
		},
		// JSON访问函数
		"json": func(data any) map[string]any {
			result, _ := EnsureJSONFormat(data)
			return result
		},
		// 字符串工具函数
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		// 类型转换函数
		"toString": func(v any) string {
			return fmt.Sprintf("%v", v)
		},
		"toInt": func(v any) int {
			if i, ok := v.(int); ok {
				return i
			}
			if f, ok := v.(float64); ok {
				return int(f)
			}
			return 0
		},
		// 条件函数
		"default": func(defaultValue any, value any) any {
			if value == nil || (reflect.ValueOf(value).Kind() == reflect.String && value.(string) == "") {
				return defaultValue
			}
			return value
		},
	}
}

// ResolveVariables 解析字符串中的变量引用
func (vr *VariableResolver) ResolveVariables(input string) (string, error) {
	// 检查是否包含模板语法
	if !strings.Contains(input, "{{") {
		return input, nil
	}

	// 先尝试使用向后兼容的正则表达式解析
	if vr.hasLegacySyntax(input) {
		return vr.resolveLegacyVariables(input)
	}

	// 使用Go Template语法解析
	tmpl, err := template.New("resolver").Funcs(vr.funcMap).Parse(input)
	if err != nil {
		return input, fmt.Errorf("template parse error: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		return input, fmt.Errorf("template execute error: %w", err)
	}

	return buf.String(), nil
}

// ResolveVariablesInMap 解析map中的所有变量引用
// 单个key解析失败时保留原始值，不影响其他key的解析
func (vr *VariableResolver) ResolveVariablesInMap(input map[string]any) (map[string]any, error) {
	result := make(map[string]any)

	for key, value := range input {
		resolvedValue, err := vr.resolveValue(value)
		if err != nil {
			// 解析失败：保留原始值，记录日志，继续处理其他字段
			logc.Infof(vr.ctx, "[VariableResolver] Keeping original value for key '%s': %v", key, err)
			result[key] = value
			continue
		}
		result[key] = resolvedValue
	}

	return result, nil
}

// resolveValue 递归解析值中的变量
func (vr *VariableResolver) resolveValue(value any) (any, error) {
	switch v := value.(type) {
	case string:
		return vr.ResolveVariables(v)
	case map[string]any:
		return vr.ResolveVariablesInMap(v)
	case []any:
		// 处理数组：单个元素解析失败时保留原始元素，不影响其他元素
		result := make([]any, len(v))
		for i, item := range v {
			resolvedItem, err := vr.resolveValue(item)
			if err != nil {
				result[i] = item
				continue
			}
			result[i] = resolvedItem
		}
		return result, nil
	default:
		// 其他类型直接返回
		return value, nil
	}
}

// hasLegacySyntax 检查是否包含旧的语法格式
func (vr *VariableResolver) hasLegacySyntax(input string) bool {
	// 检查是否包含 {{nodeId.outputKey}} 或 {{variables.varName}} 格式
	return strings.Contains(input, ".") && strings.Contains(input, "{{")
}

// resolveLegacyVariables 使用正则表达式解析旧的变量语法
func (vr *VariableResolver) resolveLegacyVariables(input string) (string, error) {
	// 匹配 {{xxx.xxx}} 格式的变量
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return re.ReplaceAllStringFunc(input, func(match string) string {
		// 提取变量路径，去掉 {{ }}
		varPath := strings.TrimSpace(match[2 : len(match)-2])

		// 解析变量
		value, err := vr.resolveVariablePath(varPath)
		if err != nil {
			// 如果解析失败，返回原始字符串
			logc.Infof(vr.ctx, "[VariableResolver] Failed to resolve variable '%s': %v", varPath, err)
			return match
		}

		return stringifyForTemplate(value)
	}), nil
}

// stringifyForTemplate 将解析后的值转为适合嵌入模板字符串的格式
// - string/int/float/bool：直接 fmt.Sprintf
// - map/array：JSON 序列化，方便 GoScript 等节点 json.Unmarshal 使用
func stringifyForTemplate(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any, []any:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// resolveVariablePath 解析变量路径
func (vr *VariableResolver) resolveVariablePath(path string) (any, error) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 || parts[0] == "" {
		return nil, fmt.Errorf("invalid variable path: %s", path)
	}

	switch parts[0] {
	case "variables":
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid variable path: %s", path)
		}
		// 引用全局变量
		return vr.resolveVariable(parts[1:])
	default:
		// 引用节点输出
		// 如果没有指定字段（只有 nodeId），返回节点所有 output
		if len(parts) == 1 {
			return vr.resolveNodeAllOutputs(parts[0])
		}
		return vr.resolveNodeOutput(parts[0], parts[1:])
	}
}

// resolveVariable 解析全局变量
func (vr *VariableResolver) resolveVariable(pathParts []string) (any, error) {
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("variable path is empty")
	}

	varName := pathParts[0]
	if vr.variables == nil {
		return nil, fmt.Errorf("variables map is nil")
	}

	value, exists := vr.variables[varName]
	if !exists {
		return nil, fmt.Errorf("variable '%s' not found", varName)
	}

	// 如果有更多路径部分，继续深入解析
	if len(pathParts) > 1 {
		return vr.navigateObject(value, pathParts[1:])
	}

	return value, nil
}

// resolveNodeAllOutputs 获取节点所有输出（当只指定 nodeId 没有字段时）
func (vr *VariableResolver) resolveNodeAllOutputs(nodeId string) (any, error) {
	result, exists := vr.nodeResults[nodeId]
	if !exists {
		return nil, fmt.Errorf("node result not found: %s", nodeId)
	}

	if result.Outputs == nil {
		return nil, fmt.Errorf("node %s has no outputs", nodeId)
	}

	return result.Outputs, nil
}

// resolveNodeOutput 解析节点输出
func (vr *VariableResolver) resolveNodeOutput(nodeId string, pathParts []string) (any, error) {
	if len(pathParts) == 0 {
		return nil, fmt.Errorf("output path is empty for node %s", nodeId)
	}

	result, exists := vr.nodeResults[nodeId]
	if !exists {
		return nil, fmt.Errorf("node result not found: %s", nodeId)
	}

	if result.Outputs == nil {
		return nil, fmt.Errorf("node %s has no outputs", nodeId)
	}

	// 确保输出是JSON格式
	outputKey := pathParts[0]
	value, exists := result.Outputs[outputKey]
	if !exists {
		return nil, fmt.Errorf("output key '%s' not found in node %s", outputKey, nodeId)
	}

	// 如果有更多路径部分，继续深入解析
	if len(pathParts) > 1 {
		return vr.navigateObject(value, pathParts[1:])
	}

	return value, nil
}

// navigateObject 在对象中导航
func (vr *VariableResolver) navigateObject(obj any, pathParts []string) (any, error) {
	current := obj

	for i, part := range pathParts {
		switch v := current.(type) {
		case map[string]any:
			next, exists := v[part]
			if !exists {
				return nil, fmt.Errorf("path part '%s' not found at index %d", part, i)
			}
			current = next
		case []any:
			// 处理数组索引
			if part == "length" || part == "size" {
				current = len(v)
			} else {
				return nil, fmt.Errorf("array navigation not fully supported for part '%s'", part)
			}
		default:
			// 尝试将对象转换为map进行导航
			jsonBytes, err := json.Marshal(current)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal object for navigation at part '%s': %w", part, err)
			}

			var objMap map[string]any
			if err := json.Unmarshal(jsonBytes, &objMap); err != nil {
				return nil, fmt.Errorf("failed to unmarshal object for navigation at part '%s': %w", part, err)
			}

			next, exists := objMap[part]
			if !exists {
				return nil, fmt.Errorf("path part '%s' not found at index %d", part, i)
			}
			current = next
		}
	}

	return current, nil
}

// EnsureJSONFormat 确保数据是JSON格式
func EnsureJSONFormat(data any) (map[string]any, error) {
	switch v := data.(type) {
	case map[string]any:
		return v, nil
	case string:
		// 尝试解析JSON字符串
		var result map[string]any
		if err := json.Unmarshal([]byte(v), &result); err != nil {
			// 如果不是有效的JSON，将其包装为JSON
			return map[string]any{
				"value": v,
			}, nil
		}
		return result, nil
	default:
		// 将其他类型包装为JSON
		return map[string]any{
			"value": v,
		}, nil
	}
}

// RestructureOutputs 将节点结果从 { "nodeId.field": { "value": ... } } 格式
// 转换为 { "nodeId": { "field": value } } 格式，其中 value 统一为 string
// 兼容旧数据：如果字段值是 { "value": actualValue } 形式，先提取出 actualValue 再 stringfy
func RestructureOutputs(nodeResults map[string]*models.NodeResult) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for nodeID, nodeResult := range nodeResults {
		if nodeResult == nil {
			result[nodeID] = make(map[string]string)
			continue
		}

		fieldMap := make(map[string]string)
		for fieldName, rawValue := range nodeResult.Outputs {
			// 兼容旧数据：{ "value": actualValue } 形式，先提取出裸值
			extracted := extractValue(rawValue)

			// 扁平化处理：request.body → requestbody
			if fieldName == "request" {
				if reqMap, ok := extracted.(map[string]any); ok {
					if body, ok := reqMap["body"]; ok {
						fieldMap["requestbody"] = stringifyValue(extractValue(body))
						continue
					}
				}
			}

			fieldMap[fieldName] = stringifyValue(extracted)
		}
		// 添加 error 字段（如果有）
		if nodeResult.Error != "" {
			fieldMap["error"] = nodeResult.Error
		}
		result[nodeID] = fieldMap
	}

	return result
}

// extractValue 提取出 { "value": ... } 嵌套结构的裸值
// 如果不是这种结构则返回原始值
func extractValue(v any) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	// 只有当 map 只有 "value" 这一个键时，才认为是 EnsureJSONFormat 包装的结果
	if len(m) == 1 {
		if val, exists := m["value"]; exists {
			return val
		}
	}
	return v
}

// stringifyValue 将任意值转换为 string
// - 如果已经是 string，直接返回
// - 如果是数字（int/float），格式化为字符串（JSON 序列化保证精度）
// - 如果是 bool，转换为 "true"/"false"
// - 如果是其他类型，尝试 JSON 序列化
// - 如果是 nil，返回空字符串
func stringifyValue(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", val)
	case int8:
		return fmt.Sprintf("%d", val)
	case int16:
		return fmt.Sprintf("%d", val)
	case int32:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case uint:
		return fmt.Sprintf("%d", val)
	case uint8:
		return fmt.Sprintf("%d", val)
	case uint16:
		return fmt.Sprintf("%d", val)
	case uint32:
		return fmt.Sprintf("%d", val)
	case uint64:
		return fmt.Sprintf("%d", val)
	case float32:
		return fmt.Sprintf("%v", val)
	case float64:
		return fmt.Sprintf("%v", val)
	default:
		// 尝试 JSON 序列化（复杂对象或嵌套结构）
		jsonBytes, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(jsonBytes)
	}
}
