package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-loop-orchestrator/server/core/agents"
	"agent-loop-orchestrator/server/core/tasks"
)

// Scheduler periodically runs enabled tasks based on their configured interval.
type Scheduler struct {
	taskRepo       *tasks.TaskRepository
	runRepo        *tasks.RunRepository
	agentMgr       agents.AgentManager
	workDir        string
	stopCh         chan struct{}
	running        bool
	mu             sync.Mutex
	lastRun        map[string]time.Time
	taskRunning    map[string]bool // tracks currently executing task IDs
	cleanupCounter int
}

// New creates a new Scheduler.
func New(taskRepo *tasks.TaskRepository, runRepo *tasks.RunRepository, agentMgr agents.AgentManager, workDir string) *Scheduler {
	return &Scheduler{
		taskRepo:    taskRepo,
		runRepo:     runRepo,
		agentMgr:    agentMgr,
		workDir:     workDir,
		stopCh:      make(chan struct{}),
		lastRun:     make(map[string]time.Time),
		taskRunning: make(map[string]bool),
	}
}

// Start begins the scheduler loop. It checks enabled tasks every 10 seconds.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("scheduler: starting")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.executeDueTasks()
		case <-s.stopCh:
			log.Println("scheduler: stopped")
			return
		}
	}
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	s.running = false
	close(s.stopCh)
	s.stopCh = make(chan struct{}) // reset for potential restart
}

// RunTaskNow runs a specific task immediately and returns the Run record.
// Returns an error if the task is already running.
func (s *Scheduler) RunTaskNow(ctx context.Context, task *tasks.Task) (*tasks.Run, error) {
	if !s.tryAcquireTask(task.ID) {
		return nil, fmt.Errorf("task %q is already running", task.TaskName)
	}
	defer s.releaseTask(task.ID)

	agent, err := s.agentMgr.Get(task.AgentRunner)
	if err != nil {
		return nil, fmt.Errorf("get agent %s: %w", task.AgentRunner, err)
	}

	run := &tasks.Run{
		TaskID:    task.ID,
		StartedAt: time.Now().UTC(),
	}

	log.Printf("scheduler: running task %q (id=%s) with agent %s", task.TaskName, task.ID, task.AgentRunner)

	workDir := task.WorkDir
	if workDir == "" {
		workDir = s.workDir
	}
	output, err := agent.Run(ctx, workDir, task.InitMessage, task.AgentModel, task.AgentMode)
	if err != nil {
		run.HasError = true
		run.Output = fmt.Sprintf("error: %v", err)
		log.Printf("scheduler: task %q failed: %v", task.TaskName, err)
	} else {
		run.Output = output
		log.Printf("scheduler: task %q completed successfully", task.TaskName)
	}

	now := time.Now().UTC()
	run.FinishedAt = &now

	// Persist the run
	if err := s.runRepo.Create(run); err != nil {
		return nil, fmt.Errorf("save run: %w", err)
	}

	// Update last run time
	s.mu.Lock()
	s.lastRun[task.ID] = now
	s.mu.Unlock()

	return run, nil
}

// tryAcquireTask atomically checks and sets the running flag for a task.
// Returns true if the task was NOT running and has been acquired.
func (s *Scheduler) tryAcquireTask(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.taskRunning[taskID] {
		return false
	}
	s.taskRunning[taskID] = true
	return true
}

// releaseTask clears the running flag for a task.
func (s *Scheduler) releaseTask(taskID string) {
	s.mu.Lock()
	delete(s.taskRunning, taskID)
	s.mu.Unlock()
}

// isTaskRunning checks whether a task is currently executing (lock-free read under mutex).
func (s *Scheduler) isTaskRunning(taskID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.taskRunning[taskID]
}

// executeDueTasks checks all enabled tasks and runs any that are due.
func (s *Scheduler) executeDueTasks() {
	taskList, err := s.taskRepo.List(true) // enabled only
	if err != nil {
		log.Printf("scheduler: error listing tasks: %v", err)
		return
	}

	now := time.Now().UTC()

	s.mu.Lock()
	lastRunCopy := make(map[string]time.Time, len(s.lastRun))
	for k, v := range s.lastRun {
		lastRunCopy[k] = v
	}
	s.mu.Unlock()

	for i := range taskList {
		t := taskList[i]

		lastTime, exists := lastRunCopy[t.ID]
		if !exists {
			// Never run before, check if interval has passed since task created
			lastTime = t.CreatedAt
		}

		interval := time.Duration(t.IntervalSeconds) * time.Second
		if now.Sub(lastTime) >= interval {
			// Skip if already running (RunTaskNow will also guard at acquire level)
			if s.isTaskRunning(t.ID) {
				log.Printf("scheduler: task %q is still running, skipping this tick", t.TaskName)
				continue
			}
			// Run in background
			go func(task tasks.Task) {
				// Create a context with timeout (5 minutes max per run)
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
				defer cancel()

				if _, err := s.RunTaskNow(ctx, &task); err != nil {
					log.Printf("scheduler: error running task %q: %v", task.TaskName, err)
				}
			}(t)
		}
	}

	// Periodic cleanup of stale lastRun entries
	s.cleanupCounter++
	if s.cleanupCounter%10 == 0 && len(s.lastRun) > len(taskList)*3 {
		s.mu.Lock()
		valid := make(map[string]time.Time, len(taskList))
		for _, t := range taskList {
			if v, ok := s.lastRun[t.ID]; ok {
				valid[t.ID] = v
			}
		}
		s.lastRun = valid
		s.mu.Unlock()
	}
}
