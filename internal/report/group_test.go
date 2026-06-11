package report

import (
	"testing"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
)

func ts(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

func int64Ptr(v int64) *int64 { return &v }

func TestBuildEntries_InRange(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	noon := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "task1", CompletedAt: ts(noon)},
	}

	entries := BuildEntries(tasks, p)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].TaskID != 1 {
		t.Errorf("TaskID = %d, want 1", entries[0].TaskID)
	}
}

func TestBuildEntries_OutOfRange(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	yesterday := time.Date(2025, 6, 10, 12, 0, 0, 0, time.Local).UTC()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "task1", CompletedAt: ts(yesterday)},
	}

	entries := BuildEntries(tasks, p)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestBuildEntries_ExclusiveUpperBound(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	// Exactly at EndUTC — must NOT be included.
	atEnd := p.EndUTC()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "task1", CompletedAt: ts(atEnd)},
	}
	entries := BuildEntries(tasks, p)
	if len(entries) != 0 {
		t.Errorf("task exactly at EndUTC should be excluded")
	}

	// One nanosecond before EndUTC — must be included.
	justBefore := atEnd.Add(-1)
	tasks2 := []*taskv1.Task{
		{Id: 2, Name: "task2", CompletedAt: ts(justBefore)},
	}
	entries2 := BuildEntries(tasks2, p)
	if len(entries2) != 1 {
		t.Errorf("task just before EndUTC should be included")
	}
}

func TestBuildEntries_UncompletedExcluded(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	tasks := []*taskv1.Task{
		{Id: 1, Name: "incomplete"},
	}
	entries := BuildEntries(tasks, p)
	if len(entries) != 0 {
		t.Fatalf("incomplete task should not appear in entries")
	}
}

func TestBuildEntries_SubtaskParentContext(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	noon := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "parent"},
		{Id: 2, Name: "child", ParentId: int64Ptr(1), CompletedAt: ts(noon)},
	}

	entries := BuildEntries(tasks, p)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	e := entries[0]
	if e.ParentName != "parent" {
		t.Errorf("ParentName = %q, want 'parent'", e.ParentName)
	}
	if e.TopLevelID != 1 {
		t.Errorf("TopLevelID = %d, want 1", e.TopLevelID)
	}
	if e.Depth != 1 {
		t.Errorf("Depth = %d, want 1", e.Depth)
	}
}

func TestBuildEntries_NearMidnightLocalDay(t *testing.T) {
	// A task completed at 23:59:59 on Jun 11 local time should be in Jun 11's window.
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	nearMidnight := time.Date(2025, 6, 11, 23, 59, 59, 0, time.Local).UTC()
	tasks := []*taskv1.Task{
		{Id: 1, Name: "task1", CompletedAt: ts(nearMidnight)},
	}
	entries := BuildEntries(tasks, p)
	if len(entries) != 1 {
		t.Fatalf("task at 23:59:59 local should be in today's entries")
	}
	day := truncateDay(entries[0].CompletedAt)
	wantDay := time.Date(2025, 6, 11, 0, 0, 0, 0, time.Local)
	if !day.Equal(wantDay) {
		t.Errorf("entry attributed to day %v, want %v", day, wantDay)
	}
}

func TestGroupByDay_OrderingAndGrouping(t *testing.T) {
	d1 := time.Date(2025, 6, 11, 10, 0, 0, 0, time.Local)
	d2 := time.Date(2025, 6, 10, 15, 0, 0, 0, time.Local)
	d3 := time.Date(2025, 6, 10, 9, 0, 0, 0, time.Local)

	entries := []Entry{
		{TaskID: 1, CompletedAt: d1},
		{TaskID: 2, CompletedAt: d2},
		{TaskID: 3, CompletedAt: d3},
	}

	groups := GroupByDay(entries)
	if len(groups) != 2 {
		t.Fatalf("got %d groups, want 2", len(groups))
	}
	// Most recent day first.
	if !groups[0].Day.Equal(truncateDay(d1)) {
		t.Errorf("groups[0].Day = %v, want Jun 11", groups[0].Day)
	}
	if !groups[1].Day.Equal(truncateDay(d2)) {
		t.Errorf("groups[1].Day = %v, want Jun 10", groups[1].Day)
	}
	// Jun 10 group: entries ordered by CompletedAt desc.
	if groups[1].Entries[0].TaskID != 2 {
		t.Errorf("Jun 10 first entry = task %d, want task 2 (15:00)", groups[1].Entries[0].TaskID)
	}
	if groups[1].Entries[1].TaskID != 3 {
		t.Errorf("Jun 10 second entry = task %d, want task 3 (09:00)", groups[1].Entries[1].TaskID)
	}
}

func TestGroupByAccomplishment_FinishedVsOngoing(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	noon := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()

	// topA completed in period — Finished.
	// topB not completed — ongoing.
	tasks := []*taskv1.Task{
		{Id: 1, Name: "topA", CompletedAt: ts(noon)},
		{Id: 2, Name: "childA", ParentId: int64Ptr(1), CompletedAt: ts(noon)},
		{Id: 3, Name: "topB"},
		{Id: 4, Name: "childB", ParentId: int64Ptr(3), CompletedAt: ts(noon)},
	}

	entries := BuildEntries(tasks, p)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	finished, ongoing := GroupByAccomplishment(entries, tasks, p)

	if len(finished) != 1 {
		t.Fatalf("finished groups = %d, want 1", len(finished))
	}
	if finished[0].TopLevelID != 1 {
		t.Errorf("finished[0] TopLevelID = %d, want 1", finished[0].TopLevelID)
	}
	if !finished[0].Finished {
		t.Errorf("finished[0].Finished should be true")
	}
	wantDone := noon.In(time.Local)
	if !finished[0].TopLevelCompletedAt.Equal(wantDone) {
		t.Errorf("finished[0].TopLevelCompletedAt = %v, want %v", finished[0].TopLevelCompletedAt, wantDone)
	}

	if len(ongoing) != 1 {
		t.Fatalf("ongoing groups = %d, want 1", len(ongoing))
	}
	if ongoing[0].TopLevelID != 3 {
		t.Errorf("ongoing[0] TopLevelID = %d, want 3", ongoing[0].TopLevelID)
	}
	if ongoing[0].Finished {
		t.Errorf("ongoing[0].Finished should be false")
	}
	if !ongoing[0].TopLevelCompletedAt.IsZero() {
		t.Errorf("ongoing[0].TopLevelCompletedAt = %v, want zero", ongoing[0].TopLevelCompletedAt)
	}
}

func TestGroupByAccomplishment_AllEntriesPlacedExactlyOnce(t *testing.T) {
	// SC-004 invariant: every entry appears in exactly one group in exactly one section.
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("week", "", "", now)

	noon := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()

	tasks := []*taskv1.Task{
		{Id: 1, Name: "topA", CompletedAt: ts(noon)},
		{Id: 2, Name: "sub1", ParentId: int64Ptr(1), CompletedAt: ts(noon)},
		{Id: 3, Name: "topB"},
		{Id: 4, Name: "sub2", ParentId: int64Ptr(3), CompletedAt: ts(noon)},
		{Id: 5, Name: "deep", ParentId: int64Ptr(2), CompletedAt: ts(noon)},
	}

	entries := BuildEntries(tasks, p)
	finished, ongoing := GroupByAccomplishment(entries, tasks, p)

	seen := make(map[int64]bool)
	for _, g := range finished {
		for _, e := range g.Entries {
			if seen[e.TaskID] {
				t.Errorf("task %d appears more than once", e.TaskID)
			}
			seen[e.TaskID] = true
		}
	}
	for _, g := range ongoing {
		for _, e := range g.Entries {
			if seen[e.TaskID] {
				t.Errorf("task %d appears more than once", e.TaskID)
			}
			seen[e.TaskID] = true
		}
	}
	if len(seen) != len(entries) {
		t.Errorf("placed %d entries, total = %d — some entries are missing", len(seen), len(entries))
	}
}

func TestGroupByAccomplishment_TopLevelCompletedOutsidePeriodIsOngoing(t *testing.T) {
	// A top-level task completed before the period but with in-period descendants
	// goes to ongoing (not finished).
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.Local)
	p, _ := ParsePeriod("today", "", "", now)

	yesterday := time.Date(2025, 6, 10, 12, 0, 0, 0, time.Local).UTC()
	noon := time.Date(2025, 6, 11, 12, 0, 0, 0, time.Local).UTC()

	tasks := []*taskv1.Task{
		{Id: 1, Name: "topA", CompletedAt: ts(yesterday)}, // completed outside period
		{Id: 2, Name: "child", ParentId: int64Ptr(1), CompletedAt: ts(noon)},
	}

	entries := BuildEntries(tasks, p)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	finished, ongoing := GroupByAccomplishment(entries, tasks, p)
	if len(finished) != 0 {
		t.Errorf("finished groups = %d, want 0", len(finished))
	}
	if len(ongoing) != 1 {
		t.Errorf("ongoing groups = %d, want 1", len(ongoing))
	}
}
