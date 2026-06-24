package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "agent-loop-orchestrator",
	Short: "Agent Loop Orchestrator - CLI for managing agent tasks",
	Long: `Agent Loop Orchestrator is a CLI tool for scheduling and managing
AI agent tasks with support for opencode and other agent runners.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// You can add global initialization here if needed
		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags that apply to all commands
	rootCmd.PersistentFlags().String("server", "", "API server URL (default: http://localhost:8080 or ALO_SERVER env var)")

	// Disable the auto-generated completion command
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}

// getServerURL returns the API server URL from flag or environment variable
func getServerURL(cmd *cobra.Command) string {
	url, _ := cmd.Flags().GetString("server")
	if url == "" {
		url = os.Getenv("ALO_SERVER")
	}
	if url == "" {
		url = "http://localhost:8080"
	}
	return url
}

// printError prints an error message to stderr
func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

// printSuccess prints a success message to stdout
func printSuccess(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

// printInfo prints an info message to stdout
func printInfo(format string, args ...interface{}) {
	fmt.Printf("ℹ "+format+"\n", args...)
}