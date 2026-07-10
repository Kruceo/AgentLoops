package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"agentloops/cli/client"
	"agentloops/cli/tui"
)

// taskDisableCmd represents the task disable command
var taskDisableCmd = &cobra.Command{
	Use:          "disable [task-id]",
	Short:        "Disable a task",
	Long:         `Disable a task by its ID. If no task ID is provided, an interactive TUI lets you choose one.`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runTaskDisable,
	SilenceUsage: true,
}

func init() {
	taskCmd.AddCommand(taskDisableCmd)
}

func runTaskDisable(cmd *cobra.Command, args []string) error {
	serverURL := getServerURL(cmd)
	apiClient := client.NewClient(serverURL)

	// Health check with 10-second timeout
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := apiClient.HealthCheck(healthCtx); err != nil {
		healthCancel()
		return fmt.Errorf("cannot connect to server at %s: %w", serverURL, err)
	}
	healthCancel()

	var taskID string
	if len(args) == 0 {
		// Interactive TUI selection
		selected, err := tui.RunTaskSelectTUI(serverURL, "Select a Task to Disable", "➜ Disable Task")
		if err != nil {
			return err
		}
		if selected == "" {
			printInfo("Cancelled")
			return nil
		}
		taskID = selected
	} else {
		taskID = args[0]
	}

	// Fetch task with 15-second timeout
	fetchCtx, fetchCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer fetchCancel()

	task, err := apiClient.GetTask(fetchCtx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	if !task.Enabled {
		printInfo(fmt.Sprintf("Task %s (%s) is already disabled", task.ID, task.TaskName))
		return nil
	}

	falseVal := false
	req := client.UpdateTaskRequest{
		Enabled: &falseVal,
	}

	// Update with 15-second timeout
	updateCtx, updateCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer updateCancel()

	updatedTask, err := apiClient.UpdateTask(updateCtx, taskID, req)
	if err != nil {
		return fmt.Errorf("failed to disable task: %w", err)
	}

	printSuccess(fmt.Sprintf("Task %s (%s) disabled", updatedTask.ID, updatedTask.TaskName))
	return nil
}
