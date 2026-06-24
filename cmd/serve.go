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

	"github.com/spf13/cobra"

	"agentloops/api"
	"agentloops/core/agents"
	"agentloops/core/db"
	"agentloops/core/scheduler"
	"agentloops/core/tasks"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the Agent Loop Orchestrator server and scheduler",
	Long:  `Start the HTTP server and background scheduler for managing agent tasks.`,
	RunE:  runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)

	serveCmd.Flags().String("port", "", "Port to listen on (default: 8080 or PORT env var)")
	serveCmd.Flags().String("data-dir", "", "Data directory for database (default: ./data or DATA_DIR env var)")
}

func runServe(cmd *cobra.Command, args []string) error {
	// --- Configuration ---
	port, _ := cmd.Flags().GetString("port")
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "8080"
	}

	dataDir, _ := cmd.Flags().GetString("data-dir")
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "./data"
	}

	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// --- Ensure data directory exists ---
	if err := os.MkdirAll(dataDir, 0750); err != nil {
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
	errCh := make(chan error, 1)
	go func() {
		log.Printf("server starting on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Printf("received signal %v, shutting down...", sig)
	case err := <-errCh:
		log.Printf("http server error: %v, shutting down...", err)
	}

	sched.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http server shutdown: %w", err)
	}

	log.Println("server shut down gracefully")
	return nil
}