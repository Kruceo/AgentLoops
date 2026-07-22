package cmd

import (
	"github.com/spf13/cobra"

	"agentloops/internal/tui"
)

// taskEditCmd represents the task edit command
var taskEditCmd = &cobra.Command{
	Use:   "edit [task-id]",
	Short: "Edit an existing task via interactive TUI wizard",
	Long: `Launch an interactive wizard to edit an existing agent task.

The wizard pre-fills all fields with current values and lets you:
  - Change task name and init message
  - Switch agent, model, or mode
  - Change working directory
  - Adjust execution interval

Only changed fields are submitted to the server.

If no task ID is provided, shows a list of tasks to choose from.`,
	Args:         cobra.MaximumNArgs(1),
	RunE:         runTaskEdit,
	SilenceUsage: true,
}

func init() {
	taskCmd.AddCommand(taskEditCmd)
}

func runTaskEdit(cmd *cobra.Command, args []string) error {
	serverURL := getServerURL(cmd)

	var taskID string
	if len(args) == 1 {
		taskID = args[0]
	} else {
		// No ID provided — let user pick via TUI
		selected, err := tui.RunTaskSelectTUI(serverURL, "Select a Task to Edit", "➜ Edit Task")
		if err != nil {
			return err
		}
		if selected == "" {
			printInfo("No task selected")
			return nil
		}
		taskID = selected
	}

	if err := tui.RunEditWizardTUI(taskID, serverURL); err != nil {
		return err
	}

	return nil
}
