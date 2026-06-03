package tui

import (
	"testing"
)

// ── T005: resolveEditor and trailing-newline normalization (US2) ──────────────

func TestResolveEditor_EmptyUsesVim(t *testing.T) {
	name, args := resolveEditor("", "/tmp/twig-desc-123.md")
	if name != "vim" {
		t.Errorf("empty EDITOR: name = %q, want %q", name, "vim")
	}
	if len(args) != 1 || args[0] != "/tmp/twig-desc-123.md" {
		t.Errorf("empty EDITOR: args = %v, want [/tmp/twig-desc-123.md]", args)
	}
}

func TestResolveEditor_PlainVim(t *testing.T) {
	name, args := resolveEditor("vim", "/tmp/f.md")
	if name != "vim" {
		t.Errorf("name = %q, want %q", name, "vim")
	}
	if len(args) != 1 || args[0] != "/tmp/f.md" {
		t.Errorf("args = %v, want [/tmp/f.md]", args)
	}
}

func TestResolveEditor_CodeWait(t *testing.T) {
	name, args := resolveEditor("code --wait", "/tmp/f.md")
	if name != "code" {
		t.Errorf("name = %q, want %q", name, "code")
	}
	if len(args) != 2 || args[0] != "--wait" || args[1] != "/tmp/f.md" {
		t.Errorf("args = %v, want [--wait /tmp/f.md]", args)
	}
}

func TestResolveEditor_EmacsClientWithSpaces(t *testing.T) {
	name, args := resolveEditor("  emacsclient  -nw ", "/tmp/f.md")
	if name != "emacsclient" {
		t.Errorf("name = %q, want %q", name, "emacsclient")
	}
	if len(args) != 2 || args[0] != "-nw" || args[1] != "/tmp/f.md" {
		t.Errorf("args = %v, want [-nw /tmp/f.md]", args)
	}
}

func TestResolveEditor_PathIsAlwaysLast(t *testing.T) {
	_, args := resolveEditor("code --wait", "/tmp/desc.md")
	if args[len(args)-1] != "/tmp/desc.md" {
		t.Errorf("path must be last arg; got %v", args)
	}
}

// ── trailing-newline normalization ────────────────────────────────────────────

func TestNormalizeEditorContent_TrailingNewline(t *testing.T) {
	if got := normalizeEditorContent("hello\n"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestNormalizeEditorContent_NoTrailingNewline(t *testing.T) {
	if got := normalizeEditorContent("hello"); got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

func TestNormalizeEditorContent_InternalBlankLineTrailingNewline(t *testing.T) {
	// "a\n\nb\n" → "a\n\nb" (internal blank line kept; one trailing \n removed)
	if got := normalizeEditorContent("a\n\nb\n"); got != "a\n\nb" {
		t.Errorf("got %q, want %q", got, "a\n\nb")
	}
}

func TestNormalizeEditorContent_Empty(t *testing.T) {
	if got := normalizeEditorContent(""); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
