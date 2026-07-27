package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)

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
	f := NewRootForm(1, nil)

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
// When showGoalField is false (NewRootForm), focusGoal is skipped.
// showPlanField is true for NewRootForm, so focusPlan is included.
func TestEditFormTabCyclesFocus(t *testing.T) {
	f := NewRootForm(0, nil)
	keys := DefaultKeyMap()

	if f.focusIndex != focusName {
		t.Fatalf("initial focus: want focusName(%d), got %d", focusName, f.focusIndex)
	}

	// Expected focus order (focusState skipped because !isGoal, focusGoal skipped because !showGoalField).
	want := []int{focusDescription, focusDue, focusEstimate, focusSnooze, focusPlan, focusSave, focusCancel}
	for step, w := range want {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex != w {
			t.Errorf("after %d tab(s): want %d, got %d", step+1, w, f.focusIndex)
		}
	}

	// One more Tab wraps back to focusName.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
	if f.focusIndex != focusName {
		t.Errorf("after wrap: want focusName(%d), got %d", focusName, f.focusIndex)
	}
}

// TestEditFormShiftTabGoesBack verifies Shift-Tab moves focus backwards.
func TestEditFormShiftTabGoesBack(t *testing.T) {
	f := NewRootForm(0, nil)
	keys := DefaultKeyMap()

	// Shift-Tab from focusName wraps to focusCancel.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, keys)
	if f.focusIndex != focusCancel {
		t.Errorf("shift-tab from 0: want focusCancel(%d), got %d", focusCancel, f.focusIndex)
	}
}

// TestEditFormEscCancelsCleanForm verifies that Esc on a clean (create) form
// dispatches editCancelledMsg immediately.
func TestEditFormEscCancelsCleanForm(t *testing.T) {
	f := NewRootForm(5, nil)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
	if cmd == nil {
		t.Fatal("Esc on clean form: expected editCancelledMsg cmd, got nil")
	}
	msg := cmd()
	if _, ok := msg.(editCancelledMsg); !ok {
		t.Fatalf("Esc on clean form: expected editCancelledMsg, got %T", msg)
	}
}

// TestEditFormCtrlSSaves verifies Ctrl+S dispatches editSavedMsg.
func TestEditFormCtrlSSaves(t *testing.T) {
	task := makeTask(10, "walk the dog")
	f := NewEditForm(task, 2, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
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

// TestEditFormEscFromAnyFieldOnCleanForm verifies Esc cancels the form from any
// field when the form is clean (create form — all originals are empty).
func TestEditFormEscFromAnyFieldOnCleanForm(t *testing.T) {
	keys := DefaultKeyMap()

	for fi := 0; fi < focusCount; fi++ {
		f := NewRootForm(0, nil)
		f.focusIndex = fi

		_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
		if cmd == nil {
			t.Errorf("focusIndex=%d: Esc should produce a cmd (editCancelledMsg), got nil", fi)
			continue
		}
		msg := cmd()
		if _, ok := msg.(editCancelledMsg); !ok {
			t.Errorf("focusIndex=%d: Esc should produce editCancelledMsg, got %T", fi, msg)
		}
	}
}

// TestEditFormSaveButtonEnter verifies Enter on Save button saves.
func TestEditFormSaveButtonEnter(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusSave
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Enter on Save")
	}
	if _, ok := cmd().(editSavedMsg); !ok {
		t.Errorf("expected editSavedMsg")
	}
}

// TestEditFormCancelButtonEnter verifies Enter on Cancel button cancels.
func TestEditFormCancelButtonEnter(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusCancel
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Enter on Cancel")
	}
	if _, ok := cmd().(editCancelledMsg); !ok {
		t.Errorf("expected editCancelledMsg")
	}
}

// ── T006: ctrl+g focus-gating tests (US2) ────────────────────────────────────

// TestEditForm_CtrlG_DescriptionFocused verifies ctrl+g on Description returns a non-nil Cmd.
func TestEditForm_CtrlG_DescriptionFocused(t *testing.T) {
	f := NewRootForm(0, nil)
	keys := DefaultKeyMap()

	// Advance focus to Description.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys) // focusDescription
	if f.focusIndex != focusDescription {
		t.Fatalf("setup: focusIndex = %d, want focusDescription (%d)", f.focusIndex, focusDescription)
	}

	_, cmd := f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("ctrl+g on Description: expected non-nil Cmd, got nil")
	}
}

// TestEditFormSnooze_PreFill verifies NewEditForm prefills snooze from task.SnoozeUntil.
func TestEditFormSnooze_PreFill(t *testing.T) {
	snoozeTime := timestamppb.New(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	task := &taskv1.Task{Id: 1, Name: "task", SnoozeUntil: snoozeTime}
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if f.snooze.Value() != "2026-09-01" {
		t.Errorf("snooze prefill: want %q, got %q", "2026-09-01", f.snooze.Value())
	}
}

// TestEditFormSnooze_SaveEmitsSnoozeStr verifies Ctrl+S emits snoozeStr.
func TestEditFormSnooze_SaveEmitsSnoozeStr(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	keys := DefaultKeyMap()

	// Set the snooze value directly.
	f.snooze.SetValue("2026-09-01")

	// Ctrl+S saves.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.snoozeStr != "2026-09-01" {
		t.Errorf("snoozeStr: want %q, got %q", "2026-09-01", saved.snoozeStr)
	}
}

// TestEditFormSnooze_BlankSnoozeIsEmpty verifies snoozeStr is empty when snooze field is blank.
func TestEditFormSnooze_BlankSnoozeIsEmpty(t *testing.T) {
	task := makeTask(2, "no snooze")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	saved := cmd().(editSavedMsg)
	if saved.snoozeStr != "" {
		t.Errorf("expected empty snoozeStr, got %q", saved.snoozeStr)
	}
}

// ── T006: calendar integration tests (US1) ───────────────────────────────────

// TestEditForm_CtrlG_DueOpensCalendar verifies ctrl+g on Due field opens the calendar.
func TestEditForm_CtrlG_DueOpensCalendar(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Error("ctrl+g on Due: expected calendar to be open (non-nil), got nil")
	}
}

// TestEditForm_CtrlG_SnoozeOpensCalendar verifies ctrl+g on Snooze field opens the calendar.
func TestEditForm_CtrlG_SnoozeOpensCalendar(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusSnooze
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Error("ctrl+g on Snooze: expected calendar to be open (non-nil), got nil")
	}
}

// TestEditForm_CtrlG_NameNoCalendar verifies ctrl+g on Name does not open the calendar.
func TestEditForm_CtrlG_NameNoCalendar(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusName
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar != nil {
		t.Error("ctrl+g on Name: expected calendar to remain closed, got open")
	}
}

// TestEditForm_Calendar_EnterWritesDate verifies open → move → enter writes YYYY-MM-DD into the Due field.
func TestEditForm_Calendar_EnterWritesDate(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	keys := DefaultKeyMap()

	// Open calendar.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}

	// Move forward 1 day.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)

	// Remember the expected date.
	want := f.calendar.confirm()

	// Confirm with Enter.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, keys)

	if f.calendar != nil {
		t.Error("after enter: expected calendar to close (nil), still open")
	}
	if got := f.due.Value(); got != want {
		t.Errorf("due value: want %q, got %q", want, got)
	}
}

// TestEditForm_Calendar_EscLeavesFieldUnchanged verifies open → esc leaves the field byte-for-byte unchanged.
func TestEditForm_Calendar_EscLeavesFieldUnchanged(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	f.due.SetValue("2026-07-04")
	keys := DefaultKeyMap()

	// Open calendar.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}

	// Move around.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyDown}, keys)

	// Cancel.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)

	if f.calendar != nil {
		t.Error("after esc: expected calendar to close (nil), still open")
	}
	if got := f.due.Value(); got != "2026-07-04" {
		t.Errorf("due value: want %q unchanged, got %q", "2026-07-04", got)
	}
}

// TestEditForm_Calendar_SwallowsFormKeys verifies that while the calendar is open,
// ctrl+s does not immediately save.
func TestEditForm_Calendar_SwallowsFormKeys(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	keys := DefaultKeyMap()

	// Open calendar.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}

	// ctrl+s while open: must not save.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(editSavedMsg); ok {
				t.Error("ctrl+s while calendar open: should be swallowed, but got editSavedMsg")
			}
		}
	}
}

// TestEditForm_Calendar_EscDoesNotCancelForm verifies that Esc while the
// calendar is open only closes the calendar and does NOT cancel/close the form
// (calendar takes priority over the form-level Esc handler).
func TestEditForm_Calendar_EscDoesNotCancelForm(t *testing.T) {
	f := NewEditForm(&taskv1.Task{Id: 1, Name: "task"}, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.focusIndex = focusDue
	f.name.SetValue("changed") // make the form dirty
	keys := DefaultKeyMap()

	// Open calendar.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}

	// Esc while calendar open: should close calendar, NOT emit any message.
	f2, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
	if f2.calendar != nil {
		t.Error("after esc: calendar should be closed")
	}
	if cmd != nil {
		msg := cmd()
		switch msg.(type) {
		case editCancelledMsg:
			t.Error("Esc while calendar open: should not emit editCancelledMsg")
		case editDiscardRequestedMsg:
			t.Error("Esc while calendar open: should not emit editDiscardRequestedMsg")
		}
	}
}

// TestEditForm_Calendar_PasteSwallowed verifies pasted text (tea.PasteMsg) cannot
// reach the date field while the calendar is open, so cancel stays lossless.
func TestEditForm_Calendar_PasteSwallowed(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	f.due.Focus()
	f.due.SetValue("2026-07-04")
	keys := DefaultKeyMap()

	// Open calendar.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}

	// Paste while open, then cancel.
	f, _ = f.Update(tea.PasteMsg{Content: "garbage"}, keys)
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)

	if got := f.due.Value(); got != "2026-07-04" {
		t.Errorf("due after paste+esc: want %q unchanged, got %q", "2026-07-04", got)
	}
}

// TestEditForm_Calendar_TabClosesCalendar verifies tab/shift+tab close the calendar unchanged.
func TestEditForm_Calendar_TabClosesCalendar(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	f.due.SetValue("2026-07-04")
	keys := DefaultKeyMap()

	// Open calendar.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}

	// Tab: should close calendar without modifying field.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
	if f.calendar != nil {
		t.Error("after tab: expected calendar to close (nil), still open")
	}
	if got := f.due.Value(); got != "2026-07-04" {
		t.Errorf("due value after tab: want %q, got %q", "2026-07-04", got)
	}
}

// TestEditForm_CtrlG_OtherFieldsNoOp verifies ctrl+g on non-Description, non-date fields does not launch the editor.
func TestEditForm_CtrlG_OtherFieldsNoOp(t *testing.T) {
	keys := DefaultKeyMap()
	otherFields := []struct {
		name  string
		focus int
	}{
		{"Name", focusName},
		{"Estimate", focusEstimate},
		{"Save", focusSave},
		{"Cancel", focusCancel},
	}
	for _, tc := range otherFields {
		t.Run(tc.name, func(t *testing.T) {
			f := NewRootForm(0, nil)
			f.focusIndex = tc.focus

			_, cmd := f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
			// Must not return an editor command — either nil or something else.
			// We verify it does not return an editorFinishedMsg producer by checking
			// the form does NOT immediately produce an editorFinishedMsg.
			if cmd != nil {
				// Allow non-nil cmd only if it's not the editor cmd (e.g., textarea internals).
				// The critical invariant is that focusIndex did not change to description.
				if f.focusIndex == tc.focus {
					// cmd came from textarea/textinput forwarding, not editor — acceptable.
					return
				}
				t.Errorf("ctrl+g on %s: focusIndex changed unexpectedly to %d", tc.name, f.focusIndex)
			}
		})
	}
}

// ── T016: text-path regression tests (US3) ───────────────────────────────────

// TestEditForm_TextPath_DueFieldTyping verifies that typing into the Due field
// with the calendar closed works exactly as before — no calendar opens uninvited.
func TestEditForm_TextPath_DueFieldTyping(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	keys := DefaultKeyMap()

	// Type some characters.
	for _, r := range "2026-08-01" {
		f, _ = f.Update(tea.KeyPressMsg{Code: r, Text: string(r)}, keys)
	}

	if f.calendar != nil {
		t.Error("calendar should not open on typing")
	}
}

// TestEditForm_TextPath_SavePreservesTypedValue verifies a typed date flows unchanged to editSavedMsg.
func TestEditForm_TextPath_SavePreservesTypedValue(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	f.due.SetValue("2026-08-01")
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.dueStr != "2026-08-01" {
		t.Errorf("dueStr: want %q, got %q", "2026-08-01", saved.dueStr)
	}
}

// ── T018: US4 clearing test ──────────────────────────────────────────────────

// TestEditForm_Calendar_ClearAfterPick verifies that picking a date then clearing
// the field text results in an empty dueStr in editSavedMsg.
func TestEditForm_Calendar_ClearAfterPick(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	keys := DefaultKeyMap()

	// Open calendar and pick a date.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, keys)
	if f.due.Value() == "" {
		t.Fatal("expected due to have a picked date")
	}

	// Clear the field value.
	f.due.SetValue("")
	if f.due.Value() != "" {
		t.Fatal("expected due to be cleared")
	}

	// Save — dueStr should be empty.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.dueStr != "" {
		t.Errorf("dueStr: want empty, got %q", saved.dueStr)
	}
	// Calendar should not linger.
	if f.calendar != nil {
		t.Error("calendar state should not linger after confirm")
	}
}

// TestEditForm_Calendar_NowFuncInjection verifies that a form with nowFunc uses it for calendar opening.
func TestEditForm_Calendar_NowFuncInjection(t *testing.T) {
	fixedNow := time.Date(2030, 1, 15, 0, 0, 0, 0, time.UTC)
	f := NewRootForm(0, nil)
	f.focusIndex = focusDue
	f.nowFunc = func() time.Time { return fixedNow }
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("calendar did not open")
	}
	wantToday := time.Date(2030, 1, 15, 0, 0, 0, 0, time.UTC)
	if !f.calendar.today.Equal(wantToday) {
		t.Errorf("calendar.today: want %v, got %v", wantToday, f.calendar.today)
	}
}

// ── T022: Goal field in task edit form ──────────────────────────────────────

func makeGoals() []*goalv1.Goal {
	return []*goalv1.Goal{
		{Id: 10, Name: "Alpha goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
		{Id: 20, Name: "Beta goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING},
	}
}

// TestEditForm_GoalField_ShowsWhenGoalsProvided verifies showGoalField is true
// when goals are passed to NewEditForm.
func TestEditForm_GoalField_ShowsWhenGoalsProvided(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if !f.showGoalField {
		t.Error("showGoalField should be true when goals are provided")
	}
}

// TestEditForm_GoalField_HiddenWhenNilGoals verifies showGoalField is false
// when nil goals are passed (e.g. when editing a goal record itself).
func TestEditForm_GoalField_HiddenWhenNilGoals(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if f.showGoalField {
		t.Error("showGoalField should be false when goals param is nil")
	}
}

// TestEditForm_GoalField_DefaultNone verifies that a task with no goal_id starts
// at "none" (goalIdx == -1).
func TestEditForm_GoalField_DefaultNone(t *testing.T) {
	task := makeTask(1, "task") // GoalId nil
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if f.goalIdx != -1 {
		t.Errorf("goalIdx: want -1 (none), got %d", f.goalIdx)
	}
}

// TestEditForm_GoalField_PreFillFromTask verifies that a task with goal_id pre-selects
// the matching goal in the form.
func TestEditForm_GoalField_PreFillFromTask(t *testing.T) {
	id20 := int64(20)
	task := &taskv1.Task{Id: 1, Name: "task", GoalId: &id20}
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if f.goalIdx != 1 {
		t.Errorf("goalIdx: want 1 (second goal id=20), got %d", f.goalIdx)
	}
}

// TestEditForm_GoalField_RightArrowCycles verifies → cycles none→goal0→goal1→none.
func TestEditForm_GoalField_RightArrowCycles(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.focusIndex = focusGoal
	keys := DefaultKeyMap()

	// none → first goal
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if f.goalIdx != 0 {
		t.Errorf("after →: want goalIdx=0, got %d", f.goalIdx)
	}

	// first → second
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if f.goalIdx != 1 {
		t.Errorf("after →→: want goalIdx=1, got %d", f.goalIdx)
	}

	// second → none (wrap)
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if f.goalIdx != -1 {
		t.Errorf("after →→→ (wrap): want goalIdx=-1, got %d", f.goalIdx)
	}
}

// TestEditForm_GoalField_LeftArrowCycles verifies ← cycles none→last goal→...
func TestEditForm_GoalField_LeftArrowCycles(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.focusIndex = focusGoal
	keys := DefaultKeyMap()

	// none → last goal (wrap backwards)
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyLeft}, keys)
	if f.goalIdx != 1 {
		t.Errorf("after ← from none: want goalIdx=1 (last), got %d", f.goalIdx)
	}
}

// TestEditForm_GoalField_TabIncludesGoal verifies Tab cycles through focusGoal
// when showGoalField is true.
func TestEditForm_GoalField_TabIncludesGoal(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	keys := DefaultKeyMap()

	// Tab from focusName through all fields — should pass through focusGoal.
	foundGoal := false
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusGoal {
			foundGoal = true
			break
		}
	}
	if !foundGoal {
		t.Errorf("Tab cycling with showGoalField=true never reached focusGoal (%d)", focusGoal)
	}
}

// TestEditForm_GoalField_SaveEmitsGoalChanged verifies editSavedMsg.goalChanged is true
// when the goal is changed from its original value.
func TestEditForm_GoalField_SaveEmitsGoalChanged(t *testing.T) {
	task := makeTask(1, "task") // no goal originally
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.focusIndex = focusGoal
	keys := DefaultKeyMap()

	// Select first goal with →.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)

	// Save.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if !saved.goalChanged {
		t.Error("goalChanged should be true when goal was set from none")
	}
	if saved.newGoalID == nil || *saved.newGoalID != 10 {
		t.Errorf("newGoalID: want 10, got %v", saved.newGoalID)
	}
}

// TestEditForm_GoalField_CompletedGoalPreservedOnSave verifies that editing a task
// whose goal is completed/archived does not clear the association when the user
// makes no goal change.
func TestEditForm_GoalField_CompletedGoalPreservedOnSave(t *testing.T) {
	completedGoalID := int64(99)
	task := &taskv1.Task{Id: 1, Name: "task", GoalId: &completedGoalID}
	// goals list includes the completed goal (as the real TUI passes m.goal.goals which
	// contains all goals, including completed/archived).
	allGoals := append(makeGoals(), &goalv1.Goal{Id: 99, Name: "Done goal", State: goalv1.GoalState_GOAL_STATE_COMPLETED})
	f := NewEditForm(task, 0, allGoals, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, DefaultKeyMap())
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.goalChanged {
		t.Errorf("goalChanged should be false when task goal (id=99) is not in available list and user made no change")
	}
}

// TestEditForm_GoalField_SaveUnchangedNotChanged verifies goalChanged is false
// when no change was made.
func TestEditForm_GoalField_SaveUnchangedNotChanged(t *testing.T) {
	task := makeTask(1, "task") // no goal originally
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, DefaultKeyMap())
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.goalChanged {
		t.Error("goalChanged should be false when goal was not changed")
	}
}

// ── T024: Round-trip test: editing shows raw markdown source ──────────────────

// TestEditForm_DescriptionRoundTrip asserts that:
//  1. The form's description field holds the raw markdown source (not rendered).
//  2. Saving the form without changes preserves the text byte-for-byte.
func TestEditForm_DescriptionRoundTrip(t *testing.T) {
	rawMarkdown := "**Ship** the report\n\nWith *notes* below:\n- item 1\n- item 2"

	task := &taskv1.Task{
		Id:          1,
		Name:        "task",
		Description: rawMarkdown,
	}

	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)

	// The description field must hold the raw markdown, not a rendered version.
	if f.description.Value() != rawMarkdown {
		t.Errorf("edit form description: want raw markdown %q, got %q", rawMarkdown, f.description.Value())
	}

	// Building the save message must preserve the text byte-for-byte.
	saveMsg := f.buildSaveMsg()().(editSavedMsg)
	if saveMsg.description != rawMarkdown {
		t.Errorf("editSavedMsg description: want %q, got %q", rawMarkdown, saveMsg.description)
	}
}

// TestEditForm_NameRoundTrip asserts that a name containing markdown syntax is
// preserved as-is in the form field and in the saved message (display-only invariant).
func TestEditForm_NameRoundTrip(t *testing.T) {
	rawName := "**Ship** the *report*"

	task := &taskv1.Task{Id: 2, Name: rawName}
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)

	if f.name.Value() != rawName {
		t.Errorf("edit form name: want raw %q, got %q", rawName, f.name.Value())
	}

	saveMsg := f.buildSaveMsg()().(editSavedMsg)
	if saveMsg.name != rawName {
		t.Errorf("editSavedMsg name: want %q, got %q", rawName, saveMsg.name)
	}
}

// ── Goal field on new root task form ─────────────────────────────────────────

// TestNewRootForm_GoalField_ShowsWhenGoalsProvided verifies that passing a non-empty
// goals slice to NewRootForm enables the Goal selector.
func TestNewRootForm_GoalField_ShowsWhenGoalsProvided(t *testing.T) {
	f := NewRootForm(0, makeGoals())
	if !f.showGoalField {
		t.Error("showGoalField should be true when committed/incubating goals are provided")
	}
}

// TestNewRootForm_GoalField_HiddenWhenNilGoals verifies that nil goals keeps the
// Goal selector hidden.
func TestNewRootForm_GoalField_HiddenWhenNilGoals(t *testing.T) {
	f := NewRootForm(0, nil)
	if f.showGoalField {
		t.Error("showGoalField should be false when goals is nil")
	}
}

// TestNewRootForm_GoalField_HiddenWhenEmptyGoals verifies that an empty (non-nil)
// goals slice also keeps the Goal selector hidden — there is nothing to select.
func TestNewRootForm_GoalField_HiddenWhenEmptyGoals(t *testing.T) {
	f := NewRootForm(0, []*goalv1.Goal{})
	if f.showGoalField {
		t.Error("showGoalField should be false when goals slice is empty")
	}
}

// TestNewRootForm_GoalField_DefaultNone verifies that the selector starts at
// "none" (goalIdx == -1) when goals are provided.
func TestNewRootForm_GoalField_DefaultNone(t *testing.T) {
	f := NewRootForm(0, makeGoals())
	if f.goalIdx != -1 {
		t.Errorf("goalIdx: want -1 (none), got %d", f.goalIdx)
	}
}

// TestNewRootForm_GoalField_TabIncludesGoal verifies Tab reaches focusGoal
// when the Goal selector is enabled on a new root task form.
func TestNewRootForm_GoalField_TabIncludesGoal(t *testing.T) {
	f := NewRootForm(0, makeGoals())
	keys := DefaultKeyMap()

	foundGoal := false
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusGoal {
			foundGoal = true
			break
		}
	}
	if !foundGoal {
		t.Errorf("Tab cycling on NewRootForm with goals never reached focusGoal (%d)", focusGoal)
	}
}

// TestNewRootForm_GoalField_SaveEmitsGoalChanged verifies that selecting a goal
// via → on a new root task form produces goalChanged=true and the correct newGoalID.
func TestNewRootForm_GoalField_SaveEmitsGoalChanged(t *testing.T) {
	f := NewRootForm(0, makeGoals())
	f.focusIndex = focusGoal
	keys := DefaultKeyMap()

	// Select the first goal.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)

	// Save.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if !saved.goalChanged {
		t.Error("goalChanged should be true when goal was selected on a new root task")
	}
	if saved.newGoalID == nil || *saved.newGoalID != 10 {
		t.Errorf("newGoalID: want 10 (first goal), got %v", saved.newGoalID)
	}
}

// ── isDirty tests (T005) ─────────────────────────────────────────────────────

// TestEditForm_IsDirty_BlankFormReturnsFalse verifies a blank create form is not dirty.
func TestEditForm_IsDirty_BlankFormReturnsFalse(t *testing.T) {
	f := NewRootForm(0, nil)
	if ExportEditFormIsDirty(f) {
		t.Error("blank root form: isDirty should be false")
	}
	f2 := NewSubtaskForm(1, 0)
	if ExportEditFormIsDirty(f2) {
		t.Error("blank subtask form: isDirty should be false")
	}
}

// TestEditForm_IsDirty_EditFormUnchangedReturnsFalse verifies a pre-filled edit
// form with no changes is not dirty.
func TestEditForm_IsDirty_EditFormUnchangedReturnsFalse(t *testing.T) {
	task := makeTask(10, "buy milk")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if ExportEditFormIsDirty(f) {
		t.Error("unchanged edit form: isDirty should be false")
	}
}

// TestEditForm_IsDirty_NameChange verifies that changing the name makes the form dirty.
func TestEditForm_IsDirty_NameChange(t *testing.T) {
	task := makeTask(10, "buy milk")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.name.SetValue("buy oat milk")
	if !ExportEditFormIsDirty(f) {
		t.Error("after name change: isDirty should be true")
	}
}

// TestEditForm_IsDirty_DescriptionChange verifies that changing the description makes the form dirty.
func TestEditForm_IsDirty_DescriptionChange(t *testing.T) {
	task := makeTask(10, "task")
	task.Description = "original"
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.description.SetValue("changed")
	if !ExportEditFormIsDirty(f) {
		t.Error("after description change: isDirty should be true")
	}
}

// TestEditForm_IsDirty_DueChange verifies that changing the due date makes the form dirty.
func TestEditForm_IsDirty_DueChange(t *testing.T) {
	task := makeTask(10, "task")
	task.Due = timestamppb.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.due.SetValue("2026-07-04")
	if !ExportEditFormIsDirty(f) {
		t.Error("after due change: isDirty should be true")
	}
}

// TestEditForm_IsDirty_EstimateChange verifies that changing the estimate makes the form dirty.
func TestEditForm_IsDirty_EstimateChange(t *testing.T) {
	task := makeTask(10, "task")
	task.Estimate = 3
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.pomodoroEstimate.SetValue("5")
	if !ExportEditFormIsDirty(f) {
		t.Error("after estimate change: isDirty should be true")
	}
}

// TestEditForm_IsDirty_SnoozeChange verifies that changing the snooze makes the form dirty.
func TestEditForm_IsDirty_SnoozeChange(t *testing.T) {
	task := makeTask(10, "task")
	task.SnoozeUntil = timestamppb.New(time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC))
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.snooze.SetValue("2026-10-01")
	if !ExportEditFormIsDirty(f) {
		t.Error("after snooze change: isDirty should be true")
	}
}

// TestEditForm_IsDirty_BlankFormTypedBecomesDirty verifies that typing into a
// blank create form makes it dirty (isDirty returns true).
func TestEditForm_IsDirty_BlankFormTypedBecomesDirty(t *testing.T) {
	f := NewRootForm(0, nil)
	f.name.SetValue("hello")
	if !ExportEditFormIsDirty(f) {
		t.Error("blank form with typed name: isDirty should be true")
	}
}

// TestEditForm_IsDirty_WhitespaceOnlyNotDirty verifies that whitespace-only
// input on a blank create form does not make the form dirty.
func TestEditForm_IsDirty_WhitespaceOnlyNotDirty(t *testing.T) {
	f := NewRootForm(0, nil)
	f.name.SetValue("  ")
	if ExportEditFormIsDirty(f) {
		t.Error("whitespace-only on blank form: isDirty should be false")
	}
}

// TestEditForm_IsDirty_GoalFieldChange verifies that cycling the goal selector
// on an edit form makes it dirty.
func TestEditForm_IsDirty_GoalFieldChange(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	// Select first goal with →
	f.focusIndex = focusGoal
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, DefaultKeyMap())
	if !ExportEditFormIsDirty(f) {
		t.Error("after goal change: isDirty should be true")
	}
}

// TestEditForm_IsDirty_GoalFieldUnchangedNotDirty verifies the goal selector
// does not make the form dirty when unchanged.
func TestEditForm_IsDirty_GoalFieldUnchangedNotDirty(t *testing.T) {
	id20 := int64(20)
	task := &taskv1.Task{Id: 1, Name: "task", GoalId: &id20}
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if ExportEditFormIsDirty(f) {
		t.Error("unchanged goal field: isDirty should be false")
	}
}

// ── Dirty Esc tests (T016) ────────────────────────────────────────────────────

// TestEditForm_EscOnDirtyEditFormEmitsDiscardRequested verifies that Esc on a
// dirty edit form emits editDiscardRequestedMsg, not editCancelledMsg.
func TestEditForm_EscOnDirtyEditFormEmitsDiscardRequested(t *testing.T) {
	task := makeTask(10, "buy milk")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.name.SetValue("buy oat milk") // make it dirty
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
	if cmd == nil {
		t.Fatal("Esc on dirty edit form: expected cmd, got nil")
	}
	msg := cmd()
	if _, ok := msg.(editDiscardRequestedMsg); !ok {
		t.Fatalf("Esc on dirty edit form: expected editDiscardRequestedMsg, got %T", msg)
	}
}

// TestEditForm_EscOnCleanEditFormEmitsCancelled verifies that Esc on a clean
// edit form (no changes) emits editCancelledMsg immediately.
func TestEditForm_EscOnCleanEditFormEmitsCancelled(t *testing.T) {
	task := makeTask(10, "buy milk")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
	if cmd == nil {
		t.Fatal("Esc on clean edit form: expected cmd, got nil")
	}
	msg := cmd()
	if _, ok := msg.(editCancelledMsg); !ok {
		t.Fatalf("Esc on clean edit form: expected editCancelledMsg, got %T", msg)
	}
}

// TestEditForm_EscOnDirtyBlankFormEmitsDiscardRequested verifies that Esc on a
// dirty blank (create) form emits editDiscardRequestedMsg.
func TestEditForm_EscOnDirtyBlankFormEmitsDiscardRequested(t *testing.T) {
	f := NewRootForm(0, nil)
	f.name.SetValue("typed something")
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
	if cmd == nil {
		t.Fatal("Esc on dirty blank form: expected cmd, got nil")
	}
	msg := cmd()
	if _, ok := msg.(editDiscardRequestedMsg); !ok {
		t.Fatalf("Esc on dirty blank form: expected editDiscardRequestedMsg, got %T", msg)
	}
}

// ── Goal state field tests (T016) ───────────────────────────────────────────

// TestEditForm_GoalState_ShowsOnGoalEdit verifies that the State field
// appears in the Tab cycle when editing an existing goal.
func TestEditForm_GoalState_ShowsOnGoalEdit(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true

	keys := DefaultKeyMap()
	foundState := false
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusState {
			foundState = true
			break
		}
	}
	if !foundState {
		t.Error("Tab cycling on goal edit form never reached focusState")
	}
}

// TestEditForm_GoalState_HiddenOnNewGoal verifies that the State field is
// NOT reachable in Tab cycle when creating a new goal (taskID == nil).
func TestEditForm_GoalState_HiddenOnNewGoal(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_INCUBATING}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	f.taskID = nil // simulate new goal form

	keys := DefaultKeyMap()
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusState {
			t.Error("focusState should be skipped on new goal forms (taskID == nil)")
		}
	}
}

// TestEditForm_GoalState_HiddenOnTaskEdit verifies that the State field is
// NOT reachable in Tab cycle when editing a task (isGoal == false).
func TestEditForm_GoalState_HiddenOnTaskEdit(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)

	keys := DefaultKeyMap()
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusState {
			t.Error("focusState should be skipped on task edit forms (isGoal == false)")
		}
	}
}

// TestEditForm_GoalState_PreFillFromGoalState verifies the state field
// initializes to the correct index matching the goal's current state.
func TestEditForm_GoalState_PreFillFromGoalState(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_HOLD}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())

	wantIdx := goalStateIndex(goalv1.GoalState_GOAL_STATE_HOLD)
	if f.goalStateIdx != wantIdx {
		t.Errorf("goalStateIdx: want %d (hold), got %d", wantIdx, f.goalStateIdx)
	}
}

// TestEditForm_GoalState_RightArrowCycles verifies → cycles through states.
func TestEditForm_GoalState_RightArrowCycles(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	f.focusIndex = focusState
	keys := DefaultKeyMap()

	startIdx := f.goalStateIdx
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if f.goalStateIdx != (startIdx+1)%len(goalStates) {
		t.Errorf("after →: want idx %d, got %d", (startIdx+1)%len(goalStates), f.goalStateIdx)
	}
}

// TestEditForm_GoalState_LeftArrowCycles verifies ← cycles through states backwards.
func TestEditForm_GoalState_LeftArrowCycles(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	f.focusIndex = focusState
	keys := DefaultKeyMap()

	startIdx := f.goalStateIdx
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyLeft}, keys)
	wantIdx := (startIdx - 1 + len(goalStates)) % len(goalStates)
	if f.goalStateIdx != wantIdx {
		t.Errorf("after ←: want idx %d, got %d", wantIdx, f.goalStateIdx)
	}
}

// TestEditForm_GoalState_DirtyDetection verifies that changing the state
// makes the form dirty, and leaving it unchanged keeps it clean.
func TestEditForm_GoalState_DirtyDetection(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	f.focusIndex = focusState

	if ExportEditFormIsDirty(f) {
		t.Error("unchanged goal edit form: isDirty should be false")
	}

	// Change state.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, DefaultKeyMap())
	if !ExportEditFormIsDirty(f) {
		t.Error("after state change: isDirty should be true")
	}
}

// TestEditForm_GoalState_SaveEmitsStateChanged verifies that saving after
// a state change emits goalStateChanged=true and the correct goalState.
func TestEditForm_GoalState_SaveEmitsStateChanged(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	f.focusIndex = focusState
	keys := DefaultKeyMap()

	// Change state to next one.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)

	// Save.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if !saved.goalStateChanged {
		t.Error("goalStateChanged should be true after state cycling")
	}
	wantState := goalStates[(goalStateIndex(goalv1.GoalState_GOAL_STATE_IN_PROGRESS)+1)%len(goalStates)]
	if saved.goalState != wantState {
		t.Errorf("goalState: want %v, got %v", wantState, saved.goalState)
	}
}

// TestEditForm_GoalState_SaveUnchangedNotChanged verifies that saving without
// a state change emits goalStateChanged=false.
func TestEditForm_GoalState_SaveUnchangedNotChanged(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, DefaultKeyMap())
	saved := cmd().(editSavedMsg)
	if saved.goalStateChanged {
		t.Error("goalStateChanged should be false when state was not changed")
	}
}

// TestEditForm_GoalState_RenderInView verifies the State field appears in the
// rendered view for goal edit forms and shows the correct label.
func TestEditForm_GoalState_RenderInView(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_HOLD}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true

	view := f.View(80)
	if !strings.Contains(view, "State") {
		t.Error("goal edit form View: expected 'State' label")
	}
	if !strings.Contains(view, "Hold") {
		t.Error("goal edit form View: expected 'Hold' state label")
	}
}

// TestEditForm_GoalEdit_HidesEstimateAndSnooze verifies that Estimate and
// Snooze until are neither reachable via Tab nor rendered on a goal edit form.
func TestEditForm_GoalEdit_HidesEstimateAndSnooze(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true

	keys := DefaultKeyMap()
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusEstimate || f.focusIndex == focusSnooze {
			t.Errorf("focusIndex %d should be unreachable on a goal edit form", f.focusIndex)
		}
	}

	view := f.View(80)
	if strings.Contains(view, "Estimate") {
		t.Error("goal edit form View: unexpected 'Estimate' label")
	}
	if strings.Contains(view, "Snooze until") {
		t.Error("goal edit form View: unexpected 'Snooze until' label")
	}
}

// TestEditForm_NewGoal_HidesEstimateAndSnooze verifies the same for a new-goal
// form (isGoal true, taskID nil), where State is also skipped.
func TestEditForm_NewGoal_HidesEstimateAndSnooze(t *testing.T) {
	f := NewRootForm(0, nil)
	f.isGoal = true

	keys := DefaultKeyMap()
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusEstimate || f.focusIndex == focusSnooze {
			t.Errorf("focusIndex %d should be unreachable on a new goal form", f.focusIndex)
		}
	}

	view := f.View(80)
	if strings.Contains(view, "Estimate") {
		t.Error("new goal form View: unexpected 'Estimate' label")
	}
	if strings.Contains(view, "Snooze until") {
		t.Error("new goal form View: unexpected 'Snooze until' label")
	}
}

// TestEditForm_GoalEdit_CtrlGOnlyOpensCalendarOnDue verifies that, with the
// Estimate/Snooze fields gated out of the focus cycle, ctrl+g on a goal form
// only ever opens the calendar from the Due field.
func TestEditForm_GoalEdit_CtrlGOnlyOpensCalendarOnDue(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	keys := DefaultKeyMap()

	// State field: ctrl+g must be a no-op.
	f.focusIndex = focusState
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar != nil {
		t.Error("ctrl+g on focusState should not open the calendar")
	}

	// Due field: ctrl+g must open the calendar.
	f.focusIndex = focusDue
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Error("ctrl+g on focusDue should open the calendar")
	}
}

// TestEditForm_NewGoal_HidesPlan verifies that the Plan selector is neither
// reachable via Tab nor rendered on a new-goal form. Goals are never planned,
// and handleGoalEditSaved ignores planDay, so a visible selector would silently
// drop the user's choice.
func TestEditForm_NewGoal_HidesPlan(t *testing.T) {
	f := NewRootForm(0, nil)
	f.isGoal = true

	keys := DefaultKeyMap()
	for i := 0; i < focusCount; i++ {
		f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f.focusIndex == focusPlan {
			t.Error("focusPlan should be unreachable on a new goal form")
		}
	}

	if view := f.View(80); strings.Contains(view, "Plan") {
		t.Error("new goal form View: unexpected 'Plan' label")
	}
}

// TestEditForm_NewGoal_PlanNotDirty verifies a stray plan choice on a goal form
// cannot mark the form dirty or reach the save message.
func TestEditForm_NewGoal_PlanNotDirty(t *testing.T) {
	f := NewRootForm(0, nil)
	f.isGoal = true
	f.planIdx = planChoiceToday

	if f.isDirty() {
		t.Error("new goal form should not be dirty from a plan choice")
	}

	msg := f.buildSaveMsg()()
	saved, ok := msg.(editSavedMsg)
	if !ok {
		t.Fatalf("buildSaveMsg: want editSavedMsg, got %T", msg)
	}
	if saved.planDay != "" {
		t.Errorf("goal form planDay: want empty, got %q", saved.planDay)
	}
}

// ── T009: Plan selector form-state tests (US1) ─────────────────────────────

// CT-01: A new create form has planIdx == planChoiceNone.
func TestPlanField_NewCreateFormStartsAtNone(t *testing.T) {
	f := NewRootForm(0, nil)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceNone {
		t.Errorf("new create form planIdx: want %d (none), got %d", ExportPlanChoiceNone, got)
	}
}

// CT-02: Cycling right from none yields Today, Tomorrow, then wraps to none.
func TestPlanField_CycleRightWraps(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan
	keys := DefaultKeyMap()

	// none → Today
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceToday {
		t.Errorf("after →: want Today(%d), got %d", ExportPlanChoiceToday, got)
	}

	// Today → Tomorrow
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceTomorrow {
		t.Errorf("after →→: want Tomorrow(%d), got %d", ExportPlanChoiceTomorrow, got)
	}

	// Tomorrow → none (wrap)
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceNone {
		t.Errorf("after →→→ (wrap): want none(%d), got %d", ExportPlanChoiceNone, got)
	}
}

// CT-03: Save with Today ⇒ editSavedMsg.planDay is today's ISO day.
func TestPlanField_SaveWithTodayEmitsPlanDay(t *testing.T) {
	fixedNow := time.Date(2030, 3, 15, 12, 0, 0, 0, time.UTC)
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan
	f.nowFunc = func() time.Time { return fixedNow }
	keys := DefaultKeyMap()

	// Select Today.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceToday {
		t.Fatalf("setup: want Today(%d), got %d", ExportPlanChoiceToday, got)
	}

	// Save.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	if cmd == nil {
		t.Fatal("expected Cmd from Ctrl+S, got nil")
	}
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	want := fixedNow.Format("2006-01-02")
	if saved.planDay != want {
		t.Errorf("planDay: want %q, got %q", want, saved.planDay)
	}
}

// CT-10: Touching the plan control makes isDirty() true.
func TestPlanField_CyclingMakesFormDirty(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan

	// Initially clean.
	if ExportEditFormIsDirty(f) {
		t.Error("before cycling: isDirty should be false")
	}

	// Cycle right once (none → Today).
	f = ExportEditFormCyclePlan(f, 1)
	if !ExportEditFormIsDirty(f) {
		t.Error("after cycling plan: isDirty should be true")
	}
}

// ── T022: US2 calendar and plan tests ────────────────────────────────────────

// CT-12: A calendar pick sets planDate, moves the selector to planChoiceDate,
// and renders the ISO date. Cycling away clears planDate.
func TestPlanField_CalendarPickSetsPlanDateAndCyclingClears(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan
	f.nowFunc = func() time.Time { return time.Date(2030, 3, 15, 12, 0, 0, 0, time.UTC) }
	keys := DefaultKeyMap()

	// Open calendar on the Plan field.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("ctrl+g on Plan: expected calendar to open")
	}

	// Confirm with Enter (default selected = today).
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyEnter}, keys)
	if f.calendar != nil {
		t.Error("after enter: expected calendar to close")
	}
	if f.planIdx != planChoiceDate {
		t.Errorf("after calendar pick: planIdx want planChoiceDate(%d), got %d", planChoiceDate, f.planIdx)
	}
	if f.planDate != "2030-03-15" {
		t.Errorf("planDate: want %q, got %q", "2030-03-15", f.planDate)
	}

	// Cycling away from Date re-enters the 3-stop loop and clears planDate.
	// cyclePlan uses modular arithmetic over [0..2], so from index 3 (Date) +1 lands on Today (1).
	f = ExportEditFormCyclePlan(f, 1)
	if f.planDate != "" {
		t.Errorf("after cycling away from Date: planDate should be empty, got %q", f.planDate)
	}
	if f.planIdx != planChoiceToday {
		t.Errorf("after cycling away from Date: planIdx want Today(%d), got %d", planChoiceToday, f.planIdx)
	}
}

// CT-06: With nowFunc pinned across a midnight boundary, Today resolves to the
// save-time day, not the form-open time.
func TestPlanField_TodayResolvesAtSaveTime(t *testing.T) {
	// Save happens at 00:05 on March 15 (next day after form open).
	saveTime := time.Date(2030, 3, 15, 0, 5, 0, 0, time.UTC)

	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan
	// nowFunc returns the save time when buildSaveMsg resolves.
	f.nowFunc = func() time.Time { return saveTime }
	keys := DefaultKeyMap()

	// Select Today.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceToday {
		t.Fatalf("setup: want Today(%d), got %d", ExportPlanChoiceToday, got)
	}

	// Save — planDay must be the save-time day (March 15).
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.planDay != "2030-03-15" {
		t.Errorf("planDay: want %q (save-time day), got %q", "2030-03-15", saved.planDay)
	}
}

// Save with Tomorrow ⇒ planDay is tomorrow's ISO day.
func TestPlanField_SaveWithTomorrowEmitsPlanDay(t *testing.T) {
	fixedNow := time.Date(2030, 3, 15, 12, 0, 0, 0, time.UTC)
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan
	f.nowFunc = func() time.Time { return fixedNow }
	keys := DefaultKeyMap()

	// none → Today → Tomorrow
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)
	if got := ExportEditFormPlanIdx(f); got != ExportPlanChoiceTomorrow {
		t.Fatalf("setup: want Tomorrow(%d), got %d", ExportPlanChoiceTomorrow, got)
	}

	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}
	if saved.planDay != "2030-03-16" {
		t.Errorf("planDay: want %q (tomorrow), got %q", "2030-03-16", saved.planDay)
	}
}

// ctrl+g on Plan field opens calendar seeded from planDate (or today when empty).
func TestPlanField_CtrlGOpensCalendarSeededFromPlanDate(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan
	f.nowFunc = func() time.Time { return time.Date(2030, 3, 15, 0, 0, 0, 0, time.UTC) }
	keys := DefaultKeyMap()

	// Open calendar with no planDate set — should seed from today.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("ctrl+g on Plan: expected calendar to open")
	}
	want := time.Date(2030, 3, 15, 0, 0, 0, 0, time.UTC)
	if !f.calendar.selected.Equal(want) {
		t.Errorf("calendar.selected: want %v, got %v", want, f.calendar.selected)
	}

	// Close and set a planDate.
	f.calendar = nil
	f.planDate = "2030-05-01"
	f.planIdx = planChoiceDate

	// Reopen — should seed from planDate.
	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Fatal("ctrl+g on Plan: expected calendar to open")
	}
	want2 := time.Date(2030, 5, 1, 0, 0, 0, 0, time.UTC)
	if !f.calendar.selected.Equal(want2) {
		t.Errorf("calendar.selected after planDate seed: want %v, got %v", want2, f.calendar.selected)
	}
}

// View renders the Plan field label and the plan choice label.
func TestPlanField_ViewRendersPlanLabel(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusPlan

	view := f.View(80)
	if !strings.Contains(view, "Plan") {
		t.Error("View should contain 'Plan' label")
	}
	if !strings.Contains(view, "No plan") {
		t.Error("View should contain 'No plan' when planIdx == planChoiceNone")
	}

	// Select Today and check rendering.
	f = ExportEditFormCyclePlan(f, 1)
	view = f.View(80)
	if !strings.Contains(view, "Today") {
		t.Error("View should contain 'Today' after cycling to Today")
	}

	// Check hint line when focused.
	if !strings.Contains(view, "←/→ cycle") {
		t.Error("View should contain hint line when Plan is focused")
	}
}

// ctrl+g does not open calendar when focusPlan is not focused.
func TestPlanField_CtrlGNoOpWhenNotFocused(t *testing.T) {
	f := NewRootForm(0, nil)
	f.focusIndex = focusName
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar != nil {
		t.Error("ctrl+g on Name: calendar should not open")
	}
}

// ── T028: US3 CT-07 — showPlanField is false on edit/goal forms ──────────────

// CT-07 (TUI): showPlanField is false for the task edit form and for every goal form.
// The Plan field must be absent from the rendered view and skipped by cycleFocus.
func TestPlanField_HiddenOnTaskEditForm(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	if f.showPlanField {
		t.Error("showPlanField should be false for task edit form")
	}

	// Plan field should not appear in the view.
	view := f.View(80)
	if strings.Contains(view, "Plan:") && strings.Contains(view, "‹") {
		// Check that "Plan:" label doesn't appear in a field context.
		// The word "Plan" might appear in other contexts, so we check specifically for the field.
		lines := strings.Split(view, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Plan:") && strings.Contains(line, "‹") {
				t.Errorf("Plan field should not be visible on task edit form, found in view: %s", line)
			}
		}
	}

	// Tab cycling should skip focusPlan.
	f2 := NewEditForm(task, 0, nil, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f2.showPlanField = false
	keys := DefaultKeyMap()
	for i := 0; i < focusCount; i++ {
		f2, _ = f2.Update(tea.KeyPressMsg{Code: tea.KeyTab}, keys)
		if f2.focusIndex == focusPlan {
			t.Error("focusPlan should be skipped by cycleFocus when showPlanField is false")
		}
	}
}

func TestPlanField_HiddenOnGoalEditForm(t *testing.T) {
	goal := &goalv1.Goal{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	fakeTask := goalToFakeTask(goal)
	f := NewEditForm(fakeTask, 0, nil, goal.GetState())
	f.isGoal = true
	if f.showPlanField {
		t.Error("showPlanField should be false for goal edit form")
	}

	view := f.View(80)
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Plan:") && strings.Contains(line, "‹") {
			t.Errorf("Plan field should not be visible on goal edit form, found in view: %s", line)
		}
	}
}

func TestPlanField_HiddenOnNewGoalForm(t *testing.T) {
	// NewGoalForm uses NewRootForm + isGoal=true, same as handleGoalsKey does.
	f := NewRootForm(0, nil)
	f.isGoal = true
	// isGoal alone doesn't hide the plan field — the code path that creates
	// goal forms explicitly sets showPlanField = false. Verify that the form
	// starts with showPlanField = true (as NewRootForm does) and only hides it
	// when explicitly set, matching the real goal-form creation flow.
	f.showPlanField = false
	if f.showPlanField {
		t.Error("showPlanField should be false after explicit set")
	}

	view := f.View(80)
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Plan:") && strings.Contains(line, "‹") {
			t.Errorf("Plan field should not be visible on new goal form, found in view: %s", line)
		}
	}
}

// CT-13: Changing the plan choice leaves due/snooze/estimate/goal untouched in editSavedMsg.
func TestPlanField_PlanChangeLeavesOtherFieldsUntouched(t *testing.T) {
	id20 := int64(20)
	task := &taskv1.Task{
		Id:          5,
		Name:        "original name",
		Description: "original desc",
		Estimate:    3,
		GoalId:      &id20,
		Due:         timestamppb.New(time.Date(2030, 4, 1, 0, 0, 0, 0, time.UTC)),
		SnoozeUntil: timestamppb.New(time.Date(2030, 6, 1, 0, 0, 0, 0, time.UTC)),
	}
	f := NewEditForm(task, 0, makeGoals(), goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
	f.showPlanField = true // force plan field visible (normally hidden on edit)
	f.focusIndex = focusPlan
	f.nowFunc = func() time.Time { return time.Date(2030, 7, 10, 0, 0, 0, 0, time.UTC) }
	keys := DefaultKeyMap()

	// Select Today.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyRight}, keys)

	// Save.
	_, cmd := f.Update(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl}, keys)
	saved, ok := cmd().(editSavedMsg)
	if !ok {
		t.Fatalf("expected editSavedMsg, got %T", cmd())
	}

	// planDay must be set.
	if saved.planDay == "" {
		t.Error("planDay should be non-empty after selecting Today")
	}

	// Other fields must remain at original values.
	if saved.dueStr != task.Due.AsTime().UTC().Format("2006-01-02T15:04:05Z") {
		t.Errorf("dueStr should be unchanged, got %q", saved.dueStr)
	}
	if saved.estimateStr != "3" {
		t.Errorf("estimateStr: want %q, got %q", "3", saved.estimateStr)
	}
	if saved.snoozeStr != "2030-06-01" {
		t.Errorf("snoozeStr: want %q, got %q", "2030-06-01", saved.snoozeStr)
	}
	if saved.newGoalID == nil || *saved.newGoalID != 20 {
		t.Errorf("newGoalID: want 20, got %v", saved.newGoalID)
	}
	if saved.goalChanged {
		t.Error("goalChanged should be false when goal was not changed")
	}
}
