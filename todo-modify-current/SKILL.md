---
name: "todo:modify-current"
description: Use when the user wants to revise, retarget, rename, narrow, expand, or otherwise change the action currently referenced by a split living-plan current.md file.
---

# Modify Todo Current

## Purpose

Modify the content or target of the action currently referenced by `current.md`
without switching to a different backlog action.

## CLI

Use `todo:init` first to ensure `TODO_CLI` is available. Then run:

```sh
"$TODO_CLI" modify-current \
  --title "<new title>" \
  --evidence "<replacement evidence>" \
  --next-check "<replacement next check>"
```

Pass only the fields that should change. The CLI keeps the same action current,
keeps status `in_progress`, refreshes plan state, and checks freshness.

## Manual Fallback

Fall back to manual editing only when the user asks for edits outside title,
evidence, or next check. If the request selects a different action, use the
switch-current workflow instead.

## Guardrails

- Do not move the current action to completed.
- Do not promote a backlog action.
- Do not change status away from `in_progress`.
- Do not invent evidence; record only observed facts or user-provided intent.

## Final Response

Report the CLI `action`, `detail`, and `freshness` fields, plus whether
`current.md` changed.

