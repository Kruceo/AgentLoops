package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxResponseSize = 10 << 20 // 10 MB

// APIError represents a structured error returned by the API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string { return e.Message }

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
	ID              string    `json:"id"`
	TaskName        string    `json:"taskName"`
	InitMessage     string    `json:"initMessage"`
	AgentRunner     string    `json:"agentRunner"`
	AgentModel      string    `json:"agentModel"`
	AgentMode       string    `json:"agentMode"`
	WorkDir         string    `json:"workDir"`
	Enabled         bool      `json:"enabled"`
	CronExpr        string    `json:"cronExpr"`
	IntervalSeconds int       `json:"intervalSeconds"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
	LastRunStatus   *string   `json:"lastRunStatus,omitempty"`
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
	defer func() { _ = resp.Body.Close() }()

	limiter := io.LimitReader(resp.Body, maxResponseSize+1)
	respBody, err := io.ReadAll(limiter)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}
	if len(respBody) > maxResponseSize {
		return fmt.Errorf("response too large (%d bytes)", len(respBody))
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		if jsonErr := json.Unmarshal(respBody, &errResp); jsonErr == nil && errResp.Code != "" {
			return &APIError{
				StatusCode: resp.StatusCode,
				Code:       errResp.Code,
				Message:    errResp.Message,
			}
		}
		// Fallback for non-structured error responses
		return &APIError{
			StatusCode: resp.StatusCode,
			Code:       "UNKNOWN",
			Message:    fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody)),
		}
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
	return nil, &APIError{Code: "AGENT_NOT_FOUND", Message: "agent not found: " + agentID}
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

// UpdateTaskRequest represents the request body for updating a task.
// Only non-nil fields will be updated.
type UpdateTaskRequest struct {
	TaskName        *string `json:"taskName,omitempty"`
	InitMessage     *string `json:"initMessage,omitempty"`
	AgentRunner     *string `json:"agentRunner,omitempty"`
	AgentModel      *string `json:"agentModel,omitempty"`
	AgentMode       *string `json:"agentMode,omitempty"`
	WorkDir         *string `json:"workDir,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	CronExpr        *string `json:"cronExpr,omitempty"`
	IntervalSeconds *int    `json:"intervalSeconds,omitempty"`
}

// UpdateTask updates an existing task by ID. Only non-nil fields in the request are updated.
func (c *Client) UpdateTask(ctx context.Context, id string, req UpdateTaskRequest) (*Task, error) {
	var task Task
	err := c.doRequest(ctx, http.MethodPut, "/api/tasks/"+url.PathEscape(id), req, &task)
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

// Run represents a task run from the API.
type Run struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"taskId"`
	Output     string     `json:"output"`
	HasError   bool       `json:"hasError"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
	Status     string     `json:"status,omitempty"`
}

// RunTaskNowResponse represents the response from POST /api/tasks/:id/run.
type RunTaskNowResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// RunTaskNow triggers an immediate execution of a task.
// Returns the run ID and status (typically "running" with a 202 response).
func (c *Client) RunTaskNow(ctx context.Context, taskID string) (*RunTaskNowResponse, error) {
	var resp RunTaskNowResponse
	err := c.doRequest(ctx, http.MethodPost, "/api/tasks/"+url.PathEscape(taskID)+"/run", nil, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetRun retrieves a run by its ID.
func (c *Client) GetRun(ctx context.Context, runID string) (*Run, error) {
	var run Run
	err := c.doRequest(ctx, http.MethodGet, "/api/runs/"+url.PathEscape(runID), nil, &run)
	if err != nil {
		return nil, err
	}
	return &run, nil
}

// StreamEvent represents an SSE event from the runs stream endpoint.
type StreamEvent struct {
	Type string // "output", "error", or "done"
	Data string // The raw data payload (JSON-encoded string for "output"/"error", JSON object for "done")
}

// StreamRunOutput connects to the SSE stream for a run and returns a channel of events.
// The channel is closed when the stream ends (after receiving "done" or on error).
// The caller should drain the channel to avoid blocking.
func (c *Client) StreamRunOutput(ctx context.Context, runID string) (<-chan StreamEvent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/runs/"+url.PathEscape(runID)+"/stream", nil)
	if err != nil {
		return nil, fmt.Errorf("create stream request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	// Use a separate HTTP client with no timeout for SSE (long-lived connection)
	sseClient := &http.Client{}
	resp, err := sseClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do stream request: %w", err)
	}

	ch := make(chan StreamEvent)

		go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()

		scanner := bufio.NewScanner(resp.Body)
		var currentType, currentData string

		for scanner.Scan() {
			line := scanner.Text()

			switch {
			case strings.HasPrefix(line, "event: "):
				currentType = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				currentData = strings.TrimPrefix(line, "data: ")
			case line == "":
				// Empty line dispatches the accumulated event
				if currentType != "" || currentData != "" {
					ch <- StreamEvent{Type: currentType, Data: currentData}
					if currentType == "done" {
						return
					}
					currentType = ""
					currentData = ""
				}
			}
			// Lines starting with ":" (comments/keep-alive) and "id: " are silently skipped
		}

		// If the scanner stopped due to a real error (not context cancellation),
		// send an error event before closing the channel.
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			ch <- StreamEvent{
				Type: "error",
				Data: fmt.Sprintf("stream read error: %v", err),
			}
			return
		}

		// If the stream ended without a "done" event (e.g., network glitch or
		// context cancellation), fall back to querying the REST API for the
		// run's final status. This makes the client resilient to stream
		// interruptions.
		if ctx.Err() == nil {
			run, err := c.GetRun(ctx, runID)
			if err == nil && run != nil {
				status := "success"
				if run.HasError {
					status = "error"
				}
				ch <- StreamEvent{Type: "done", Data: fmt.Sprintf(`{"status":"%s"}`, status)}
				return
			}
			// API fallback also failed — report the error.
			ch <- StreamEvent{
				Type: "error",
				Data: fmt.Sprintf("stream ended unexpectedly and API fallback failed: %v", err),
			}
		}
	}()

	return ch, nil
}
