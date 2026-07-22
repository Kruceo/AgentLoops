package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"agentloops/internal/client"
)

// DashboardAction represents what the dashboard wants to do after quitting.
type DashboardAction int

const (
	DashboardQuit       DashboardAction = iota
	DashboardCreateTask
	DashboardEditTask
)

// --- View state machine ---

type dashboardView int

const (
	viewDashboard dashboardView = iota
	viewDeleteConfirm
	viewStartResult
)

// --- Custom messages ---

type taskDeletedMsg struct {
	err error
}

type taskToggledMsg struct {
	task *client.Task
	err  error
}

type taskStartedMsg struct {
	runID  string
	status string
	err    error
}

// returnToDashboardMsg signals the dashboard to refresh its task list (used
// by the start-result overlay auto-return timer).
type returnToDashboardMsg struct{}

// --- Model ---

// TaskDashboardModel is the central interactive TUI for managing tasks.
// It uses a currentView state machine to switch between the main dashboard
// list and lightweight overlays (delete confirm, start result).
// Complex flows such as the create and edit wizards are launched as
// separate tea.Program instances from RunTaskDashboardTUI instead of being
// embedded and forwarded manually.
type TaskDashboardModel struct {
	serverURL string
	client    *client.Client

	// View state
	currentView dashboardView
	width       int
	height      int
	quitting    bool

	// Dashboard list
	taskList   list.Model
	tasks      []client.Task
	loading    bool
	loadingMsg string
	err        error
	spinner    spinner.Model

	// Delete confirmation
	deleteTargetID   string
	deleteTargetName string

	// Start result
	startResultMsg string

	// Action signal (chain pattern)
	pendingAction DashboardAction
	editTargetID  string
}

// NewTaskDashboardModel creates a new dashboard model.
func NewTaskDashboardModel(serverURL string) TaskDashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	taskList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	taskList.Title = "Tasks"
	taskList.SetShowHelp(false)
	taskList.SetShowTitle(true)
	taskList.SetFilteringEnabled(true)
	taskList.SetShowStatusBar(false)

	return TaskDashboardModel{
		serverURL:   serverURL,
		client:      client.NewClient(serverURL),
		currentView: viewDashboard,
		taskList:    taskList,
		spinner:     s,
		loading:     true,
		loadingMsg:  "Loading tasks...",
	}
}

// --- Init ---

func (m TaskDashboardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchTasks(),
	)
}

// --- Commands ---

func (m TaskDashboardModel) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		tasks, err := m.client.ListTasks(ctx)
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m TaskDashboardModel) toggleTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		// Get current state
		task, err := m.client.GetTask(ctx, taskID)
		if err != nil {
			return taskToggledMsg{err: err}
		}

		// Flip enabled
		enabled := !task.Enabled
		req := client.UpdateTaskRequest{
			Enabled: &enabled,
		}

		updatedTask, err := m.client.UpdateTask(ctx, taskID, req)
		if err != nil {
			return taskToggledMsg{err: err}
		}
		return taskToggledMsg{task: updatedTask}
	}
}

func (m TaskDashboardModel) startTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		resp, err := m.client.RunTaskNow(ctx, taskID)
		if err != nil {
			return taskStartedMsg{err: err}
		}
		return taskStartedMsg{runID: resp.ID, status: resp.Status}
	}
}

func (m TaskDashboardModel) deleteTask(taskID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := m.client.DeleteTask(ctx, taskID)
		return taskDeletedMsg{err: err}
	}
}

// --- Update ---

func (m TaskDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.taskList.SetWidth(min(60, msg.Width-10))
		m.taskList.SetHeight(min(20, msg.Height-10))
		return m, nil

	case tea.QuitMsg:
		m.quitting = true
		return m, tea.Quit

	case tea.KeyPressMsg:
		switch m.currentView {
		case viewDashboard:
			return m.handleDashboardKey(msg)

		case viewDeleteConfirm:
			return m.handleDeleteConfirmKey(msg)

		case viewStartResult:
			if msg.String() == "ctrl+c" || msg.String() == "q" {
				m.quitting = true
				return m, tea.Quit
			}
			// Any other key returns to dashboard and re-fetches
			m.currentView = viewDashboard
			return m, m.fetchTasks()
		}

	case list.FilterMatchesMsg:
		// Forward filter results to the task list.
		var cmd tea.Cmd
		m.taskList, cmd = m.taskList.Update(msg)
		return m, cmd

	case tasksLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.tasks = msg.tasks
		m.err = nil
		items := make([]list.Item, len(msg.tasks))
		for i, t := range msg.tasks {
			items[i] = taskListItem{task: t}
		}
		m.taskList.SetItems(items)
		return m, nil

	case taskToggledMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			return m, nil
		}
		m.loading = true
		m.loadingMsg = "Refreshing..."
		return m, m.fetchTasks()

	case taskDeletedMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			m.currentView = viewDashboard
			return m, nil
		}
		m.loading = true
		m.loadingMsg = "Refreshing..."
		m.currentView = viewDashboard
		return m, m.fetchTasks()

	case taskStartedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.startResultMsg = fmt.Sprintf("Task started ✓ (run: %s)", msg.runID)
		m.currentView = viewStartResult
		// Auto-return to dashboard after 2 seconds
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return returnToDashboardMsg{}
		})

	case returnToDashboardMsg:
		m.currentView = viewDashboard
		return m, m.fetchTasks()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// --- Key handlers ---

func (m TaskDashboardModel) handleDashboardKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "r":
		// Retry after error
		if m.err != nil {
			m.err = nil
			m.loading = true
			m.loadingMsg = "Loading tasks..."
			return m, m.fetchTasks()
		}
		return m, nil

	case "n":
		m.pendingAction = DashboardCreateTask
		m.quitting = true
		return m, tea.Quit

	case "e":
		if item, ok := m.taskList.SelectedItem().(taskListItem); ok {
			m.pendingAction = DashboardEditTask
			m.editTargetID = item.task.ID
			m.quitting = true
			return m, tea.Quit
		}
		// No task selected — no-op
		return m, nil

	case "d":
		if item, ok := m.taskList.SelectedItem().(taskListItem); ok {
			m.deleteTargetID = item.task.ID
			m.deleteTargetName = item.task.TaskName
			m.currentView = viewDeleteConfirm
			return m, nil
		}
		return m, nil

	case "t":
		if item, ok := m.taskList.SelectedItem().(taskListItem); ok {
			m.loading = true
			m.loadingMsg = "Toggling task..."
			return m, m.toggleTask(item.task.ID)
		}
		return m, nil

	case "s":
		if item, ok := m.taskList.SelectedItem().(taskListItem); ok {
			m.loading = true
			m.loadingMsg = "Starting task..."
			return m, m.startTask(item.task.ID)
		}
		return m, nil
	}

	// Forward to the task list for navigation / filtering
	var cmd tea.Cmd
	m.taskList, cmd = m.taskList.Update(msg)
	return m, cmd
}

func (m TaskDashboardModel) handleDeleteConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.loading = true
		m.loadingMsg = "Deleting task..."
		return m, m.deleteTask(m.deleteTargetID)
	case "n", "N", "esc":
		m.currentView = viewDashboard
		return m, nil
	}
	return m, nil
}

// --- View ---

func (m TaskDashboardModel) View() tea.View {
	if m.quitting {
		return tea.NewView("\n")
	}

	switch m.currentView {
	case viewDeleteConfirm:
		var b strings.Builder
		if m.loading {
			b.WriteString(fmt.Sprintf("\n  %s %s\n", m.spinner.View(), m.loadingMsg))
			b.WriteString("\n")
		} else {
			b.WriteString("\n")
			b.WriteString(warnStyle.Render("  Delete Task"))
			b.WriteString("\n\n")
			b.WriteString(fmt.Sprintf("  Name: %s\n", confirmValStyle.Render(m.deleteTargetName)))
			b.WriteString(fmt.Sprintf("  ID:   %s\n", confirmValStyle.Render(m.deleteTargetID)))
			b.WriteString("\n")
			b.WriteString(confirmKeyStyle.Render("  Delete this task? (y/N) "))
			b.WriteString("\n")
		}
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v

	case viewStartResult:
		var b strings.Builder
		b.WriteString("\n\n")
		b.WriteString(successStyle.Render("  " + m.startResultMsg))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("  Press any key to return to dashboard"))
		b.WriteString("\n")
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	// viewDashboard (default)
	var b strings.Builder
	m.renderDashboard(&b)
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (m TaskDashboardModel) renderDashboard(b *strings.Builder) {
	// Header
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("  ➜ Task Dashboard"))
	b.WriteString("\n\n")

	// Loading spinner
	if m.loading {
		b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), m.loadingMsg))
		return
	}

	// Error state
	if m.err != nil {
		b.WriteString(formatError(m.err))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("  Press r to retry, q to quit"))
		b.WriteString("\n")
		return
	}

	// Empty state
	if len(m.tasks) == 0 {
		b.WriteString(stepDescStyle.Render("  No tasks yet."))
		b.WriteString("\n")
		b.WriteString(hintStyle.Render("  Press n to create one."))
		b.WriteString("\n\n")
	} else {
		// Task list
		b.WriteString(m.taskList.View())
		b.WriteString("\n")
	}

	// Keybindings footer
	b.WriteString(hintStyle.Render("  n: new  e: edit  d: delete  t: toggle  s: start  /: filter  q: quit"))
	b.WriteString("\n")
}

// RunTaskDashboardTUI launches the main task dashboard TUI.
// Returns the action the dashboard wants to take after quitting, along with
// any associated data (e.g. task ID for edit) and an error.
func RunTaskDashboardTUI(serverURL string) (DashboardAction, string, error) {
	m := NewTaskDashboardModel(serverURL)
	program := tea.NewProgram(m)
	result, err := program.Run()
	if err != nil {
		return DashboardQuit, "", err
	}
	dm, ok := result.(TaskDashboardModel)
	if !ok {
		return DashboardQuit, "", fmt.Errorf("unexpected model type from dashboard TUI")
	}
	return dm.pendingAction, dm.editTargetID, nil
}
