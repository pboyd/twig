# Specification Quality Checklist: Goal Handling When Moving Tasks

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-08-01
**Feature**: [spec.md](../spec.md)

## Content Quality

- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

## Requirement Completeness

- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Success criteria are technology-agnostic (no implementation details)
- [x] All acceptance scenarios are defined
- [x] Edge cases are identified
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

## Feature Readiness

- [x] All functional requirements have clear acceptance criteria
- [x] User scenarios cover primary flows
- [x] Feature meets measurable outcomes defined in Success Criteria
- [x] No implementation details leak into specification

## Notes

- Two ambiguities in the original report were resolved with the user before drafting rather than
  left as markers:
  1. "Moving a sub-task under a task with a goal causes it to lose its goal link" was confirmed to
     mean **promotion to the top level** ("no parent"), not a move under another goal-bearing task.
     Captured as User Story 2 / FR-004.
  2. Repair of pre-existing nested tasks that hold their own goal link is **out of scope as a
     migration**; those rows are corrected opportunistically on the next move. Captured as
     User Story 3 and in Assumptions.
- FR-012 references the exact wording of an existing error message. This is user-facing copy being
  retired, not an implementation detail.
- Items marked incomplete require spec updates before `/speckit-clarify` or `/speckit-plan`
