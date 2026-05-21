package agents

import "context"

// Agent defines the interface for an AI agent runner (e.g., opencode, claudecode).
type Agent interface {
	// Name returns the unique identifier for this agent type.
	Name() string

	// Run executes the agent with the given configuration and returns the output.
	Run(ctx context.Context, workDir string, initMessage string, model string, mode string) (string, error)

	// GetModels returns a list of available models for this agent.
	GetModels() ([]string, error)

	// GetModes returns a list of available modes/agents for this agent runner.
	GetModes() ([]string, error)

	// IsInstalled checks whether the agent binary is available on the system.
	IsInstalled() bool
}

// AgentInfo provides a summary of an agent for API responses.
type AgentInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// AgentManager manages multiple agent implementations.
type AgentManager interface {
	// List returns information about all registered agents.
	List() []AgentInfo

	// Get returns an agent by its ID.
	Get(id string) (Agent, error)
}
