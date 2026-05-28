# Quickstart: Pomodoro Config File

## 1. Create the config file

```bash
mkdir -p ~/.config/todo
cat > ~/.config/todo/config.toml <<'EOF'
api_url = "http://localhost:8080"
api_key = "<paste the token from todo-server --provision-user>"

[pomodoro]
on_start    = "do-not-disturb on"
on_cancel   = "do-not-disturb off"
on_complete = "do-not-disturb off && notify-send 'Pomodoro done'"
EOF
```

## 2. Unset the env vars (optional)

```bash
unset TODO_API_KEY TODO_ADDR
```

If you keep them set, they win over the config file. That's intentional — existing scripts and `make dev` workflows continue to work.

## 3. Use the CLI normally

```bash
todo task list
todo pom start 42        # OnStart fires after the RPC succeeds
# ... 25 minutes pass ...
                         # OnComplete fires when the countdown hits zero
```

Or in the TUI:

```bash
todo                     # interactive TUI on a TTY
# Press the pomodoro key on a task → OnStart fires
# Press cancel during the countdown → OnCancel fires
```

## 4. Migrating from `--exec`

Old:
```bash
todo pom start 42 --exec 'notify-send "Done"'
```

New:
```toml
# ~/.config/todo/config.toml
[pomodoro]
on_complete = 'notify-send "Done"'
```

```bash
todo pom start 42        # no flag needed; same command runs every time
```

The `--exec` flag is removed in this release. Passing it now produces an unknown-flag error.

## 5. Verifying it works

```bash
# Quick smoke test using a marker file
cat > ~/.config/todo/config.toml <<'EOF'
api_url = "http://localhost:8080"
api_key = "$YOUR_KEY"

[pomodoro]
on_start    = "touch /tmp/pom-started"
on_cancel   = "touch /tmp/pom-canceled"
on_complete = "touch /tmp/pom-complete"
EOF

rm -f /tmp/pom-*
todo pom start <some_task_id>
# After RPC succeeds:
ls /tmp/pom-started      # exists
# Cancel with Ctrl-C in the cancel pathway (or `todo pom cancel`):
ls /tmp/pom-canceled     # exists; /tmp/pom-complete does NOT exist
```

## Troubleshooting

| Symptom                                                  | Cause                                                | Fix                                                                       |
|----------------------------------------------------------|------------------------------------------------------|---------------------------------------------------------------------------|
| `error: API key not set; set TODO_API_KEY ...`           | Neither env var nor config has `api_key`             | Add `api_key` to the config file or `export TODO_API_KEY=...`             |
| `config: failed to parse ...`                            | Malformed TOML                                       | Fix the line/column the error points to                                   |
| Hook didn't run                                          | Hook field is empty or command name not on `$PATH`   | Confirm spelling; check stderr for the warning                            |
| `unknown flag: --exec`                                   | Using the removed flag                               | Move the command to `[pomodoro].on_complete` in the config file           |
| Hook runs from CLI but not from TUI (or vice versa)      | Should not happen — file a bug                       | The CLI and TUI share one hook path                                       |
