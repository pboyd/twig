# Contract: Plan day objective and notes (`plan.v1`)

**Feature**: 074-plan-objective-notes | **Date**: 2026-08-06

Additive changes to `api/proto/plan/v1/plan.proto`. Regenerate with `make proto`.
Nothing here removes or renumbers an existing field, so current clients keep working
unchanged.

## Messages

```proto
// PlanDay carries the day-scoped fields a user can set on a plan day. It exists
// independently of whether the day has any entries. Empty string means "not set" for
// both text fields; there is no separate absent state.
message PlanDay {
  // The day these values belong to, in YYYY-MM-DD.
  string day = 1;
  // Short markdown reminder of the main thing to get done that day. "" when unset.
  string objective = 2;
  // Long-form markdown notes about the day. "" when unset.
  string notes = 3;
}
```

## Changed message

```proto
message ListPlanEntriesResponse {
  repeated PlanEntry entries = 1;
  // Always populated. day.day echoes the request; objective and notes are "" when unset.
  PlanDay day = 2;
}
```

`day` is never null on a successful response, including for a day with no entries and no
stored row. Clients read the objective and notes from the call they already make to load a
day, so no extra round trip is needed.

## New RPCs

```proto
service PlanService {
  // ... existing RPCs unchanged ...

  // SetPlanObjective replaces the day's objective. An empty or whitespace-only
  // objective clears it. Leaves notes untouched.
  rpc SetPlanObjective(SetPlanObjectiveRequest) returns (SetPlanObjectiveResponse);

  // SetPlanNotes replaces the day's notes. Empty or whitespace-only notes clear them.
  // Leaves the objective untouched.
  rpc SetPlanNotes(SetPlanNotesRequest) returns (SetPlanNotesResponse);
}

message SetPlanObjectiveRequest {
  string day = 1;
  // Markdown source. Trimmed of surrounding whitespace by the server. At most 255
  // runes after trimming. Empty (or whitespace-only) clears the objective.
  string objective = 2;
}
message SetPlanObjectiveResponse { PlanDay day = 1; }

message SetPlanNotesRequest {
  string day = 1;
  // Markdown source. Trimmed of surrounding whitespace by the server. No length limit.
  // Empty (or whitespace-only) clears the notes.
  string notes = 2;
}
message SetPlanNotesResponse { PlanDay day = 1; }
```

## Behaviour

### `ListPlanEntries`

| Given | Then |
|---|---|
| Day has a stored row | `day` carries the stored `objective` and `notes` verbatim |
| Day has no stored row | `day` carries `{day: <requested>, objective: "", notes: ""}` |
| Day has entries but no row, or a row but no entries | Both parts of the response are independent; neither implies the other |
| `day` is not `YYYY-MM-DD` | `INVALID_ARGUMENT` (unchanged from today) |

Entry ordering and content are unaffected.

### `SetPlanObjective`

| Given | Then |
|---|---|
| Valid day, objective ≤ 255 runes after trimming | Stored trimmed; response carries the new `PlanDay` including untouched `notes` |
| Objective is `""` or whitespace only | Stored as `""`; the day reads as having no objective |
| Objective exceeds 255 runes after trimming | `INVALID_ARGUMENT`; nothing is written |
| Day is not `YYYY-MM-DD` | `INVALID_ARGUMENT`; nothing is written |
| No row exists for the day | Row is created; `notes` defaults to `""` |
| Row exists | Only `objective` is written; `notes` is preserved |
| Caller is unauthenticated | `UNAUTHENTICATED` (via the existing `auth.Middleware`) |

Idempotent: setting the same value twice leaves the same state.

### `SetPlanNotes`

Identical, with `notes` in place of `objective` and no length limit. Only `notes` is
written; `objective` is preserved.

## Isolation

Every operation is scoped to the authenticated caller's `user_id`, taken from the auth
context and never from the request. Two users' values for the same date are independent
and invisible to each other.

## Consumers

| Surface | Uses |
|---|---|
| TUI planning tab | `ListPlanEntries` (read both), `SetPlanObjective`, `SetPlanNotes` |
| CLI `twig plan [--date D] objective [text]` | `ListPlanEntries` (read), `SetPlanObjective` |
| Web app | None in this feature (FR-030) |

Notes are deliberately absent from the CLI (FR-029). The RPC exists and is usable, but no
CLI subcommand is added for it.

## CLI contract

```
twig plan [--date YYYY-MM-DD] objective              # print the day's objective
twig plan [--date YYYY-MM-DD] objective '<text>'     # set the day's objective
twig plan [--date YYYY-MM-DD] objective ''           # clear the day's objective
```

| Invocation | Output | Exit |
|---|---|---|
| Read, objective set | The objective verbatim, followed by a newline. No label, no styling | 0 |
| Read, objective unset | Nothing at all | 0 |
| Write, accepted | A short warm confirmation on stdout | 0 |
| Write, objective too long | Actionable error on stderr, naming the 255-character limit | 1 |
| Invalid `--date` | Existing plan-command date error | 1 |
| More than one positional argument after `objective` | Usage error on stderr | 1 |

`--date` is parsed by the plan command before subcommand dispatch, so `objective` inherits
it and the existing today-default with no flag handling of its own (FR-027).

The read path prints bare text so it can be piped, captured, or dropped into a shell
prompt. This is the one place the playful-tone principle yields to machine-readability;
the write path and all errors carry the warm tone.

## Compatibility

- Additive proto changes only; field numbers `PlanDay = 2` on `ListPlanEntriesResponse` and
  the two new RPCs are unused today.
- The web app calls `ListPlanEntries` and will silently receive the new `day` field without
  reading it — no change required there.
- A client built before this change and talking to a new server is unaffected.
- Adding new `*.v1` **services** requires a `vite.config.ts` proxy entry; this feature adds
  RPCs to an existing service, so no proxy change is needed.
