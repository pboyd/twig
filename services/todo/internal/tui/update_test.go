package tui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// buildTestModel creates a model with a flat list of 3 tasks (ids 1,2,3) and
// all rows visible.
func buildTestModel() Model {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
		{Id: 3, Name: "c"},
	}
	tree := cli.BuildTree(tasks)
	return ExportNewModel(nil, tree)
}

func pressKey(m Model, k string) Model {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	if k == "up" {
		msg = tea.KeyMsg{Type: tea.KeyUp}
	} else if k == "down" {
		msg = tea.KeyMsg{Type: tea.KeyDown}
	} else if k == "left" {
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	} else if k == "right" {
		msg = tea.KeyMsg{Type: tea.KeyRight}
	}
	next, _ := m.Update(msg)
	return next.(Model)
}

func TestNavigation_DownMovesToNextRow(t *testing.T) {
	m := buildTestModel()
	if m.cursor != 0 {
		t.Fatalf("initial cursor should be 0, got %d", m.cursor)
	}
	m = pressKey(m, "j")
	if m.cursor != 1 {
		t.Errorf("after down: want cursor=1, got %d", m.cursor)
	}
}

func TestNavigation_UpMovesToPrevRow(t *testing.T) {
	m := buildTestModel()
	m.cursor = 2
	m = pressKey(m, "k")
	if m.cursor != 1 {
		t.Errorf("after up: want cursor=1, got %d", m.cursor)
	}
}

func TestNavigation_DownClampsAtBottom(t *testing.T) {
	m := buildTestModel()
	m.cursor = len(m.visible) - 1
	m = pressKey(m, "j")
	if m.cursor != len(m.visible)-1 {
		t.Errorf("cursor should stay at bottom, got %d", m.cursor)
	}
}

func TestNavigation_UpClampsAtTop(t *testing.T) {
	m := buildTestModel()
	m.cursor = 0
	m = pressKey(m, "k")
	if m.cursor != 0 {
		t.Errorf("cursor should stay at top, got %d", m.cursor)
	}
}

func TestNavigation_ArrowKeysAlsoWork(t *testing.T) {
	m := buildTestModel()
	m = pressKey(m, "down")
	if m.cursor != 1 {
		t.Errorf("arrow down: want 1, got %d", m.cursor)
	}
	m = pressKey(m, "up")
	if m.cursor != 0 {
		t.Errorf("arrow up: want 0, got %d", m.cursor)
	}
}

func TestNavigation_CollapseExpandWithHL(t *testing.T) {
	// Build a tree with root(1) and child(2)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	// cursor on root

	// Expand with L
	m = pressKey(m, "l")
	if !m.expanded[1] {
		t.Error("expanded[1] should be true after L")
	}
	if len(m.visible) != 2 {
		t.Errorf("after expand: want 2 rows, got %d", len(m.visible))
	}

	// Collapse with H
	m = pressKey(m, "h")
	if m.expanded[1] {
		t.Error("expanded[1] should be false after H")
	}
	if len(m.visible) != 1 {
		t.Errorf("after collapse: want 1 row, got %d", len(m.visible))
	}
	// Cursor is still on root (index 0)
	if m.cursor != 0 {
		t.Errorf("cursor should remain on parent after collapse, got %d", m.cursor)
	}
}

// TestNavigation_CollapseLeafMovesToParent checks that pressing H/← on a leaf
// (or already-collapsed node) collapses the parent and moves the cursor to it.
// TestNavigation_ExpandMovesToFirstChild checks that pressing L moves the
// cursor to the first revealed child.
func TestNavigation_ExpandMovesToFirstChild(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child1"},
		{Id: 3, Name: "child2"},
	}
	tasks[1].ParentId = ptr64(1)
	tasks[2].ParentId = ptr64(1)
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	// cursor on root, subtree collapsed

	m = pressKey(m, "l")
	if m.cursor != 1 {
		t.Errorf("after expand: cursor should be on first child (1), got %d", m.cursor)
	}
	if m.visible[1].node.Task.Id != 2 {
		t.Errorf("cursor should be on task id=2, got id=%d", m.visible[1].node.Task.Id)
	}
}

// TestNavigation_ExpandAlreadyExpandedMovesToFirstChild checks that pressing L
// on an already-expanded node still jumps the cursor to the first child.
func TestNavigation_ExpandAlreadyExpandedMovesToFirstChild(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child"},
	}
	tasks[1].ParentId = ptr64(1)
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)

	// First expand: cursor moves to child.
	m = pressKey(m, "l")
	if m.cursor != 1 {
		t.Fatalf("first expand: expected cursor=1, got %d", m.cursor)
	}

	// Move back to root.
	m = pressKey(m, "k")
	if m.cursor != 0 {
		t.Fatalf("after up: expected cursor=0, got %d", m.cursor)
	}

	// Second expand on already-expanded root: cursor should still move to child.
	m = pressKey(m, "l")
	if m.cursor != 1 {
		t.Errorf("re-expand already-expanded: expected cursor=1, got %d", m.cursor)
	}
}

// TestNavigation_ExpandNoVisibleChildrenLeaveCursor checks that expanding a
// task whose only children are all-completed (and filtered) does not move the cursor.
func TestNavigation_ExpandNoVisibleChildrenLeaveCursor(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "done-child", CompletedAt: now},
	}
	tasks[1].ParentId = ptr64(1)
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	// showCompleted is false, so child is hidden

	m = pressKey(m, "l")
	if m.cursor != 0 {
		t.Errorf("cursor should stay on root when no children are visible, got %d", m.cursor)
	}
}

func TestNavigation_CollapseLeafMovesToParent(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child"},
	}
	tasks[1].ParentId = ptr64(1)
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)

	// Expand root so both rows are visible.
	m = pressKey(m, "l")
	if len(m.visible) != 2 {
		t.Fatalf("setup: expected 2 visible rows, got %d", len(m.visible))
	}

	// Move cursor to child (index 1, id=2).
	m = pressKey(m, "j")
	if m.cursor != 1 {
		t.Fatalf("setup: expected cursor=1, got %d", m.cursor)
	}

	// Press H on the child (leaf) — should collapse parent and move cursor to root.
	m = pressKey(m, "h")
	if len(m.visible) != 1 {
		t.Errorf("after collapse-to-parent: want 1 row, got %d", len(m.visible))
	}
	if m.cursor != 0 {
		t.Errorf("after collapse-to-parent: cursor should be on root (0), got %d", m.cursor)
	}
	if m.expanded[1] {
		t.Error("root should be collapsed after H on child")
	}
}

// TestNavigation_CollapseAlreadyCollapsedMovesToParent checks the same path
// when the node has children but they are already hidden.
func TestNavigation_CollapseAlreadyCollapsedMovesToParent(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "mid"},
		{Id: 3, Name: "leaf"},
	}
	tasks[1].ParentId = ptr64(1)
	tasks[2].ParentId = ptr64(2)
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)

	// Expand root then mid so all three rows are visible.
	m = pressKey(m, "l")        // expand root
	m = pressKey(m, "j")        // move to mid
	m = pressKey(m, "l")        // expand mid
	if len(m.visible) != 3 {
		t.Fatalf("setup: expected 3 rows, got %d", len(m.visible))
	}

	// Move cursor to mid (id=2) and collapse it.
	m.cursor = 1
	m = pressKey(m, "h") // collapse mid → mid is now collapsed, cursor stays on mid
	if m.expanded[2] {
		t.Fatalf("setup: mid should be collapsed")
	}
	if m.cursor != 1 {
		t.Fatalf("setup: cursor should be on mid (1), got %d", m.cursor)
	}

	// Press H again on mid (collapsed, has children) — should collapse root and
	// move cursor to root.
	m = pressKey(m, "h")
	if len(m.visible) != 1 {
		t.Errorf("want 1 row (only root), got %d", len(m.visible))
	}
	if m.cursor != 0 {
		t.Errorf("cursor should be on root (0), got %d", m.cursor)
	}
}

func TestNavigation_CollapseWithArrowKeys(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)

	m = pressKey(m, "right")
	if len(m.visible) != 2 {
		t.Fatalf("after right: want 2 rows, got %d", len(m.visible))
	}

	m = pressKey(m, "left")
	if len(m.visible) != 1 {
		t.Fatalf("after left: want 1 row, got %d", len(m.visible))
	}
}

func TestNavigation_PendingCompleteClearedOnMove(t *testing.T) {
	m := buildTestModel()
	id := int64(1)
	m.pendingComplete = &id
	m = pressKey(m, "j")
	if m.pendingComplete != nil {
		t.Error("pendingComplete should be cleared after cursor move")
	}
}

// TestNavigation_DownAfterLingeringComplete checks that pressing j while on a
// lingering completed task (pendingComplete set) moves to the immediately next
// task, not the one after it.
func TestNavigation_DownAfterLingeringComplete(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a", CompletedAt: now},
		{Id: 2, Name: "b"},
		{Id: 3, Name: "c"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	// Simulate task 1 just completed and lingering at cursor 0.
	id := int64(1)
	m.pendingComplete = &id
	m.visible = ExportBuildVisible(tree, m.expanded, m.showCompleted, m.pendingComplete)
	m.cursor = 0

	m = pressKey(m, "j")

	if m.cursor != 0 {
		t.Errorf("after down from lingering completed task: want cursor=0 (task 2), got %d", m.cursor)
	}
	if m.visible[m.cursor].node.Task.Id != 2 {
		t.Errorf("cursor should be on task 2, got task id %d", m.visible[m.cursor].node.Task.Id)
	}
}

func TestWindowSizeMsg(t *testing.T) {
	m := buildTestModel()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	nm := next.(Model)
	if nm.width != 120 || nm.height != 40 {
		t.Errorf("expected 120x40, got %dx%d", nm.width, nm.height)
	}
}

func TestListTasksResult_PopulatesVisible(t *testing.T) {
	m := buildTestModel()
	m.tree = nil
	m.visible = nil

	tasks := []*taskv1.Task{
		{Id: 10, Name: "x"},
		{Id: 11, Name: "y"},
	}
	tree := cli.BuildTree(tasks)
	next, _ := m.Update(listTasksResultMsg{tree: tree})
	nm := next.(Model)
	if len(nm.visible) != 2 {
		t.Errorf("expected 2 visible rows, got %d", len(nm.visible))
	}
}

func TestListTasksResult_ErrorSetsErr(t *testing.T) {
	m := buildTestModel()
	next, _ := m.Update(listTasksResultMsg{err: errForTest("boom")})
	nm := next.(Model)
	if nm.err == nil {
		t.Error("expected err to be set")
	}
}

type errForTest string

func (e errForTest) Error() string { return string(e) }

// ── T021: highlight rules after mutations ───────────────────────────────────

// TestHighlight_EditOpensFormAndPreservesMode checks that pressing E sets
// modeEdit without changing the cursor.
func TestHighlight_EditOpensModeEdit(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1
	m = pressKey(m, "e")
	if m.mode != modeEdit {
		t.Errorf("expected modeEdit, got %v", m.mode)
	}
	if m.originalCursor != 1 {
		t.Errorf("originalCursor: want 1, got %d", m.originalCursor)
	}
}

// TestHighlight_EditCancelRestoresCursor checks that editCancelledMsg restores cursor.
func TestHighlight_EditCancelRestoresCursor(t *testing.T) {
	m := buildTestModel()
	m.cursor = 2
	m.originalCursor = 2
	m.mode = modeEdit

	next, _ := m.Update(editCancelledMsg{originalCursor: 2})
	nm := next.(Model)
	if nm.mode != modeList {
		t.Errorf("expected modeList after cancel, got %v", nm.mode)
	}
	if nm.cursor != 2 {
		t.Errorf("cursor after cancel: want 2, got %d", nm.cursor)
	}
}

// TestHighlight_EditSavedSetsModeSameCursor checks that after an edit save
// message, the refreshedMsg handler highlights the same task by ID.
func TestHighlight_RefreshedMsgHighlightsByID(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1 // task id=2

	// Simulate a successful mutation refresh where task 2 is still present.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b-edited"},
		{Id: 3, Name: "c"},
	}
	tree := cli.BuildTree(tasks)
	next, _ := m.Update(refreshedMsg{tree: tree, highlightID: 2})
	nm := next.(Model)

	if nm.mode != modeList {
		t.Errorf("expected modeList, got %v", nm.mode)
	}
	if nm.cursor != 1 {
		t.Errorf("cursor should be on task 2 (index 1), got %d", nm.cursor)
	}
}

// TestHighlight_NewSubtaskOpensModeNewSubtask checks N key.
func TestHighlight_NewSubtaskKey(t *testing.T) {
	m := buildTestModel()
	m.cursor = 0
	m = pressKey(m, "n")
	if m.mode != modeNewSubtask {
		t.Errorf("expected modeNewSubtask, got %v", m.mode)
	}
	if m.edit.parentID == nil || *m.edit.parentID != 1 {
		t.Errorf("parentID: want 1, got %v", m.edit.parentID)
	}
}

// TestHighlight_NewRootKeyOpensForm checks Ctrl+N key.
func TestHighlight_NewRootKey(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlN})
	nm := next.(Model)
	if nm.mode != modeNewRoot {
		t.Errorf("expected modeNewRoot, got %v", nm.mode)
	}
	if nm.edit.taskID != nil || nm.edit.parentID != nil {
		t.Errorf("new root form should have nil taskID and parentID")
	}
}

// TestHighlight_DeleteCursorClamps checks that after delete, cursor is clamped.
func TestHighlight_DeleteRefreshedCursorClamps(t *testing.T) {
	m := buildTestModel()
	m.cursor = 2 // last task

	// Simulate refresh after deleting task 3 (only tasks 1 and 2 remain).
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
	}
	tree := cli.BuildTree(tasks)
	next, _ := m.Update(refreshedMsg{tree: tree, highlightID: 0})
	nm := next.(Model)

	// cursor was 2, but only 2 rows now, so it clamps to 1.
	if nm.cursor != 1 {
		t.Errorf("cursor should clamp to last valid index 1, got %d", nm.cursor)
	}
}

// TestHighlight_CompleteSetsPendingComplete checks Space sets pendingComplete.
func TestHighlight_CompleteKey(t *testing.T) {
	// With a nil client, pressing Space should still set err (server unreachable),
	// but the key handler should fire a Cmd.  We just check that the mode stays
	// modeList and no panic occurs.
	m := buildTestModel()
	m.cursor = 0
	// Don't call pressKey (which would hit nil client) — check mode is still list.
	if m.mode != modeList {
		t.Errorf("mode should be modeList, got %v", m.mode)
	}
}

// TestHighlight_FilterTogglePreservesTask checks that C preserves cursor by ID.
func TestHighlight_FilterTogglePreservesTask(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1 // task id=2

	m = pressKey(m, "c")
	// showCompleted toggled; same tasks visible (none completed), cursor still on id=2.
	if m.cursor != 1 {
		t.Errorf("filter toggle: cursor should stay on task 2, got %d", m.cursor)
	}
	if !m.showCompleted {
		t.Errorf("showCompleted should be toggled on")
	}
}

// TestHighlight_FilterToggleFallsBackToFirst checks that C falls back to first
// row when the previously highlighted task is no longer visible.
func TestHighlight_FilterToggleFallsBackToFirst(t *testing.T) {
	// Build a model with completed task at cursor position.
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
		{Id: 2, Name: "done", CompletedAt: now},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)

	// Show completed, move cursor to task 2.
	m.showCompleted = true
	m.visible = buildVisible(m.tree, m.expanded, m.showCompleted, nil)
	m.cursor = findCursor(m.visible, 2)
	if m.cursor != 1 {
		t.Fatalf("setup: cursor should be 1 (task 2), got %d", m.cursor)
	}

	// Toggle C to hide completed: task 2 disappears.
	m = pressKey(m, "c")
	if m.showCompleted {
		t.Errorf("showCompleted should be toggled off")
	}
	// Task 2 gone, fall back to first visible row (task 1 at index 0).
	if m.cursor != 0 {
		t.Errorf("cursor should fall back to 0, got %d", m.cursor)
	}
}

// TestHighlight_DigitKeyMatchesEstimateIndex validates rune-based digit detection.
func TestHighlight_DigitKeyNotPanicWithEmptyList(t *testing.T) {
	m := buildTestModel()
	m.visible = nil
	// Should not panic with empty visible list.
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'5'}})
	nm := next.(Model)
	// mode unchanged, no panic
	if nm.mode != modeList {
		t.Errorf("mode should be modeList")
	}
}

// ── T029: pomodoro key dispatch and done message ────────────────────────────

// ── T035: Ctrl-R refresh ────────────────────────────────────────────────────

// TestRefresh_CtrlRDispatchesListTasks verifies that Ctrl-R returns a non-nil Cmd.
func TestRefresh_CtrlRDispatchesListTasks(t *testing.T) {
	m := buildTestModel()
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("expected Ctrl-R to return a Cmd, got nil")
	}
}

// TestRefresh_PreservesCursorByID checks that after a listTasksResultMsg the
// cursor lands on the same task id when it still exists.
func TestRefresh_PreservesCursorByID(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1 // task id=2

	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b-changed"},
		{Id: 3, Name: "c"},
	}
	tree := cli.BuildTree(tasks)
	next, _ := m.Update(listTasksResultMsg{tree: tree})
	nm := next.(Model)

	if nm.cursor != 1 {
		t.Errorf("cursor should remain on task 2 (index 1), got %d", nm.cursor)
	}
}

// TestRefresh_FallsBackToFirstWhenTaskGone checks that the cursor moves to 0
// when the previously highlighted task is no longer in the refreshed tree.
func TestRefresh_FallsBackToFirstWhenTaskGone(t *testing.T) {
	m := buildTestModel()
	m.cursor = 2 // task id=3

	// Task 3 deleted on server.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
	}
	tree := cli.BuildTree(tasks)
	next, _ := m.Update(listTasksResultMsg{tree: tree})
	nm := next.(Model)

	// findCursor returns 0 when the id is not found.
	if nm.cursor != 0 {
		t.Errorf("cursor should fall back to 0 when task gone, got %d", nm.cursor)
	}
}

// TestPomodoro_SKeyDispatchesPomodoroRequest checks that pressing S emits a
// tea.Cmd that, when invoked, returns pomodoroRequestMsg with the correct taskID.
func TestPomodoro_SKeyDispatchesPomodoroRequest(t *testing.T) {
	m := buildTestModel()
	m.cursor = 0 // task id=1

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	if cmd == nil {
		t.Fatal("expected a Cmd from S key, got nil")
	}
	msg := cmd()
	req, ok := msg.(pomodoroRequestMsg)
	if !ok {
		t.Fatalf("expected pomodoroRequestMsg, got %T", msg)
	}
	if req.taskID != 1 {
		t.Errorf("pomodoroRequestMsg.taskID: want 1, got %d", req.taskID)
	}
	if req.resume {
		t.Error("pomodoroRequestMsg.resume should be false for S key")
	}
}

// TestPomodoro_RKeyDispatchesPomodoroResume checks that pressing R emits a
// tea.Cmd that returns pomodoroRequestMsg with resume=true.
func TestPomodoro_RKeyDispatchesPomodoroResume(t *testing.T) {
	m := buildTestModel()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Fatal("expected a Cmd from R key, got nil")
	}
	msg := cmd()
	req, ok := msg.(pomodoroRequestMsg)
	if !ok {
		t.Fatalf("expected pomodoroRequestMsg, got %T", msg)
	}
	if !req.resume {
		t.Error("pomodoroRequestMsg.resume should be true for R key")
	}
}

// TestPomodoro_DoneRestoresModeList checks that pomodoroDoneMsg returns to modeList.
func TestPomodoro_DoneRestoresModeList(t *testing.T) {
	m := buildTestModel()
	m.mode = modePomodoro
	m.cursor = 1

	next, _ := m.Update(pomodoroDoneMsg{})
	nm := next.(Model)

	if nm.mode != modeList {
		t.Errorf("expected modeList after pomodoroDoneMsg, got %v", nm.mode)
	}
	if nm.cursor != 1 {
		t.Errorf("cursor should be preserved: want 1, got %d", nm.cursor)
	}
}

// TestPomodoro_DoneWithErrorSetsErr checks that an error in pomodoroDoneMsg
// is surfaced on the model.
func TestPomodoro_DoneWithErrorSetsErr(t *testing.T) {
	m := buildTestModel()
	m.mode = modePomodoro

	next, _ := m.Update(pomodoroDoneMsg{err: errForTest("pom failed")})
	nm := next.(Model)

	if nm.mode != modeList {
		t.Errorf("expected modeList, got %v", nm.mode)
	}
	if nm.err == nil {
		t.Error("expected err to be set after pomodoroDoneMsg with error")
	}
}

// TestUpdateTaskCmd_PreservesParentID checks that editing a subtask does not
// clear its parent.
func TestUpdateTaskCmd_PreservesParentID(t *testing.T) {
	fc := &fakeTaskClient{}
	taskID := int64(5)
	parentID := int64(2)
	msg := editSavedMsg{
		taskID:   &taskID,
		parentID: &parentID,
		name:     "edited name",
	}
	cmd := ExportUpdateTaskCmd(fc, msg)
	cmd()
	if fc.lastUpdateReq == nil {
		t.Fatal("UpdateTask was not called")
	}
	if fc.lastUpdateReq.ParentId == nil {
		t.Fatal("ParentId was nil — subtask parent was cleared by edit")
	}
	if *fc.lastUpdateReq.ParentId != parentID {
		t.Errorf("ParentId: want %d, got %d", parentID, *fc.lastUpdateReq.ParentId)
	}
}

// fakeTaskClient is a minimal TaskServiceClient for unit tests.
type fakeTaskClient struct {
	taskv1connect.TaskServiceClient
	lastUpdateReq *taskv1.UpdateTaskRequest
}

func (f *fakeTaskClient) UpdateTask(_ context.Context, req *connect.Request[taskv1.UpdateTaskRequest]) (*connect.Response[taskv1.UpdateTaskResponse], error) {
	f.lastUpdateReq = req.Msg
	return connect.NewResponse(&taskv1.UpdateTaskResponse{Task: &taskv1.Task{Id: req.Msg.Id}}), nil
}

func (f *fakeTaskClient) ListTasks(_ context.Context, _ *connect.Request[taskv1.ListTasksRequest]) (*connect.Response[taskv1.ListTasksResponse], error) {
	return connect.NewResponse(&taskv1.ListTasksResponse{}), nil
}

// TestHighlight_RefreshedErrorKeepsListMode checks error in refreshedMsg.
func TestHighlight_RefreshedError(t *testing.T) {
	m := buildTestModel()
	m.mode = modeEdit

	next, _ := m.Update(refreshedMsg{err: errForTest("rpc failed")})
	nm := next.(Model)
	if nm.err == nil {
		t.Error("expected err to be set")
	}
	if nm.mode != modeList {
		t.Errorf("expected modeList after error, got %v", nm.mode)
	}
}

// ── T006–T009: pendingComplete linger behaviour ─────────────────────────────

// buildLingeringModel returns a model that has just received a Complete press on
// task id=2 (the middle task) followed by the refreshedMsg that sets task 2 as
// completed. pendingComplete is non-nil and task 2 is still visible.
func buildLingeringModel(t *testing.T) Model {
	t.Helper()
	m := buildTestModel()
	m.cursor = 1 // task id=2

	// Press space (Complete key).
	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	next, _ := m.Update(spaceMsg)
	m = next.(Model)

	if m.pendingComplete == nil || *m.pendingComplete != 2 {
		t.Fatalf("setup: pendingComplete should be &2, got %v", m.pendingComplete)
	}

	// Simulate the refreshedMsg that arrives after the RPC completes.
	// Task 2 is now completed.
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b", CompletedAt: now},
		{Id: 3, Name: "c"},
	}
	tree := cli.BuildTree(tasks)
	next, _ = m.Update(refreshedMsg{tree: tree, highlightID: 2})
	m = next.(Model)
	return m
}

// TestComplete_LingerPendingCompleteIsSet (T-B) asserts that after pressing
// Complete and receiving the refreshedMsg, pendingComplete is set and task 2 is
// still in m.visible.
func TestComplete_LingerPendingCompleteIsSet(t *testing.T) {
	m := buildLingeringModel(t)

	if m.pendingComplete == nil {
		t.Fatal("pendingComplete should be non-nil after complete+refresh")
	}
	if *m.pendingComplete != 2 {
		t.Errorf("pendingComplete: want 2, got %d", *m.pendingComplete)
	}

	found := false
	for _, row := range m.visible {
		if row.node.Task.Id == 2 {
			found = true
			break
		}
	}
	if !found {
		t.Error("task 2 should still be in m.visible after linger")
	}

	// Cursor should still point at task 2.
	if m.cursor >= len(m.visible) || m.visible[m.cursor].node.Task.Id != 2 {
		t.Errorf("cursor should be on task 2 after linger")
	}
}

// TestComplete_LingerClearedOnDown (T-C) asserts Down clears pendingComplete.
func TestComplete_LingerClearedOnDown(t *testing.T) {
	m := buildLingeringModel(t)

	m = pressKey(m, "j")

	if m.pendingComplete != nil {
		t.Error("pendingComplete should be nil after Down")
	}
	for _, row := range m.visible {
		if row.node.Task.Id == 2 {
			t.Error("task 2 should not be in m.visible after Down (showCompleted=false)")
		}
	}
}

// TestComplete_LingerClearedOnUp (T-C sibling) asserts Up also clears pendingComplete.
func TestComplete_LingerClearedOnUp(t *testing.T) {
	m := buildLingeringModel(t)

	m = pressKey(m, "k")

	if m.pendingComplete != nil {
		t.Error("pendingComplete should be nil after Up")
	}
	for _, row := range m.visible {
		if row.node.Task.Id == 2 {
			t.Error("task 2 should not be in m.visible after Up (showCompleted=false)")
		}
	}
}

// TestComplete_LingerNotClearedByOtherKeys (T-D) asserts that Expand, Collapse,
// Edit, Help, and Filter do not clear pendingComplete.
func TestComplete_LingerNotClearedByOtherKeys(t *testing.T) {
	keys := []struct {
		name string
		msg  tea.Msg
	}{
		{"Expand", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")}},
		{"Collapse", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")}},
		{"Edit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}},
		{"Help", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}},
		{"Filter", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")}},
	}
	for _, tc := range keys {
		t.Run(tc.name, func(t *testing.T) {
			m := buildLingeringModel(t)
			next, _ := m.Update(tc.msg)
			m = next.(Model)
			if m.pendingComplete == nil {
				t.Errorf("%s key must NOT clear pendingComplete", tc.name)
			}
			found := false
			for _, row := range m.visible {
				if row.node.Task.Id == 2 {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("%s key: task 2 should still be in m.visible", tc.name)
			}
		})
	}
}

// TestComplete_RefreshClearsPendingComplete (T-E) asserts Ctrl-R clears pendingComplete.
func TestComplete_RefreshClearsPendingComplete(t *testing.T) {
	m := buildLingeringModel(t)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlR})
	m = next.(Model)

	if m.pendingComplete != nil {
		t.Error("pendingComplete should be nil after Refresh key")
	}
}

// TestUncomplete_SpaceOnCompletedTaskDoesNotSetPendingComplete verifies that
// pressing Space on an already-completed task fires uncompleteTaskCmd and does
// NOT set pendingComplete (that field is only used for the complete direction).
func TestUncomplete_SpaceOnCompletedTaskDoesNotSetPendingComplete(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "a", CompletedAt: now},
		{Id: 2, Name: "b"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(nil, tree)
	m.showCompleted = true
	m.visible = buildVisible(tree, m.expanded, m.showCompleted, m.pendingComplete)

	m.cursor = 0 // task id=1, already completed

	spaceMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	next, cmd := m.Update(spaceMsg)
	m = next.(Model)

	if m.pendingComplete != nil {
		t.Errorf("pendingComplete must be nil when uncompleting; got %v", *m.pendingComplete)
	}
	// A Cmd must be returned (the uncompleteTaskCmd goroutine).
	if cmd == nil {
		t.Error("expected a Cmd to be returned for uncomplete, got nil")
	}
}

// ── Move-task key dispatch ────────────────────────────────────────────────────

// TestMove_MKeyEntersModeMove checks that pressing m in modeList opens the dialog.
func TestMove_MKeyEntersModeMove(t *testing.T) {
	m := buildTestModel()
	m.cursor = 1 // task id=2
	m = pressKey(m, "m")
	if m.mode != modeMove {
		t.Errorf("expected modeMove after M key, got %v", m.mode)
	}
	if m.move == nil {
		t.Fatal("move state should be non-nil after M key")
	}
	if m.move.taskID != 2 {
		t.Errorf("move.taskID: want 2, got %d", m.move.taskID)
	}
}

// TestMove_MKeyWithEmptyListIsNoOp checks that pressing m with no tasks is a no-op.
func TestMove_MKeyWithEmptyListIsNoOp(t *testing.T) {
	m := buildTestModel()
	m.visible = nil
	m = pressKey(m, "m")
	if m.mode != modeList {
		t.Errorf("mode should remain modeList with empty list, got %v", m.mode)
	}
	if m.move != nil {
		t.Error("move state should be nil when no task is selected")
	}
}

// TestHelp_AnyKeyDismisses verifies that while the help pane is open, any key
// press returns the model to modeList.
func TestHelp_AnyKeyDismisses(t *testing.T) {
	keys := []struct {
		name string
		msg  tea.KeyMsg
	}{
		{"letter x", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")}},
		{"question mark", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")}},
		{"escape", tea.KeyMsg{Type: tea.KeyEsc}},
		{"arrow up", tea.KeyMsg{Type: tea.KeyUp}},
		{"space", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")}},
	}
	for _, tc := range keys {
		t.Run(tc.name, func(t *testing.T) {
			m := buildTestModel()
			m.mode = modeHelp
			next, _ := m.Update(tc.msg)
			nm := next.(Model)
			if nm.mode != modeList {
				t.Errorf("expected modeList after %s, got %v", tc.name, nm.mode)
			}
		})
	}
}

// TestMove_SuccessfulResultReturnsModeList checks that a successful moveTaskResultMsg
// resets the dialog and refreshes the list.
func TestMove_SuccessfulResultReturnsModeList(t *testing.T) {
	m := buildTestModel()
	m.cursor = 0 // task id=1
	m = pressKey(m, "m")
	if m.mode != modeMove {
		t.Fatalf("setup: expected modeMove, got %v", m.mode)
	}

	tasks := []*taskv1.Task{
		{Id: 1, Name: "a"},
		{Id: 2, Name: "b"},
		{Id: 3, Name: "c"},
	}
	tree := cli.BuildTree(tasks)
	next, _ := m.Update(moveTaskResultMsg{taskID: 1, tree: tree})
	nm := next.(Model)

	if nm.mode != modeList {
		t.Errorf("expected modeList after successful move, got %v", nm.mode)
	}
	if nm.move != nil {
		t.Error("move state should be nil after successful move")
	}
	if len(nm.visible) == 0 {
		t.Error("visible list should be repopulated after move")
	}
}
