package tui

import (
	"os"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/config"
	"github.com/pboyd/twig/internal/markdown"
)

// TestMain initialises the theme palette before any test runs.
// Without this, the package-level color vars remain nil and styled tests fail.
func TestMain(m *testing.M) {
	initPalette(false) // light background default for tests
	os.Exit(m.Run())
}

func pint32(v int32) *int32 { return &v }

// ExportBuildVisible exposes buildVisible for tests.
func ExportBuildVisible(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, pendingComplete *int64) []*visibleRow {
	return buildVisible(tree, expanded, showAll, pendingComplete, time.Now().Local())
}

// ExportBuildVisibleOn exposes buildVisible with an injected reference date.
func ExportBuildVisibleOn(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, pendingComplete *int64, today time.Time) []*visibleRow {
	return buildVisible(tree, expanded, showAll, pendingComplete, today)
}

// ExportNewModel creates a Model with a fake tree for unit tests.
func ExportNewModel(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode) Model {
	m := newModel(client, nil, "", config.PomodoroConfig{}, false, nil)
	m.tree = tree
	m.visible = buildVisible(tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	return m
}

// ExportNewPlanModel creates a Model wired for planning tab tests.
func ExportNewPlanModel(taskClient taskv1connect.TaskServiceClient, planClient planv1connect.PlanServiceClient, day string) Model {
	m := newModel(taskClient, planClient, "", config.PomodoroConfig{}, false, nil)
	m.activeTab = tabPlanning
	m.plan.day = day
	return m
}

// ExportSetPlanEntries seeds the plan state for tests.
func ExportSetPlanEntries(m *Model, entries []*planv1.PlanEntry, cursor int) {
	m.plan.entries = entries
	m.plan.loaded = true
	m.plan.cursor = clampCursor(cursor, len(entries))
}

// ExportSetPlanDay sets the plan's in-view day.
func ExportSetPlanDay(m *Model, day string) {
	m.plan.day = day
}

// ExportPlanCursor returns the current plan cursor.
func ExportPlanCursor(m Model) int {
	return m.plan.cursor
}

// ExportPlanDay returns the current plan in-view day.
func ExportPlanDay(m Model) string {
	return m.plan.day
}

// ExportPlanEntries returns the current plan entries.
func ExportPlanEntries(m Model) []*planv1.PlanEntry {
	return m.plan.entries
}

// ExportPlanMode returns the current plan sub-mode.
func ExportPlanMode(m Model) int {
	return int(m.plan.mode)
}

// ExportPlanModeList is the planList constant for tests.
const ExportPlanModeList = int(planList)

// ExportSelectedPlanEntry returns the currently selected plan entry, or nil.
func ExportSelectedPlanEntry(m Model) *planv1.PlanEntry {
	if len(m.plan.entries) == 0 {
		return nil
	}
	return m.plan.entries[m.plan.cursor]
}

// ExportActiveTab returns the current active tab as an int (0=tasks, 1=planning).
func ExportActiveTab(m Model) int {
	return int(m.activeTab)
}

// ExportTabGoals is the tabGoals constant for tests.
const ExportTabGoals = int(tabGoals)

// ExportTabTasks is the tabTasks constant for tests.
const ExportTabTasks = int(tabTasks)

// ExportTabPlanning is the tabPlanning constant for tests.
const ExportTabPlanning = int(tabPlanning)

// ExportVisibleRow exposes the visibleRow type for inspection in tests.
type ExportVisibleRow = visibleRow

// ExportUpdateTaskCmd exposes updateTaskCmd for unit tests.
func ExportUpdateTaskCmd(client taskv1connect.TaskServiceClient, msg editSavedMsg) tea.Cmd {
	return updateTaskCmd(client, msg)
}

// ExportCreateTaskCmd exposes createTaskCmd for unit tests.
func ExportCreateTaskCmd(client taskv1connect.TaskServiceClient, msg editSavedMsg) tea.Cmd {
	return createTaskCmd(client, msg)
}

// ExportEditSavedMsg constructs an editSavedMsg for unit tests.
func ExportEditSavedMsg(name string, parentID *int64) editSavedMsg {
	return editSavedMsg{name: name, parentID: parentID}
}

// ExportNewStyledModel creates a Model with the styled flag set for rendering tests.
func ExportNewStyledModel(client taskv1connect.TaskServiceClient, tree []*cli.TreeNode, styled bool) Model {
	m := ExportNewModel(client, tree)
	m.styled = styled
	m.height = 40
	return m
}

// ExportRenderList exposes renderList for unit tests.
func ExportRenderList(m Model, width int) string {
	return m.renderList(width)
}

// ExportRenderDetails exposes renderDetails for unit tests.
func ExportRenderDetails(task *taskv1.Task, md *markdown.Renderer, width int, styled bool, scheduledDays []string, effectiveGoalName string) string {
	return renderDetails(task, md, width, styled, scheduledDays, effectiveGoalName, time.Now())
}

// ExportSplitPlanEntries exposes splitPlanEntries for tests.
func ExportSplitPlanEntries(entries []*planv1.PlanEntry) (untimed, timed []*planv1.PlanEntry) {
	return splitPlanEntries(entries)
}

// ExportVisibleSiblings exposes visibleSiblings for tree tests.
func ExportVisibleSiblings(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, taskID int64) (prev, next int64) {
	return visibleSiblings(tree, expanded, showAll, taskID)
}

// ExportReorderResultMsg exposes reorderResultMsg for update tests.
type ExportReorderResultMsg = reorderResultMsg

// ExportHandleReorderResult exposes handleReorderResult for update tests.
func ExportHandleReorderResult(m Model, msg reorderResultMsg) Model {
	result, _ := m.handleReorderResult(msg)
	return result
}

// ExportSetNowFunc injects a time source into the model for deterministic
// auto-schedule floor tests (FR-003).
func ExportSetNowFunc(m *Model, f func() time.Time) {
	m.nowFunc = f
}

// ExportNotice returns the model's current transient notice string.
func ExportNotice(m Model) string {
	return m.notice
}

// ExportEditFormIsDirty exposes editFormModel.isDirty for tests.
func ExportEditFormIsDirty(f editFormModel) bool {
	return f.isDirty()
}

// SetConfirmingDiscard sets m.confirmingDiscard for tests.
func (m *Model) SetConfirmingDiscard(v bool) {
	m.confirmingDiscard = v
}

// ConfirmingDiscard returns m.confirmingDiscard for tests.
func (m *Model) ConfirmingDiscard() bool {
	return m.confirmingDiscard
}

// PlanFormDirty exposes planFormDirty for tests.
func PlanFormDirty(f planFormState) bool {
	return planFormDirty(f)
}

// PlanFormFieldValues returns the current field values for tests.
func PlanFormFieldValues(f planFormState) []string {
	return planFormFieldValues(f)
}

// ExportTabReport is the tabReport constant for tests.
const ExportTabReport = int(tabReport)

// ExportNewReportModel creates a Model in the Report tab for unit tests.
func ExportNewReportModel(client taskv1connect.TaskServiceClient) Model {
	m := newModel(client, nil, "", config.PomodoroConfig{}, false, nil)
	m.activeTab = tabReport
	m.keys.ReportMode = true
	return m
}

// ExportSetReportData seeds the report state for render tests.
func ExportSetReportData(m *Model, data reportState) {
	m.reportData = data
}

// ExportNewGoalModel creates a Model in the Goals tab for unit tests.
func ExportNewGoalModel(taskClient taskv1connect.TaskServiceClient, goals []*goalv1.Goal) Model {
	m := newModel(taskClient, nil, "", config.PomodoroConfig{}, false, nil)
	m.activeTab = tabGoals
	m.goal.goals = goals
	m.goal.loaded = true
	m.keys.GoalMode = true
	return m
}

// ExportGoalCursor returns the current goal cursor.
func ExportGoalCursor(m Model) int { return m.goal.cursor }

// ExportDispatchListGoalsResult dispatches a listGoalsResultMsg for testing.
// highlightID of 0 means "no highlight, just clamp".
func ExportDispatchListGoalsResult(m Model, goals []*goalv1.Goal, highlightID int64) (Model, tea.Cmd) {
	updated, cmd := m.Update(listGoalsResultMsg{goals: goals, highlightID: highlightID})
	return updated.(Model), cmd
}

// ExportGoalShowAll returns the current goal showAll state.
func ExportGoalShowAll(m Model) bool { return m.goal.showAll }

// ExportGoalErr returns the current goal error.
func ExportGoalErr(m Model) error { return m.goal.err }

// ExportDispatchTaskGoalMutation dispatches a taskGoalMutationMsg for testing.
func ExportDispatchTaskGoalMutation(m Model, err error) (Model, tea.Cmd) {
	updated, cmd := m.Update(taskGoalMutationMsg{err: err})
	return updated.(Model), cmd
}

// ExportGoalMode returns the current goalViewMode as an int for tests.
func ExportGoalMode(m Model) int { return int(m.goal.mode) }

// ExportGoalModeStatusHistory is the goalStatusHistory constant for tests.
const ExportGoalModeStatusHistory = int(goalStatusHistory)

// ExportGoalModeStatusReader is the goalStatusReader constant for tests.
const ExportGoalModeStatusReader = int(goalStatusReader)

// ExportGoalModeList is the goalList constant for tests.
const ExportGoalModeList = int(goalList)

// ExportGoalStatusUpdates returns the current status updates slice for tests.
func ExportGoalStatusUpdates(m Model) []*goalv1.StatusUpdate { return m.goal.statusUpdates }

// ExportGoalStatusCursor returns the current status cursor for tests.
func ExportGoalStatusCursor(m Model) int { return m.goal.statusCursor }

// ExportGoalReaderOffset returns the current reader scroll offset for tests.
func ExportGoalReaderOffset(m Model) int { return m.goal.readerOffset }

// ExportGoalComposeActive returns whether a status compose is in progress.
func ExportGoalComposeActive(m Model) bool { return m.goal.compose.active }

// ExportGoalComposeEditingID returns the editingID from the current compose state.
func ExportGoalComposeEditingID(m Model) int64 { return m.goal.compose.editingID }

// ExportDispatchGoalStatusList dispatches a goalStatusListMsg for testing.
func ExportDispatchGoalStatusList(m Model, updates []*goalv1.StatusUpdate, err error) (Model, tea.Cmd) {
	updated, cmd := m.Update(goalStatusListMsg{updates: updates, err: err})
	return updated.(Model), cmd
}

// ExportDispatchGoalStatusMutation dispatches a goalStatusMutationMsg for testing.
func ExportDispatchGoalStatusMutation(m Model, goalID int64, err error) (Model, tea.Cmd) {
	updated, cmd := m.Update(goalStatusMutationMsg{goalID: goalID, err: err})
	return updated.(Model), cmd
}

// ExportRenderGoalDetail exposes renderGoalDetail for tests.
func ExportRenderGoalDetail(m Model, width int) string {
	return m.renderGoalDetail(width)
}

// ExportRenderStatusHistory exposes renderStatusHistory for tests.
func ExportRenderStatusHistory(m Model, width, height int) string {
	return m.renderStatusHistory(width, height)
}

// ExportRenderStatusReader exposes renderStatusReader for tests.
func ExportRenderStatusReader(m Model, width, height int) string {
	return m.renderStatusReader(width, height)
}

// ExportHandleGoalStatusEditorFinished dispatches an editorFinishedMsg while
// goal.compose.active is set, for testing the status compose flow.
func ExportHandleGoalStatusEditorFinished(m Model, content string, err error) (Model, tea.Cmd) {
	m.goal.compose.active = true
	updated, cmd := m.Update(editorFinishedMsg{content: content, err: err})
	return updated.(Model), cmd
}

// ExportSetGoalStatusUpdates seeds the status updates for tests.
func ExportSetGoalStatusUpdates(m *Model, updates []*goalv1.StatusUpdate, mode int) {
	m.goal.statusUpdates = updates
	m.goal.mode = goalViewMode(mode)
}

// ExportSetGoalCompose sets the compose state for tests.
func ExportSetGoalCompose(m *Model, goalID, editingID int64) {
	m.goal.compose = statusCompose{goalID: goalID, editingID: editingID, active: true}
}

// ExportBuildVisibleFiltered exposes buildVisibleFiltered for tests.
func ExportBuildVisibleFiltered(tree []*cli.TreeNode, expanded map[int64]bool, showAll bool, pendingComplete *int64, filteredIDs map[int64]bool) []*visibleRow {
	return buildVisibleFiltered(tree, expanded, showAll, pendingComplete, time.Now().Local(), filteredIDs)
}

// ExportSetFilterState seeds the filter state on a model for tests.
func ExportSetFilterState(m *Model, expr string, matches []int64) {
	m.filterExpr = expr
	m.filterMatches = matches
	m.filteredIDs = make(map[int64]bool, len(matches))
	for _, id := range matches {
		m.filteredIDs[id] = true
	}
	m.visible = buildVisibleFiltered(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local(), m.filteredIDs)
	m.cursor = clampCursor(m.cursor, len(m.visible))
}

// ExportFilterExpr returns the active filter expression.
func ExportFilterExpr(m Model) string {
	return m.filterExpr
}

// ExportFilterMatches returns the matched task IDs.
func ExportFilterMatches(m Model) []int64 {
	return m.filterMatches
}
