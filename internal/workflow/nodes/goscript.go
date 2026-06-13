package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"SREFlow/internal/models"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
	"github.com/zeromicro/go-zero/core/logc"
)

// GoScriptExecutor GoScript代码执行器
type GoScriptExecutor struct {
	DefaultNodeExecutor
}

// NewGoScriptExecutor 创建GoScript执行器
func NewGoScriptExecutor() *GoScriptExecutor {
	return &GoScriptExecutor{
		DefaultNodeExecutor: DefaultNodeExecutor{
			NodeType:    "GoScript",
			Category:    "processing",
			Description: "Execute GoScript code for custom data processing",
		},
	}
}

// ExecuteGoScript 执行GoScript代码
func ExecuteGoScript(ctx context.Context, nc *models.NodeContext) (*models.NodeResult, error) {
	gsExecutor := NewGoScriptExecutor()
	return gsExecutor.Execute(ctx, nc)
}

func (e *GoScriptExecutor) Execute(ctx context.Context, nodeCtx *models.NodeContext) (*models.NodeResult, error) {
	// 验证配置
	code, ok := nodeCtx.Config["code"].(string)
	if !ok || strings.TrimSpace(code) == "" {
		return e.createErrorResult(fmt.Errorf("code not found or empty"), "validation"), nil
	}

	timeout := 30 * time.Second // 默认 30 秒
	if t, ok := nodeCtx.Config["timeout"].(float64); ok && t > 0 {
		// timeout 配置以毫秒为单位
		timeout = time.Duration(t) * time.Millisecond
	}

	// 构建输入数据
	inputData := make(map[string]interface{})
	for key, value := range nodeCtx.Input {
		inputData[key] = value
	}

	// 执行Go代码
	result, err := e.executeGoCode(ctx, code, inputData, timeout)
	if err != nil {
		return e.createErrorResult(err, "execution"), nil
	}

	return &models.NodeResult{
		Outputs: result,
	}, nil
}

// executeGoCode 执行Go代码
func (e *GoScriptExecutor) executeGoCode(ctx context.Context, code string, input map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	// 验证代码基本语法
	if err := e.validateGoCode(code); err != nil {
		return nil, fmt.Errorf("code validation error: %v", err)
	}

	// 创建yaegi解释器
	i := interp.New(interp.Options{})

	// 导入标准库符号
	for name, symbols := range stdlib.Symbols {
		if err := i.Use(interp.Exports{name: symbols}); err != nil {
			logc.Infof(ctx, "[GoScript] Warning: failed to import package %s: %v", name, err)
		}
	}

	// 先声明输出变量（使用 map 类型以便支持结构化数据）
	_, err := i.Eval("var output map[string]interface{}")
	if err != nil {
		logc.Infof(ctx, "[GoScript] Warning: failed to declare output variable: %v", err)
	}

	// 创建带超时的执行通道
	resultChan := make(chan map[string]interface{}, 1)
	errChan := make(chan error, 1)

	go func() {
		// 使用 defer 捕获 panic
		defer func() {
			if r := recover(); r != nil {
				errChan <- fmt.Errorf("code execution panicked: %v", r)
			}
		}()

		// 执行用户代码
		_, err := i.Eval(code)
		if err != nil {
			errChan <- fmt.Errorf("execution error: %v", err)
			return
		}

		// 尝试调用 main 函数（如果存在）
		if _, err := i.Eval("main"); err == nil {
			if mainResult, err := i.Eval("main()"); err == nil {
				if resultMap := e.extractToMap(mainResult); resultMap != nil {
					resultChan <- resultMap
					return
				}
				if wrapped := e.wrapValue(mainResult); wrapped != nil {
					resultChan <- wrapped
					return
				}
			}
		}

		// 获取 output 变量的值
		if outputVal, err := i.Eval("output"); err == nil {
			if resultMap := e.wrapValue(outputVal); resultMap != nil {
				resultChan <- resultMap
				return
			}
		}

		// 如果没有结构化数据，尝试获取 result 变量
		if resultVal, err := i.Eval("result"); err == nil {
			if resultMap := e.wrapValue(resultVal); resultMap != nil {
				resultChan <- resultMap
				return
			}
		}

		// 如果都没有，返回空 map
		resultChan <- make(map[string]interface{})
	}()

	// 等待执行完成或超时
	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	case <-time.After(timeout):
		return nil, fmt.Errorf("execution timeout after %v", timeout)
	}
}

// wrapValue 将任意值包装为符合 nodeId.output 调用约定的 map[string]interface{}
// - 基本类型(string/int/float/bool)：{"output": value} → 通过 {{nodeId.output}} 直接获取
// - 复杂类型(数组/map)：{"output": {...}} → 通过 {{nodeId.output.key}} 获取子属性
func (e *GoScriptExecutor) wrapValue(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}

	// 如果是 reflect.Value，解包
	if rv, ok := v.(reflect.Value); ok {
		v = rv.Interface()
	}

	// 如果是 map[string]interface{}，直接返回（让 extractToMap 处理）
	if _, ok := v.(map[string]interface{}); ok {
		return nil // 让 extractToMap 处理，返回的 map 会被包装
	}

	// 其他类型（基本类型、切片、结构体等）统一包装在 output 键下
	// 这样调用方可以通过 {{nodeId.output}} 直接获取值
	return map[string]interface{}{"output": v}
}

// extractToMap 将解释器的返回值转换为 map[string]interface{}
// 返回值统一包装在 "output" 键下，以便调用方通过 {{nodeId.output.key}} 访问
func (e *GoScriptExecutor) extractToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}

	// 如果是 reflect.Value，解包
	if rv, ok := v.(reflect.Value); ok {
		v = rv.Interface()
	}

	// 尝试 JSON 序列化/反序列化来转换复杂类型
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		// 如果无法序列化，将值直接包装在 output 下
		if str, ok := v.(string); ok {
			return map[string]interface{}{"output": str}
		}
		return map[string]interface{}{"output": fmt.Sprintf("%v", v)}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		// 如果无法反序列化为 map，将值直接包装在 output 下
		if str, ok := v.(string); ok {
			return map[string]interface{}{"output": str}
		}
		return map[string]interface{}{"output": fmt.Sprintf("%v", v)}
	}

	// 成功反序列化为 map，将整个 map 包装在 output 键下
	// 调用方可通过 {{nodeId.output}} 获取整个 map，{{nodeId.output.key}} 获取子属性
	return map[string]interface{}{"output": result}
}

// createErrorResult 创建错误结果
func (e *GoScriptExecutor) createErrorResult(err error, errorType string) *models.NodeResult {
	return &models.NodeResult{
		Outputs: map[string]interface{}{
			"error":      err.Error(),
			"error_type": errorType,
			"success":    false,
		},
	}
}

// validateGoCode 验证Go代码的兼容性
func (e *GoScriptExecutor) validateGoCode(code string) error {
	code = strings.TrimSpace(code)

	// 检查代码是否为空
	if code == "" {
		return fmt.Errorf("code is empty")
	}

	// yaegi 不支持的特性检测（使用正则表达式匹配任意空白）
	unsupported := []struct {
		pattern *regexp.Regexp
		reason  string
	}{
		{regexp.MustCompile(`\bgo\s+func\s*\(`), "goroutine is not supported in GoScript"},
		{regexp.MustCompile(`\bselect\s*\{`), "select statement is not fully supported"},
		{regexp.MustCompile(`\bchan\s`), "channel operations are not fully supported"},
		{regexp.MustCompile(`<-`), "channel receive operator is not supported"},
		{regexp.MustCompile(`\bdefer\s`), "defer is not fully supported in all contexts"},
		{regexp.MustCompile(`\brange\s+`), "range with channels is not supported"},
		{regexp.MustCompile(`\bfallthrough`), "fallthrough is not supported"},
		{regexp.MustCompile(`\bpackage\s+\w+`), "package declaration is not allowed in GoScript"},
	}

	for _, u := range unsupported {
		if u.pattern.MatchString(code) {
			return fmt.Errorf("%s", u.reason)
		}
	}

	return nil
}
