package todo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func Current(root string, opts CurrentOptions) (CurrentResult, error) {
	paths, err := loadPlanPaths(root)
	if err != nil {
		return CurrentResult{}, err
	}
	if err := runFreshness(paths); err != nil {
		return CurrentResult{}, err
	}

	current, err := currentAction(paths.current)
	if err != nil {
		return CurrentResult{}, err
	}
	result := CurrentResult{
		Title:      current.Title,
		DetailFile: detailPath(paths, current),
	}
	if !opts.Complete {
		return result, nil
	}
	if strings.TrimSpace(opts.Evidence) == "" {
		return CurrentResult{}, errors.New("completion evidence is required")
	}

	if err := completeCurrent(paths, current, opts); err != nil {
		return CurrentResult{}, err
	}
	result.Completed = true
	if err := refreshAndCheck(paths); err != nil {
		return CurrentResult{}, err
	}
	return result, nil
}

func completeCurrent(paths planPaths, current actionRef, opts CurrentOptions) error {
	date := strings.TrimSpace(opts.Date)
	if date == "" {
		date = time.Now().Format(time.DateOnly)
	}

	if err := os.WriteFile(paths.current, []byte("# Current\n\n"), 0o644); err != nil {
		return err
	}
	row := fmt.Sprintf("- [%s](%s) - `completed`\n", current.Title, current.Link)
	if err := appendCompletedRow(paths.completed, row); err != nil {
		return err
	}
	if err := os.MkdirAll(paths.completedDir, 0o755); err != nil {
		return err
	}
	evidencePath := filepath.Join(paths.completedDir, date+".md")
	entry := fmt.Sprintf("\n## %s\n\n- action: [%s](../%s)\n- completed: %s\n\n%s\n", current.Title, current.Title, current.Link, date, opts.Evidence)
	return appendFile(evidencePath, "# Completed "+date+"\n", entry)
}

func appendCompletedRow(path, row string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte("# Completed\n\n"), 0o644); err != nil {
			return err
		}
	}
	return appendBacklogRow(path, row)
}

func appendFile(path, initial, extra string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(extra)
	return err
}
