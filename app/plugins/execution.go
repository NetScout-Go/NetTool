// Package plugins provides plugin execution with progress reporting and error handling.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/NetScout-Go/NetTool/app/plugins/types"
)

// ExecutionStatus represents the status of a plugin execution.
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// ExecutionProgress represents the progress of a plugin execution.
type ExecutionProgress struct {
	Current int    `json:"current"`
	Total   int    `json:"total"`
	Message string `json:"message"`
}

// ExecutionResult represents the result of a plugin execution.
type ExecutionResult struct {
	ID        string                 `json:"id"`
	PluginID  string                 `json:"pluginId"`
	Status    ExecutionStatus        `json:"status"`
	Progress  *ExecutionProgress     `json:"progress,omitempty"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	StartTime time.Time              `json:"startTime"`
	EndTime   *time.Time             `json:"endTime,omitempty"`
	Duration  int64                  `json:"duration"` // in milliseconds
	Params    map[string]interface{} `json:"params"`
}

// ExecutionContext provides context for a plugin execution.
type ExecutionContext struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ID        string
	PluginID  string
	Params    map[string]interface{}
	Config    *types.PluginExecutionConfig
	listeners []ProgressListener
	result    *ExecutionResult
	mu        sync.Mutex
}

// ProgressListener is called when execution progress updates.
type ProgressListener func(result *ExecutionResult)

// NewExecutionContext creates a new execution context.
func NewExecutionContext(pluginID string, params map[string]interface{}, config *types.PluginExecutionConfig) *ExecutionContext {
	ctx, cancel := context.WithCancel(context.Background())
	executionID := fmt.Sprintf("%s-%d", pluginID, time.Now().UnixNano())

	return &ExecutionContext{
		ctx:       ctx,
		cancel:    cancel,
		ID:        executionID,
		PluginID:  pluginID,
		Params:    params,
		Config:    config,
		listeners: []ProgressListener{},
		result: &ExecutionResult{
			ID:        executionID,
			PluginID:  pluginID,
			Status:    StatusPending,
			Params:    params,
			StartTime: time.Now(),
		},
	}
}

// Context returns the context for cancellation support.
func (ec *ExecutionContext) Context() context.Context {
	return ec.ctx
}

// Cancel cancels the execution.
func (ec *ExecutionContext) Cancel() {
	ec.cancel()
	ec.updateStatus(StatusCancelled, nil, "Execution cancelled by user")
}

// AddProgressListener adds a listener for progress updates.
func (ec *ExecutionContext) AddProgressListener(listener ProgressListener) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.listeners = append(ec.listeners, listener)
}

// UpdateProgress updates the execution progress.
func (ec *ExecutionContext) UpdateProgress(current, total int, message string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.result.Progress = &ExecutionProgress{
		Current: current,
		Total:   total,
		Message: message,
	}

	ec.notifyListeners()
}

// updateStatus updates the execution status.
func (ec *ExecutionContext) updateStatus(status ExecutionStatus, result interface{}, errMsg string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.result.Status = status
	ec.result.Result = result
	ec.result.Error = errMsg

	if status == StatusCompleted || status == StatusFailed || status == StatusCancelled {
		now := time.Now()
		ec.result.EndTime = &now
		ec.result.Duration = now.Sub(ec.result.StartTime).Milliseconds()
	}

	ec.notifyListeners()
}

// notifyListeners notifies all progress listeners.
// Must be called with mutex held.
func (ec *ExecutionContext) notifyListeners() {
	// Create a copy of the result
	resultCopy := *ec.result
	for _, listener := range ec.listeners {
		go listener(&resultCopy)
	}
}

// GetResult returns the current execution result.
func (ec *ExecutionContext) GetResult() *ExecutionResult {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	resultCopy := *ec.result
	return &resultCopy
}

// PluginExecutor handles enhanced plugin execution.
type PluginExecutor struct {
	manager    *PluginManager
	executions map[string]*ExecutionContext
	mu         sync.RWMutex
}

// NewPluginExecutor creates a new plugin executor.
func NewPluginExecutor(manager *PluginManager) *PluginExecutor {
	return &PluginExecutor{
		manager:    manager,
		executions: make(map[string]*ExecutionContext),
	}
}

// Execute runs a plugin with enhanced execution context.
func (pe *PluginExecutor) Execute(pluginID string, params map[string]interface{}, config *types.PluginExecutionConfig) *ExecutionContext {
	// Create execution context
	execCtx := NewExecutionContext(pluginID, params, config)

	// Store execution
	pe.mu.Lock()
	pe.executions[execCtx.ID] = execCtx
	pe.mu.Unlock()

	// Run in background
	go pe.runExecution(execCtx)

	return execCtx
}

// ExecuteSync runs a plugin synchronously with enhanced error handling.
func (pe *PluginExecutor) ExecuteSync(pluginID string, params map[string]interface{}) (*ExecutionResult, error) {
	execCtx := NewExecutionContext(pluginID, params, nil)
	pe.runExecution(execCtx)

	result := execCtx.GetResult()
	if result.Status == StatusFailed {
		return result, fmt.Errorf("%s", result.Error)
	}

	return result, nil
}

// runExecution performs the actual plugin execution.
func (pe *PluginExecutor) runExecution(execCtx *ExecutionContext) {
	// Update status to running
	execCtx.updateStatus(StatusRunning, nil, "")

	// Check if plugin exists
	plugin, err := pe.manager.GetPlugin(execCtx.PluginID)
	if err != nil {
		execCtx.updateStatus(StatusFailed, nil, fmt.Sprintf("Plugin not found: %s", execCtx.PluginID))
		return
	}

	// Validate parameters
	if err := pe.validateParams(plugin, execCtx.Params); err != nil {
		execCtx.updateStatus(StatusFailed, nil, fmt.Sprintf("Parameter validation failed: %v", err))
		return
	}

	// Check for cancellation
	select {
	case <-execCtx.ctx.Done():
		execCtx.updateStatus(StatusCancelled, nil, "Execution cancelled")
		return
	default:
	}

	// Handle iterable execution
	if execCtx.Config != nil && execCtx.Config.Iterate {
		pe.runIterableExecution(execCtx, plugin)
		return
	}

	// Single execution
	result, err := plugin.Execute(execCtx.Params)
	if err != nil {
		execCtx.updateStatus(StatusFailed, nil, fmt.Sprintf("Execution failed: %v", err))
		return
	}

	execCtx.updateStatus(StatusCompleted, result, "")
}

// runIterableExecution handles iterable plugin execution.
func (pe *PluginExecutor) runIterableExecution(execCtx *ExecutionContext, plugin *Plugin) {
	maxIterations := execCtx.Config.MaxIterations
	if maxIterations <= 0 {
		maxIterations = 100 // Default max
	}

	delay := time.Duration(execCtx.Config.IterationDelay) * time.Millisecond
	if delay <= 0 {
		delay = 1 * time.Second // Default delay
	}

	var results []interface{}

	for i := 0; i < maxIterations; i++ {
		// Check for cancellation
		select {
		case <-execCtx.ctx.Done():
			execCtx.updateStatus(StatusCancelled, results, "Execution cancelled")
			return
		default:
		}

		// Update progress
		execCtx.UpdateProgress(i+1, maxIterations, fmt.Sprintf("Running iteration %d of %d", i+1, maxIterations))

		// Run iteration
		result, err := plugin.Execute(execCtx.Params)
		if err != nil {
			if execCtx.Config.ContinueOnError {
				results = append(results, map[string]interface{}{
					"iteration": i + 1,
					"error":     err.Error(),
				})
			} else {
				execCtx.updateStatus(StatusFailed, results, fmt.Sprintf("Iteration %d failed: %v", i+1, err))
				return
			}
		} else {
			results = append(results, map[string]interface{}{
				"iteration": i + 1,
				"result":    result,
			})
		}

		// Wait before next iteration
		if i < maxIterations-1 {
			select {
			case <-execCtx.ctx.Done():
				execCtx.updateStatus(StatusCancelled, results, "Execution cancelled")
				return
			case <-time.After(delay):
			}
		}
	}

	execCtx.updateStatus(StatusCompleted, map[string]interface{}{
		"iterations": results,
		"count":      len(results),
	}, "")
}

// validateParams validates the parameters against the plugin definition.
func (pe *PluginExecutor) validateParams(plugin *Plugin, params map[string]interface{}) error {
	for _, param := range plugin.Parameters {
		value, exists := params[param.ID]

		// Check required parameters
		if param.Required && !exists {
			return fmt.Errorf("missing required parameter: %s", param.ID)
		}

		// Skip validation for optional parameters that aren't provided
		if !exists {
			continue
		}

		// Type validation
		if err := pe.validateParamType(param, value); err != nil {
			return fmt.Errorf("parameter '%s': %v", param.ID, err)
		}
	}

	return nil
}

// validateParamType validates a parameter value against its type.
func (pe *PluginExecutor) validateParamType(param Parameter, value interface{}) error {
	switch param.Type {
	case TypeString:
		if _, ok := value.(string); !ok {
			return fmt.Errorf("expected string, got %T", value)
		}

	case TypeNumber, TypeRange:
		switch v := value.(type) {
		case float64:
			if param.Min != nil && v < *param.Min {
				return fmt.Errorf("value %v is below minimum %v", v, *param.Min)
			}
			if param.Max != nil && v > *param.Max {
				return fmt.Errorf("value %v is above maximum %v", v, *param.Max)
			}
		case int:
			fv := float64(v)
			if param.Min != nil && fv < *param.Min {
				return fmt.Errorf("value %v is below minimum %v", v, *param.Min)
			}
			if param.Max != nil && fv > *param.Max {
				return fmt.Errorf("value %v is above maximum %v", v, *param.Max)
			}
		case json.Number:
			fv, _ := v.Float64()
			if param.Min != nil && fv < *param.Min {
				return fmt.Errorf("value %v is below minimum %v", fv, *param.Min)
			}
			if param.Max != nil && fv > *param.Max {
				return fmt.Errorf("value %v is above maximum %v", fv, *param.Max)
			}
		default:
			return fmt.Errorf("expected number, got %T", value)
		}

	case TypeBoolean:
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("expected boolean, got %T", value)
		}

	case TypeSelect:
		// Validate that the value is one of the allowed options
		valid := false
		for _, opt := range param.Options {
			if opt.Value == value {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid option: %v", value)
		}
	}

	return nil
}

// GetExecution returns an execution context by ID.
func (pe *PluginExecutor) GetExecution(executionID string) *ExecutionContext {
	pe.mu.RLock()
	defer pe.mu.RUnlock()
	return pe.executions[executionID]
}

// GetActiveExecutions returns all active executions.
func (pe *PluginExecutor) GetActiveExecutions() []*ExecutionResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var active []*ExecutionResult
	for _, exec := range pe.executions {
		result := exec.GetResult()
		if result.Status == StatusPending || result.Status == StatusRunning {
			active = append(active, result)
		}
	}

	return active
}

// CancelExecution cancels an execution by ID.
func (pe *PluginExecutor) CancelExecution(executionID string) error {
	pe.mu.RLock()
	exec, exists := pe.executions[executionID]
	pe.mu.RUnlock()

	if !exists {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	exec.Cancel()
	return nil
}

// CleanupOldExecutions removes old completed executions.
func (pe *PluginExecutor) CleanupOldExecutions(maxAge time.Duration) int {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, exec := range pe.executions {
		result := exec.GetResult()

		// Only remove completed/failed/cancelled executions
		if result.Status != StatusPending && result.Status != StatusRunning {
			if result.EndTime != nil && result.EndTime.Before(cutoff) {
				delete(pe.executions, id)
				removed++
			}
		}
	}

	return removed
}
