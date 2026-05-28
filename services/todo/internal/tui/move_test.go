package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"github.com/pboyd/todo/services/todo/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// buildMoveModel builds a model suitable for move-dialog tests.
// tree: root1(1) → child1(2), child2(3); root2(4) → child3(5)
func buildMoveModel() Model {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root1"},
		{Id: 2, Name: "child1", ParentId: ptr64(1)},
		{Id: 3, Name: "child2", ParentId: ptr64(1)},
		{Id: 4, Name: "root2"},
		{Id: 5, Name: "child3", ParentId: ptr64(4)},
	}
	tree := cli.BuildTree(tasks)
	return ExportNewModel(nil, tree)
}

// ── buildCandidates ───────────────────────────────────────────────────────────

func TestBuildCandidates_SentinelAtIndexZero(t *testing.T) {
	m := buildMoveModel()
	cands := buildCandidates(m.tree, 2)
	if len(cands) == 0 {
		t.Fatal("expected at least 1 candidate")
	}
	if cands[0].taskID != 0 {
		t.Errorf("sentinel taskID: want 0, got %d", cands[0].taskID)
	}
	if cands[0].label != "(no parent)" {
		t.Errorf("sentinel label: want '(no parent)', got %q", cands[0].label)
	}
}

func TestBuildCandidates_ExcludesMovingTask(t *testing.T) {
	m := buildMoveModel()
	cands := buildCandidates(m.tree, 2) // moving child1 (id=2)
	for _, c := range cands {
		if c.taskID == 2 {
			t.Error("candidate list must not include the moving task (id=2)")
		}
	}
}

func TestBuildCandidates_ExcludesDescendants(t *testing.T) {
	// Build a deeper tree: root(1) → mid(2) → leaf(3)
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "mid", ParentId: ptr64(1)},
		{Id: 3, Name: "leaf", ParentId: ptr64(2)},
	}
	tree := cli.BuildTree(tasks)
	// Move mid (id=2); leaf (id=3) is its descendant and must be excluded.
	cands := buildCandidates(tree, 2)
	for _, c := range cands {
		if c.taskID == 2 || c.taskID == 3 {
			t.Errorf("candidate list must not include moving task or its descendant; got taskID=%d", c.taskID)
		}
	}
	// root (id=1) must be present.
	found := false
	for _, c := range cands {
		if c.taskID == 1 {
			found = true
		}
	}
	if !found {
		t.Error("root (id=1) must appear in candidates")
	}
}

func TestBuildCandidates_ExcludesCompletedTasks(t *testing.T) {
	now := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
		{Id: 2, Name: "done", CompletedAt: now},
		{Id: 3, Name: "moving"},
	}
	tree := cli.BuildTree(tasks)
	cands := buildCandidates(tree, 3)
	for _, c := range cands {
		if c.taskID == 2 {
			t.Error("completed task (id=2) must not appear in candidates")
		}
	}
}

func TestBuildCandidates_IncludesOtherTasks(t *testing.T) {
	m := buildMoveModel()
	cands := buildCandidates(m.tree, 2) // move child1
	ids := make(map[int64]bool)
	for _, c := range cands {
		ids[c.taskID] = true
	}
	// root1(1), child2(3), root2(4), child3(5) must all be present
	for _, wantID := range []int64{1, 3, 4, 5} {
		if !ids[wantID] {
			t.Errorf("task id=%d must appear in candidates", wantID)
		}
	}
}

// ── newMoveState ──────────────────────────────────────────────────────────────

func TestNewMoveState_PreSelectsCurrentParent(t *testing.T) {
	m := buildMoveModel()
	// child1 (id=2) has parent root1 (id=1)
	ms := newMoveState(&m, 2)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	if ms.candidates[ms.cursor].taskID != 1 {
		t.Errorf("cursor should be on parent (taskID=1), got taskID=%d", ms.candidates[ms.cursor].taskID)
	}
}

func TestNewMoveState_SentinelSelectedForTopLevel(t *testing.T) {
	m := buildMoveModel()
	// root1 (id=1) has no parent
	ms := newMoveState(&m, 1)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	if ms.cursor != 0 {
		t.Errorf("top-level task: cursor should be 0 (sentinel), got %d", ms.cursor)
	}
	if ms.candidates[0].taskID != 0 {
		t.Errorf("index 0 should be sentinel, got taskID=%d", ms.candidates[0].taskID)
	}
}

func TestNewMoveState_ReturnsNilForMissingTask(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 999)
	if ms != nil {
		t.Error("newMoveState should return nil for unknown taskID")
	}
}

// ── moveState.Update navigation ───────────────────────────────────────────────

func moveMsgKey(k string) tea.Msg {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
	if k == "up" {
		msg = tea.KeyMsg{Type: tea.KeyUp}
	} else if k == "down" {
		msg = tea.KeyMsg{Type: tea.KeyDown}
	}
	return msg
}

func TestMoveState_DownMovesCursor(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 1)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	initial := ms.cursor
	_, keepOpen := ms.Update(moveMsgKey("j"), &m)
	if !keepOpen {
		t.Error("Down should keep dialog open")
	}
	if ms.cursor != initial+1 {
		t.Errorf("cursor after Down: want %d, got %d", initial+1, ms.cursor)
	}
}

func TestMoveState_UpMovesCursor(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 1)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	// Move down first so we can go up.
	ms.Update(moveMsgKey("j"), &m) //nolint
	before := ms.cursor
	_, keepOpen := ms.Update(moveMsgKey("k"), &m)
	if !keepOpen {
		t.Error("Up should keep dialog open")
	}
	if ms.cursor != before-1 {
		t.Errorf("cursor after Up: want %d, got %d", before-1, ms.cursor)
	}
}

func TestMoveState_UpClampsAtTop(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 4) // root2, cursor starts at sentinel (0)
	ms.cursor = 0
	ms.Update(moveMsgKey("k"), &m) //nolint
	if ms.cursor != 0 {
		t.Errorf("cursor should stay at 0, got %d", ms.cursor)
	}
}

func TestMoveState_DownClampsAtBottom(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 1)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	ms.cursor = len(ms.candidates) - 1
	ms.Update(moveMsgKey("j"), &m) //nolint
	if ms.cursor != len(ms.candidates)-1 {
		t.Errorf("cursor should stay at bottom, got %d", ms.cursor)
	}
}

func TestMoveState_EscClosesDialog(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 2)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	cmd, keepOpen := ms.Update(tea.KeyMsg{Type: tea.KeyEsc}, &m)
	if keepOpen {
		t.Error("Esc should close the dialog (keepOpen=false)")
	}
	if cmd != nil {
		t.Error("Esc should return nil cmd")
	}
}

func TestMoveState_EnterReturnsCmd(t *testing.T) {
	fc := &fakeTaskClient{}
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(fc, tree)

	ms := newMoveState(&m, 2) // move child (id=2), cursor on parent root (id=1)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	cmd, keepOpen := ms.Update(tea.KeyMsg{Type: tea.KeyEnter}, &m)
	if !keepOpen {
		t.Error("Enter should keep dialog open while RPC runs")
	}
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	// Execute the cmd and check that UpdateTask was called.
	result := cmd()
	msg, ok := result.(moveTaskResultMsg)
	if !ok {
		t.Fatalf("expected moveTaskResultMsg, got %T", result)
	}
	if msg.taskID != 2 {
		t.Errorf("moveTaskResultMsg.taskID: want 2, got %d", msg.taskID)
	}
	// The chosen parent is root (id=1).
	if fc.lastUpdateReq == nil {
		t.Fatal("UpdateTask was not called")
	}
	if fc.lastUpdateReq.ParentId == nil || *fc.lastUpdateReq.ParentId != 1 {
		t.Errorf("UpdateTask ParentId: want &1, got %v", fc.lastUpdateReq.ParentId)
	}
}

func TestMoveState_EnterWithSentinelSendsNilParentID(t *testing.T) {
	fc := &fakeTaskClient{}
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(fc, tree)

	ms := newMoveState(&m, 2) // move child; cursor starts on parent (root=1)
	ms.cursor = 0              // move cursor to sentinel "(no parent)"

	cmd, _ := ms.Update(tea.KeyMsg{Type: tea.KeyEnter}, &m)
	if cmd == nil {
		t.Fatal("Enter should return a Cmd")
	}
	cmd()
	if fc.lastUpdateReq == nil {
		t.Fatal("UpdateTask was not called")
	}
	if fc.lastUpdateReq.ParentId != nil {
		t.Errorf("selecting sentinel must send ParentId=nil; got %v", fc.lastUpdateReq.ParentId)
	}
}

func TestMoveState_ErrorResponsePopulatesErrMsg(t *testing.T) {
	m := buildMoveModel()
	ms := newMoveState(&m, 2)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	// Put the model into modeMove as if the user already opened the dialog.
	m.move = ms
	m.mode = modeMove

	// Simulate a failed RPC arriving as moveTaskResultMsg.
	next, _ := m.Update(moveTaskResultMsg{taskID: 2, err: errForTest("cycle detected")})
	nm := next.(Model)

	// Dialog must stay open.
	if nm.mode != modeMove {
		t.Errorf("mode should remain modeMove after error, got %v", nm.mode)
	}
	if nm.move == nil || nm.move.errMsg == "" {
		t.Error("errMsg should be set on move state after error")
	}
}

// ── US2: sentinel pre-selection for top-level tasks ─────────────────────────

func TestNewMoveState_SentinelPreSelectedForTopLevel(t *testing.T) {
	m := buildMoveModel()
	// root2 (id=4) has no parent — sentinel should be pre-selected.
	ms := newMoveState(&m, 4)
	if ms == nil {
		t.Fatal("newMoveState returned nil")
	}
	if ms.cursor != 0 {
		t.Errorf("cursor for top-level task: want 0 (sentinel), got %d", ms.cursor)
	}
}

func TestMoveState_SentinelEnterClearsParentID(t *testing.T) {
	fc := &fakeTaskClient{}
	tasks := []*taskv1.Task{
		{Id: 1, Name: "root"},
		{Id: 2, Name: "child", ParentId: ptr64(1)},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewModel(fc, tree)

	ms := newMoveState(&m, 2)
	ms.cursor = 0 // sentinel

	cmd, _ := ms.Update(tea.KeyMsg{Type: tea.KeyEnter}, &m)
	if cmd == nil {
		t.Fatal("expected a cmd")
	}
	cmd()
	if fc.lastUpdateReq == nil {
		t.Fatal("UpdateTask was not called")
	}
	if fc.lastUpdateReq.ParentId != nil {
		t.Errorf("sentinel must produce nil ParentId; got %v", fc.lastUpdateReq.ParentId)
	}
}
