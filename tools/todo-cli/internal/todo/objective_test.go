package todo

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCreatesTodoLivingPlanGate(t *testing.T) {
	root := newGitRepo(t)
	storeRoot := filepath.Join(t.TempDir(), "skills", "todo-plans")

	result, err := Init(root, InitOptions{
		Scope:         "sample",
		StoreRoot:     storeRoot,
		SensitivePath: ".",
		AgentLinkPath: "AGENTS.md",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	assertContains(t, readFile(t, result.ConfigPath), `PLAN_KIND="todo-action-plan"`)
	assertContains(t, readFile(t, result.ConfigPath), `PLAN_PATH="`+result.PlanPath+`"`)
	projectDir := filepath.Join(storeRoot, filepath.Base(root))

	for _, path := range []string{
		result.PlanPath,
		filepath.Join(projectDir, "current.md"),
		filepath.Join(projectDir, "backlog.md"),
		filepath.Join(projectDir, "completed.md"),
		filepath.Join(projectDir, "action"),
		filepath.Join(projectDir, "completed"),
		filepath.Join(projectDir, "todo.bleve"),
		filepath.Join(root, ".living-plan", "scripts", "check_plan_freshness.sh"),
		filepath.Join(root, ".living-plan", "scripts", "refresh_plan_state.sh"),
		filepath.Join(root, ".living-plan", "scripts", "user_prompt_plan_gate.sh"),
		filepath.Join(root, ".living-plan", "scripts", "pre_commit_plan_gate.sh"),
		filepath.Join(root, ".githooks", "pre-commit"),
		filepath.Join(root, ".codex", "config.toml"),
		filepath.Join(root, "AGENTS.md"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}

	assertContains(t, readFile(t, result.PlanPath), "[Current](current.md)")
	assertNotContains(t, readFile(t, result.PlanPath), "decision")
	assertContains(t, readFile(t, filepath.Join(projectDir, "current.md")), "# Current")
	assertContains(t, readFile(t, filepath.Join(projectDir, "backlog.md")), "# Backlog")
	assertContains(t, readFile(t, filepath.Join(root, "AGENTS.md")), result.PlanPath)
	assertContains(t, readFile(t, filepath.Join(root, ".codex/config.toml")), ".living-plan/scripts/user_prompt_plan_gate.sh")
	assertContains(t, readFile(t, filepath.Join(root, ".living-plan", "scripts", "check_plan_freshness.sh")), "check-freshness --root")
	assertContains(t, readFile(t, filepath.Join(root, ".living-plan", "scripts", "refresh_plan_state.sh")), "refresh-state --root")
	assertContains(t, gitConfig(t, root, "core.hooksPath"), ".githooks")

	output, err := CheckFreshness(root)
	if err != nil {
		t.Fatalf("freshness check failed: %v\n%s", err, output)
	}
	assertContains(t, output, "FRESH:")
	if _, err := os.Stat(filepath.Join(projectDir, "decision.md")); !os.IsNotExist(err) {
		t.Fatalf("decision.md should not exist: %v", err)
	}
}

func TestInitSucceedsInUnbornGitRepository(t *testing.T) {
	root := newUnbornGitRepo(t)
	storeRoot := filepath.Join(t.TempDir(), "skills", "todo-plans")

	result, err := Init(root, InitOptions{
		Scope:         "sample",
		StoreRoot:     storeRoot,
		SensitivePath: ".",
		AgentLinkPath: "AGENTS.md",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	output, err := CheckFreshness(root)
	if err != nil {
		t.Fatalf("freshness check failed: %v\n%s", err, output)
	}
	assertContains(t, output, "FRESH:")
	assertContains(t, readFile(t, result.StatePath), `"plan_base_ref": "UNBORN"`)
}

func TestInitScriptsUseTodoCommandFromLivingPlanEnv(t *testing.T) {
	root := newGitRepo(t)
	storeRoot := filepath.Join(t.TempDir(), "skills", "todo-plans")
	todoCommand := buildTodoCLI(t)
	t.Setenv("TODO_COMMAND", todoCommand)

	result, err := Init(root, InitOptions{
		Scope:         "sample",
		StoreRoot:     storeRoot,
		SensitivePath: ".",
		AgentLinkPath: "AGENTS.md",
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	env := readFile(t, result.ConfigPath)
	assertContains(t, env, `TODO_COMMAND="`)

	cmd := exec.Command(filepath.Join(root, ".living-plan", "scripts", "check_plan_freshness.sh"))
	cmd.Dir = root
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check_plan_freshness.sh failed: %v\n%s", err, output)
	}
	assertContains(t, string(output), "FRESH:")
}

func TestCreateActionAddsDetailAndBacklogRow(t *testing.T) {
	root := newPlanFixture(t)

	result, err := Create(root, CreateOptions{
		Title:     "Add Go todo CLI",
		Evidence:  "User wants deterministic plan edits without prompt-heavy skill flows.",
		NextCheck: "Run todo create against a sample living plan.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if result.DetailFile != filepath.Join(root, "plan", "action", "003-add-go-todo-cli.md") {
		t.Fatalf("DetailFile = %q", result.DetailFile)
	}

	backlog := readFile(t, filepath.Join(root, "plan", "backlog.md"))
	assertContains(t, backlog, "- [Add Go todo CLI](action/003-add-go-todo-cli.md) - `not_started`")

	detail := readFile(t, result.DetailFile)
	assertContains(t, detail, "# Add Go todo CLI")
	assertContains(t, detail, "- rank: 003")
	assertContains(t, detail, "- status: not_started")
	assertContains(t, detail, "User wants deterministic plan edits without prompt-heavy skill flows.")
	assertContains(t, detail, "Run todo create against a sample living plan.")
}

func TestCreateRejectsEmptyTitle(t *testing.T) {
	root := newPlanFixture(t)

	_, err := Create(root, CreateOptions{Title: "   "})
	if err == nil {
		t.Fatal("Create() error = nil, want title validation error")
	}
	assertContains(t, err.Error(), "title is required")
}

func TestSwitchCurrentPromotesBacklogAndDemotesCurrent(t *testing.T) {
	root := newPlanFixture(t)

	result, err := Switch(root, SwitchOptions{Target: "002-next-task"})
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}

	if result.PromotedTitle != "Next Task" {
		t.Fatalf("PromotedTitle = %q", result.PromotedTitle)
	}
	if result.DemotedTitle != "Current Task" {
		t.Fatalf("DemotedTitle = %q", result.DemotedTitle)
	}

	current := readFile(t, filepath.Join(root, "plan", "current.md"))
	if strings.Count(current, "](action/") != 1 {
		t.Fatalf("current.md should contain exactly one action link:\n%s", current)
	}
	assertContains(t, current, "- [Next Task](action/002-next-task.md) - `in_progress`")

	backlog := readFile(t, filepath.Join(root, "plan", "backlog.md"))
	assertContains(t, backlog, "- [Current Task](action/001-current-task.md) - `not_started`")
	assertNotContains(t, backlog, "002-next-task.md")

	assertContains(t, readFile(t, filepath.Join(root, "plan", "action", "001-current-task.md")), "- status: not_started")
	assertContains(t, readFile(t, filepath.Join(root, "plan", "action", "002-next-task.md")), "- status: in_progress")
}

func TestSwitchCurrentPromotesBacklogWhenCurrentIsEmpty(t *testing.T) {
	root := newPlanFixture(t)
	writeFile(t, filepath.Join(root, "plan", "current.md"), "# Current\n\n")

	result, err := Switch(root, SwitchOptions{Target: "002-next-task"})
	if err != nil {
		t.Fatalf("Switch() error = %v", err)
	}

	if result.PromotedTitle != "Next Task" {
		t.Fatalf("PromotedTitle = %q", result.PromotedTitle)
	}
	if result.DemotedTitle != "" {
		t.Fatalf("DemotedTitle = %q, want empty title", result.DemotedTitle)
	}

	current := readFile(t, filepath.Join(root, "plan", "current.md"))
	assertContains(t, current, "- [Next Task](action/002-next-task.md) - `in_progress`")

	backlog := readFile(t, filepath.Join(root, "plan", "backlog.md"))
	assertNotContains(t, backlog, "002-next-task.md")
	assertContains(t, readFile(t, filepath.Join(root, "plan", "action", "002-next-task.md")), "- status: in_progress")
}

func TestSwitchRejectsInvalidDemoteStatus(t *testing.T) {
	root := newPlanFixture(t)

	_, err := Switch(root, SwitchOptions{
		Target:       "002-next-task",
		DemoteStatus: "completed",
	})
	if err == nil {
		t.Fatal("Switch() error = nil, want demote status validation error")
	}
	assertContains(t, err.Error(), "demote status must be not_started or paused")
}

func TestModifyCurrentUpdatesLinkedDetailAndCurrentTitle(t *testing.T) {
	root := newPlanFixture(t)

	result, err := ModifyCurrent(root, ModifyCurrentOptions{
		Title:     "Current Task Refined",
		Evidence:  "Refined evidence from the user.",
		NextCheck: "Verify the refined target.",
	})
	if err != nil {
		t.Fatalf("ModifyCurrent() error = %v", err)
	}

	if result.DetailFile != filepath.Join(root, "plan", "action", "001-current-task.md") {
		t.Fatalf("DetailFile = %q", result.DetailFile)
	}

	current := readFile(t, filepath.Join(root, "plan", "current.md"))
	assertContains(t, current, "- [Current Task Refined](action/001-current-task.md) - `in_progress`")

	detail := readFile(t, result.DetailFile)
	assertContains(t, detail, "# Current Task Refined")
	assertContains(t, detail, "- status: in_progress")
	assertContains(t, detail, "Refined evidence from the user.")
	assertContains(t, detail, "Verify the refined target.")
}

func TestDeleteActionRemovesBacklogRowAndDetail(t *testing.T) {
	root := newPlanFixture(t)

	result, err := Delete(root, DeleteOptions{Target: "002-next-task"})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.DeletedTitle != "Next Task" {
		t.Fatalf("DeletedTitle = %q", result.DeletedTitle)
	}
	if result.DetailFile != filepath.Join(root, "plan", "action", "002-next-task.md") {
		t.Fatalf("DetailFile = %q", result.DetailFile)
	}

	backlog := readFile(t, filepath.Join(root, "plan", "backlog.md"))
	assertNotContains(t, backlog, "002-next-task.md")
	if _, err := os.Stat(result.DetailFile); !os.IsNotExist(err) {
		t.Fatalf("deleted detail still exists or stat failed unexpectedly: %v", err)
	}

	current := readFile(t, filepath.Join(root, "plan", "current.md"))
	assertContains(t, current, "001-current-task.md")
}

func TestDeleteBacklogActionAllowsEmptyCurrent(t *testing.T) {
	root := newPlanFixture(t)
	writeFile(t, filepath.Join(root, "plan", "current.md"), "# Current\n")

	result, err := Delete(root, DeleteOptions{Target: "002-next-task"})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.DeletedTitle != "Next Task" {
		t.Fatalf("DeletedTitle = %q", result.DeletedTitle)
	}
	if result.DetailFile != filepath.Join(root, "plan", "action", "002-next-task.md") {
		t.Fatalf("DetailFile = %q", result.DetailFile)
	}

	backlog := readFile(t, filepath.Join(root, "plan", "backlog.md"))
	assertNotContains(t, backlog, "002-next-task.md")
	if _, err := os.Stat(result.DetailFile); !os.IsNotExist(err) {
		t.Fatalf("deleted detail still exists or stat failed unexpectedly: %v", err)
	}
}

func TestDeleteCurrentActionRemovesCurrentRowAndDetail(t *testing.T) {
	root := newPlanFixture(t)

	result, err := Delete(root, DeleteOptions{Target: "001-current-task"})
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	if result.DeletedTitle != "Current Task" {
		t.Fatalf("DeletedTitle = %q", result.DeletedTitle)
	}
	if result.DetailFile != filepath.Join(root, "plan", "action", "001-current-task.md") {
		t.Fatalf("DetailFile = %q", result.DetailFile)
	}

	current := readFile(t, filepath.Join(root, "plan", "current.md"))
	assertNotContains(t, current, "001-current-task.md")
	backlog := readFile(t, filepath.Join(root, "plan", "backlog.md"))
	assertNotContains(t, backlog, "001-current-task.md")
	if _, err := os.Stat(result.DetailFile); !os.IsNotExist(err) {
		t.Fatalf("deleted detail still exists or stat failed unexpectedly: %v", err)
	}
}

func TestCurrentReturnsActiveAction(t *testing.T) {
	root := newPlanFixture(t)

	result, err := Current(root, CurrentOptions{})
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if result.Title != "Current Task" {
		t.Fatalf("Title = %q", result.Title)
	}
	if result.DetailFile != filepath.Join(root, "plan", "action", "001-current-task.md") {
		t.Fatalf("DetailFile = %q", result.DetailFile)
	}
	if result.Completed {
		t.Fatal("Completed = true, want false")
	}
}

func TestCurrentCompleteMovesActiveActionToCompleted(t *testing.T) {
	root := newPlanFixture(t)

	result, err := Current(root, CurrentOptions{
		Complete: true,
		Evidence: "Implemented the current todo CLI flow.",
		Date:     "2026-05-25",
	})
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}

	if !result.Completed {
		t.Fatal("Completed = false, want true")
	}
	if result.Title != "Current Task" {
		t.Fatalf("Title = %q", result.Title)
	}

	current := readFile(t, filepath.Join(root, "plan", "current.md"))
	assertNotContains(t, current, "001-current-task.md")

	completed := readFile(t, filepath.Join(root, "plan", "completed.md"))
	assertContains(t, completed, "- [Current Task](action/001-current-task.md) - `completed`")

	evidence := readFile(t, filepath.Join(root, "plan", "completed", "2026-05-25.md"))
	assertContains(t, evidence, "## Current Task")
	assertContains(t, evidence, "Implemented the current todo CLI flow.")
}

func TestListReturnsCurrentBacklogCompletedAndFreshness(t *testing.T) {
	root := newPlanFixture(t)
	writeFile(t, filepath.Join(root, "plan", "completed.md"), "# Completed\n\n- [Finished Task](action/003-finished-task.md) - `completed`\n")
	writeFile(t, filepath.Join(root, "plan", "action", "003-finished-task.md"), "# Finished Task\n\n- rank: 003\n- status: completed\n")

	result, err := List(root)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(result.Current) != 1 || result.Current[0].Title != "Current Task" {
		t.Fatalf("Current = %#v, want Current Task", result.Current)
	}
	if len(result.Backlog) != 1 || result.Backlog[0].Title != "Next Task" {
		t.Fatalf("Backlog = %#v, want Next Task", result.Backlog)
	}
	if len(result.Completed) != 1 || result.Completed[0].Title != "Finished Task" {
		t.Fatalf("Completed = %#v, want Finished Task", result.Completed)
	}
	assertContains(t, result.Freshness, "FRESH")
}

func TestCurrentCompleteRequiresEvidence(t *testing.T) {
	root := newPlanFixture(t)

	_, err := Current(root, CurrentOptions{Complete: true, Evidence: "   "})
	if err == nil {
		t.Fatal("Current(...Complete) error = nil, want evidence validation error")
	}
	assertContains(t, err.Error(), "completion evidence is required")
}

func newPlanFixture(t *testing.T) string {
	t.Helper()

	root := newGitRepo(t)
	mustMkdir(t, filepath.Join(root, ".living-plan", "scripts"))
	mustMkdir(t, filepath.Join(root, "plan", "action"))
	mustMkdir(t, filepath.Join(root, "plan", "completed"))

	writeFile(t, filepath.Join(root, ".living-plan", "living-plan.env"), strings.Join([]string{
		`PLAN_SCOPE="test"`,
		`PLAN_KIND="todo-action-plan"`,
		`PLAN_PATH="` + filepath.Join(root, "plan", "plan.md") + `"`,
		`STATE_PATH="` + filepath.Join(root, "plan", "test-todo-action-plan.state.json") + `"`,
		`SENSITIVE_PATH="."`,
		"",
	}, "\n"))
	writeFile(t, filepath.Join(root, ".living-plan", "scripts", "check_plan_freshness.sh"), "#!/bin/sh\necho FRESH\n")
	writeFile(t, filepath.Join(root, ".living-plan", "scripts", "refresh_plan_state.sh"), "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(root, ".living-plan", "scripts", "check_plan_freshness.sh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(filepath.Join(root, ".living-plan", "scripts", "refresh_plan_state.sh"), 0o755); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(root, "plan", "plan.md"), "# Plan\n")
	writeFile(t, filepath.Join(root, "plan", "current.md"), "# Current\n\n- [Current Task](action/001-current-task.md) - `in_progress`\n")
	writeFile(t, filepath.Join(root, "plan", "backlog.md"), "# Backlog\n\n- [Next Task](action/002-next-task.md) - `not_started`\n")
	writeFile(t, filepath.Join(root, "plan", "completed.md"), "# Completed\n")
	writeFile(t, filepath.Join(root, "plan", "completed", "2026-05-25.md"), "# Completed 2026-05-25\n")
	writeFile(t, filepath.Join(root, "plan", "action", "001-current-task.md"), "# Current Task\n\n- rank: 001\n- status: in_progress\n\n## Evidence\n\nStarted.\n\n## Next Check\n\nContinue.\n")
	writeFile(t, filepath.Join(root, "plan", "action", "002-next-task.md"), "# Next Task\n\n- rank: 002\n- status: not_started\n\n## Evidence\n\nQueued.\n\n## Next Check\n\nStart.\n")
	if _, err := RefreshPlanState(root); err != nil {
		t.Fatal(err)
	}
	if err := Reindex(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func newGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(root, "README.md"), "# Test\n")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "init")
	return root
}

func newUnbornGitRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	return root
}

func buildTodoCLI(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "todo")
	cmd := exec.Command("go", "build", "-o", path, "../../cmd/todo")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build todo CLI failed: %v\n%s", err, output)
	}
	return path
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}

func gitConfig(t *testing.T, root, key string) string {
	t.Helper()
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git config --get %s failed: %v\n%s", key, err, output)
	}
	return strings.TrimSpace(string(output))
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected content to contain %q:\n%s", want, got)
	}
}

func assertNotContains(t *testing.T, got, want string) {
	t.Helper()
	if strings.Contains(got, want) {
		t.Fatalf("expected content not to contain %q:\n%s", want, got)
	}
}
