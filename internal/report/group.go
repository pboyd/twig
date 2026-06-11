package report

import (
	"sort"
	"time"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
)

// Entry is one completed task occurrence within a report period.
type Entry struct {
	TaskID      int64
	Name        string
	CompletedAt time.Time // in local time
	ParentName  string    // immediate parent name; empty for top-level tasks
	TopLevelID  int64     // root ancestor's id
	Depth       int       // distance from top-level ancestor (0 = top-level)
}

// DayGroup groups entries by a single local calendar day.
type DayGroup struct {
	Day     time.Time // midnight local of the grouping day
	Entries []Entry   // ordered by CompletedAt descending
}

// AccomplishmentGroup groups entries by top-level task.
type AccomplishmentGroup struct {
	TopLevelID   int64
	TopLevelName string
	// TopLevelCompletedAt is the top-level task's completion moment in local
	// time; zero unless Finished.
	TopLevelCompletedAt time.Time
	Finished            bool    // true when the top-level task itself completed within the period
	Entries             []Entry // completed-in-period descendants, ordered by Depth then CompletedAt
}

// Totals summarises a report period.
type Totals struct {
	TasksCompleted     int
	PomodorosCompleted int64
}

// BuildEntries filters a flat task list to entries completed within [p.StartUTC, p.EndUTC).
// It resolves ParentName, TopLevelID, and Depth by walking the full task list.
func BuildEntries(tasks []*taskv1.Task, p Period) []Entry {
	byID := make(map[int64]*taskv1.Task, len(tasks))
	for _, t := range tasks {
		byID[t.Id] = t
	}

	var entries []Entry
	for _, t := range tasks {
		if t.CompletedAt == nil {
			continue
		}
		completedAt := t.CompletedAt.AsTime()
		if completedAt.Before(p.StartUTC()) || !completedAt.Before(p.EndUTC()) {
			continue
		}

		parentName := ""
		if t.ParentId != nil {
			if parent, ok := byID[t.GetParentId()]; ok {
				parentName = parent.Name
			}
		}

		topLevelID, depth := resolveAncestry(t, byID)

		entries = append(entries, Entry{
			TaskID:      t.Id,
			Name:        t.Name,
			CompletedAt: completedAt.In(time.Local),
			ParentName:  parentName,
			TopLevelID:  topLevelID,
			Depth:       depth,
		})
	}
	return entries
}

// resolveAncestry walks parent_id to find the root ancestor and depth.
func resolveAncestry(t *taskv1.Task, byID map[int64]*taskv1.Task) (topLevelID int64, depth int) {
	cur := t
	for cur.ParentId != nil {
		parent, ok := byID[cur.GetParentId()]
		if !ok {
			break
		}
		depth++
		cur = parent
	}
	return cur.Id, depth
}

// GroupByDay groups entries into day buckets ordered by day descending (most recent first).
// Each bucket's entries are ordered by CompletedAt descending.
func GroupByDay(entries []Entry) []DayGroup {
	byDay := make(map[time.Time][]Entry)
	for _, e := range entries {
		day := truncateDay(e.CompletedAt)
		byDay[day] = append(byDay[day], e)
	}

	groups := make([]DayGroup, 0, len(byDay))
	for day, es := range byDay {
		sort.Slice(es, func(i, j int) bool {
			return es[i].CompletedAt.After(es[j].CompletedAt)
		})
		groups = append(groups, DayGroup{Day: day, Entries: es})
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Day.After(groups[j].Day)
	})
	return groups
}

// GroupByAccomplishment groups entries by top-level task.
// tasks is the full task list (needed to resolve top-level name and completion state).
// Returns (finished, ongoing) sections per data-model.md.
func GroupByAccomplishment(entries []Entry, tasks []*taskv1.Task, p Period) (finished, ongoing []AccomplishmentGroup) {
	byID := make(map[int64]*taskv1.Task, len(tasks))
	for _, t := range tasks {
		byID[t.Id] = t
	}

	byTopLevel := make(map[int64][]Entry)
	for _, e := range entries {
		byTopLevel[e.TopLevelID] = append(byTopLevel[e.TopLevelID], e)
	}

	topLevelIDs := make([]int64, 0, len(byTopLevel))
	for id := range byTopLevel {
		topLevelIDs = append(topLevelIDs, id)
	}
	sort.Slice(topLevelIDs, func(i, j int) bool { return topLevelIDs[i] < topLevelIDs[j] })

	for _, tlID := range topLevelIDs {
		es := byTopLevel[tlID]
		topTask, ok := byID[tlID]
		name := ""
		if ok {
			name = topTask.Name
		}

		isFinished := ok && topTask.CompletedAt != nil &&
			!topTask.CompletedAt.AsTime().Before(p.StartUTC()) &&
			topTask.CompletedAt.AsTime().Before(p.EndUTC())

		// Sort entries by Depth asc then CompletedAt asc within depth.
		sort.Slice(es, func(i, j int) bool {
			if es[i].Depth != es[j].Depth {
				return es[i].Depth < es[j].Depth
			}
			return es[i].CompletedAt.Before(es[j].CompletedAt)
		})

		g := AccomplishmentGroup{
			TopLevelID:   tlID,
			TopLevelName: name,
			Finished:     isFinished,
			Entries:      es,
		}
		if isFinished {
			g.TopLevelCompletedAt = topTask.CompletedAt.AsTime().In(time.Local)
		}
		if isFinished {
			finished = append(finished, g)
		} else {
			ongoing = append(ongoing, g)
		}
	}

	// Finished: newest top-level completion first.
	sort.Slice(finished, func(i, j int) bool {
		ti, tj := byID[finished[i].TopLevelID], byID[finished[j].TopLevelID]
		if ti == nil || tj == nil {
			return false
		}
		return ti.CompletedAt.AsTime().After(tj.CompletedAt.AsTime())
	})

	return finished, ongoing
}
