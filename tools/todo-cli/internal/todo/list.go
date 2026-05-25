package todo

func List(root string) (ListResult, error) {
	paths, err := loadPlanPaths(root)
	if err != nil {
		return ListResult{}, err
	}
	freshness, err := CheckFreshness(root)
	if err != nil {
		return ListResult{}, err
	}

	currentRefs, err := readActionRefs(paths.current)
	if err != nil {
		return ListResult{}, err
	}
	backlogRefs, err := readActionRefs(paths.backlog)
	if err != nil {
		return ListResult{}, err
	}
	completedRefs, err := readActionRefs(paths.completed)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Current:   listItems(currentRefs),
		Backlog:   listItems(backlogRefs),
		Completed: listItems(completedRefs),
		Freshness: freshness,
	}, nil
}

func listItems(refs []actionRef) []ListItem {
	items := make([]ListItem, 0, len(refs))
	for _, ref := range refs {
		items = append(items, ListItem{
			Title:  ref.Title,
			Link:   ref.Link,
			Status: ref.Status,
		})
	}
	return items
}
