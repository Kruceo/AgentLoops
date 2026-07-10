package tasks

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"agentloops/core/db"
)

// RunRepository provides CRUD operations for runs backed by SQLite.
type RunRepository struct {
	db *sql.DB
}

// NewRunRepository creates a new RunRepository.
func NewRunRepository(db *sql.DB) *RunRepository {
	return &RunRepository{db: db}
}

// Create inserts a new run record into the database.
func (r *RunRepository) Create(run *Run) error {
	if run.ID == "" {
		run.ID = db.NewUUID()
	}

	hasError := 0
	if run.HasError {
		hasError = 1
	}

	var finishedAt interface{}
	if run.FinishedAt != nil {
		finishedAt = run.FinishedAt.UTC().Format("2006-01-02 15:04:05")
	}

	query := `INSERT INTO runs (id, task_id, output, has_error, started_at, finished_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query,
		run.ID, run.TaskID, run.Output, hasError,
		run.StartedAt.UTC().Format("2006-01-02 15:04:05"),
		finishedAt,
	)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}

	return nil
}

// Update modifies an existing run record in the database.
func (r *RunRepository) Update(run *Run) error {
	hasError := 0
	if run.HasError {
		hasError = 1
	}

	var finishedAt interface{}
	if run.FinishedAt != nil {
		finishedAt = run.FinishedAt.UTC().Format("2006-01-02 15:04:05")
	}

	query := `UPDATE runs
		SET output = ?, has_error = ?, finished_at = ?
		WHERE id = ?`

	_, err := r.db.Exec(query, run.Output, hasError, finishedAt, run.ID)
	if err != nil {
		return fmt.Errorf("update run: %w", err)
	}

	return nil
}

// GetByID retrieves a run by its ID.
func (r *RunRepository) GetByID(id string) (*Run, error) {
	query := `SELECT id, task_id, output, has_error, started_at, finished_at
		FROM runs WHERE id = ?`

	row := r.db.QueryRow(query, id)
	run, err := scanRun(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get run by id: %w", err)
	}

	return run, nil
}

// ListByTaskID returns runs for a specific task, ordered by started_at DESC, with an optional limit.
func (r *RunRepository) ListByTaskID(taskID string, limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, task_id, output, has_error, started_at, finished_at
		FROM runs WHERE task_id = ? ORDER BY started_at DESC LIMIT ?`

	rows, err := r.db.Query(query, taskID, limit)
	if err != nil {
		return nil, fmt.Errorf("list runs by task id: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows)
}

// ListAll returns all runs ordered by started_at DESC, with an optional limit.
func (r *RunRepository) ListAll(limit int) ([]Run, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, task_id, output, has_error, started_at, finished_at
		FROM runs ORDER BY started_at DESC LIMIT ?`

	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("list all runs: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows)
}

// GetLatestRunStatus returns the status of the latest run for a task.
// Returns "" if no runs exist, "running" if unfinished, "success" or "error".
func (r *RunRepository) GetLatestRunStatus(taskID string) (string, error) {
	query := `SELECT has_error, finished_at FROM runs WHERE task_id = ? ORDER BY started_at DESC LIMIT 1`
	var hasError int
	var finishedAt sql.NullString
	err := r.db.QueryRow(query, taskID).Scan(&hasError, &finishedAt)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get latest run status: %w", err)
	}
	if !finishedAt.Valid {
		return "running", nil
	}
	if hasError == 1 {
		return "error", nil
	}
	return "success", nil
}

func scanRun(s scanner) (*Run, error) {
	var run Run
	if err := scanRunRow(s, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

func scanRuns(rows *sql.Rows) ([]Run, error) {
	var runs []Run
	for rows.Next() {
		var run Run
		if err := scanRunRow(rows, &run); err != nil {
			return nil, fmt.Errorf("scan run row: %w", err)
		}
		runs = append(runs, run)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return runs, nil
}

func scanRunRow(s scanner, run *Run) error {
	var hasError int
	var startedAt, finishedAt sql.NullString

	err := s.Scan(&run.ID, &run.TaskID, &run.Output, &hasError, &startedAt, &finishedAt)
	if err != nil {
		return fmt.Errorf("scan run: %w", err)
	}

	run.HasError = hasError == 1

	timeFormats := []string{
		"2006-01-02 15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z",
	}

	if startedAt.Valid {
		run.StartedAt, _ = parseTime(startedAt.String, timeFormats)
	}

	if finishedAt.Valid {
		t, err := parseTime(finishedAt.String, timeFormats)
		if err == nil {
			run.FinishedAt = &t
		}
	}

	// Compute status from has_error and finished_at
	if run.FinishedAt == nil {
		run.Status = "running"
	} else if run.HasError {
		run.Status = "error"
	} else {
		run.Status = "success"
	}

	return nil
}
