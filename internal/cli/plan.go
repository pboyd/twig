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

	planv1 "github.com/pboyd/twig/api/gen/plan/v1"
	planv1connect "github.com/pboyd/twig/api/gen/plan/v1/planv1connect"
	"github.com/pboyd/twig/internal/cli/timeparse"
	"github.com/pboyd/twig/internal/config"
)

func runPlan(profile string, args []string) int {
	cfg, err := loadConfig(profile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if cfg.APIKey == "" {
		path, _ := config.DefaultPath()
		fmt.Fprintf(os.Stderr, "no API key found — set TWIG_API_KEY or add api_key to %s\n", path)
		return 1
	}

	client := planv1connect.NewPlanServiceClient(
		&http.Client{},
		cfg.APIURL,
		connect.WithSendGzip(),
		connect.WithInterceptors(BearerInterceptor(cfg.APIKey)),
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
		fmt.Fprintln(os.Stderr, "Run 'twig help plan' for usage.")
		return 1
	}
}

func printPlanUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: twig plan [--date YYYY-MM-DD] [subcommand]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  (none)                              Show today's plan")
	fmt.Fprintln(w, "  task <id> [start|null] [dur|end]    Schedule a task (omit start or use 'null' to add untimed)")
	fmt.Fprintln(w, "  event <name> <start> [dur|end]      Block off time (start required)")
	fmt.Fprintln(w, "  rm <n>                              Remove entry n")
	fmt.Fprintln(w, "  rename <n> <name>                   Rename entry n")
	fmt.Fprintln(w, "  mv <n> [start|null] [dur|end]        Move entry n (omit start or 'null' to unschedule)")
	fmt.Fprintln(w, "  clear [start]                       Clear entries from start (default: now)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  --date YYYY-MM-DD    Target a specific day (default: today)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Time formats:     13:15  1315  1:15pm  01:15 PM")
	fmt.Fprintln(w, "Duration formats: 90m  1h  2h  1h30m")
	fmt.Fprintln(w, "End time:         use any time format in place of a duration")
	fmt.Fprintln(w, "Untimed sentinel: null (case-insensitive) in the start slot")
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

	var untimed, timed []*planv1.PlanEntry
	for _, e := range resp.Msg.Entries {
		if e.StartMinute == nil {
			untimed = append(untimed, e)
		} else {
			timed = append(timed, e)
		}
	}

	fmt.Print(RenderUntimed(untimed, width, isTTY, GridOptions{}))
	fmt.Print(RenderGrid(timed, day, time.Now(), width, isTTY, GridOptions{}))
	return 0
}

func runPlanTask(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: twig plan task <task_id> [start|null] [dur|end]")
		return 1
	}
	taskID, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: task_id must be an integer")
		return 1
	}

	req := &planv1.AddPlanTaskRequest{
		Day:    day,
		TaskId: taskID,
	}

	// Parse optional start argument.
	argIdx := 1
	if argIdx < len(args) {
		minute, isTimed, err := timeparse.ParseStartOrNull(args[argIdx])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		if isTimed {
			sm := int32(minute)
			req.StartMinute = &sm
			argIdx++
			// Parse optional duration/end.
			if argIdx < len(args) {
				dur, err := timeparse.ParseDurationOrEnd(args[argIdx], minute)
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					return 1
				}
				req.DurationMinute = int32(dur)
			}
		} else {
			// null sentinel: start is absent; trailing arg is duration.
			argIdx++
			if argIdx < len(args) {
				dur, err := timeparse.ParseDuration(args[argIdx])
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					return 1
				}
				req.DurationMinute = int32(dur)
			}
		}
	}

	_, err = client.AddPlanTask(context.Background(), connect.NewRequest(req))
	if err != nil {
		return printPlanError(err)
	}
	return runPlanShow(client, day)
}

func runPlanEvent(client planv1connect.PlanServiceClient, day string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: twig plan event <name> <start> [duration|end]")
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
		dur, err = timeparse.ParseDurationOrEnd(args[2], start)
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
		fmt.Fprintln(os.Stderr, "Usage: twig plan rm <n>")
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
		fmt.Fprintln(os.Stderr, "Usage: twig plan rename <n> <name>")
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
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: twig plan mv <n> [start|null] [duration|end]")
		return 1
	}
	n, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: entry id must be an integer")
		return 1
	}

	req := &planv1.MovePlanEntryRequest{Day: day, Id: int32(n)}

	argIdx := 1
	if argIdx < len(args) {
		minute, isTimed, err := timeparse.ParseStartOrNull(args[argIdx])
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			return 1
		}
		argIdx++
		if isTimed {
			sm := int32(minute)
			req.StartMinute = &sm
			if argIdx < len(args) {
				dur, err := timeparse.ParseDurationOrEnd(args[argIdx], minute)
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					return 1
				}
				req.DurationMinute = int32(dur)
			}
		} else {
			// null sentinel: unschedule; trailing arg is duration
			if argIdx < len(args) {
				dur, err := timeparse.ParseDuration(args[argIdx])
				if err != nil {
					fmt.Fprintln(os.Stderr, "error:", err)
					return 1
				}
				req.DurationMinute = int32(dur)
			}
		}
	}
	// If no start arg given: unschedule (StartMinute remains nil).

	_, err = client.MovePlanEntry(context.Background(), connect.NewRequest(req))
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
		fmt.Fprintln(os.Stderr, UserMessage(err))
	default:
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	return 1
}
