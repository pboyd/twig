# Contract: UI / UX consistency & tone (Principles III & IV)

This is the first **web** surface for the project. To satisfy Constitution Principle III (UI/UX Consistency) and Principle IV (Playful User Messages), this contract defines the shared foundations every page MUST use. Per-page one-off styles or ad-hoc tone are prohibited without justification.

## Shared theme (Principle III)

A single source of design tokens lives in `src/theme/` and is wired into `tailwind.config.ts`. All components consume tokens; no hard-coded hex values or one-off spacing in pages.

- **Color tokens**: `bg`, `surface`, `text`, `muted`, `accent`, `danger`, `success` (light + dark via mobile `prefers-color-scheme`).
- **Spacing scale**: Tailwind default scale only; no arbitrary `px` values in JSX.
- **Typography**: one font stack; a small set of size tokens (`title`, `body`, `caption`).
- **Touch targets**: interactive elements ≥ 44×44px (mobile-first; SC-004).

## Shared component set (Principle III)

Pages compose these; they do not restyle them inline:

| Component | Responsibility |
|-----------|----------------|
| `Button` | Primary/secondary/danger variants; loading + disabled states. |
| `Field` | Labeled input/textarea with inline validation message. |
| `TreeRow` | One task row: completion toggle, name, sub-task affordance, expand/collapse chevron, indentation by `depth`. |
| `TaskForm` | Add/edit form (name required, description optional); reused for create and edit. |
| `EmptyState` | Shown when the user has no tasks (FR-016). |
| `ErrorBanner` | Connectivity/expiry/precondition errors with a retry affordance (FR-015). |
| `Spinner` | Loading state for queries/mutations. |

## Interaction patterns (Principle III)

- Navigation: tree at `/tasks`; tapping a row opens `/tasks/:id`; back returns to the tree with expand state intact.
- Mutations show a pending state and surface failures via `ErrorBanner`; on success the relevant query is invalidated and the view refreshes.
- No horizontal scroll or pinch-zoom required at any breakpoint (SC-004).

## Tone (Principle IV)

All user-facing copy is centralized in `src/theme/messages.ts` for tone review. Strings are warm and lightly playful but accurate and actionable. Canonical strings:

| Context | String |
|---------|--------|
| Empty task list | "Nothing here yet — your future self is grateful. Add the first task." |
| Empty-name validation | "A task needs a name to live by." |
| Login failure | "That didn't match. Mind trying again?" |
| Session expired | "Your session clocked out. Let's sign back in." |
| Complete blocked (sub-tasks) | "Hold on — finish its sub-tasks first." |
| Reopen blocked (parent done) | "Reopen its parent first to reopen this one." |
| Task not found | "That task seems to have wandered off." |
| Connectivity error | "Couldn't reach the server. Want to try again?" |
| Save success (subtle) | "Saved. ✨" (used sparingly) |

Rules (from the constitution): playfulness via word choice, not emoji spam; errors stay actionable; destructive/blocking messages are warm but measured; empty states may be more whimsical.
