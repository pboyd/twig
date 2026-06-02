# Data Model: Pomodoro Progress Display

This feature is presentation-centric. It introduces no new persisted entities; it surfaces existing pomodoro data and derives a render model from it.

## Entities

### Task pomodoro state (existing + one surfaced field)

| Field | Source | Range / Rules | Notes |
|-------|--------|---------------|-------|
| `estimate` | `Task.estimate` (existing) | 0..10 (server-enforced); 0 = "no estimate" | Drives the dimmed/baseline glyph count |
| `completed_pomodoro_count` | **NEW** on `Task`, populated by `ListTasks` | ≥ 0 (`count` of completed pomodoros for the task, for the calling user) | Drives the red (≤ estimate) and yellow (> estimate) glyphs |

`completed_pomodoro_count` is a per-user derived count (`pomodoros WHERE task_id = ? AND user_id = ? AND complete`). It is read-only and computed server-side; clients never write it.

### Active pomodoro (existing)

The single in-progress pomodoro (`pomodoros` row with `end_at IS NULL`). Already modeled in the TUI as `activePom`. Only its **glyph styling** changes (red + bold); no data change.

## Derived render model — `PomodoroRow`

Computed purely from `(estimate, completed)`; not persisted.

```
n        = max(estimate, completed)          // total glyphs in the row
done     = min(completed, estimate)          // bold-red glyphs (completed within estimate)
remain   = estimate - done                   // dimmed glyphs (estimated, not yet done)
over     = max(0, completed - estimate)      // yellow+bold glyphs (completed beyond estimate)
// invariant: done + remain + over == n
```

Row layout, left to right: `done` × (bold red 🍅) · `remain` × (dim 🍅) · `over` × (yellow bold 🍅).

### Visibility / empty state

- If `estimate == 0 && completed == 0` → **omit the row entirely** (render nothing).
- If `estimate == 0 && completed > 0` → `n == completed`, all glyphs in the `over` (yellow+bold) segment.
- If `estimate > 0 && completed == 0` → `n == estimate`, all glyphs dimmed.

### Worked examples (from the spec)

| estimate | completed | n | done (red) | remain (dim) | over (yellow) |
|---------:|----------:|--:|-----------:|-------------:|--------------:|
| 5 | 0 | 5 | 0 | 5 | 0 |
| 5 | 2 | 5 | 2 | 3 | 0 |
| 5 | 5 | 5 | 5 | 0 | 0 |
| 5 | 6 | 6 | 5 | 0 | 1 |
| 3 | 1 | 3 | 1 | 2 | 0 |
| 0 | 2 | 2 | 0 | 0 | 2 |
| 0 | 0 | — | — | — | — (omitted) |

## Styling tokens (shared palette, `internal/tui/theme.go`)

| Token | Role | Suggested value (adaptive) |
|-------|------|----------------------------|
| `pomodoroDone` | bold-red completed glyphs **and** the active-pomodoro status glyph | red (light `#CC0000` / dark `#FF5555`) |
| `pomodoroOver` | yellow+bold over-estimate glyphs | yellow (light `#B58900` / dark `#FFD75F`) |
| `dim` (existing) | dimmed estimated-not-done glyphs | reuse existing `dim` token |

Exact hex values are an implementation detail; they MUST be added as palette tokens (no inline colors) to satisfy Principle III. Bold is applied via Lipgloss `Bold(true)` on the done/over/active styles.

## Consumers

- **Task-tree details** (`renderDetails`): replaces the numeric `Est:` line with the glyph row (when non-empty).
- **Planning-tab details** (`renderPlanDetail`): adds the glyph row for a linked task, looked up from the in-memory tree by `PlanEntry.task_id`.
- **Active-pomodoro status line** (`renderStatus` in `view.go`): leading 🍅 styled with `pomodoroDone` (bold).
