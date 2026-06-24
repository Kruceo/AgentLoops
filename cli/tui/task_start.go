package tui

import (
	"context"
	"fmt"
	"log"
	"strings"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"agentloops/cli/client"
)

const (
	stepTaskSelect int = iota
	stepConnecting
)

// TaskStartModel is the Bubble Tea model for the TUI-first task start flow.
type TaskStartModel struct {
	// State
	step   int
	width  int
	height int
	err    error

	// Selection
	taskList list.Model

	// Streaming
	spinner    spinner.Model
	finalError string
	quitting   bool
	eventCh    <-chan client.SSEEvent
	StreamCh   <-chan client.SSEEvent // Export the channel for the caller after connection

	// API
	client *client.Client
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

// NewTaskStartModel creates a new TUI model for starting a task.
func NewTaskStartModel(serverURL string) TaskStartModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	taskList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	taskList.Title = "Select a Task to Start"
	taskList.SetShowHelp(false)
	taskList.SetShowTitle(true)
	taskList.SetFilteringEnabled(true)
	taskList.SetShowStatusBar(false)

	return TaskStartModel{
		step:     stepTaskSelect,
		spinner:  s,
		taskList: taskList,
		client:   client.NewClient(serverURL),
	}
}

// --- Messages ---

type tasksLoadedMsg struct {
	tasks []client.Task
	err   error
}

type streamConnectedMsg struct {
	eventCh <-chan client.SSEEvent
	err     error
}

// --- Init ---

func (m TaskStartModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchTasks(),
	)
}

func (m TaskStartModel) fetchTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := m.client.ListTasks(context.Background())
		return tasksLoadedMsg{tasks: tasks, err: err}
	}
}

func (m TaskStartModel) connectStream(taskID string) tea.Cmd {
	return func() tea.Msg {
		log.Printf("[DEBUG-TUI] connecting stream for task %s", taskID)
		eventCh, err := m.client.StartTaskStream(context.Background(), taskID)
		if err != nil {
			log.Printf("[DEBUG-TUI] stream connection failed: %v", err)
			return streamConnectedMsg{err: err}
		}
		log.Printf("[DEBUG-TUI] stream connected, eventCh=%p", eventCh)
		return streamConnectedMsg{eventCh: eventCh}
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
		}

		switch m.step {
		case stepTaskSelect:
			switch msg.String() {
			case "enter":
				return m.selectTask()
			case "esc":
				m.quitting = true
				return m, tea.Quit
			}
			var cmd tea.Cmd
			m.taskList, cmd = m.taskList.Update(msg)
			return m, cmd
		case stepConnecting:
			if msg.String() == "esc" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		return m, nil

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

	case streamConnectedMsg:
		if msg.err != nil {
			m.finalError = msg.err.Error()
			return m, tea.Quit
		}
		log.Printf("[DEBUG-TUI] streamConnectedMsg handler: connected, quitting TUI")
		m.eventCh = msg.eventCh
		m.StreamCh = msg.eventCh
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m TaskStartModel) selectTask() (tea.Model, tea.Cmd) {
	if i, ok := m.taskList.SelectedItem().(taskListItem); ok {
		m.step = stepConnecting
		m.finalError = ""
		m.err = nil
		return m, m.connectStream(i.task.ID)
	}
	return m, nil
}

// --- View ---

func (m TaskStartModel) View() tea.View {
	if m.quitting {
		return tea.NewView("\n  Cancelled.\n\n")
	}

	switch m.step {
	case stepTaskSelect:
		return m.viewTaskSelect()
	case stepConnecting:
		return m.viewConnecting()
	}

	return tea.NewView("")
}

func (m TaskStartModel) viewTaskSelect() tea.View {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("  ➜ Start Task"))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(errorStyle.Render("  ✗ Failed to load tasks: " + m.err.Error()))
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

func (m TaskStartModel) viewConnecting() tea.View {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("  ➜ Start Task"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s Connecting to server...\n", m.spinner.View()))
	b.WriteString("\n")
	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// RunStartTaskTUI launches the interactive task-start TUI and returns the connected SSE event channel.
func RunStartTaskTUI(serverURL string) (<-chan client.SSEEvent, error) {
	m := NewTaskStartModel(serverURL)
	program := tea.NewProgram(m)

	finalModel, runErr := program.Run()
	if runErr != nil {
		return nil, fmt.Errorf("TUI error: %w", runErr)
	}

	fm := finalModel.(TaskStartModel)
	if fm.finalError != "" {
		return nil, fmt.Errorf("%s", fm.finalError)
	}
	if fm.StreamCh == nil {
		return nil, fmt.Errorf("no stream channel (user cancelled)")
	}
	return fm.StreamCh, nil
}
