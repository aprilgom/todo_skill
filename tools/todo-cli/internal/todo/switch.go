package todo

import (
	"errors"
	"fmt"
	"strings"
)

func Switch(root string, opts SwitchOptions) (SwitchResult, error) {
	if strings.TrimSpace(opts.Target) == "" {
		return SwitchResult{}, errors.New("target is required")
	}
	demoteStatus := strings.TrimSpace(opts.DemoteStatus)
	if demoteStatus == "" {
		demoteStatus = "not_started"
	}
	if demoteStatus != "not_started" && demoteStatus != "paused" {
		return SwitchResult{}, errors.New("demote status must be not_started or paused")
	}

	paths, err := loadPlanPaths(root)
	if err != nil {
		return SwitchResult{}, err
	}
	if err := runFreshness(paths); err != nil {
		return SwitchResult{}, err
	}

	currentRefs, err := readActionRefs(paths.current)
	if err != nil {
		return SwitchResult{}, err
	}
	if len(currentRefs) > 1 {
		return SwitchResult{}, fmt.Errorf("current.md must contain at most one action link, found %d", len(currentRefs))
	}
	backlogRefs, err := readActionRefs(paths.backlog)
	if err != nil {
		return SwitchResult{}, err
	}
	promoted, ok := findAction(backlogRefs, opts.Target)
	if !ok {
		return SwitchResult{}, fmt.Errorf("target action not found in backlog: %s", opts.Target)
	}

	if err := removeActionLink(paths.backlog, promoted.Link); err != nil {
		return SwitchResult{}, err
	}
	var demotedTitle string
	if len(currentRefs) == 1 {
		current := currentRefs[0]
		demotedTitle = current.Title
		demotedRow := fmt.Sprintf("- [%s](%s) - `%s`\n", current.Title, current.Link, demoteStatus)
		if err := appendBacklogRow(paths.backlog, demotedRow); err != nil {
			return SwitchResult{}, err
		}
		if err := setDetailStatus(detailPath(paths, current), demoteStatus); err != nil {
			return SwitchResult{}, err
		}
	}

	promoted.Status = "in_progress"
	if err := setCurrent(paths.current, promoted); err != nil {
		return SwitchResult{}, err
	}
	if err := setDetailStatus(detailPath(paths, promoted), "in_progress"); err != nil {
		return SwitchResult{}, err
	}

	if err := refreshAndCheck(paths); err != nil {
		return SwitchResult{}, err
	}

	return SwitchResult{
		PromotedTitle: promoted.Title,
		DemotedTitle:  demotedTitle,
		CurrentFile:   paths.current,
		BacklogFile:   paths.backlog,
	}, nil
}
