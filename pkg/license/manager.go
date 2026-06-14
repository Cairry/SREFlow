package license

import (
	"sync"
)

// Manager holds the current validated license
type Manager struct {
	mu     sync.RWMutex
	license *License
}

var globalManager = &Manager{}

// SetLicense sets the global license
func SetLicense(lic *License) {
	globalManager.mu.Lock()
	defer globalManager.mu.Unlock()
	globalManager.license = lic
}

// GetLicense returns the current license
func GetLicense() *License {
	globalManager.mu.RLock()
	defer globalManager.mu.RUnlock()
	return globalManager.license
}

// CheckWorkspaceLimit checks if the workspace count exceeds the limit
func CheckWorkspaceLimit(currentCount int) bool {
	lic := GetLicense()
	if lic == nil {
		return true // No license, allow
	}
	if lic.MaxWorkspaces == 0 {
		return true // Unlimited
	}
	return currentCount < lic.MaxWorkspaces
}

// CheckWorkflowLimit checks if the workflow count exceeds the limit
func CheckWorkflowLimit(currentCount int) bool {
	lic := GetLicense()
	if lic == nil {
		return true // No license, allow
	}
	if lic.MaxWorkflows == 0 {
		return true // Unlimited
	}
	return currentCount < lic.MaxWorkflows
}

// CheckMCPLimit checks if the MCP server count exceeds the limit
func CheckMCPLimit(currentCount int) bool {
	lic := GetLicense()
	if lic == nil {
		return true // No license, allow
	}
	if lic.MaxMCP == 0 {
		return true // Unlimited
	}
	return currentCount < lic.MaxMCP
}

// CheckSkillLimit checks if the skill count exceeds the limit
func CheckSkillLimit(currentCount int) bool {
	lic := GetLicense()
	if lic == nil {
		return true // No license, allow
	}
	if lic.MaxSkills == 0 {
		return true // Unlimited
	}
	return currentCount < lic.MaxSkills
}
