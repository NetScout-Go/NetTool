package types

import (
	"fmt"
	"time"
)

// IterationResult represents the result of a plugin iteration
type IterationResult struct {
	IterationCount    int         `json:"iterationCount"`    // Current iteration count
	Result            interface{} `json:"result"`            // Result from this iteration
	ContinueIteration bool        `json:"continueIteration"` // Whether to continue iterating
	Error             string      `json:"error,omitempty"`   // Error message, if any
	Timestamp         time.Time   `json:"timestamp"`         // Time this iteration was completed
}

// IterationManager handles the execution of iterable plugins
type IterationManager struct {
	plugin         IterablePlugin
	config         PluginExecutionConfig
	results        []IterationResult
	isRunning      bool
	stopRequested  bool
	completionChan chan bool
}

// NewIterationManager creates a new iteration manager for a plugin
func NewIterationManager(plugin IterablePlugin, config PluginExecutionConfig) *IterationManager {
	return &IterationManager{
		plugin:         plugin,
		config:         config,
		results:        []IterationResult{},
		isRunning:      false,
		stopRequested:  false,
		completionChan: make(chan bool, 1),
	}
}

// Start begins the iteration process
func (im *IterationManager) Start(params map[string]interface{}) error {
	if im.isRunning {
		return fmt.Errorf("iteration is already running")
	}

	if !im.plugin.SupportsIteration() {
		return fmt.Errorf("plugin does not support iteration")
	}

	im.isRunning = true
	im.stopRequested = false
	im.results = []IterationResult{}

	// Execute iterations in a goroutine
	go func() {
		iterationCount := 0

		for {
			// Check if stop requested
			if im.stopRequested {
				break
			}

			// Check max iterations
			if im.config.MaxIterations > 0 && iterationCount >= im.config.MaxIterations {
				break
			}

			// Execute the iteration
			result, continueIteration, err := im.plugin.ExecuteIteration(params, iterationCount)

			// Record the result
			iterationResult := IterationResult{
				IterationCount:    iterationCount,
				Result:            result,
				ContinueIteration: continueIteration,
				Timestamp:         time.Now(),
			}

			if err != nil {
				iterationResult.Error = err.Error()
				iterationResult.ContinueIteration = im.config.ContinueOnError
			}

			im.results = append(im.results, iterationResult)

			// Stop if requested not to continue
			if !continueIteration && err == nil {
				break
			}

			// Stop on error if not configured to continue
			if err != nil && !im.config.ContinueOnError {
				break
			}

			// Increment iteration count
			iterationCount++

			// Delay before next iteration
			if im.config.IterationDelay > 0 {
				time.Sleep(time.Duration(im.config.IterationDelay) * time.Millisecond)
			}
		}

		im.isRunning = false
		im.completionChan <- true
	}()

	return nil
}

// Stop requests the iteration to stop
func (im *IterationManager) Stop() {
	if im.isRunning {
		im.stopRequested = true
	}
}

// IsRunning returns whether the iteration is currently running
func (im *IterationManager) IsRunning() bool {
	return im.isRunning
}

// GetResults returns the results collected so far
func (im *IterationManager) GetResults() []IterationResult {
	return im.results
}

// WaitForCompletion waits for the iteration to complete
func (im *IterationManager) WaitForCompletion() {
	if im.isRunning {
		<-im.completionChan
	}
}

// PromptToContinueIteration asks whether to continue iterating
// This can be used by UI to prompt users with "Continue to iterate?"
func (im *IterationManager) PromptToContinueIteration() bool {
	// This function is a placeholder that would typically be implemented in UI code
	// It represents the functionality behind the "Continue to iterate?" prompt

	// The real implementation would show a UI dialog or prompt
	// Here we just return true to continue iteration for demonstration
	return true
}
