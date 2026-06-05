package tui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pboyd/twig/internal/cli"
)

// treeStateFile is the on-disk JSON format: profile key → expanded task IDs.
// See specs/036-tui-tree-state-persistence/contracts/tree-state-file.md.
type treeStateFile map[string][]int64

// treeStatePath resolves the path to the tree-state JSON file following the
// XDG Base Directory Specification ($XDG_STATE_HOME, fallback ~/.local/state).
func treeStatePath() (string, error) {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		stateHome = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(stateHome, "twig", "tree-state.json"), nil
}

// profileKey normalizes the profile name: "" and "default" both become "default".
func profileKey(profile string) string {
	if profile == "" || profile == "default" {
		return "default"
	}
	return profile
}

// loadTreeState reads the state file at path. A missing or corrupt file returns
// an empty treeStateFile with no error (FR-005, FR-006).
func loadTreeState(path string) treeStateFile {
	data, err := os.ReadFile(path)
	if err != nil {
		return treeStateFile{}
	}
	var tsf treeStateFile
	if err := json.Unmarshal(data, &tsf); err != nil {
		return treeStateFile{}
	}
	return tsf
}

// expandedSet returns the set of expanded task IDs for the given profile key.
func (tsf treeStateFile) expandedSet(key string) map[int64]bool {
	ids := tsf[key]
	if len(ids) == 0 {
		return nil
	}
	m := make(map[int64]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}

// saveTreeState atomically writes the active profile's expanded IDs to path,
// preserving other profiles' entries and pruning IDs not in liveIDs (FR-007,
// FR-009). The directory is created if absent.
func saveTreeState(path, key string, expanded map[int64]bool, liveIDs map[int64]bool) error {
	// Load existing state to preserve other profiles (FR-009).
	tsf := loadTreeState(path)

	// Build pruned list for the active profile.
	var ids []int64
	for id, ok := range expanded {
		if ok && liveIDs[id] {
			ids = append(ids, id)
		}
	}
	if ids == nil {
		ids = []int64{}
	}
	tsf[key] = ids

	data, err := json.Marshal(tsf)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	// Atomic write: temp file in same directory, then rename.
	tmp, err := os.CreateTemp(dir, "tree-state-*.json")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}

// liveTaskIDs returns the set of all task IDs present in the tree.
func liveTaskIDs(tree []*cli.TreeNode) map[int64]bool {
	if len(tree) == 0 {
		return map[int64]bool{}
	}
	ids := make(map[int64]bool)
	collectTreeIDs(tree, ids)
	return ids
}

func collectTreeIDs(nodes []*cli.TreeNode, ids map[int64]bool) {
	for _, n := range nodes {
		ids[n.Task.Id] = true
		collectTreeIDs(n.Children, ids)
	}
}
