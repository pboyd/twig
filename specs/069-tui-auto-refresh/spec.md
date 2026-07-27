# Feature Specification: TUI Auto-Refresh

**Feature Branch**: `069-tui-auto-refresh`

**Created**: 2026-07-27

**Status**: Draft

**Input**: User description: "TUI auto-refresh. Two mechanisms replacing today's fully-manual refresh: (1) entering any tab always re-fetches that tab's data; (2) a background heartbeat refreshes the active tab once its data is more than ten minutes old, skipping the refresh entirely while the user is mid-interaction and failing silently."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Switching tabs shows current data (Priority: P1)

A user completes a task from the Plan tab, then switches to the Tasks tab to keep working. The task shows as completed there, because entering a tab always reloads what that tab displays.

**Why this priority**: This is the bug users hit constantly, in a single session, with no network partition or second device involved. The TUI contradicts itself within seconds of an action the user just took, which erodes trust in everything else on screen. It is also the smallest slice: it delivers value with no timer, no staleness tracking, and no background behavior.

**Independent Test**: Complete a task on the Plan tab, press the tab key to reach the Tasks tab, and confirm the task renders as completed without pressing the refresh key. Repeat for the Goals tab, which currently loads once per session and then never reloads.

**Acceptance Scenarios**:

1. **Given** a task appears on both the Plan and Tasks tabs, **When** the user completes it on the Plan tab and switches to the Tasks tab, **Then** the Tasks tab shows it as completed.
2. **Given** the user has already visited the Goals tab once this session, **When** goal data changes and the user returns to the Goals tab, **Then** the Goals tab shows the changed data rather than the data from the first visit.
3. **Given** the user is on any tab, **When** they switch to any other tab, **Then** that tab's data is reloaded from the server.
4. **Given** a tab reload is in flight, **When** the user looks at the tab, **Then** the previously loaded content remains visible until the new data arrives, rather than blanking or showing a loading placeholder.
5. **Given** a tab reload fails, **When** the response comes back, **Then** the failure is reported the same way a manually requested reload failure is reported today.

---

### User Story 2 - Changes made elsewhere appear without asking (Priority: P2)

A user leaves the TUI running on a desktop, works from a laptop for the afternoon, and comes back to the desktop. Within about ten minutes of the last load, the screen catches up with the changes made elsewhere instead of showing hours-old state.

**Why this priority**: This is the second problem the user described. It is real but lower priority than Story 1 because it requires a second device or a long-running session, and because a manual refresh key already exists as a workaround. It also builds on the staleness tracking that Story 1 makes natural.

**Independent Test**: Load a tab, change the underlying data through another client, wait past the staleness threshold without touching the keyboard, and confirm the display catches up on its own.

**Acceptance Scenarios**:

1. **Given** the TUI sits idle on a tab and data changes elsewhere, **When** more than ten minutes have passed since that tab last loaded, **Then** the tab reloads on its own and shows the change.
2. **Given** a tab loaded thirty seconds ago, **When** the background mechanism next evaluates whether to refresh, **Then** it does not reload, because the data is not yet stale.
3. **Given** the user presses the manual refresh key, **When** ten minutes pass from that moment without further activity, **Then** the next automatic reload happens ten minutes after the manual refresh, not on a fixed schedule that ignores it.
4. **Given** the user is on the Tasks tab, **When** a background reload occurs, **Then** only the Tasks tab's data is fetched; other tabs are not fetched in the background.
5. **Given** a background reload fails because the server is unreachable, **When** the failure occurs, **Then** the currently displayed data stays on screen, no error is surfaced, and the mechanism tries again later.

---

### User Story 3 - Automatic refreshes never disturb work in progress (Priority: P3)

A user is halfway through typing a task description, or has a picker open, when a background refresh would otherwise fire. Nothing moves: the form stays open, the text stays typed, and the highlighted row stays on the same item.

**Why this priority**: An automatic refresh that interrupts the user is worse than no automatic refresh at all, so this must ship alongside Story 2. It is listed separately because it is independently testable and because it also constrains how the reload result is applied, not just when the reload is triggered.

**Independent Test**: Open each interactive mode in turn, hold it open past the staleness threshold, and confirm no reload occurs and no state is lost. Separately, force a reload whose result adds an item above the highlighted row, and confirm the highlight stays on the same item.

**Acceptance Scenarios**:

1. **Given** an edit form, a new-item form, a task picker, move mode, a delete confirmation, the help screen, or the filter bar is open, **When** the tab's data is stale, **Then** no background reload is triggered.
2. **Given** the user closes such a mode after suppressing one or more background reloads, **When** the next staleness evaluation occurs, **Then** the reload proceeds normally.
3. **Given** a background reload was already in flight and the user opens a form before the response arrives, **When** the response arrives, **Then** the open form is not closed and its contents are not discarded.
4. **Given** the highlight sits on a particular item and a background reload returns a list with a new item inserted above it, **When** the new data is applied, **Then** the highlight remains on the same item it was on before.
5. **Given** the highlighted item was deleted elsewhere and disappears in a background reload, **When** the new data is applied, **Then** the highlight moves to a nearby position without error.
6. **Given** an error message is already displayed, **When** a background reload succeeds, **Then** the existing error message is not silently cleared by the background reload.
7. **Given** the user has expanded or collapsed rows, set a filter, or scrolled, **When** a background reload is applied, **Then** the expansion state, filter, and scroll position are preserved.

---

### Edge Cases

- **Machine sleeps and wakes**: after a suspend that spans hours, the first staleness evaluation after wake finds the data stale and reloads once. It does not fire a burst of catch-up reloads for each interval that elapsed while suspended.
- **User quits mid-reload**: a reload in flight when the user quits does not delay or block shutdown.
- **Server unreachable for a long stretch**: repeated failed background reloads leave the last good data on screen and produce no accumulating error state or visible noise.
- **Rapid tab cycling**: tabbing quickly through all four tabs issues one reload per tab entry; the display never shows another tab's data.
- **Background reload lands while the user is on a different tab**: a reload that was dispatched for a tab the user has since left must not overwrite the tab they are now on.
- **Plan tab date not today**: the background mechanism refreshes whichever day the Plan tab is currently showing, not today's date.
- **Item completed locally and pending display retention**: a just-completed item being briefly retained on screen is not yanked away by a background reload arriving in that window.

## Requirements *(mandatory)*

### Functional Requirements

#### Tab-switch refresh

- **FR-001**: Entering any tab MUST reload that tab's data, unconditionally and without user action.
- **FR-002**: The tab-switch reload MUST apply to all four tabs, replacing today's inconsistent behavior where two tabs always reload, one reloads only if it has never loaded, and one never reloads on entry.
- **FR-003**: During a tab-switch reload, the previously loaded content MUST remain visible until the new data arrives.
- **FR-004**: Failures of a tab-switch reload MUST be surfaced to the user in the same manner as failures of a manually requested reload.

#### Background refresh

- **FR-005**: The system MUST evaluate, at a regular interval no longer than one minute, whether the active tab's data should be reloaded.
- **FR-006**: The system MUST track, per tab, when that tab's data was last loaded, updating that mark whenever data lands from any source: tab entry, manual refresh, background refresh, or a change the user made.
- **FR-007**: A background reload MUST be triggered only when the active tab's data is more than ten minutes old.
- **FR-008**: A background reload MUST fetch only the active tab's data.
- **FR-009**: The staleness interval MUST be fixed at ten minutes and MUST NOT be user-configurable.
- **FR-010**: Exactly one recurring evaluation MUST be active at a time, so that repeated user actions cannot cause reload evaluations to accumulate or multiply.
- **FR-011**: After a suspend or any gap longer than the interval, the system MUST perform at most one reload per tab rather than one per elapsed interval.

#### Suppression

- **FR-012**: The system MUST NOT trigger a background reload while the active tab is in any interactive mode rather than its plain list view. This includes edit forms, new-item forms, pickers, move mode, delete confirmations, the help screen, and the filter bar.
- **FR-013**: When a background reload is suppressed, the system MUST simply skip it and re-evaluate at the next interval, without queueing a deferred reload.

#### Applying background results

- **FR-014**: Applying a background reload MUST NOT close, reset, or discard any interactive mode that is open when the result arrives.
- **FR-015**: Applying a background reload MUST keep the highlight on the same item it was on before, identified by the item itself and not by its position in the list.
- **FR-016**: When the highlighted item is absent from the reloaded data, the system MUST move the highlight to a valid nearby position without error.
- **FR-017**: Applying a background reload MUST preserve expansion and collapse state, any active filter, and scroll position.
- **FR-018**: Applying a background reload MUST NOT clear an error message that is already displayed.
- **FR-019**: A background reload result MUST be discarded if the user has switched tabs since it was dispatched.

#### Error handling

- **FR-020**: A failed background reload MUST leave the currently displayed data on screen unchanged.
- **FR-021**: A failed background reload MUST NOT surface an error message to the user.
- **FR-022**: A failed background reload MUST NOT prevent later background reloads from being attempted.

#### Preserved behavior

- **FR-023**: The manual refresh key MUST continue to work on every tab exactly as it does today, including its error reporting.
- **FR-024**: No new visible interface element, indicator, or status text is introduced by this feature.

### Key Entities

- **Tab load timestamp**: per tab, the moment that tab's displayed data was last successfully loaded. Determines staleness. Reset by any successful load regardless of what triggered it.
- **Refresh trigger**: the reason a reload is happening — user action, tab entry, or background staleness. Determines whether errors are shown, whether the highlight is re-anchored by identity, and whether interactive modes may be disturbed.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After changing an item on one tab and switching to another tab that shows the same item, the change is visible 100% of the time with no additional keystrokes.
- **SC-002**: A TUI left running while data is changed elsewhere reflects that change within eleven minutes without user action, in place of the current behavior where it never does.
- **SC-003**: Zero instances of an automatic refresh closing an open form, discarding typed input, or moving the highlight to a different item, across a test suite covering every interactive mode on every tab.
- **SC-004**: With the server unreachable, an idle session running for an hour displays the same data it had when the server went away and shows no error text attributable to automatic refreshes.
- **SC-005**: An idle session on a single tab issues no more than six background data requests per hour.
- **SC-006**: Users no longer need to press the refresh key during ordinary tab-to-tab work, reducing manual refresh to an explicit "check now" action rather than a routine correction.

## Assumptions

- Ten minutes of staleness is acceptable to the user; near-real-time cross-device sync is explicitly not a goal.
- The evaluation interval of one minute is an internal detail chosen so the ten-minute threshold can be honored closely without a request every minute; only the ten-minute threshold is user-visible.
- Each tab already has a working data-loading path that a background refresh can reuse; no new server capability, endpoint, or push mechanism is required.
- Server load from tab-switch reloads is negligible, since tab switches happen at human pace.
- The existing per-tab reload paths other than the task tree already re-anchor or reset selection acceptably; where they do not, FR-015 through FR-017 apply to them equally.
- Terminal focus is not tracked; a background refresh may occur whether or not the terminal window is focused.
- The pomodoro timer's own countdown and any existing plan-tab timers are unaffected by this feature and continue to operate independently.
