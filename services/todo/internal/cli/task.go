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
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

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
		roots = pruneIncomplete(roots)
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

func runAdd(client taskv1connect.TaskServiceClient, addr string, args []string) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var parentStr string
	var dueStr string
	fs.StringVar(&parentStr, "parent", "", "parent task id")
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig task add [--parent <id>] [--due <timestamp>] <name>")
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
	fmt.Printf("created task %d\n", resp.Msg.Task.Id)
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
		fmt.Fprintln(os.Stderr, "usage: twig task mod <id> [<name>] [--parent <id>] [--due <timestamp>]")
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
	fs.StringVar(&parentStr, "parent", "", "parent task id")
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")

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
		fmt.Fprintln(os.Stderr, "usage: twig task mod <id> [<name>] [--parent <id>] [--due <timestamp>]")
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

	if !hasName && parentStr == "" && dueStr == "" {
		fmt.Fprintln(os.Stderr, "nothing to update")
		return 1
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
	fmt.Printf("updated task %d\n", id)
	return 0
}
