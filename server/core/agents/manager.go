package agents

import "fmt"

// DefaultAgentManager is the standard implementation of AgentManager.
type DefaultAgentManager struct {
	agents map[string]Agent
}

// NewDefaultAgentManager creates a new DefaultAgentManager with the given agent map.
func NewDefaultAgentManager(agents map[string]Agent) *DefaultAgentManager {
	return &DefaultAgentManager{agents: agents}
}

// List returns information about all registered agents.
func (m *DefaultAgentManager) List() []AgentInfo {
	var infos []AgentInfo
	for id, agent := range m.agents {
		infos = append(infos, AgentInfo{
			ID:        id,
			Name:      agent.Name(),
			Installed: agent.IsInstalled(),
		})
	}
	return infos
}

// Get returns an agent by its ID.
func (m *DefaultAgentManager) Get(id string) (Agent, error) {
	agent, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}
