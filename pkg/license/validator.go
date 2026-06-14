package license

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logc"
)

// Violation represents a license violation
type Violation struct {
	ResourceType string
	CurrentCount int
	MaxAllowed   int
	Message      string
}

// Validator checks license compliance
type Validator struct {
	countProvider CountProvider
	checkInterval time.Duration
	violations    chan Violation
}

// CountProvider provides resource counts
type CountProvider interface {
	CountWorkspaces() (int64, error)
	CountAllWorkflows() (int64, error)
	CountAllMCP() (int64, error)
	CountAllSkills() (int64, error)
}

// NewValidator creates a new license validator
func NewValidator(provider CountProvider, checkInterval time.Duration) *Validator {
	if checkInterval == 0 {
		checkInterval = 1 * time.Hour
	}
	return &Validator{
		countProvider: provider,
		checkInterval: checkInterval,
		violations:    make(chan Violation, 10),
	}
}

// ValidateStartup checks all limits at startup
func (v *Validator) ValidateStartup() []Violation {
	lic := GetLicense()
	if lic == nil {
		return nil
	}

	violations := []Violation{}

	// Check workspace limit
	if lic.MaxWorkspaces > 0 {
		count, err := v.countProvider.CountWorkspaces()
		if err == nil && int(count) > lic.MaxWorkspaces {
			violations = append(violations, Violation{
				ResourceType: "workspaces",
				CurrentCount: int(count),
				MaxAllowed:   lic.MaxWorkspaces,
				Message:      fmt.Sprintf("Workspace limit exceeded: %d/%d", count, lic.MaxWorkspaces),
			})
		}
	}

	// Check workflow limit (global)
	if lic.MaxWorkflows > 0 {
		count, err := v.countProvider.CountAllWorkflows()
		if err == nil && int(count) > lic.MaxWorkflows {
			violations = append(violations, Violation{
				ResourceType: "workflows",
				CurrentCount: int(count),
				MaxAllowed:   lic.MaxWorkflows,
				Message:      fmt.Sprintf("Workflow limit exceeded: %d/%d", count, lic.MaxWorkflows),
			})
		}
	}

	// Check MCP limit (global)
	if lic.MaxMCP > 0 {
		count, err := v.countProvider.CountAllMCP()
		if err == nil && int(count) > lic.MaxMCP {
			violations = append(violations, Violation{
				ResourceType: "mcp_servers",
				CurrentCount: int(count),
				MaxAllowed:   lic.MaxMCP,
				Message:      fmt.Sprintf("MCP server limit exceeded: %d/%d", count, lic.MaxMCP),
			})
		}
	}

	// Check skill limit (global)
	if lic.MaxSkills > 0 {
		count, err := v.countProvider.CountAllSkills()
		if err == nil && int(count) > lic.MaxSkills {
			violations = append(violations, Violation{
				ResourceType: "skills",
				CurrentCount: int(count),
				MaxAllowed:   lic.MaxSkills,
				Message:      fmt.Sprintf("Skill limit exceeded: %d/%d", count, lic.MaxSkills),
			})
		}
	}

	return violations
}

// StartPeriodicCheck starts periodic license validation
func (v *Validator) StartPeriodicCheck(ctx context.Context) {
	ticker := time.NewTicker(v.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			violations := v.ValidateStartup()
			for _, violation := range violations {
				logc.Errorf(ctx, "[License] VIOLATION: %s", violation.Message)
				select {
				case v.violations <- violation:
				default:
					// Channel full, skip
				}
			}
		}
	}
}

// GetViolations returns the violation channel
func (v *Validator) GetViolations() <-chan Violation {
	return v.violations
}
