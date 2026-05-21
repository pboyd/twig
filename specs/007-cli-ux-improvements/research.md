# Phase 0 Research: CLI UX Improvements

## R-001: Styling completed tasks (gray + strikethrough)

**Decision**: Emit raw ANSI escape sequences directly. Use `\x1b[2;9m` (dim + strikethrough) for the styled line and `\x1b[0m` to reset. Gate the styling behind a TTY check using `golang.org/x/term.IsTerminal(int(os.Stdout.Fd()))`.

**Rationale**:
- No third-party styling library is currently in use; adding one (lipgloss, fatih/color) would violate Principle I (simplicity / YAGNI).
- `golang.org/x/term` is already an indirect dependency (`go.mod` line 22), so promoting it to a direct import adds no module to `go.sum`.
- `\x1b[2m` (dim/faint) is the standard "gray" rendering and is honored by virtually every modern terminal; `\x1b[9m` (strikethrough) is supported in iTerm2, modern xterm, GNOME Terminal, and Windows Terminal.
- The implementation surface is one tiny helper: `styler(enabled bool)` returning a struct (or two free functions) that wrap a string with codes or pass it through unchanged.

**Alternatives considered**:
- `github.com/charmbracelet/lipgloss`: rich, but pulls in `termenv` + `muesli/ansi` + several others. Unjustified for ~10 lines of styling.
- `github.com/fatih/color`: simpler, but still a new direct dep, and does not natively support strikethrough.
- Detecting `NO_COLOR` env var: defer — not requested, and TTY check already covers the most common "piped output" case.

## R-002: `todo task mod` flag-after-positional parsing

**Decision**: Take `<id>` from `args[0]` *before* calling `fs.Parse`, then parse the remainder. Treat `fs.Args()` as the optional name (first positional after flags). Reject a second positional.

**Rationale**:
- Go's stdlib `flag` package stops parsing at the first non-flag token; that's why `mod 3 --parent 5` today binds `3` as the only positional and the flags are never seen — but wait: looking again at the current code, `fs.Parse(args)` *does* see `--parent` because flag parsing happens left-to-right, and `3` is the first non-flag, so parsing stops there and `--parent` is treated as part of `fs.Args()`. Hence the bug: `args[1]` (`--parent`) becomes the new "name". Splitting the id off the front before `fs.Parse` lets `flag` see the flags regardless of where they sit relative to the (now-absent) positional.
- This keeps the stdlib usage and avoids pulling in `pflag` / `cobra`.

**Algorithm**:
1. If `len(args) < 1` → usage error.
2. Parse `args[0]` as `id` (int64).
3. `fs.Parse(args[1:])`.
4. `rest := fs.Args()`. If `len(rest) == 0`: name is unchanged. If `len(rest) == 1`: name becomes `rest[0]` (reject empty string). If `len(rest) > 1`: usage error.
5. If no flags are set and `rest` is empty → "no changes specified" error.

**Alternatives considered**:
- `pflag` (POSIX-style): handles interleaved flags, but adds a dependency. Rejected.
- Manually scanning for `--` markers: error-prone for a tiny CLI. Rejected.

## R-003: `task` command default and removal of `list` subcommand

**Decision**: In `runTask`, when `args` is empty, call `runList(client, nil)` directly (after setting up the client). Remove the `case "list":` branch from the dispatch switch. Update `printTaskUsage` to drop the `list` line and document that `todo task [--completed | --all]` shows the list. Mirror `runPlan`'s pattern (`cli/plan.go:43-45`).

**Rationale**: Direct, minimal change. `runList` already accepts an arg slice and exposes `--completed`/`--all`; passing it `nil` works.

**Alternatives considered**: Keeping `list` as a hidden alias. Rejected — the user explicitly requested removal.

## R-004: Estimate rendering

**Decision**: Append ` (N)` to the task name when `task.GetEstimate() > 0`. Place it *before* the existing `(due …)` / `(completed …)` suffixes, so the line reads `[id] Name (N) (due …) (completed …)`.

**Rationale**: Per `proto/task/v1/task.proto:75-77`, `estimate` is an `int32` where `0` means "no estimate". The parens-after-name placement matches the user's request. Putting the estimate before the due/completed suffixes groups it with the task identity rather than its scheduling metadata.

**Alternatives considered**:
- After all suffixes: less prominent, harder to scan when planning.
- Inside the strikethrough style for completed tasks: yes — the whole line (after the `[id]` prefix) should be styled uniformly; the estimate participates in that styling.
