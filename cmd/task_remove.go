package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"agentloops/internal/client"
)

// taskRemoveCmd represents the task remove command
var taskRemoveCmd = &cobra.Command{
	Use:   "remove [task-id]",
	Short: "Remove a task by ID",
	Long:  `Remove a task by its ID. Requires confirmation unless --force is used.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskRemove,
}

func init() {
	taskCmd.AddCommand(taskRemoveCmd)

	taskRemoveCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}

func runTaskRemove(cmd *cobra.Command, args []string) error {
	taskID := args[0]
	force, _ := cmd.Flags().GetBool("force")

	serverURL := getServerURL(cmd)
	apiClient := client.NewClient(serverURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Health check
	if err := apiClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("cannot connect to server at %s: %w", serverURL, err)
	}

	// First, get the task to show details
	task, err := apiClient.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found: %w", err)
	}

	// Show task details
	fmt.Printf("Task to remove:\n")
	fmt.Printf("  ID:        %s\n", task.ID)
	fmt.Printf("  Name:      %s\n", task.TaskName)
	fmt.Printf("  Agent:     %s\n", task.AgentRunner)
	fmt.Printf("  Enabled:   %v\n", task.Enabled)
	fmt.Printf("  WorkDir:   %s\n", task.WorkDir)
	fmt.Println()

	// Confirm unless --force
	if !force {
		fmt.Print("Are you sure you want to delete this task? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			printInfo("Deletion cancelled")
			return nil
		}
	}

	// Delete the task
	if err := apiClient.DeleteTask(ctx, taskID); err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}

	printSuccess("Task %s deleted successfully", taskID)
	return nil
}