package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
)

const (
	focusName        = 0
	focusDescription = 1
	focusDue         = 2
	focusEstimate    = 3
	focusSnooze      = 4
	focusSave        = 5
	focusCancel      = 6
	focusCount       = 7
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
}

// editCancelledMsg is dispatched when the user cancels the edit form.
type editCancelledMsg struct {
	originalCursor int
}

// NewEditForm creates a form pre-filled with an existing task's values.
func NewEditForm(task *taskv1.Task, originalCursor int) editFormModel {
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

// Update handles messages for the edit form.
func (f editFormModel) Update(msg tea.Msg, keys KeyMap) (editFormModel, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		return f.updateFocusedField(msg)
	}

	// Global shortcuts take priority over field forwarding.
	switch {
	case key.Matches(keyMsg, keys.Editor) && f.focusIndex == focusDescription:
		return f, openEditorCmd(f.description.Value())

	case key.Matches(keyMsg, keys.Save):
		return f, func() tea.Msg {
			return editSavedMsg{
				taskID:         f.taskID,
				parentID:       f.parentID,
				name:           f.name.Value(),
				description:    f.description.Value(),
				dueStr:         f.due.Value(),
				estimateStr:    f.pomodoroEstimate.Value(),
				snoozeStr:      f.snooze.Value(),
				originalCursor: f.originalCursor,
			}
		}

	case key.Matches(keyMsg, keys.Cancel):
		return f, func() tea.Msg { return editCancelledMsg{originalCursor: f.originalCursor} }

	case key.Matches(keyMsg, keys.Tab):
		f = f.cycleFocus(1)
		return f, nil

	case key.Matches(keyMsg, keys.ShiftTab):
		f = f.cycleFocus(-1)
		return f, nil
	}

	// Enter on Save/Cancel buttons.
	if keyMsg.Code == tea.KeyEnter {
		switch f.focusIndex {
		case focusSave:
			return f, func() tea.Msg {
				return editSavedMsg{
					taskID:         f.taskID,
					parentID:       f.parentID,
					name:           f.name.Value(),
					description:    f.description.Value(),
					dueStr:         f.due.Value(),
					estimateStr:    f.pomodoroEstimate.Value(),
					snoozeStr:      f.snooze.Value(),
					originalCursor: f.originalCursor,
				}
			}
		case focusCancel:
			return f, func() tea.Msg { return editCancelledMsg{originalCursor: f.originalCursor} }
		// Enter on single-line fields advances focus.
		case focusName, focusDue, focusEstimate, focusSnooze:
			f = f.cycleFocus(1)
			return f, nil
		}
	}

	return f.updateFocusedField(msg)
}

// cycleFocus moves the focus index by delta, wrapping around, and updates field
// focus state.
func (f editFormModel) cycleFocus(delta int) editFormModel {
	f.focusIndex = (f.focusIndex + delta + focusCount) % focusCount
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

	sb.WriteString(fieldLabel("Due", f.focusIndex == focusDue))
	sb.WriteString(f.due.View() + "\n\n")

	sb.WriteString(fieldLabel("Estimate", f.focusIndex == focusEstimate))
	sb.WriteString(f.pomodoroEstimate.View() + "\n\n")

	sb.WriteString(fieldLabel("Snooze until", f.focusIndex == focusSnooze))
	sb.WriteString(f.snooze.View() + "\n\n")

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

func fieldLabel(label string, focused bool) string {
	if focused {
		return fmt.Sprintf("► %s: ", label)
	}
	return fmt.Sprintf("  %s: ", label)
}
