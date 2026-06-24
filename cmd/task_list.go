package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"agentloops/cli/client"
)

// taskListCmd represents the task list command
var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	Long:  `Display all tasks in a tabular format.`,
	RunE:  runTaskList,
}

func init() {
	taskCmd.AddCommand(taskListCmd)

	taskListCmd.Flags().Bool("json", false, "Output as JSON")
	taskListCmd.Flags().Bool("enabled-only", false, "Show only enabled tasks")
}

func runTaskList(cmd *cobra.Command, args []string) error {
	serverURL := getServerURL(cmd)
	apiClient := client.NewClient(serverURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Health check
	if err := apiClient.HealthCheck(ctx); err != nil {
		return fmt.Errorf("cannot connect to server at %s: %w", serverURL, err)
	}

	tasks, err := apiClient.ListTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to list tasks: %w", err)
	}

	// Filter enabled only if requested
	enabledOnly, _ := cmd.Flags().GetBool("enabled-only")
	if enabledOnly {
		var filtered []client.Task
		for _, t := range tasks {
			if t.Enabled {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	// Check for JSON output
	asJSON, _ := cmd.Flags().GetBool("json")
	if asJSON {
		return printTasksJSON(tasks)
	}

	// Print as table
	return printTasksTable(tasks)
}

func printTasksJSON(tasks []client.Task) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal tasks: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func printTasksTable(tasks []client.Task) error {
	if len(tasks) == 0 {
		printInfo("No tasks found")
		return nil
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("8")).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("8"))

	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Width(12)
	nameStyle := lipgloss.NewStyle().Width(25)
	agentStyle := lipgloss.NewStyle().Width(15)
	statusStyle := lipgloss.NewStyle().Width(12)
	intervalStyle := lipgloss.NewStyle().Width(10)
	workDirStyle := lipgloss.NewStyle().Width(30)

	// Print header
	fmt.Println(headerStyle.Render(fmt.Sprintf("%-12s %-25s %-15s %-12s %-10s %-30s",
		"ID", "NAME", "AGENT", "STATUS", "INTERVAL", "WORKDIR")))

	// Print rows
	for i, task := range tasks {
		rowStyle := lipgloss.NewStyle()
		if i%2 == 1 {
			rowStyle = rowStyle.Background(lipgloss.Color("235"))
		}

		id := idStyle.Render(truncate(task.ID, 12))
		name := nameStyle.Render(truncate(task.TaskName, 25))
		agent := agentStyle.Render(truncate(task.AgentRunner, 15))

		var statusStyled string
		if task.Enabled {
			statusStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("enabled")
		} else {
			statusStyled = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("disabled")
		}
		statusStyled = statusStyle.Render(statusStyled)

		interval := intervalStyle.Render(fmt.Sprintf("%ds", task.IntervalSeconds))
		workDir := workDirStyle.Render(truncate(task.WorkDir, 30))

		fmt.Println(rowStyle.Render(fmt.Sprintf("%s %s %s %s %s %s",
			id, name, agent, statusStyled, interval, workDir)))
	}

	fmt.Printf("\nTotal: %d task(s)\n", len(tasks))
	return nil
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}