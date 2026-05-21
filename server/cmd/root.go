package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"agent-loop-orchestrator/server/api"
	"agent-loop-orchestrator/server/core/agents"
	"agent-loop-orchestrator/server/core/db"
	"agent-loop-orchestrator/server/core/scheduler"
	"agent-loop-orchestrator/server/core/tasks"
)

// Execute is the main entry point for the server application.
// It wires all dependencies together and starts the HTTP server and scheduler.
func Execute() error {
	// --- Configuration ---
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// --- Ensure data directory exists ---
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "agentloops.db")

	// --- Database ---
	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	// --- Repositories ---
	taskRepo := tasks.NewTaskRepository(database)
	runRepo := tasks.NewRunRepository(database)

	// --- Agents ---
	opencodeAgent := agents.NewOpencodeAgent()
	agentMap := map[string]agents.Agent{
		"opencode": opencodeAgent,
	}
	agentMgr := agents.NewDefaultAgentManager(agentMap)

	// --- Scheduler ---
	sched := scheduler.New(taskRepo, runRepo, agentMgr, workDir)

	// --- API Handler ---
	handler := &api.Handler{
		DB:        database,
		Tasks:     taskRepo,
		Runs:      runRepo,
		Agents:    agentMgr,
		Scheduler: sched,
	}

	// --- HTTP Server ---
	addr := fmt.Sprintf(":%s", port)
	httpServer := api.NewServer(addr, handler)

	// --- Start Scheduler ---
	go sched.Start()

	// --- Start HTTP Server ---
	go func() {
		log.Printf("server starting on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Printf("received signal %v, shutting down...", sig)

	sched.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	log.Println("server shut down gracefully")
	return nil
}
