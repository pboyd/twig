package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

const defaultAddr = "http://localhost:8080"

// Run is the entrypoint for the CLI. It returns the process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		printRootUsage()
		return 1
	}

	switch args[0] {
	case "task":
		return runTask(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		printRootUsage()
		return 1
	}
}

func printRootUsage() {
	fmt.Fprintln(os.Stderr, "Usage: todo <command> [arguments]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Commands:")
	fmt.Fprintln(os.Stderr, "  task  Manage tasks")
}

func runTask(args []string) int {
	if len(args) == 0 {
		printTaskUsage()
		return 1
	}

	apiKey := os.Getenv("TODO_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: TODO_API_KEY is not set")
		return 1
	}

	addr := os.Getenv("TODO_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	client := taskv1connect.NewTaskServiceClient(
		&http.Client{},
		addr,
		connect.WithSendGzip(),
		connect.WithInterceptors(bearerInterceptor(apiKey)),
	)

	switch args[0] {
	case "list":
		return runList(client, args[1:])
	case "add":
		return runAdd(client, args[1:])
	case "rm":
		return runRm(client, args[1:])
	case "mod":
		return runMod(client, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		printTaskUsage()
		return 1
	}
}

func printTaskUsage() {
	fmt.Fprintln(os.Stderr, "Usage: todo task <subcommand> [arguments]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  list             List all tasks as a tree")
	fmt.Fprintln(os.Stderr, "  add [--parent <id>] [--due <timestamp>] <name>")
	fmt.Fprintln(os.Stderr, "                   Add a new task")
	fmt.Fprintln(os.Stderr, "  rm <id>          Remove a task")
	fmt.Fprintln(os.Stderr, "  mod [--parent <id>] [--due <timestamp>] <id> <name>")
	fmt.Fprintln(os.Stderr, "                   Modify a task")
}

// bearerInterceptor returns a Connect interceptor that adds an
// Authorization: Bearer header to every outgoing unary request.
func bearerInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+apiKey)
			return next(ctx, req)
		})
	})
}
