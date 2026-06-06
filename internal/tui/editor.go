package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// editorFinishedMsg is returned when the external editor process exits.
type editorFinishedMsg struct {
	content string
	err     error
}

// resolveEditor parses editorEnv (the $EDITOR value) into a command name and
// args list with path appended last. Falls back to vim when editorEnv is empty.
func resolveEditor(editorEnv, path string) (name string, args []string) {
	parts := strings.Fields(editorEnv)
	if len(parts) == 0 {
		return "vim", []string{path}
	}
	return parts[0], append(parts[1:], path)
}

// normalizeEditorContent strips exactly one trailing newline from s so that
// repeated open/save cycles do not accumulate blank lines.
func normalizeEditorContent(s string) string {
	return strings.TrimSuffix(s, "\n")
}

// openEditorCmd writes text to a temp file, opens the user's $EDITOR via
// tea.ExecProcess, reads the result back on exit, and returns an editorFinishedMsg.
func openEditorCmd(text string) tea.Cmd {
	f, err := os.CreateTemp("", "twig-desc-*.md")
	if err != nil {
		return func() tea.Msg { return editorFinishedMsg{err: err} }
	}
	tmpPath := f.Name()
	if _, err := f.WriteString(text); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return func() tea.Msg { return editorFinishedMsg{err: err} }
	}
	f.Close()

	name, args := resolveEditor(os.Getenv("EDITOR"), tmpPath)
	cmd := exec.Command(name, args...)

	return tea.ExecProcess(cmd, func(execErr error) tea.Msg {
		defer os.Remove(tmpPath)
		if execErr != nil {
			return editorFinishedMsg{err: execErr}
		}
		content, readErr := os.ReadFile(tmpPath)
		if readErr != nil {
			return editorFinishedMsg{err: readErr}
		}
		return editorFinishedMsg{content: normalizeEditorContent(string(content))}
	})
}
