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
	return tui.RunTaskDashboardTUI(serverURL)
}
