package tui

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"connectrpc.com/connect"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/cli"
	"github.com/pboyd/twig/internal/goal"
	"github.com/pboyd/twig/internal/pomodoro"
	"github.com/pboyd/twig/internal/report"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── message types ──────────────────────────────────────────────────────────

// listTasksResultMsg carries the result of a bare ListTasks (init / Ctrl-R).
type listTasksResultMsg struct {
	tree []*cli.TreeNode
	err  error
}

// reorderResultMsg carries the result of a ReorderTask RPC.
type reorderResultMsg struct {
	taskID   int64
	siblings []*taskv1.Task
	err      error
}

// refreshedMsg is returned by mutation commands: RPC + re-fetch bundled together.
type refreshedMsg struct {
	tree        []*cli.TreeNode
	highlightID int64 // task to focus after refresh; 0 = use clamped cursor
	err         error // hard error: the refresh did not happen, tree/highlightID are unset
	// partialErr is a soft error: the tree WAS refreshed (a mutation partially
	// succeeded), but something after it failed. Shown alongside the fresh tree
	// instead of suppressing it.
	partialErr error
}

// moveTaskResultMsg carries the result of a move-parent UpdateTask call.
type moveTaskResultMsg struct {
	taskID int64
	tree   []*cli.TreeNode
	err    error
}

// reportResultMsg carries the result of a report fetch (tasks + pom count).
type reportResultMsg struct {
	dayGroups []report.DayGroup
	finished  []report.AccomplishmentGroup
	ongoing   []report.AccomplishmentGroup
	totals    report.Totals
	period    report.Period
	err       error
}

// scheduledDaysResultMsg carries the result of a ListScheduledDays RPC.
type scheduledDaysResultMsg struct {
	days map[int64][]string
	err  error
}

// listGoalsResultMsg carries the result of a ListGoals RPC.
// highlightID, when non-zero, causes the cursor to follow that goal by ID
// after the list is refreshed (used after a reorder to keep the moved goal
// highlighted).
type listGoalsResultMsg struct {
	goals       []*goalv1.Goal
	err         error
	highlightID int64
}

// goalMutationMsg carries the result of a create or update goal RPC.
type goalMutationMsg struct {
	goal *goalv1.Goal
	err  error
}

// goalDeletedMsg carries the result of a delete goal RPC.
type goalDeletedMsg struct {
	err error
}

// goalPickerTasksMsg carries the task tree for the goal-tab task picker (L/U flows).
type goalPickerTasksMsg struct {
	tree []*cli.TreeNode
	err  error
}

// taskGoalMutationMsg carries the result of a SetTaskGoal RPC called from the goal pane.
type taskGoalMutationMsg struct {
	err error
}

// goalStatusListMsg carries the result of a ListGoalStatusUpdates RPC.
type goalStatusListMsg struct {
	updates []*goalv1.StatusUpdate
	err     error
}

// goalStatusMutationMsg carries the result of an add/update/delete status-update RPC.
type goalStatusMutationMsg struct {
	goalID int64
	err    error
}

// filterResultMsg carries the result of a FilterTasks RPC.
type filterResultMsg struct {
	gen  int     // generation counter to discard stale responses
	expr string  // the expression this result was dispatched for
	ids  []int64 // matched task IDs (nil on error)
	err  error   // parse/validation error
}

// filterFocus says where the cursor should land when a filter result arrives.
type filterFocus int

const (
	// filterKeepCursor clamps the existing cursor. Used when the user is
	// refining or re-firing a filter and expects to stay where they were.
	filterKeepCursor filterFocus = iota
	// filterFirstMatch moves the cursor to the first row that actually matched.
	// Used when a jump brings the user to the Tasks tab and the old cursor
	// index means nothing.
	filterFirstMatch
)

// ── command factories ───────────────────────────────────────────────────────

// persistTreeStateCmd snapshots the current expanded set and saves it
// asynchronously. Errors are silently discarded (FR-005).
func (m Model) persistTreeStateCmd() tea.Cmd {
	path := m.statePath
	key := m.activeProfile
	expanded := make(map[int64]bool, len(m.expanded))
	for id, v := range m.expanded {
		expanded[id] = v
	}
	live := liveTaskIDs(m.tree)
	return func() tea.Msg {
		if path != "" {
			_ = saveTreeState(path, key, expanded, live)
		}
		return nil
	}
}

// clearFilter removes any active filter (applied or in-progress) and
// restores the unfiltered task list.
func (m Model) clearFilter() Model {
	m.mode = modeList
	m.filterInput.SetValue("")
	m.filterInput.Blur()
	m.filterExpr = ""
	m.filterMatches = nil
	m.filteredIDs = nil
	m.filterInvalid = false
	m.filterGen++ // cancel: discard any response already in flight
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, m.nowOrDefault())
	m.cursor = clampCursor(m.cursor, len(m.visible))
	return m
}

// applyFilter records expr as the pending filter and dispatches it to the
// server. Bumping filterGen both invalidates any response already in flight and
// keeps the counter in step with clearFilter, which uses the same bump to
// cancel. The expression is not committed to filterExpr here — that happens in
// handleFilterResult when the result actually lands, so filterExpr always
// describes the rows currently on screen. Callers own m.mode.
func (m Model) applyFilter(expr string, focus filterFocus) (Model, tea.Cmd) {
	m.filterGen++
	m.filterFocus = focus
	m.filterInput.SetValue(expr)
	m.filterInput.Blur()
	m.filterInvalid = false
	return m, filterCmd(m.client, expr, m.showAll, m.nowOrDefault(), m.filterGen)
}

// rebuildVisible recomputes the visible row list, honoring an active filter so
// list navigation and local edits stay within the matched set. With no filter
// applied it rebuilds the full tree view.
func (m Model) rebuildVisible() []*visibleRow {
	if m.filterExpr != "" {
		return buildVisibleFiltered(m.tree, m.expanded, m.showAll, m.pendingComplete, m.nowOrDefault(), m.filteredIDs)
	}
	return buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, m.nowOrDefault())
}

// handleFilterKey handles key events while the filter bar is active.
func (m Model) handleFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.clearFilter(), nil

	case "enter":
		expr := strings.TrimSpace(m.filterInput.Value())
		if expr == "" {
			return m.clearFilter(), nil
		}
		m2, cmd := m.applyFilter(expr, filterKeepCursor)
		return m2, cmd

	default:
		// Any other key just edits the input; the expression is only
		// evaluated on Enter (see the "enter" case above).
		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)
		return m, cmd
	}
}

// handleFilterResult handles the response from FilterTasks.
func (m Model) handleFilterResult(msg filterResultMsg) (tea.Model, tea.Cmd) {
	// Discard stale responses.
	if msg.gen != m.filterGen {
		return m, nil
	}

	if msg.err != nil {
		m.filterInvalid = true
		m.filterMatches = nil
		m.filteredIDs = nil
		m.err = msg.err
		return m, nil
	}

	m.filterInvalid = false
	m.filterMatches = msg.ids
	m.filterExpr = msg.expr
	m.err = nil

	// Build the set for O(1) lookup.
	m.filteredIDs = make(map[int64]bool, len(msg.ids))
	for _, id := range msg.ids {
		m.filteredIDs[id] = true
	}

	// Rebuild visible list in filter mode.
	m.visible = buildVisibleFiltered(m.tree, m.expanded, m.showAll, m.pendingComplete, m.nowOrDefault(), m.filteredIDs)
	if m.filterFocus == filterFirstMatch {
		m.cursor = firstMatchingRow(m.visible, m.filteredIDs)
	} else {
		m.cursor = clampCursor(m.cursor, len(m.visible))
	}
	// Re-clamp scroll offset after the filtered list is rebuilt.
	m = m.reconcileScroll()

	m.mode = modeList
	return m, nil
}

// firstMatchingRow returns the index of the first row whose task is in matched.
// buildVisibleFiltered also emits non-matching ancestors as scaffold, so row 0
// is not necessarily part of the match set. Returns 0 when nothing matched.
func firstMatchingRow(rows []*visibleRow, matched map[int64]bool) int {
	for i, row := range rows {
		if matched[row.node.Task.Id] {
			return i
		}
	}
	return 0
}

// saveAndQuitCmd saves the tree state synchronously and then signals bubbletea
// to quit. Errors from the save are silently discarded (FR-005).
func (m Model) saveAndQuitCmd() tea.Cmd {
	path := m.statePath
	key := m.activeProfile
	expanded := make(map[int64]bool, len(m.expanded))
	for id, v := range m.expanded {
		expanded[id] = v
	}
	live := liveTaskIDs(m.tree)
	return func() tea.Msg {
		if path != "" {
			_ = saveTreeState(path, key, expanded, live)
		}
		return tea.QuitMsg{}
	}
}

func listScheduledDaysCmd(client planv1connect.PlanServiceClient) tea.Cmd {
	fromDay := time.Now().Format("2006-01-02")
	return func() tea.Msg {
		resp, err := client.ListScheduledDays(context.Background(), connect.NewRequest(&planv1.ListScheduledDaysRequest{
			FromDay: fromDay,
		}))
		if err != nil {
			return scheduledDaysResultMsg{err: err}
		}
		m := make(map[int64][]string)
		for _, sd := range resp.Msg.Days {
			m[sd.TaskId] = append(m[sd.TaskId], sd.Day)
		}
		return scheduledDaysResultMsg{days: m}
	}
}

func listTasksCmd(client taskv1connect.TaskServiceClient) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return listTasksResultMsg{err: err}
		}
		return listTasksResultMsg{tree: cli.BuildTree(resp.Msg.Tasks)}
	}
}

func updateTaskCmd(client taskv1connect.TaskServiceClient, msg editSavedMsg) tea.Cmd {
	return func() tea.Msg {
		id := *msg.taskID
		req := &taskv1.UpdateTaskRequest{
			Id:          id,
			Name:        msg.name,
			Description: msg.description,
		}
		if msg.parentID != nil {
			req.ParentId = msg.parentID
		}
		if msg.dueStr != "" {
			ts, err := cli.ParseDue(msg.dueStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.Due = ts
		}
		if msg.snoozeStr != "" {
			ts, err := cli.ParseDue(msg.snoozeStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.SnoozeUntil = ts
		}

		_, err := client.UpdateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return refreshedMsg{err: err}
		}

		// If the goal association changed, update it via SetTaskGoal.
		if msg.goalChanged {
			goalReq := &taskv1.SetTaskGoalRequest{TaskId: id, GoalId: msg.newGoalID}
			_, err = client.SetTaskGoal(context.Background(), connect.NewRequest(goalReq))
			if err != nil {
				// Route through taskGoalMutationMsg so the friendly
				// FailedPrecondition notice is shown (not a raw error).
				return taskGoalMutationMsg{err: err}
			}
		}

		return fetchAfterMutation(client, id)
	}
}

func createTaskCmd(client taskv1connect.TaskServiceClient, planClient planv1connect.PlanServiceClient, msg editSavedMsg) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.CreateTaskRequest{
			Name:        msg.name,
			Description: msg.description,
		}
		if msg.parentID != nil {
			req.ParentId = msg.parentID
		}
		if msg.dueStr != "" {
			ts, err := cli.ParseDue(msg.dueStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.Due = ts
		}
		if msg.snoozeStr != "" {
			ts, err := cli.ParseDue(msg.snoozeStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.SnoozeUntil = ts
		}

		resp, err := client.CreateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return refreshedMsg{err: err}
		}
		newID := resp.Msg.Task.Id

		// If a goal was selected on the new-root-task form, associate via SetTaskGoal.
		if msg.newGoalID != nil {
			goalReq := &taskv1.SetTaskGoalRequest{TaskId: newID, GoalId: msg.newGoalID}
			if _, err := client.SetTaskGoal(context.Background(), connect.NewRequest(goalReq)); err != nil {
				return taskGoalMutationMsg{err: err}
			}
		}

		// If a plan day was chosen, add the task to that day's plan.
		if msg.planDay != "" && planClient != nil {
			_, err := planClient.AddPlanTask(context.Background(), connect.NewRequest(&planv1.AddPlanTaskRequest{
				Day:            msg.planDay,
				TaskId:         newID,
				DurationMinute: 0,
			}))
			if err != nil {
				// Partial failure: task was created but plan failed (FR-007).
				// Report the partial success — the task stays, and the tree
				// must still be refreshed so it's visible.
				var partial error
				var ce *connect.Error
				if errors.As(err, &ce) && ce.Code() == connect.CodeFailedPrecondition {
					partial = fmt.Errorf("task created, but already on that day's plan")
				} else {
					partial = fmt.Errorf("task saved, but it didn't make it onto the plan. Want to add it from the list?")
				}
				if rm, ok := fetchAfterMutation(client, newID).(refreshedMsg); ok && rm.err == nil {
					rm.partialErr = partial
					return rm
				}
				// ListTasks itself failed — fall back to a plain hard error.
				return refreshedMsg{highlightID: newID, err: partial}
			}
		}

		// For subtasks, keep the parent highlighted so the user can
		// immediately press 'n' again to add another sibling.
		if msg.parentID != nil {
			return fetchAfterMutation(client, *msg.parentID)
		}
		return fetchAfterMutation(client, newID)
	}
}

func deleteTaskCmd(client taskv1connect.TaskServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.DeleteTask(context.Background(), connect.NewRequest(&taskv1.DeleteTaskRequest{Id: id}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		// highlightID 0 means: caller will compute cursor from position.
		return refreshedMsg{tree: cli.BuildTree(resp.Msg.Tasks), highlightID: 0}
	}
}

func completeTaskCmd(client taskv1connect.TaskServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CompleteTask(context.Background(), connect.NewRequest(&taskv1.CompleteTaskRequest{Id: id}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func uncompleteTaskCmd(client taskv1connect.TaskServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.UncompleteTask(context.Background(), connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: id}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func setEstimateCmd(client taskv1connect.TaskServiceClient, id int64, estimate int32) tea.Cmd {
	return func() tea.Msg {
		_, err := client.SetEstimate(context.Background(), connect.NewRequest(&taskv1.SetEstimateRequest{
			TaskId:   id,
			Estimate: estimate,
		}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(client, id)
	}
}

func reorderTaskCmd(client taskv1connect.TaskServiceClient, taskID, anchorID int64, insertBefore bool) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.ReorderTaskRequest{TaskId: taskID}
		if insertBefore {
			req.Anchor = &taskv1.ReorderTaskRequest_BeforeTaskId{BeforeTaskId: anchorID}
		} else {
			req.Anchor = &taskv1.ReorderTaskRequest_AfterTaskId{AfterTaskId: anchorID}
		}
		resp, err := client.ReorderTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return reorderResultMsg{taskID: taskID, err: err}
		}
		return reorderResultMsg{taskID: taskID, siblings: resp.Msg.Siblings}
	}
}

func moveTaskCmd(client taskv1connect.TaskServiceClient, task *taskv1.Task, newParentID *int64) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.UpdateTaskRequest{
			Id:          task.Id,
			Name:        task.Name,
			Description: task.GetDescription(),
		}
		if task.Due != nil {
			req.Due = task.Due
		}
		if newParentID != nil {
			req.ParentId = newParentID
		}
		_, err := client.UpdateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return moveTaskResultMsg{taskID: task.Id, err: err}
		}
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return moveTaskResultMsg{taskID: task.Id, err: err}
		}
		return moveTaskResultMsg{taskID: task.Id, tree: cli.BuildTree(resp.Msg.Tasks)}
	}
}

// ── goal command factories ───────────────────────────────────────────────────

func listGoalsCmd(client goalv1connect.GoalServiceClient) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListGoals(context.Background(), connect.NewRequest(&goalv1.ListGoalsRequest{}))
		if err != nil {
			return listGoalsResultMsg{err: err}
		}
		return listGoalsResultMsg{goals: resp.Msg.Goals}
	}
}

func createGoalCmd(client goalv1connect.GoalServiceClient, name, desc string, due *timestamppb.Timestamp) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.CreateGoal(context.Background(), connect.NewRequest(&goalv1.CreateGoalRequest{
			Name:        name,
			Description: desc,
			Due:         due,
		}))
		if err != nil {
			return goalMutationMsg{err: err}
		}
		return goalMutationMsg{goal: resp.Msg.Goal}
	}
}

func updateGoalCmd(client goalv1connect.GoalServiceClient, id int64, name, desc string, due *timestamppb.Timestamp) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.UpdateGoal(context.Background(), connect.NewRequest(&goalv1.UpdateGoalRequest{
			Id:          id,
			Name:        name,
			Description: desc,
			Due:         due,
		}))
		if err != nil {
			return goalMutationMsg{err: err}
		}
		return goalMutationMsg{goal: resp.Msg.Goal}
	}
}

func setGoalStateCmd(client goalv1connect.GoalServiceClient, id int64, state goalv1.GoalState) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.SetGoalState(context.Background(), connect.NewRequest(&goalv1.SetGoalStateRequest{
			Id:    id,
			State: state,
		}))
		if err != nil {
			return goalMutationMsg{err: err}
		}
		return goalMutationMsg{goal: resp.Msg.Goal}
	}
}

// updateGoalThenSetStateCmd updates a goal's fields then sets its state in a single chain.
func updateGoalThenSetStateCmd(client goalv1connect.GoalServiceClient, id int64, name, desc string, due *timestamppb.Timestamp, state goalv1.GoalState) tea.Cmd {
	return func() tea.Msg {
		_, err := client.UpdateGoal(context.Background(), connect.NewRequest(&goalv1.UpdateGoalRequest{
			Id:          id,
			Name:        name,
			Description: desc,
			Due:         due,
		}))
		if err != nil {
			return goalMutationMsg{err: err}
		}
		resp, err := client.SetGoalState(context.Background(), connect.NewRequest(&goalv1.SetGoalStateRequest{
			Id:    id,
			State: state,
		}))
		if err != nil {
			return goalMutationMsg{err: err}
		}
		return goalMutationMsg{goal: resp.Msg.Goal}
	}
}

func deleteGoalCmd(client goalv1connect.GoalServiceClient, id int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.DeleteGoal(context.Background(), connect.NewRequest(&goalv1.DeleteGoalRequest{Id: id}))
		return goalDeletedMsg{err: err}
	}
}

func reorderGoalCmd(client goalv1connect.GoalServiceClient, goalID, anchorID int64, insertBefore bool) tea.Cmd {
	return func() tea.Msg {
		req := &goalv1.ReorderGoalRequest{GoalId: goalID}
		if insertBefore {
			req.Anchor = &goalv1.ReorderGoalRequest_BeforeGoalId{BeforeGoalId: anchorID}
		} else {
			req.Anchor = &goalv1.ReorderGoalRequest_AfterGoalId{AfterGoalId: anchorID}
		}
		_, err := client.ReorderGoal(context.Background(), connect.NewRequest(req))
		if err != nil {
			return goalMutationMsg{err: err}
		}
		// Re-fetch goals after reorder to get updated positions.
		resp, err := client.ListGoals(context.Background(), connect.NewRequest(&goalv1.ListGoalsRequest{}))
		if err != nil {
			return listGoalsResultMsg{err: err}
		}
		return listGoalsResultMsg{goals: resp.Msg.Goals, highlightID: goalID}
	}
}

// listGoalStatusUpdatesCmd fetches the status-update history for a goal.
func listGoalStatusUpdatesCmd(client goalv1connect.GoalServiceClient, goalID int64) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListGoalStatusUpdates(context.Background(), connect.NewRequest(&goalv1.ListGoalStatusUpdatesRequest{
			GoalId: goalID,
		}))
		if err != nil {
			return goalStatusListMsg{err: err}
		}
		return goalStatusListMsg{updates: resp.Msg.Updates}
	}
}

// addGoalStatusUpdateCmd records a new status update on a goal.
func addGoalStatusUpdateCmd(client goalv1connect.GoalServiceClient, goalID int64, body string) tea.Cmd {
	return func() tea.Msg {
		_, err := client.AddGoalStatusUpdate(context.Background(), connect.NewRequest(&goalv1.AddGoalStatusUpdateRequest{
			GoalId: goalID,
			Body:   body,
		}))
		return goalStatusMutationMsg{goalID: goalID, err: err}
	}
}

// updateGoalStatusUpdateCmd replaces the body of an existing status update.
func updateGoalStatusUpdateCmd(client goalv1connect.GoalServiceClient, id int64, body string) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.UpdateGoalStatusUpdate(context.Background(), connect.NewRequest(&goalv1.UpdateGoalStatusUpdateRequest{
			Id:   id,
			Body: body,
		}))
		if err != nil {
			return goalStatusMutationMsg{err: err}
		}
		return goalStatusMutationMsg{goalID: resp.Msg.Update.GoalId, err: nil}
	}
}

// deleteGoalStatusUpdateCmd removes a single status update.
func deleteGoalStatusUpdateCmd(client goalv1connect.GoalServiceClient, id int64, goalID int64) tea.Cmd {
	return func() tea.Msg {
		_, err := client.DeleteGoalStatusUpdate(context.Background(), connect.NewRequest(&goalv1.DeleteGoalStatusUpdateRequest{
			Id: id,
		}))
		return goalStatusMutationMsg{goalID: goalID, err: err}
	}
}

// goalListTasksForPickerCmd fetches all tasks for the goal-tab task picker.
func goalListTasksForPickerCmd(client taskv1connect.TaskServiceClient) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return goalPickerTasksMsg{err: err}
		}
		return goalPickerTasksMsg{tree: cli.BuildTree(resp.Msg.Tasks)}
	}
}

// setTaskGoalAndRefreshCmd calls SetTaskGoal then re-fetches tasks.
func setTaskGoalAndRefreshCmd(taskClient taskv1connect.TaskServiceClient, taskID int64, goalID *int64) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.SetTaskGoalRequest{TaskId: taskID, GoalId: goalID}
		_, err := taskClient.SetTaskGoal(context.Background(), connect.NewRequest(req))
		if err != nil {
			return taskGoalMutationMsg{err: err}
		}
		return taskGoalMutationMsg{}
	}
}

// createAndLinkTaskCmd creates a task then associates it with goalID via SetTaskGoal.
func createAndLinkTaskCmd(taskClient taskv1connect.TaskServiceClient, msg editSavedMsg, goalID int64) tea.Cmd {
	return func() tea.Msg {
		req := &taskv1.CreateTaskRequest{
			Name:        msg.name,
			Description: msg.description,
		}
		if msg.dueStr != "" {
			ts, err := cli.ParseDue(msg.dueStr)
			if err != nil {
				return refreshedMsg{err: err}
			}
			req.Due = ts
		}
		createResp, err := taskClient.CreateTask(context.Background(), connect.NewRequest(req))
		if err != nil {
			return refreshedMsg{err: err}
		}
		newTaskID := createResp.Msg.Task.Id
		_, err = taskClient.SetTaskGoal(context.Background(), connect.NewRequest(&taskv1.SetTaskGoalRequest{
			TaskId: newTaskID,
			GoalId: &goalID,
		}))
		if err != nil {
			return refreshedMsg{err: err}
		}
		return fetchAfterMutation(taskClient, newTaskID)
	}
}

// fetchAfterMutation calls ListTasks and returns a refreshedMsg.
func fetchAfterMutation(client taskv1connect.TaskServiceClient, highlightID int64) tea.Msg {
	resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		return refreshedMsg{err: err}
	}
	return refreshedMsg{tree: cli.BuildTree(resp.Msg.Tasks), highlightID: highlightID}
}

// filterCmd calls FilterTasks on the server and returns a filterResultMsg.
func filterCmd(client taskv1connect.TaskServiceClient, expr string, showAll bool, now time.Time, gen int) tea.Cmd {
	return func() tea.Msg {
		resp, err := client.FilterTasks(context.Background(), connect.NewRequest(&taskv1.FilterTasksRequest{
			Expression: expr,
			ShowAll:    showAll,
			Today:      now.Format("2006-01-02"),
		}))
		if err != nil {
			var ce *connect.Error
			if errors.As(err, &ce) && ce.Code() == connect.CodeInvalidArgument {
				return filterResultMsg{gen: gen, expr: expr, err: ce}
			}
			return filterResultMsg{gen: gen, expr: expr, err: err}
		}
		return filterResultMsg{gen: gen, expr: expr, ids: resp.Msg.TaskIds}
	}
}

// ── Init ───────────────────────────────────────────────────────────────────

func pomodoroRemaining(startAt time.Time, now time.Time) time.Duration {
	return pomodoro.Remaining(startAt, now)
}

// fetchReportCmd fetches all tasks and the pomodoro count for the given period,
// builds day-grouped and accomplishment-grouped data, and returns a reportResultMsg.
func fetchReportCmd(client taskv1connect.TaskServiceClient, p report.Period) tea.Cmd {
	return func() tea.Msg {
		tasksResp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
		if err != nil {
			return reportResultMsg{err: err, period: p}
		}

		pomResp, err := client.CountCompletedPomodoros(context.Background(), connect.NewRequest(&taskv1.CountCompletedPomodorosRequest{
			Start: timestamppb.New(p.StartUTC()),
			End:   timestamppb.New(p.EndUTC()),
		}))
		if err != nil {
			return reportResultMsg{err: err, period: p}
		}

		tasks := tasksResp.Msg.Tasks
		entries := report.BuildEntries(tasks, p)
		totals := report.Totals{
			TasksCompleted:     len(entries),
			PomodorosCompleted: pomResp.Msg.Count,
		}

		var dayGroups []report.DayGroup
		var finished, ongoing []report.AccomplishmentGroup

		switch p.ReportLayout() {
		case report.DayGrouped:
			dayGroups = report.GroupByDay(entries)
		case report.AccomplishmentGrouped:
			finished, ongoing = report.GroupByAccomplishment(entries, tasks, p)
		}

		return reportResultMsg{
			dayGroups: dayGroups,
			finished:  finished,
			ongoing:   ongoing,
			totals:    totals,
			period:    p,
		}
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{listTasksCmd(m.client), getActivePomCmd(m.client, m.tree), listScheduledDaysCmd(m.planClient)}
	if m.goalClient != nil {
		cmds = append(cmds, listGoalsCmd(m.goalClient))
	}
	return tea.Batch(cmds...)
}

// ── Update ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.SetWidth(msg.Width)
		m.plainHelp.SetWidth(msg.Width)
		// Give the filter input a real width; textinput's placeholderView
		// truncates the placeholder to a single character when Width == 0.
		filterW := msg.Width - 4
		if filterW < 1 {
			filterW = 1
		}
		m.filterInput.SetWidth(filterW)
		// Re-clamp scroll offset after the viewport height changes.
		m = m.reconcileScroll()
		return m, nil

	case listTasksResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		// Preserve cursor by task id.
		var curID int64
		if len(m.visible) > 0 && m.cursor < len(m.visible) {
			curID = m.visible[m.cursor].node.Task.Id
		}
		m.tree = msg.tree
		m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
		if curID != 0 {
			m.cursor = findCursor(m.visible, curID)
		} else {
			m.cursor = clampCursor(m.cursor, len(m.visible))
		}
		// Re-clamp scroll offset after the list is rebuilt.
		m = m.reconcileScroll()
		return m, nil

	case refreshedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.mode = modeList
			return m, nil
		}
		prevCursor := m.cursor
		m.tree = msg.tree
		m.err = msg.partialErr
		m.mode = modeList
		m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
		if msg.highlightID != 0 {
			m.cursor = findCursor(m.visible, msg.highlightID)
			// If it's a subtask, ensure parent is expanded.
			if msg.highlightID != 0 {
				m.ensureVisible(msg.highlightID)
				m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
				m.cursor = findCursor(m.visible, msg.highlightID)
			}
		} else {
			m.cursor = clampCursor(prevCursor, len(m.visible))
		}
		// Re-clamp scroll offset after the list is rebuilt.
		m = m.reconcileScroll()
		// Re-fire filter after mutation so results stay consistent.
		if m.filterExpr != "" {
			m2, cmd := m.applyFilter(m.filterExpr, filterKeepCursor)
			return m2, cmd
		}
		return m, nil

	case pomTickMsg:
		if m.pom == nil || m.pom.completed {
			return m, nil
		}
		remaining := pomodoroRemaining(m.pom.startAt, time.Now())
		if remaining == 0 && !m.pom.completed {
			m.pom.completed = true
			taskName := m.pom.taskName
			return m, completePomCmd(m.client, taskName)
		}
		return m, pomTickCmd()

	case pomStartedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.pom = &activePom{
			taskID:   msg.taskID,
			taskName: msg.taskName,
			startAt:  msg.startAt,
		}
		m.err = nil
		return m, tea.Batch(pomTickCmd(), runPomHook(m.pomConfig.OnStart, "on_start"))

	case pomCancelledMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		hookCmd := runPomHook(m.pomConfig.OnCancel, "on_cancel")
		m.pom = nil
		m.err = nil
		return m, hookCmd

	case pomCompletedMsg:
		if m.pom != nil {
			banner := "🍅 Pomodoro complete! · " + msg.taskName
			if !m.styled {
				banner = "Pomodoro complete! " + msg.taskName
			}
			m.pom.banner = banner
		}
		if msg.err != nil {
			m.err = msg.err
		}
		hookCmd := runPomHook(m.pomConfig.OnComplete, "on_complete")
		return m, tea.Batch(hookCmd, pomBannerExpireCmd(), listTasksCmd(m.client))

	case pomBannerExpireMsg:
		m.pom = nil
		return m, nil

	case pomHookErrMsg:
		m.err = msg.err
		return m, nil

	case pomActiveMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		if msg.pom != nil {
			m.pom = msg.pom
			return m, pomTickCmd()
		}
		return m, nil

	case scheduledDaysResultMsg:
		if msg.err == nil {
			m.scheduledDays = msg.days
		}
		return m, nil

	case reorderResultMsg:
		return m.handleReorderResult(msg)

	case moveTaskResultMsg:
		return m.handleMoveTaskResult(msg)

	case filterResultMsg:
		return m.handleFilterResult(msg)

	case editSavedMsg:
		return m.handleEditSaved(msg)

	case editorFinishedMsg:
		if m.goal.compose.active {
			return m.handleGoalStatusEditorFinished(msg)
		}
		if m.mode == modeEdit || m.mode == modeNewSubtask || m.mode == modeNewRoot {
			if msg.err != nil {
				m.err = fmt.Errorf("couldn't open the editor — your description is safe, though! (%w)", msg.err)
			} else {
				m.edit.description.SetValue(msg.content)
				m.err = nil
			}
		}
		return m, nil

	case editCancelledMsg:
		if m.activeTab == tabGoals {
			m.goal.mode = goalList
		}
		m.mode = modeList
		m.cursor = msg.originalCursor
		m.err = nil
		m.confirmingDiscard = false
		return m, nil

	case editDiscardRequestedMsg:
		m.confirmingDiscard = true
		return m, nil

	case planEntriesMsg:
		m = m.handlePlanEntriesMsg(msg, msg.highlightID)
		return m, nil

	case planTasksMsg:
		if msg.err != nil {
			m.plan.err = msg.err
			m.plan.mode = planList
			return m, nil
		}
		// Expand all nodes so sub-tasks are visible in the picker.
		expanded := allTaskIDs(msg.tree)
		m.plan.picker = pickerState{
			tree:     msg.tree,
			visible:  buildVisible(msg.tree, expanded, false, nil, time.Now().Local()),
			cursor:   0,
			expanded: expanded,
		}
		// mode was already set to planPickTask by initAddTaskForm
		return m, nil

	case planMutatedMsg:
		if msg.err != nil {
			if msg.tabAgnosticErr {
				m.err = msg.err
			} else {
				m.plan.err = msg.err
			}
			return m, nil
		}
		// Success: close any open form and reload the day.
		m.plan.mode = planList
		if msg.notice != "" {
			m.notice = msg.notice
		}
		return m, tea.Batch(listPlanHighlightCmd(m.planClient, m.plan.day, msg.highlightID), listScheduledDaysCmd(m.planClient))

	case planTickMsg:
		if m.activeTab == tabPlanning && planIsToday(m.plan.day) {
			return m, planTickCmd()
		}
		return m, nil

	case tea.PasteMsg:
		return m.handlePaste(msg)

	case reportResultMsg:
		m.reportData.err = msg.err
		if msg.err == nil {
			m.reportData.dayGroups = msg.dayGroups
			m.reportData.finished = msg.finished
			m.reportData.ongoing = msg.ongoing
			m.reportData.totals = msg.totals
			m.reportData.period = msg.period
			m.reportData.scroll = 0
		}
		m.reportData.loaded = true
		return m, nil

	case listGoalsResultMsg:
		m.goal.err = msg.err
		if msg.err == nil {
			m.goal.goals = msg.goals
		}
		m.goal.loaded = true
		visible := visibleGoals(m.goal.goals, m.goal.showAll)
		if msg.highlightID != 0 {
			for i, g := range visible {
				if g.Id == msg.highlightID {
					m.goal.cursor = i
					return m, nil
				}
			}
		}
		m.goal.cursor = clampCursor(m.goal.cursor, len(visible))
		return m, nil

	case goalMutationMsg:
		if msg.err != nil {
			m.goal.err = msg.err
			return m, nil
		}
		m.goal.err = nil
		m.goal.mode = goalList
		m.mode = modeList
		// Re-fetch to get updated list.
		return m, listGoalsCmd(m.goalClient)

	case goalDeletedMsg:
		if msg.err != nil {
			m.goal.err = msg.err
			return m, nil
		}
		m.goal.err = nil
		m.goal.mode = goalList
		return m, listGoalsCmd(m.goalClient)

	case goalPickerTasksMsg:
		if msg.err != nil {
			m.goal.err = msg.err
			m.goal.mode = goalList
			return m, nil
		}
		expanded := allTaskIDs(msg.tree)
		m.goal.picker = pickerState{
			tree:     msg.tree,
			visible:  buildVisible(msg.tree, expanded, false, nil, time.Now().Local()),
			cursor:   0,
			expanded: expanded,
		}
		return m, nil

	case taskGoalMutationMsg:
		if msg.err != nil {
			var ce *connect.Error
			if errors.As(msg.err, &ce) && ce.Code() == connect.CodeFailedPrecondition {
				m.notice = cli.UserMessage(msg.err)
			} else if m.activeTab == tabGoals {
				m.goal.err = msg.err
			} else {
				// On the Tasks tab goal.err is invisible; surface the error where it
				// will actually be shown.
				m.err = msg.err
			}
			m.goal.mode = goalList
			m.mode = modeList
			// Re-fetch tasks so the list reflects the partial update (name/due were
			// already persisted by UpdateTask before SetTaskGoal failed).
			return m, listTasksCmd(m.client)
		}
		m.goal.err = nil
		m.goal.mode = goalList
		// Re-fetch both tasks (to update goal_id fields) and goals (for detail pane).
		return m, tea.Batch(listTasksCmd(m.client), listGoalsCmd(m.goalClient))

	case goalStatusListMsg:
		if msg.err != nil {
			m.goal.err = msg.err
			return m, nil
		}
		m.goal.statusUpdates = msg.updates
		m.goal.statusCursor = 0
		m.goal.readerOffset = 0
		m.goal.mode = goalStatusHistory
		return m, nil

	case goalStatusMutationMsg:
		if msg.err != nil {
			m.goal.err = msg.err
			m.goal.compose = statusCompose{}
			return m, nil
		}
		m.goal.err = nil
		m.goal.compose = statusCompose{}
		// Refresh the goal list (embedded latest) and, if in history view, the history.
		var cmds []tea.Cmd
		cmds = append(cmds, listGoalsCmd(m.goalClient))
		if m.goal.mode == goalStatusHistory || m.goal.mode == goalStatusReader || m.goal.mode == goalStatusConfirmDel {
			cmds = append(cmds, listGoalStatusUpdatesCmd(m.goalClient, msg.goalID))
			// Stay in history mode; the list will update when goalStatusListMsg arrives.
		}
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handlePaste routes bracketed-paste text (tea.PasteMsg, new in bubbletea v2)
// to the currently focused text input. Without this the message is dropped
// because PasteMsg is no longer a tea.KeyPressMsg.
func (m Model) handlePaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	if m.activeTab == tabGoals {
		if m.goal.mode == goalEdit || m.goal.mode == goalNew || m.goal.mode == goalNewTask {
			newEdit, cmd := m.edit.Update(msg, m.keys)
			m.edit = newEdit
			return m, cmd
		}
		return m, nil
	}
	if m.activeTab == tabPlanning {
		switch m.plan.mode {
		case planTaskTime, planEventForm, planEdit:
			if m.plan.form.focus < len(m.plan.form.fields) {
				var cmd tea.Cmd
				m.plan.form.fields[m.plan.form.focus], cmd =
					m.plan.form.fields[m.plan.form.focus].Update(msg)
				return m, cmd
			}
		}
		return m, nil
	}

	switch m.mode {
	case modeEdit, modeNewSubtask, modeNewRoot:
		newEdit, cmd := m.edit.Update(msg, m.keys)
		m.edit = newEdit
		return m, cmd
	case modeDatePrompt:
		// While the calendar is open, drop paste messages so the text field
		// can't change out from under it (mirrors editFormModel behaviour).
		if m.datePromptCalendar != nil {
			return m, nil
		}
		var cmd tea.Cmd
		m.datePromptInput, cmd = m.datePromptInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.notice = "" // clear transient notice on every user action

	// Discard confirmation overlay intercept.
	if m.confirmingDiscard {
		switch msg.String() {
		case "y":
			m.confirmingDiscard = false
			if m.activeTab == tabPlanning {
				m.plan.mode = planList
				m.plan.err = nil
				return m, nil
			}
			m.mode = modeList
			m.err = nil
			if m.activeTab == tabGoals {
				m.goal.mode = goalList
			}
			return m, func() tea.Msg {
				return editCancelledMsg{originalCursor: m.edit.originalCursor}
			}
		case "n", "esc":
			m.confirmingDiscard = false
		}
		return m, nil
	}

	if m.activeTab == tabGoals {
		return m.handleGoalsKey(msg)
	}
	if m.activeTab == tabPlanning {
		return m.handlePlanningKey(msg)
	}
	if m.activeTab == tabReport {
		return m.handleReportKey(msg)
	}
	switch m.mode {
	case modeList:
		return m.handleListKey(msg)
	case modeEdit, modeNewSubtask, modeNewRoot:
		return m.handleEditKey(msg)
	case modeHelp:
		return m.handleHelpKey(msg)
	case modeMove:
		return m.handleMoveKey(msg)
	case modeDatePrompt:
		return m.handleDatePromptKey(msg)
	case modeFilter:
		return m.handleFilterKey(msg)
	}
	return m, nil
}

// handleReportKey handles all key events while the Report tab is active.
func (m Model) handleReportKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Handle quit confirmation overlay.
	if m.confirmingQuit {
		switch msg.String() {
		case "y":
			return m, m.saveAndQuitCmd()
		case "n", "esc":
			m.confirmingQuit = false
		}
		return m, nil
	}

	if m.mode == modeHelp {
		m.mode = modeList
		return m, nil
	}

	now := m.nowOrDefault()
	presets := report.PresetOrder()

	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.pom != nil && !m.pom.completed {
			m.confirmingQuit = true
			return m, nil
		}
		return m, m.saveAndQuitCmd()

	case key.Matches(msg, m.keys.NextTab):
		m.activeTab = tabGoals
		m.keys.ReportMode = false
		m.keys.PlanningMode = false
		m.keys.GoalMode = true
		m.reportData.err = nil
		if !m.goal.loaded {
			return m, listGoalsCmd(m.goalClient)
		}
		return m, nil

	case key.Matches(msg, m.keys.PrevTab):
		m.activeTab = tabPlanning
		m.keys.ReportMode = false
		m.keys.PlanningMode = true
		m.reportData.err = nil
		return m, tea.Batch(listPlanCmd(m.planClient, m.plan.day))

	case key.Matches(msg, m.keys.ReportPrevPreset):
		idx := (m.reportData.presetIdx - 1 + len(presets)) % len(presets)
		m.reportData.presetIdx = idx
		m.reportData.loaded = false
		p, _ := report.ParsePeriod(presets[idx], "", "", now)
		m.reportData.period = p
		return m, fetchReportCmd(m.client, p)

	case key.Matches(msg, m.keys.ReportNextPreset):
		idx := (m.reportData.presetIdx + 1) % len(presets)
		m.reportData.presetIdx = idx
		m.reportData.loaded = false
		p, _ := report.ParsePeriod(presets[idx], "", "", now)
		m.reportData.period = p
		return m, fetchReportCmd(m.client, p)

	case key.Matches(msg, m.keys.Refresh):
		m.reportData.loaded = false
		p, _ := report.ParsePeriod(presets[m.reportData.presetIdx], "", "", now)
		m.reportData.period = p
		return m, fetchReportCmd(m.client, p)

	case key.Matches(msg, m.keys.Up):
		if m.reportData.scroll > 0 {
			m.reportData.scroll--
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		maxScroll := len(m.renderReportLines(m.width)) - m.reportBodyHeight()
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.reportData.scroll < maxScroll {
			m.reportData.scroll++
		}
		return m, nil

	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp
		return m, nil
	}
	return m, nil
}

// handleGoalsKey handles all key events while the Goals tab is active.
func (m Model) handleGoalsKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// If in edit/new form mode (goal or task), route to edit handler.
	if m.goal.mode == goalEdit || m.goal.mode == goalNew || m.goal.mode == goalNewTask {
		newEdit, cmd := m.edit.Update(msg, m.keys)
		m.edit = newEdit
		return m, cmd
	}

	// Status-update history/reader/delete-confirm modes.
	if m.goal.mode == goalStatusHistory || m.goal.mode == goalStatusReader || m.goal.mode == goalStatusConfirmDel {
		return m.handleGoalStatusKey(msg)
	}

	// Picker modes (L: link, U: unlink).
	if m.goal.mode == goalPickLink || m.goal.mode == goalPickUnlink {
		return m.handleGoalPickerKey(msg)
	}

	// Confirm-delete mode.
	if m.goal.mode == goalConfirmDelete {
		switch msg.String() {
		case "y":
			visible := visibleGoals(m.goal.goals, m.goal.showAll)
			if len(visible) > 0 && m.goal.cursor < len(visible) {
				id := visible[m.goal.cursor].Id
				m.goal.mode = goalList
				return m, deleteGoalCmd(m.goalClient, id)
			}
			m.goal.mode = goalList
		case "n", "esc":
			m.goal.mode = goalList
		}
		return m, nil
	}

	// Help overlay.
	if m.mode == modeHelp {
		m.mode = modeList
		return m, nil
	}

	// Quit confirmation.
	if m.confirmingQuit {
		switch msg.String() {
		case "y":
			return m, m.saveAndQuitCmd()
		case "n", "esc":
			m.confirmingQuit = false
		}
		return m, nil
	}

	visible := visibleGoals(m.goal.goals, m.goal.showAll)

	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.pom != nil && !m.pom.completed {
			m.confirmingQuit = true
			return m, nil
		}
		return m, m.saveAndQuitCmd()

	case key.Matches(msg, m.keys.NextTab):
		// Goals → Tasks
		m.activeTab = tabTasks
		m.keys.GoalMode = false
		m.goal.err = nil
		return m, nil

	case key.Matches(msg, m.keys.PrevTab):
		// Goals → Report (wrap around)
		m.activeTab = tabReport
		m.keys.GoalMode = false
		m.keys.ReportMode = true
		m.goal.err = nil
		now := m.nowOrDefault()
		presets := report.PresetOrder()
		p, _ := report.ParsePeriod(presets[m.reportData.presetIdx], "", "", now)
		m.reportData.period = p
		m.reportData.loaded = false
		return m, fetchReportCmd(m.client, p)

	case key.Matches(msg, m.keys.Up):
		if m.goal.cursor > 0 {
			m.goal.cursor--
		}

	case key.Matches(msg, m.keys.Down):
		if m.goal.cursor < len(visible)-1 {
			m.goal.cursor++
		}

	case key.Matches(msg, m.keys.GoalToggleAll):
		m.goal.showAll = !m.goal.showAll
		m.goal.cursor = clampCursor(m.goal.cursor, len(visibleGoals(m.goal.goals, m.goal.showAll)))

	// Space: toggle complete for goals (Active → Completed, Completed → Active).
	case key.Matches(msg, m.keys.Complete):
		if len(visible) > 0 {
			g := visible[m.goal.cursor]
			id := g.Id
			switch g.GetState() {
			case goalv1.GoalState_GOAL_STATE_COMMITTED, goalv1.GoalState_GOAL_STATE_INCUBATING, goalv1.GoalState_GOAL_STATE_HOLD:
				m.notice = "Goal achieved — take a bow!"
				return m, setGoalStateCmd(m.goalClient, id, goalv1.GoalState_GOAL_STATE_COMPLETED)
			case goalv1.GoalState_GOAL_STATE_COMPLETED:
				return m, setGoalStateCmd(m.goalClient, id, goalv1.GoalState_GOAL_STATE_COMMITTED)
			}
		}

	case key.Matches(msg, m.keys.GoalNew):
		m.originalCursor = m.goal.cursor
		m.edit = NewRootForm(m.goal.cursor, nil)
		m.edit.isGoal = true
		m.goal.mode = goalNew
		m.mode = modeNewRoot
		m.goal.err = nil

	case key.Matches(msg, m.keys.GoalEdit):
		if len(visible) > 0 {
			g := visible[m.goal.cursor]
			// Build a fake task to reuse the edit form.
			fakeTask := goalToFakeTask(g)
			m.originalCursor = m.goal.cursor
			m.edit = NewEditForm(fakeTask, m.goal.cursor, nil, g.GetState())
			m.edit.isGoal = true
			m.goal.mode = goalEdit
			m.mode = modeEdit
			m.goal.err = nil
		}

	case key.Matches(msg, m.keys.GoalDelete):
		if len(visible) > 0 {
			m.goal.mode = goalConfirmDelete
			m.notice = "Delete this goal? Tasks attached to it will stick around. [y]es [n]o"
		}

	case key.Matches(msg, m.keys.GoalRankUp):
		if len(visible) > 0 && m.goal.cursor > 0 {
			cur := visible[m.goal.cursor]
			prev := visible[m.goal.cursor-1]
			// Only reorder within same state group.
			if cur.State == prev.State {
				return m, reorderGoalCmd(m.goalClient, cur.Id, prev.Id, true)
			}
		}

	case key.Matches(msg, m.keys.GoalRankDown):
		if len(visible) > 0 && m.goal.cursor < len(visible)-1 {
			cur := visible[m.goal.cursor]
			next := visible[m.goal.cursor+1]
			// Only reorder within same state group.
			if cur.State == next.State {
				return m, reorderGoalCmd(m.goalClient, cur.Id, next.Id, false)
			}
		}

	case key.Matches(msg, m.keys.GoalAddTask):
		// `a`: open a new-task form; on save, create task and link to current goal.
		if len(visible) > 0 {
			m.originalCursor = m.goal.cursor
			m.edit = NewRootForm(m.goal.cursor, nil)
			m.goal.mode = goalNewTask
			m.mode = modeNewRoot
			m.goal.err = nil
		}

	case key.Matches(msg, m.keys.GoalGoToTasks):
		// ctrl+t: jump to Tasks filtered to this goal's whole tree.
		if len(visible) == 0 || m.goal.cursor >= len(visible) {
			return m, nil // FR-006: no goal under cursor, do nothing
		}
		g := visible[m.goal.cursor]

		m.activeTab = tabTasks  // FR-001
		m.keys.GoalMode = false // help flags follow the tab
		m.goal.err = nil
		m.err = nil
		m.mode = modeList

		m2, cmd := m.applyFilter(fmt.Sprintf("^goal_id=%d", g.Id), filterFirstMatch)
		return m2, cmd

	case key.Matches(msg, m.keys.GoalLinkTask):
		// `L`: open task picker; on selection, call SetTaskGoal.
		if len(visible) > 0 {
			m.goal.mode = goalPickLink
			m.goal.err = nil
			return m, goalListTasksForPickerCmd(m.client)
		}

	case key.Matches(msg, m.keys.GoalUnlinkTask):
		// `U`: open picker of the current goal's association roots; on selection, clear goal.
		if len(visible) > 0 {
			g := visible[m.goal.cursor]
			allTasks := flattenTree(m.tree)
			roots := goal.AssociationRoots(allTasks, g.GetId())
			if len(roots) == 0 {
				m.notice = "No tasks linked to this goal yet."
				return m, nil
			}
			var rootNodes []*cli.TreeNode
			for _, t := range roots {
				rootNodes = append(rootNodes, &cli.TreeNode{Task: t})
			}
			expanded := allTaskIDs(rootNodes)
			m.goal.picker = pickerState{
				tree:     rootNodes,
				visible:  buildVisible(rootNodes, expanded, true, nil, time.Now().Local()),
				cursor:   0,
				expanded: expanded,
			}
			m.goal.mode = goalPickUnlink
			m.goal.err = nil
		}

	case key.Matches(msg, m.keys.GoalAddStatus):
		// `S`: quick-add a status update via $EDITOR.
		if len(visible) > 0 {
			g := visible[m.goal.cursor]
			m.goal.compose = statusCompose{goalID: g.Id, active: true}
			m.goal.err = nil
			return m, openEditorCmd("")
		}

	case key.Matches(msg, m.keys.GoalStatusHistory):
		// `s`: open status-update history view.
		if len(visible) > 0 {
			g := visible[m.goal.cursor]
			m.goal.err = nil
			return m, listGoalStatusUpdatesCmd(m.goalClient, g.Id)
		}

	case key.Matches(msg, m.keys.Refresh):
		m.goal.loaded = false
		m.goal.err = nil
		return m, listGoalsCmd(m.goalClient)

	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp
	}

	return m, nil
}

// handleGoalStatusEditorFinished routes the editor result from a status-update compose/edit.
func (m Model) handleGoalStatusEditorFinished(msg editorFinishedMsg) (tea.Model, tea.Cmd) {
	compose := m.goal.compose
	m.goal.compose = statusCompose{} // always clear

	if msg.err != nil {
		m.goal.err = fmt.Errorf("editor trouble — your update wasn't saved. (%w)", msg.err)
		return m, nil
	}

	body := strings.TrimSpace(msg.content)
	if body == "" {
		m.notice = "Nothing written — your status update was not recorded."
		return m, nil
	}

	if compose.editingID > 0 {
		return m, updateGoalStatusUpdateCmd(m.goalClient, compose.editingID, body)
	}
	return m, addGoalStatusUpdateCmd(m.goalClient, compose.goalID, body)
}

// handleGoalStatusKey handles all key events in the status-history, reader, and confirm-delete modes.
func (m Model) handleGoalStatusKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.goal.mode {
	case goalStatusConfirmDel:
		switch msg.String() {
		case "y":
			if m.goal.statusCursor < len(m.goal.statusUpdates) {
				su := m.goal.statusUpdates[m.goal.statusCursor]
				visible := visibleGoals(m.goal.goals, m.goal.showAll)
				var goalID int64
				if len(visible) > 0 && m.goal.cursor < len(visible) {
					goalID = visible[m.goal.cursor].Id
				}
				m.goal.mode = goalStatusHistory
				return m, deleteGoalStatusUpdateCmd(m.goalClient, su.Id, goalID)
			}
			m.goal.mode = goalStatusHistory
		case "n", "esc":
			m.goal.mode = goalStatusHistory
		}
		return m, nil

	case goalStatusReader:
		switch {
		case key.Matches(msg, m.keys.Cancel):
			m.goal.mode = goalStatusHistory
			m.goal.readerOffset = 0
		case key.Matches(msg, m.keys.Up):
			if m.goal.readerOffset > 0 {
				m.goal.readerOffset--
			}
		case key.Matches(msg, m.keys.Down):
			maxOffset := len(m.statusReaderLines()) - m.statusReaderContentHeight()
			if maxOffset < 0 {
				maxOffset = 0
			}
			if m.goal.readerOffset < maxOffset {
				m.goal.readerOffset++
			}
		case msg.String() == "pgup":
			m.goal.readerOffset -= 10
			if m.goal.readerOffset < 0 {
				m.goal.readerOffset = 0
			}
		case msg.String() == "pgdown":
			maxOffset := len(m.statusReaderLines()) - m.statusReaderContentHeight()
			if maxOffset < 0 {
				maxOffset = 0
			}
			m.goal.readerOffset += 10
			if m.goal.readerOffset > maxOffset {
				m.goal.readerOffset = maxOffset
			}
		}
		return m, nil

	case goalStatusHistory:
		switch {
		case key.Matches(msg, m.keys.Cancel):
			m.goal.mode = goalList
			m.goal.statusUpdates = nil
			m.goal.statusCursor = 0
			m.goal.readerOffset = 0

		case key.Matches(msg, m.keys.Up):
			if m.goal.statusCursor > 0 {
				m.goal.statusCursor--
			}

		case key.Matches(msg, m.keys.Down):
			if m.goal.statusCursor < len(m.goal.statusUpdates)-1 {
				m.goal.statusCursor++
			}

		case msg.Code == tea.KeyEnter:
			if len(m.goal.statusUpdates) > 0 {
				m.goal.mode = goalStatusReader
				m.goal.readerOffset = 0
			}

		case key.Matches(msg, m.keys.GoalAddStatus):
			// `S` (or the GoalStatusNew binding): add a new update from history view.
			visible := visibleGoals(m.goal.goals, m.goal.showAll)
			if len(visible) > 0 {
				g := visible[m.goal.cursor]
				m.goal.compose = statusCompose{goalID: g.Id, active: true}
				return m, openEditorCmd("")
			}

		case key.Matches(msg, m.keys.GoalStatusEdit):
			if len(m.goal.statusUpdates) > 0 && m.goal.statusCursor < len(m.goal.statusUpdates) {
				su := m.goal.statusUpdates[m.goal.statusCursor]
				visible := visibleGoals(m.goal.goals, m.goal.showAll)
				var goalID int64
				if len(visible) > 0 && m.goal.cursor < len(visible) {
					goalID = visible[m.goal.cursor].Id
				}
				m.goal.compose = statusCompose{goalID: goalID, editingID: su.Id, active: true}
				return m, openEditorCmd(su.Body)
			}

		case key.Matches(msg, m.keys.GoalStatusDelete):
			if len(m.goal.statusUpdates) > 0 {
				m.goal.mode = goalStatusConfirmDel
				m.notice = "Delete this status update? [y]es [n]o"
			}
		}
		return m, nil
	}
	return m, nil
}

// handleGoalPickerKey handles key events in the goal-tab task picker (L and U flows).
func (m Model) handleGoalPickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.goal.mode = goalList
		m.goal.err = nil
	case key.Matches(msg, m.keys.Up):
		if m.goal.picker.cursor > 0 {
			m.goal.picker.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.goal.picker.cursor < len(m.goal.picker.visible)-1 {
			m.goal.picker.cursor++
		}
	case msg.Code == tea.KeyEnter:
		if len(m.goal.picker.visible) == 0 {
			m.goal.mode = goalList
			return m, nil
		}
		row := m.goal.picker.visible[m.goal.picker.cursor]
		taskID := row.node.Task.Id

		visible := visibleGoals(m.goal.goals, m.goal.showAll)
		if len(visible) == 0 || m.goal.cursor >= len(visible) {
			m.goal.mode = goalList
			return m, nil
		}
		goalID := visible[m.goal.cursor].Id
		prevMode := m.goal.mode
		m.goal.mode = goalList

		if prevMode == goalPickLink {
			return m, setTaskGoalAndRefreshCmd(m.client, taskID, &goalID)
		}
		// goalPickUnlink: clear the goal association.
		return m, setTaskGoalAndRefreshCmd(m.client, taskID, nil)
	}
	return m, nil
}

// handleGoalNewTaskSaved handles an editSavedMsg when creating a task attached to a goal.
func (m Model) handleGoalNewTaskSaved(msg editSavedMsg) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(msg.name) == "" {
		m.goal.err = fmt.Errorf("task name cannot be empty")
		return m, nil
	}
	visible := visibleGoals(m.goal.goals, m.goal.showAll)
	if len(visible) == 0 || m.goal.cursor >= len(visible) {
		m.goal.mode = goalList
		m.mode = modeList
		return m, nil
	}
	goalID := visible[m.goal.cursor].Id
	m.goal.err = nil
	m.goal.mode = goalList
	m.mode = modeList
	return m, createAndLinkTaskCmd(m.client, msg, goalID)
}

// handleGoalEditSaved handles an editSavedMsg when the Goals tab is active.
func (m Model) handleGoalEditSaved(msg editSavedMsg) (tea.Model, tea.Cmd) {
	if strings.TrimSpace(msg.name) == "" {
		m.goal.err = fmt.Errorf("goal name cannot be empty")
		return m, nil
	}

	var due *timestamppb.Timestamp
	if msg.dueStr != "" {
		ts, err := cli.ParseDue(msg.dueStr)
		if err != nil {
			m.goal.err = err
			return m, nil
		}
		due = ts
	}

	m.goal.err = nil

	if m.goal.mode == goalEdit {
		// Find the goal being edited.
		visible := visibleGoals(m.goal.goals, m.goal.showAll)
		if len(visible) > 0 && m.goal.cursor < len(visible) {
			id := visible[m.goal.cursor].Id
			if msg.goalStateChanged {
				return m, updateGoalThenSetStateCmd(m.goalClient, id, msg.name, msg.description, due, msg.goalState)
			}
			return m, updateGoalCmd(m.goalClient, id, msg.name, msg.description, due)
		}
	}

	// New goal.
	return m, createGoalCmd(m.goalClient, msg.name, msg.description, due)
}

// handlePlanningKey handles all key events while the Planning tab is active.
func (m Model) handlePlanningKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// While a modal is open, route to the modal handler before any global
	// keybinding (including Quit) so typing into a focused field can't leak
	// into e.g. quitting the app. Tab-switch is ignored while a modal is open.
	if m.plan.mode != planList {
		return m.handlePlanModalKey(msg)
	}

	// Dismiss help overlay — any key closes it, mirroring handleHelpKey.
	if m.mode == modeHelp {
		m.mode = modeList
		return m, nil
	}

	// Handle quit confirmation overlay (mirrors handleListKey).
	if m.confirmingQuit {
		switch msg.String() {
		case "y":
			return m, m.saveAndQuitCmd()
		case "n", "esc":
			m.confirmingQuit = false
		}
		return m, nil
	}

	// Quit is always available.
	if key.Matches(msg, m.keys.Quit) {
		if m.pom != nil && !m.pom.completed {
			m.confirmingQuit = true
			return m, nil
		}
		return m, m.saveAndQuitCmd()
	}

	// Tab → Report; Shift+Tab → Tasks.
	if key.Matches(msg, m.keys.NextTab) {
		m.activeTab = tabReport
		m.keys.PlanningMode = false
		m.keys.ReportMode = true
		m.plan.err = nil
		m.plan.pendingComplete = nil
		now := m.nowOrDefault()
		presets := report.PresetOrder()
		p, _ := report.ParsePeriod(presets[m.reportData.presetIdx], "", "", now)
		m.reportData.period = p
		m.reportData.loaded = false
		return m, fetchReportCmd(m.client, p)
	}
	if key.Matches(msg, m.keys.PrevTab) {
		m.activeTab = tabTasks
		m.keys.PlanningMode = false
		m.plan.err = nil
		m.plan.pendingComplete = nil
		return m, listScheduledDaysCmd(m.planClient)
	}

	// Navigation.
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.plan.cursor > 0 {
			m.plan.cursor--
		}
		if m.plan.pendingComplete != nil {
			m.plan.pendingComplete = nil
			m.plan.entries = displayedPlanEntries(m.plan.entries, nil)
			m.plan.cursor = clampCursor(m.plan.cursor, len(m.plan.entries))
		}
	case key.Matches(msg, m.keys.Down):
		if m.plan.cursor < len(m.plan.entries)-1 {
			m.plan.cursor++
		}
		if m.plan.pendingComplete != nil {
			m.plan.pendingComplete = nil
			m.plan.entries = displayedPlanEntries(m.plan.entries, nil)
			m.plan.cursor = clampCursor(m.plan.cursor, len(m.plan.entries))
		}
	case key.Matches(msg, m.keys.First):
		m.plan.cursor = clampCursor(0, len(m.plan.entries))
		if m.plan.pendingComplete != nil {
			m.plan.pendingComplete = nil
			m.plan.entries = displayedPlanEntries(m.plan.entries, nil)
			m.plan.cursor = clampCursor(m.plan.cursor, len(m.plan.entries))
		}
	case key.Matches(msg, m.keys.Last):
		m.plan.cursor = clampCursor(len(m.plan.entries)-1, len(m.plan.entries))
		if m.plan.pendingComplete != nil {
			m.plan.pendingComplete = nil
			m.plan.entries = displayedPlanEntries(m.plan.entries, nil)
			m.plan.cursor = clampCursor(m.plan.cursor, len(m.plan.entries))
		}

	// Day navigation.
	case key.Matches(msg, m.keys.PlanPrevDay):
		t, err := time.Parse("2006-01-02", m.plan.day)
		if err == nil {
			m.plan.pendingComplete = nil
			m.plan.day = t.AddDate(0, 0, -1).Format("2006-01-02")
			m.plan.loaded = false
			return m, listPlanCmd(m.planClient, m.plan.day)
		}
	case key.Matches(msg, m.keys.PlanNextDay):
		t, err := time.Parse("2006-01-02", m.plan.day)
		if err == nil {
			m.plan.pendingComplete = nil
			m.plan.day = t.AddDate(0, 0, 1).Format("2006-01-02")
			m.plan.loaded = false
			return m, listPlanCmd(m.planClient, m.plan.day)
		}
	case key.Matches(msg, m.keys.PlanToday):
		m.plan.pendingComplete = nil
		today := time.Now().Format("2006-01-02")
		if m.plan.day != today {
			m.plan.day = today
			m.plan.loaded = false
			return m, tea.Batch(listPlanCmd(m.planClient, m.plan.day), planTickCmd())
		}
	case key.Matches(msg, m.keys.Refresh):
		m.plan.loaded = false
		return m, listPlanCmd(m.planClient, m.plan.day)

	// Entry actions.
	case key.Matches(msg, m.keys.PlanAddTask):
		m.initAddTaskForm()
		return m, listTasksForPickerCmd(m.client)
	case key.Matches(msg, m.keys.PlanAddEvent):
		m.initAddEventForm()
	case key.Matches(msg, m.keys.PlanEdit):
		if len(m.plan.entries) > 0 {
			m.initEditForm()
		}
	case key.Matches(msg, m.keys.PlanRemove):
		if len(m.plan.entries) > 0 {
			entry := m.plan.entries[m.plan.cursor]
			return m, removePlanCmd(m.planClient, m.plan.day, entry.Id)
		}

	case key.Matches(msg, m.keys.PlanGoToTask):
		if len(m.plan.entries) == 0 {
			return m, nil
		}
		entry := m.plan.entries[m.plan.cursor]
		if entry.TaskId == 0 {
			return m, nil
		}
		m.activeTab = tabTasks
		m.keys.PlanningMode = false
		m.plan.err = nil
		m.plan.pendingComplete = nil
		m.filterExpr = ""
		m.filterMatches = nil
		m.filteredIDs = nil
		m.filterInvalid = false
		m.filterGen++
		m.ensureVisible(entry.TaskId)
		m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
		m.cursor = findCursor(m.visible, entry.TaskId)
		return m, nil

	// Toggle complete on the linked task.
	case key.Matches(msg, m.keys.Complete):
		if len(m.plan.entries) > 0 {
			entry := m.plan.entries[m.plan.cursor]
			if entry.TaskId == 0 {
				m.notice = "That's an event, not a task — nothing to check off here."
				return m, nil
			}
			m.err = nil
			complete := !entry.Completed
			var notice string
			if complete {
				notice = "Nice — '" + entry.Name + "' is done and dusted."
				if entry.StartMinute == nil {
					m.plan.pendingComplete = &entry.Id
				}
			} else {
				notice = "Marked '" + entry.Name + "' as incomplete."
				m.plan.pendingComplete = nil
			}
			return m, completePlanTaskCmd(m.client, m.plan.day, entry.TaskId, complete, entry.Id, notice)
		}

	// Start a pomodoro for the linked task.
	case key.Matches(msg, m.keys.PomStart):
		if len(m.plan.entries) > 0 {
			entry := m.plan.entries[m.plan.cursor]
			if entry.TaskId == 0 {
				m.notice = "Events don't run on tomatoes — pick a task to focus on."
				return m, nil
			}
			m.err = nil
			return m, startPomCmd(m.client, entry.TaskId, entry.Name)
		}

	// Auto-schedule: place/re-home the highlighted task in the earliest free slot.
	case key.Matches(msg, m.keys.PlanAutoSchedule):
		if len(m.plan.entries) == 0 {
			return m, nil
		}
		entry := m.plan.entries[m.plan.cursor]
		if entry.TaskId == 0 {
			m.notice = "Auto-schedule is for tasks only — events are already right where they belong."
			return m, nil
		}
		_, timed := splitPlanEntries(m.plan.entries)
		floorMin := 480
		now := m.nowOrDefault()
		if m.plan.day == now.Format("2006-01-02") {
			nowMin := now.Hour()*60 + now.Minute()
			if nowMin > floorMin {
				floorMin = nowMin
			}
		}
		floorMin = (floorMin / 15) * 15
		if floorMin < 480 {
			floorMin = 480
		}
		startMin, ok := cli.AutoScheduleSlot(timed, int(entry.DurationMinute), floorMin, entry.Id)
		if !ok {
			m.notice = "Day's packed — no room left to squeeze this one in."
			return m, nil
		}
		if entry.StartMinute != nil && int(*entry.StartMinute) == startMin {
			// Already in earliest slot — bump past the next closing entry.
			bumpFloor, ok2 := cli.NextGapFloor(timed, int(*entry.StartMinute), entry.Id)
			if !ok2 {
				m.notice = "Day's packed — no room left to squeeze this one in."
				return m, nil
			}
			start2, ok3 := cli.AutoScheduleSlot(timed, int(entry.DurationMinute), bumpFloor, entry.Id)
			if !ok3 {
				m.notice = "Day's packed — no room left to squeeze this one in."
				return m, nil
			}
			return m, movePlanCmd(m.planClient, m.plan.day, entry.Id, start2, int(entry.DurationMinute), true)
		}
		return m, movePlanCmd(m.planClient, m.plan.day, entry.Id, startMin, int(entry.DurationMinute), true)

	// Unschedule: clear the highlighted task's start time, keeping it on the day.
	case key.Matches(msg, m.keys.PlanUnschedule):
		if len(m.plan.entries) == 0 {
			return m, nil
		}
		entry := m.plan.entries[m.plan.cursor]
		if entry.TaskId == 0 {
			m.notice = "Unschedule is for tasks only — events keep their time."
			return m, nil
		}
		if entry.StartMinute == nil {
			m.notice = "That task is already unscheduled."
			return m, nil
		}
		return m, movePlanCmd(m.planClient, m.plan.day, entry.Id, 0, int(entry.DurationMinute), false)

	// Pomodoro cancel — mirrors the Tasks tab handler.
	case key.Matches(msg, m.keys.PomCancel):
		if m.pom != nil && !m.pom.completed {
			m.err = nil
			return m, cancelPomCmd(m.client)
		}

	// Reorder untimed entries.
	case key.Matches(msg, m.keys.RankUp):
		if len(m.plan.entries) > 0 {
			entry := m.plan.entries[m.plan.cursor]
			if entry.StartMinute == nil {
				if prev := prevVisibleUntimed(m.plan.entries, m.plan.cursor); prev != nil {
					m.err = nil
					return m, reorderPlanEntryCmd(m.planClient, m.plan.day, entry.Id, prev.Id, true)
				}
			}
		}

	case key.Matches(msg, m.keys.RankDown):
		if len(m.plan.entries) > 0 {
			entry := m.plan.entries[m.plan.cursor]
			if entry.StartMinute == nil {
				if next := nextVisibleUntimed(m.plan.entries, m.plan.cursor); next != nil {
					m.err = nil
					return m, reorderPlanEntryCmd(m.planClient, m.plan.day, entry.Id, next.Id, false)
				}
			}
		}

	// Help.
	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp
	}

	return m, nil
}

// handlePlanModalKey handles keys when a planning modal is open.
func (m Model) handlePlanModalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch m.plan.mode {
	case planPickTask:
		return m.handlePickerKey(msg)
	case planTaskTime, planEventForm, planEdit:
		return m.handlePlanFormKey(msg)
	}
	return m, nil
}

// handlePickerKey handles key events in the task picker.
func (m Model) handlePickerKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.plan.mode = planList
		m.plan.err = nil
	case key.Matches(msg, m.keys.Up):
		if m.plan.picker.cursor > 0 {
			m.plan.picker.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.plan.picker.cursor < len(m.plan.picker.visible)-1 {
			m.plan.picker.cursor++
		}
	case msg.Code == tea.KeyEnter:
		if len(m.plan.picker.visible) > 0 {
			row := m.plan.picker.visible[m.plan.picker.cursor]
			taskID := row.node.Task.Id
			m.initTaskTimeForm(taskID)
		} else {
			m.plan.mode = planList
		}
	}
	return m, nil
}

// handlePlanFormKey handles key events in a planning text-input form.
func (m Model) handlePlanFormKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Save):
		cmd := m.submitPlanForm()
		return m, cmd
	case msg.Code == tea.KeyEnter:
		if planFocusSave(m.plan.form) {
			cmd := m.submitPlanForm()
			return m, cmd
		}
		if planFocusCancel(m.plan.form) {
			m.plan.mode = planList
			m.plan.err = nil
			return m, nil
		}
		m.cyclePlanFormFocus(1)
		return m, nil
	case key.Matches(msg, m.keys.Tab):
		m.cyclePlanFormFocus(1)
		return m, nil
	case key.Matches(msg, m.keys.ShiftTab):
		m.cyclePlanFormFocus(-1)
		return m, nil
	case key.Matches(msg, m.keys.Cancel):
		if planFormDirty(m.plan.form) {
			m.confirmingDiscard = true
			return m, nil
		}
		m.plan.mode = planList
		m.plan.err = nil
		return m, nil
	}

	// Forward to focused text input only when a text field is focused.
	if m.plan.form.focus < len(m.plan.form.fields) {
		var cmd tea.Cmd
		m.plan.form.fields[m.plan.form.focus], cmd = m.plan.form.fields[m.plan.form.focus].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleListKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Handle quit confirmation overlay.
	if m.confirmingQuit {
		switch msg.String() {
		case "y":
			return m, m.saveAndQuitCmd()
		case "n", "esc":
			m.confirmingQuit = false
		}
		return m, nil
	}

	// Clear completion banner on any keypress (banner clears, key still acts).
	if m.pom != nil && m.pom.completed && m.pom.banner != "" {
		m.pom = nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.pom != nil && !m.pom.completed {
			m.confirmingQuit = true
			return m, nil
		}
		return m, m.saveAndQuitCmd()

	case key.Matches(msg, m.keys.NextTab):
		// Tasks → Planning
		m.activeTab = tabPlanning
		m.keys.PlanningMode = true
		m.keys.GoalMode = false
		m.err = nil
		var cmds []tea.Cmd
		cmds = append(cmds, listPlanCmd(m.planClient, m.plan.day))
		if planIsToday(m.plan.day) {
			cmds = append(cmds, planTickCmd())
		}
		return m, tea.Batch(cmds...)

	case key.Matches(msg, m.keys.PrevTab):
		// Tasks → Goals (wrapping backwards)
		m.activeTab = tabGoals
		m.keys.GoalMode = true
		m.err = nil
		if !m.goal.loaded {
			return m, listGoalsCmd(m.goalClient)
		}
		return m, nil

	case key.Matches(msg, m.keys.Cancel):
		if m.filterExpr != "" || m.filterInvalid {
			return m.clearFilter(), nil
		}

	case key.Matches(msg, m.keys.Up):
		m.pendingComplete = nil
		m.visible = m.rebuildVisible()
		if m.cursor > 0 {
			m.cursor--
		}
		m.cursor = clampCursor(m.cursor, len(m.visible))

	case key.Matches(msg, m.keys.Down):
		var targetID int64
		if m.cursor+1 < len(m.visible) {
			targetID = m.visible[m.cursor+1].node.Task.Id
		}
		m.pendingComplete = nil
		m.visible = m.rebuildVisible()
		if targetID != 0 {
			m.cursor = findCursor(m.visible, targetID)
		} else {
			m.cursor = clampCursor(m.cursor, len(m.visible))
		}

	case key.Matches(msg, m.keys.First):
		m.pendingComplete = nil
		m.visible = m.rebuildVisible()
		m.cursor = clampCursor(0, len(m.visible))

	case key.Matches(msg, m.keys.Last):
		m.pendingComplete = nil
		m.visible = m.rebuildVisible()
		m.cursor = clampCursor(len(m.visible)-1, len(m.visible))

	case key.Matches(msg, m.keys.PageUp):
		h := m.listViewportHeight()
		m.cursor -= h
		if m.cursor < 0 {
			m.cursor = 0
		}

	case key.Matches(msg, m.keys.PageDown):
		h := m.listViewportHeight()
		m.cursor += h
		m.cursor = clampCursor(m.cursor, len(m.visible))

	case key.Matches(msg, m.keys.Collapse):
		if len(m.visible) > 0 && m.filterExpr == "" {
			row := m.visible[m.cursor]
			id := row.node.Task.Id
			if len(row.node.Children) > 0 && m.expanded[id] {
				// Task is expanded — collapse it in place.
				m.expanded[id] = false
			} else if parent := findParentNode(m.tree, id); parent != nil {
				// Task is a leaf or already collapsed — collapse the parent and
				// move the cursor up to it.
				m.expanded[parent.Task.Id] = false
				m.visible = m.rebuildVisible()
				m.cursor = findCursor(m.visible, parent.Task.Id)
				m = m.reconcileScroll()
				return m, m.persistTreeStateCmd()
			}
			m.visible = m.rebuildVisible()
			m = m.reconcileScroll()
			return m, m.persistTreeStateCmd()
		}

	case key.Matches(msg, m.keys.Expand):
		if len(m.visible) > 0 && m.filterExpr == "" {
			id := m.visible[m.cursor].node.Task.Id
			m.expanded[id] = true
			m.visible = m.rebuildVisible()
			// Move to first child if it is visible immediately after the cursor.
			// Works whether the subtree was just expanded or was already open.
			if next := m.cursor + 1; next < len(m.visible) {
				if pid := m.visible[next].node.Task.ParentId; pid != nil && *pid == id {
					m.cursor = next
				}
			}
			m = m.reconcileScroll()
			return m, m.persistTreeStateCmd()
		}

	case key.Matches(msg, m.keys.Edit):
		if len(m.visible) > 0 {
			task := m.visible[m.cursor].node.Task
			m.originalCursor = m.cursor
			m.edit = NewEditForm(task, m.cursor, m.goal.goals, goalv1.GoalState_GOAL_STATE_UNSPECIFIED)
			m.mode = modeEdit
			m.err = nil
		}

	case key.Matches(msg, m.keys.NewSub):
		if len(m.visible) > 0 {
			parentID := m.visible[m.cursor].node.Task.Id
			m.originalCursor = m.cursor
			m.edit = NewSubtaskForm(parentID, m.cursor)
			m.mode = modeNewSubtask
			m.err = nil
		}

	case key.Matches(msg, m.keys.NewRoot):
		m.originalCursor = m.cursor
		m.edit = NewRootForm(m.cursor, m.goal.goals)
		m.mode = modeNewRoot
		m.err = nil

	case key.Matches(msg, m.keys.Move):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			if ms := newMoveState(&m, id); ms != nil {
				m.move = ms
				m.mode = modeMove
			}
		}

	case key.Matches(msg, m.keys.PomStart):
		if len(m.visible) > 0 {
			task := m.visible[m.cursor].node.Task
			m.err = nil
			return m, startPomCmd(m.client, task.Id, task.Name)
		}

	case key.Matches(msg, m.keys.PomCancel):
		if m.pom != nil && !m.pom.completed {
			m.err = nil
			return m, cancelPomCmd(m.client)
		}

	case key.Matches(msg, m.keys.PlanSendToday):
		if len(m.visible) > 0 {
			task := m.visible[m.cursor].node.Task
			today := time.Now().Format("2006-01-02")
			notice := fmt.Sprintf("Tucked '%s' into today's plan.", task.Name)
			m.err = nil
			return m, addPlanTaskCmd(m.planClient, today, task.Id, 0, 0, false, notice)
		}

	case key.Matches(msg, m.keys.PlanSendPickDay):
		if len(m.visible) > 0 {
			task := m.visible[m.cursor].node.Task
			m.datePromptTaskID = task.Id
			m.datePromptTaskName = task.Name
			tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
			ti := newPlanInput("YYYY-MM-DD")
			ti.SetValue(tomorrow)
			ti.Focus()
			m.datePromptInput = ti
			m.datePromptCalendar = nil
			m.err = nil
			m.mode = modeDatePrompt
		}

	case key.Matches(msg, m.keys.RankUp):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			prev, _ := visibleSiblings(m.tree, m.expanded, m.showAll, id)
			if prev != 0 {
				m.err = nil
				return m, reorderTaskCmd(m.client, id, prev, true)
			}
		}

	case key.Matches(msg, m.keys.RankDown):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			_, next := visibleSiblings(m.tree, m.expanded, m.showAll, id)
			if next != 0 {
				m.err = nil
				return m, reorderTaskCmd(m.client, id, next, false)
			}
		}

	case key.Matches(msg, m.keys.Delete):
		if len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			m.err = nil
			return m, deleteTaskCmd(m.client, id)
		}

	case key.Matches(msg, m.keys.Complete):
		if len(m.visible) > 0 {
			node := m.visible[m.cursor].node
			id := node.Task.Id
			m.err = nil
			if node.Task.GetCompletedAt() == nil {
				m.pendingComplete = &id
				return m, completeTaskCmd(m.client, id)
			}
			m.pendingComplete = nil
			return m, uncompleteTaskCmd(m.client, id)
		}

	case key.Matches(msg, m.keys.Refresh):
		m.pendingComplete = nil
		m.err = nil
		return m, tea.Batch(listTasksCmd(m.client), listScheduledDaysCmd(m.planClient))

	case key.Matches(msg, m.keys.ToggleAll):
		m.showAll = !m.showAll
		var curID int64
		if len(m.visible) > 0 {
			curID = m.visible[m.cursor].node.Task.Id
		}
		m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
		m.cursor = findCursor(m.visible, curID)
		// Re-fire the active filter so the filtered view stays consistent.
		if m.filterExpr != "" {
			m2, cmd := m.applyFilter(m.filterExpr, filterKeepCursor)
			return m2, cmd
		}

	case key.Matches(msg, m.keys.Filter):
		m.mode = modeFilter
		m.filterInput.SetValue(m.filterExpr)
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, m.keys.Help):
		m.mode = modeHelp

	default:
		// Digit keys 0-9 set the pomodoro estimate.
		if len(msg.Text) == 1 && msg.Text[0] >= '0' && msg.Text[0] <= '9' && len(m.visible) > 0 {
			id := m.visible[m.cursor].node.Task.Id
			est := int32(msg.Text[0] - '0')
			m.err = nil
			return m, setEstimateCmd(m.client, id, est)
		}
	}

	// Reconcile listScroll after any key that may have moved the cursor or
	// rebuilt the visible list (Up, Down, Home, End, PageUp, PageDown,
	// ToggleAll, Filter clear). Collapse and Expand reconcile explicitly at
	// their own early-returns above, since they stay in modeList and move the
	// cursor. Cases that switch modes (Edit, NewSub, NewRoot, Pom* etc.) do
	// not reach here, which is fine — their mode switch freezes the list
	// until the user returns.
	m = m.reconcileScroll()

	return m, nil
}

// handleDatePromptKey handles key events while the Tasks-tab date prompt is open.
func (m Model) handleDatePromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// When the calendar is open, route all keys to it first.
	if m.datePromptCalendar != nil {
		switch {
		case msg.Code == tea.KeyEscape:
			// Close calendar; leave text field unchanged.
			m.datePromptCalendar = nil
			return m, nil
		case msg.Code == tea.KeyEnter || key.Matches(msg, m.keys.Save):
			// Confirm selection: write date into the text input, close calendar.
			m.datePromptInput.SetValue(m.datePromptCalendar.confirm())
			m.datePromptCalendar = nil
			return m, nil
		case msg.Code == tea.KeyTab:
			// Tab: close calendar without changing the field.
			m.datePromptCalendar = nil
			return m, nil
		default:
			m.datePromptCalendar.handleKey(msg)
			return m, nil
		}
	}

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.mode = modeList
		m.datePromptCalendar = nil
		m.err = nil
		return m, nil
	case key.Matches(msg, m.keys.Calendar):
		// Open the calendar pre-seeded with the current field value.
		m.datePromptCalendar = newCalendar(m.datePromptInput.Value(), time.Now(), false)
		return m, nil
	case key.Matches(msg, m.keys.Save) || msg.Code == tea.KeyEnter:
		dateStr := strings.TrimSpace(m.datePromptInput.Value())
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			m.err = fmt.Errorf("hmm, that date didn't parse — try YYYY-MM-DD (e.g. 2026-06-15)")
			return m, nil
		}
		m.mode = modeList
		m.datePromptCalendar = nil
		m.err = nil
		notice := fmt.Sprintf("Tucked '%s' into %s's plan.", m.datePromptTaskName, dateStr)
		return m, addPlanTaskCmd(m.planClient, dateStr, m.datePromptTaskID, 0, 0, false, notice)
	}
	var cmd tea.Cmd
	m.datePromptInput, cmd = m.datePromptInput.Update(msg)
	return m, cmd
}

func (m Model) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	newEdit, cmd := m.edit.Update(msg, m.keys)
	m.edit = newEdit
	return m, cmd
}

func (m Model) handleHelpKey(_ tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.mode = modeList
	return m, nil
}

func (m Model) handleEditSaved(msg editSavedMsg) (tea.Model, tea.Cmd) {
	// Route to goal handlers if on Goals tab.
	if m.activeTab == tabGoals {
		if m.goal.mode == goalNewTask {
			return m.handleGoalNewTaskSaved(msg)
		}
		return m.handleGoalEditSaved(msg)
	}

	// Validate name.
	if strings.TrimSpace(msg.name) == "" {
		m.err = fmt.Errorf("task name cannot be empty")
		return m, nil
	}

	// Validate estimate if provided.
	if msg.estimateStr != "" {
		_, err := strconv.ParseInt(msg.estimateStr, 10, 32)
		if err != nil {
			m.err = fmt.Errorf("estimate must be an integer")
			return m, nil
		}
	}

	// Validate due if provided (ParseDue is called inside the cmd; we do a
	// pre-check here so validation errors stay in edit mode).
	if msg.dueStr != "" {
		if _, err := cli.ParseDue(msg.dueStr); err != nil {
			m.err = err
			return m, nil
		}
	}

	if msg.snoozeStr != "" {
		if _, err := cli.ParseDue(msg.snoozeStr); err != nil {
			m.err = fmt.Errorf("snooze: %w", err)
			return m, nil
		}
	}

	m.err = nil

	if msg.taskID != nil {
		return m, updateTaskCmd(m.client, msg)
	}

	// New task: after create, expand the parent so the new subtask is visible.
	if msg.parentID != nil {
		m.expanded[*msg.parentID] = true
	}
	if msg.planDay != "" {
		// A plan day was chosen on the create form: refresh the Plan tab's
		// scheduled-days markers alongside the task tree so the new day
		// shows up without waiting for some other action to refresh it.
		return m, tea.Batch(createTaskCmd(m.client, m.planClient, msg), listScheduledDaysCmd(m.planClient))
	}
	return m, createTaskCmd(m.client, m.planClient, msg)
}

func (m Model) handleMoveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.move == nil {
		m.mode = modeList
		return m, nil
	}
	cmd, keepOpen := m.move.Update(msg, &m)
	if !keepOpen {
		m.mode = modeList
		m.move = nil
	}
	return m, cmd
}

func (m Model) handleMoveTaskResult(msg moveTaskResultMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		if m.move != nil {
			m.move.errMsg = msg.err.Error()
		}
		return m, nil
	}
	m.mode = modeList
	m.move = nil
	m.err = nil
	m.tree = msg.tree
	m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
	if msg.taskID != 0 {
		m.ensureVisible(msg.taskID)
		m.visible = buildVisible(m.tree, m.expanded, m.showAll, m.pendingComplete, time.Now().Local())
		m.cursor = findCursor(m.visible, msg.taskID)
	} else {
		m.cursor = clampCursor(m.cursor, len(m.visible))
	}
	// Re-clamp scroll offset after the list is rebuilt.
	m = m.reconcileScroll()
	// Re-fire filter after move mutation.
	if m.filterExpr != "" {
		m2, cmd := m.applyFilter(m.filterExpr, filterKeepCursor)
		return m2, cmd
	}
	return m, nil
}

func (m Model) handleReorderResult(msg reorderResultMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.err = msg.err
		return m, nil
	}
	// Patch the in-memory model: update positions for all tasks in the returned
	// sibling group, then rebuild the tree without a server round-trip.
	posMap := make(map[int64]int64, len(msg.siblings))
	for _, s := range msg.siblings {
		posMap[s.Id] = s.Position
	}
	for _, root := range m.tree {
		patchPositions(root, posMap)
	}
	m.tree = cli.BuildTree(flattenTree(m.tree))
	m.err = nil
	m.visible = m.rebuildVisible()
	m.cursor = findCursor(m.visible, msg.taskID)
	// Re-clamp scroll offset after the list is rebuilt.
	m = m.reconcileScroll()
	return m, nil
}

// patchPositions recursively updates Task.Position fields from posMap.
func patchPositions(node *cli.TreeNode, posMap map[int64]int64) {
	if pos, ok := posMap[node.Task.Id]; ok {
		node.Task.Position = pos
	}
	for _, child := range node.Children {
		patchPositions(child, posMap)
	}
}

// flattenTree returns a flat slice of all tasks in the tree.
func flattenTree(nodes []*cli.TreeNode) []*taskv1.Task {
	var tasks []*taskv1.Task
	for _, n := range nodes {
		tasks = append(tasks, n.Task)
		tasks = append(tasks, flattenTree(n.Children)...)
	}
	return tasks
}

// ensureVisible walks the tree to find the task with the given id and expands
// all its ancestors so it appears in the visible list.
func (m *Model) ensureVisible(id int64) {
	var walk func(nodes []*cli.TreeNode) bool
	walk = func(nodes []*cli.TreeNode) bool {
		for _, n := range nodes {
			if n.Task.Id == id {
				return true
			}
			if walk(n.Children) {
				m.expanded[n.Task.Id] = true
				return true
			}
		}
		return false
	}
	walk(m.tree)
}

// ── helpers ────────────────────────────────────────────────────────────────

// findParentNode returns the tree node whose Children slice contains childID,
// or nil if childID is a root task.
func findParentNode(tree []*cli.TreeNode, childID int64) *cli.TreeNode {
	for _, node := range tree {
		for _, child := range node.Children {
			if child.Task.Id == childID {
				return node
			}
		}
		if found := findParentNode(node.Children, childID); found != nil {
			return found
		}
	}
	return nil
}

func findCursor(visible []*visibleRow, taskID int64) int {
	for i, r := range visible {
		if r.node.Task.Id == taskID {
			return i
		}
	}
	return 0
}

func clampCursor(cur, length int) int {
	if length == 0 {
		return 0
	}
	if cur >= length {
		return length - 1
	}
	if cur < 0 {
		return 0
	}
	return cur
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// nowOrDefault returns the model's injected time source (for tests) or time.Now().Local().
func (m Model) nowOrDefault() time.Time {
	if m.nowFunc != nil {
		return m.nowFunc()
	}
	return time.Now().Local()
}

// prevVisibleUntimed returns the previous untimed entry before cursor in the
// displayed (pre-filtered) entry list. Returns nil if none found.
func prevVisibleUntimed(entries []*planv1.PlanEntry, cursor int) *planv1.PlanEntry {
	for i := cursor - 1; i >= 0; i-- {
		if entries[i].StartMinute == nil {
			return entries[i]
		}
	}
	return nil
}

// nextVisibleUntimed returns the next untimed entry after cursor in the
// displayed (pre-filtered) entry list. Returns nil if none found.
func nextVisibleUntimed(entries []*planv1.PlanEntry, cursor int) *planv1.PlanEntry {
	for i := cursor + 1; i < len(entries); i++ {
		if entries[i].StartMinute == nil {
			return entries[i]
		}
	}
	return nil
}
