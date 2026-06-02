# Implementation Plan: CLI Task Uncomplete

**Branch**: `032-cli-task-uncomplete` | **Date**: 2026-06-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/032-cli-task-uncomplete/spec.md`

## Summary

Add `twig task uncomplete <id>` as a CLI subcommand that marks a completed task incomplete again, calling the existing `UncompleteTask` RPC. The implementation mirrors `runComplete` exactly in structure, argument handling, and error reporting.

## Technical Context

**Language/Version**: Go 1.23 (module at `services/twig/`)

**Primary Dependencies**: ConnectRPC (`connectrpc.com/connect`), protobuf-generated client in `gen/task/v1/taskv1connect`

**Storage**: N/A — state lives on the server; CLI is a thin client

**Testing**: `go test ./internal/cli/...` — table-driven unit tests with `fakeTaskService` HTTP test server

**Target Platform**: Linux/macOS CLI

**Project Type**: CLI client

**Performance Goals**: Standard CLI response latency (<2 s on local network)

**Constraints**: Must not introduce new dependencies or abstractions; must mirror `complete` command conventions

**Scale/Scope**: Single function addition; two minor edits to existing files

## Constitution Check

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Simplicity / YAGNI | ✅ | `runUncomplete` is a direct copy-adapt of `runComplete`; no new abstractions |
| II. API-First Design | ✅ | `UncompleteTask` RPC contract already exists and is stable — no schema changes needed |
| III. UI/UX Consistency | ✅ | Same argument pattern, exit codes, and output channel conventions as `complete` |
| IV. Playful User Messages | ✅ | Success: `"task %d is back on your list\n"` — warm without being silly; idempotent: `"task %d was already incomplete\n"` |

## Project Structure

### Documentation (this feature)

```text
specs/032-cli-task-uncomplete/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit-tasks)
```

### Source Code Changes

```text
services/twig/
├── internal/cli/
│   ├── cli.go           # Add case "uncomplete" dispatch + usage line
│   ├── task.go          # Add runUncomplete function
│   └── task_test.go     # Add UncompleteTask stub + 4 tests
```

**Structure Decision**: Three targeted edits to existing files. No new files, no new packages.

## Phase 0: Research

No unknowns requiring external research. All information is directly observable from the existing codebase:

- `UncompleteTask` RPC exists in `gen/task/v1/taskv1connect/task.connect.go` and accepts `UncompleteTaskRequest{Id: int64}`, returning `UncompleteTaskResponse{Task: *Task}`
- `runComplete` in `internal/cli/task.go:57` is the direct implementation model
- `fakeTaskService` in `task_test.go` uses `UnimplementedTaskServiceHandler` embedding; adding `UncompleteTask` method is sufficient to enable testing
- No playfulness gap in the existing `complete` messages — `uncomplete` messages are proposed as `"task %d is back on your list\n"` (success) and `"task %d was already incomplete\n"` (idempotent), satisfying Principle IV while matching the brevity of the existing complete messages

**Decisions**:

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Success message | `"task %d is back on your list\n"` | Warm (Principle IV), unambiguous, matches message length of complete |
| Idempotent message | `"task %d was already incomplete\n"` | Parallels "task %d already complete" pattern; warm but clear |
| No `before` timestamp check | Omitted | Complete uses it only to detect idempotency; for uncomplete, check `CompletedAt == nil` on response |

## Phase 1: Design & Contracts

### Contract

The `UncompleteTask` RPC contract is already committed to the repository (`gen/task/v1/task.pb.go`). No new contract file is required. The CLI simply invokes the existing endpoint.

For completeness, the interface this command exposes to the user:

**contracts/cli-uncomplete.md** — see below.

### data-model.md

No new entities. The `Task` proto message already carries `completed_at` (nullable timestamp); clearing it to `nil` is the server-side operation. No client-side data model changes.

### Implementation Details

#### `internal/cli/task.go` — add `runUncomplete`

```go
func runUncomplete(client taskv1connect.TaskServiceClient, addr string, args []string) int {
    if len(args) < 1 {
        fmt.Fprintln(os.Stderr, "usage: twig task uncomplete <id>")
        return 1
    }

    id, err := strconv.ParseInt(args[0], 10, 64)
    if err != nil {
        fmt.Fprintf(os.Stderr, "<id> must be an integer, got %q\n", args[0])
        return 1
    }

    resp, err := client.UncompleteTask(context.Background(), connect.NewRequest(&taskv1.UncompleteTaskRequest{Id: id}))
    if err != nil {
        fmt.Fprintln(os.Stderr, mapError(err, addr))
        return 1
    }

    task := resp.Msg.Task
    if task.CompletedAt != nil {
        fmt.Printf("task %d was already incomplete\n", id)
    } else {
        fmt.Printf("task %d is back on your list\n", id)
    }
    return 0
}
```

#### `internal/cli/cli.go` — dispatch and usage

In the task switch block, after `case "complete"`, add:
```go
case "uncomplete":
    return runUncomplete(client, addr, args[1:])
```

In `printTaskUsage`, after the `complete` line, add:
```go
fmt.Fprintln(w, "  uncomplete <id>  Mark a completed task incomplete again")
```

#### `internal/cli/task_test.go` — fake service stub

Add to `fakeTaskService`:
```go
func (s *fakeTaskService) UncompleteTask(_ context.Context, req *connect.Request[taskv1.UncompleteTaskRequest]) (*connect.Response[taskv1.UncompleteTaskResponse], error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    task, ok := s.tasks[req.Msg.Id]
    if !ok {
        return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("task not found"))
    }
    task.CompletedAt = nil
    return connect.NewResponse(&taskv1.UncompleteTaskResponse{Task: task}), nil
}
```

Add tests mirroring the Complete test block:

- `TestUncompleteLeaf` — complete a task then uncomplete it; expect exit 0 and "back on your list"
- `TestUncompleteIdempotent` — uncomplete an already-incomplete task; expect exit 0 and "already incomplete"
- `TestUncompleteMalformedID` — pass `"abc"`; expect exit 1 and "integer" on stderr
- `TestUncompleteNotFound` — pass `999`; expect exit 1 and non-empty stderr
