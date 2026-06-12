package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/config"
)

const defaultAddr = "http://localhost:8080"

// ResolveProfileName determines the effective profile name.
// flagValue is the value from --profile (if present); flagSet indicates
// whether the flag was provided. When the flag is absent, TWIG_PROFILE is
// consulted. An empty value from either source means the default/root profile.
// Note: env vars TWIG_ADDR/TWIG_API_KEY override credentials after profile
// selection via Resolve(), guaranteeing env wins for credentials.
func ResolveProfileName(flagValue string, flagSet bool) string {
	if flagSet {
		return flagValue
	}
	return os.Getenv("TWIG_PROFILE")
}

// ExtractProfileFlag scans the leading args for --profile <name> or
// --profile=<name>, strips it, and returns the name, whether the flag was
// present, and the remaining args.
// Returns an error if --profile is present but has no value.
func ExtractProfileFlag(args []string) (name string, set bool, rest []string, err error) {
	if len(args) == 0 {
		return "", false, args, nil
	}
	if strings.HasPrefix(args[0], "--profile=") {
		val := strings.TrimPrefix(args[0], "--profile=")
		if val == "" {
			return "", false, nil, fmt.Errorf("twig: --profile needs a profile name (try: twig --profile home …)")
		}
		return val, true, args[1:], nil
	}
	if args[0] == "--profile" {
		if len(args) < 2 {
			return "", false, nil, fmt.Errorf("twig: --profile needs a profile name (try: twig --profile home …)")
		}
		return args[1], true, args[2:], nil
	}
	return "", false, args, nil
}

func loadConfig(profile string) (config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return config.Config{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return config.Config{}, err
	}
	selected, ok := cfg.Profile(profile)
	if !ok {
		return config.Config{}, fmt.Errorf("twig: hmm, I couldn't find a profile named %q — check the spelling, or add a [profile.%s] section to %s", profile, profile, path)
	}
	return selected.Resolve(), nil
}

// Run is the entrypoint for the CLI. It returns the process exit code.
func Run(profile string, args []string) int {
	if len(args) == 0 {
		printRootUsage(os.Stderr)
		return 1
	}

	if args[0] == "--help" || args[0] == "-h" {
		return runHelp(nil)
	}

	switch args[0] {
	case "goal":
		return runGoal(profile, args[1:])
	case "task":
		return runTask(profile, args[1:])
	case "pom":
		return runPomTop(profile, args[1:])
	case "plan":
		return runPlan(profile, args[1:])
	case "report":
		return runReport(profile, args[1:])
	case "help":
		return runHelp(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'twig help' for usage.")
		return 1
	}
}

func runHelp(args []string) int {
	if len(args) == 0 {
		printRootUsage(os.Stdout)
		return 0
	}
	switch args[0] {
	case "goal":
		printGoalUsage(os.Stdout)
		return 0
	case "task":
		printTaskUsage(os.Stdout)
		return 0
	case "pom":
		printPomUsage(os.Stdout)
		return 0
	case "plan":
		printPlanUsage(os.Stdout)
		return 0
	case "report":
		printReportUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\nRun 'twig help' for usage.\n", args[0])
		return 1
	}
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: twig [--profile <name>] <command> [arguments]")
	fmt.Fprintln(w, "       twig [--profile <name>]")
	fmt.Fprintln(w, "       twig help [command]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Global options:")
	fmt.Fprintln(w, "  --profile <name>   Use a named account profile (or set TWIG_PROFILE)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  goal     Long-term goals (incubate, commit, complete)")
	fmt.Fprintln(w, "  task     Manage tasks (add, remove, modify, list)")
	fmt.Fprintln(w, "  pom      Pomodoro timer (estimate, start, resume, cancel, status)")
	fmt.Fprintln(w, "  plan     Daily planning (schedule tasks and events)")
	fmt.Fprintln(w, "  report   Activity report (what did I get done?)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Run 'twig help <command>' for command-specific help.")
	fmt.Fprintln(w, "Config file: ~/.config/twig/config.toml (see specs/033-alternate-profiles/contracts/config-schema.md)")
}

func runTask(profile string, args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printTaskUsage(os.Stdout)
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
	client := taskv1connect.NewTaskServiceClient(
		&http.Client{},
		addr,
		connect.WithSendGzip(),
		connect.WithInterceptors(BearerInterceptor(cfg.APIKey)),
	)

	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		return runList(client, addr, args)
	}

	switch args[0] {
	case "add":
		return runAdd(client, addr, args[1:])
	case "rm":
		return runRm(client, addr, args[1:])
	case "mod":
		return runMod(client, addr, args[1:])
	case "complete":
		return runComplete(client, addr, args[1:])
	case "uncomplete":
		return runUncomplete(client, addr, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run 'twig help task' for usage.")
		return 1
	}
}

func printTaskUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: twig task [<subcommand>] [arguments]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Subcommands (if omitted, lists tasks):")
	fmt.Fprintln(w, "  add [--parent <id>] [--due <date>] <name>")
	fmt.Fprintln(w, "                   Add a new task")
	fmt.Fprintln(w, "  rm <id>          Remove a task")
	fmt.Fprintln(w, "  mod <id> [<name>] [--parent <id>] [--due <date>]")
	fmt.Fprintln(w, "                   Modify a task")
	fmt.Fprintln(w, "  complete <id>    Mark a task complete")
	fmt.Fprintln(w, "  uncomplete <id>  Mark a completed task incomplete again")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Listing tasks:")
	fmt.Fprintln(w, "  twig task [--completed | --all]")
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
