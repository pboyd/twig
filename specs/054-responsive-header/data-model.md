# Phase 1 Data Model: Responsive Header for Mobile

This feature introduces **no persisted entities** and **no backend data**. It is a presentation/navigation change only.

## Ephemeral UI state

| State | Owner | Type | Description |
|-------|-------|------|-------------|
| `menuOpen` | `HeaderMenu` component | boolean | Whether the compact dropdown is currently open. Resets to `false` on item activation, outside pointer-down, `Escape`, and route change. Not persisted. |

## Navigation items (static configuration)

The header presents a fixed set of items. There is no dynamic data; this table documents intended placement, not a stored structure.

| Item | Destination / Action | Wide viewport (≥640px) | Small viewport (<640px) |
|------|----------------------|------------------------|--------------------------|
| Logo | `/tasks` (home) | Visible | Visible |
| "Twig" wordmark | — (label) | Visible | Hidden |
| Tasks | `/tasks` | Header (direct) | Header (direct) |
| Plan | `/plan` | Header (direct) | Header (direct) |
| Download | `/download` | Header (direct) | Compact menu |
| Account | `/account` | Header (direct) | Compact menu |
| Sign out | `POST /auth/logout` → `/login` | Header (direct) | Compact menu |

**Active state**: the current destination is visually marked wherever it appears (header link active styling on wide; reflected in the menu when open on small viewports).

No validation rules, lifecycle transitions, or relationships apply beyond the open/close toggle described above.
