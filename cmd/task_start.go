package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"agentloops/cli/client"
)

// taskStartCmd represents the task start command
var taskStartCmd = &cobra.Command{
	Use:          "start <task-id>",
	Short:        "Start a task immediately with streaming output",
	Long:         `Trigger immediate execution of a task and stream its output to the terminal.`,
	Args:         cobra.ExactArgs(1),
	RunE:         runTaskStart,
	SilenceUsage: true,
}

func init() {
	taskCmd.AddCommand(taskStartCmd)
}

func runTaskStart(command *cobra.Command, args []string) error {
	taskID := args[0]
	serverURL := getServerURL(command)

	c := client.NewClient(serverURL)

	printInfo(fmt.Sprintf("Starting task %s...", taskID))

	ctx := context.Background()
	eventCh, err := c.StartTaskStream(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to start task: %w", err)
	}

	for event := range eventCh {
		switch event.Type {
		case "output":
			fmt.Print(event.Content)
		case "error":
			return fmt.Errorf("task error: %s", event.Content)
		case "done":
			if event.Status == "success" {
				printSuccess(fmt.Sprintf("Task completed successfully (run: %s)", event.RunID))
				return nil
			}
			return fmt.Errorf("task failed (run: %s)", event.RunID)
		}
	}

	return nil
}
