package todo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReindexAndSearchTodos(t *testing.T) {
	root := newPlanFixture(t)

	if err := Reindex(root); err != nil {
		t.Fatalf("Reindex() error = %v", err)
	}

	results, err := Search(root, SearchOptions{Query: "Queued", Limit: 10})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("Search() returned %d results, want 1: %#v", len(results), results)
	}
	if results[0].ID != "002-next-task" {
		t.Fatalf("Search()[0].ID = %q, want 002-next-task", results[0].ID)
	}
	if results[0].Status != "not_started" {
		t.Fatalf("Search()[0].Status = %q, want not_started", results[0].Status)
	}

	filtered, err := Search(root, SearchOptions{Query: "Task", Status: "in_progress", Limit: 10})
	if err != nil {
		t.Fatalf("Search() with status error = %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != "001-current-task" {
		t.Fatalf("Search() with status = %#v, want current task only", filtered)
	}
}

func TestCRUDRefreshesBleveIndex(t *testing.T) {
	root := newPlanFixture(t)

	created, err := Create(root, CreateOptions{
		Title:     "Add Bleve Scorch Search",
		Evidence:  "Need fast local full text search.",
		NextCheck: "Run search against the generated index.",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if _, err := os.Stat(TodoIndexPath(root)); err != nil {
		t.Fatalf("expected index at %s: %v", TodoIndexPath(root), err)
	}

	results, err := Search(root, SearchOptions{Query: "Scorch", Status: "not_started", Limit: 10})
	if err != nil {
		t.Fatalf("Search() after create error = %v", err)
	}
	if len(results) != 1 || results[0].Title != "Add Bleve Scorch Search" {
		t.Fatalf("Search() after create = %#v, want created action", results)
	}

	if _, err := Delete(root, DeleteOptions{Target: filepath.Base(created.DetailFile)}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	results, err = Search(root, SearchOptions{Query: "Scorch", Limit: 10})
	if err != nil {
		t.Fatalf("Search() after delete error = %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("Search() after delete = %#v, want no results", results)
	}
}

func TestSwitchModifyAndCompleteRefreshBleveIndex(t *testing.T) {
	root := newPlanFixture(t)

	if _, err := Switch(root, SwitchOptions{Target: "002-next-task"}); err != nil {
		t.Fatalf("Switch() error = %v", err)
	}
	results, err := Search(root, SearchOptions{Query: "Next", Status: "in_progress", Limit: 10})
	if err != nil {
		t.Fatalf("Search() after switch error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "002-next-task" {
		t.Fatalf("Search() after switch = %#v, want promoted next task", results)
	}

	if _, err := ModifyCurrent(root, ModifyCurrentOptions{
		Title:     "Next Task Refined",
		Evidence:  "Refined searchable evidence.",
		NextCheck: "Confirm modified current appears in Bleve.",
	}); err != nil {
		t.Fatalf("ModifyCurrent() error = %v", err)
	}
	results, err = Search(root, SearchOptions{Query: "searchable", Status: "in_progress", Limit: 10})
	if err != nil {
		t.Fatalf("Search() after modify error = %v", err)
	}
	if len(results) != 1 || results[0].Title != "Next Task Refined" {
		t.Fatalf("Search() after modify = %#v, want refined current", results)
	}

	if _, err := Current(root, CurrentOptions{
		Complete: true,
		Evidence: "Completed the refined task.",
		Date:     "2026-05-25",
	}); err != nil {
		t.Fatalf("Current(...Complete) error = %v", err)
	}
	results, err = Search(root, SearchOptions{Query: "Refined", Status: "completed", Limit: 10})
	if err != nil {
		t.Fatalf("Search() after complete error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "002-next-task" {
		t.Fatalf("Search() after complete = %#v, want completed refined task", results)
	}
}
