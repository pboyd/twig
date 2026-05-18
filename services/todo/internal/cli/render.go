package cli

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// mapError converts an RPC or transport error to a user-facing message.
// Returns the message string (to be printed to stderr).
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

// parseDue parses a --due flag value. Accepts RFC 3339 or bare YYYY-MM-DD
// (interpreted as 00:00:00Z). Returns a usage error on failure.
func parseDue(v string) (*timestamppb.Timestamp, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return timestamppb.New(t), nil
	}
	if t, err := time.Parse("2006-01-02", v); err == nil {
		ts := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		return timestamppb.New(ts), nil
	}
	return nil, fmt.Errorf("invalid --due value %q: use RFC 3339 (e.g. 2006-01-02T15:04:05Z) or YYYY-MM-DD", v)
}

// formatDue formats a proto timestamp as RFC 3339 UTC, or empty string if nil.
func formatDue(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

// treeNode is used internally for rendering the task tree.
type treeNode struct {
	task     *taskv1.Task
	children []*treeNode
}

// buildTree builds a tree from a flat list of tasks.
// Roots are tasks with no parent; children are attached in id-ascending order.
func buildTree(tasks []*taskv1.Task) []*treeNode {
	byID := make(map[int64]*treeNode, len(tasks))
	for _, t := range tasks {
		byID[t.Id] = &treeNode{task: t}
	}

	var roots []*treeNode
	for _, t := range tasks {
		node := byID[t.Id]
		if t.ParentId == nil {
			roots = append(roots, node)
		} else {
			parent, ok := byID[t.GetParentId()]
			if !ok {
				// Parent not in list — treat as root
				roots = append(roots, node)
			} else {
				parent.children = append(parent.children, node)
			}
		}
	}

	sortNodes(roots)
	return roots
}

func sortNodes(nodes []*treeNode) {
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].task.Id < nodes[j].task.Id
	})
	for _, n := range nodes {
		sortNodes(n.children)
	}
}

// renderTree writes the tree to w using tree-style ASCII connectors.
func renderTree(w io.Writer, nodes []*treeNode, prefix string, isLast bool) {
	for i, node := range nodes {
		last := i == len(nodes)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if last {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		due := formatDue(node.task.Due)
		line := fmt.Sprintf("%s%s[%d] %s", prefix, connector, node.task.Id, node.task.Name)
		if due != "" {
			line += fmt.Sprintf(" (due %s)", due)
		}
		fmt.Fprintln(w, line)
		renderTree(w, node.children, childPrefix, last)
	}
}

// renderRoots writes root nodes (no leading connector).
func renderRoots(w io.Writer, roots []*treeNode) {
	for _, root := range roots {
		due := formatDue(root.task.Due)
		line := fmt.Sprintf("[%d] %s", root.task.Id, root.task.Name)
		if due != "" {
			line += fmt.Sprintf(" (due %s)", due)
		}
		fmt.Fprintln(w, line)
		renderTree(w, root.children, "", false)
	}
}
