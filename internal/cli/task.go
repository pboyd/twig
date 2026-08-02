package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
)

// setTaskGoal calls SetTaskGoal for the given task. goalStr is the --goal flag
// value: a numeric ID to associate, or "none" to clear. Returns false and prints
// an error on failure.
func setTaskGoal(client taskv1connect.TaskServiceClient, addr string, taskID int64, goalStr string) bool {
	req := &taskv1.SetTaskGoalRequest{TaskId: taskID}
	if goalStr != "none" {
		gid, err := strconv.ParseInt(goalStr, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--goal must be a goal id or 'none', got %q\n", goalStr)
			return false
		}
		req.GoalId = &gid
	}
	_, err := client.SetTaskGoal(context.Background(), connect.NewRequest(req))
	if err != nil {
		if isFailedPrecondition(err) {
			fmt.Fprintln(os.Stderr, nestingConflictMsg(err))
		} else {
			fmt.Fprintln(os.Stderr, mapError(err, addr))
		}
		return false
	}
	return true
}

// isFailedPrecondition returns true when err is a ConnectRPC FailedPrecondition error.
func isFailedPrecondition(err error) bool {
	ce, ok := err.(*connect.Error)
	return ok && ce.Code() == connect.CodeFailedPrecondition
}

// nestingConflictMsg returns the playful nesting-conflict error copy.
func nestingConflictMsg(_ error) string {
	return "twig: that subtree already belongs to a goal — clear that link first."
}

func runList(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var showCompleted bool
	var showAll bool
	fs.BoolVar(&showCompleted, "completed", false, "show only completed tasks")
	fs.BoolVar(&showAll, "all", false, "show all tasks")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if showCompleted && showAll {
		fmt.Fprintln(os.Stderr, "--completed and --all are mutually exclusive")
		return 1
	}

	resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	roots := BuildTree(resp.Msg.Tasks)
	switch {
	case showCompleted:
		roots = filterCompleted(roots)
	case showAll:
		// no filtering
	default:
		roots = pruneIncomplete(roots, time.Now().Local())
	}

	if len(roots) == 0 {
		fmt.Println("no tasks")
		return 0
	}

	renderRoots(os.Stdout, roots, WantStyled(os.Stdout))
	return 0
}

func runComplete(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task complete <id>")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	before := time.Now()
	resp, err := client.CompleteTask(context.Background(), connect.NewRequest(&taskv1.CompleteTaskRequest{Id: id}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	task := resp.Msg.Task
	if task.CompletedAt != nil && task.CompletedAt.AsTime().Before(before) {
		fmt.Printf("task %d already complete (at %s)\n", id, task.CompletedAt.AsTime().UTC().Format(time.RFC3339))
	} else {
		fmt.Printf("completed task %d\n", id)
	}
	return 0
}

func runUncomplete(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task uncomplete <id>")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	getResp, err := client.GetTask(context.Background(), connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	if getResp.Msg.Task.CompletedAt == nil {
		fmt.Printf("task %d was already incomplete\n", id)
		return 0
	}

	_, err = client.UncompleteTask(context.Background(), connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: id}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	fmt.Printf("task %d is back on your list\n", id)
	return 0
}

func runAdd(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var parentStr string
	var dueStr string
	var goalStr string
	fs.StringVar(&parentStr, "parent", "", "parent task id")
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")
	fs.StringVar(&goalStr, "goal", "", "goal id to associate with")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task add [--parent <id>] [--due <timestamp>] [--goal <id>] <name>")
		return 1
	}
	name := fs.Arg(0)

	req := &taskv1.CreateTaskRequest{Name: name}

	if parentStr != "" {
		pid, err := strconv.ParseInt(parentStr, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--parent must be an integer, got %q\n", parentStr)
			return 1
		}
		req.ParentId = &pid
	}

	if dueStr != "" {
		ts, err := ParseDue(dueStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		req.Due = ts
	}

	resp, err := client.CreateTask(context.Background(), connect.NewRequest(req))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	newID := resp.Msg.Task.Id
	fmt.Printf("created task %d\n", newID)

	if goalStr != "" {
		if !setTaskGoal(client, addr, newID, goalStr) {
			return 1
		}
	}
	return 0
}

func runTaskShow(client taskv1connect.TaskServiceClient, goalClient goalv1connect.GoalServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task show <id>")
		return 1
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	resp, err := client.GetTask(context.Background(), connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
	if err != nil {
		if isNotFound(err) {
			fmt.Fprintf(os.Stderr, "twig: task %d not found.\n", id)
			return 1
		}
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	task := resp.Msg.Task

	fmt.Printf("Name:  %s\n", task.Name)
	if task.Description != "" {
		fmt.Printf("Desc:  %s\n", task.Description)
	}
	if due := FormatDue(task.Due); due != "" {
		fmt.Printf("Due:   %s\n", due)
	}

	if task.GoalId != nil {
		gResp, err := goalClient.GetGoal(context.Background(), connect.NewRequest(&goalv1.GetGoalRequest{Id: task.GetGoalId()}))
		if err == nil {
			fmt.Printf("Goal:  %s\n", gResp.Msg.Goal.Name)
		}
	}
	return 0
}

func runRm(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task rm <id>")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	_, err = client.DeleteTask(context.Background(), connect.NewRequest(&taskv1.DeleteTaskRequest{Id: id}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	fmt.Printf("deleted task %d\n", id)
	return 0
}

func runMod(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task mod <id> [<name>] [--parent <id>] [--due <timestamp>] [--goal <id>|none]")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	fs := flag.NewFlagSet("mod", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var parentStr string
	var dueStr string
	var goalStr string
	fs.StringVar(&parentStr, "parent", "", "parent task id")
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")
	fs.StringVar(&goalStr, "goal", "", "goal id to associate with, or 'none' to clear")

	// Pre-separate flag tokens from positionals so flags work in any position.
	var flagTokens []string
	var positionals []string
	for i := 1; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flagTokens = append(flagTokens, a)
			// Consume next token as flag value if it doesn't look like a flag.
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flagTokens = append(flagTokens, args[i+1])
				i++
			}
		} else {
			positionals = append(positionals, a)
		}
	}

	if err := fs.Parse(flagTokens); err != nil {
		return 1
	}

	if len(positionals) > 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task mod <id> [<name>] [--parent <id>] [--due <timestamp>] [--goal <id>|none]")
		return 1
	}

	var newName string
	hasName := false
	if len(positionals) == 1 {
		if positionals[0] == "" {
			fmt.Fprintln(os.Stderr, "task name cannot be empty")
			return 1
		}
		newName = positionals[0]
		hasName = true
	}

	if !hasName && parentStr == "" && dueStr == "" && goalStr == "" {
		fmt.Fprintln(os.Stderr, "nothing to update")
		return 1
	}

	// If only --goal is being changed, call SetTaskGoal without touching UpdateTask.
	if goalStr != "" && !hasName && parentStr == "" && dueStr == "" {
		if !setTaskGoal(client, addr, id, goalStr) {
			return 1
		}
		fmt.Printf("updated task %d\n", id)
		return 0
	}

	// Fetch current state to preserve unflagged fields (fetch-then-update)
	getResp, err := client.GetTask(context.Background(), connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	existing := getResp.Msg.Task

	name := existing.Name
	if hasName {
		name = newName
	}

	req := &taskv1.UpdateTaskRequest{
		Id:          id,
		Name:        name,
		Description: existing.Description,
		SnoozeUntil: existing.SnoozeUntil,
		Due:         existing.Due,
		ParentId:    existing.ParentId,
	}

	if parentStr != "" {
		pid, err := strconv.ParseInt(parentStr, 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "--parent must be an integer, got %q\n", parentStr)
			return 1
		}
		req.ParentId = &pid
	}

	if dueStr != "" {
		ts, err := ParseDue(dueStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		req.Due = ts
	}

	_, err = client.UpdateTask(context.Background(), connect.NewRequest(req))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	// Report the name/due update before attempting the goal association so the
	// user knows those changes were applied even if the goal change fails.
	fmt.Printf("updated task %d\n", id)

	// Handle goal association after UpdateTask.
	if goalStr != "" {
		if !setTaskGoal(client, addr, id, goalStr) {
			return 1
		}
	}

	return 0
}
