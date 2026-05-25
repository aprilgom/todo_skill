package todo

import (
	"errors"
	"strings"
)

func ModifyCurrent(root string, opts ModifyCurrentOptions) (ModifyCurrentResult, error) {
	if strings.TrimSpace(opts.Title) == "" && strings.TrimSpace(opts.Evidence) == "" && strings.TrimSpace(opts.NextCheck) == "" {
		return ModifyCurrentResult{}, errors.New("at least one of title, evidence, or next-check is required")
	}

	paths, err := loadPlanPaths(root)
	if err != nil {
		return ModifyCurrentResult{}, err
	}
	if err := runFreshness(paths); err != nil {
		return ModifyCurrentResult{}, err
	}

	current, err := currentAction(paths.current)
	if err != nil {
		return ModifyCurrentResult{}, err
	}
	current.Status = "in_progress"
	detail := detailPath(paths, current)

	if title := strings.TrimSpace(opts.Title); title != "" {
		current.Title = title
		if err := setDetailTitle(detail, title); err != nil {
			return ModifyCurrentResult{}, err
		}
		if err := setCurrent(paths.current, current); err != nil {
			return ModifyCurrentResult{}, err
		}
	}
	if evidence := strings.TrimSpace(opts.Evidence); evidence != "" {
		if err := replaceDetailSection(detail, "## Evidence", evidence); err != nil {
			return ModifyCurrentResult{}, err
		}
	}
	if nextCheck := strings.TrimSpace(opts.NextCheck); nextCheck != "" {
		if err := replaceDetailSection(detail, "## Next Check", nextCheck); err != nil {
			return ModifyCurrentResult{}, err
		}
	}
	if err := setDetailStatus(detail, "in_progress"); err != nil {
		return ModifyCurrentResult{}, err
	}

	if err := refreshAndCheck(paths); err != nil {
		return ModifyCurrentResult{}, err
	}

	return ModifyCurrentResult{DetailFile: detail, Title: current.Title}, nil
}
