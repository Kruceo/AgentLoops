# Server
    - TUI first for CLI.
    - Server is a separate command

# Commits/Branches

Branches must follow the tag/name pattern (e.g. feature/dynamic-mosaic).

Allowed branch tags:

- **feature** — genuinely new capabilities that didn't exist before; the user can now do something they couldn't in the previous version.
- **improvement** — smaller changes that don't introduce new behavior: bug fixes, code improvements, refactors that don't break the current minor (0.x.0) experience, or behavioral tweaks that don't change the outcome.
- **chore** — repository-level concerns, not application code per se: CI/CD, documentation, scripts, build config.

When working on a branch other than main or dev, use the "wip:" prefix on every commit, e.g. "wip: fix for customers page title"

If committing directly to main or dev (hotfixes or similar), use the "fix:" or "feat:" prefixes.

**Commit messages must always be in English** (title and body), regardless of the code or conversation language.

- **Never** add plans (typically md files) to commits.
- **Never** commit compiled binaries.
- **Never** commit test shell scripts.

# Purpose

The idea is for the user to have the daemon running in the background and use the CLI (e.g. "task add" or "task delete") to manage the app.
That said, it's not CLI-only — we also have an HTTP API, leaving room for anyone who wants to create a Web UI or similar.

# Internal vs Core

[internal](./internal): Responsible for the presentation layer — what the user interacts with, e.g. CLI, TUI, and HTTP API.
[core](./core/): Responsible for the application domain.

# Core/agents

- Must follow [agents.go](core/agents/agent.go)
- Prefer insecure mode
- If a task field is not available (e.g. Hermes doesn't have agent modes like opencode's build/plan), assume a single option called "default" that has no practical effect — just to keep the pattern consistent.   

# The app

The app is split in 2 sides: Client and Server, all in one binary app.

Client always need to use the server to run anything. The server is the core.

# Error Handling

All errors follow the pattern documented in [docs/errors.md](docs/errors.md). Use `core/errors` for error types, `handleError` in API handlers, and `formatError` in TUI.

# TUI Architecture

## Prefer separate programs over embedded sub-views

Complex, self-contained flows (e.g. the create-task wizard, the edit-task wizard) should be launched as separate `tea.Program` instances instead of being embedded and manually forwarded inside another Bubble Tea model.

- Reuses the exact same wizard used by dedicated CLI commands (`task add`, `task edit`).
- Avoids lifecycle bugs such as missing `tea.WindowSizeMsg` and manual list sizing.
- Keeps the parent model (e.g. dashboard) small and focused on its own state.

## Chain pattern for sub-programs

When a parent TUI needs to launch a sub-program, use the **chain pattern**: the parent quits with an action signal, the orchestration layer runs the sub-program, then re-launches the parent. This ensures only one `tea.Program` is active at any time, avoiding stdin conflicts.

**Never** launch a `tea.Program` from inside a `tea.Cmd` returned by another program's `Update`. Two programs reading from `/dev/tty` simultaneously will cause a race condition where keystrokes go to the wrong program.

Example:
```
func runDashboard(...) error {
    for {
        action, data, err := RunDashboardTUI(...)  // dashboard quits with action
        switch action {
        case ActionQuit:
            return nil
        case ActionCreate:
            RunCreateWizardTUI(...)  // sub-program runs while dashboard is stopped
            // loop → re-launch dashboard
        }
    }
}
```

Rule of thumb:
- **Separate program + chain pattern**: multi-step wizards, forms, or any flow that has its own model and `Init`/`Update`/`View` lifecycle.
- **Embedded overlay**: lightweight, single-screen interactions such as delete confirmation or a transient result message.