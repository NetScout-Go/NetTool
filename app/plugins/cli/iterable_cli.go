package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/NetScout-Go/NetTool/app/plugins/types"
)

// IterableCLI provides a command-line interface for running plugins with iteration support
type IterableCLI struct {
	Plugin         types.IterablePlugin
	Params         map[string]interface{}
	MaxIterations  int
	IterationDelay time.Duration
	Results        []types.IterationResult
	RunInTerminal  bool
}

// NewIterableCLI creates a new CLI for iterable plugins
func NewIterableCLI(plugin types.IterablePlugin) *IterableCLI {
	return &IterableCLI{
		Plugin:         plugin,
		Params:         make(map[string]interface{}),
		MaxIterations:  0,
		IterationDelay: 5 * time.Second,
		Results:        []types.IterationResult{},
		RunInTerminal:  true,
	}
}

// SetParams sets the parameters for the plugin
func (cli *IterableCLI) SetParams(params map[string]interface{}) *IterableCLI {
	cli.Params = params
	return cli
}

// SetMaxIterations sets the maximum number of iterations
func (cli *IterableCLI) SetMaxIterations(max int) *IterableCLI {
	cli.MaxIterations = max
	return cli
}

// SetIterationDelay sets the delay between iterations
func (cli *IterableCLI) SetIterationDelay(delay time.Duration) *IterableCLI {
	cli.IterationDelay = delay
	return cli
}

// SetRunInTerminal sets whether to run in terminal mode
func (cli *IterableCLI) SetRunInTerminal(run bool) *IterableCLI {
	cli.RunInTerminal = run
	return cli
}

// Run executes the plugin with iteration support
func (cli *IterableCLI) Run() error {
	if !cli.Plugin.SupportsIteration() {
		// Run once without iteration
		result, err := cli.Plugin.Execute(cli.Params)
		if err != nil {
			return err
		}

		// Print result
		return cli.printResult(result, 0)
	}

	// Initialize iteration
	iterationCount := 0
	cli.Results = []types.IterationResult{}

	for {
		// Check max iterations
		if cli.MaxIterations > 0 && iterationCount >= cli.MaxIterations {
			break
		}

		// Execute the iteration
		result, continueIteration, err := cli.Plugin.ExecuteIteration(cli.Params, iterationCount)

		// Record the result
		iterationResult := types.IterationResult{
			IterationCount:    iterationCount,
			Result:            result,
			ContinueIteration: continueIteration,
			Timestamp:         time.Now(),
		}

		if err != nil {
			iterationResult.Error = err.Error()
		}

		cli.Results = append(cli.Results, iterationResult)

		// Print the result
		cli.printResult(result, iterationCount)

		// Stop if not continuing
		if !continueIteration && err == nil {
			break
		}

		// Stop on error
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			break
		}

		// Increment iteration count
		iterationCount++

		// Ask to continue iteration after every 5 iterations
		if iterationCount > 0 && iterationCount%5 == 0 {
			if !cli.promptContinueIteration(iterationCount) {
				fmt.Println("Iteration stopped by user.")
				break
			}
		}

		// Delay before next iteration
		if iterationCount < cli.MaxIterations || cli.MaxIterations == 0 {
			fmt.Printf("Waiting %s before next iteration...\n", cli.IterationDelay.String())
			time.Sleep(cli.IterationDelay)
		}
	}

	return nil
}

// printResult prints the result of a plugin execution
func (cli *IterableCLI) printResult(result interface{}, iterationCount int) error {
	// Prepare output
	var output string

	// Format based on terminal mode
	if cli.RunInTerminal {
		// Print formatted for terminal
		output = fmt.Sprintf("\n=== Iteration %d ===\n", iterationCount)

		// Convert result to formatted string
		resultJSON, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return err
		}

		output += string(resultJSON)
	} else {
		// Return JSON result
		resultMap := map[string]interface{}{
			"iterationCount": iterationCount,
			"result":         result,
			"timestamp":      time.Now().Format(time.RFC3339),
		}

		resultJSON, err := json.Marshal(resultMap)
		if err != nil {
			return err
		}

		output = string(resultJSON)
	}

	fmt.Println(output)
	return nil
}

// promptContinueIteration asks the user if they want to continue iterating
func (cli *IterableCLI) promptContinueIteration(iterationCount int) bool {
	fmt.Printf("\nCompleted %d iterations. Continue to iterate? (y/n): ", iterationCount)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// GetResults returns the results collected so far
func (cli *IterableCLI) GetResults() []types.IterationResult {
	return cli.Results
}

// SaveResultsToFile saves the results to a JSON file
func (cli *IterableCLI) SaveResultsToFile(filename string) error {
	// Convert results to JSON
	resultsJSON, err := json.MarshalIndent(cli.Results, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(filename, resultsJSON, 0644)
}
