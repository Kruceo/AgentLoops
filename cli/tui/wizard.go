package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
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

var (
	stepTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6")).
			MarginBottom(1)

	stepDescStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

	hintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("11"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	confirmKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6"))

	confirmValStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7"))
)

// Step identifiers
const (
	stepTaskName int = iota
	stepInitMessage
	stepAgent
	stepModel
	stepMode
	stepWorkDir
	stepInterval
	stepConfirm
	stepDone
)

// stepNames maps step indices to human-readable names.
var stepNames = map[int]string{
	stepTaskName:    "Task Name",
	stepInitMessage: "Init Message",
	stepAgent:       "Agent",
	stepModel:       "Model",
	stepMode:        "Mode",
	stepWorkDir:     "Working Directory",
	stepInterval:    "Interval",
	stepConfirm:     "Confirm",
}

// WizardModel is the Bubble Tea model for the task creation wizard.
type WizardModel struct {
	// State
	currentStep int
	width       int
	height      int
	err         error
	Quitting    bool
	Submitted   bool

	// Inputs
	taskNameInput    textinput.Model
	initMessageInput textarea.Model
	filePicker       filepicker.Model
	intervalInput    textinput.Model

	// Dynamic selects (built from list.Model)
	agentList list.Model
	modelList list.Model
	modeList  list.Model

	// Parsed interval in seconds (from natural language input)
	intervalSeconds int

	// Selections
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
	modelsLoaded bool
	modesLoaded  bool

	// API client
	client *client.Client

	// Result
	CreatedTask *client.Task
}

// NewWizardModel creates a new wizard model.
func NewWizardModel(serverURL string) WizardModel {
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
	// initMsg.Prompt = "│ "
	initMsg.SetHeight(5)
	// initMsg.DynamicHeight = true
	// initMsg.MaxHeight = 10
	// initMsg.ShowLineNumbers = false

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

	return WizardModel{
		currentStep:      stepTaskName,
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

// agentListItem wraps an AgentInfo for use in list.Model.
type agentListItem struct {
	info client.AgentInfo
}

func (i agentListItem) Title() string {
	status := "✓"
	if !i.info.Installed {
		status = "✗"
	}
	return fmt.Sprintf("%s %s (%s)", status, i.info.Name, i.info.ID)
}

func (i agentListItem) Description() string {
	if !i.info.Installed {
		return "NOT INSTALLED"
	}
	return ""
}

func (i agentListItem) FilterValue() string {
	return i.info.ID + " " + i.info.Name
}

// stringListItem wraps a string for use in list.Model.
type stringListItem struct {
	value string
}

func (i stringListItem) Title() string       { return i.value }
func (i stringListItem) Description() string { return "" }
func (i stringListItem) FilterValue() string { return i.value }

// --- Bubble Tea messages ---

type agentsLoadedMsg struct {
	agents []client.AgentInfo
	err    error
}

type modelsLoadedMsg struct {
	models []string
	err    error
}

type modesLoadedMsg struct {
	modes []string
	err   error
}

type taskCreatedMsg struct {
	task *client.Task
	err  error
}

// --- Init ---

func (m WizardModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchAgents(),
		m.filePicker.Init(),
	)
}

func (m WizardModel) fetchAgents() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		agents, err := m.client.ListAgents(ctx)
		return agentsLoadedMsg{agents: agents, err: err}
	}
}

func (m WizardModel) fetchModels(agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		models, err := m.client.GetAgentModels(ctx, agentID)
		return modelsLoadedMsg{models: models, err: err}
	}
}

func (m WizardModel) fetchModes(agentID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		modes, err := m.client.GetAgentModes(ctx, agentID)
		return modesLoadedMsg{modes: modes, err: err}
	}
}

func (m WizardModel) createTask() tea.Cmd {
	return func() tea.Msg {
		req := client.CreateTaskRequest{
			TaskName:        m.taskNameInput.Value(),
			InitMessage:     m.initMessageInput.Value(),
			AgentRunner:     m.selectedAgent,
			AgentModel:      m.selectedModel,
			AgentMode:       m.selectedMode,
			WorkDir:         m.filePicker.CurrentDirectory,
			IntervalSeconds: m.intervalSeconds,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		task, err := m.client.CreateTask(ctx, req)
		return taskCreatedMsg{task: task, err: err}
	}
}

// --- Update ---

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	// The list generates this message internally (from filterItems) and
	// needs to receive it back to populate filteredItems.
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
		// Populate agent list
		items := make([]list.Item, len(msg.agents))
		for i, a := range msg.agents {
			items[i] = agentListItem{info: a}
		}
		m.agentList.SetItems(items)
		return m, nil

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
		}
		// Only clear loading once both have arrived
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
		}
		// Only clear loading once both have arrived
		if m.modelsLoaded && m.modesLoaded {
			m.loading = false
		}
		return m, m.maybeAdvanceFromAgent()

	case taskCreatedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.CreatedTask = msg.task
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

func (m WizardModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m WizardModel) handleInputKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m WizardModel) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
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

func (m WizardModel) handleConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.loading = true
		m.loadingMsg = "Creating task..."
		m.err = nil
		return m, m.createTask()
	case "n", "N", "esc":
		return m.prevStep()
	}
	return m, nil
}

func (m WizardModel) selectFromList() (tea.Model, tea.Cmd) {
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

func (m WizardModel) maybeAdvanceFromAgent() tea.Cmd {
	if m.currentStep != stepAgent || !m.modelsLoaded || !m.modesLoaded {
		return nil
	}
	return func() tea.Msg { return agentCapabilitiesReadyMsg{} }
}

type agentCapabilitiesReadyMsg struct{}

func (m WizardModel) nextStep() (tea.Model, tea.Cmd) {
	if err := m.validateCurrentStep(); err != nil {
		m.err = err
		return m, nil
	}
	m.err = nil

	// Store parsed interval before advancing (validateCurrentStep can't
	// modify the model since it's a value receiver).
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

func (m WizardModel) prevStep() (tea.Model, tea.Cmd) {
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

func (m WizardModel) validateCurrentStep() error {
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

func (m WizardModel) View() tea.View {
	if m.Quitting {
		return tea.NewView("\n  Wizard cancelled.\n\n")
	}

	if m.currentStep == stepDone {
		return tea.NewView(m.viewDone())
	}

	var b strings.Builder

	// Title
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render("  ➜ Create New Task"))
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

func (m WizardModel) viewProgress() string {
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

func (m WizardModel) viewCurrentStep() string {
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

func (m WizardModel) viewConfirm() string {
	var b strings.Builder

	b.WriteString(stepTitleStyle.Render("  Review & Confirm"))
	b.WriteString("\n\n")

	maxKeyW := 18
	maxValW := 50

	row := func(key, val string) string {
		k := confirmKeyStyle.Render(fmt.Sprintf("  %-*s", maxKeyW, key))
		v := confirmValStyle.Render(truncateStr(val, maxValW))
		return k + v
	}

	b.WriteString(row("Name", m.taskNameInput.Value()) + "\n")
	b.WriteString(confirmKeyStyle.Render(fmt.Sprintf("  %-*s", maxKeyW, "Init Message")) + "\n")
	for _, line := range strings.Split(m.initMessageInput.Value(), "\n") {
		b.WriteString("     " + confirmValStyle.Render(line) + "\n")
	}
	b.WriteString(row("Agent", m.selectedAgent) + "\n")
	if m.selectedModel != "" {
		b.WriteString(row("Model", m.selectedModel) + "\n")
	}
	if m.selectedMode != "" {
		b.WriteString(row("Mode", m.selectedMode) + "\n")
	}
	b.WriteString(row("WorkDir", m.filePicker.CurrentDirectory) + "\n")
	b.WriteString(row("Interval", formatDuration(m.intervalSeconds)) + "\n")

	b.WriteString("\n")
	b.WriteString(confirmKeyStyle.Render("  Press Y to create, N/Esc to go back"))
	b.WriteString("\n")

	return b.String()
}

func (m WizardModel) viewDone() string {
	var b strings.Builder

	b.WriteString("\n\n")
	b.WriteString(successStyle.Render("  ✓ Task created successfully!"))
	b.WriteString("\n\n")

	if m.CreatedTask != nil {
		b.WriteString(confirmKeyStyle.Render("  ID:    "))
		b.WriteString(confirmValStyle.Render(m.CreatedTask.ID) + "\n")
		b.WriteString(confirmKeyStyle.Render("  Name:  "))
		b.WriteString(confirmValStyle.Render(m.CreatedTask.TaskName) + "\n")
		b.WriteString(confirmKeyStyle.Render("  Agent: "))
		b.WriteString(confirmValStyle.Render(m.CreatedTask.AgentRunner) + "\n")
		b.WriteString(confirmKeyStyle.Render("  Status:"))
		if m.CreatedTask.Enabled {
			b.WriteString(successStyle.Render(" enabled") + "\n")
		} else {
			b.WriteString(errorStyle.Render(" disabled") + "\n")
		}
	}

	b.WriteString("\n")
	return b.String()
}

// --- Helpers ---

// parseDuration converts natural language duration strings to seconds.
// Supported formats: "30s" (seconds), "5m" (minutes), "2h" (hours), "1d" (days),
// or a plain number (treated as seconds).
func parseDuration(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	last := s[len(s)-1]
	if last >= '0' && last <= '9' {
		// Plain number — treat as seconds
		n, err := strconv.Atoi(s)
		if err != nil {
			return 0, fmt.Errorf("expected a number or duration like 5m, 2h, 1d")
		}
		return n, nil
	}

	numStr := s[:len(s)-1]
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("expected a number followed by s/m/h/d")
	}

	switch last {
	case 's':
		return n, nil
	case 'm':
		return n * 60, nil
	case 'h':
		return n * 3600, nil
	case 'd':
		return n * 86400, nil
	default:
		return 0, fmt.Errorf("unknown suffix %q (use s, m, h, or d)", string(last))
	}
}

// formatDuration formats seconds into a human-readable string.
func formatDuration(secs int) string {
	switch {
	case secs >= 86400 && secs%86400 == 0:
		return fmt.Sprintf("%dd", secs/86400)
	case secs >= 3600 && secs%3600 == 0:
		return fmt.Sprintf("%dh", secs/3600)
	case secs >= 60 && secs%60 == 0:
		return fmt.Sprintf("%dm", secs/60)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-3]) + "..."
}

// formatError formats an error message with appropriate styling.
// API errors get a red ✗, validation/local errors get a yellow ⚠.
func formatError(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return errorStyle.Render("  ✗ " + err.Error())
	}
	return warnStyle.Render("  ⚠ " + err.Error())
}

// ErrWizardCancelled is returned when the user exits a wizard without
// submitting.
var ErrWizardCancelled = fmt.Errorf("wizard cancelled")

// RunCreateWizardTUI launches the interactive create-task wizard as a
// standalone Bubble Tea program. Returns the created task, nil if the user
// cancelled, or an error if the program failed.
func RunCreateWizardTUI(serverURL string) (*client.Task, error) {
	program := tea.NewProgram(NewWizardModel(serverURL))
	result, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("TUI error: %w", err)
	}
	wm, ok := result.(WizardModel)
	if !ok {
		return nil, fmt.Errorf("unexpected model type")
	}
	if wm.CreatedTask == nil && !wm.Submitted {
		return nil, nil
	}
	return wm.CreatedTask, nil
}
