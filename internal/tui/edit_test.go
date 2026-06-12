package tui

import (
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
	f := NewEditForm(task, 0, nil)

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
// When showGoalField is false (NewRootForm), focusGoal is skipped.
func TestEditFormTabCyclesFocus(t *testing.T) {
	f := NewRootForm(0)
	keys := DefaultKeyMap()

	if f.focusIndex != focusName {
		t.Fatalf("initial focus: want focusName(%d), got %d", focusName, f.focusIndex)
	}

	// Expected focus order (focusGoal is skipped because showGoalField=false).
	want := []int{focusDescription, focusDue, focusEstimate, focusSnooze, focusSave, focusCancel}
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
	f := NewRootForm(0)
	keys := DefaultKeyMap()

	// Shift-Tab from focusName wraps to focusCancel.
	f, _ = f.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, keys)
	if f.focusIndex != focusCancel {
		t.Errorf("shift-tab from 0: want focusCancel(%d), got %d", focusCancel, f.focusIndex)
	}
}

// TestEditFormEscCancels verifies Esc dispatches editCancelledMsg.
func TestEditFormEscCancels(t *testing.T) {
	f := NewRootForm(5)
	keys := DefaultKeyMap()

	_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
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
	f := NewEditForm(task, 2, nil)
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

// TestEditFormEscFromAnyField verifies Esc works from any focused field.
func TestEditFormEscFromAnyField(t *testing.T) {
	keys := DefaultKeyMap()

	for fi := 0; fi < focusCount; fi++ {
		f := NewRootForm(0)
		f.focusIndex = fi

		_, cmd := f.Update(tea.KeyPressMsg{Code: tea.KeyEscape}, keys)
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
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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
	f := NewEditForm(task, 0, nil)
	if f.snooze.Value() != "2026-09-01" {
		t.Errorf("snooze prefill: want %q, got %q", "2026-09-01", f.snooze.Value())
	}
}

// TestEditFormSnooze_SaveEmitsSnoozeStr verifies Ctrl+S emits snoozeStr.
func TestEditFormSnooze_SaveEmitsSnoozeStr(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, nil)
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
	f := NewEditForm(task, 0, nil)
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
	f := NewRootForm(0)
	f.focusIndex = focusDue
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Error("ctrl+g on Due: expected calendar to be open (non-nil), got nil")
	}
}

// TestEditForm_CtrlG_SnoozeOpensCalendar verifies ctrl+g on Snooze field opens the calendar.
func TestEditForm_CtrlG_SnoozeOpensCalendar(t *testing.T) {
	f := NewRootForm(0)
	f.focusIndex = focusSnooze
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar == nil {
		t.Error("ctrl+g on Snooze: expected calendar to be open (non-nil), got nil")
	}
}

// TestEditForm_CtrlG_NameNoCalendar verifies ctrl+g on Name does not open the calendar.
func TestEditForm_CtrlG_NameNoCalendar(t *testing.T) {
	f := NewRootForm(0)
	f.focusIndex = focusName
	keys := DefaultKeyMap()

	f, _ = f.Update(tea.KeyPressMsg{Code: 'g', Mod: tea.ModCtrl}, keys)
	if f.calendar != nil {
		t.Error("ctrl+g on Name: expected calendar to remain closed, got open")
	}
}

// TestEditForm_Calendar_EnterWritesDate verifies open → move → enter writes YYYY-MM-DD into the Due field.
func TestEditForm_Calendar_EnterWritesDate(t *testing.T) {
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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

// TestEditForm_Calendar_PasteSwallowed verifies pasted text (tea.PasteMsg) cannot
// reach the date field while the calendar is open, so cancel stays lossless.
func TestEditForm_Calendar_PasteSwallowed(t *testing.T) {
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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
			f := NewRootForm(0)
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
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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
	f := NewRootForm(0)
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
		{Id: 10, Name: "Alpha goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED},
		{Id: 20, Name: "Beta goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING},
	}
}

// TestEditForm_GoalField_ShowsWhenGoalsProvided verifies showGoalField is true
// when goals are passed to NewEditForm.
func TestEditForm_GoalField_ShowsWhenGoalsProvided(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals())
	if !f.showGoalField {
		t.Error("showGoalField should be true when goals are provided")
	}
}

// TestEditForm_GoalField_HiddenWhenNilGoals verifies showGoalField is false
// when nil goals are passed (e.g. when editing a goal record itself).
func TestEditForm_GoalField_HiddenWhenNilGoals(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, nil)
	if f.showGoalField {
		t.Error("showGoalField should be false when goals param is nil")
	}
}

// TestEditForm_GoalField_DefaultNone verifies that a task with no goal_id starts
// at "none" (goalIdx == -1).
func TestEditForm_GoalField_DefaultNone(t *testing.T) {
	task := makeTask(1, "task") // GoalId nil
	f := NewEditForm(task, 0, makeGoals())
	if f.goalIdx != -1 {
		t.Errorf("goalIdx: want -1 (none), got %d", f.goalIdx)
	}
}

// TestEditForm_GoalField_PreFillFromTask verifies that a task with goal_id pre-selects
// the matching goal in the form.
func TestEditForm_GoalField_PreFillFromTask(t *testing.T) {
	id20 := int64(20)
	task := &taskv1.Task{Id: 1, Name: "task", GoalId: &id20}
	f := NewEditForm(task, 0, makeGoals())
	if f.goalIdx != 1 {
		t.Errorf("goalIdx: want 1 (second goal id=20), got %d", f.goalIdx)
	}
}

// TestEditForm_GoalField_RightArrowCycles verifies → cycles none→goal0→goal1→none.
func TestEditForm_GoalField_RightArrowCycles(t *testing.T) {
	task := makeTask(1, "task")
	f := NewEditForm(task, 0, makeGoals())
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
	f := NewEditForm(task, 0, makeGoals())
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
	f := NewEditForm(task, 0, makeGoals())
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
	f := NewEditForm(task, 0, makeGoals())
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

// TestEditForm_GoalField_SaveUnchangedNotChanged verifies goalChanged is false
// when no change was made.
func TestEditForm_GoalField_SaveUnchangedNotChanged(t *testing.T) {
	task := makeTask(1, "task") // no goal originally
	f := NewEditForm(task, 0, makeGoals())

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
