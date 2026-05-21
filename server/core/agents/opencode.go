package agents

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// OpencodeAgent implements the Agent interface for the `opencode` CLI tool.
type OpencodeAgent struct{}

// NewOpencodeAgent creates a new OpencodeAgent.
func NewOpencodeAgent() *OpencodeAgent {
	return &OpencodeAgent{}
}

// Name returns the agent identifier.
func (a *OpencodeAgent) Name() string {
	return "opencode"
}

// Run executes the opencode CLI with the given configuration.
func (a *OpencodeAgent) Run(ctx context.Context, workDir string, initMessage string, model string, mode string) (string, error) {
	args := []string{"run"}

	if mode != "" {
		args = append(args, "--agent", mode)
	}

	if model != "" {
		args = append(args, "--model", model)
	}

	args = append(args, "--", initMessage)

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = workDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Include stderr in the error for debugging
		if stderr.Len() > 0 {
			return "", fmt.Errorf("opencode run failed: %w\nstderr: %s", err, stderr.String())
		}
		return "", fmt.Errorf("opencode run failed: %w", err)
	}

	// If stderr has content but no error, still include it
	output := stdout.String()
	if stderr.Len() > 0 {
		output = output + "\n" + stderr.String()
	}

	return output, nil
}

// GetModels returns the list of available models from the opencode CLI.
func (a *OpencodeAgent) GetModels() ([]string, error) {
	cmd := exec.Command("opencode", "models")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("opencode models failed: %w\nstderr: %s", err, stderr.String())
	}

	return parseLines(stdout.String()), nil
}

// GetModes returns the list of available agents/modes from the opencode CLI.
func (a *OpencodeAgent) GetModes() ([]string, error) {
	cmd := exec.Command("opencode", "agents")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("opencode agents failed: %w\nstderr: %s", err, stderr.String())
	}

	return parseLines(stdout.String()), nil
}

// IsInstalled checks if the opencode binary is available on the system PATH.
func (a *OpencodeAgent) IsInstalled() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

// parseLines splits a multi-line string into a slice of non-empty trimmed lines.
func parseLines(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
