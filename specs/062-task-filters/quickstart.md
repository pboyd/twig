# Quickstart: Task Filters

**Feature**: 062-task-filters

## Try it in the TUI

```bash
# 1. Start the stack and provision (once)
make dev
# TWIG_API_KEY / TWIG_ADDR configured per CLAUDE.md

# 2. Build and launch the TUI
go build -o twig ./cmd/twig && ./twig
```

On the **Tasks** tab:

| Key | Action |
|-----|--------|
| `/` | Open the filter bar (pre-filled if a filter is already applied) |
| type | Results narrow with every keystroke |
| `Enter` | Apply the filter, focus returns to the list (filter stays visible) |
| `Esc` (in filter bar) | Clear the filter, back to the full list |
| `Esc` (in list, filter applied) | Clear the filter |
| `c` | Toggle "show all" — the active filter re-evaluates under the new default |

Expressions to try:

```text
groceries
completed=true AND parent_id=1
completed=false AND ^parent_id=1
snoozed=true
completed < 2026-01-01
completed=false AND snoozed=false AND ^goal_id=1
groceries AND ^goal_id=2
"AND review"
```

While an expression is half-typed or malformed the list keeps its last valid results and the bar shows an invalid indicator — it should never blank or crash (mash keys to verify).

## Exercise the server directly (SC-005)

ConnectRPC accepts plain JSON over HTTP POST:

```bash
curl -s http://localhost:8080/task.v1.TaskService/FilterTasks \
  -H "Authorization: Bearer $TWIG_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"expression":"completed=false AND ^goal_id=1","showAll":false,"today":"2026-07-10"}'
# → {"taskIds":["3","7","9"]}
```

An invalid expression returns HTTP 400 with `code: invalid_argument` and a readable message.

## Run the tests

```bash
go test ./...                        # root module: TUI filter-mode behavior
cd services/twig && go test ./...    # filter package (grammar + semantics) + handler
```

The `internal/filter` tests are table-driven off the worked-examples table in `contracts/filter-grammar.md` — every row there must have a corresponding passing case.

## Suggested user-facing copy (Principle IV — finalize at implementation)

- Empty filter result: `Nothing matches that filter — even the twigs came up bare.`
- Invalid expression indicator (short, inline): `hmm, that's not quite a filter yet…`
- Filter bar placeholder: `search, or try completed=false AND ^goal_id=1`
