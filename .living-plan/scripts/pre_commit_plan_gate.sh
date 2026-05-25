#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"
config_path="${LIVING_PLAN_CONFIG:-$repo_root/.living-plan/living-plan.env}"

if [[ ! -f "$config_path" ]]; then
	echo "living plan config missing: $config_path" >&2
	exit 2
fi

# shellcheck disable=SC1090
source "$config_path"
todo_cmd="${TODO_COMMAND:-todo}"

SENSITIVE_PATH="${SENSITIVE_PATH:?SENSITIVE_PATH is required}"

cd "$repo_root"

staged_sensitive="$(git diff --cached --name-only -- "$SENSITIVE_PATH")"
if [[ -z "$staged_sensitive" ]]; then
	exit 0
fi

if ! "$todo_cmd" check-freshness --root "$repo_root" >/tmp/living-plan-precommit.out 2>&1; then
	cat /tmp/living-plan-precommit.out >&2
	echo "Refresh the plan state before committing." >&2
	echo "Run: $todo_cmd refresh-state --root $repo_root" >&2
	exit 1
fi
