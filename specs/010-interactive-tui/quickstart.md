# Quickstart: Trying the Interactive TUI locally

## Prerequisites

- The dev stack (server + postgres) is running:
  ```bash
  make dev
  ```
- A user has been provisioned and you have their API key:
  ```bash
  podman exec -it <server-container> ./todo-server --provision-user me:hunter2
  # copy the printed API key
  ```
- Environment variables exported:
  ```bash
  export TODO_API_KEY=<the api key>
  export TODO_ADDR=http://localhost:8080
  ```

## Build

```bash
cd services/todo
go build -o todo ./cmd/todo
```

## Launch the TUI

```bash
./todo
```

With no subcommand and a TTY, you land in the TUI: incomplete tasks on the left, details on the right.

## Smoke walkthrough (matches user stories)

1. Press `↓`/`↑` to move the highlight. Confirm the details pane updates each time.
2. On a task with subtasks, press `→` (or `L`). Confirm the subtree expands and the marker flips from `[+]` to `[-]`.
3. Press `←` (or `H`). Subtree collapses; marker back to `[+]`.
4. Press `?`. The help overlay lists every key. Press `?` or `Esc` to dismiss.
5. Press `E` on the highlighted task, edit the name, then `Ctrl-S`. Focus returns to the list; the name update is visible.
6. Press `N`. The form opens blank for a new subtask. Type a name, press `Ctrl-S`. The parent's subtree is now expanded and the new subtask is highlighted.
7. Press `Ctrl-N`. Enter a name, save. The new task appears at the root and is highlighted.
8. Press `Space`. Task gets a strikethrough/dim. Move the cursor; the just-completed task disappears (with the default incomplete-only filter).
9. Press `C`. Completed tasks reappear (struck through and dim). Press `C` again to hide them.
10. Press `Ctrl-D` on a throwaway task. It vanishes immediately; the highlight moves to a neighbor.
11. Press `S` on a task. The TUI yields to the existing pomodoro countdown. When it ends, you land back in the list.
12. From a separate shell, run `todo pom start --background <task-id>` to background a pomodoro. Back in the TUI, press `R` to resume it.
13. Press a digit `0`–`9`. The highlighted task's pomodoro estimate updates accordingly.
14. Press `Ctrl-R`. The full tree reloads from the server (handy if you made changes from another shell).
15. Press `Q` to exit.

## Non-TTY fallback

```bash
./todo | cat
```

Prints the existing root-usage banner and exits — does not attempt a TUI.

## Running the unit tests

```bash
cd services/todo
go test ./internal/tui/...
```

Tests cover: visible-row flattening, marker computation, cursor preservation across mutations and filter toggles, edit-form lifecycle, and message dispatch. They do not require a running server (the ConnectRPC clients are swapped for fakes via `export_test.go`).
