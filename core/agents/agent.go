package agents

import "context"

// OutputChunk represents a piece of output from a running agent.
type OutputChunk struct {
	Text string `json:"text"`
}

// Agent defines the interface for an AI agent runner (e.g., opencode, claudecode).
type Agent interface {
	// Name returns the unique identifier for this agent type.
	Name() string

	// Run executes the agent with the given configuration and returns the output.
	Run(ctx context.Context, workDir string, initMessage string, model string, mode string) (string, error)

	// RunStreaming executes the agent and emits output chunks in real-time via the provided channel.
	// The channel is closed when execution completes (whether success or error).
	// If an error occurs, the last chunk sent before closing contains the error message,
	// and the method returns that error.
	// The caller is responsible for consuming all chunks from the channel.
	RunStreaming(ctx context.Context, workDir string, initMessage string, model string, mode string, chunks chan<- OutputChunk) (string, error)

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
