# AgentLoops

> Schedule, manage, and run AI agent tasks — from the terminal or over HTTP.

**TUI-first CLI tool and daemon for orchestrating scheduled AI agent tasks.**

Written in Go. Single binary — client and server in one.

---

## Features

- **TUI dashboard** — interactive terminal UI for managing tasks: list, filter, create, edit, delete, toggle enable/disable, trigger runs. Keyboard-driven.
- **Task wizards** — step-by-step interactive forms for creating/editing tasks (agent selection, model, mode, interval, init message, file picker).
- **HTTP API** — full REST API + SSE streaming. All CLI commands go through the API.
- **Periodic scheduling** — tasks run on configurable intervals.
- **Multi-agent** — pluggable agent interface. Add new AI agents by implementing one interface.
- **Single binary** — client + server in one binary (`agentloops serve` for daemon, `agentloops task` for TUI management).

---

## Prerequisites

- Go 1.25+
- An agent CLI installed (currently [opencode](https://opencode.ai))

---

## Build

```bash
go build -o agentloops .
```

---

## Usage

### Start the daemon (API + scheduler)

```bash
agentloops serve
```

### Launch the TUI dashboard

```bash
agentloops task
```

### Quick commands (non-interactive)

```bash
agentloops task add       # Interactive wizard
agentloops task list      # List all tasks
agentloops task edit      # Edit a task
agentloops task remove    # Remove a task
agentloops task disable   # Disable a task
agentloops task enable    # Enable a task
agentloops task start     # Trigger a run immediately
```

---

## API Routes

| Method   | Path                     | Description        |
| -------- | ------------------------ | ------------------ |
| `GET`    | `/api/health`            | Health check       |
| `GET`    | `/api/agents`            | List agents        |
| `GET`    | `/api/agents/:id`        | Get agent details  |
| `GET`    | `/api/agents/:id/models` | List agent models  |
| `GET`    | `/api/agents/:id/modes`  | List agent modes   |
| `GET`    | `/api/tasks`             | List tasks         |
| `POST`   | `/api/tasks`             | Create a task      |
| `GET`    | `/api/tasks/:id`         | Get a task         |
| `PUT`    | `/api/tasks/:id`         | Update a task      |
| `DELETE` | `/api/tasks/:id`         | Delete a task      |
| `GET`    | `/api/tasks/:id/runs`    | List runs for task |
| `POST`   | `/api/tasks/:id/run`     | Trigger a run      |
| `GET`    | `/api/runs`              | List all runs      |
| `GET`    | `/api/runs/:id`          | Get a run          |
| `GET`    | `/api/runs/:id/stream`   | SSE stream         |

---

## Project Structure

```
├── cmd/          ← cobra command wiring (root, serve, task, task add, task edit, etc.)
├── internal/     ← implementation: client (HTTP client), server (HTTP API router + handlers), tui (Bubble Tea dashboard + wizards)
├── core/         ← domain: agents (interface + opencode impl), tasks, runs, scheduler, db (SQLite), errors
├── main.go       ← entry point
```

---

## Agent Interface

To add a new AI agent (e.g. Claude Code, Hermes, etc.), implement the `Agent` interface from `core/agents/agent.go`:

```go
type Agent interface {
    Name() string
    Run(ctx context.Context, workDir string, initMessage string, model string, mode string, chunks chan<- OutputChunk) (string, error)
    GetModels() ([]string, error)
    GetModes() ([]string, error)
    IsInstalled() bool
}
```

Register your implementation with `DefaultAgentManager`. Example skeleton:

```go
type MyAgent struct{}

func (a *MyAgent) Name() string            { return "my-agent" }
func (a *MyAgent) IsInstalled() bool       { /* check binary in PATH */ }
func (a *MyAgent) GetModels() ([]string, error) { /* list models */ }
func (a *MyAgent) GetModes() ([]string, error)  { /* list modes */ }
func (a *MyAgent) Run(ctx context.Context, workDir, initMessage, model, mode string, chunks chan<- OutputChunk) (string, error) {
    // execute agent, stream output to chunks channel, return final output
}
```

Currently only `OpencodeAgent` is implemented.
