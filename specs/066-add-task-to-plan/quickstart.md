# Quickstart: Add New Tasks to a Plan

**Feature**: 066-add-task-to-plan | **Branch**: `066-add-task-to-plan`

How to build, run, and actually see this feature work. No server or database work is involved — but
you still need the stack running, because both surfaces talk to it.

## 1. Bring up the stack

```bash
make dev     # podman-compose: postgres + server, migrations run on startup
```

The server is on `:8080`. If you have no user yet:

```bash
podman exec -it <server-container> ./twig-server --provision-user you:yourpassword
# prints an API key
```

```bash
export TWIG_API_KEY=<key-from-provisioning>
export TWIG_ADDR=http://localhost:8080
```

## 2. TUI

```bash
go build -o twig ./cmd/twig
./twig                       # no args on a TTY → TUI, opens on the Tasks tab
```

**See the feature**:

1. Press `n` to create a task. Type a name.
2. `tab` down to the **Plan** field (between Snooze and Goal).
3. `→` cycles `No plan → Today → Tomorrow`. Stop on **Today**.
4. Save. The task appears in the tree.
5. `tab` to the **Plan** tab — the task is there as an untimed entry.

**The other-day path**: on the Plan field press `ctrl+g` for the calendar, pick a date, save, then
check that day on the Plan tab.

**The default**: create a task without touching the Plan field. It must appear nowhere on any plan.
This is the behavior SC-003/SC-005 protect — check it every time you touch this code.

## 3. Web

```bash
cd services/twig-web
npm install                  # first time only
npm run dev                  # → http://localhost:5173
```

Log in through the SPA. **Never** hit `:8080` cross-origin — the dev proxy keeps the SPA and API on
one origin so the `SameSite=Strict` session cookie works.

**See the feature**:

1. Open the add-task form.
2. The **Add to plan** row reads **No plan**.
3. Click **Today**, submit. A toast confirms; the task is on the tree.
4. Go to the plan view — the task is there, untimed.
5. Repeat with the 📅 affordance and a date a week out.

Also check the sub-task form (expand a task, add a child) — it has the control too. And check the
**edit** form on a task's detail page — it must **not** have it (FR-011).

## 4. Tests

```bash
# TUI (from repo root)
go test ./internal/tui/...

# Web
cd services/twig-web && npm test
```

Contract tests CT-01..CT-14 are enumerated in `contracts/ui-contract.md` §5. Each maps to a
requirement; a red one names the requirement it breaks.

## 5. Verifying the two failure paths

These are the parts most easily gotten wrong, and they are invisible in the happy path.

**Task creation fails ⇒ nothing is planned** (FR-006). Stop the server, then create a task with
**Today** selected. You should be told the task wasn't created — and when the server returns, no
stray plan entry should exist.

**Plan write fails ⇒ the task survives** (FR-007). Harder to stage by hand; the honest check is
CT-09, which forces `AddPlanTask` to fail and asserts the task remains and the message says so. The
rule this protects: *a failed convenience must never destroy the user's real work.* If you find
yourself writing a rollback that deletes the created task, re-read `data-model.md` § "Write sequence
and failure modes" — that path was rejected deliberately.

**Duplicate** (`FAILED_PRECONDITION`): add a task to today's plan, then from the task list add the
same task to today again. It should say it's already waiting there — not present as a hard error.

## 6. Where things live

| What | Where |
|---|---|
| TUI form state, cycling, render | `internal/tui/edit.go` |
| TUI save + RPC chain | `internal/tui/update.go` (`createTaskCmd`) |
| TUI calendar binding | `internal/tui/keymap.go` (`keys.Calendar` = `ctrl+g`) |
| Web control | `services/twig-web/src/components/TaskForm.tsx` |
| Web day helpers | `services/twig-web/src/lib/planDays.ts` |
| Web copy (all user-facing text) | `services/twig-web/src/theme/messages.ts` |
| The RPC being reused | `api/proto/plan/v1/plan.proto` — `AddPlanTask` (**do not modify**) |

## 7. Things that will bite you

- **`AddPlanTask` field values are pinned by contract**: `start_minute` **omitted** (untimed) and
  `duration_minute: 0` (server default). Sending a start minute invents a time the user never chose.
  See `contracts/rpc-usage.md`.
- **Adding `focusPlan` renumbers the focus constants** after it. They're referenced symbolically, so
  this should be safe — but grep for bare numeric literals before landing.
- **Today/Tomorrow resolve at save, not at open** (FR-005). Use the form's existing `nowFunc` hook,
  which is what makes CT-06's midnight test possible.
- **New user-facing text needs the house tone** (Constitution IV) and lives in `messages.ts` on the
  web. Control labels stay literal, though — "No plan"/"Today"/"Tomorrow". Playfulness belongs in
  messages, not in the labels of a control the user has to read quickly.
