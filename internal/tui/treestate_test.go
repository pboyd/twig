package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// helpers

func sortedIDs(m map[int64]bool) []int64 {
	var ids []int64
	for id, ok := range m {
		if ok {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// ── treeStatePath ─────────────────────────────────────────────────────────────

func TestTreeStatePath_XDGStateHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	got, err := treeStatePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dir, "twig", "tree-state.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTreeStatePath_FallbackHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", home)

	got, err := treeStatePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, ".local", "state", "twig", "tree-state.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// ── profileKey ────────────────────────────────────────────────────────────────

func TestProfileKey_EmptyIsDefault(t *testing.T) {
	if profileKey("") != "default" {
		t.Error(`profileKey("") should return "default"`)
	}
}

func TestProfileKey_DefaultIsDefault(t *testing.T) {
	if profileKey("default") != "default" {
		t.Error(`profileKey("default") should return "default"`)
	}
}

func TestProfileKey_EmptyAndDefaultMatch(t *testing.T) {
	if profileKey("") != profileKey("default") {
		t.Error(`profileKey("") and profileKey("default") must return the same key`)
	}
}

func TestProfileKey_Named(t *testing.T) {
	if got := profileKey("work"); got != "work" {
		t.Errorf(`profileKey("work") = %q, want "work"`, got)
	}
}

// ── loadTreeState ─────────────────────────────────────────────────────────────

func TestLoadTreeState_MissingFile(t *testing.T) {
	tsf := loadTreeState(filepath.Join(t.TempDir(), "no-such-file.json"))
	if len(tsf) != 0 {
		t.Errorf("expected empty treeStateFile, got %v", tsf)
	}
}

func TestLoadTreeState_CorruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tree-state.json")
	if err := os.WriteFile(path, []byte("not json {{{{"), 0o600); err != nil {
		t.Fatal(err)
	}

	tsf := loadTreeState(path)
	if len(tsf) != 0 {
		t.Errorf("expected empty treeStateFile on corrupt file, got %v", tsf)
	}
}

func TestLoadTreeState_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tree-state.json")
	if err := os.WriteFile(path, []byte(`{"default":[1,4,9]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tsf := loadTreeState(path)
	got := sortedIDs(tsf.expandedSet("default"))
	want := []int64{1, 4, 9}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// ── expandedSet ───────────────────────────────────────────────────────────────

func TestExpandedSet_MissingKey(t *testing.T) {
	tsf := treeStateFile{}
	if m := tsf.expandedSet("default"); len(m) != 0 {
		t.Errorf("expected nil/empty set for missing key, got %v", m)
	}
}

// ── save → load round-trip ────────────────────────────────────────────────────

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "twig", "tree-state.json")

	expanded := map[int64]bool{1: true, 4: true, 9: true}
	liveIDs := map[int64]bool{1: true, 4: true, 9: true, 10: true}

	if err := saveTreeState(path, "default", expanded, liveIDs); err != nil {
		t.Fatalf("saveTreeState: %v", err)
	}

	tsf := loadTreeState(path)
	got := sortedIDs(tsf.expandedSet("default"))
	want := []int64{1, 4, 9}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip: got %v, want %v", got, want)
	}
}

// ── pruning ───────────────────────────────────────────────────────────────────

func TestSaveTreeState_PrunesDeletedIDs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tree-state.json")

	expanded := map[int64]bool{1: true, 99: true, 7: true}
	// 99 is not in the live set — simulates a deleted task.
	liveIDs := map[int64]bool{1: true, 7: true}

	if err := saveTreeState(path, "default", expanded, liveIDs); err != nil {
		t.Fatalf("saveTreeState: %v", err)
	}

	tsf := loadTreeState(path)
	got := sortedIDs(tsf.expandedSet("default"))
	want := []int64{1, 7}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pruning: got %v, want %v", got, want)
	}
}

// ── per-profile isolation (US3 / FR-009) ─────────────────────────────────────

func TestSaveTreeState_PreservesOtherProfiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tree-state.json")

	// Seed a "default" entry.
	if err := os.WriteFile(path, []byte(`{"default":[1,2]}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Save under "work" — must not touch "default".
	if err := saveTreeState(path, "work", map[int64]bool{12: true}, map[int64]bool{12: true}); err != nil {
		t.Fatalf("saveTreeState: %v", err)
	}

	tsf := loadTreeState(path)

	// "default" must remain unchanged.
	gotDefault := sortedIDs(tsf.expandedSet("default"))
	wantDefault := []int64{1, 2}
	if !reflect.DeepEqual(gotDefault, wantDefault) {
		t.Errorf("default profile changed: got %v, want %v", gotDefault, wantDefault)
	}

	// "work" must be written correctly.
	gotWork := sortedIDs(tsf.expandedSet("work"))
	wantWork := []int64{12}
	if !reflect.DeepEqual(gotWork, wantWork) {
		t.Errorf("work profile: got %v, want %v", gotWork, wantWork)
	}
}

func TestSaveTreeState_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	// Use a nested path that doesn't exist yet.
	path := filepath.Join(dir, "newdir", "sub", "tree-state.json")

	if err := saveTreeState(path, "default", map[int64]bool{3: true}, map[int64]bool{3: true}); err != nil {
		t.Fatalf("saveTreeState: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}
