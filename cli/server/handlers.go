package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"agentloops/core/agents"
	"agentloops/core/runs"
	"agentloops/core/scheduler"
	"agentloops/core/tasks"
)

// Handler holds all dependencies for API handlers.
type Handler struct {
	DB        *sql.DB
	Tasks     *tasks.TaskRepository
	Runs      *tasks.RunRepository
	Agents    agents.AgentManager
	Scheduler *scheduler.Scheduler
}

// getBroadcaster returns the broadcaster from the scheduler.
func (h *Handler) getBroadcaster() *runs.RunBroadcaster {
	return h.Scheduler.GetBroadcaster()
}

// RegisterRoutes registers all API routes on the given router.
func (h *Handler) RegisterRoutes(router *Router) {
	// Health
	router.GET("/api/health", h.HealthCheck)

	// Agents
	router.GET("/api/agents", h.ListAgents)
	router.GET("/api/agents/:id", h.GetAgent)
	router.GET("/api/agents/:id/models", h.GetAgentModels)
	router.GET("/api/agents/:id/modes", h.GetAgentModes)

	// Tasks
	router.GET("/api/tasks", h.ListTasks)
	router.POST("/api/tasks", h.CreateTask)
	router.GET("/api/tasks/:id", h.GetTask)
	router.PUT("/api/tasks/:id", h.UpdateTask)
	router.DELETE("/api/tasks/:id", h.DeleteTask)

	// Task runs
	router.GET("/api/tasks/:id/runs", h.ListTaskRuns)
	router.POST("/api/tasks/:id/run", h.RunTaskNow)

	// Runs
	router.GET("/api/runs", h.ListRuns)
	router.GET("/api/runs/:id", h.GetRun)
	router.GET("/api/runs/:id/stream", h.StreamRunOutput)
}

// HealthCheck returns a simple health status.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ListAgents returns all registered agents.
func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	agentList := h.Agents.List()
	writeJSON(w, http.StatusOK, agentList)
}

// GetAgent returns a single agent by ID.
func (h *Handler) GetAgent(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing agent id")
		return
	}

	agentList := h.Agents.List()
	for _, a := range agentList {
		if a.ID == id {
			writeJSON(w, http.StatusOK, a)
			return
		}
	}

	writeError(w, http.StatusNotFound, "agent not found")
}

// GetAgentModels returns available models for a specific agent.
func (h *Handler) GetAgentModels(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing agent id")
		return
	}

	agent, err := h.Agents.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found: "+id)
		return
	}
	models, err := agent.GetModels()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get models: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, models)
}

// GetAgentModes returns available modes for a specific agent.
func (h *Handler) GetAgentModes(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing agent id")
		return
	}

	agent, err := h.Agents.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "agent not found: "+id)
		return
	}
	modes, err := agent.GetModes()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get modes: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, modes)
}

// --- Task Handlers ---

// ListTasks returns all tasks, optionally filtered by enabled status.
func (h *Handler) ListTasks(w http.ResponseWriter, r *http.Request) {
	enabledOnly := false
	if r.URL.Query().Get("enabled") == "true" {
		enabledOnly = true
	}

	taskList, err := h.Tasks.List(enabledOnly)
	if err != nil {
		log.Printf("error listing tasks: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}

	// Populate last run status for each task
	for i := range taskList {
		if status, err := h.Runs.GetLatestRunStatus(taskList[i].ID); err == nil && status != "" {
			taskList[i].LastRunStatus = &status
		}
	}

	writeJSON(w, http.StatusOK, taskList)
}

// CreateTaskRequest represents the JSON body for creating a task.
type CreateTaskRequest struct {
	TaskName        string `json:"taskName"`
	InitMessage     string `json:"initMessage"`
	AgentRunner     string `json:"agentRunner"`
	AgentModel      string `json:"agentModel"`
	AgentMode       string `json:"agentMode"`
	WorkDir         string `json:"workDir"`
	Enabled         *bool  `json:"enabled"`
	CronExpr        string `json:"cronExpr"`
	IntervalSeconds int    `json:"intervalSeconds"`
}

// CreateTask creates a new task.
func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.TaskName == "" {
		writeError(w, http.StatusBadRequest, "taskName is required")
		return
	}
	if req.InitMessage == "" {
		writeError(w, http.StatusBadRequest, "initMessage is required")
		return
	}
	if req.AgentRunner == "" {
		writeError(w, http.StatusBadRequest, "agentRunner is required")
		return
	}
	if _, err := h.Agents.Get(req.AgentRunner); err != nil {
		writeError(w, http.StatusBadRequest, "unknown agent runner: "+req.AgentRunner)
		return
	}
	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 60
	}
	if req.IntervalSeconds < 10 {
		req.IntervalSeconds = 10
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	task := &tasks.Task{
		TaskName:        req.TaskName,
		InitMessage:     req.InitMessage,
		AgentRunner:     req.AgentRunner,
		AgentModel:      req.AgentModel,
		AgentMode:       req.AgentMode,
		WorkDir:         req.WorkDir,
		Enabled:         enabled,
		CronExpr:        req.CronExpr,
		IntervalSeconds: req.IntervalSeconds,
	}

	if err := h.Tasks.Create(task); err != nil {
		log.Printf("error creating task: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// GetTask returns a single task by ID.
func (h *Handler) GetTask(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	task, err := h.Tasks.GetByID(id)
	if err != nil {
		log.Printf("error getting task: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	// Populate last run status
	if status, err := h.Runs.GetLatestRunStatus(task.ID); err == nil && status != "" {
		task.LastRunStatus = &status
	}

	writeJSON(w, http.StatusOK, task)
}

// UpdateTaskRequest represents the JSON body for updating a task.
type UpdateTaskRequest struct {
	TaskName        *string `json:"taskName"`
	InitMessage     *string `json:"initMessage"`
	AgentRunner     *string `json:"agentRunner"`
	AgentModel      *string `json:"agentModel"`
	AgentMode       *string `json:"agentMode"`
	WorkDir         *string `json:"workDir"`
	Enabled         *bool   `json:"enabled"`
	CronExpr        *string `json:"cronExpr"`
	IntervalSeconds *int    `json:"intervalSeconds"`
}

// UpdateTask updates an existing task.
func (h *Handler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	task, err := h.Tasks.GetByID(id)
	if err != nil {
		log.Printf("error getting task: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB limit
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.TaskName != nil {
		task.TaskName = *req.TaskName
	}
	if req.InitMessage != nil {
		task.InitMessage = *req.InitMessage
	}
	if req.AgentRunner != nil {
		if _, err := h.Agents.Get(*req.AgentRunner); err != nil {
			writeError(w, http.StatusBadRequest, "unknown agent runner: "+*req.AgentRunner)
			return
		}
		task.AgentRunner = *req.AgentRunner
	}
	if req.AgentModel != nil {
		task.AgentModel = *req.AgentModel
	}
	if req.AgentMode != nil {
		task.AgentMode = *req.AgentMode
	}
	if req.WorkDir != nil {
		task.WorkDir = *req.WorkDir
	}
	if req.Enabled != nil {
		task.Enabled = *req.Enabled
	}
	if req.CronExpr != nil {
		task.CronExpr = *req.CronExpr
	}
	if req.IntervalSeconds != nil {
		task.IntervalSeconds = *req.IntervalSeconds
	}

	if err := h.Tasks.Update(task); err != nil {
		log.Printf("error updating task: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// DeleteTask deletes a task by ID.
func (h *Handler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	if err := h.Tasks.Delete(id); err != nil {
		if errors.Is(err, tasks.ErrTaskNotFound) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		log.Printf("error deleting task: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to delete task")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// --- Run Handlers ---

// ListTaskRuns returns runs for a specific task.
func (h *Handler) ListTaskRuns(w http.ResponseWriter, r *http.Request) {
	taskID := GetParam(r, "id")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	runList, err := h.Runs.ListByTaskID(taskID, limit)
	if err != nil {
		log.Printf("error listing runs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	writeJSON(w, http.StatusOK, runList)
}

// RunTaskNow triggers an immediate execution of a task.
func (h *Handler) RunTaskNow(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing task id")
		return
	}

	task, err := h.Tasks.GetByID(id)
	if err != nil {
		log.Printf("error getting task: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get task")
		return
	}
	if task == nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}

	if h.Scheduler.IsTaskRunning(id) {
		writeError(w, http.StatusConflict, "task is already running")
		return
	}

	runID := uuid.New().String()

	// Run task in the background with a 5-minute timeout.
	// We use a detached context because the HTTP request returns immediately.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	go func() {
		defer cancel()
		if _, err := h.Scheduler.RunTaskNow(ctx, task, runID); err != nil {
			log.Printf("error running task %s: %v", id, err)
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"id":     runID,
		"status": "running",
	})
}

// ListRuns returns all runs across all tasks.
func (h *Handler) ListRuns(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	runList, err := h.Runs.ListAll(limit)
	if err != nil {
		log.Printf("error listing all runs: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}

	writeJSON(w, http.StatusOK, runList)
}

// GetRun returns a single run by ID.
func (h *Handler) GetRun(w http.ResponseWriter, r *http.Request) {
	id := GetParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing run id")
		return
	}

	run, err := h.Runs.GetByID(id)
	if err != nil {
		log.Printf("error getting run: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	writeJSON(w, http.StatusOK, run)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("error encoding json response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// setSSEHeaders sets the required headers for Server-Sent Events.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

// StreamRunOutput streams run output as SSE (Server-Sent Events).
// GET /api/runs/:id/stream
func (h *Handler) StreamRunOutput(w http.ResponseWriter, r *http.Request) {
	runID := GetParam(r, "id")
	if runID == "" {
		writeError(w, http.StatusBadRequest, "missing run id")
		return
	}

	// Try to subscribe to a running stream first — no TOCTOU window.
	ch, subID, err := h.getBroadcaster().Subscribe(runID)
	if err != nil {
		// Run not active — fall back to DB for completed runs.
		h.streamFinishedRun(w, runID)
		return
	}
	defer h.getBroadcaster().Unsubscribe(runID, subID)

	// Check flusher support BEFORE setting SSE headers.
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	setSSEHeaders(w)

	seq := 0
	hadError := false
	doneSent := false

	// Ensure we always send a done event before the handler returns,
	// even if the context is cancelled mid-stream.
	sendDone := func() {
		if doneSent {
			return
		}
		doneSent = true
		status := "success"
		if hadError {
			status = "error"
		}
		doneData, _ := json.Marshal(map[string]string{"status": status})
		fmt.Fprintf(w, "event: done\ndata: %s\nid: %d\n\n", doneData, seq)
		flusher.Flush()
	}
	defer sendDone()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			// Client disconnected — send done before returning (defer handles it).
			return
		case chunk, ok := <-ch:
			if !ok {
				// Channel closed — run finished. sendDone (via defer) will
				// emit the final done event.
				return
			}

			// Track whether we received any error chunks.
			if chunk.Type == "error" {
				hadError = true
			}

			// If the broadcaster sent a "done" chunk (from FinishRun), send
			// our own done event with the accumulated status and return.
			if chunk.Type == "done" {
				sendDone()
				return
			}

			// Forward the chunk as an SSE event. JSON-encode the data string
			// so the client can parse it as a JSON value.
			dataJSON, _ := json.Marshal(chunk.Data)
			fmt.Fprintf(w, "event: %s\ndata: %s\nid: %d\n\n", chunk.Type, dataJSON, seq)
			seq++
			flusher.Flush()

		case <-ticker.C:
			// Keep-alive: send an SSE comment to prevent proxy timeouts.
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}

// streamFinishedRun serves a completed (or failed) run from the database
// as a single SSE response. Used as a fallback when the run is no longer
// active in the broadcaster.
func (h *Handler) streamFinishedRun(w http.ResponseWriter, runID string) {
	run, err := h.Runs.GetByID(runID)
	if err != nil {
		log.Printf("error getting run: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to get run")
		return
	}
	if run == nil {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	// Check flusher support BEFORE setting SSE headers.
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	setSSEHeaders(w)

	// Send the full output as a single output event.
	outputJSON, _ := json.Marshal(run.Output)
	fmt.Fprintf(w, "event: output\ndata: %s\nid: 0\n\n", outputJSON)
	flusher.Flush()

	// Send the done event with the run's final status.
	status := "success"
	if run.HasError {
		status = "error"
	}
	doneData, _ := json.Marshal(map[string]string{"status": status})
	fmt.Fprintf(w, "event: done\ndata: %s\nid: 1\n\n", doneData)
	flusher.Flush()
}
