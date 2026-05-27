package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/term"

	planv1 "github.com/pboyd/todo/services/todo/gen/plan/v1"
	planv1connect "github.com/pboyd/todo/services/todo/gen/plan/v1/planv1connect"
	"github.com/pboyd/todo/services/todo/internal/cli/timeparse"
)

func runPlan(args []string) int {
	apiKey := os.Getenv("TODO_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: TODO_API_KEY is not set")
		return 1
	}
	addr := os.Getenv("TODO_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	client := planv1connect.NewPlanServiceClient(
		&http.Client{},
		addr,
		connect.WithSendGzip(),
		connect.WithInterceptors(BearerInterceptor(apiKey)),
	)

	// Parse optional --date YYYY-MM-DD flag.
	day := time.Now().Format("2006-01-02")
	for len(args) >= 2 && args[0] == "--date" {
		day = args[1]
		args = args[2:]
	}

	if len(args) == 0 {
		return runPlanShow(client, day)
	}

	if args[0] == "--help" || args[0] == "-h" {
		printPlanUsage(os.Stdout)
		return 0
	}

	switch args[0] {
	case "task":
		return runPlanTask(client, day, args[1:])
	case "event":
		return runPlanEvent(client, day, args[1:])
	case "rm":
		return runPlanRm(client, day, args[1:])
	case "rename":
		return runPlanRename(client, day, args[1:])
	case "mv":
		return runPlanMv(client, day, args[1:])
	case "clear":
		return runPlanClear(client, day, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown plan subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'todo help plan' for usage.")
		return 1
	}
}

func printPlanUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: todo plan [--date YYYY-MM-DD] [subcommand]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  (none)                      Show today's plan as a grid")
	fmt.Fprintln(w, "  task <id> <start> [dur]     Schedule a task")
	fmt.Fprintln(w, "  event <name> <start> [dur]  Block off time")
	fmt.Fprintln(w, "  rm <n>                      Remove entry n")
	fmt.Fprintln(w, "  rename <n> <name>           Rename entry n")
	fmt.Fprintln(w, "  mv <n> <start> [dur]        Move entry n")
	fmt.Fprintln(w, "  clear [start]               Clear entries from start (default: now)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --date YYYY-MM-DD    Target a specific day (default: today)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Time formats:     13:15  1:15pm  01:15 PM")
	fmt.Fprintln(w, "Duration formats: 90m  1h  2h  1h30m")
}

func runPlanShow(client planv1connect.PlanServiceClient, day string) int {
	resp, err := client.ListPlanEntries(context.Background(), connect.NewRequest(&planv1.ListPlanEntriesRequest{Day: day}))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	width := 80
	if isTTY {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w >= 60 {
			width = w
		}
	}

	fmt.Print(RenderGrid(resp.Msg.Entries, day, time.Now(), width, isTTY))
	return 0
}

func runPlanTask(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: todo plan task <task_id> <start> [duration]")
		return 1
	}
	taskID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: task_id must be an integer")
		return 1
	}
	start, err := timeparse.ParseStart(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	var dur int
	if len(args) >= 3 {
		dur, err = timeparse.ParseDuration(args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}

	_, err = client.AddPlanTask(context.Background(), connect.NewRequest(&planv1.AddPlanTaskRequest{
		Day:            day,
		TaskId:         taskID,
		StartMinute:    int32(start),
		DurationMinute: int32(dur),
	}))
	if err != nil {
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func runPlanEvent(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: todo plan event <name> <start> [duration]")
		return 1
	}
	name := args[0]
	start, err := timeparse.ParseStart(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	var dur int
	if len(args) >= 3 {
		dur, err = timeparse.ParseDuration(args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}

	_, err = client.AddPlanEvent(context.Background(), connect.NewRequest(&planv1.AddPlanEventRequest{
		Day:            day,
		Name:           name,
		StartMinute:    int32(start),
		DurationMinute: int32(dur),
	}))
	if err != nil {
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func runPlanRm(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: todo plan rm <n>")
		return 1
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: entry id must be an integer")
		return 1
	}

	_, err = client.RemovePlanEntry(context.Background(), connect.NewRequest(&planv1.RemovePlanEntryRequest{
		Day: day, Id: int32(n),
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			fmt.Fprintln(os.Stderr, "entry not found")
			return 1
		}
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func runPlanRename(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: todo plan rename <n> <name>")
		return 1
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: entry id must be an integer")
		return 1
	}

	_, err = client.RenamePlanEntry(context.Background(), connect.NewRequest(&planv1.RenamePlanEntryRequest{
		Day: day, Id: int32(n), Name: args[1],
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeNotFound {
			fmt.Fprintln(os.Stderr, "entry not found")
			return 1
		}
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func runPlanMv(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: todo plan mv <n> <start> [duration]")
		return 1
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: entry id must be an integer")
		return 1
	}
	start, err := timeparse.ParseStart(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	var dur int
	if len(args) >= 3 {
		dur, err = timeparse.ParseDuration(args[2])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
	}

	_, err = client.MovePlanEntry(context.Background(), connect.NewRequest(&planv1.MovePlanEntryRequest{
		Day:            day,
		Id:             int32(n),
		StartMinute:    int32(start),
		DurationMinute: int32(dur),
	}))
	if err != nil {
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func runPlanClear(client planv1connect.PlanServiceClient, day string, args []string) int {
	var startMinute int
	if len(args) >= 1 {
		m, err := timeparse.ParseStart(args[0])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		startMinute = m
	} else {
		now := time.Now()
		startMinute = now.Hour()*60 + now.Minute()
	}

	_, err := client.ClearPlan(context.Background(), connect.NewRequest(&planv1.ClearPlanRequest{
		Day:         day,
		StartMinute: int32(startMinute),
	}))
	if err != nil {
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func printPlanError(err error) int {
	code := connect.CodeOf(err)
	switch code {
	case connect.CodeNotFound:
		fmt.Fprintln(os.Stderr, "not found:", err)
	case connect.CodeFailedPrecondition, connect.CodeInvalidArgument:
		fmt.Fprintln(os.Stderr, err)
	default:
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 1
}
