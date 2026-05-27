package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	taskv1connect "github.com/pboyd/todo/services/todo/gen/task/v1/taskv1connect"
)

const defaultAddr = "http://localhost:8080"

// Run is the entrypoint for the CLI. It returns the process exit code.
func Run(args []string) int {
	if len(args) == 0 {
		printRootUsage(os.Stderr)
		return 1
	}

	if args[0] == "--help" || args[0] == "-h" {
		return runHelp(nil)
	}

	switch args[0] {
	case "task":
		return runTask(args[1:])
	case "pom":
		return runPomTop(args[1:])
	case "plan":
		return runPlan(args[1:])
	case "help":
		return runHelp(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'todo help' for usage.")
		return 1
	}
}

func runHelp(args []string) int {
	if len(args) == 0 {
		printRootUsage(os.Stdout)
		return 0
	}
	switch args[0] {
	case "task":
		printTaskUsage(os.Stdout)
		return 0
	case "pom":
		printPomUsage(os.Stdout)
		return 0
	case "plan":
		printPlanUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nRun 'todo help' for usage.\n", args[0])
		return 1
	}
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: todo <command> [arguments]")
	fmt.Fprintln(w, "       todo help [command]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  task   Manage tasks (add, remove, modify, list)")
	fmt.Fprintln(w, "  pom    Pomodoro timer (estimate, start, resume, cancel, status)")
	fmt.Fprintln(w, "  plan   Daily planning (schedule tasks and events)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Run 'todo help <command>' for command-specific help.")
}

func runTask(args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printTaskUsage(os.Stdout)
		return 0
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
		connect.WithInterceptors(BearerInterceptor(apiKey)),
	)

	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runList(client, args)
	}

	switch args[0] {
	case "add":
		return runAdd(client, args[1:])
	case "rm":
		return runRm(client, args[1:])
	case "mod":
		return runMod(client, args[1:])
	case "complete":
		return runComplete(client, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'todo help task' for usage.")
		return 1
	}
}

func printTaskUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: todo task [<subcommand>] [arguments]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands (if omitted, lists tasks):")
	fmt.Fprintln(w, "  add [--parent <id>] [--due <date>] <name>")
	fmt.Fprintln(w, "                   Add a new task")
	fmt.Fprintln(w, "  rm <id>          Remove a task")
	fmt.Fprintln(w, "  mod <id> [<name>] [--parent <id>] [--due <date>]")
	fmt.Fprintln(w, "                   Modify a task")
	fmt.Fprintln(w, "  complete <id>    Mark a task complete")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Listing tasks:")
	fmt.Fprintln(w, "  todo task [--completed | --all]")
	fmt.Fprintln(w, "  --completed      Show only completed tasks")
	fmt.Fprintln(w, "  --all            Show all tasks (including completed)")
}

// BearerInterceptor returns a Connect interceptor that adds an
// Authorization: Bearer header to every outgoing unary request.
func BearerInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return connect.UnaryFunc(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+apiKey)
			return next(ctx, req)
		})
	})
}
