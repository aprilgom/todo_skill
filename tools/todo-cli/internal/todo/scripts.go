package todo

const checkPlanFreshnessScript = `#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
config_path="${LIVING_PLAN_CONFIG:-$repo_root/.living-plan/living-plan.env}"

if [[ -f "$config_path" ]]; then
	# shellcheck disable=SC1090
	source "$config_path"
fi

todo_cmd="${TODO_COMMAND:-todo}"

"$todo_cmd" check-freshness --root "$repo_root"
`

const refreshPlanStateScript = `#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
config_path="${LIVING_PLAN_CONFIG:-$repo_root/.living-plan/living-plan.env}"

if [[ -f "$config_path" ]]; then
	# shellcheck disable=SC1090
	source "$config_path"
fi

todo_cmd="${TODO_COMMAND:-todo}"

"$todo_cmd" refresh-state --root "$repo_root"
`

const userPromptPlanGateScript = `#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
config_path="${LIVING_PLAN_CONFIG:-$repo_root/.living-plan/living-plan.env}"

if [[ ! -f "$config_path" ]]; then
	printf '{"continue":true}\n'
	exit 0
fi

# shellcheck disable=SC1090
source "$config_path"
todo_cmd="${TODO_COMMAND:-todo}"

prompt="$(cat)"
scope="${PLAN_SCOPE:-}"
sensitive="${SENSITIVE_PATH:-}"
plan_path="${PLAN_PATH:-}"
plan_dir="$(dirname "$plan_path")"
goal_context=""

if printf '%s' "$prompt" | grep -Eiq '(^|[[:space:]])/goal[[:space:]]+(todo-current|todo:current)($|[[:space:]])'; then
	goal_context="$(cat <<EOF
todo goal alias: use $plan_dir/current.md as the only active goal pointer and read its linked detail. States: current=in_progress; backlog=not_started unless genuinely paused. Done: use todo current --complete --evidence. To revise current action content, use todo modify-current. Refresh state after plan edits.
EOF
)"
fi

if [[ -n "$scope" || -n "$sensitive" ]]; then
	if [[ -z "$goal_context" ]] && ! printf '%s' "$prompt" | grep -Eiq "(${scope}|${sensitive}|living plan|todo plan|migration plan|roadmap|action plan)"; then
		printf '{"continue":true}\n'
		exit 0
	fi
fi

if output="$("$todo_cmd" check-freshness --root "$repo_root" 2>&1)"; then
	python3 - "$output" "$goal_context" <<'PY'
import json
import sys
context = "Living plan freshness check passed. " + sys.argv[1]
if sys.argv[2]:
    context += "\n\n" + sys.argv[2]
print(json.dumps({
    "continue": True,
    "hookSpecificOutput": {
        "hookEventName": "UserPromptSubmit",
        "additionalContext": context,
    },
}))
PY
else
	python3 - "$output" <<'PY'
import json
import sys
print(json.dumps({
    "decision": "block",
    "reason": "Living plan is stale or missing:\n" + sys.argv[1],
}))
PY
	exit 0
fi
`

const preCommitPlanGateScript = `#!/usr/bin/env bash
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
`
