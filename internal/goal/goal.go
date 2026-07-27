package goal

import (
	"fmt"

	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
)

// StateDisplayOrder returns states in display order: In Progress, Incubating, Hold, Completed, Archived.
func StateDisplayOrder() []goalv1.GoalState {
	return []goalv1.GoalState{
		goalv1.GoalState_GOAL_STATE_IN_PROGRESS,
		goalv1.GoalState_GOAL_STATE_INCUBATING,
		goalv1.GoalState_GOAL_STATE_HOLD,
		goalv1.GoalState_GOAL_STATE_COMPLETED,
		goalv1.GoalState_GOAL_STATE_ARCHIVED,
	}
}

// DefaultVisible returns true if the state is shown without a "show all" toggle.
// In progress, incubating, and hold are visible by default; completed and archived are hidden.
func DefaultVisible(s goalv1.GoalState) bool {
	switch s {
	case goalv1.GoalState_GOAL_STATE_IN_PROGRESS, goalv1.GoalState_GOAL_STATE_INCUBATING, goalv1.GoalState_GOAL_STATE_HOLD:
		return true
	default:
		return false
	}
}

// StateName returns the lowercase CLI/display name for a state.
func StateName(s goalv1.GoalState) string {
	switch s {
	case goalv1.GoalState_GOAL_STATE_INCUBATING:
		return "incubating"
	case goalv1.GoalState_GOAL_STATE_IN_PROGRESS:
		return "in progress"
	case goalv1.GoalState_GOAL_STATE_HOLD:
		return "hold"
	case goalv1.GoalState_GOAL_STATE_COMPLETED:
		return "completed"
	case goalv1.GoalState_GOAL_STATE_ARCHIVED:
		return "archived"
	default:
		return "unspecified"
	}
}

// DisplayLabel returns the human-facing label for a state, e.g. for group headers.
func DisplayLabel(s goalv1.GoalState) string {
	switch s {
	case goalv1.GoalState_GOAL_STATE_INCUBATING:
		return "Incubating"
	case goalv1.GoalState_GOAL_STATE_IN_PROGRESS:
		return "In Progress"
	case goalv1.GoalState_GOAL_STATE_HOLD:
		return "Hold"
	case goalv1.GoalState_GOAL_STATE_COMPLETED:
		return "Completed"
	case goalv1.GoalState_GOAL_STATE_ARCHIVED:
		return "Archived"
	default:
		return "Unspecified"
	}
}

// CLIName returns the parseable CLI spelling for a state, i.e. a string that
// ParseState accepts. Unlike StateName, this uses "in-progress" (hyphen) so
// the result round-trips through ParseState.
func CLIName(s goalv1.GoalState) string {
	switch s {
	case goalv1.GoalState_GOAL_STATE_IN_PROGRESS:
		return "in-progress"
	default:
		return StateName(s)
	}
}

// StateNames returns the parseable CLI names for all states in display order,
// excluding UNSPECIFIED.
func StateNames() []string {
	order := StateDisplayOrder()
	names := make([]string, len(order))
	for i, s := range order {
		names[i] = CLIName(s)
	}
	return names
}

// ParseState parses a CLI string to a GoalState. Returns an error for unknown values.
func ParseState(s string) (goalv1.GoalState, error) {
	switch s {
	case "incubating":
		return goalv1.GoalState_GOAL_STATE_INCUBATING, nil
	case "in-progress", "in progress":
		return goalv1.GoalState_GOAL_STATE_IN_PROGRESS, nil
	case "hold":
		return goalv1.GoalState_GOAL_STATE_HOLD, nil
	case "completed":
		return goalv1.GoalState_GOAL_STATE_COMPLETED, nil
	case "archived":
		return goalv1.GoalState_GOAL_STATE_ARCHIVED, nil
	default:
		return goalv1.GoalState_GOAL_STATE_UNSPECIFIED, fmt.Errorf("unknown goal state %q", s)
	}
}

// EffectiveGoalID returns the goal_id of the nearest self-or-ancestor for taskID
// in a flat task list. Returns 0 and false if no goal association exists in the ancestry.
func EffectiveGoalID(tasks []*taskv1.Task, taskID int64) (int64, bool) {
	byID := make(map[int64]*taskv1.Task, len(tasks))
	for _, t := range tasks {
		byID[t.Id] = t
	}

	cur, ok := byID[taskID]
	if !ok {
		return 0, false
	}

	// Walk self then ancestors, stopping at the first goal_id we find.
	seen := make(map[int64]bool) // guard against cycles
	for cur != nil {
		if seen[cur.Id] {
			break
		}
		seen[cur.Id] = true

		if cur.GoalId != nil {
			return cur.GetGoalId(), true
		}

		if cur.ParentId == nil {
			break
		}
		cur = byID[cur.GetParentId()]
	}

	return 0, false
}

// AssociationRoots returns tasks directly associated with goalID (goal_id == goalID).
func AssociationRoots(tasks []*taskv1.Task, goalID int64) []*taskv1.Task {
	var roots []*taskv1.Task
	for _, t := range tasks {
		if t.GoalId != nil && t.GetGoalId() == goalID {
			roots = append(roots, t)
		}
	}
	return roots
}

// SubtreeForGoal returns all tasks that belong to goalID — the association roots
// and all their descendants.
func SubtreeForGoal(tasks []*taskv1.Task, goalID int64) []*taskv1.Task {
	// Build parent → children index.
	children := make(map[int64][]*taskv1.Task, len(tasks))
	for _, t := range tasks {
		if t.ParentId != nil {
			pid := t.GetParentId()
			children[pid] = append(children[pid], t)
		}
	}

	roots := AssociationRoots(tasks, goalID)

	// BFS from each root to collect the full subtree.
	var result []*taskv1.Task
	queue := make([]*taskv1.Task, 0, len(roots))
	queue = append(queue, roots...)

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		result = append(result, cur)
		queue = append(queue, children[cur.Id]...)
	}

	return result
}
