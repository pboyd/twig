package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	taskv1 "github.com/pboyd/twig/api/gen/task/v1"
	taskv1connect "github.com/pboyd/twig/api/gen/task/v1/taskv1connect"
	"github.com/pboyd/twig/internal/config"
	"github.com/pboyd/twig/internal/report"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func runReport(profile string, args []string) int {
	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		printReportUsage(os.Stdout)
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

	client := taskv1connect.NewTaskServiceClient(
		&http.Client{},
		cfg.APIURL,
		connect.WithSendGzip(),
		connect.WithInterceptors(BearerInterceptor(cfg.APIKey)),
	)

	p, exitCode := parseReportArgs(args)
	if exitCode != 0 {
		return exitCode
	}

	return doReport(client, cfg.APIURL, p, os.Stdout)
}

// parseReportArgs parses the report command arguments.
// Returns the resolved Period and an exit code (0 = ok, 1 = error).
func parseReportArgs(args []string) (report.Period, int) {
	var preset string
	var fromStr, toStr string

	now := time.Now()

	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--from":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "oops — --from needs a date like --from 2025-06-01")
				return report.Period{}, 1
			}
			fromStr = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--from="):
			fromStr = strings.TrimPrefix(args[i], "--from=")
		case args[i] == "--to":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "oops — --to needs a date like --to 2025-06-30")
				return report.Period{}, 1
			}
			toStr = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--to="):
			toStr = strings.TrimPrefix(args[i], "--to=")
		case !strings.HasPrefix(args[i], "-"):
			if preset != "" {
				fmt.Fprintf(os.Stderr, "too many presets — pick just one (got %q and %q)\n", preset, args[i])
				fmt.Fprintln(os.Stderr, "Try: twig report week  or  twig report --from 2025-06-01 --to 2025-06-30")
				return report.Period{}, 1
			}
			preset = args[i]
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
			fmt.Fprintln(os.Stderr, "Run 'twig help report' for usage.")
			return report.Period{}, 1
		}
	}

	p, err := report.ParsePeriod(preset, fromStr, toStr, now)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return report.Period{}, 1
	}
	return p, 0
}

// doReport fetches data and renders the report to w.
func doReport(client taskv1connect.TaskServiceClient, addr string, p report.Period, w io.Writer) int {
	tasksResp, err := client.ListTasks(context.Background(), connect.NewRequest(&taskv1.ListTasksRequest{}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	pomResp, err := client.CountCompletedPomodoros(context.Background(), connect.NewRequest(&taskv1.CountCompletedPomodorosRequest{
		Start: timestamppb.New(p.StartUTC()),
		End:   timestamppb.New(p.EndUTC()),
	}))
	if err != nil {
		fmt.Fprintln(os.Stderr, mapError(err, addr))
		return 1
	}

	tasks := tasksResp.Msg.Tasks
	entries := report.BuildEntries(tasks, p)
	totals := report.Totals{
		TasksCompleted:     len(entries),
		PomodorosCompleted: pomResp.Msg.Count,
	}

	styled := WantStyled(w)
	renderReport(w, p, entries, tasks, totals, styled)
	return 0
}

// renderReport writes the report to w.
func renderReport(w io.Writer, p report.Period, entries []report.Entry, tasks []*taskv1.Task, totals report.Totals, styled bool) {
	header := fmt.Sprintf("What you got done — %s", p.Label)
	if styled {
		header = "\x1b[1;34m" + header + "\x1b[0m"
	}
	fmt.Fprintln(w, header)
	fmt.Fprintln(w)

	switch p.ReportLayout() {
	case report.AccomplishmentGrouped:
		renderReportAccomplishment(w, p, entries, tasks, totals, styled)
	default:
		renderReportDayGrouped(w, entries, totals, p, styled)
	}
}

func renderReportDayGrouped(w io.Writer, entries []report.Entry, totals report.Totals, p report.Period, styled bool) {
	groups := report.GroupByDay(entries)
	if len(groups) == 0 {
		fmt.Fprintln(w, renderReportEmptyState(p, styled))
		return
	}

	for _, g := range groups {
		dayLabel := g.Day.Format("Mon, Jan 2")
		if styled {
			dayLabel = "\x1b[1;34m" + dayLabel + "\x1b[0m"
		}
		fmt.Fprintln(w, dayLabel)
		for _, e := range g.Entries {
			line := "  ✓ " + e.Name
			if e.ParentName != "" {
				ctx := fmt.Sprintf("(under: %s)", e.ParentName)
				if styled {
					ctx = "\x1b[2m" + ctx + "\x1b[0m"
				}
				line += "  " + ctx
			}
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, renderTotalsLine(totals, styled))
}

func renderReportAccomplishment(w io.Writer, p report.Period, entries []report.Entry, tasks []*taskv1.Task, totals report.Totals, styled bool) {
	fmt.Fprintln(w, renderTotalsLine(totals, styled))
	fmt.Fprintln(w)

	finished, ongoing := report.GroupByAccomplishment(entries, tasks, p)

	if len(finished) == 0 && len(ongoing) == 0 {
		fmt.Fprintln(w, renderReportEmptyState(p, styled))
		return
	}

	check := "✓"

	if len(finished) > 0 {
		sectionHeader := "Finished"
		if styled {
			sectionHeader = "\x1b[1;34m" + sectionHeader + "\x1b[0m"
		}
		fmt.Fprintln(w, sectionHeader)
		for _, g := range finished {
			done := fmt.Sprintf("(done %s)", g.TopLevelCompletedAt.Format("Jan 2"))
			if styled {
				done = "\x1b[2m" + done + "\x1b[0m"
			}
			fmt.Fprintf(w, "  %s %s  %s\n", check, g.TopLevelName, done)
			for _, e := range g.Entries {
				if e.Depth == 0 {
					continue
				}
				indent := 6 + (e.Depth-1)*4
				if indent < 6 {
					indent = 6
				}
				done := fmt.Sprintf("(done %s)", e.CompletedAt.Format("Jan 2"))
				if styled {
					done = "\x1b[2m" + done + "\x1b[0m"
				}
				fmt.Fprintf(w, "%s%s %s  %s\n", strings.Repeat(" ", indent), check, e.Name, done)
			}
		}
		fmt.Fprintln(w)
	}

	if len(ongoing) > 0 {
		sectionHeader := "Progress on ongoing work"
		if styled {
			sectionHeader = "\x1b[1;34m" + sectionHeader + "\x1b[0m"
		}
		fmt.Fprintln(w, sectionHeader)
		for _, g := range ongoing {
			fmt.Fprintf(w, "  … %s\n", g.TopLevelName)
			for _, e := range g.Entries {
				indent := 6 + (e.Depth-1)*4
				if indent < 6 {
					indent = 6
				}
				done := fmt.Sprintf("(done %s)", e.CompletedAt.Format("Jan 2"))
				if styled {
					done = "\x1b[2m" + done + "\x1b[0m"
				}
				fmt.Fprintf(w, "%s%s %s  %s\n", strings.Repeat(" ", indent), check, e.Name, done)
			}
		}
	}
}

func renderTotalsLine(totals report.Totals, styled bool) string {
	tasks := fmt.Sprintf("%d task", totals.TasksCompleted)
	if totals.TasksCompleted != 1 {
		tasks += "s"
	}
	poms := fmt.Sprintf("%d pomodoro", totals.PomodorosCompleted)
	if totals.PomodorosCompleted != 1 {
		poms += "s"
	}
	line := fmt.Sprintf("%s done · %s burned", tasks, poms)
	if styled {
		return "\x1b[2m" + line + "\x1b[0m"
	}
	return line
}

func renderReportEmptyState(p report.Period, styled bool) string {
	from := p.From.Format("Jan 2")
	to := p.To.Format("Jan 2")
	var msg string
	if from == to {
		msg = fmt.Sprintf("Nothing checked off on %s — a blank page, full of potential.", from)
	} else {
		msg = fmt.Sprintf("Nothing checked off between %s and %s — a blank page, full of potential.", from, to)
	}
	if styled {
		return "\x1b[2m" + msg + "\x1b[0m"
	}
	return msg
}

func printReportUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage: twig report [<preset>]")
	fmt.Fprintln(w, "       twig report --from <YYYY-MM-DD> --to <YYYY-MM-DD>")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Presets (default: recent):")
	fmt.Fprintln(w, "  recent      Yesterday and today (default)")
	fmt.Fprintln(w, "  today       Today only")
	fmt.Fprintln(w, "  yesterday   Yesterday only")
	fmt.Fprintln(w, "  week        This week (Monday through today)")
	fmt.Fprintln(w, "  last-week   Last Monday through Sunday")
	fmt.Fprintln(w, "  month       This month (1st through today)")
	fmt.Fprintln(w, "  quarter     This quarter (1st day through today)")
	fmt.Fprintln(w, "  year        This year (Jan 1 through today)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Explicit range:")
	fmt.Fprintln(w, "  --from <YYYY-MM-DD> --to <YYYY-MM-DD>")
	fmt.Fprintln(w, "  (Both flags required together; mutually exclusive with a preset)")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Output is plain text when piped; styled with colors when on a terminal.")
	fmt.Fprintln(w, "Exit code 0 on success (including an empty period); 1 on error.")
}
