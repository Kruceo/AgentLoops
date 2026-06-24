package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"agentloops/cli/client"
	"agentloops/cli/tui"
)

// taskStartCmd represents the task start command
var taskStartCmd = &cobra.Command{
	Use:          "start [task-id]",
	Short:        "Start a task immediately",
	Long:         `Trigger immediate execution of a task. If no task ID is provided, an interactive TUI lets you choose one.`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runTaskStart,
	SilenceUsage: true,
}

func init() {
	taskCmd.AddCommand(taskStartCmd)
}

func runTaskStart(command *cobra.Command, args []string) error {
	serverURL := getServerURL(command)
	c := client.NewClient(serverURL)
	ctx := context.Background()

	var taskID string
	if len(args) == 0 {
		// Interactive TUI selection
		selected, err := tui.RunStartTaskTUI(serverURL)
		if err != nil {
			return err
		}
		taskID = selected
	} else {
		taskID = args[0]
	}

	printInfo(fmt.Sprintf("Starting task %s...", taskID))

	// POST /api/tasks/:id/run → get run ID
	resp, err := c.RunTaskNow(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	if resp.Status == "already running" {
		printError(fmt.Sprintf("Task %s is already running", taskID))
		return nil
	}

	runID := resp.ID
	printInfo(fmt.Sprintf("Run %s started, waiting for completion...", runID))

	// Poll GET /api/runs/:id until finished.
	// The run record may not exist yet since RunTaskNow is async —
	// the goroutine creates the record after the agent finishes.
	pollCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	for {
		run, err := c.GetRun(pollCtx, runID)
		if err != nil || run == nil {
			// Run not created yet or not found — keep polling.
			select {
			case <-pollCtx.Done():
				return fmt.Errorf("timed out waiting for task to complete (run: %s)", runID)
			case <-time.After(1 * time.Second):
				continue
			}
		}

		if run.FinishedAt != nil {
			// Run completed
			if run.Output != "" {
				fmt.Println(run.Output)
			}

			if run.HasError {
				printError(fmt.Sprintf("Task failed (run: %s)", runID))
				return fmt.Errorf("task failed")
			}

			printSuccess(fmt.Sprintf("Task completed successfully (run: %s)", runID))
			return nil
		}

		select {
		case <-pollCtx.Done():
			return fmt.Errorf("timed out waiting for task to complete (run: %s)", runID)
		case <-time.After(1 * time.Second):
			// Continue polling
		}
	}
}