package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"charm.land/bubbles/v2/filepicker"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"agentloops/cli/client"
)

// message types for the edit wizard
type taskFetchedMsg struct {
	task *client.Task
	err  error
}

type taskUpdatedMsg struct {
	task *client.Task
	err  error
}

// EditWizardModel is the Bubble Tea model for the task edit wizard.
type EditWizardModel struct {
	// State
	currentStep int
	width       int
	height      int
	err         error
	Quitting    bool
	Submitted   bool

	// Task being edited
	taskID   string
	original *client.Task // original values for change detection

	// Inputs (pre-filled with current values)
	taskNameInput    textinput.Model
	initMessageInput textarea.Model
	filePicker       filepicker.Model
	intervalInput    textinput.Model

	// Dynamic selects
	agentList list.Model
	modelList list.Model
	modeList  list.Model

	// Parsed interval
	intervalSeconds int

	// Selections (pre-filled)
	selectedAgent string
	selectedModel string
	selectedMode  string

	// Data from API
	agents []client.AgentInfo
	models []string
	modes  []string

	// Loading state
	spinner      spinner.Model
	loading      bool
	loadingMsg   string
	agentsLoaded bool
	taskLoaded   bool
	modelsLoaded bool
	modesLoaded  bool

	// API client
	client *client.Client

	// Result
	UpdatedTask *client.Task
}

// NewEditWizardModel creates a new edit wizard model.
func NewEditWizardModel(taskID, serverURL string) EditWizardModel {
	// Task Name input
	taskName := textinput.New()
	taskName.Placeholder = "e.g., daily-code-review"
	taskName.Focus()
	taskName.CharLimit = 100
	taskName.SetWidth(60)

	// Init Message textarea
	initMsg := textarea.New()
	initMsg.Placeholder = "Enter the message to send to the agent..."
	initMsg.Focus()
	initMsg.SetWidth(60)
	initMsg.SetHeight(5)

	// File picker for working directory
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = false
	fp.ShowHidden = false
	fp.ShowSize = false
	fp.ShowPermissions = false
	fp.AutoHeight = false
	fp.SetHeight(5)
	if wd, err := os.Getwd(); err == nil {
		fp.CurrentDirectory = wd
	}

	// Interval input
	interval := textinput.New()
	interval.Placeholder = "60 (or 5m, 2h, 1d, 30s)"
	interval.SetValue("60")
	interval.SetWidth(30)
	interval.CharLimit = 20

	// Spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	// Agent list (will be populated later)
	agentList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	agentList.Title = "Select Agent"
	agentList.SetShowHelp(false)
	agentList.SetShowTitle(true)
	agentList.SetFilteringEnabled(false)
	agentList.SetShowStatusBar(false)

	// Model list
	modelList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	modelList.Title = "Select Model"
	modelList.SetShowHelp(false)
	modelList.SetShowTitle(true)
	modelList.SetFilteringEnabled(true)
	modelList.SetShowStatusBar(false)

	// Mode list
	modeList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	modeList.Title = "Select Mode"
	modeList.SetShowHelp(false)
	modeList.SetShowTitle(true)
	modeList.SetFilteringEnabled(false)
	modeList.SetShowStatusBar(false)

	return EditWizardModel{
		currentStep:      stepTaskName,
		taskID:           taskID,
		taskNameInput:    taskName,
		initMessageInput: initMsg,
		filePicker:       fp,
		intervalInput:    interval,
		spinner:          s,
		client:           client.NewClient(serverURL),
		agentList:        agentList,
		modelList:        modelList,
		modeList:         modeList,
	}
}

// --- Init ---

func (m EditWizardModel) Init() tea.Cmd {
	m.loading = true
	m.loadingMsg = "Loading task data..."
	return tea.Batch(
		m.spinner.Tick,
		m.fetchAgents(),
		m.fetchTask(),
		m.filePicker.Init(),
	)
}

func (m EditWizardModel) fetchAgents() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		agents, err := m.client.ListAgents(ctx)
		return agentsLoadedMsg{agents: agents, err: err}
	}
}

func (m EditWizardModel) fetchTask() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		task, err := m.client.GetTask(ctx, m.taskID)
		return taskFetchedMsg{task: task, err: err}
	}
}

func (m EditWizardModel) fetchModels(agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := m.client.GetAgentModels(ctx, agentID)
		return modelsLoadedMsg{models: models, err: err}
	}
}

func (m EditWizardModel) fetchModes(agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		modes, err := m.client.GetAgentModes(ctx, agentID)
		return modesLoadedMsg{modes: modes, err: err}
	}
}

func (m EditWizardModel) updateTask() tea.Cmd {
	return func() tea.Msg {
		var req client.UpdateTaskRequest

		// Only set non-nil fields for values that changed
		if m.taskNameInput.Value() != m.original.TaskName {
			v := m.taskNameInput.Value()
			req.TaskName = &v
		}
		if m.initMessageInput.Value() != m.original.InitMessage {
			v := m.initMessageInput.Value()
			req.InitMessage = &v
		}
		if m.selectedAgent != m.original.AgentRunner {
			v := m.selectedAgent
			req.AgentRunner = &v
		}
		if m.selectedModel != m.original.AgentModel {
			v := m.selectedModel
			req.AgentModel = &v
		}
		if m.selectedMode != m.original.AgentMode {
			v := m.selectedMode
			req.AgentMode = &v
		}
		if m.filePicker.CurrentDirectory != m.original.WorkDir {
			v := m.filePicker.CurrentDirectory
			req.WorkDir = &v
		}
		if m.intervalSeconds != m.original.IntervalSeconds {
			v := m.intervalSeconds
			req.IntervalSeconds = &v
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		task, err := m.client.UpdateTask(ctx, m.taskID, req)
		return taskUpdatedMsg{task: task, err: err}
	}
}

func (m EditWizardModel) maybeAdvanceFromAgent() tea.Cmd {
	if m.currentStep != stepAgent || !m.modelsLoaded || !m.modesLoaded {
		return nil
	}
	return func() tea.Msg { return agentCapabilitiesReadyMsg{} }
}

func (m EditWizardModel) maybeLoadCapabilities() tea.Cmd {
	if !m.agentsLoaded || !m.taskLoaded {
		return nil
	}

	// Pre-select agent in the list
	for i, a := range m.agents {
		if a.ID == m.selectedAgent {
			m.agentList.Select(i)
			break
		}
	}

	// Fetch models and modes for the pre-selected agent
	m.loading = true
	m.loadingMsg = fmt.Sprintf("Loading %s capabilities...", m.selectedAgent)
	m.modelsLoaded = false
	m.modesLoaded = false
	m.models = nil
	m.modes = nil
	return tea.Batch(
		m.fetchModels(m.selectedAgent),
		m.fetchModes(m.selectedAgent),
	)
}

// --- Update ---

func (m EditWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.agentList.SetWidth(min(50, msg.Width-10))
		m.agentList.SetHeight(min(15, msg.Height-10))
		m.modelList.SetWidth(min(50, msg.Width-10))
		m.modelList.SetHeight(min(15, msg.Height-10))
		m.modeList.SetWidth(min(50, msg.Width-10))
		m.modeList.SetHeight(min(15, msg.Height-10))
		m.filePicker.SetHeight(min(15, msg.Height-10))
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	// Forward filter-match results back to the currently active list.
	case list.FilterMatchesMsg:
		var cmd tea.Cmd
		switch m.currentStep {
		case stepModel:
			m.modelList, cmd = m.modelList.Update(msg)
			return m, cmd
		}
		return m, nil

	case agentsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.agents = msg.agents
		m.agentsLoaded = true
		// Populate agent list
		items := make([]list.Item, len(msg.agents))
		for i, a := range msg.agents {
			items[i] = agentListItem{info: a}
		}
		m.agentList.SetItems(items)
		return m, m.maybeLoadCapabilities()

	case taskFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.original = msg.task
		m.taskLoaded = true

		// Pre-fill all inputs with current task values
		m.taskNameInput.SetValue(msg.task.TaskName)
		m.initMessageInput.SetValue(msg.task.InitMessage)
		m.selectedAgent = msg.task.AgentRunner
		m.selectedModel = msg.task.AgentModel
		m.selectedMode = msg.task.AgentMode
		m.filePicker.CurrentDirectory = msg.task.WorkDir
		m.intervalInput.SetValue(formatDuration(msg.task.IntervalSeconds))
		m.intervalSeconds = msg.task.IntervalSeconds

		return m, m.maybeLoadCapabilities()

	case modelsLoadedMsg:
		m.modelsLoaded = true
		if msg.err != nil {
			m.models = []string{}
			m.modelList.SetItems([]list.Item{})
		} else {
			m.models = msg.models
			items := make([]list.Item, len(msg.models))
			for i, model := range msg.models {
				items[i] = stringListItem{value: model}
			}
			m.modelList.SetItems(items)
			// Pre-select the current model
			for i, model := range msg.models {
				if model == m.selectedModel {
					m.modelList.Select(i)
					break
				}
			}
		}
		if m.modelsLoaded && m.modesLoaded {
			m.loading = false
		}
		return m, m.maybeAdvanceFromAgent()

	case modesLoadedMsg:
		m.modesLoaded = true
		if msg.err != nil {
			m.modes = []string{}
			m.modeList.SetItems([]list.Item{})
		} else {
			m.modes = msg.modes
			items := make([]list.Item, len(msg.modes))
			for i, mode := range msg.modes {
				items[i] = stringListItem{value: mode}
			}
			m.modeList.SetItems(items)
			// Pre-select the current mode
			for i, mode := range msg.modes {
				if mode == m.selectedMode {
					m.modeList.Select(i)
					break
				}
			}
		}
		if m.modelsLoaded && m.modesLoaded {
			m.loading = false
		}
		return m, m.maybeAdvanceFromAgent()

	case taskUpdatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.UpdatedTask = msg.task
		m.Submitted = true
		m.currentStep = stepDone
		return m, tea.Quit

	case agentCapabilitiesReadyMsg:
		return m.nextStep()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// Forward unhandled messages to filepicker (readDirMsg, errorMsg, etc.)
	default:
		var cmd tea.Cmd
		m.filePicker, cmd = m.filePicker.Update(msg)
		return m, cmd
	}
}

func (m EditWizardModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Global keys
	switch msg.String() {
	case "ctrl+c":
		m.Quitting = true
		return m, tea.Quit
	}

	// Don't handle keys while loading
	if m.loading {
		return m, nil
	}

	// Step-specific key handling
	switch m.currentStep {
	case stepConfirm:
		return m.handleConfirmKey(msg)
	case stepAgent, stepModel, stepMode:
		return m.handleListKey(msg)
	default:
		return m.handleInputKey(msg)
	}
}

func (m EditWizardModel) handleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// For the textarea step, only Tab advances — Enter creates newlines
	if m.currentStep == stepInitMessage {
		switch msg.String() {
		case "tab":
			return m.nextStep()
		case "shift+tab", "up":
			return m.prevStep()
		case "esc":
			return m.prevStep()
		}
		var cmd tea.Cmd
		m.initMessageInput, cmd = m.initMessageInput.Update(msg)
		return m, cmd
	}

	// File picker step — forward keys to filepicker
	if m.currentStep == stepWorkDir {
		switch msg.String() {
		case "tab":
			return m.nextStep()
		case "shift+tab":
			return m.prevStep()
		}
		var cmd tea.Cmd
		m.filePicker, cmd = m.filePicker.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd

	switch msg.String() {
	case "tab", "down":
		return m.nextStep()
	case "shift+tab", "up":
		return m.prevStep()
	case "enter":
		if err := m.validateCurrentStep(); err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		return m.nextStep()
	case "esc":
		return m.prevStep()
	}

	// Update the active input
	switch m.currentStep {
	case stepTaskName:
		m.taskNameInput, cmd = m.taskNameInput.Update(msg)
	case stepInterval:
		m.intervalInput, cmd = m.intervalInput.Update(msg)
	}

	return m, cmd
}

func (m EditWizardModel) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.currentStep == stepModel && m.modelList.FilterState() == list.Filtering {
		switch msg.String() {
		case "tab":
			return m.nextStep()
		case "shift+tab":
			return m.prevStep()
		}
		var cmd tea.Cmd
		m.modelList, cmd = m.modelList.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "enter":
		return m.selectFromList()
	case "tab":
		return m.nextStep()
	case "shift+tab":
		return m.prevStep()
	}

	var cmd tea.Cmd
	switch m.currentStep {
	case stepAgent:
		m.agentList, cmd = m.agentList.Update(msg)
	case stepModel:
		m.modelList, cmd = m.modelList.Update(msg)
	case stepMode:
		m.modeList, cmd = m.modeList.Update(msg)
	}

	return m, cmd
}

func (m EditWizardModel) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.loading = true
		m.loadingMsg = "Updating task..."
		m.err = nil
		return m, m.updateTask()
	case "n", "N", "esc":
		return m.prevStep()
	}
	return m, nil
}

func (m EditWizardModel) selectFromList() (tea.Model, tea.Cmd) {
	switch m.currentStep {
	case stepAgent:
		if i, ok := m.agentList.SelectedItem().(agentListItem); ok {
			m.selectedAgent = i.info.ID
			m.err = nil
			m.modelsLoaded = false
			m.modesLoaded = false
			m.models = nil
			m.modes = nil
			m.loading = true
			m.loadingMsg = fmt.Sprintf("Loading %s capabilities...", i.info.Name)
			return m, tea.Batch(
				m.fetchModels(i.info.ID),
				m.fetchModes(i.info.ID),
			)
		}
	case stepModel:
		if i, ok := m.modelList.SelectedItem().(stringListItem); ok {
			m.selectedModel = i.value
			m.err = nil
			return m.nextStep()
		}
		m.selectedModel = ""
		m.err = nil
		return m.nextStep()
	case stepMode:
		if i, ok := m.modeList.SelectedItem().(stringListItem); ok {
			m.selectedMode = i.value
			m.err = nil
			return m.nextStep()
		}
		m.selectedMode = ""
		m.err = nil
		return m.nextStep()
	}
	return m, nil
}

func (m EditWizardModel) nextStep() (tea.Model, tea.Cmd) {
	if err := m.validateCurrentStep(); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil

	// Store parsed interval before advancing
	if m.currentStep == stepInterval {
		val := strings.TrimSpace(m.intervalInput.Value())
		n, _ := parseDuration(val)
		m.intervalSeconds = n
	}

	m.currentStep++
	if m.currentStep > stepConfirm {
		m.currentStep = stepConfirm
	}

	m.taskNameInput.Blur()
	m.initMessageInput.Blur()
	m.intervalInput.Blur()

	switch m.currentStep {
	case stepInitMessage:
		m.initMessageInput.Focus()
	case stepAgent:
		return m, nil
	case stepModel:
		if len(m.models) == 0 {
			m.selectedModel = ""
			m.currentStep = stepMode
			if len(m.modes) == 0 {
				m.selectedMode = ""
				m.currentStep = stepWorkDir
			}
		}
	case stepMode:
		if len(m.modes) == 0 {
			m.selectedMode = ""
			m.currentStep = stepWorkDir
		}
	case stepInterval:
		m.intervalInput.Focus()
	case stepConfirm:
	}

	return m, nil
}

func (m EditWizardModel) prevStep() (tea.Model, tea.Cmd) {
	m.currentStep--
	if m.currentStep < stepTaskName {
		m.currentStep = stepTaskName
	}

	m.taskNameInput.Blur()
	m.initMessageInput.Blur()
	m.intervalInput.Blur()

	switch m.currentStep {
	case stepTaskName:
		m.taskNameInput.Focus()
	case stepInitMessage:
		m.initMessageInput.Focus()
	case stepInterval:
		m.intervalInput.Focus()
	}

	return m, nil
}

func (m EditWizardModel) validateCurrentStep() error {
	switch m.currentStep {
	case stepTaskName:
		if strings.TrimSpace(m.taskNameInput.Value()) == "" {
			return fmt.Errorf("task name is required")
		}
	case stepInitMessage:
		if strings.TrimSpace(m.initMessageInput.Value()) == "" {
			return fmt.Errorf("init message is required")
		}
	case stepAgent:
		if m.selectedAgent == "" {
			return fmt.Errorf("agent is required")
		}
	case stepWorkDir:
		if strings.TrimSpace(m.filePicker.CurrentDirectory) == "" {
			return fmt.Errorf("working directory is required")
		}
	case stepInterval:
		val := strings.TrimSpace(m.intervalInput.Value())
		if val == "" {
			return fmt.Errorf("interval is required")
		}
		n, err := parseDuration(val)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}
		if n < 10 {
			return fmt.Errorf("interval must be at least 10 seconds (got %ds)", n)
		}
	}
	return nil
}

// --- View ---

func (m EditWizardModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("\n  Edit cancelled.\n\n")
	}

	if m.currentStep == stepDone {
		return tea.NewView(m.viewDone())
	}

	var b strings.Builder

	// Title
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("  ➜ Edit Task"))
	b.WriteString("\n\n")

	// Progress bar
	b.WriteString(m.viewProgress())
	b.WriteString("\n\n")

	// Loading state
	if m.loading {
		b.WriteString(fmt.Sprintf("  %s %s\n", m.spinner.View(), m.loadingMsg))
		v := tea.NewView(b.String())
		v.AltScreen = true
		return v
	}

	// Current step content
	b.WriteString(m.viewCurrentStep())

	// Error
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(formatError(m.err))
	}

	// Hints
	b.WriteString("\n\n")
	if m.currentStep == stepModel {
		b.WriteString(hintStyle.Render("  ↑↓: navigate  Enter: select  / filter  Esc: clear  Tab: next  Shift+Tab: back"))
	} else if m.currentStep == stepWorkDir {
		b.WriteString(hintStyle.Render("  j/k: navigate  Enter: open dir  h/Esc: parent  Tab: confirm  Shift+Tab: back  Ctrl+C: quit"))
	} else {
		b.WriteString(hintStyle.Render("  ↑↓: navigate  Enter: select  Tab: next  Shift+Tab/Esc: back  Ctrl+C: quit"))
	}
	b.WriteString("\n")

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

// viewProgress reuses the same progress bar from wizard.go (via stepNames constant).
func (m EditWizardModel) viewProgress() string {
	total := stepConfirm
	current := m.currentStep
	barWidth := 40
	filled := 0
	if total > 0 {
		filled = (current * barWidth) / total
	}

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "━"
		} else if i == filled {
			bar += "┼"
		} else {
			bar += "─"
		}
	}

	stepName := stepNames[current]
	return fmt.Sprintf("  Step %d/%d: %s\n  %s",
		current+1, total, stepName,
		lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Render(bar))
}

func (m EditWizardModel) viewCurrentStep() string {
	var b strings.Builder

	switch m.currentStep {
	case stepTaskName:
		b.WriteString(stepTitleStyle.Render("  What should we name this task?"))
		b.WriteString("\n")
		b.WriteString(stepDescStyle.Render("  A short, descriptive name for identification."))
		b.WriteString("\n\n")
		b.WriteString("  " + m.taskNameInput.View())
		b.WriteString("\n")

	case stepInitMessage:
		b.WriteString(stepTitleStyle.Render("  What message should the agent receive?"))
		b.WriteString("\n")
		b.WriteString(stepDescStyle.Render("  The initial instruction or prompt sent to the agent."))
		b.WriteString("\n\n")
		b.WriteString(m.initMessageInput.View())

	case stepAgent:
		b.WriteString(stepTitleStyle.Render("  Which agent should run this task?"))
		b.WriteString("\n")
		b.WriteString(stepDescStyle.Render("  Use ↑↓ to navigate, Enter to select."))
		b.WriteString("\n\n")
		b.WriteString("  " + m.agentList.View())
		b.WriteString("\n")

	case stepModel:
		b.WriteString(stepTitleStyle.Render("  Which model should the agent use?"))
		b.WriteString("\n")
		if len(m.models) == 0 {
			b.WriteString(stepDescStyle.Render("  No models available. Press Enter to skip."))
		} else {
			b.WriteString(stepDescStyle.Render("  ↑↓ to navigate, Enter to select, type to search."))
		}
		b.WriteString("\n\n")
		b.WriteString("  " + m.modelList.View())
		b.WriteString("\n")

	case stepMode:
		b.WriteString(stepTitleStyle.Render("  Which mode should the agent use?"))
		b.WriteString("\n")
		if len(m.modes) == 0 {
			b.WriteString(stepDescStyle.Render("  No modes available. Press Enter to skip."))
		} else {
			b.WriteString(stepDescStyle.Render("  Use ↑↓ to navigate, Enter to select."))
		}
		b.WriteString("\n\n")
		b.WriteString("  " + m.modeList.View())
		b.WriteString("\n")

	case stepWorkDir:
		b.WriteString(stepTitleStyle.Render("  Working directory?"))
		b.WriteString("\n")
		b.WriteString(stepDescStyle.Render("  Navigate to the working directory. Tab to confirm."))
		b.WriteString("\n\n")
		b.WriteString(m.filePicker.View())
		b.WriteString(hintStyle.Render("  " + m.filePicker.CurrentDirectory))
		b.WriteString("\n")

	case stepInterval:
		b.WriteString(stepTitleStyle.Render("  Execution interval?"))
		b.WriteString("\n")
		b.WriteString(stepDescStyle.Render("  How often the task should run. Use natural format: 30s, 5m, 2h, 1d, or plain seconds."))
		b.WriteString("\n\n")
		b.WriteString("  " + m.intervalInput.View())
		b.WriteString("\n")

	case stepConfirm:
		b.WriteString(m.viewConfirm())
	}

	return b.String()
}

func (m EditWizardModel) viewConfirm() string {
	var b strings.Builder

	b.WriteString(stepTitleStyle.Render("  Review & Confirm Changes"))
	b.WriteString("\n\n")

	maxKeyW := 18
	maxValW := 50

	// Helper to detect if a value changed from original
	changedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11")) // yellow

	rowWithChange := func(key, oldVal, newVal string) string {
		k := confirmKeyStyle.Render(fmt.Sprintf("  %-*s", maxKeyW, key))
		if oldVal != newVal {
			v := changedStyle.Render(fmt.Sprintf("%s → %s  (changed)", truncateStr(oldVal, maxValW/2), truncateStr(newVal, maxValW/2)))
			return k + v
		}
		v := confirmValStyle.Render(truncateStr(newVal, maxValW))
		return k + v
	}

	b.WriteString(rowWithChange("Name", m.original.TaskName, m.taskNameInput.Value()) + "\n")

	b.WriteString(confirmKeyStyle.Render(fmt.Sprintf("  %-*s", maxKeyW, "Init Message")) + "\n")
	initMsg := m.initMessageInput.Value()
	if initMsg != m.original.InitMessage {
		for _, line := range strings.Split(initMsg, "\n") {
			b.WriteString("     " + changedStyle.Render(line) + "\n")
		}
		b.WriteString("     " + changedStyle.Render("(changed)") + "\n")
	} else {
		for _, line := range strings.Split(initMsg, "\n") {
			b.WriteString("     " + confirmValStyle.Render(line) + "\n")
		}
	}

	b.WriteString(rowWithChange("Agent", m.original.AgentRunner, m.selectedAgent) + "\n")
	if m.selectedModel != "" || m.original.AgentModel != "" {
		b.WriteString(rowWithChange("Model", m.original.AgentModel, m.selectedModel) + "\n")
	}
	if m.selectedMode != "" || m.original.AgentMode != "" {
		b.WriteString(rowWithChange("Mode", m.original.AgentMode, m.selectedMode) + "\n")
	}
	b.WriteString(rowWithChange("WorkDir", m.original.WorkDir, m.filePicker.CurrentDirectory) + "\n")
	b.WriteString(rowWithChange("Interval", formatDuration(m.original.IntervalSeconds), formatDuration(m.intervalSeconds)) + "\n")

	b.WriteString("\n")
	b.WriteString(confirmKeyStyle.Render("  Press Y to confirm changes, N/Esc to go back"))
	b.WriteString("\n")

	return b.String()
}

func (m EditWizardModel) viewDone() string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(successStyle.Render("  ✓ Task updated successfully!"))
	b.WriteString("\n\n")

	if m.UpdatedTask != nil {
		b.WriteString(confirmKeyStyle.Render("  ID:    "))
		b.WriteString(confirmValStyle.Render(m.UpdatedTask.ID) + "\n")
		b.WriteString(confirmKeyStyle.Render("  Name:  "))
		b.WriteString(confirmValStyle.Render(m.UpdatedTask.TaskName) + "\n")
		b.WriteString(confirmKeyStyle.Render("  Agent: "))
		b.WriteString(confirmValStyle.Render(m.UpdatedTask.AgentRunner) + "\n")
		b.WriteString(confirmKeyStyle.Render("  Status:"))
		if m.UpdatedTask.Enabled {
			b.WriteString(successStyle.Render(" enabled") + "\n")
		} else {
			b.WriteString(errorStyle.Render(" disabled") + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}
