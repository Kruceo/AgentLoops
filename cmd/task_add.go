package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"agentloops/internal/tui"
)

// taskAddCmd represents the task add command
var taskAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new task via interactive TUI wizard",
	Long: `Launch an interactive wizard to create a new agent task.

The wizard walks you through:
  - Task name and description
  - Agent selection (fetched from running server)
  - Model and mode selection
  - Working directory
  - Execution interval

Requires the server to be running at the configured URL.`,
	RunE: runTaskAdd,
}

func init() {
	taskCmd.AddCommand(taskAddCmd)

	taskAddCmd.Flags().Bool("non-interactive", false, "Skip TUI, use flags for automation (not yet implemented)")
}

func runTaskAdd(cmd *cobra.Command, args []string) error {
	serverURL := getServerURL(cmd)

	// Check if non-interactive mode is requested
	nonInteractive, _ := cmd.Flags().GetBool("non-interactive")
	if nonInteractive {
		return fmt.Errorf("non-interactive mode not yet implemented; use the TUI wizard instead")
	}

	// Verify server is reachable before launching TUI
	// We do a quick health check outside the TUI context
	// so we can show a clear error message
	fmt.Fprintf(os.Stderr, "Connecting to %s...\n", serverURL)

	// Launch the TUI wizard
	// Alt-screen is handled inside the model's View() via tea.View.AltScreen
	program := tea.NewProgram(tui.NewWizardModel(serverURL))

	result, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if the user quit without creating a task
	wm, ok := result.(tui.WizardModel)
	if ok && wm.CreatedTask == nil && !wm.Submitted {
		return fmt.Errorf("task was not created")
	}

	return nil
}