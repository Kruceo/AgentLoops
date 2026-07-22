package tui

import (
	"testing"

	"agentloops/internal/client"
)

func TestTaskStart_NewModel(t *testing.T) {
	m := NewTaskStartModel("http://localhost:8080")
	if m.step != 0 {
		t.Errorf("expected step 0 (taskSelect), got %d", m.step)
	}
	if m.client == nil {
		t.Error("expected client to be set")
	}
}

func TestTaskStart_TaskListItem(t *testing.T) {
	task := client.Task{
		ID:              "test-id",
		TaskName:        "Test Task",
		AgentRunner:     "opencode",
		IntervalSeconds: 60,
		Enabled:         true,
	}
	item := taskListItem{task: task}

	if item.Title() != "● Test Task" {
		t.Errorf("expected title '● Test Task', got %q", item.Title())
	}
	if item.FilterValue() != "Test Task test-id" {
		t.Errorf("expected filter 'Test Task test-id', got %q", item.FilterValue())
	}

	// Test disabled task
	task.Enabled = false
	item2 := taskListItem{task: task}
	if item2.Title() != "○ Test Task" {
		t.Errorf("expected title '○ Test Task', got %q", item2.Title())
	}
}

func TestTaskStart_TaskListItemDescription(t *testing.T) {
	lastRun := "success"
	task := client.Task{
		ID:              "test-id",
		TaskName:        "Test Task",
		AgentRunner:     "opencode",
		IntervalSeconds: 120,
		LastRunStatus:   &lastRun,
	}
	item := taskListItem{task: task}

	desc := item.Description()
	if desc == "" {
		t.Fatal("expected non-empty description")
	}
}

func TestTaskStart_TaskListItemFilterValue(t *testing.T) {
	task := client.Task{
		ID:       "abc-123",
		TaskName: "My Task",
	}
	item := taskListItem{task: task}

	fv := item.FilterValue()
	if fv != "My Task abc-123" {
		t.Fatalf("expected 'My Task abc-123', got %q", fv)
	}
}