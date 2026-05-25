---
name: "todo:list"
description: Use when the user wants to list, show, inspect, summarize, or check todo current, backlog, completed, or freshness state in a split living-plan todo structure.
---

# List Todo Actions

## Purpose

Show the current, backlog, completed, and freshness state of a split living plan
without changing todo status or plan files.

## CLI

Use `todo:init` first to ensure `TODO_CLI` is available. Then run:

```sh
"$TODO_CLI" list
```

The CLI reads `.living-plan/living-plan.env`, checks freshness, and prints
concise `current`, `backlog`, and `completed` sections.

## Workflow

1. Ensure `TODO_CLI` is available with `todo:init`.
2. Run `"$TODO_CLI" list` from the target repository.
3. If the command fails because freshness is stale, stop and report the freshness error instead of summarizing stale todos.
4. If a section prints `none`, report that section as empty.
5. Do not open action detail files unless the user asks for expanded details.

## Guardrails

- Do not modify todo files while listing.
- Do not read action detail files unless the user asks for details.
- Treat `freshness` other than `FRESH` as a stop condition before reporting todo state as current.

## Validation

- CLI output includes `current:`, `backlog:`, `completed:`, and `freshness: FRESH`.
- Listing does not change `current.md`, `backlog.md`, `completed.md`, or action detail files.
- If details are requested, verify links from the listed rows before opening detail files.

## Final Response

Report the CLI `current`, `backlog`, `completed`, and `freshness` fields. Keep
the answer concise unless the user asks for detail.
