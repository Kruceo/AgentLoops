package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const maxResponseSize = 10 << 20 // 10 MB

// Client is an HTTP client for the Agent Loop Orchestrator API.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// AgentInfo represents an agent from the API.
type AgentInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

// AgentDetail represents detailed agent information including models and modes.
type AgentDetail struct {
	AgentInfo
	Models []string `json:"models,omitempty"`
	Modes  []string `json:"modes,omitempty"`
}

// Task represents a task from the API.
type Task struct {
	ID              string     `json:"id"`
	TaskName        string     `json:"taskName"`
	InitMessage     string     `json:"initMessage"`
	AgentRunner     string     `json:"agentRunner"`
	AgentModel      string     `json:"agentModel"`
	AgentMode       string     `json:"agentMode"`
	WorkDir         string     `json:"workDir"`
	Enabled         bool       `json:"enabled"`
	CronExpr        string     `json:"cronExpr"`
	IntervalSeconds int        `json:"intervalSeconds"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	LastRunStatus   *string    `json:"lastRunStatus,omitempty"`
}

// CreateTaskRequest represents the request body for creating a task.
type CreateTaskRequest struct {
	TaskName        string `json:"taskName"`
	InitMessage     string `json:"initMessage"`
	AgentRunner     string `json:"agentRunner"`
	AgentModel      string `json:"agentModel"`
	AgentMode       string `json:"agentMode"`
	WorkDir         string `json:"workDir"`
	Enabled         *bool  `json:"enabled,omitempty"`
	CronExpr        string `json:"cronExpr,omitempty"`
	IntervalSeconds int    `json:"intervalSeconds,omitempty"`
}

// NewClient creates a new API client with the given base URL.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request and decodes the JSON response.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	limiter := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limiter)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if len(respBody) > maxResponseSize {
		return fmt.Errorf("response too large (%d bytes)", len(respBody))
	}

	if resp.StatusCode >= 400 {
		var apiErr map[string]string
		if err := json.Unmarshal(respBody, &apiErr); err == nil {
			if msg, ok := apiErr["error"]; ok {
				return errors.New(msg)
			}
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}

// ListAgents returns all registered agents.
func (c *Client) ListAgents(ctx context.Context) ([]AgentInfo, error) {
	var agents []AgentInfo
	err := c.doRequest(ctx, http.MethodGet, "/api/agents", nil, &agents)
	return agents, err
}

// GetAgentModels returns available models for a specific agent.
func (c *Client) GetAgentModels(ctx context.Context, agentID string) ([]string, error) {
	var models []string
	err := c.doRequest(ctx, http.MethodGet, "/api/agents/"+url.PathEscape(agentID)+"/models", nil, &models)
	return models, err
}

// GetAgentModes returns available modes for a specific agent.
func (c *Client) GetAgentModes(ctx context.Context, agentID string) ([]string, error) {
	var modes []string
	err := c.doRequest(ctx, http.MethodGet, "/api/agents/"+url.PathEscape(agentID)+"/modes", nil, &modes)
	return modes, err
}

// GetAgentDetail returns an agent with its models and modes.
func (c *Client) GetAgentDetail(ctx context.Context, agentID string) (*AgentDetail, error) {
	agentInfo, err := c.getAgentInfo(ctx, agentID)
	if err != nil {
		return nil, err
	}

	models, _ := c.GetAgentModels(ctx, agentID)
	modes, _ := c.GetAgentModes(ctx, agentID)

	return &AgentDetail{
		AgentInfo: *agentInfo,
		Models:    models,
		Modes:     modes,
	}, nil
}

func (c *Client) getAgentInfo(ctx context.Context, agentID string) (*AgentInfo, error) {
	agents, err := c.ListAgents(ctx)
	if err != nil {
		return nil, err
	}
	for _, a := range agents {
		if a.ID == agentID {
			return &a, nil
		}
	}
	return nil, errors.New("agent not found: " + agentID)
}

// ListTasks returns all tasks.
func (c *Client) ListTasks(ctx context.Context) ([]Task, error) {
	var tasks []Task
	err := c.doRequest(ctx, http.MethodGet, "/api/tasks", nil, &tasks)
	return tasks, err
}

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, req CreateTaskRequest) (*Task, error) {
	var task Task
	err := c.doRequest(ctx, http.MethodPost, "/api/tasks", req, &task)
	return &task, err
}

// DeleteTask deletes a task by ID.
func (c *Client) DeleteTask(ctx context.Context, id string) error {
	return c.doRequest(ctx, http.MethodDelete, "/api/tasks/"+url.PathEscape(id), nil, nil)
}

// GetTask returns a single task by ID.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
	var task Task
	err := c.doRequest(ctx, http.MethodGet, "/api/tasks/"+url.PathEscape(id), nil, &task)
	return &task, err
}

// HealthCheck checks if the API server is reachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	var result map[string]string
	return c.doRequest(ctx, http.MethodGet, "/api/health", nil, &result)
}