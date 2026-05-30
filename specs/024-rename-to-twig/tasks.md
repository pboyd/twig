# Tasks: Rename App to Twig

**Input**: Design documents from `specs/024-rename-to-twig/`

**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅

**Organization**: Tasks grouped by user story. US3 (config path) is fully served by the Foundational phase and carries no dedicated phase of its own.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no shared dependencies)
- **[Story]**: Which user story this task belongs to (US1–US4)

---

## Phase 1: Setup

**Purpose**: Confirm baseline before any changes.

- [x] T001 Run `cd services/todo && go test ./...` to confirm all tests pass before starting

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config package changes underpin US2 (env vars) and US3 (config path). Must complete before US2/US3 verification.

**⚠️ CRITICAL**: US2 and US3 acceptance tests cannot pass until this phase is complete.

- [x] T002 Update env var names (`TODO_ADDR`→`TWIG_ADDR`, `TODO_API_KEY`→`TWIG_API_KEY`) and config path (`/todo/config.toml`→`/twig/config.toml`) in `services/todo/internal/config/config.go`
- [x] T003 Update env var names and any config path assertions in `services/todo/internal/config/config_test.go`

**Checkpoint**: Config package reads `TWIG_API_KEY`, `TWIG_ADDR`, and resolves `~/.config/twig/config.toml`

---

## Phase 3: User Story 1 — CLI Binary Named `twig` (Priority: P1) 🎯 MVP

**Goal**: All user-facing CLI text says `twig`; the binary builds as `twig`.

**Independent Test**: Build with `cd services/todo && go build -o twig ./cmd/todo`, run `./twig --help` — output contains `twig`, no `todo` product references. Running `./twig task list` works.

- [x] T004 [P] [US1] Update usage strings and error messages in `services/todo/internal/cli/cli.go` — replace all `todo` in user-facing strings with `twig`; update config path hint to `~/.config/twig/config.toml`; update error message to `"no API key found — set TWIG_API_KEY or add api_key to %s\n"`
- [x] T005 [P] [US1] Update usage strings in `services/todo/internal/cli/task.go` — replace all `todo task` prefixes with `twig task`
- [x] T006 [P] [US1] Update usage strings and error message in `services/todo/internal/cli/pom.go` — replace `todo pom` with `twig pom`; update `TODO_API_KEY` in error message to `TWIG_API_KEY` using the warmer pattern
- [x] T007 [P] [US1] Update usage strings and error message in `services/todo/internal/cli/plan.go` — replace all `todo plan` prefixes with `twig plan`; update `TODO_API_KEY` in error message to `TWIG_API_KEY` using the warmer pattern
- [x] T008 [P] [US1] Update Makefile build targets: `go build -o todo` → `go build -o twig`; `podman build -t todo-server` → `podman build -t twig-server`
- [x] T009 [US1] Update `services/todo/internal/cli/cli_test.go` — replace `t.Setenv("TODO_ADDR", ...)` with `TWIG_ADDR`; replace `t.Setenv("TODO_API_KEY", ...)` with `TWIG_API_KEY`; update any string assertions that check for `"TODO_API_KEY"` in error output to expect `"TWIG_API_KEY"`

**Checkpoint**: `go build -o twig ./cmd/todo` succeeds; `./twig --help` shows `twig`-only text; `go test ./internal/cli/...` passes

---

## Phase 4: User Story 2 — Env Var Names (`TWIG_API_KEY`, `TWIG_ADDR`) (Priority: P2)

**Goal**: The TUI uses the new env var name in its error message. (CLI files updated in US1; config.go updated in Foundational.)

**Independent Test**: Set only `TWIG_API_KEY` and launch the TUI — it connects. Set only `TODO_API_KEY` — it reports the key is missing, naming `TWIG_API_KEY` in the error.

- [x] T010 [US2] Update error message in `services/todo/internal/tui/tui.go` — `TODO_API_KEY` → `TWIG_API_KEY` using the warmer pattern: `"no API key found — set TWIG_API_KEY or add api_key to %s\n"`

**Checkpoint**: `go test ./internal/tui/...` passes (if tests exist); TUI error message names `TWIG_API_KEY`

---

## Phase 5: User Story 4 — Web UI Branding "Twig" (Priority: P4)

**Goal**: The web app shows "Twig" in page title, header, and all visible branding.

**Independent Test**: Load the web app — browser tab and app header read "Twig"; no "Todo" product name appears in the UI. Expanded-task state persists correctly under the new storage key.

- [x] T011 [P] [US4] Update `<title>Todo</title>` → `<title>Twig</title>` in `services/todo-web/index.html`
- [x] T012 [P] [US4] Update app header display text `Todo` → `Twig` in `services/todo-web/src/components/AppHeader.tsx`
- [x] T013 [P] [US4] Update localStorage key `"todo-expanded-tasks"` → `"twig-expanded-tasks"` in `services/todo-web/src/pages/TaskTreePage.tsx`

**Checkpoint**: `cd services/todo-web && npm test` passes; page title and header show "Twig"

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation and final verification across all stories.

- [x] T014 [P] Update `CLAUDE.md` — replace `Running \`todo\`` with `Running \`twig\``; update `go build -o todo` to `go build -o twig`; update `TODO_API_KEY`→`TWIG_API_KEY`, `TODO_ADDR`→`TWIG_ADDR`; update both `~/.config/todo/config.toml` references to `~/.config/twig/config.toml`
- [x] T015 Run full Go test suite: `cd services/todo && go test ./...` — all tests pass
- [x] T016 [P] Run web test suite: `cd services/todo-web && npm test` — all tests pass
- [x] T017 [P] Build and smoke-test: `cd services/todo && go build -o twig ./cmd/todo && ./twig --help` — verify output contains no `todo` product references

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies — run first
- **Phase 2 (Foundational)**: Depends on Phase 1 — blocks US2/US3 acceptance verification
- **Phase 3 (US1)**: Can start after Phase 1 (does not depend on Phase 2)
- **Phase 4 (US2)**: Depends on Phase 2 completion (config.go must be updated first)
- **Phase 5 (US4)**: Independent — can start any time after Phase 1
- **Phase 6 (Polish)**: Depends on all prior phases

### User Story Dependencies

- **US1 (P1)**: Independent — no dependency on Foundational changes
- **US2 (P2)**: Depends on Foundational (T002/T003) for config.go env var changes
- **US3 (P3)**: Fully handled in Foundational phase — no dedicated tasks
- **US4 (P4)**: Fully independent — web changes touch no shared files

### Parallel Opportunities

Within Phase 3: T004, T005, T006, T007, T008 can all run in parallel (different files).
Within Phase 5: T011, T012, T013 can all run in parallel (different files).
T009 (cli_test.go) must follow T004–T007 to match updated error message text.

---

## Parallel Example: Phase 3 (US1)

```bash
# All of these touch different files — run simultaneously:
Task: T004 — services/todo/internal/cli/cli.go
Task: T005 — services/todo/internal/cli/task.go
Task: T006 — services/todo/internal/cli/pom.go
Task: T007 — services/todo/internal/cli/plan.go
Task: T008 — Makefile

# Then, after the above complete:
Task: T009 — services/todo/internal/cli/cli_test.go (matches updated error text)
```

---

## Implementation Strategy

### MVP (User Story 1 Only)

1. Phase 1: Confirm baseline
2. Phase 3: Rename CLI usage strings and binary target
3. Build `twig` binary and confirm `./twig --help` is clean
4. **STOP and VALIDATE** before touching config or web

### Full Delivery (All Stories)

1. Phase 1 → Phase 2 → Phase 3 → Phase 4 → Phase 5 → Phase 6
2. Each phase is independently verifiable before proceeding

---

## Notes

- [P] tasks touch different files with no shared dependencies — safe to parallelize
- T009 has a soft ordering constraint: complete T004–T007 first so error message assertions match
- US3 (config path) has no dedicated phase — entirely covered by T002 (config.go) and T003 (config_test.go) in the Foundational phase
- Existing user config at `~/.config/todo/config.toml` is silently ignored after this change — no migration provided (per spec assumption)
- localStorage key change (T013) drops any previously persisted expanded-tree state — expected and acceptable
