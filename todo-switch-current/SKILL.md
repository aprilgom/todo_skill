---
name: "todo:switch-current"
description: Use when the user wants to switch todo focus, promote a backlog todo action to current, demote the current todo action to backlog, swap current/backlog items, or change which split living-plan action is active.
---

# Switch Todo Current

## Purpose

Move one backlog todo action into `current.md` and move the old current action
back to `backlog.md` without completing or deleting either action.

## CLI

Use `todo:init` first to ensure `TODO_CLI` is available. Then run:

```sh
"$TODO_CLI" switch \
  --to "<rank|slug|filename|title>"
```

Use `--demote-status paused` only when there is evidence that the previous
current action started and is intentionally stopped.

The CLI reads `.living-plan/living-plan.env`, promotes the selected backlog
action, demotes the old current action, updates both detail statuses, refreshes
plan state, and checks freshness.

## Status Changes

- promoted action: `not_started` or `paused` -> `in_progress`
- demoted action: `in_progress` -> `not_started` unless there is evidence it
  started and is intentionally stopped, then `paused`

## Manual Fallback

Fall back to manual editing only when the switch requires priority rationale or
other notes that the CLI does not support.

## Guardrails

- Do not mark either action completed.
- Do not delete action detail files.
- Do not leave more than one current action.
- Do not read unrelated backlog details.
- Do not use `paused` without evidence that the demoted action had started.

## Validation

- `current.md` contains exactly one active action link.
- `backlog.md` contains the demoted action and not the promoted action.
- Promoted detail status is `in_progress`.
- Demoted detail status is `not_started` or `paused`.
- `git diff --check` passes for touched plan files.
- Freshness check reports `FRESH`.

## Final Response

Report the CLI `promoted`, `demoted`, `current`, `backlog`, and `freshness`
fields.

