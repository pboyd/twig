package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"golang.org/x/term"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mapError converts an RPC or transport error to a user-facing message.
func mapError(err error, addr string) string {
	var connectErr *connect.Error
	if errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnavailable:
			return fmt.Sprintf("cannot reach backend at %s", addr)
		default:
			return connectErr.Message()
		}
	}
	return fmt.Sprintf("cannot reach backend at %s: %v", addr, err)
}

// ParseDue parses a --due flag value. Accepts RFC 3339 or bare YYYY-MM-DD
// (interpreted as 00:00:00Z). Returns a usage error on failure.
func ParseDue(v string) (*timestamppb.Timestamp, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return timestamppb.New(t), nil
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		ts := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return timestamppb.New(ts), nil
	}
	return nil, fmt.Errorf("invalid --due value %q: use RFC 3339 (e.g. 2006-01-02T15:04:05Z) or YYYY-MM-DD", v)
}

// FormatDue formats a proto timestamp as RFC 3339 UTC, or empty string if nil.
func FormatDue(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

// TreeNode is the in-memory representation of a task and its children,
// used for tree rendering by both the CLI and the TUI.
type TreeNode struct {
	Task     *taskv1.Task
	Children []*TreeNode
}

// BuildTree builds a tree from a flat list of tasks.
// Roots are tasks with no parent; children are attached in id-ascending order.
func BuildTree(tasks []*taskv1.Task) []*TreeNode {
	byID := make(map[int64]*TreeNode, len(tasks))
	for _, t := range tasks {
		byID[t.Id] = &TreeNode{Task: t}
	}

	var roots []*TreeNode
	for _, t := range tasks {
		node := byID[t.Id]
		if t.ParentId == nil {
			roots = append(roots, node)
		} else {
			parent, ok := byID[t.GetParentId()]
			if !ok {
				roots = append(roots, node)
			} else {
				parent.Children = append(parent.Children, node)
			}
		}
	}

	SortNodes(roots)
	return roots
}

// SortNodes sorts nodes by id ascending, recursively.
func SortNodes(nodes []*TreeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Task.Id < nodes[j].Task.Id
	})
	for _, n := range nodes {
		SortNodes(n.Children)
	}
}

// FormatCompletedAt formats a proto timestamp as RFC 3339 UTC, or empty string if nil.
func FormatCompletedAt(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

// DimStrike wraps s with ANSI dim+strikethrough codes.
func DimStrike(s string) string {
	return "\x1b[2;9m" + s + "\x1b[0m"
}

// WantStyled reports whether w supports ANSI styling (i.e. is a TTY).
func WantStyled(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// BuildPostID constructs the post-id portion of a task line (name, estimate, due, completed).
func BuildPostID(task *taskv1.Task) string {
	s := task.Name
	if task.GetEstimate() > 0 {
		s += fmt.Sprintf(" (%d)", task.GetEstimate())
	}
	if due := FormatDue(task.Due); due != "" {
		s += fmt.Sprintf(" (due %s)", due)
	}
	if cat := FormatCompletedAt(task.CompletedAt); cat != "" {
		s += fmt.Sprintf(" (completed %s)", cat)
	}
	return s
}

// renderTree writes the tree to w using tree-style ASCII connectors.
func renderTree(w io.Writer, nodes []*TreeNode, prefix string, isLast bool, styled bool) {
	for i, node := range nodes {
		last := i == len(nodes)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if last {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		postID := BuildPostID(node.Task)
		if styled && node.Task.GetCompletedAt() != nil {
			postID = DimStrike(postID)
		}
		fmt.Fprintln(w, fmt.Sprintf("%s%s[%d] %s", prefix, connector, node.Task.Id, postID))
		renderTree(w, node.Children, childPrefix, last, styled)
	}
}

// renderRoots writes root nodes (no leading connector).
func renderRoots(w io.Writer, roots []*TreeNode, styled bool) {
	for _, root := range roots {
		postID := BuildPostID(root.Task)
		if styled && root.Task.GetCompletedAt() != nil {
			postID = DimStrike(postID)
		}
		fmt.Fprintln(w, fmt.Sprintf("[%d] %s", root.Task.Id, postID))
		renderTree(w, root.Children, "", false, styled)
	}
}

// pruneIncomplete returns nodes where the node itself or any descendant is incomplete.
func pruneIncomplete(roots []*TreeNode) []*TreeNode {
	var result []*TreeNode
	for _, node := range roots {
		prunedChildren := pruneIncomplete(node.Children)
		incomplete := node.Task.GetCompletedAt() == nil
		if incomplete || len(prunedChildren) > 0 {
			kept := &TreeNode{Task: node.Task, Children: prunedChildren}
			result = append(result, kept)
		}
	}
	return result
}

// filterCompleted returns nodes that are complete, promoting completed descendants
// of incomplete parents to the nearest kept ancestor's level (or root).
func filterCompleted(roots []*TreeNode) []*TreeNode {
	var result []*TreeNode
	for _, node := range roots {
		childResults := filterCompleted(node.Children)
		if node.Task.GetCompletedAt() != nil {
			kept := &TreeNode{Task: node.Task, Children: childResults}
			result = append(result, kept)
		} else {
			result = append(result, childResults...)
		}
	}
	return result
}
