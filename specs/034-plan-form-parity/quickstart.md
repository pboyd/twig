# Quickstart: Verifying Plan Form Parity

Manual + automated verification for feature `034-plan-form-parity`.

## Prerequisites

- Stack running: `make dev` (server + postgres), with a provisioned user and `TWIG_API_KEY` / `TWIG_ADDR` set.
- At least one plan entry for today (one scheduled, one untimed) to exercise both edit paths.

## Automated checks

```bash
cd services/twig
go test ./internal/tui/...        # form rendering + submit behavior
go test ./internal/cli/timeparse/ # duration format/parse round-trip (if FormatDuration added)
go test ./...                     # full suite
go build -o twig ./cmd/twig       # binary builds
```

Expected: all green; no references to `blank=keep`, `null=unschedule`, or the old `[ctrl+s/enter] save` help line remain in `internal/tui`.

```bash
! grep -rn "blank=keep\|null=unschedule\|ctrl+s/enter" internal/tui/*.go
```

## Manual walkthrough (TUI)

Launch `./twig`, go to the **Planning** tab.

### Edit form parity (User Story 1)

1. Select a **scheduled** entry (e.g. "Farm pigs", 09:00, 30m) and press **Enter**.
   - ✅ Title line reads `Edit entry`, blank line below it.
   - ✅ Name shows `Farm pigs`; Start shows `09:00`; Duration shows `30m`. No `blank=keep` text anywhere.
   - ✅ Blank line separates each field.
   - ✅ `[ Save ]  [ Cancel ]` shown below fields.
   - ✅ Bottom line reads exactly `Ctrl+S: save  Esc: cancel  Tab: next field`.
2. Press **Enter** while a text field is focused.
   - ✅ Form does **not** submit; focus advances to the next field.
3. Press **Tab** repeatedly.
   - ✅ Focus cycles fields → `[>Save<]` → `[>Cancel<]` → back to Name.
4. Press **Ctrl+S** without changing anything.
   - ✅ Form closes; entry unchanged (no duplicate/move flicker).
5. Re-open, change the name, press **Ctrl+S**.
   - ✅ Entry is renamed.
6. Re-open the scheduled entry, **clear the Start field**, press **Ctrl+S**.
   - ✅ Entry is unscheduled (moves to the untimed pane); any Duration text is ignored.
7. Re-open an **untimed** entry.
   - ✅ Start and Duration are empty; leaving them empty and saving keeps it untimed.
8. Re-open, focus **Cancel**, press **Enter** (or press **Esc**) after editing a field.
   - ✅ Changes discarded; back to the plan list.

### Other forms parity (User Story 2)

9. Press the add-task key → pick a task → schedule form.
   - ✅ Title `Schedule task`, blank-line spacing, `[ Save ]  [ Cancel ]`, the new help line; Enter in a field does not submit.
10. Press the add-event key.
    - ✅ Title `Add event`, same parity treatment.

## Success criteria mapping

| Check | Spec criterion |
|---|---|
| Steps 1, 9, 10 side-by-side with task form | SC-001 |
| Step 1 (all fields pre-filled) + step 4 (no-op save) | SC-002 |
| Step 2 (Enter never submits) | SC-003 |
| Steps 5–7 (rename / reschedule / unschedule all reachable) | SC-004 |
| Step 1 help line exact match | SC-005 |
