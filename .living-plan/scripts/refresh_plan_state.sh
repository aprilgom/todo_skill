#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
config_path="${LIVING_PLAN_CONFIG:-$repo_root/.living-plan/living-plan.env}"

if [[ -f "$config_path" ]]; then
	# shellcheck disable=SC1090
	source "$config_path"
fi

todo_cmd="${TODO_COMMAND:-todo}"

"$todo_cmd" refresh-state --root "$repo_root"
