# Server
    - TUI first for CLI CRUD-based operations
    - Server is a separate command

# Commits/Branches

Branches must follow the tag/name pattern (e.g. feature/dynamic-mosaic).

Allowed branch tags:

- feature - new implementations, something new
- improvement - improvements that don't add behaviors or fixes (bug fix)
- chore - repository-only changes, cleanups, workflows, scripts

When working on a branch other than main or dev, use the "wip:" prefix on every commit, e.g. "wip: fix for customers page title"

If committing directly to main or dev (hotfixes or similar), use the "fix:" or "feat:" prefixes.

**Commit messages must always be in English** (title and body), regardless of the code or conversation language.

**Never** add plans (typically md files) to commits.
**Never** commit compiled binaries.

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