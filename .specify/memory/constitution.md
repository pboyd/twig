<!--
SYNC IMPACT REPORT
==================
Version change: (none) → 1.0.0
Modified principles: N/A (initial ratification)
Added sections: Core Principles (I. Simplicity/YAGNI, II. API-First Design), Development Workflow, Quality Gates, Governance
Removed sections: Principle slots 3, 4, 5 (template placeholders — not needed; user selected 2 principles)
Templates requiring updates:
  ✅ plan-template.md — Constitution Check gates are generic; no structural update required
  ✅ spec-template.md — No constitution-specific mandatory sections added
  ✅ tasks-template.md — No new principle-driven task types required
Deferred TODOs: none
-->

# Todo Constitution

## Core Principles

### I. Simplicity / YAGNI

Every implementation decision MUST start from the simplest solution that satisfies the current requirement. Speculative abstractions, helper layers, and generalization for hypothetical future needs are prohibited. Three similar lines of code are preferable to a premature abstraction. Complexity MUST be explicitly justified in the plan's Complexity Tracking table when a Constitution Check violation is required.

**Rationale**: Full-stack projects accumulate accidental complexity quickly. Enforcing simplicity at the governance level keeps the codebase navigable and the delivery pace sustainable.

### II. API-First Design

Data contracts (request/response schemas, API endpoints, shared types) MUST be defined and reviewed before any frontend or backend implementation begins for a feature. The contract is the source of truth for frontend/backend alignment; implementation MUST conform to it, not the other way around.

**Rationale**: In a full-stack todo app, the frontend and backend can be developed in parallel only when the contract is stable. Defining the API first eliminates integration surprises and enables independent testability of each tier.

## Development Workflow

This project follows a feature-branch workflow governed by Spec Kit. Each feature MUST progress through: specification → planning → task generation → implementation. Skipping phases is not permitted.

- Branch naming: `###-feature-name` (sequential numbering, managed by `/speckit-git-feature`)
- All design artifacts live under `specs/###-feature-name/` and are committed with the feature branch
- API contracts MUST be committed to `specs/###-feature-name/contracts/` before implementation tasks begin
- Implementation tasks MUST reference the contract file they implement

## Quality Gates

Before any feature branch is merged to `main`:

1. All API contracts defined in `specs/###-feature-name/contracts/` MUST exist and match the implementation
2. No added abstractions without a corresponding entry in the plan's Complexity Tracking table
3. The plan's Constitution Check section MUST be completed and passing for Principles I and II
4. No placeholder tokens (`[ALL_CAPS_IDENTIFIER]`) may remain in committed spec, plan, or contract files

## Governance

This constitution supersedes all other written practices for this project. Amendments require:

1. A documented rationale (what changed and why)
2. A version bump per semantic versioning:
   - **MAJOR**: Principle removal or redefinition that breaks existing compliance
   - **MINOR**: New principle or section added
   - **PATCH**: Clarifications, wording, or non-semantic refinements
3. A propagation check across all Spec Kit templates and any runtime guidance files
4. The `LAST_AMENDED_DATE` updated to the amendment date

All pull requests and spec reviews MUST verify compliance with Principles I and II. Complexity that violates Principle I MUST be justified in the plan before implementation begins; unjustified complexity is grounds for rejection.

Runtime development guidance lives in `CLAUDE.md` at the repository root.

**Version**: 1.0.0 | **Ratified**: 2026-05-15 | **Last Amended**: 2026-05-15
