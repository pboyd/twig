package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"connectrpc.com/connect"
	goalv1 "github.com/pboyd/twig/api/gen/goal/v1"
	goalv1connect "github.com/pboyd/twig/api/gen/goal/v1/goalv1connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/config"
	"github.com/pboyd/twig/internal/goal"
)

func runGoal(profile string, args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printGoalUsage(os.Stdout)
		return 0
	}

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

	addr := cfg.APIURL
	httpClient := &http.Client{}
	opts := []connect.ClientOption{
		connect.WithSendGzip(),
		connect.WithInterceptors(BearerInterceptor(cfg.APIKey)),
	}
	goalClient := goalv1connect.NewGoalServiceClient(httpClient, addr, opts...)
	taskClient := taskv1connect.NewTaskServiceClient(httpClient, addr, opts...)

	if len(args) == 0 || (len(args) > 0 && strings.HasPrefix(args[0], "-")) {
		return runGoalList(goalClient, addr, args)
	}

	switch args[0] {
	case "add":
		return runGoalAdd(goalClient, addr, args[1:])
	case "show":
		return runGoalShow(goalClient, taskClient, addr, args[1:])
	case "mod":
		return runGoalMod(goalClient, addr, args[1:])
	case "state":
		return runGoalState(goalClient, addr, args[1:])
	case "rm":
		return runGoalRm(goalClient, addr, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'twig help goal' for usage.")
		return 1
	}
}

func printGoalUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: twig goal [<subcommand>] [arguments]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands (if omitted, lists goals):")
	fmt.Fprintln(w, "  add [--due <date>] [--desc <text>] <name>")
	fmt.Fprintln(w, "                   Add a new goal (starts incubating)")
	fmt.Fprintln(w, "  show <id>        Show goal details and associated tasks")
	fmt.Fprintln(w, "  mod <id> [<name>] [--due <date>] [--desc <text>]")
	fmt.Fprintln(w, "                   Modify a goal")
	fmt.Fprintln(w, "  state <id> <incubating|committed|completed|archived>")
	fmt.Fprintln(w, "                   Change goal state")
	fmt.Fprintln(w, "  rm <id>          Delete a goal (tasks survive)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Listing goals:")
	fmt.Fprintln(w, "  twig goal [--all]")
	fmt.Fprintln(w, "  --all            Show all goals (including completed and archived)")
}

func runGoalList(client goalv1connect.GoalServiceClient, addr string, args []string) int {
	fs := flag.NewFlagSet("goal", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var showAll bool
	fs.BoolVar(&showAll, "all", false, "show all goals")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	resp, err := client.ListGoals(context.Background(), connect.NewRequest(&goalv1.ListGoalsRequest{}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	goals := resp.Msg.Goals

	// Determine visible goals.
	var visible []*goalv1.Goal
	for _, g := range goals {
		if showAll || goal.DefaultVisible(g.State) {
			visible = append(visible, g)
		}
	}

	if len(visible) == 0 {
		if !showAll {
			fmt.Println("No goals on the horizon yet — plant one with 'twig goal add <name>'.")
		} else {
			fmt.Println("No goals yet — plant one with 'twig goal add <name>'.")
		}
		return 0
	}

	styled := WantStyled(os.Stdout)
	renderGoalGroups(os.Stdout, visible, styled)
	return 0
}

// renderGoalGroups prints goals grouped by state in display order.
func renderGoalGroups(w io.Writer, goals []*goalv1.Goal, styled bool) {
	order := goal.StateDisplayOrder()
	byState := make(map[goalv1.GoalState][]*goalv1.Goal)
	for _, g := range goals {
		byState[g.State] = append(byState[g.State], g)
	}

	first := true
	for _, state := range order {
		group := byState[state]
		if len(group) == 0 {
			continue
		}
		if !first {
			fmt.Fprintln(w)
		}
		first = false

		header := capitalize(goal.StateName(state))
		if styled {
			header = "\x1b[1m" + header + "\x1b[0m"
		}
		fmt.Fprintln(w, header)

		for _, g := range group {
			line := fmt.Sprintf("[%d] %s", g.Id, g.Name)
			if due := FormatDue(g.Due); due != "" {
				line += fmt.Sprintf(" (due %s)", due)
			}
			fmt.Fprintln(w, line)
		}
	}
}

func runGoalAdd(client goalv1connect.GoalServiceClient, addr string, args []string) int {
	fs := flag.NewFlagSet("goal add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dueStr string
	var desc string
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")
	fs.StringVar(&desc, "desc", "", "description")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig goal add [--due <date>] [--desc <text>] <name>")
		return 1
	}
	name := fs.Arg(0)
	if name == "" {
		fmt.Fprintln(os.Stderr, "twig: a goal needs a name — give it one!")
		return 1
	}

	req := &goalv1.CreateGoalRequest{
		Name:        name,
		Description: desc,
	}
	if dueStr != "" {
		ts, err := ParseDue(dueStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		req.Due = ts
	}

	resp, err := client.CreateGoal(context.Background(), connect.NewRequest(req))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	fmt.Printf("created goal %d\n", resp.Msg.Goal.Id)
	return 0
}

func runGoalShow(goalClient goalv1connect.GoalServiceClient, taskClient taskv1connect.TaskServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig goal show <id>")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	getResp, err := goalClient.GetGoal(context.Background(), connect.NewRequest(&goalv1.GetGoalRequest{Id: id}))
	if err != nil {
		if isNotFound(err) {
			fmt.Fprintf(os.Stderr, "twig: goal %d isn't in my book — 'twig goal' lists what is.\n", id)
			return 1
		}
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	g := getResp.Msg.Goal

	fmt.Printf("Name:  %s\n", g.Name)
	fmt.Printf("State: %s\n", goal.StateName(g.State))
	if due := FormatDue(g.Due); due != "" {
		fmt.Printf("Due:   %s\n", due)
	}
	if g.Description != "" {
		fmt.Printf("Desc:  %s\n", g.Description)
	}

	// Fetch tasks and show associated subtrees.
	taskResp, err := taskClient.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	subtaskList := goal.SubtreeForGoal(taskResp.Msg.Tasks, id)
	fmt.Println()
	fmt.Println("Tasks:")
	if len(subtaskList) == 0 {
		fmt.Println("  No tasks attached yet — every great goal starts as a wish.")
		return 0
	}

	roots := BuildTree(subtaskList)
	styled := WantStyled(os.Stdout)
	renderRoots(os.Stdout, roots, styled)
	return 0
}

func runGoalMod(client goalv1connect.GoalServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig goal mod <id> [<name>] [--due <date>] [--desc <text>]")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	fs := flag.NewFlagSet("goal mod", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var dueStr string
	var desc string
	fs.StringVar(&dueStr, "due", "", "due date (RFC 3339 or YYYY-MM-DD)")
	fs.StringVar(&desc, "desc", "", "description")

	var flagTokens []string
	var positionals []string
	for i := 1; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "-") {
			flagTokens = append(flagTokens, a)
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
		fmt.Fprintln(os.Stderr, "usage: twig goal mod <id> [<name>] [--due <date>] [--desc <text>]")
		return 1
	}

	// Fetch current state to preserve unflagged fields.
	getResp, err := client.GetGoal(context.Background(), connect.NewRequest(&goalv1.GetGoalRequest{Id: id}))
	if err != nil {
		if isNotFound(err) {
			fmt.Fprintf(os.Stderr, "twig: goal %d isn't in my book — 'twig goal' lists what is.\n", id)
			return 1
		}
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	existing := getResp.Msg.Goal

	name := existing.Name
	if len(positionals) == 1 {
		if positionals[0] == "" {
			fmt.Fprintln(os.Stderr, "twig: a goal needs a name — give it one!")
			return 1
		}
		name = positionals[0]
	}

	description := existing.Description
	if desc != "" {
		description = desc
	}

	req := &goalv1.UpdateGoalRequest{
		Id:          id,
		Name:        name,
		Description: description,
		Due:         existing.Due,
	}

	if dueStr != "" {
		ts, err := ParseDue(dueStr)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return 1
		}
		req.Due = ts
	}

	if len(positionals) == 0 && dueStr == "" && desc == "" {
		fmt.Fprintln(os.Stderr, "nothing to update")
		return 1
	}

	_, err = client.UpdateGoal(context.Background(), connect.NewRequest(req))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	fmt.Printf("updated goal %d\n", id)
	return 0
}

func runGoalState(client goalv1connect.GoalServiceClient, addr string, args []string) int {
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: twig goal state <id> <incubating|committed|completed|archived>")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	state, err := goal.ParseState(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "twig: %q isn't a valid state — try one of: incubating, committed, completed, archived\n", args[1])
		return 1
	}

	_, err = client.SetGoalState(context.Background(), connect.NewRequest(&goalv1.SetGoalStateRequest{
		Id:    id,
		State: state,
	}))
	if err != nil {
		if isNotFound(err) {
			fmt.Fprintf(os.Stderr, "twig: goal %d isn't in my book — 'twig goal' lists what is.\n", id)
			return 1
		}
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	fmt.Printf("goal %d is now %s\n", id, goal.StateName(state))
	return 0
}

func runGoalRm(client goalv1connect.GoalServiceClient, addr string, args []string) int {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: twig goal rm <id>")
		return 1
	}

	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
		return 1
	}

	_, err = client.DeleteGoal(context.Background(), connect.NewRequest(&goalv1.DeleteGoalRequest{Id: id}))
	if err != nil {
		if isNotFound(err) {
			fmt.Fprintf(os.Stderr, "twig: goal %d isn't in my book — 'twig goal' lists what is.\n", id)
			return 1
		}
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}
	fmt.Printf("deleted goal %d\n", id)
	return 0
}

// isNotFound returns true when err is a ConnectRPC NotFound error.
func isNotFound(err error) bool {
	ce, ok := err.(*connect.Error)
	return ok && ce.Code() == connect.CodeNotFound
}

// capitalize returns s with the first rune uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
