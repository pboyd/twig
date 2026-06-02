# CLI Contract: `twig task uncomplete`

## Synopsis

```
twig task uncomplete <id>
```

## Arguments

| Argument | Type    | Required | Description          |
|----------|---------|----------|----------------------|
| `<id>`   | integer | yes      | Numeric task id      |

## Exit Codes

| Code | Condition                                          |
|------|----------------------------------------------------|
| 0    | Task successfully marked incomplete, or already incomplete |
| 1    | Missing argument, non-integer id, task not found, or server error |

## Stdout

| Condition                        | Output                                  |
|----------------------------------|-----------------------------------------|
| Task was completed, now cleared  | `task <id> is back on your list`        |
| Task was already incomplete      | `task <id> was already incomplete`      |

## Stderr

| Condition             | Output                                            |
|-----------------------|---------------------------------------------------|
| Missing `<id>`        | `usage: twig task uncomplete <id>`                |
| Non-integer `<id>`    | `<id> must be an integer, got "<value>"`          |
| Task not found        | Human-readable error from server                  |
| Network / auth error  | Human-readable error from server                  |

## Underlying RPC

`task.v1.TaskService/UncompleteTask` — clears `completed_at` on the task.
