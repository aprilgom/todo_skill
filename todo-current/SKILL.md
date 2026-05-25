---
name: "todo:current"
description: Use when the user invokes or discusses `/goal todo-current` or `todo:current`, asks to resume current todo work, complete the active todo goal, or keep current/backlog/completed living-plan files in sync.
---

# Todo Current

## Purpose

Operate the split living-plan current goal with minimal prompt tokens. This
skill assumes `todo:init` has created `.living-plan/living-plan.env` and
`PLAN_PATH` points to a split plan index. If the user wants to change the
content or target of the current action itself, use the `todo:modify-current`
workflow. If the user wants to promote a backlog action and demote the current
action, use the `todo:switch-current` workflow.

## CLI

Use `todo:init` first to ensure `TODO_CLI` is available. Then run:

```sh
"$TODO_CLI" current
```

To complete the active action:

```sh
"$TODO_CLI" current \
  --complete \
  --evidence "<observed completion evidence>"
```

The CLI reads `.living-plan/living-plan.env`, clears `current.md` on
completion, appends `completed.md`, records dated evidence in
`completed/YYYY-MM-DD.md`, refreshes plan state, and checks freshness.

## Manual Fallback

Fall back to manual editing only when the user asks for unsupported behavior,
such as custom completion records or immediate backlog promotion.

## Guardrails

- Do not read backlog details unless the user explicitly asks to switch actions.
- Do not invent completion evidence.
- Do not leave `current.md` with `not_started` or `paused`.
- Do not use `paused` unless the task actually started and was stopped.

## Final Response

Report the CLI `action`, `detail`, and `freshness` fields, plus whether the
goal remains current or was completed.

