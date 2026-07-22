package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"agentloops/internal/client"
	"agentloops/internal/tui"
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

	// Stream run output via SSE with a 20-minute timeout.
	streamCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	defer cancel()

	events, err := c.StreamRunOutput(streamCtx, runID)
	if err != nil {
		return fmt.Errorf("failed to stream run output: %w", err)
	}

	for evt := range events {
		switch evt.Type {
		case "output":
			var text string
			if err := json.Unmarshal([]byte(evt.Data), &text); err != nil {
				fmt.Print(evt.Data)
			} else {
				fmt.Print(text)
			}

		case "error":
			var text string
			if err := json.Unmarshal([]byte(evt.Data), &text); err != nil {
				printError(evt.Data)
			} else {
				printError(text)
			}

		case "done":
			var doneData struct {
				Status string `json:"status"`
			}
			if err := json.Unmarshal([]byte(evt.Data), &doneData); err != nil {
				return fmt.Errorf("failed to parse done event: %w", err)
			}

			if doneData.Status == "success" {
				printSuccess(fmt.Sprintf("Task completed successfully (run: %s)", runID))
				return nil
			}

			printError(fmt.Sprintf("Task failed (run: %s)", runID))
			return fmt.Errorf("task failed")
		}
	}

	// If the channel closed without receiving a "done" event, check why.
	if streamCtx.Err() != nil {
		return fmt.Errorf("timed out waiting for task to complete (run: %s)", runID)
	}
	return fmt.Errorf("stream ended unexpectedly (run: %s)", runID)
}