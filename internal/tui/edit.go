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
)

const (
	focusName        = 0
	focusDescription = 1
	focusDue         = 2
	focusEstimate    = 3
	focusSnooze      = 4
	focusGoal        = 5
	focusSave        = 6
	focusCancel      = 7
	focusCount       = 8
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
	// Goal selector (task edit forms only; hidden when showGoalField is false).
	availableGoals []*goalv1.Goal // committed + incubating goals
	goalIdx        int            // -1 = none, 0..N-1 = index into availableGoals
	showGoalField  bool
	originalGoalID *int64
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
}

// editCancelledMsg is dispatched when the user cancels the edit form.
type editCancelledMsg struct {
	originalCursor int
}

// NewEditForm creates a form pre-filled with an existing task's values.
// goals, when non-nil, enables the Goal selector field showing committed/incubating goals.
// Pass nil (e.g. when editing a goal via its fake-task) to hide the Goal field.
func NewEditForm(task *taskv1.Task, originalCursor int, goals []*goalv1.Goal) editFormModel {
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
		// Collect only committed + incubating goals for the selector.
		for _, g := range goals {
			s := g.GetState()
			if s == goalv1.GoalState_GOAL_STATE_COMMITTED || s == goalv1.GoalState_GOAL_STATE_INCUBATING {
				f.availableGoals = append(f.availableGoals, g)
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

	return f
}

// NewSubtaskForm creates a blank form for a new subtask under parentID.
func NewSubtaskForm(parentID int64, originalCursor int) editFormModel {
	f := newBlankForm(originalCursor)
	f.parentID = &parentID
	return f
}

// NewRootForm creates a blank form for a new root task.
func NewRootForm(originalCursor int) editFormModel {
	return newBlankForm(originalCursor)
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
	}
}

// now returns the current time, using nowFunc if set.
func (f editFormModel) now() time.Time {
	if f.nowFunc != nil {
		return f.nowFunc()
	}
	return time.Now()
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

	case key.Matches(keyMsg, keys.Calendar) && (f.focusIndex == focusDue || f.focusIndex == focusSnooze):
		f = f.openCalendar()
		return f, nil

	case key.Matches(keyMsg, keys.Save):
		return f, f.buildSaveMsg()

	case key.Matches(keyMsg, keys.Cancel):
		return f, func() tea.Msg { return editCancelledMsg{originalCursor: f.originalCursor} }

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

	// Enter on Save/Cancel buttons.
	if keyMsg.Code == tea.KeyEnter {
		switch f.focusIndex {
		case focusSave:
			return f, f.buildSaveMsg()
		case focusCancel:
			return f, func() tea.Msg { return editCancelledMsg{originalCursor: f.originalCursor} }
		// Enter on single-line fields advances focus.
		case focusName, focusDue, focusEstimate, focusSnooze, focusGoal:
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

// cycleFocus moves the focus index by delta, wrapping around, and updates field
// focus state. focusGoal is skipped when showGoalField is false.
func (f editFormModel) cycleFocus(delta int) editFormModel {
	idx := f.focusIndex
	for {
		idx = (idx + delta + focusCount) % focusCount
		if idx == focusGoal && !f.showGoalField {
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
	if f.taskID != nil {
		title = fmt.Sprintf("Edit Task #%d", *f.taskID)
	} else if f.parentID != nil {
		title = fmt.Sprintf("New Subtask (parent #%d)", *f.parentID)
	}
	sb.WriteString(title + "\n\n")

	sb.WriteString(fieldLabel("Name", f.focusIndex == focusName))
	sb.WriteString(f.name.View() + "\n\n")

	sb.WriteString(fieldLabel("Description", f.focusIndex == focusDescription))
	sb.WriteString(f.description.View() + "\n")
	if f.focusIndex == focusDescription {
		sb.WriteString("  ctrl+g: open editor\n")
	}
	sb.WriteString("\n")

	f.writeDateField(&sb, "Due", f.due, focusDue)

	sb.WriteString(fieldLabel("Estimate", f.focusIndex == focusEstimate))
	sb.WriteString(f.pomodoroEstimate.View() + "\n\n")

	f.writeDateField(&sb, "Snooze until", f.snooze, focusSnooze)

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
	sb.WriteString("\nCtrl+S: save  Esc: cancel  Tab: next field")

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
