# Quickstart: Show Actual Times in Plan Grid Labels

## What you are changing

Two files, roughly ten lines:

| File | Change |
|---|---|
| `internal/cli/plan_grid.go` | Build the label from exact times (lines 121-127) |
| `internal/tui/plan_preview.go` | Compare exact intervals; mark overlap-region slots (lines 110-144) |

Do **not** touch any other `snapDown15`/`snapUp15` call site — the other seven are geometry and are correct. See research.md for the full inventory.

## Reproduce the bugs first

Both are visible in the TUI's Plan tab with two back-to-back entries at non-aligned times:

```bash
go build -o twig ./cmd/twig && ./twig
```

Create entries at 13:00 (50 min) and 13:50 (20 min), then open the Plan tab.

- **Bug 1**: labels read `13:00-14:00` and `13:45-14:15` instead of `13:00-13:50` and `13:50-14:10`.
- **Bug 2**: press edit on the first entry — a red conflict marker appears at the 13:45 slot even though the entries only touch.

If you prefer not to run the TUI, both reproduce as failing unit tests; write those first (see below).

## Change 1 — label text

In the `layouts` loop of `RenderGrid`, `sn`/`se` stay exactly as they are for `tl`/`bl`. Add the exact pair and use it only in the two `Sprintf` calls:

```go
es := int(e.GetStartMinute())
ee := es + int(e.DurationMinute)
```

Then swap `sn`→`es` and `se`→`ee` inside the format arguments at lines 123 and 125. The format string, the `HideID` branch, and the `wrapLabel` call all stay put.

## Change 2 — conflict detection

Rewrite the body of the `others` loop in `planPreviewConflicts` to compare exact intervals:

```go
ovStart := max(pStart, oStart)
ovEnd := min(pEnd, oEnd)
if ovStart >= ovEnd {
    continue // no overlap; touching boundaries land here
}
for t := snapDown15(ovStart); t < ovEnd; t += 15 {
    // mark t
}
```

Delete the two `if xEnd <= xStart { xEnd = xStart + 15 }` clamps — exact intervals get no artificial minimum. Keep `snapDown15`; it still maps the overlap region onto slot keys.

## Verify

```bash
go test ./...
```

New tests must use **non-aligned** times — the existing two conflict tests use only 15-minute-aligned values, which is why they pass against the buggy code and cannot catch either defect. Cover at minimum:

- Label for an entry at 13:50 + 20 min reads `13:50-14:10` while its box still spans 13:45–14:15.
- Label with `HideID = false` keeps its `[id]` prefix.
- 13:00–13:50 vs 13:50–14:10 yields no conflict.
- 13:00–14:00 vs 13:50–14:10 yields exactly `{825}`.
- 13:00–13:05 vs 13:10–13:20 yields no conflict (sub-slot neighbors).

Then confirm in the running TUI that box geometry is visually identical to before and only the label text moved (SC-005).

## Watch out for

- **Don't change the hour gutter** at line 244 — it labels grid rows and is correctly snapped (FR-008).
- **Don't apply the `se < sn+15` minimum to labels** — that clamp is geometry-only. A 5-minute entry labels as its true 5 minutes while still drawing one full slot.
- **Conflict slots must stay multiples of 15** — the renderer indexes the map by slot minute at `plan_grid.go:193` and `:215`, so a non-aligned key would silently never render.
