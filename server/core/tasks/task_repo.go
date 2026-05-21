package tasks

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"agent-loop-orchestrator/server/core/db"
)

// ErrTaskNotFound is returned when a task is not found in the repository.
var ErrTaskNotFound = errors.New("task not found")

// TaskRepository provides CRUD operations for tasks backed by SQLite.
type TaskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new TaskRepository.
func NewTaskRepository(db *sql.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

// Create inserts a new task into the database.
func (r *TaskRepository) Create(task *Task) error {
	if task.ID == "" {
		task.ID = db.NewUUID()
	}

	enabled := 0
	if task.Enabled {
		enabled = 1
	}

	now := time.Now().UTC()
	task.CreatedAt = now
	task.UpdatedAt = now

	query := `INSERT INTO tasks (id, task_name, init_message, agent_runner, agent_model, agent_mode, work_dir, enabled, cron_expr, interval_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		task.ID, task.TaskName, task.InitMessage, task.AgentRunner,
		task.AgentModel, task.AgentMode, task.WorkDir, enabled, task.CronExpr,
		task.IntervalSeconds, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}

	return nil
}

// GetByID retrieves a task by its ID. Returns nil if not found.
func (r *TaskRepository) GetByID(id string) (*Task, error) {
	query := `SELECT id, task_name, init_message, agent_runner, agent_model, agent_mode, work_dir, enabled, cron_expr, interval_seconds, created_at, updated_at
		FROM tasks WHERE id = ?`

	row := r.db.QueryRow(query, id)
	task, err := scanTask(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}

	return task, nil
}

// List returns all tasks, optionally filtering by enabled status.
func (r *TaskRepository) List(enabledOnly bool) ([]Task, error) {
	var query string
	var args []interface{}

	if enabledOnly {
		query = `SELECT id, task_name, init_message, agent_runner, agent_model, agent_mode, work_dir, enabled, cron_expr, interval_seconds, created_at, updated_at
			FROM tasks WHERE enabled = 1 ORDER BY created_at DESC`
	} else {
		query = `SELECT id, task_name, init_message, agent_runner, agent_model, agent_mode, work_dir, enabled, cron_expr, interval_seconds, created_at, updated_at
			FROM tasks ORDER BY created_at DESC`
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		if err := scanTaskRows(rows, &task); err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}
		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return tasks, nil
}

// Update modifies an existing task's fields.
func (r *TaskRepository) Update(task *Task) error {
	enabled := 0
	if task.Enabled {
		enabled = 1
	}

	task.UpdatedAt = time.Now().UTC()

	query := `UPDATE tasks SET task_name = ?, init_message = ?, agent_runner = ?, agent_model = ?, agent_mode = ?, work_dir = ?,
		enabled = ?, cron_expr = ?, interval_seconds = ?, updated_at = ? WHERE id = ?`

	result, err := r.db.Exec(query,
		task.TaskName, task.InitMessage, task.AgentRunner,
		task.AgentModel, task.AgentMode, task.WorkDir, enabled, task.CronExpr,
		task.IntervalSeconds, task.UpdatedAt, task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update task rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, task.ID)
	}

	return nil
}

// Delete removes a task by its ID and all associated runs.
func (r *TaskRepository) Delete(id string) error {
	// Delete associated runs first (belt and suspenders with ON DELETE CASCADE)
	if _, err := r.db.Exec(`DELETE FROM runs WHERE task_id = ?`, id); err != nil {
		return fmt.Errorf("delete runs for task: %w", err)
	}

	query := `DELETE FROM tasks WHERE id = ?`
	result, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("%w: %s", ErrTaskNotFound, id)
	}

	return nil
}

// scanner abstracts both *sql.Row and *sql.Rows for scanning a single task.
type scanner interface {
	Scan(dest ...interface{}) error
}

func scanTask(s scanner) (*Task, error) {
	var task Task
	if err := scanTaskRows(s, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

func scanTaskRows(s scanner, task *Task) error {
	var enabled int
	var createdAt, updatedAt string

	err := s.Scan(
		&task.ID, &task.TaskName, &task.InitMessage, &task.AgentRunner,
		&task.AgentModel, &task.AgentMode, &task.WorkDir, &enabled, &task.CronExpr,
		&task.IntervalSeconds, &createdAt, &updatedAt,
	)
	if err != nil {
		return fmt.Errorf("scan task: %w", err)
	}

	task.Enabled = enabled == 1

	// Parse time fields
	// SQLite datetime format: "2006-01-02 15:04:05"
	timeFormats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}

	task.CreatedAt, _ = parseTime(createdAt, timeFormats)
	task.UpdatedAt, _ = parseTime(updatedAt, timeFormats)

	return nil
}

func parseTime(value string, formats []string) (time.Time, error) {
	for _, f := range formats {
		if t, err := time.Parse(f, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", value)
}
