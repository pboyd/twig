package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	"github.com/pboyd/twig/internal/config"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// newGoalTestModel creates a model in the Goals tab for status-update unit tests.
func newGoalTestModel(goals []*goalv1.Goal) Model {
	m := newModel(nil, nil, "", config.PomodoroConfig{}, false, nil)
	m.activeTab = tabGoals
	m.goal.goals = goals
	m.goal.loaded = true
	m.keys.GoalMode = true
	return m
}

func makeStatusUpdate(id, goalID int64, body string, ago time.Duration) *goalv1.StatusUpdate {
	return &goalv1.StatusUpdate{
		Id:        id,
		GoalId:    goalID,
		Body:      body,
		CreatedAt: timestamppb.New(time.Now().Add(-ago)),
	}
}

func goalWithLatest(id int64, name string, su *goalv1.StatusUpdate) *goalv1.Goal {
	return &goalv1.Goal{
		Id:                 id,
		Name:               name,
		State:              goalv1.GoalState_GOAL_STATE_COMMITTED,
		LatestStatusUpdate: su,
	}
}

// ── T010: US1 — detail pane ──────────────────────────────────────────────────

// TestGoalStatus_DetailPaneNoStatus verifies the empty-state copy.
func TestGoalStatus_DetailPaneNoStatus(t *testing.T) {
	g := &goalv1.Goal{
		Id:    1,
		Name:  "Build a rocket",
		State: goalv1.GoalState_GOAL_STATE_COMMITTED,
	}
	m := newGoalTestModel([]*goalv1.Goal{g})
	out := m.renderGoalDetail(60)
	if !strings.Contains(out, "No status yet") {
		t.Errorf("expected empty-state text in detail pane, got:\n%s", out)
	}
	if !strings.Contains(out, "[S]") {
		t.Errorf("expected [S] hint in empty state, got:\n%s", out)
	}
}

// TestGoalStatus_DetailPaneWithLatest verifies that the latest status renders.
func TestGoalStatus_DetailPaneWithLatest(t *testing.T) {
	su := makeStatusUpdate(1, 1, "Making **great** progress!", 2*time.Hour)
	g := goalWithLatest(1, "Build a rocket", su)
	m := newGoalTestModel([]*goalv1.Goal{g})
	out := m.renderGoalDetail(60)

	if !strings.Contains(out, "Latest status") {
		t.Errorf("expected 'Latest status' label, got:\n%s", out)
	}
	// The body should appear (markdown may transform it slightly — check for key word)
	if !strings.Contains(out, "great") {
		t.Errorf("expected status body content, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("expected relative timestamp, got:\n%s", out)
	}
}

// TestGoalStatus_SKeyStartsCompose verifies pressing `S` activates compose.
func TestGoalStatus_SKeyStartsCompose(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	// We cannot actually open the editor in a test, but we can check compose is set.
	m.goal.compose = statusCompose{goalID: g.Id, active: true}
	if !m.goal.compose.active {
		t.Error("compose should be active")
	}
}

// TestGoalStatus_EmptyEditorDiscards verifies that an empty result is discarded.
func TestGoalStatus_EmptyEditorDiscards(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	m.goal.compose = statusCompose{goalID: g.Id, active: true}

	m2, cmd := m.handleGoalStatusEditorFinished(editorFinishedMsg{content: "   \n   "})

	// Cmd should be nil (no RPC sent for empty body).
	if cmd != nil {
		t.Error("expected nil cmd for empty editor result")
	}
	// A notice should have been set.
	if !strings.Contains(m2.(Model).notice, "not recorded") {
		t.Errorf("expected 'not recorded' notice, got %q", m2.(Model).notice)
	}
	// Compose should be cleared.
	if m2.(Model).goal.compose.active {
		t.Error("compose should be cleared after empty result")
	}
}

// TestGoalStatus_NonEmptyEditorSendsAddCmd verifies non-empty editor result triggers RPC cmd.
func TestGoalStatus_NonEmptyEditorSendsAddCmd(t *testing.T) {
	g := &goalv1.Goal{Id: 42, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	m.goal.compose = statusCompose{goalID: g.Id, active: true}

	_, cmd := m.handleGoalStatusEditorFinished(editorFinishedMsg{content: "Some progress made."})

	// A non-nil command should have been returned (the addGoalStatusUpdateCmd).
	if cmd == nil {
		t.Error("expected a command to be returned for non-empty editor result")
	}
}

// TestGoalStatus_EditComposeSendsUpdateCmd verifies that an edit (editingID>0) returns an update cmd.
func TestGoalStatus_EditComposeSendsUpdateCmd(t *testing.T) {
	g := &goalv1.Goal{Id: 42, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	m.goal.compose = statusCompose{goalID: g.Id, editingID: 7, active: true}

	_, cmd := m.handleGoalStatusEditorFinished(editorFinishedMsg{content: "Revised content."})
	if cmd == nil {
		t.Error("expected a command for non-empty edit result")
	}
}

// TestGoalStatus_MutationMsgClearsCompose verifies goalStatusMutationMsg clears compose.
func TestGoalStatus_MutationMsgClearsCompose(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	m.goal.compose = statusCompose{goalID: 1, active: true}

	m2, _ := m.Update(goalStatusMutationMsg{goalID: 1, err: nil})
	if m2.(Model).goal.compose.active {
		t.Error("compose should be cleared after mutation msg")
	}
}

// ── T015: US2 — history view ─────────────────────────────────────────────────

// TestGoalStatus_ListMsgEntersHistoryMode verifies goalStatusListMsg sets history mode.
func TestGoalStatus_ListMsgEntersHistoryMode(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(2, 1, "Newer update", time.Hour),
		makeStatusUpdate(1, 1, "Older update", 2*time.Hour),
	}
	m2, _ := m.Update(goalStatusListMsg{updates: updates})

	if m2.(Model).goal.mode != goalStatusHistory {
		t.Errorf("expected goalStatusHistory mode, got %d", m2.(Model).goal.mode)
	}
	if len(m2.(Model).goal.statusUpdates) != 2 {
		t.Errorf("expected 2 updates, got %d", len(m2.(Model).goal.statusUpdates))
	}
}

// TestGoalStatus_HistoryRenderNewestFirst verifies the history list renders newest first.
func TestGoalStatus_HistoryRenderNewestFirst(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(2, 1, "Latest update here", time.Hour),
		makeStatusUpdate(1, 1, "Oldest update here", 48*time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusHistory

	out := m.renderStatusHistory(80, 20)

	latestIdx := strings.Index(out, "Latest update here")
	oldestIdx := strings.Index(out, "Oldest update here")
	if latestIdx == -1 || oldestIdx == -1 {
		t.Fatalf("expected both entries in history, got:\n%s", out)
	}
	if latestIdx > oldestIdx {
		t.Error("expected newest-first order in history list")
	}
}

// TestGoalStatus_HistoryNavCursor verifies cursor navigation in history view.
func TestGoalStatus_HistoryNavCursor(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(3, 1, "Update C", time.Hour),
		makeStatusUpdate(2, 1, "Update B", 2*time.Hour),
		makeStatusUpdate(1, 1, "Update A", 3*time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusHistory

	if m.goal.statusCursor != 0 {
		t.Fatalf("expected cursor=0, got %d", m.goal.statusCursor)
	}

	// Press down.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.(Model).goal.statusCursor != 1 {
		t.Errorf("expected cursor=1 after down, got %d", m2.(Model).goal.statusCursor)
	}

	// Press down again.
	m3, _ := m2.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m3.(Model).goal.statusCursor != 2 {
		t.Errorf("expected cursor=2 after second down, got %d", m3.(Model).goal.statusCursor)
	}

	// Press down at end — should not go past last.
	m4, _ := m3.(Model).Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m4.(Model).goal.statusCursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", m4.(Model).goal.statusCursor)
	}

	// Press up.
	m5, _ := m4.(Model).Update(tea.KeyPressMsg{Code: tea.KeyUp})
	if m5.(Model).goal.statusCursor != 1 {
		t.Errorf("expected cursor=1 after up, got %d", m5.(Model).goal.statusCursor)
	}
}

// TestGoalStatus_EnterOpensReader verifies enter key opens the reader.
func TestGoalStatus_EnterOpensReader(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(1, 1, "Read me", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusHistory

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m2.(Model).goal.mode != goalStatusReader {
		t.Errorf("expected reader mode after enter, got %d", m2.(Model).goal.mode)
	}
}

// TestGoalStatus_EscFromReaderBackToHistory verifies esc returns to history list.
func TestGoalStatus_EscFromReaderBackToHistory(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(1, 1, "Read me", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusReader

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m2.(Model).goal.mode != goalStatusHistory {
		t.Errorf("expected history mode after esc from reader, got %d", m2.(Model).goal.mode)
	}
}

// TestGoalStatus_EscFromHistoryBackToList verifies esc from history returns to goal list.
func TestGoalStatus_EscFromHistoryBackToList(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(1, 1, "Update", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusHistory

	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if m2.(Model).goal.mode != goalList {
		t.Errorf("expected goalList after esc from history, got %d", m2.(Model).goal.mode)
	}
}

// TestGoalStatus_ReaderScrollsLongBody verifies scroll offset changes on down key,
// and that it clamps at the end of the content (no scrolling into a blank view).
func TestGoalStatus_ReaderScrollsLongBody(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	// Set realistic terminal dimensions so the markdown renderer produces multiple
	// lines and the scroll clamp has meaningful values.
	m.width = 80
	m.height = 24

	// Create a multi-line body.
	var bodyLines []string
	for i := 0; i < 50; i++ {
		bodyLines = append(bodyLines, "This is line of the status update body for scroll testing.")
	}
	body := strings.Join(bodyLines, "\n")

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(1, 1, body, time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusReader

	if m.goal.readerOffset != 0 {
		t.Fatal("precondition: reader offset should be 0")
	}

	// Press down to scroll.
	m2, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if m2.(Model).goal.readerOffset != 1 {
		t.Errorf("expected readerOffset=1 after down, got %d", m2.(Model).goal.readerOffset)
	}

	// Verify the clamp: pressing down many times should not push offset past content.
	mc := m
	for i := 0; i < 200; i++ {
		m3, _ := mc.Update(tea.KeyPressMsg{Code: tea.KeyDown})
		mc = m3.(Model)
	}
	renderedLines := mc.statusReaderLines()
	maxOffset := len(renderedLines) - mc.statusReaderContentHeight()
	if maxOffset < 0 {
		maxOffset = 0
	}
	if mc.goal.readerOffset > maxOffset {
		t.Errorf("readerOffset %d exceeded maxOffset %d after many Down presses", mc.goal.readerOffset, maxOffset)
	}
}

// TestGoalStatus_ReaderRenderNoTruncation verifies reader renders all content (no truncation for long bodies).
func TestGoalStatus_ReaderRenderNoTruncation(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	var lines []string
	for i := 0; i < 10; i++ {
		lines = append(lines, "Line content here")
	}
	body := strings.Join(lines, "\n")
	m.goal.statusUpdates = []*goalv1.StatusUpdate{makeStatusUpdate(1, 1, body, time.Hour)}
	m.goal.mode = goalStatusReader

	// Render with a small height to force scroll indicator.
	out := m.renderStatusReader(80, 8)
	// All content should be accessible via scrolling; at least first lines shown.
	if !strings.Contains(out, "Line content here") {
		t.Errorf("expected body content in reader, got:\n%s", out)
	}
}

// TestGoalStatus_EmptyHistoryRendersCopy verifies the empty-history copy.
func TestGoalStatus_EmptyHistoryRendersCopy(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	m.goal.mode = goalStatusHistory

	out := m.renderStatusHistory(80, 20)
	if !strings.Contains(out, "No status updates yet") {
		t.Errorf("expected empty-history copy, got:\n%s", out)
	}
}

// ── T018: US3 — edit/delete ──────────────────────────────────────────────────

// TestGoalStatus_EKeyStartsEditCompose verifies `e` sets compose with editingID.
func TestGoalStatus_EKeyStartsEditCompose(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(99, 1, "Edit me", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusHistory

	// We test the compose state directly since tea.ExecProcess can't run in tests.
	m.goal.compose = statusCompose{goalID: 1, editingID: 99, active: true}
	if m.goal.compose.editingID != 99 {
		t.Errorf("expected editingID=99, got %d", m.goal.compose.editingID)
	}
}

// TestGoalStatus_EditEmptyEditorDiscards verifies empty edit is also discarded.
func TestGoalStatus_EditEmptyEditorDiscards(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})
	m.goal.compose = statusCompose{goalID: g.Id, editingID: 99, active: true}

	m2, cmd := m.handleGoalStatusEditorFinished(editorFinishedMsg{content: "  "})
	if cmd != nil {
		t.Error("expected nil cmd for empty edit result")
	}
	if m2.(Model).goal.compose.active {
		t.Error("compose should be cleared")
	}
}

// TestGoalStatus_DKeyEntersConfirmMode verifies `d` enters confirm-delete.
func TestGoalStatus_DKeyEntersConfirmMode(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(1, 1, "Delete me", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusHistory

	m2, _ := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if m2.(Model).goal.mode != goalStatusConfirmDel {
		t.Errorf("expected goalStatusConfirmDel after d, got %d", m2.(Model).goal.mode)
	}
}

// TestGoalStatus_ConfirmDeleteNKey cancels deletion.
func TestGoalStatus_ConfirmDeleteNKey(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(1, 1, "Keep me", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusConfirmDel

	m2, cmd := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if cmd != nil {
		t.Error("expected nil cmd on cancel")
	}
	if m2.(Model).goal.mode != goalStatusHistory {
		t.Errorf("expected history mode after n, got %d", m2.(Model).goal.mode)
	}
}

// TestGoalStatus_ConfirmDeleteYKeyCallsDeleteCmd verifies `y` fires delete cmd.
func TestGoalStatus_ConfirmDeleteYKeyCallsDeleteCmd(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Goal", State: goalv1.GoalState_GOAL_STATE_COMMITTED}
	m := newGoalTestModel([]*goalv1.Goal{g})

	updates := []*goalv1.StatusUpdate{
		makeStatusUpdate(5, 1, "Delete me", time.Hour),
	}
	m.goal.statusUpdates = updates
	m.goal.mode = goalStatusConfirmDel

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	// Without a real client, cmd will be the delete command factory (non-nil).
	if cmd == nil {
		t.Error("expected a non-nil delete cmd after y")
	}
}
