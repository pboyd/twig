package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/todo/services/todo/gen/task/v1"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

func runList(client taskv1connect.TaskServiceClient, args []string) int {
	addr := backendAddr()
	resp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	tasks := resp.Msg.Tasks
	if len(tasks) == 0 {
		fmt.Println("no tasks")
		return 0
	}

	roots := buildTree(tasks)
	renderRoots(os.Stdout, roots)
	return 0
}

func runAdd(client taskv1connect.TaskServiceClient, args []string) int {
	addr := backendAddr()
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
		fmt.Fprintln(os.Stderr, "usage: todo task add [--parent <id>] [--due <timestamp>] <name>")
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
		ts, err := parseDue(dueStr)
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

func runRm(client taskv1connect.TaskServiceClient, args []string) int {
	addr := backendAddr()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: todo task rm <id>")
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

func runMod(client taskv1connect.TaskServiceClient, args []string) int {
	addr := backendAddr()
	fs := flag.NewFlagSet("mod", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var parentStr string
	var dueStr string
	fs.StringVar(&parentStr, "parent", "", "parent task id")
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: todo task mod [--parent <id>] [--due <timestamp>] <id> <name>")
		return 1
	}

	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", fs.Arg(0))
		return 1
	}
	name := fs.Arg(1)

	// Fetch current state to preserve unflagged fields (fetch-then-update)
	getResp, err := client.GetTask(context.Background(), connect.NewRequest(&taskv1.GetTaskRequest{Id: id}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	existing := getResp.Msg.Task

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
		ts, err := parseDue(dueStr)
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

func backendAddr() string {
	addr := os.Getenv("TODO_ADDR")
	if addr == "" {
		addr = defaultAddr
	}
	return addr
}
