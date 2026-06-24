package tui

import (
	"testing"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

// expandBatch expands a BatchMsg from the bubbletea runtime:
// BatchMsg is []tea.Cmd — execute each, then route each resulting Msg back
// through the model's Update.
func expandBatch(m WizardModel, cmd tea.Cmd) WizardModel {
	if cmd == nil {
		return m
	}
	msg := cmd()
	if msg == nil {
		return m
	}
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		mm, _ := m.Update(msg)
		return mm.(WizardModel)
	}
	for _, subCmd := range batch {
		if subCmd == nil {
			continue
		}
		subMsg := subCmd()
		if subMsg == nil {
			continue
		}
		mm, _ := m.Update(subMsg)
		m = mm.(WizardModel)
	}
	return m
}

func TestModelListFilter_ActuallyFilters(t *testing.T) {
	m := NewWizardModel("http://localhost:9999")

	items := []list.Item{
		stringListItem{value: "gpt-4o"},
		stringListItem{value: "gpt-4o-mini"},
		stringListItem{value: "claude-opus-4"},
		stringListItem{value: "claude-sonnet-4"},
		stringListItem{value: "gemini-2.0-flash"},
	}
	m.modelList.SetItems(items)
	m.currentStep = stepModel
	m.modelsLoaded = true
	m.modesLoaded = true

	// Step 1: Press '/' to activate filter
	mm, cmd := m.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	m = mm.(WizardModel)
	m = expandBatch(m, cmd) // textinput.Blink
	if m.modelList.FilterState() != list.Filtering {
		t.Fatalf("expected Filtering state after '/', got %v", m.modelList.FilterState())
	}

	// Step 2: Type 'g' — should trigger filterItems
	mm, cmd = m.Update(tea.KeyPressMsg{Text: "g", Code: 'g'})
	m = mm.(WizardModel)
	m = expandBatch(m, cmd) // expands to route FilterMatchesMsg back to list

	// Step 3: Verify the list is now filtered
	visible := m.modelList.VisibleItems()
	if len(visible) == 0 {
		t.Fatal("expected visible items after filter")
	}
	if len(visible) >= len(items) {
		t.Fatalf("filter didn't reduce items: expected < %d visible, got %d", len(items), len(visible))
	}

	expected := map[string]bool{"gpt-4o": true, "gpt-4o-mini": true, "gemini-2.0-flash": true}
	for _, vi := range visible {
		val := vi.(stringListItem).value
		if !expected[val] {
			t.Errorf("unexpected visible item %q (should not match filter 'g')", val)
		}
		delete(expected, val)
	}
	if len(expected) > 0 {
		t.Errorf("missing expected items: %v", expected)
	}
}
