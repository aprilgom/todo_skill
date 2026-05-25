package todo

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func Delete(root string, opts DeleteOptions) (DeleteResult, error) {
	if strings.TrimSpace(opts.Target) == "" {
		return DeleteResult{}, errors.New("target is required")
	}

	paths, err := loadPlanPaths(root)
	if err != nil {
		return DeleteResult{}, err
	}
	if err := runFreshness(paths); err != nil {
		return DeleteResult{}, err
	}

	currentRefs, err := readActionRefs(paths.current)
	if err != nil {
		return DeleteResult{}, err
	}
	if len(currentRefs) > 1 {
		return DeleteResult{}, fmt.Errorf("current.md must contain at most one action link, found %d", len(currentRefs))
	}
	if len(currentRefs) == 1 && refMatches(currentRefs[0], opts.Target) {
		target := currentRefs[0]
		targetPath := detailPath(paths, target)
		if err := setCurrentEmpty(paths.current); err != nil {
			return DeleteResult{}, err
		}
		if err := os.Remove(targetPath); err != nil {
			return DeleteResult{}, err
		}
		if err := refreshAndCheck(paths); err != nil {
			return DeleteResult{}, err
		}
		return DeleteResult{DeletedTitle: target.Title, DetailFile: targetPath}, nil
	}

	backlogRefs, err := readActionRefs(paths.backlog)
	if err != nil {
		return DeleteResult{}, err
	}
	target, ok := findAction(backlogRefs, opts.Target)
	if !ok {
		return DeleteResult{}, fmt.Errorf("target action not found in backlog: %s", opts.Target)
	}
	targetPath := detailPath(paths, target)

	if err := removeActionLink(paths.backlog, target.Link); err != nil {
		return DeleteResult{}, err
	}
	if err := os.Remove(targetPath); err != nil {
		return DeleteResult{}, err
	}

	if err := refreshAndCheck(paths); err != nil {
		return DeleteResult{}, err
	}

	return DeleteResult{DeletedTitle: target.Title, DetailFile: targetPath}, nil
}

func refMatches(ref actionRef, target string) bool {
	_, ok := findAction([]actionRef{ref}, target)
	return ok
}
