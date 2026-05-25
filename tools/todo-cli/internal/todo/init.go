package todo

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const todoPlanKind = "todo-action-plan"

func Init(root string, opts InitOptions) (InitResult, error) {
	if root == "" {
		root = "."
	}
	repoRoot, err := resolveRepoRoot(root)
	if err != nil {
		return InitResult{}, err
	}
	opts = defaultInitOptions(opts)
	if strings.TrimSpace(opts.Scope) == "" {
		return InitResult{}, errors.New("scope is required")
	}
	planPath, statePath, err := resolvePlanStorage(repoRoot, opts)
	if err != nil {
		return InitResult{}, err
	}

	installDir := filepath.Join(repoRoot, ".living-plan")
	scriptDir := filepath.Join(installDir, "scripts")

	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		return InitResult{}, err
	}
	todoCommand := os.Getenv("TODO_COMMAND")
	if todoCommand == "" {
		todoCommand, err = os.Executable()
		if err != nil {
			todoCommand = "todo"
		}
	}
	if err := writeLivingPlanEnv(filepath.Join(installDir, "living-plan.env"), opts, planPath, statePath, todoCommand); err != nil {
		return InitResult{}, err
	}
	if err := writeProjectScripts(scriptDir); err != nil {
		return InitResult{}, err
	}
	if err := createTodoPlanFiles(opts, planPath, statePath); err != nil {
		return InitResult{}, err
	}
	if err := appendAgentLink(filepath.Join(repoRoot, opts.AgentLinkPath), planPath); err != nil {
		return InitResult{}, err
	}
	if !opts.NoGitHook {
		if err := installGitHook(repoRoot); err != nil {
			return InitResult{}, err
		}
	}
	if !opts.NoCodexHook {
		if err := installCodexHook(filepath.Join(repoRoot, ".codex", "config.toml")); err != nil {
			return InitResult{}, err
		}
	}
	if _, err := RefreshPlanState(repoRoot); err != nil {
		return InitResult{}, err
	}
	if err := Reindex(repoRoot); err != nil {
		return InitResult{}, err
	}

	return InitResult{
		ConfigPath: filepath.Join(installDir, "living-plan.env"),
		PlanPath:   planPath,
		StatePath:  statePath,
	}, nil
}

func defaultInitOptions(opts InitOptions) InitOptions {
	if opts.Scope == "" {
		opts.Scope = "todo"
	}
	if opts.SensitivePath == "" {
		opts.SensitivePath = "."
	}
	if opts.AgentLinkPath == "" {
		opts.AgentLinkPath = "AGENTS.md"
	}
	return opts
}

func resolvePlanStorage(repoRoot string, opts InitOptions) (string, string, error) {
	storeRoot := opts.StoreRoot
	if storeRoot == "" {
		storeRoot = defaultSkillTodoRoot()
	}
	if !filepath.IsAbs(storeRoot) {
		storeRoot = filepath.Join(repoRoot, storeRoot)
	}
	projectDir := filepath.Join(storeRoot, slugify(filepath.Base(repoRoot)))
	return filepath.Join(projectDir, "action-plan.md"), filepath.Join(projectDir, opts.Scope+"-"+todoPlanKind+".state.json"), nil
}

func defaultSkillTodoRoot() string {
	if codexHome := os.Getenv("CODEX_HOME"); codexHome != "" {
		return filepath.Join(codexHome, "skills", "todo-plans")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codex", "skills", "todo-plans")
	}
	return filepath.Join(".codex", "skills", "todo-plans")
}

func resolveRepoRoot(root string) (string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = absRoot
	output, err := cmd.Output()
	if err != nil {
		return absRoot, nil
	}
	return strings.TrimSpace(string(output)), nil
}

func writeLivingPlanEnv(path string, opts InitOptions, planPath string, statePath string, todoCommand string) error {
	content := fmt.Sprintf(`PLAN_SCOPE="%s"
PLAN_KIND="%s"
PLAN_PATH="%s"
STATE_PATH="%s"
SENSITIVE_PATH="%s"
TODO_COMMAND="%s"
`, opts.Scope, todoPlanKind, filepath.ToSlash(planPath), filepath.ToSlash(statePath), filepath.ToSlash(opts.SensitivePath), filepath.ToSlash(todoCommand))
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeProjectScripts(scriptDir string) error {
	scripts := map[string]string{
		"check_plan_freshness.sh":  checkPlanFreshnessScript,
		"refresh_plan_state.sh":    refreshPlanStateScript,
		"user_prompt_plan_gate.sh": userPromptPlanGateScript,
		"pre_commit_plan_gate.sh":  preCommitPlanGateScript,
	}
	for name, content := range scripts {
		path := filepath.Join(scriptDir, name)
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			return err
		}
	}
	return nil
}

func createTodoPlanFiles(opts InitOptions, planPath string, statePath string) error {
	planDir := filepath.Dir(planPath)
	for _, dir := range []string{
		planDir,
		filepath.Join(planDir, "action"),
		filepath.Join(planDir, "completed"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	files := map[string]string{
		planPath: filepath.ToSlash(fmt.Sprintf(`# %s Todo Action Plan

- scope: %s
- plan_kind: %s
- state_file: %s
- freshness_check: .living-plan/scripts/check_plan_freshness.sh

## Index

- [Current](current.md)
- [Backlog](backlog.md)
- [Completed](completed.md)
`, opts.Scope, opts.Scope, todoPlanKind, statePath)),
		filepath.Join(planDir, "current.md"):   "# Current\n\n",
		filepath.Join(planDir, "backlog.md"):   "# Backlog\n\n",
		filepath.Join(planDir, "completed.md"): "# Completed\n\n",
	}
	for path, content := range files {
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func appendAgentLink(path, planPath string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data = nil
	} else if err != nil {
		return err
	}
	if strings.Contains(string(data), planPath) {
		return nil
	}
	block := fmt.Sprintf("\n## Living Plan\n\nCurrent plan: [%s](%s)\n\nBefore work starts, the project-local living-plan gate must report `FRESH`.\n", planPath, planPath)
	return os.WriteFile(path, append(data, []byte(block)...), 0o644)
}

func installGitHook(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".githooks")
	preCommit := filepath.Join(hooksDir, "pre-commit")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return err
	}
	content := `#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel)"

if [[ -x "$repo_root/.living-plan/scripts/pre_commit_plan_gate.sh" ]]; then
	"$repo_root/.living-plan/scripts/pre_commit_plan_gate.sh"
fi
`
	if _, err := os.Stat(preCommit); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(preCommit, []byte(content), 0o755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	cmd := exec.Command("git", "config", "core.hooksPath", ".githooks")
	cmd.Dir = repoRoot
	return cmd.Run()
}

func installCodexHook(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	hook := `[features]
hooks = true

[[hooks.UserPromptSubmit]]

[[hooks.UserPromptSubmit.hooks]]
type = "command"
command = "bash .living-plan/scripts/user_prompt_plan_gate.sh"
timeout = 10
statusMessage = "Checking living plan freshness"
`
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.WriteFile(path, []byte(hook), 0o644)
	}
	if err != nil {
		return err
	}
	if strings.Contains(string(data), ".living-plan/scripts/user_prompt_plan_gate.sh") {
		return nil
	}
	return os.WriteFile(path, append(append(data, '\n'), []byte(hook)...), 0o644)
}
