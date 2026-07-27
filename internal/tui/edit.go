package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	"github.com/pboyd/twig/internal/goal"
)

// goalStates defines the cycling order for the goal state selector in the edit form.
var goalStates = goal.StateDisplayOrder()

// goalStateIndex returns the index of the given state in goalStates.
func goalStateIndex(s goalv1.GoalState) int {
	for i, gs := range goalStates {
		if gs == s {
			return i
		}
	}
	return 0
}

// pgMsgGoalStateToggle is sent when the user cycles the state field in the goal edit form.
type pgMsgGoalStateToggle struct {
	delta int
}

// Plan selector choices for the create-task form.
const (
	planChoiceNone     = 0
	planChoiceToday    = 1
	planChoiceTomorrow = 2
	planChoiceDate     = 3
)

var planChoiceLabels = []string{"No plan", "Today", "Tomorrow", ""}

const (
	focusName        = 0
	focusDescription = 1
	focusDue         = 2
	focusEstimate    = 3
	focusSnooze      = 4
	focusState       = 5
	focusPlan        = 6
	focusGoal        = 7
	focusSave        = 8
	focusCancel      = 9
	focusCount       = 10
)

// editFormModel holds the state of the task edit/create form.
type editFormModel struct {
	taskID           *int64
	parentID         *int64
	name             textinput.Model
	description      textarea.Model
	due              textinput.Model
	pomodoroEstimate textinput.Model
	snooze           textinput.Model
	focusIndex       int
	originalCursor   int
	calendar         *calendarModel
	nowFunc          func() time.Time
	isGoal           bool // true when this form edits/creates a Goal, not a Task
	// Goal selector (task edit forms only; hidden when showGoalField is false).
	availableGoals []*goalv1.Goal // in progress + incubating goals
	goalIdx        int            // -1 = none, 0..N-1 = index into availableGoals
	showGoalField  bool
	originalGoalID *int64
	// Goal state selector (goal edit forms only; hidden when !isGoal || taskID == nil).
	goalStateIdx     int // index into goalStates cycle
	origGoalStateIdx int
	// Plan selector (task create forms only; hidden when showPlanField is false).
	planIdx       int    // index into plan choices; 0 = none
	planDate      string // ISO day, set only when a calendar pick made planIdx == planChoiceDate
	showPlanField bool   // true only when creating a task (not editing, not a goal)
	origPlanIdx   int
	// Opened-state snapshot for dirty detection.
	origName        string
	origDescription string
	origDue         string
	origEstimate    string
	origSnooze      string
	origGoalIdx     int
}

// editSavedMsg is dispatched when the user confirms the edit form.
type editSavedMsg struct {
	taskID         *int64
	parentID       *int64
	name           string
	description    string
	dueStr         string
	estimateStr    string
	snoozeStr      string
	originalCursor int
	// Goal association: populated only when showGoalField was true.
	newGoalID   *int64 // nil = clear association; non-nil = associate with this goal
	goalChanged bool   // true if goal differs from the task's original goal_id
	// Goal state: populated only when editing an existing goal (isGoal && taskID != nil).
	goalState        goalv1.GoalState // the cycling selector value
	goalStateChanged bool             // true if state differs from original
	// Plan: resolved ISO day (YYYY-MM-DD) for the new task's plan entry. Empty = no plan.
	planDay string
}

// editCancelledMsg is dispatched when the user cancels the edit form.
type editCancelledMsg struct {
	originalCursor int
}

// editDiscardRequestedMsg is dispatched when the user presses Esc on a dirty
// edit form, requesting a discard-confirmation overlay.
type editDiscardRequestedMsg struct {
	originalCursor int
}

// NewEditForm creates a form pre-filled with an existing task's values.
// goals, when non-nil, enables the Goal selector field showing in-progress/incubating goals.
// Pass nil (e.g. when editing a goal via its fake-task) to hide the Goal field.
// goalState, when non-empty, sets the goal state cycling field (for goal edit forms).
func NewEditForm(task *taskv1.Task, originalCursor int, goals []*goalv1.Goal, goalState goalv1.GoalState) editFormModel {
	f := newBlankForm(originalCursor)
	f.taskID = &task.Id
	if task.ParentId != nil {
		pid := task.GetParentId()
		f.parentID = &pid
	}
	f.name.SetValue(task.Name)
	f.description.SetValue(task.GetDescription())
	if task.Due != nil {
		f.due.SetValue(task.Due.AsTime().UTC().Format("2006-01-02T15:04:05Z"))
	}
	if task.GetEstimate() > 0 {
		f.pomodoroEstimate.SetValue(fmt.Sprintf("%d", task.GetEstimate()))
	}
	if task.SnoozeUntil != nil {
		f.snooze.SetValue(task.SnoozeUntil.AsTime().UTC().Format("2006-01-02"))
	}

	if goals != nil {
		f.showGoalField = true
		f.originalGoalID = task.GoalId
		// Collect in progress + incubating goals for the selector. Also include
		// the task's current goal if it's completed/archived so that saving
		// without changing the goal field doesn't clear the association.
		var currentGoalIncluded bool
		for _, g := range goals {
			s := g.GetState()
			if s == goalv1.GoalState_GOAL_STATE_IN_PROGRESS || s == goalv1.GoalState_GOAL_STATE_INCUBATING {
				f.availableGoals = append(f.availableGoals, g)
				if task.GoalId != nil && g.GetId() == task.GetGoalId() {
					currentGoalIncluded = true
				}
			}
		}
		// If the task's goal is completed/archived and not in the list above,
		// find it and append it so goalIdx can point to it.
		if task.GoalId != nil && !currentGoalIncluded {
			for _, g := range goals {
				if g.GetId() == task.GetGoalId() {
					f.availableGoals = append(f.availableGoals, g)
					break
				}
			}
		}
		f.goalIdx = -1
		if task.GoalId != nil {
			for i, g := range f.availableGoals {
				if g.GetId() == task.GetGoalId() {
					f.goalIdx = i
					break
				}
			}
		}
	}

	// Initialize goal state cycling field when goalState is provided (goal edit forms).
	if goalState != goalv1.GoalState_GOAL_STATE_UNSPECIFIED {
		f.goalStateIdx = goalStateIndex(goalState)
	}

	// Plan field is only for create forms; edit forms never show it.
	f.showPlanField = false

	// Capture opened-state snapshot for dirty detection.
	f.origName = f.name.Value()
	f.origDescription = f.description.Value()
	f.origDue = f.due.Value()
	f.origEstimate = f.pomodoroEstimate.Value()
	f.origSnooze = f.snooze.Value()
	f.origGoalIdx = f.goalIdx
	f.origGoalStateIdx = f.goalStateIdx
	f.origPlanIdx = f.planIdx

	return f
}

// NewSubtaskForm creates a blank form for a new subtask under parentID.
func NewSubtaskForm(parentID int64, originalCursor int) editFormModel {
	f := newBlankForm(originalCursor)
	f.parentID = &parentID
	return f
}

// NewRootForm creates a blank form for a new root task.
// goals, when non-nil and non-empty, enables the Goal selector showing
// in-progress/incubating goals. Pass nil (or an empty slice) to hide it.
func NewRootForm(originalCursor int, goals []*goalv1.Goal) editFormModel {
	f := newBlankForm(originalCursor)
	for _, g := range goals {
		s := g.GetState()
		if s == goalv1.GoalState_GOAL_STATE_IN_PROGRESS || s == goalv1.GoalState_GOAL_STATE_INCUBATING {
			f.availableGoals = append(f.availableGoals, g)
		}
	}
	if len(f.availableGoals) > 0 {
		f.showGoalField = true
		f.goalIdx = -1 // default: no goal selected
	}
	return f
}

func newBlankForm(originalCursor int) editFormModel {
	name := textinput.New()
	name.Placeholder = "Task name"
	name.Focus()

	desc := textarea.New()
	desc.Placeholder = "Description (optional)"
	desc.SetHeight(4)
	desc.ShowLineNumbers = false

	due := textinput.New()
	due.Placeholder = "YYYY-MM-DD or RFC3339 (optional)"

	est := textinput.New()
	est.Placeholder = "Pomodoro estimate (optional)"
	est.CharLimit = 3

	snooze := textinput.New()
	snooze.Placeholder = "YYYY-MM-DD (optional)"

	return editFormModel{
		name:             name,
		description:      desc,
		due:              due,
		pomodoroEstimate: est,
		snooze:           snooze,
		focusIndex:       focusName,
		originalCursor:   originalCursor,
		showPlanField:    true,
		origGoalIdx:      -1,
	}
}

// now returns the current time, using nowFunc if set.
func (f editFormModel) now() time.Time {
	if f.nowFunc != nil {
		return f.nowFunc()
	}
	return time.Now()
}

// isDirty returns true when any form field differs from its
// opened-state snapshot (after trimming whitespace). Only meaningful
// for edit forms (taskID != nil); create forms skip the check.
// Goal selector is only compared when showGoalField is true.
func (f editFormModel) isDirty() bool {
	if strings.TrimSpace(f.name.Value()) != f.origName {
		return true
	}
	if strings.TrimSpace(f.description.Value()) != f.origDescription {
		return true
	}
	if strings.TrimSpace(f.due.Value()) != f.origDue {
		return true
	}
	if strings.TrimSpace(f.pomodoroEstimate.Value()) != f.origEstimate {
		return true
	}
	if strings.TrimSpace(f.snooze.Value()) != f.origSnooze {
		return true
	}
	if f.showGoalField && f.goalIdx != f.origGoalIdx {
		return true
	}
	if f.isGoal && f.taskID != nil && f.goalStateIdx != f.origGoalStateIdx {
		return true
	}
	if f.showPlanField && f.planIdx != f.origPlanIdx {
		return true
	}
	return false
}

// openCalendar creates and attaches a calendarModel for the focused date field.
func (f editFormModel) openCalendar() editFormModel {
	var fieldValue string
	rfc3339Field := false
	switch f.focusIndex {
	case focusDue:
		fieldValue = f.due.Value()
		rfc3339Field = true
	case focusSnooze:
		fieldValue = f.snooze.Value()
	case focusPlan:
		fieldValue = f.planDate
		if fieldValue == "" {
			fieldValue = f.now().Format("2006-01-02")
		}
	}
	f.calendar = newCalendar(fieldValue, f.now(), rfc3339Field)
	return f
}

// Update handles messages for the edit form.
func (f editFormModel) Update(msg tea.Msg, keys KeyMap) (editFormModel, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		// While the calendar is open it captures all input: drop pasted text
		// so the field underneath can't change out from under it.
		if f.calendar != nil {
			if _, isPaste := msg.(tea.PasteMsg); isPaste {
				return f, nil
			}
		}
		return f.updateFocusedField(msg)
	}

	// When the calendar is open, route all key messages to it first.
	if f.calendar != nil {
		switch {
		case keyMsg.Code == tea.KeyTab:
			// tab/shift+tab: close calendar without modifying the field, then cycle focus.
			f.calendar = nil
			if keyMsg.Mod == tea.ModShift {
				f = f.cycleFocus(-1)
			} else {
				f = f.cycleFocus(1)
			}
			return f, nil

		case keyMsg.Code == tea.KeyEscape:
			// esc: close calendar, field unchanged.
			f.calendar = nil
			return f, nil

		case keyMsg.Code == tea.KeyEnter:
			// enter: confirm — write formatted date to the field, close calendar.
			dateStr := f.calendar.confirm()
			switch f.focusIndex {
			case focusDue:
				f.due.SetValue(dateStr)
			case focusSnooze:
				f.snooze.SetValue(dateStr)
			case focusPlan:
				f.planDate = dateStr
				f.planIdx = planChoiceDate
			}
			f.calendar = nil
			return f, nil

		default:
			// Delegate navigation keys to the calendar model.
			f.calendar.handleKey(keyMsg)
			return f, nil
		}
	}

	// Global shortcuts take priority over field forwarding.
	switch {
	case key.Matches(keyMsg, keys.Editor) && f.focusIndex == focusDescription:
		return f, openEditorCmd(f.description.Value())

	case key.Matches(keyMsg, keys.Calendar) && (f.focusIndex == focusDue || f.focusIndex == focusSnooze || f.focusIndex == focusPlan):
		f = f.openCalendar()
		return f, nil

	case key.Matches(keyMsg, keys.Cancel):
		if f.isDirty() {
			return f, func() tea.Msg {
				return editDiscardRequestedMsg{originalCursor: f.originalCursor}
			}
		}
		return f, func() tea.Msg {
			return editCancelledMsg{originalCursor: f.originalCursor}
		}

	case key.Matches(keyMsg, keys.Save):
		return f, f.buildSaveMsg()

	case key.Matches(keyMsg, keys.Tab):
		f = f.cycleFocus(1)
		return f, nil

	case key.Matches(keyMsg, keys.ShiftTab):
		f = f.cycleFocus(-1)
		return f, nil
	}

	// Handle goal field key events (left/right cycle the selector).
	if f.focusIndex == focusGoal && f.showGoalField {
		switch keyMsg.Code {
		case tea.KeyLeft:
			f = f.cycleGoal(-1)
			return f, nil
		case tea.KeyRight:
			f = f.cycleGoal(1)
			return f, nil
		}
	}

	// Handle goal state field key events (left/right cycle state).
	if f.focusIndex == focusState && f.isGoal && f.taskID != nil {
		switch keyMsg.Code {
		case tea.KeyLeft:
			f.goalStateIdx = (f.goalStateIdx - 1 + len(goalStates)) % len(goalStates)
			return f, nil
		case tea.KeyRight:
			f.goalStateIdx = (f.goalStateIdx + 1) % len(goalStates)
			return f, nil
		}
	}

	// Handle plan field key events (left/right cycle plan choice).
	if f.focusIndex == focusPlan && f.showPlanField {
		switch keyMsg.Code {
		case tea.KeyLeft:
			f = f.cyclePlan(-1)
			return f, nil
		case tea.KeyRight:
			f = f.cyclePlan(1)
			return f, nil
		}
	}

	// Enter on Save/Cancel buttons.
	if keyMsg.Code == tea.KeyEnter {
		switch f.focusIndex {
		case focusSave:
			return f, f.buildSaveMsg()
		case focusCancel:
			return f, func() tea.Msg { return editCancelledMsg{originalCursor: f.originalCursor} }
		// Enter on single-line fields advances focus.
		case focusName, focusDue, focusEstimate, focusSnooze, focusState, focusGoal, focusPlan:
			f = f.cycleFocus(1)
			return f, nil
		}
	}

	return f.updateFocusedField(msg)
}

// buildSaveMsg builds the editSavedMsg command, including goal association state.
func (f editFormModel) buildSaveMsg() func() tea.Msg {
	return func() tea.Msg {
		msg := editSavedMsg{
			taskID:         f.taskID,
			parentID:       f.parentID,
			name:           f.name.Value(),
			description:    f.description.Value(),
			dueStr:         f.due.Value(),
			estimateStr:    f.pomodoroEstimate.Value(),
			snoozeStr:      f.snooze.Value(),
			originalCursor: f.originalCursor,
		}
		if f.showGoalField {
			var newGoalID *int64
			if f.goalIdx >= 0 && f.goalIdx < len(f.availableGoals) {
				gid := f.availableGoals[f.goalIdx].GetId()
				newGoalID = &gid
			}
			msg.newGoalID = newGoalID
			orig := f.originalGoalID
			msg.goalChanged = (orig == nil) != (newGoalID == nil) ||
				(orig != nil && newGoalID != nil && *orig != *newGoalID)
		}
		if f.isGoal && f.taskID != nil {
			msg.goalState = goalStates[f.goalStateIdx]
			msg.goalStateChanged = f.goalStateIdx != f.origGoalStateIdx
		}
		if f.showPlanField && f.planIdx != planChoiceNone {
			switch f.planIdx {
			case planChoiceToday:
				msg.planDay = f.now().Format("2006-01-02")
			case planChoiceTomorrow:
				msg.planDay = f.now().AddDate(0, 0, 1).Format("2006-01-02")
			case planChoiceDate:
				msg.planDay = f.planDate
			}
		}
		return msg
	}
}

// cycleGoal moves the goal selector by delta (-1 or +1), wrapping around.
// -1 is "none"; 0..N-1 index the available goals.
func (f editFormModel) cycleGoal(delta int) editFormModel {
	n := len(f.availableGoals)
	if n == 0 {
		return f
	}
	// Treat -1 (none) as position n in the cycle so we can do modular arithmetic.
	pos := f.goalIdx + 1 // 0 = none, 1..n = goal[0..n-1]
	pos = (pos + delta + n + 1) % (n + 1)
	f.goalIdx = pos - 1
	return f
}

// cyclePlan moves the plan selector by delta (-1 or +1), cycling through
// None → Today → Tomorrow → None. planChoiceDate is a destination, not a
// cycle stop; cycling from Date re-enters the loop and clears planDate.
func (f editFormModel) cyclePlan(delta int) editFormModel {
	// 3-stop cycle: None, Today, Tomorrow
	f.planIdx = (f.planIdx + delta + 3) % 3
	f.planDate = "" // cycling away from Date clears it
	return f
}

// cycleFocus moves the focus index by delta, wrapping around, and updates field
// focus state. focusGoal is skipped when showGoalField is false.
func (f editFormModel) cycleFocus(delta int) editFormModel {
	idx := f.focusIndex
	for {
		idx = (idx + delta + focusCount) % focusCount
		if idx == focusGoal && !f.showGoalField {
			continue
		}
		if idx == focusState && (!f.isGoal || f.taskID == nil) {
			continue
		}
		if idx == focusPlan && !f.showPlanField {
			continue
		}
		break
	}
	f.focusIndex = idx
	f.name.Blur()
	f.description.Blur()
	f.due.Blur()
	f.pomodoroEstimate.Blur()
	f.snooze.Blur()
	switch f.focusIndex {
	case focusName:
		f.name.Focus()
	case focusDescription:
		f.description.Focus()
	case focusDue:
		f.due.Focus()
	case focusEstimate:
		f.pomodoroEstimate.Focus()
	case focusSnooze:
		f.snooze.Focus()
	}
	return f
}

// updateFocusedField forwards a message to the currently focused input field.
func (f editFormModel) updateFocusedField(msg tea.Msg) (editFormModel, tea.Cmd) {
	var cmd tea.Cmd
	switch f.focusIndex {
	case focusName:
		f.name, cmd = f.name.Update(msg)
	case focusDescription:
		f.description, cmd = f.description.Update(msg)
	case focusDue:
		f.due, cmd = f.due.Update(msg)
	case focusEstimate:
		f.pomodoroEstimate, cmd = f.pomodoroEstimate.Update(msg)
	case focusSnooze:
		f.snooze, cmd = f.snooze.Update(msg)
	}
	return f, cmd
}

// View renders the edit form.
func (f editFormModel) View(width int) string {
	// fieldWidth is the usable width for text inputs: total width minus the
	// label prefix ("  Label: " is ~12 chars at most, use 14 for safety).
	fieldWidth := width - 14
	if fieldWidth < 20 {
		fieldWidth = 20
	}
	f.name.SetWidth(fieldWidth)
	f.due.SetWidth(fieldWidth)
	f.pomodoroEstimate.SetWidth(fieldWidth)
	f.snooze.SetWidth(fieldWidth)

	var sb strings.Builder

	title := "New Task"
	switch {
	case f.isGoal && f.taskID != nil:
		title = fmt.Sprintf("Edit Goal #%d", *f.taskID)
	case f.isGoal:
		title = "New Goal"
	case f.taskID != nil:
		title = fmt.Sprintf("Edit Task #%d", *f.taskID)
	case f.parentID != nil:
		title = fmt.Sprintf("New Subtask (parent #%d)", *f.parentID)
	}
	sb.WriteString(title + "\n\n")

	sb.WriteString(fieldLabel("Name", f.focusIndex == focusName))
	sb.WriteString(f.name.View() + "\n\n")

	sb.WriteString(fieldLabel("Description", f.focusIndex == focusDescription) + "\n")
	sb.WriteString(f.description.View() + "\n")
	if f.focusIndex == focusDescription {
		sb.WriteString("  ctrl+g: open editor\n")
	}
	sb.WriteString("\n")

	f.writeDateField(&sb, "Due", f.due, focusDue)

	sb.WriteString(fieldLabel("Estimate", f.focusIndex == focusEstimate))
	sb.WriteString(f.pomodoroEstimate.View() + "\n\n")

	f.writeDateField(&sb, "Snooze until", f.snooze, focusSnooze)

	if f.showPlanField {
		planLabel := planChoiceLabels[f.planIdx]
		if f.planIdx == planChoiceDate {
			planLabel = f.planDate
		}
		if planLabel == "" {
			planLabel = "No plan"
		}
		sb.WriteString(fieldLabel("Plan", f.focusIndex == focusPlan))
		sb.WriteString("‹ " + planLabel + " ›\n")
		if f.focusIndex == focusPlan {
			if f.calendar != nil {
				sb.WriteString(f.calendar.View())
				sb.WriteString("  enter: pick  esc: never mind  t: today  [/]: month  {/}: year\n")
			} else {
				sb.WriteString("  ←/→ cycle · ctrl+g: summon the calendar\n")
			}
		}
		sb.WriteString("\n")
	}

	if f.isGoal && f.taskID != nil {
		stateName := goal.DisplayLabel(goalStates[f.goalStateIdx])
		sb.WriteString(fieldLabel("State", f.focusIndex == focusState))
		sb.WriteString(stateName + "\n")
		if f.focusIndex == focusState {
			sb.WriteString("  ←/→: change  tab: next field\n")
		}
		sb.WriteString("\n")
	}

	if f.showGoalField {
		goalName := "none"
		if f.goalIdx >= 0 && f.goalIdx < len(f.availableGoals) {
			goalName = f.availableGoals[f.goalIdx].GetName()
		}
		sb.WriteString(fieldLabel("Goal", f.focusIndex == focusGoal))
		sb.WriteString(goalName + "\n")
		if f.focusIndex == focusGoal {
			sb.WriteString("  ←/→: change  tab: next field\n")
		}
		sb.WriteString("\n")
	}

	saveStyle := "[ Save ]"
	cancelStyle := "[ Cancel ]"
	if f.focusIndex == focusSave {
		saveStyle = "[>Save<]"
	}
	if f.focusIndex == focusCancel {
		cancelStyle = "[>Cancel<]"
	}
	sb.WriteString(saveStyle + "  " + cancelStyle + "\n")
	sb.WriteString("\nCtrl+S: save  Tab: next field  Esc: cancel")

	return sb.String()
}

// writeDateField renders a date field's label and input, followed by the open
// calendar with its key hints, or a hint on how to summon it when focused.
func (f editFormModel) writeDateField(sb *strings.Builder, label string, input textinput.Model, focus int) {
	sb.WriteString(fieldLabel(label, f.focusIndex == focus))
	sb.WriteString(input.View() + "\n")
	if f.calendar != nil && f.focusIndex == focus {
		sb.WriteString(f.calendar.View())
		sb.WriteString("  enter: pick  esc: never mind  t: today  [/]: month  {/}: year\n")
	} else if f.focusIndex == focus {
		sb.WriteString("  ctrl+g: summon the calendar\n")
	}
	sb.WriteString("\n")
}

func fieldLabel(label string, focused bool) string {
	if focused {
		return fmt.Sprintf("► %s: ", label)
	}
	return fmt.Sprintf("  %s: ", label)
}
