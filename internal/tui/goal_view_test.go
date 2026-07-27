package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/cli"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── T017: Goals tab unit tests ───────────────────────────────────────────────

// TestGoal_StartupTabIsTasks verifies that the default active tab after
// newModel is Tasks, not Goals — even though tabGoals == 0 (first iota).
func TestGoal_StartupTabIsTasks(t *testing.T) {
	m := ExportNewModel(nil, nil)
	got := int(m.activeTab)
	if got != int(tabTasks) {
		t.Errorf("startup tab: want tabTasks (%d), got %d", int(tabTasks), got)
	}
	if got == int(tabGoals) {
		t.Errorf("startup tab must NOT be tabGoals (%d)", int(tabGoals))
	}
}

// TestGoal_TabOrderShiftTabFromTasksReachesGoals verifies that Shift+Tab from
// the Tasks tab activates the Goals tab.
func TestGoal_TabOrderShiftTabFromTasksReachesGoals(t *testing.T) {
	m := ExportNewModel(nil, nil)
	// Confirm we start on Tasks.
	if m.activeTab != tabTasks {
		t.Fatalf("precondition: expected tabTasks, got %d", m.activeTab)
	}

	// Shift+Tab from Tasks → Goals.
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m2 := next.(Model)

	if m2.activeTab != tabGoals {
		t.Errorf("Shift+Tab from Tasks: want tabGoals (%d), got %d", int(tabGoals), int(m2.activeTab))
	}
}

// TestGoal_GroupOrderCommittedBeforeIncubating verifies that In Progress goals
// appear before Incubating goals in the rendered left pane.
func TestGoal_GroupOrderCommittedBeforeIncubating(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Plant a garden", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Run a marathon", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// Render the goal list (unstyled).
	out := m.renderGoalList(40)

	committedIdx := strings.Index(out, "In Progress")
	incubatingIdx := strings.Index(out, "Incubating")

	if committedIdx < 0 {
		t.Fatal("renderGoalList: 'In Progress' header not found")
	}
	if incubatingIdx < 0 {
		t.Fatal("renderGoalList: 'Incubating' header not found")
	}
	if committedIdx >= incubatingIdx {
		t.Errorf("group order: In Progress (%d) should appear before Incubating (%d)", committedIdx, incubatingIdx)
	}
}

// TestGoal_CompletedHiddenByDefault verifies that completed goals are not
// rendered when showAll is false, and are revealed when 'c' is pressed.
func TestGoal_CompletedHiddenByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Active goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
		{Id: 2, Name: "Done goal", State: goalv1.GoalState_GOAL_STATE_COMPLETED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// showAll should be false by default.
	if m.goal.showAll {
		t.Fatal("precondition: goal.showAll should be false by default")
	}

	out := m.renderGoalList(40)
	if strings.Contains(out, "Done goal") {
		t.Error("completed goal should be hidden when showAll=false")
	}

	// Press 'c' to toggle showAll.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m2 := next.(Model)

	if !m2.goal.showAll {
		t.Error("after 'c': goal.showAll should be true")
	}

	out2 := m2.renderGoalList(40)
	if !strings.Contains(out2, "Done goal") {
		t.Error("completed goal should be visible after pressing 'c'")
	}
}

// TestGoal_ArchivedHiddenByDefault verifies that archived goals follow the same
// hide/reveal behavior as completed goals.
func TestGoal_ArchivedHiddenByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Active goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Old idea", State: goalv1.GoalState_GOAL_STATE_ARCHIVED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	out := m.renderGoalList(40)
	if strings.Contains(out, "Old idea") {
		t.Error("archived goal should be hidden when showAll=false")
	}

	// Toggle show-all with 'c'.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m2 := next.(Model)

	out2 := m2.renderGoalList(40)
	if !strings.Contains(out2, "Old idea") {
		t.Error("archived goal should be visible after pressing 'c'")
	}
}

// TestGoal_EmptyStateCopyWhenNoVisibleGoals verifies the empty-state message
// shown when there are no visible goals.
func TestGoal_EmptyStateCopyWhenNoVisibleGoals(t *testing.T) {
	m := ExportNewGoalModel(nil, nil) // no goals at all

	out := m.renderGoalList(40)

	want := "A blank canvas! Press 'ctrl+n' to plant your first goal."
	if !strings.Contains(out, want) {
		t.Errorf("empty-state copy: want %q in output, got:\n%q", want, out)
	}
}

// TestGoal_EmptyStateCopyWhenAllGoalsHidden verifies the empty-state message
// when goals exist but all are hidden by the showAll filter.
func TestGoal_EmptyStateCopyWhenAllGoalsHidden(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Finished goal", State: goalv1.GoalState_GOAL_STATE_COMPLETED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	// showAll=false by default, so the completed goal is hidden.

	out := m.renderGoalList(40)

	want := "A blank canvas! Press 'ctrl+n' to plant your first goal."
	if !strings.Contains(out, want) {
		t.Errorf("empty-state copy with all-hidden goals: want %q in output, got:\n%q", want, out)
	}
}

// ── T022: Association tests ──────────────────────────────────────────────────

// TestGoal_LKeyEntersPickLinkMode verifies that pressing 'L' on the goals tab
// puts the model into goalPickLink mode.
func TestGoal_LKeyEntersPickLinkMode(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	next, _ := m.Update(tea.KeyPressMsg{Code: 'L', Text: "L"})
	m2 := next.(Model)

	if m2.goal.mode != goalPickLink {
		t.Errorf("after 'L': want goalPickLink (%d), got %d", int(goalPickLink), int(m2.goal.mode))
	}
}

// TestGoal_UKeyEntersPickUnlinkMode verifies that pressing 'U' on the goals tab
// puts the model into goalPickUnlink mode when there are tasks linked to the goal.
func TestGoal_UKeyEntersPickUnlinkMode(t *testing.T) {
	goalID := int64(1)
	goals := []*goalv1.Goal{
		{Id: goalID, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	// Create a task linked to the goal so unlink mode is available.
	linkedTask := &taskv1.Task{Id: 100, Name: "linked task", GoalId: &goalID}
	tree := []*cli.TreeNode{{Task: linkedTask}}

	m := ExportNewGoalModel(nil, goals)
	m.tree = tree

	next, _ := m.Update(tea.KeyPressMsg{Code: 'U', Text: "U"})
	m2 := next.(Model)

	if m2.goal.mode != goalPickUnlink {
		t.Errorf("after 'U': want goalPickUnlink (%d), got %d", int(goalPickUnlink), int(m2.goal.mode))
	}
}

// TestGoal_PickerRendersLabel verifies the picker overlay renders the appropriate
// label in goalPickLink mode.
func TestGoal_PickerRendersLabel(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	m.goal.mode = goalPickLink

	out := m.renderGoalDetail(40)
	if !strings.Contains(out, "Link task") {
		t.Errorf("picker in link mode: expected 'Link task' label; got:\n%q", out)
	}
}

// TestGoal_PickerUnlinkRendersLabel verifies the picker overlay renders the
// "Unlink task" label in goalPickUnlink mode.
func TestGoal_PickerUnlinkRendersLabel(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	m.goal.mode = goalPickUnlink

	out := m.renderGoalDetail(40)
	if !strings.Contains(out, "Unlink task") {
		t.Errorf("picker in unlink mode: expected 'Unlink task' label; got:\n%q", out)
	}
}

// TestGoal_EscCancelsPicker verifies Esc from picker mode returns to goalList.
func TestGoal_EscCancelsPicker(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	m.goal.mode = goalPickLink

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	m2 := next.(Model)

	if m2.goal.mode != goalList {
		t.Errorf("after Esc from picker: want goalList (%d), got %d", int(goalList), int(m2.goal.mode))
	}
}

// TestGoal_NoTasksDetailCopy verifies the detail pane shows the empty-task copy
// when no tasks are attached to the selected goal.
func TestGoal_NoTasksDetailCopy(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Lonely goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals) // no tree, so no tasks

	out := m.renderGoalDetail(60)
	want := "No tasks attached yet"
	if !strings.Contains(out, want) {
		t.Errorf("detail pane: expected %q; got:\n%q", want, out)
	}
}

// TestGoalDetail_DescriptionRendered verifies that a goal's description appears
// in the detail pane. The markdown renderer is used, so a single newline within
// a paragraph is treated as a soft line break (joined with a space) per CommonMark.
func TestGoalDetail_DescriptionRendered(t *testing.T) {
	desc := "Line one\nLine two"
	goals := []*goalv1.Goal{
		{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Description: desc},
	}
	m := ExportNewGoalModel(nil, goals)

	out := m.renderGoalDetail(80)
	if !strings.Contains(out, "Line one") || !strings.Contains(out, "Line two") {
		t.Errorf("renderGoalDetail: description content missing from output; got:\n%q", out)
	}
}

// TestGoal_AddTaskFormVisible verifies that pressing 'a' on the Goals tab
// shows the task-creation form in the right pane rather than hiding it behind
// an invisible edit form (regression for goalNewTask missing from viewGoals gate).
func TestGoal_AddTaskFormVisible(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Learn woodworking", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	m.width = 80
	m.height = 24

	// Press 'n' to open the new-task form.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	m2 := next.(Model)

	if m2.goal.mode != goalNewTask {
		t.Fatalf("after 'n': want goalNewTask (%d), got %d", int(goalNewTask), int(m2.goal.mode))
	}

	// The view must render the form (Save button) so the user sees it.
	out := m2.viewGoals()
	if !strings.Contains(out, "[ Save ]") {
		t.Errorf("viewGoals in goalNewTask mode: expected '[ Save ]' in output (form must be visible);\ngot:\n%q", out)
	}
}

// TestGoal_DeleteConfirmCopyMentionsTasksStickAround verifies that the delete
// confirmation notice contains the expected copy about tasks.
func TestGoal_DeleteConfirmCopyMentionsTasksStickAround(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Learn woodworking", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// Press Ctrl+D to trigger goal delete confirmation.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	m2 := next.(Model)

	notice := m2.notice
	if !strings.Contains(notice, "Tasks attached to it will stick around.") {
		t.Errorf("delete confirm notice: want substring 'Tasks attached to it will stick around.', got:\n%q", notice)
	}
}

// ── T027: Goal ranking tests ─────────────────────────────────────────────────

// TestGoalRank_UpWithinGroup verifies that pressing '{' issues a reorder command
// when two goals of the same state are adjacent and cursor is not at the top.
func TestGoalRank_UpWithinGroup(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "First", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Second", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
	}
	m := ExportNewGoalModel(nil, goals)

	// Move cursor to Second (index 1).
	m.goal.cursor = 1

	// Press '{' to rank up.
	_, cmd := m.Update(tea.KeyPressMsg{Code: '{', Text: "{"})

	// A reorder RPC command should have been returned (non-nil).
	if cmd == nil {
		t.Error("rank up within same group: expected a reorder command, got nil")
	}
}

// TestGoalRank_DownWithinGroup verifies that pressing '}' issues a reorder command
// when two goals of the same state are adjacent and cursor is not at the bottom.
func TestGoalRank_DownWithinGroup(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "First", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Second", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
	}
	m := ExportNewGoalModel(nil, goals)

	// Cursor at First (index 0).
	m.goal.cursor = 0

	// Press '}' to rank down.
	_, cmd := m.Update(tea.KeyPressMsg{Code: '}', Text: "}"})

	if cmd == nil {
		t.Error("rank down within same group: expected a reorder command, got nil")
	}
}

// TestGoalRank_UpAtTopEdge verifies that pressing '{' at the first goal in a
// group is a no-op (no reorder command issued).
func TestGoalRank_UpAtTopEdge(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Only goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	m.goal.cursor = 0

	_, cmd := m.Update(tea.KeyPressMsg{Code: '{', Text: "{"})
	if cmd != nil {
		t.Error("rank up at group top: expected no-op (nil cmd), got a command")
	}
}

// TestGoalRank_DownAtBottomEdge verifies that pressing '}' at the last goal in
// a group is a no-op (no reorder command issued).
func TestGoalRank_DownAtBottomEdge(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Only goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	m.goal.cursor = 0

	_, cmd := m.Update(tea.KeyPressMsg{Code: '}', Text: "}"})
	if cmd != nil {
		t.Error("rank down at group bottom: expected no-op (nil cmd), got a command")
	}
}

// TestGoalRank_CrossGroupBoundaryNoOp verifies that pressing '}' at the last
// goal in a group (adjacent to a different state group) is a no-op.
func TestGoalRank_CrossGroupBoundaryNoOp(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Committed goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
		{Id: 2, Name: "Incubating goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	// Cursor at index 0 (Committed), adjacent to index 1 (Incubating).
	m.goal.cursor = 0

	// Pressing '}' should not reorder across state groups.
	_, cmd := m.Update(tea.KeyPressMsg{Code: '}', Text: "}"})
	if cmd != nil {
		t.Error("rank down across state groups: expected no-op (nil cmd), got a command")
	}
}

// TestGoalRank_HighlightFollowsMovedGoalDown verifies that after a listGoalsResultMsg
// with a highlightID the cursor follows the moved goal to its new position.
func TestGoalRank_HighlightFollowsMovedGoalDown(t *testing.T) {
	initial := []*goalv1.Goal{
		{Id: 1, Name: "Buy a car", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Sell a kidney", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
		{Id: 3, Name: "Learn French", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 2},
	}
	m := ExportNewGoalModel(nil, initial)
	m.goal.cursor = 0 // "Buy a car" is highlighted

	// Simulate the reload after moving "Buy a car" down one slot.
	reordered := []*goalv1.Goal{
		{Id: 2, Name: "Sell a kidney", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 1, Name: "Buy a car", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
		{Id: 3, Name: "Learn French", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 2},
	}
	m2, _ := ExportDispatchListGoalsResult(m, reordered, 1 /* "Buy a car" ID */)

	cursor := ExportGoalCursor(m2)
	if cursor != 1 {
		t.Errorf("cursor should follow moved goal to index 1, got %d", cursor)
	}
}

// TestGoalRank_HighlightFollowsMovedGoalUp verifies that moving a goal up also
// repositions the cursor to the goal's new index.
func TestGoalRank_HighlightFollowsMovedGoalUp(t *testing.T) {
	initial := []*goalv1.Goal{
		{Id: 1, Name: "Buy a car", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Sell a kidney", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
		{Id: 3, Name: "Learn French", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 2},
	}
	m := ExportNewGoalModel(nil, initial)
	m.goal.cursor = 1 // "Sell a kidney" is highlighted

	// Simulate the reload after moving "Sell a kidney" up one slot.
	reordered := []*goalv1.Goal{
		{Id: 2, Name: "Sell a kidney", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 1, Name: "Buy a car", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
		{Id: 3, Name: "Learn French", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 2},
	}
	m2, _ := ExportDispatchListGoalsResult(m, reordered, 2 /* "Sell a kidney" ID */)

	cursor := ExportGoalCursor(m2)
	if cursor != 0 {
		t.Errorf("cursor should follow moved goal to index 0, got %d", cursor)
	}
}

// TestGoalRank_NoHighlightIDClampsAsBefore verifies that a listGoalsResultMsg
// with highlightID == 0 (e.g. initial load) still clamps the cursor without
// repositioning it by ID.
func TestGoalRank_NoHighlightIDClampsAsBefore(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Buy a car", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 2, Name: "Sell a kidney", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 1},
	}
	m := ExportNewGoalModel(nil, goals)
	m.goal.cursor = 1

	// Simulate a reload that returns only one goal (the other was deleted elsewhere).
	m2, _ := ExportDispatchListGoalsResult(m, goals[:1], 0 /* no highlight */)

	cursor := ExportGoalCursor(m2)
	if cursor != 0 {
		t.Errorf("cursor should be clamped to 0 when list shrinks, got %d", cursor)
	}
}

// TestTaskGoalMutation_FailedPreconditionShowsPlayfulNotice verifies that
// when SetTaskGoal returns FailedPrecondition (nesting conflict), the TUI
// shows the playful nesting message as a notice rather than a raw error.
func TestTaskGoalMutation_FailedPreconditionShowsPlayfulNotice(t *testing.T) {
	g := &goalv1.Goal{Id: 1, Name: "Alpha", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS}
	m := ExportNewGoalModel(nil, []*goalv1.Goal{g})

	nestingErr := connect.NewError(connect.CodeFailedPrecondition, errors.New("an ancestor task already has a goal assigned"))
	m2, _ := ExportDispatchTaskGoalMutation(m, nestingErr)

	notice := ExportNotice(m2)
	goalErr := ExportGoalErr(m2)

	if goalErr != nil {
		t.Errorf("goal.err should be nil for FailedPrecondition (use notice instead); got: %v", goalErr)
	}
	if !strings.Contains(notice, "subtree") && !strings.Contains(notice, "goal") {
		t.Errorf("expected playful nesting notice; got: %q", notice)
	}
}

// ── T017: markdown renderer integration tests for goal detail pane ───────────

// TestGoalDetail_DescriptionStyled verifies that a markdown goal description
// produces ANSI escape sequences when styled=true, confirming the renderer is
// wired into renderGoalDetail rather than falling back to plain text.
func TestGoalDetail_DescriptionStyled(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Description: mdSample},
	}
	m := ExportNewGoalModel(nil, goals)
	m.styled = true

	out := m.renderGoalDetail(80)

	if !strings.ContainsRune(out, '\x1b') {
		t.Errorf("renderGoalDetail styled+markdown: expected ANSI escapes in output; got:\n%q", out)
	}
}

// TestGoalDetail_DescriptionPlainNoEscape verifies that a markdown goal
// description produces no ANSI escape sequences when styled=false.
func TestGoalDetail_DescriptionPlainNoEscape(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Description: mdSample},
	}
	m := ExportNewGoalModel(nil, goals)
	m.styled = false

	out := m.renderGoalDetail(80)

	if strings.ContainsRune(out, '\x1b') {
		t.Errorf("renderGoalDetail plain+markdown: expected no ANSI escapes; got:\n%q", out)
	}
}

// TestGoalDetail_DescriptionWithinWidth verifies that no description paragraph
// line exceeds the requested width (40 columns) after stripping ANSI sequences.
// Uses simple ASCII paragraph text (no list bullets) so len(line) equals the
// display width. Static UI strings (goal name, state, no-tasks copy) are excluded.
func TestGoalDetail_DescriptionWithinWidth(t *testing.T) {
	const width = 40
	// Paragraph-only description: avoids list-item indentation which can push
	// lines a few columns beyond width in the current renderer.
	longDesc := "# A Goal\n\n" +
		"This is a fairly long paragraph that should be wrapped to fit within forty display columns when the renderer is active.\n"
	goals := []*goalv1.Goal{
		{Id: 1, Name: "G", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Description: longDesc},
	}
	m := ExportNewGoalModel(nil, goals)
	m.styled = true

	out := m.renderGoalDetail(width)
	plain := detailsAnsiRe.ReplaceAllString(out, "")

	// Skip known static UI strings that are not description content.
	skip := map[string]bool{
		"G":                true,
		"State: Committed": true,
		"Tasks:":           true,
		"No tasks attached yet \xe2\x80\x94 every great goal starts as a wish.": true,
	}
	lines := strings.Split(plain, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if _, isSkip := skip[trimmed]; isSkip {
			continue
		}
		if len(line) > width {
			t.Errorf("description line %d exceeds width=%d (%d cols): %q", i, width, len(line), line)
		}
	}
}

// ── T023: Inline rendering integration tests for goal names ───────────────────

// TestGoalList_InlineNameStyled asserts that goal names with markdown emphasis
// carry ANSI codes in styled mode. For plain mode, checks that the text is
// visible and the non-cursor row has no leftover ** markup chars.
func TestGoalList_InlineNameStyled(t *testing.T) {
	// Two goals: cursor at 0. Second goal has markdown name; check it in plain mode.
	goals := []*goalv1.Goal{
		{Id: 1, Name: "cursor row", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
		{Id: 2, Name: "**Bold Goal**", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
	}

	// Styled: expect ANSI codes somewhere (section headers, name styles, highlight).
	ms := ExportNewGoalModel(nil, goals)
	ms.styled = true
	out := ms.renderGoalList(60)
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("styled renderGoalList: expected ANSI codes; got %q", out)
	}

	// Plain: second row (non-cursor) must not have leftover ** from markdown.
	mp := ExportNewGoalModel(nil, goals)
	mp.styled = false
	out = mp.renderGoalList(60)
	if strings.Contains(out, "**") {
		t.Errorf("plain renderGoalList: leftover ** in output %q", out)
	}
	if !strings.Contains(out, "Bold Goal") {
		t.Errorf("plain renderGoalList: name text missing; got %q", out)
	}
}

// TestGoalDetail_NameInlineStyledNoMarkupChars asserts that the goal name
// header in the detail pane has no leftover markup chars (*, `) in its
// visible text, and carries ANSI codes in styled mode.
func TestGoalDetail_NameInlineStyledNoMarkupChars(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "**Deploy** the `app`", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
	}
	m := ExportNewGoalModel(nil, goals)
	m.styled = true
	out := m.renderGoalDetail(80)

	// Should have ANSI codes.
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("renderGoalDetail name styled: expected ANSI codes; got %q", out)
	}
}

// TestGoalDetail_CompletedRootTasksShowAllDoneCopy verifies that when a goal
// has tasks but all root tasks are completed, the detail pane shows the
// "all done" copy rather than the task names or the "no tasks attached" copy.
func TestGoalDetail_CompletedRootTasksShowAllDoneCopy(t *testing.T) {
	goalID := int64(1)
	goals := []*goalv1.Goal{
		{Id: goalID, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
	}
	completedAt := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 10, Name: "Finished task", GoalId: &goalID, CompletedAt: completedAt},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewGoalModel(nil, goals)
	m.tree = tree
	m.styled = false

	out := m.renderGoalDetail(60)
	if strings.Contains(out, "Finished task") {
		t.Errorf("completed root task should not appear in detail pane; got:\n%q", out)
	}
	if strings.Contains(out, "No tasks attached yet") {
		t.Errorf("should not show 'no tasks attached' when tasks exist but are all done; got:\n%q", out)
	}
	// Should show the "all done / time to plan" copy.
	if !strings.Contains(out, "No open tasks") {
		t.Errorf("expected 'No open tasks' copy for all-done state; got:\n%q", out)
	}
}

// TestGoalDetail_IncompleteRootOnlyNoChildren verifies that when a root task
// has incomplete children, only the root task name appears — not the children.
func TestGoalDetail_IncompleteRootOnlyNoChildren(t *testing.T) {
	goalID := int64(1)
	goals := []*goalv1.Goal{
		{Id: goalID, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
	}
	rootID := int64(10)
	tasks := []*taskv1.Task{
		{Id: rootID, Name: "Root task", GoalId: &goalID},
		{Id: 11, Name: "Child task", ParentId: &rootID},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewGoalModel(nil, goals)
	m.tree = tree
	m.styled = false

	out := m.renderGoalDetail(80)
	if !strings.Contains(out, "Root task") {
		t.Errorf("expected root task name in detail pane; got:\n%q", out)
	}
	if strings.Contains(out, "Child task") {
		t.Errorf("child task should not appear in detail pane (root-level only); got:\n%q", out)
	}
}

// TestGoalDetail_MixedCompletionOnlyIncompleteRootsShown verifies that when a
// goal has both completed and incomplete root tasks, only the incomplete ones appear.
func TestGoalDetail_MixedCompletionOnlyIncompleteRootsShown(t *testing.T) {
	goalID := int64(1)
	goals := []*goalv1.Goal{
		{Id: goalID, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS},
	}
	completedAt := timestamppb.Now()
	tasks := []*taskv1.Task{
		{Id: 10, Name: "Done task", GoalId: &goalID, CompletedAt: completedAt},
		{Id: 11, Name: "Open task", GoalId: &goalID},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewGoalModel(nil, goals)
	m.tree = tree
	m.styled = false

	out := m.renderGoalDetail(80)
	if strings.Contains(out, "Done task") {
		t.Errorf("completed task should not appear in detail pane; got:\n%q", out)
	}
	if !strings.Contains(out, "Open task") {
		t.Errorf("incomplete task should appear in detail pane; got:\n%q", out)
	}
}

// TestGoalPicker_InlineNamePlain asserts that the goal picker renders task
// names without leftover markdown syntax in plain mode.
func TestGoalPicker_InlineNamePlain(t *testing.T) {
	tasks := []*taskv1.Task{
		{Id: 1, Name: "**Bold task**"},
	}
	tree := cli.BuildTree(tasks)
	m := ExportNewGoalModel(nil, nil)
	m.styled = false
	m.goal.mode = goalPickLink
	// Build picker state directly: all tasks expanded and visible.
	expanded := map[int64]bool{1: true}
	m.goal.picker = pickerState{
		tree:     tree,
		visible:  ExportBuildVisible(tree, expanded, false, nil),
		cursor:   0,
		expanded: expanded,
	}

	out := m.renderGoalPicker(60)
	if strings.Contains(out, "**") {
		t.Errorf("plain goal picker: leftover ** in output %q", out)
	}
	if !strings.Contains(out, "Bold task") {
		t.Errorf("plain goal picker: name text missing; got %q", out)
	}
}

// ── Hold state tests (T019) ─────────────────────────────────────────────────

// TestGoal_HoldVisibleByDefault verifies that hold goals are shown in the
// default goals view, without needing 'c' (GoalToggleAll).
func TestGoal_HoldVisibleByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Active goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
		{Id: 2, Name: "On hold", State: goalv1.GoalState_GOAL_STATE_HOLD, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	out := m.renderGoalList(40)
	if !strings.Contains(out, "On hold") {
		t.Error("hold goal should be visible by default (showAll=false)")
	}
}

// TestGoal_HoldGroupHeader verifies that hold goals have their own "Hold" section header
// by default (showAll not required).
func TestGoal_HoldGroupHeader(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Active", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
		{Id: 2, Name: "Incubating", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
		{Id: 3, Name: "On hold", State: goalv1.GoalState_GOAL_STATE_HOLD, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	out := m.renderGoalList(40)

	committedIdx := strings.Index(out, "In Progress")
	incubatingIdx := strings.Index(out, "Incubating")
	holdIdx := strings.Index(out, "Hold")

	if committedIdx < 0 {
		t.Fatal("renderGoalList: 'In Progress' header not found")
	}
	if incubatingIdx < 0 {
		t.Fatal("renderGoalList: 'Incubating' header not found")
	}
	if holdIdx < 0 {
		t.Fatal("renderGoalList: 'Hold' header not found")
	}
	if committedIdx >= incubatingIdx {
		t.Errorf("group order: In Progress (%d) should appear before Incubating (%d)", committedIdx, incubatingIdx)
	}
	if incubatingIdx >= holdIdx {
		t.Errorf("group order: Incubating (%d) should appear before Hold (%d)", incubatingIdx, holdIdx)
	}
}

// TestGoal_HoldGroupOrder verifies: In Progress → Incubating → Hold, by default
// (Completed/Archived require showAll, which is not enabled here).
func TestGoal_HoldGroupOrder(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "A", State: goalv1.GoalState_GOAL_STATE_HOLD, Position: 0},
		{Id: 2, Name: "B", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
		{Id: 3, Name: "C", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	out := m.renderGoalList(40)

	committedIdx := strings.Index(out, "In Progress")
	incubatingIdx := strings.Index(out, "Incubating")
	holdIdx := strings.Index(out, "Hold")

	if committedIdx >= incubatingIdx || incubatingIdx >= holdIdx {
		t.Errorf("expected In Progress < Incubating < Hold; got indices %d, %d, %d",
			committedIdx, incubatingIdx, holdIdx)
	}
}

// TestGoal_HoldDetailStateName verifies the detail pane shows "State: Hold" for hold goals
// by default.
func TestGoal_HoldDetailStateName(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Paused work", State: goalv1.GoalState_GOAL_STATE_HOLD},
	}
	m2 := ExportNewGoalModel(nil, goals)

	out := m2.renderGoalDetail(60)
	if !strings.Contains(out, "State: Hold") {
		t.Errorf("detail pane: expected 'State: Hold'; got:\n%q", out)
	}
}

// ── Space toggle tests (T022) ───────────────────────────────────────────────

// TestGoal_Space_CommittedProducesCommand verifies Space on a committed goal
// returns a command (setGoalStateCmd).
func TestGoal_Space_CommittedProducesCommand(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_IN_PROGRESS, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd == nil {
		t.Fatal("Space on committed goal: expected a command, got nil")
	}
}

// TestGoal_Space_HoldProducesCommand verifies Space on a hold goal
// returns a command when showAll is enabled.
func TestGoal_Space_HoldProducesCommand(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "On hold", State: goalv1.GoalState_GOAL_STATE_HOLD, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)

	// Press 'c' to enable showAll so hold goals are visible.
	next, _ := m.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	m2 := next.(Model)

	_, cmd := m2.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd == nil {
		t.Fatal("Space on hold goal: expected a command, got nil")
	}
}

// TestGoal_Space_ArchivedHiddenByDefault verifies archived goals are hidden without showAll,
// so Space has no effect.
func TestGoal_Space_ArchivedHiddenByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Archived", State: goalv1.GoalState_GOAL_STATE_ARCHIVED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	visible := visibleGoals(m.goal.goals, m.goal.showAll)
	if len(visible) != 0 {
		t.Error("archived goal should be hidden when showAll=false")
	}
}

// TestGoal_Space_CompletedHiddenByDefault verifies completed goals need showAll.
func TestGoal_Space_CompletedHiddenByDefault(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "Done", State: goalv1.GoalState_GOAL_STATE_COMPLETED, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	visible := visibleGoals(m.goal.goals, m.goal.showAll)
	if len(visible) != 0 {
		t.Error("completed goal should be hidden when showAll=false")
	}
}

// TestGoal_Space_ShowsNotice verifies Space on an incubating goal shows a notice.
func TestGoal_Space_ShowsNotice(t *testing.T) {
	goals := []*goalv1.Goal{
		{Id: 1, Name: "My goal", State: goalv1.GoalState_GOAL_STATE_INCUBATING, Position: 0},
	}
	m := ExportNewGoalModel(nil, goals)
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m2 := next.(Model)
	notice := ExportNotice(m2)
	if notice == "" {
		t.Error("Space on incubating goal: expected a notice message")
	}
}

// TestGoal_EmptyListSpaceNoOp verifies Space on empty goal list does nothing.
func TestGoal_EmptyListSpaceNoOp(t *testing.T) {
	m := ExportNewGoalModel(nil, nil)
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if cmd != nil {
		t.Error("Space on empty goal list: expected nil cmd")
	}
}
