package tui

import (
	"context"
	"fmt"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"agentloops/internal/client"
)

// TaskStartModel is the Bubble Tea model for the task selection TUI.
// It lets the user pick a task and returns the task ID.
type TaskStartModel struct {
	step   int
	width  int
	height int
	err    error

	taskList list.Model
	client   *client.Client

	// Customizable titles for reuse across different task selection flows.
	listTitle   string
	headerTitle string

	// Set when the user selects a task.
	SelectedTaskID string
	quitting       bool
}

// taskListItem wraps a client.Task for use in list.Model.
type taskListItem struct {
	task client.Task
}

func (i taskListItem) Title() string {
	status := "●"
	if !i.task.Enabled {
		status = "○"
	}
	return fmt.Sprintf("%s %s", status, i.task.TaskName)
}

func (i taskListItem) Description() string {
	desc := fmt.Sprintf("Agent: %s | Interval: %s", i.task.AgentRunner, formatDuration(i.task.IntervalSeconds))
	if i.task.LastRunStatus != nil {
		desc += fmt.Sprintf(" | Last: %s", *i.task.LastRunStatus)
	}
	return desc
}

func (i taskListItem) FilterValue() string {
	return i.task.TaskName + " " + i.task.ID
}

// NewTaskSelectModel creates a generic task selection TUI model with
// customizable list title and header prompt.
func NewTaskSelectModel(serverURL string, listTitle string, headerTitle string) TaskStartModel {
	taskList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	taskList.Title = listTitle
	taskList.SetShowHelp(false)
	taskList.SetShowTitle(true)
	taskList.SetFilteringEnabled(true)
	taskList.SetShowStatusBar(false)

	return TaskStartModel{
		step:        0,
		taskList:    taskList,
		client:      client.NewClient(serverURL),
		listTitle:   listTitle,
		headerTitle: headerTitle,
	}
}

// NewTaskStartModel creates a new TUI model for selecting a task to start.
func NewTaskStartModel(serverURL string) TaskStartModel {
	return NewTaskSelectModel(serverURL, "Select a Task to Start", "➜ Start Task")
}

// --- Messages ---

type tasksLoadedMsg struct {
	tasks []client.Task
	err   error
}

// --- Init ---

func (m TaskStartModel) Init() tea.Cmd {
	return m.fetchTasks()
}

func (m TaskStartModel) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListTasks(context.Background())
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

// --- Update ---

func (m TaskStartModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.taskList.SetWidth(min(60, msg.Width-10))
		m.taskList.SetHeight(min(20, msg.Height-10))
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.taskList.SelectedItem().(taskListItem); ok {
				m.SelectedTaskID = item.task.ID
				return m, tea.Quit
			}
		}
		var cmd tea.Cmd
		m.taskList, cmd = m.taskList.Update(msg)
		return m, cmd

	case list.FilterMatchesMsg:
		var cmd tea.Cmd
		m.taskList, cmd = m.taskList.Update(msg)
		return m, cmd

	case tasksLoadedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		items := make([]list.Item, len(msg.tasks))
		for i, t := range msg.tasks {
			items[i] = taskListItem{task: t}
		}
		m.taskList.SetItems(items)
		return m, nil
	}

	return m, nil
}

// --- View ---

func (m TaskStartModel) View() tea.View {
	if m.quitting {
		return tea.NewView("\n  Cancelled.\n\n")
	}

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("  " + m.headerTitle))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(formatError(m.err))
		b.WriteString("\n\n")
		b.WriteString(hintStyle.Render("  Press Ctrl+C to quit"))
		b.WriteString("\n")

		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	b.WriteString(stepDescStyle.Render("  Select a task and press Enter to start it."))
	b.WriteString("\n\n")
	b.WriteString(m.taskList.View())
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("  ↑↓: navigate  Enter: start  /: filter  Esc: quit  Ctrl+C: quit"))
	b.WriteString("\n")

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// runTaskSelectProgram is a shared helper that creates and runs a Bubble Tea
// program for task selection using the given model, returning the selected task ID.
func runTaskSelectProgram(m TaskStartModel) (string, error) {
	program := tea.NewProgram(m)

	finalModel, runErr := program.Run()
	if runErr != nil {
		return "", fmt.Errorf("TUI error: %w", runErr)
	}

	fm := finalModel.(TaskStartModel)
	if fm.err != nil {
		return "", fm.err
	}
	return fm.SelectedTaskID, nil
}

// RunTaskSelectTUI launches the interactive task-selection TUI with a
// custom list title and header prompt. Returns the selected task ID,
// or an empty string if the user cancelled.
func RunTaskSelectTUI(serverURL string, listTitle string, headerTitle string) (string, error) {
	m := NewTaskSelectModel(serverURL, listTitle, headerTitle)
	return runTaskSelectProgram(m)
}

// RunStartTaskTUI launches the interactive task-selection TUI for starting a task.
// Returns the selected task ID, or an empty string if the user cancelled.
func RunStartTaskTUI(serverURL string) (string, error) {
	return RunTaskSelectTUI(serverURL, "Select a Task to Start", "➜ Start Task")
}