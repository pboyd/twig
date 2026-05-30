<!--
SYNC IMPACT REPORT
==================
Version change: 1.0.0 → 1.1.0
Modified principles: none
Added sections:
  - Principle III: UI/UX Consistency
  - Principle IV: Playful User Messages
Removed sections: none
Templates requiring updates:
  ✅ plan-template.md — Constitution Check gates updated to reference Principles III and IV
  ✅ spec-template.md — No structural update required
  ✅ tasks-template.md — No new principle-driven task types required
  ✅ Quality Gates — Updated to cover Principles III and IV
Deferred TODOs: none
-->

# Todo Constitution

## Core Principles

### I. Simplicity / YAGNI

Every implementation decision MUST start from the simplest solution that satisfies the current requirement.
Speculative abstractions, helper layers, and generalization for hypothetical future needs are prohibited.
Three similar lines of code are preferable to a premature abstraction. Complexity MUST be explicitly
justified in the plan's Complexity Tracking table when a Constitution Check violation is required.

**Rationale**: Full-stack projects accumulate accidental complexity quickly. Enforcing simplicity at the
governance level keeps the codebase navigable and the delivery pace sustainable.

### II. API-First Design

Data contracts (request/response schemas, API endpoints, shared types) MUST be defined and reviewed before
any frontend or backend implementation begins for a feature. The contract is the source of truth for
frontend/backend alignment; implementation MUST conform to it, not the other way around.

**Rationale**: In a full-stack todo app, the frontend and backend can be developed in parallel only when
the contract is stable. Defining the API first eliminates integration surprises and enables independent
testability of each tier.

### III. UI/UX Consistency

All user-facing surfaces MUST share a coherent visual and interaction language:

- **TUI**: Colors, styles, key bindings, and layout MUST be drawn from a single shared theme/palette.
  No surface may introduce ad-hoc colors or styles that deviate from the established theme.
- **CLI**: Command structure, flag naming, and output formatting MUST follow consistent conventions
  across all subcommands. A user who learns one command MUST be able to predict how others behave.
- **Web frontend** (when introduced): Pages MUST share the same component library, spacing system,
  and interaction patterns. Per-page one-off styles are prohibited without explicit justification.

**Rationale**: The application spans multiple surfaces (TUI, CLI, future web). Visual and interaction
inconsistency erodes trust and increases the learning curve. Enforcing consistency at the governance
level prevents surfaces from drifting apart as the project grows.

### IV. Playful User Messages

All user-facing text — status messages, prompts, confirmations, errors, and empty states — MUST carry
a slightly playful tone. Dry, terse system output is prohibited where a warmer alternative exists and
does not sacrifice clarity.

Rules:
- Messages MUST remain accurate and unambiguous; playfulness MUST NOT obscure meaning.
- Playfulness is expressed through word choice and light wit, not emoji spam or forced humor.
- Error messages MUST still be actionable; the tone softens the delivery, not the information.
- The degree of playfulness SHOULD match context: a destructive-action confirmation is warm but
  measured; an empty task list can be more whimsical.

**Rationale**: Managing a daily todo list is routine work. Small moments of levity in the interface
make the tool feel less like a chore and more like a helpful companion.

## Development Workflow

This project follows a feature-branch workflow governed by Spec Kit. Each feature MUST progress through:
specification → planning → task generation → implementation. Skipping phases is not permitted.

- Branch naming: `###-feature-name` (sequential numbering, managed by `/speckit-git-feature`)
- All design artifacts live under `specs/###-feature-name/` and are committed with the feature branch
- API contracts MUST be committed to `specs/###-feature-name/contracts/` before implementation tasks begin
- Implementation tasks MUST reference the contract file they implement

## Quality Gates

Before any feature branch is merged to `main`:

1. All API contracts defined in `specs/###-feature-name/contracts/` MUST exist and match the implementation
2. No added abstractions without a corresponding entry in the plan's Complexity Tracking table
3. The plan's Constitution Check section MUST be completed and passing for Principles I, II, III, and IV
4. No placeholder tokens (`[ALL_CAPS_IDENTIFIER]`) may remain in committed spec, plan, or contract files
5. Any new user-facing text introduced by the feature MUST be reviewed for tone compliance with Principle IV
6. Any new TUI/CLI surface MUST be verified for visual and interaction consistency with Principle III

## Governance

This constitution supersedes all other written practices for this project. Amendments require:

1. A documented rationale (what changed and why)
2. A version bump per semantic versioning:
   - **MAJOR**: Principle removal or redefinition that breaks existing compliance
   - **MINOR**: New principle or section added
   - **PATCH**: Clarifications, wording, or non-semantic refinements
3. A propagation check across all Spec Kit templates and any runtime guidance files
4. The `LAST_AMENDED_DATE` updated to the amendment date

All pull requests and spec reviews MUST verify compliance with all four Principles. Complexity that
violates Principle I MUST be justified in the plan before implementation begins; unjustified complexity
is grounds for rejection. UI/UX deviations from Principle III and tone violations of Principle IV are
likewise grounds for rejection.

Runtime development guidance lives in `CLAUDE.md` at the repository root.

**Version**: 1.1.0 | **Ratified**: 2026-05-15 | **Last Amended**: 2026-05-30
