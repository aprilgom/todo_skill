---
name: "todo:delete"
description: Use when the user wants to remove, delete, drop, cancel, discard, or prune a todo action, backlog item, current todo item, or split living-plan action detail.
---

# Delete Todo Action

## Purpose

Remove a todo action from a split living plan without leaving dangling links or
silent history loss.

## CLI

Use `todo:init` first to ensure `TODO_CLI` is available. Then run this before
manual file edits when deleting a backlog action:

```sh
"$TODO_CLI" delete \
  --target "<rank|slug|filename|title>"
```

The CLI removes the backlog row, deletes the detail file, refuses to delete the
current action, refreshes plan state, and checks freshness.

## Manual Fallback

Fall back to manual editing when deleting active/current work, recording
priority rationale, or preserving the action as paused/not_started instead of
deleting it.

## Alternatives

- If the work was done, use the `todo:current` completion workflow instead.
- If the work should remain for later, keep it in `backlog.md` as `not_started`.
- If it started and is intentionally stopped, use status `paused`, not deletion.

## Guardrails

- Do not delete completed evidence.
- Do not delete a current action unless the user explicitly asked to remove it.
- Do not renumber remaining actions unless the user explicitly asks.
- Do not leave links to deleted detail files.

## Validation

Before reporting success:

- `rg` the removed title, rank, and detail filename under the plan directory.
- Confirm no index links point to deleted files.
- Confirm `current.md` still has at most one active goal.
- Run `git diff --check` for touched plan files.
- Run `.living-plan/scripts/check_plan_freshness.sh` and require `FRESH`.

## Final Response

Report the CLI `action`, `detail`, and `freshness` fields, plus any manual
dangling-link checks performed.

