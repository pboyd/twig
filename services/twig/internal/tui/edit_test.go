package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	taskv1 "github.com/pboyd/twig/services/twig/gen/task/v1"
)

func makeTask(id int64, name string) *taskv1.Task {
	return &taskv1.Task{Id: id, Name: name}
}

// TestEditFormPreFill verifies that NewEditForm pre-fills all fields from the task.
func TestEditFormPreFill(t *testing.T) {
	task := &taskv1.Task{
		Id:          42,
		Name:        "buy milk",
		Description: "2% please",
		Estimate:    2,
	}
	f := NewEditForm(task, 0)

	if f.name.Value() != "buy milk" {
		t.Errorf("name: want %q, got %q", "buy milk", f.name.Value())
	}
	if f.description.Value() != "2% please" {
		t.Errorf("description: want %q, got %q", "2% please", f.description.Value())
	}
	if f.pomodoroEstimate.Value() != "2" {
		t.Errorf("estimate: want %q, got %q", "2", f.pomodoroEstimate.Value())
	}
	if f.taskID == nil || *f.taskID != 42 {
		t.Errorf("taskID: want 42, got %v", f.taskID)
	}
}

// TestNewSubtaskFormSetsParentID verifies parentID is set and fields are blank.
func TestNewSubtaskFormSetsParentID(t *testing.T) {
	f := NewSubtaskForm(7, 3)

	if f.parentID == nil || *f.parentID != 7 {
		t.Errorf("parentID: want 7, got %v", f.parentID)
	}
	if f.taskID != nil {
		t.Errorf("taskID should be nil for subtask form")
	}
	if f.name.Value() != "" {
		t.Errorf("name should be empty, got %q", f.name.Value())
	}
	if f.originalCursor != 3 {
		t.Errorf("originalCursor: want 3, got %d", f.originalCursor)
	}
}

// TestNewRootFormIsBlank verifies all fields start empty for a root form.
func TestNewRootFormIsBlank(t *testing.T) {
	f := NewRootForm(1)

	if f.taskID != nil {
		t.Errorf("taskID should be nil for root form")
	}
	if f.parentID != nil {
		t.Errorf("parentID should be nil for root form")
	}
	if f.name.Value() != "" {
		t.Errorf("name should be empty")
	}
}

// TestEditFormTabCyclesFocus verifies Tab cycles through all focus positions.
func TestEditFormTabCyclesFocus(t *testing.T) {
	f := NewRootForm(0)
	keys := DefaultKeyMap()

	if f.focusIndex != focusName {
		t.Fatalf("initial focus: want focusName(%d), got %d", focusName, f.focusIndex)
	}

	for i := 1; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab}, keys)
		if f.focusIndex != i {
			t.Errorf("after %d tab(s): want %d, got %d", i, i, f.focusIndex)
		}
	}

	// One more Tab wraps to 0.
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab}, keys)
	if f.focusIndex != focusName {
		t.Errorf("after wrap: want focusName(%d), got %d", focusName, f.focusIndex)
	}
}

// TestEditFormShiftTabGoesBack verifies Shift-Tab moves focus backwards.
func TestEditFormShiftTabGoesBack(t *testing.T) {
	f := NewRootForm(0)
	keys := DefaultKeyMap()

	// Shift-Tab from focusName wraps to focusCancel.
	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyShiftTab}, keys)
	if f.focusIndex != focusCancel {
		t.Errorf("shift-tab from 0: want focusCancel(%d), got %d", focusCancel, f.focusIndex)
	}
}

// TestEditFormEscCancels verifies Esc dispatches editCancelledMsg.
func TestEditFormEscCancels(t *testing.T) {
	f := NewRootForm(5)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEscape}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Esc, got nil")
	}
	msg := cmd()
	cancelled, ok := msg.(editCancelledMsg)
	if !ok {
		t.Fatalf("expected editCancelledMsg, got %T", msg)
	}
	if cancelled.originalCursor != 5 {
		t.Errorf("originalCursor: want 5, got %d", cancelled.originalCursor)
	}
}

// TestEditFormCtrlSSaves verifies Ctrl+S dispatches editSavedMsg.
func TestEditFormCtrlSSaves(t *testing.T) {
	task := makeTask(10, "walk the dog")
	f := NewEditForm(task, 2)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyCtrlS}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S, got nil")
	}
	msg := cmd()
	saved, ok := msg.(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", msg)
	}
	if saved.name != "walk the dog" {
		t.Errorf("name: want %q, got %q", "walk the dog", saved.name)
	}
	if saved.taskID == nil || *saved.taskID != 10 {
		t.Errorf("taskID: want 10, got %v", saved.taskID)
	}
	if saved.originalCursor != 2 {
		t.Errorf("originalCursor: want 2, got %d", saved.originalCursor)
	}
}

// TestEditFormEscFromAnyField verifies Esc works from any focused field.
func TestEditFormEscFromAnyField(t *testing.T) {
	keys := DefaultKeyMap()

	for fi := 0; fi < focusCount; fi++ {
		f := NewRootForm(0)
		f.focusIndex = fi

		_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEscape}, keys)
		if cmd == nil {
			t.Errorf("focusIndex=%d: expected Cmd from Esc", fi)
			continue
		}
		msg := cmd()
		if _, ok := msg.(editCancelledMsg); !ok {
			t.Errorf("focusIndex=%d: expected editCancelledMsg, got %T", fi, msg)
		}
	}
}

// TestEditFormSaveButtonEnter verifies Enter on Save button saves.
func TestEditFormSaveButtonEnter(t *testing.T) {
	f := NewRootForm(0)
	f.focusIndex = focusSave
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Enter on Save")
	}
	if _, ok := cmd().(editSavedMsg); !ok {
		t.Errorf("expected editSavedMsg")
	}
}

// TestEditFormCancelButtonEnter verifies Enter on Cancel button cancels.
func TestEditFormCancelButtonEnter(t *testing.T) {
	f := NewRootForm(0)
	f.focusIndex = focusCancel
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyMsg{Type: tea.KeyEnter}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Enter on Cancel")
	}
	if _, ok := cmd().(editCancelledMsg); !ok {
		t.Errorf("expected editCancelledMsg")
	}
}
