# Contract: Filter Expression Grammar & Semantics

**Feature**: 062-task-filters | **Implemented by**: `services/twig/internal/filter`

The grammar below is normative. The surface syntax may only be extended (never changed incompatibly) in later features; `OR`, `NOT`, and parentheses are deliberately reserved.

## Grammar (EBNF)

```ebnf
expression  = condition , { ws , "AND" , ws , condition } ;
condition   = field-cond | text-term ;

field-cond  = [ "^" ] , field , [ws] , op , [ws] , value ;
field       = "completed" | "snoozed" | "parent_id" | "goal_id" ;
op          = "=" | "!=" | "<" | "<=" | ">" | ">=" ;

value       = bool | date | integer ;
bool        = "true" | "false" ;
date        = digit{4} , "-" , digit{2} , "-" , digit{2} ;      (* YYYY-MM-DD, real calendar date *)
integer     = digit , { digit } ;                                (* positive task/goal id *)

text-term   = quoted | bareword ;
quoted      = '"' , { any character except '"' } , '"' ;
bareword    = any run of non-whitespace characters that is not a
              field-cond and is not the keyword "AND" ;
ws          = one or more spaces ;
```

Notes:
- `AND` is case-insensitive as a keyword; field names and `true`/`false` are lowercase only.
- A bareword that *looks like* the start of a field condition (e.g. `completed=`) is parsed as a field condition; quoting forces text interpretation (`"completed="`).
- Whitespace around operators is optional: `completed=true` ≡ `completed = true`.

## Type rules (violations ⇒ `InvalidArgument`)

| Field | Allowed ops | Value type | `^` allowed |
|-------|------------|------------|-------------|
| `completed` | `=`, `!=` | bool | no |
| `completed` | `=`, `!=`, `<`, `<=`, `>`, `>=` | date | no |
| `snoozed` | `=`, `!=` | bool | no |
| `parent_id` | `=`, `!=` | integer | yes |
| `goal_id` | `=`, `!=` | integer | yes |
| (text term) | — | string | no |

## Evaluation semantics

A task matches the expression when it satisfies **every** condition (explicit and implicit).

| Condition | A task satisfies it when… |
|-----------|---------------------------|
| text term `T` | `T` occurs case-insensitively in the task's name or description |
| `completed=true` / `false` | `completed_at` is set / unset |
| `completed <op> D` | `completed_at` is set **and**, comparing at day granularity (`completed_at`'s UTC calendar date `<op>` `D`), the comparison holds |
| `snoozed=true` | `snooze_until` is set **and** its day is strictly after the request's `today` |
| `snoozed=false` | not `snoozed=true` |
| `parent_id=N` | the task's direct parent is task N |
| `^parent_id=N` | task N is an ancestor (any depth); N itself does not match |
| `goal_id=N` | the task is directly associated with goal N (it is the association root) |
| `^goal_id=N` | the task is an association root for goal N **or** a descendant of one |
| `!=` variants | the negation of the corresponding `=` form |

**Implicit default-visibility conditions (FR-008)** — applied by the evaluator, per attribute:

| Expression mentions `completed`? | Expression mentions `snoozed`? | `show_all` | Implicit conditions added |
|:--:|:--:|:--:|---|
| no | no | off | `completed=false AND snoozed=false` |
| yes | no | off | `snoozed=false` |
| no | yes | off | `completed=false` |
| — | — | on | none |

A date comparison on `completed` counts as "mentions `completed`".

**Nonexistent ids**: conditions referencing a task/goal id that doesn't exist (for this user) are simply unsatisfiable (`=`/`^` forms match nothing; `!=` forms exclude nothing). Never an error.

## Worked examples (from the spec — all must hold in tests)

| Expression | show_all | Result |
|------------|:--:|--------|
| `foo` | off | incomplete, unsnoozed tasks with "foo" in name/description |
| `foo` | on | all tasks with "foo" in name/description |
| `completed=true AND parent_id=1` | off | completed, unsnoozed direct children of task 1 |
| `completed=false AND ^parent_id=1` | off | incomplete, unsnoozed descendants of task 1 (any depth) |
| `snoozed=true` | off | snoozed, incomplete tasks (snooze explicit; completed default still applies) |
| `completed < 2026-01-01` | off | tasks completed before 2026-01-01 UTC, unsnoozed |
| `completed=false AND snoozed=false AND ^goal_id=1` | off | the whole incomplete, unsnoozed tree under goal 1 |
| `groceries AND completed=false AND ^goal_id=2` | any | incomplete goal-2-tree tasks containing "groceries" |
| `"AND review"` | off | incomplete, unsnoozed tasks containing the literal text "AND review" |

## Display contract (client-side, uniform across clients)

Given the matched-id set, a client renders: every matched task **plus every ancestor of a matched task**, in tree order with structure preserved. Non-matching descendants of a matched task are hidden. Collapsed/expansion state is ignored while a filter is active. Ancestor-context rows are ordinary interactive rows.
