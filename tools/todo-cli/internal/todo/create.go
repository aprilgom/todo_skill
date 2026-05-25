package todo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Create(root string, opts CreateOptions) (CreateResult, error) {
	if strings.TrimSpace(opts.Title) == "" {
		return CreateResult{}, errors.New("title is required")
	}
	if strings.TrimSpace(opts.Evidence) == "" {
		opts.Evidence = "Needs evidence."
	}
	if strings.TrimSpace(opts.NextCheck) == "" {
		opts.NextCheck = "Review this action and define the next concrete check."
	}

	paths, err := loadPlanPaths(root)
	if err != nil {
		return CreateResult{}, err
	}
	if err := runFreshness(paths); err != nil {
		return CreateResult{}, err
	}

	rank, err := nextRank(paths.actionDir)
	if err != nil {
		return CreateResult{}, err
	}
	slug := slugify(opts.Title)
	if slug == "" {
		return CreateResult{}, fmt.Errorf("title %q does not produce a usable slug", opts.Title)
	}

	fileName := rank + "-" + slug + ".md"
	detailPath := filepath.Join(paths.actionDir, fileName)
	if _, err := os.Stat(detailPath); err == nil {
		return CreateResult{}, fmt.Errorf("action detail already exists: %s", detailPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return CreateResult{}, err
	}

	detail := fmt.Sprintf("# %s\n\n- rank: %s\n- status: not_started\n\n## Evidence\n\n%s\n\n## Next Check\n\n%s\n", opts.Title, rank, opts.Evidence, opts.NextCheck)
	if err := os.WriteFile(detailPath, []byte(detail), 0o644); err != nil {
		return CreateResult{}, err
	}

	row := fmt.Sprintf("- [%s](action/%s) - `not_started`\n", opts.Title, fileName)
	if err := appendBacklogRow(paths.backlog, row); err != nil {
		return CreateResult{}, err
	}

	if err := refreshAndCheck(paths); err != nil {
		return CreateResult{}, err
	}

	return CreateResult{DetailFile: detailPath, Title: opts.Title, Rank: rank}, nil
}
