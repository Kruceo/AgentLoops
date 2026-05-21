package tasks

import "time"

// Task represents a scheduled agent task configuration.
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
}

// Run represents a single execution of a task.
type Run struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"taskId"`
	Output     string     `json:"output"`
	HasError   bool       `json:"hasError"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt"`
}
