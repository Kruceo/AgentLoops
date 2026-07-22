package agents

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"unicode"
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

// Run executes the opencode CLI with the given configuration and streams output
// in real-time via the chunks channel. The channel is closed when execution completes.
func (a *OpencodeAgent) Run(ctx context.Context, workDir string, initMessage string, model string, mode string, chunks chan<- OutputChunk) (string, error) {
	args := []string{"run"}

	if mode != "" {
		args = append(args, "--agent", mode)
	}

	if model != "" {
		args = append(args, "--model", model)
	}

	args = append(args, "--dangerously-skip-permissions")
	args = append(args, initMessage)

	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = workDir

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start opencode: %w", err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup

	// Read stdout in a goroutine, line by line
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			select {
			case chunks <- OutputChunk{Text: line}:
			case <-ctx.Done():
				return
			}
			stdoutBuf.WriteString(line)
			_, _ = os.Stdout.WriteString(line)
		}
	}()

	// Read stderr in a goroutine, line by line
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			select {
			case chunks <- OutputChunk{Text: line}:
			case <-ctx.Done():
				return
			}
			stderrBuf.WriteString(line)
			_, _ = os.Stdout.WriteString(line)
		}
	}()

	// Wait for the command to finish (pipes will close when process exits)
	err = cmd.Wait()

	// Wait for both goroutines to finish reading
	wg.Wait()

	// Close the chunks channel to signal completion
	close(chunks)

	if err != nil {
		if stderrBuf.Len() > 0 {
			return "", fmt.Errorf("opencode run failed: %w\nstderr: %s", err, stderrBuf.String())
		}
		return "", fmt.Errorf("opencode run failed: %w", err)
	}

	// Include stderr in the output even without error (matching Run behavior)
	output := stdoutBuf.String()
	if stderrBuf.Len() > 0 {
		output = output + "\n" + stderrBuf.String()
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
	cmd := exec.Command("opencode", "agent", "list")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = io.MultiWriter(&stdout, os.Stdout)
	cmd.Stderr = io.MultiWriter(&stderr, os.Stdout)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("opencode agent list failed: %w\nstderr: %s", err, stderr.String())
	}

	return parseModes(stdout.String()), nil
}

// parseModes extracts agent mode names from the opencode agent list output.
// The output format has mode names as the first word on non-indented lines,
// e.g. "build (primary)", "chat (primary)", "code-reviewer (subagent)".
func parseModes(output string) []string {
	lines := strings.Split(output, "\n")
	result := make([]string, 0)
	seen := map[string]bool{}
	for _, line := range lines {
		// Only consider non-indented lines (mode headers)
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' {
			continue
		}
		// Extract first word before space or parenthesis
		fields := strings.Fields(line)
		if len(fields) > 0 {
			name := fields[0]
			if len(name) > 0 && !unicode.IsLetter(rune(name[0])) {
				continue
			}
			if !seen[name] {
				seen[name] = true
				result = append(result, name)
			}
		}
	}
	return result
}

// IsInstalled checks if the opencode binary is available on the system PATH.
func (a *OpencodeAgent) IsInstalled() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

// parseLines splits a multi-line string into a slice of non-empty trimmed lines.
func parseLines(output string) []string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	result := make([]string, 0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
