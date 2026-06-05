package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
)

// newFormInput builds a textinput.Model with a given value, for test setup.
func newFormInput(value string) textinput.Model {
	ti := textinput.New()
	ti.SetValue(value)
	return ti
}

// ── T004: buildPlanPreview construction tests ─────────────────────────────────

// TestBuildPlanPreview_PlanList returns nil in planList mode.
func TestBuildPlanPreview_PlanList(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planList
	got := m.buildPlanPreview()
	if got != nil {
		t.Errorf("planList mode: expected nil preview, got %+v", got)
	}
}

// TestBuildPlanPreview_InvalidStart returns nil when Start does not parse.
func TestBuildPlanPreview_InvalidStart(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planEventForm
	m.plan.form = planFormState{
		fields: []textinput.Model{
			newFormInput("My event"),
			newFormInput("notatime"), // invalid start
			newFormInput("30m"),
		},
	}
	got := m.buildPlanPreview()
	if got != nil {
		t.Errorf("invalid start: expected nil preview, got %+v", got)
	}
}

// TestBuildPlanPreview_EmptyStart returns nil when Start is empty.
func TestBuildPlanPreview_EmptyStart(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planEventForm
	m.plan.form = planFormState{
		fields: []textinput.Model{
			newFormInput("Event"),
			newFormInput(""), // empty start
			newFormInput(""),
		},
	}
	got := m.buildPlanPreview()
	if got != nil {
		t.Errorf("empty start: expected nil preview, got %+v", got)
	}
}

// TestBuildPlanPreview_EventForm builds a preview for planEventForm with name, start, duration.
func TestBuildPlanPreview_EventForm(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planEventForm
	m.plan.form = planFormState{
		fields: []textinput.Model{
			newFormInput("Team sync"),
			newFormInput("10:00"),
			newFormInput("30m"),
		},
	}
	got := m.buildPlanPreview()
	if got == nil {
		t.Fatal("planEventForm: expected non-nil preview")
	}
	if got.Id != previewID {
		t.Errorf("preview Id: want %d, got %d", previewID, got.Id)
	}
	if got.GetStartMinute() != 600 { // 10:00 = 600 min
		t.Errorf("preview StartMinute: want 600, got %d", got.GetStartMinute())
	}
	if got.DurationMinute != 30 {
		t.Errorf("preview DurationMinute: want 30, got %d", got.DurationMinute)
	}
	if !strings.Contains(got.Name, "Team sync") {
		t.Errorf("preview Name: want 'Team sync', got %q", got.Name)
	}
}

// TestBuildPlanPreview_EditForm builds a preview for planEdit with name, start, duration.
func TestBuildPlanPreview_EditForm(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planEdit
	m.plan.form = planFormState{
		fields: []textinput.Model{
			newFormInput("Updated name"),
			newFormInput("09:30"),
			newFormInput("1h"),
		},
		entryID: 5,
	}
	got := m.buildPlanPreview()
	if got == nil {
		t.Fatal("planEdit: expected non-nil preview")
	}
	if got.Id != previewID {
		t.Errorf("preview Id: want %d, got %d", previewID, got.Id)
	}
	if got.GetStartMinute() != 570 { // 09:30 = 570 min
		t.Errorf("preview StartMinute: want 570, got %d", got.GetStartMinute())
	}
	if got.DurationMinute != 60 {
		t.Errorf("preview DurationMinute: want 60, got %d", got.DurationMinute)
	}
}

// TestBuildPlanPreview_TaskTimeForm builds a preview for planTaskTime (start, duration).
func TestBuildPlanPreview_TaskTimeForm(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planTaskTime
	m.plan.form = planFormState{
		fields: []textinput.Model{
			newFormInput("14:00"),
			newFormInput("45m"),
		},
		taskID: 0,
	}
	got := m.buildPlanPreview()
	if got == nil {
		t.Fatal("planTaskTime: expected non-nil preview")
	}
	if got.GetStartMinute() != 840 { // 14:00 = 840 min
		t.Errorf("preview StartMinute: want 840, got %d", got.GetStartMinute())
	}
	if got.DurationMinute != 45 {
		t.Errorf("preview DurationMinute: want 45, got %d", got.DurationMinute)
	}
}

// TestBuildPlanPreview_BlankDuration uses 0 duration when duration is empty/invalid.
func TestBuildPlanPreview_BlankDuration(t *testing.T) {
	m := ExportNewModel(nil, nil)
	m.plan.mode = planEventForm
	m.plan.form = planFormState{
		fields: []textinput.Model{
			newFormInput("Quick note"),
			newFormInput("11:00"),
			newFormInput(""), // blank duration
		},
	}
	got := m.buildPlanPreview()
	if got == nil {
		t.Fatal("blank duration: expected non-nil preview (start is valid)")
	}
	if got.DurationMinute != 30 {
		t.Errorf("blank duration: want DurationMinute=30 (server default), got %d", got.DurationMinute)
	}
}

// ── T011: planPreviewConflicts tests ──────────────────────────────────────────

// TestPlanPreviewConflicts_Overlap returns the overlapping slot when preview overlaps another entry.
func TestPlanPreviewConflicts_Overlap(t *testing.T) {
	start := int32(600)
	preview := &planv1.PlanEntry{
		Id:             previewID,
		StartMinute:    &start,
		DurationMinute: 60, // 10:00–11:00
	}
	// Existing entry at 10:30–11:30.
	otherStart := int32(630)
	others := []*planv1.PlanEntry{
		{Id: 1, StartMinute: &otherStart, DurationMinute: 60},
	}
	conflicts := planPreviewConflicts(preview, others)
	// 10:30 (slot 630) and 10:45 (slot 645) should conflict.
	if !conflicts[630] {
		t.Errorf("expected slot 630 (10:30) to conflict")
	}
	if !conflicts[645] {
		t.Errorf("expected slot 645 (10:45) to conflict")
	}
}

// TestPlanPreviewConflicts_Touching returns no conflict for touching boundaries.
func TestPlanPreviewConflicts_Touching(t *testing.T) {
	start := int32(600)
	preview := &planv1.PlanEntry{
		Id:             previewID,
		StartMinute:    &start,
		DurationMinute: 60, // 10:00–11:00
	}
	// Other entry at 11:00–12:00 (touching end).
	otherStart := int32(660)
	others := []*planv1.PlanEntry{
		{Id: 2, StartMinute: &otherStart, DurationMinute: 60},
	}
	conflicts := planPreviewConflicts(preview, others)
	if len(conflicts) > 0 {
		t.Errorf("touching boundary: expected no conflicts, got %v", conflicts)
	}
}

// TestPlanPreviewConflicts_ExcludesEditTarget verifies self-exclusion works.
// (The edit target is excluded by the caller passing timedOthers without it.)
func TestPlanPreviewConflicts_NilPreview(t *testing.T) {
	conflicts := planPreviewConflicts(nil, nil)
	if len(conflicts) > 0 {
		t.Errorf("nil preview: expected nil/empty conflicts, got %v", conflicts)
	}
}
