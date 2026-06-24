package tui

import (
	"errors"
	"testing"

	tea "charm.land/bubbletea/v2"

	"agentloops/cli/client"
)

var errTest = errors.New("connection failed")

func TestTaskStart_StreamConnectedMsg_QuitsAndExportsChannel(t *testing.T) {
	ch := make(chan client.SSEEvent, 1)
	m := TaskStartModel{step: stepConnecting}

	mm, cmd := m.Update(streamConnectedMsg{eventCh: ch})
	m = mm.(TaskStartModel)

	if m.step != stepConnecting {
		t.Fatalf("expected stepConnecting after connect, got %d", m.step)
	}
	if m.eventCh == nil {
		t.Fatal("expected eventCh to be set")
	}
	if m.StreamCh == nil {
		t.Fatal("expected StreamCh to be exported")
	}
	if m.StreamCh != ch {
		t.Fatal("expected StreamCh to be the provided channel")
	}

	// Should return tea.Quit so the TUI exits and hands control back to the caller.
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("expected cmd to produce tea.QuitMsg, got %T", cmd())
	}
}

func TestTaskStart_StreamConnectedMsg_ErrorQuits(t *testing.T) {
	m := TaskStartModel{step: stepConnecting}
	mm, cmd := m.Update(streamConnectedMsg{err: errTest})
	m = mm.(TaskStartModel)

	if m.finalError != errTest.Error() {
		t.Fatalf("expected finalError %q, got %q", errTest.Error(), m.finalError)
	}
	if cmd == nil {
		t.Fatal("expected tea.Quit cmd")
	}
}

func TestTaskStart_RunStartTaskTUI_ErrorReturned(t *testing.T) {
	// An invalid URL won't prevent model creation; the real program would fail
	// on fetch. Since program.Run() needs a TTY in v2, we only test the helper
	// surface via direct model updates above.
	m := NewTaskStartModel("http://localhost:0")
	if m.step != stepTaskSelect {
		t.Fatalf("expected stepTaskSelect, got %d", m.step)
	}
	if m.StreamCh != nil {
		t.Fatal("expected StreamCh to be nil before connection")
	}
}
