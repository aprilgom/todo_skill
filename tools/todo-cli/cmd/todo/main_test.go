package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"todo-skill/internal/todo"
)

func TestRunCreatePrintsStructuredVerification(t *testing.T) {
	root := newCLIPlanFixture(t)

	output := captureStdout(t, func() {
		if err := run([]string{
			"create",
			"--root", root,
			"--title", "Add Structured Output",
			"--evidence", "Operators need easy verification.",
			"--next-check", "Run the command and inspect stdout.",
		}); err != nil {
			t.Fatalf("run(create) error = %v", err)
		}
	})

	assertOutputContains(t, output, "action: Add Structured Output")
	assertOutputContains(t, output, "detail: "+filepath.Join(root, "plan", "action", "003-add-structured-output.md"))
	assertOutputContains(t, output, "freshness: FRESH")
}

func TestRunListPrintsTodoState(t *testing.T) {
	root := newCLIPlanFixture(t)
	writeFile(t, filepath.Join(root, "plan", "completed.md"), "# Completed\n\n- [Finished Task](action/003-finished-task.md) - `completed`\n")

	output := captureStdout(t, func() {
		if err := run([]string{"list", "--root", root}); err != nil {
			t.Fatalf("run(list) error = %v", err)
		}
	})

	assertOutputContains(t, output, "status: list")
	assertOutputContains(t, output, "current:")
	assertOutputContains(t, output, "- `in_progress` Current Task")
	assertOutputContains(t, output, "backlog:")
	assertOutputContains(t, output, "- `not_started` Next Task")
	assertOutputContains(t, output, "completed:")
	assertOutputContains(t, output, "- `completed` Finished Task")
	assertOutputContains(t, output, "freshness: FRESH")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer

	fn()

	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	os.Stdout = old

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func newCLIPlanFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	writeFile(t, filepath.Join(root, "README.md"), "# Test\n")
	runGit(t, root, "add", "README.md")
	runGit(t, root, "commit", "-m", "init")

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
	if _, err := todo.RefreshPlanState(root); err != nil {
		t.Fatal(err)
	}
	if err := todo.Reindex(root); err != nil {
		t.Fatal(err)
	}
	return root
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
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

func assertOutputContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected output to contain %q:\n%s", want, got)
	}
}
