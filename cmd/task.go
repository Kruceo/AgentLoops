package cmd

import (
	"github.com/spf13/cobra"

	"agentloops/cli/tui"
)

// taskCmd represents the task command
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage agent tasks",
	Long:  `Commands for creating, listing, and removing agent tasks.`,
	RunE:  runTaskDashboard,
}

func init() {
	rootCmd.AddCommand(taskCmd)

	// The --server flag is inherited from rootCmd.PersistentFlags()
}

func runTaskDashboard(cmd *cobra.Command, args []string) error {
	serverURL := getServerURL(cmd)
	for {
		action, taskID, err := tui.RunTaskDashboardTUI(serverURL)
		if err != nil {
			return err
		}
		switch action {
		case tui.DashboardQuit:
			return nil
		case tui.DashboardCreateTask:
			_, _ = tui.RunCreateWizardTUI(serverURL)
			// Loop back to dashboard
		case tui.DashboardEditTask:
			_ = tui.RunEditWizardTUI(taskID, serverURL)
			// Loop back to dashboard
		}
	}
}
