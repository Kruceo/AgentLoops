# AgentLoops

CLI/TUI tool that orchestrates automated agent tasks via [opencode](https://opencode.ai).

## Features

- **Task management** — add, list, and remove automated tasks
- **TUI wizard** — interactive Bubble Tea interface for easy configuration
- **Periodic scheduling** — run tasks on a recurring interval
- **REST API server mode** — manage tasks remotely via HTTP endpoints

## Prerequisites

- Go 1.25+
- [opencode](https://opencode.ai) CLI installed and available in `PATH`

## Installation

```bash
go build -o agentloops .
```

## Usage

Start the server:

```bash
agentloops serve
```

Manage tasks via the CLI:

```bash
agentloops task add       # Add a new task
agentloops task list      # List all tasks
agentloops task remove    # Remove a task
```
