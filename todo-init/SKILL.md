---
name: "todo:init"
description: Use before any todo living-plan workflow when the bundled todo CLI may be missing, unbuilt, or the target repository may not have `.living-plan/living-plan.env`.
---

# Todo Init

## Purpose

Prepare the bundled todo CLI and initialize a repository-local living plan when
needed.

## CLI Readiness

Before running any todo workflow, make the bundled CLI available:

```sh
TODO_SKILL_DIR="${CODEX_HOME:-$HOME/.codex}/skills/todo"
TODO_CLI="$TODO_SKILL_DIR/tools/todo"

if [[ ! -x "$TODO_CLI" ]]; then
  (cd "$TODO_SKILL_DIR/tools/todo-cli" && go build -o "$TODO_CLI" ./cmd/todo)
fi

"$TODO_CLI" --help
```

If `go build` fails because Go is unavailable, report that the todo CLI cannot
be built and stop before editing plan files.

## Repository Initialization

If `.living-plan/living-plan.env` is missing in the target repository, initialize
it before using todo workflows:

```sh
"$TODO_CLI" init \
  --scope "<stable-project-short-name>" \
  --sensitive-path "." \
  --agent-link-path "AGENTS.md"
```

Choose a stable `--scope`, usually the repository or project short name. Use
`--no-git-hook` or `--no-codex-hook` only when the project intentionally manages
those hooks elsewhere.

## Result

After this skill succeeds, use `"$TODO_CLI" current`, `"$TODO_CLI" create`,
`"$TODO_CLI" switch`, `"$TODO_CLI" modify-current`, or `"$TODO_CLI" delete`
from the target repository.

