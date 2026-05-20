package nodes

import "SREFlow/internal/models"

// Common utility functions shared across all nodes

func getStr(cfg map[string]any, key, defaultVal string) string {
	if v, ok := cfg[key].(string); ok && v != "" {
		return v
	}
	return defaultVal
}

func getInt(cfg map[string]any, key string, defaultVal int) int {
	if v, ok := cfg[key].(float64); ok {
		return int(v)
	}
	return defaultVal
}

func getBool(cfg map[string]any, key string, defaultVal bool) bool {
	if v, ok := cfg[key].(bool); ok {
		return v
	}
	return defaultVal
}

func getFloat64(cfg map[string]any, key string, defaultVal float64) float64 {
	if v, ok := cfg[key].(float64); ok {
		return v
	}
	return defaultVal
}

func newResult(outputs map[string]any) *models.NodeResult {
	return &models.NodeResult{Outputs: outputs}
}

func errorResult(err string) *models.NodeResult {
	return &models.NodeResult{Error: err}
}

// replaceString replaces all occurrences of old in s with new
func replaceString(s, old, new string) string {
	result := ""
	i := 0
	for i < len(s) {
		idx := findString(s, old, i)
		if idx == -1 {
			result += s[i:]
			break
		}
		result += s[i:idx] + new
		i = idx + len(old)
	}
	return result
}

// findString returns the index of the first occurrence of substr in s starting from start
func findString(s, substr string, start int) int {
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
