package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"todo-skill/internal/todo"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "todo:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return flag.ErrHelp
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "current":
		return runCurrent(args[1:])
	case "create":
		return runCreate(args[1:])
	case "switch":
		return runSwitch(args[1:])
	case "modify-current":
		return runModifyCurrent(args[1:])
	case "delete":
		return runDelete(args[1:])
	case "list":
		return runList(args[1:])
	case "index":
		return runIndex(args[1:])
	case "search":
		return runSearch(args[1:])
	case "check-freshness":
		return runCheckFreshness(args[1:])
	case "refresh-state":
		return runRefreshState(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root")
	scope := fs.String("scope", "todo", "living plan scope name")
	storeRoot := fs.String("store-root", "", "skill todo storage root")
	sensitivePath := fs.String("sensitive-path", ".", "path or directory guarded by the living plan")
	agentLinkPath := fs.String("agent-link-path", "AGENTS.md", "agent context file to receive the plan link")
	noGitHook := fs.Bool("no-git-hook", false, "skip .githooks/pre-commit installation")
	noCodexHook := fs.Bool("no-codex-hook", false, "skip .codex/config.toml UserPromptSubmit hook installation")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.Init(*root, todo.InitOptions{
		Scope:         *scope,
		StoreRoot:     *storeRoot,
		SensitivePath: *sensitivePath,
		AgentLinkPath: *agentLinkPath,
		NoGitHook:     *noGitHook,
		NoCodexHook:   *noCodexHook,
	})
	if err != nil {
		return err
	}

	fmt.Printf("initialized todo living plan\n  config: %s\n  plan:   %s\n  state:  %s\n", result.ConfigPath, result.PlanPath, result.StatePath)
	return nil
}

func runCheckFreshness(args []string) error {
	fs := flag.NewFlagSet("check-freshness", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	if err := fs.Parse(args); err != nil {
		return err
	}
	output, err := todo.CheckFreshness(*root)
	if err != nil {
		return err
	}
	fmt.Println(output)
	return nil
}

func runRefreshState(args []string) error {
	fs := flag.NewFlagSet("refresh-state", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	if err := fs.Parse(args); err != nil {
		return err
	}
	output, err := todo.RefreshPlanState(*root)
	if err != nil {
		return err
	}
	if err := todo.Reindex(*root); err != nil {
		return err
	}
	fmt.Println(output)
	return nil
}

func runCurrent(args []string) error {
	fs := flag.NewFlagSet("current", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	complete := fs.Bool("complete", false, "complete the active current action")
	evidence := fs.String("evidence", "", "completion evidence when --complete is set")
	date := fs.String("date", time.Now().Format(time.DateOnly), "completion date in YYYY-MM-DD")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.Current(*root, todo.CurrentOptions{
		Complete: *complete,
		Evidence: *evidence,
		Date:     *date,
	})
	if err != nil {
		return err
	}

	if result.Completed {
		fmt.Printf("status: completed\n")
		fmt.Printf("action: %s\n", result.Title)
		fmt.Printf("detail: %s\n", result.DetailFile)
		fmt.Printf("freshness: FRESH\n")
		return nil
	}
	fmt.Printf("status: current\n")
	fmt.Printf("action: %s\n", result.Title)
	fmt.Printf("detail: %s\n", result.DetailFile)
	fmt.Printf("freshness: FRESH\n")
	return nil
}

func runCreate(args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	title := fs.String("title", "", "action title")
	evidence := fs.String("evidence", "", "evidence or reason for the action")
	nextCheck := fs.String("next-check", "", "concrete next check for the action")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.Create(*root, todo.CreateOptions{
		Title:     *title,
		Evidence:  *evidence,
		NextCheck: *nextCheck,
	})
	if err != nil {
		return err
	}

	fmt.Printf("status: created\n")
	fmt.Printf("action: %s\n", result.Title)
	fmt.Printf("detail: %s\n", result.DetailFile)
	fmt.Printf("freshness: FRESH\n")
	return nil
}

func runSwitch(args []string) error {
	fs := flag.NewFlagSet("switch", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	to := fs.String("to", "", "backlog action to promote by rank, slug, filename, link, or title")
	demoteStatus := fs.String("demote-status", "not_started", "status for the previous current action: not_started or paused")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.Switch(*root, todo.SwitchOptions{
		Target:       *to,
		DemoteStatus: *demoteStatus,
	})
	if err != nil {
		return err
	}

	fmt.Printf("status: switched\n")
	fmt.Printf("promoted: %s\n", result.PromotedTitle)
	fmt.Printf("demoted: %s\n", result.DemotedTitle)
	fmt.Printf("current: %s\n", result.CurrentFile)
	fmt.Printf("backlog: %s\n", result.BacklogFile)
	fmt.Printf("freshness: FRESH\n")
	return nil
}

func runModifyCurrent(args []string) error {
	fs := flag.NewFlagSet("modify-current", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	title := fs.String("title", "", "replacement current action title")
	evidence := fs.String("evidence", "", "replacement evidence section")
	nextCheck := fs.String("next-check", "", "replacement next check section")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.ModifyCurrent(*root, todo.ModifyCurrentOptions{
		Title:     *title,
		Evidence:  *evidence,
		NextCheck: *nextCheck,
	})
	if err != nil {
		return err
	}

	fmt.Printf("status: modified\n")
	fmt.Printf("action: %s\n", result.Title)
	fmt.Printf("detail: %s\n", result.DetailFile)
	fmt.Printf("freshness: FRESH\n")
	return nil
}

func runDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	target := fs.String("target", "", "backlog action to delete by rank, slug, filename, link, or title")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.Delete(*root, todo.DeleteOptions{Target: *target})
	if err != nil {
		return err
	}

	fmt.Printf("status: deleted\n")
	fmt.Printf("action: %s\n", result.DeletedTitle)
	fmt.Printf("detail: %s\n", result.DetailFile)
	fmt.Printf("freshness: FRESH\n")
	return nil
}

func runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	if err := fs.Parse(args); err != nil {
		return err
	}

	result, err := todo.List(*root)
	if err != nil {
		return err
	}

	fmt.Printf("status: list\n")
	printActionSection("current", result.Current)
	printActionSection("backlog", result.Backlog)
	printActionSection("completed", result.Completed)
	fmt.Printf("freshness: %s\n", freshnessStatus(result.Freshness))
	return nil
}

func printActionSection(name string, items []todo.ListItem) {
	fmt.Printf("%s:\n", name)
	if len(items) == 0 {
		fmt.Printf("- none\n")
		return
	}
	for _, item := range items {
		fmt.Printf("- `%s` %s\n", item.Status, item.Title)
	}
}

func freshnessStatus(output string) string {
	fields := strings.Fields(output)
	if len(fields) == 0 {
		return output
	}
	return strings.TrimSuffix(fields[0], ":")
}

func runIndex(args []string) error {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := todo.Reindex(*root); err != nil {
		return err
	}
	fmt.Printf("indexed todo search data (%s)\n", todo.TodoIndexPath(*root))
	return nil
}

func runSearch(args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	root := fs.String("root", ".", "repository root containing .living-plan")
	query := fs.String("query", "", "search query")
	status := fs.String("status", "", "optional status filter")
	limit := fs.Int("limit", 20, "maximum results")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *query == "" && fs.NArg() > 0 {
		*query = strings.Join(fs.Args(), " ")
	}
	results, err := todo.Search(*root, todo.SearchOptions{
		Query:  *query,
		Status: *status,
		Limit:  *limit,
	})
	if err != nil {
		return err
	}
	fmt.Printf("# Search Results\n\nquery: %s\ncount: %d\n\n", *query, len(results))
	for _, result := range results {
		fmt.Printf("- `%s` `%s` %s\n", result.ID, result.Status, result.Title)
		fmt.Printf("  - rank: %s\n", result.Rank)
		fmt.Printf("  - file: %s\n", result.DetailFile)
		if result.NextCheck != "" {
			fmt.Printf("  - next_check: %s\n", oneLine(result.NextCheck))
		}
	}
	return nil
}

func oneLine(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	if len(value) > 160 {
		return value[:157] + "..."
	}
	return value
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  todo init [--scope NAME] [--store-root DIR] [--sensitive-path PATH] [--agent-link-path PATH] [--root DIR]
  todo current [--complete --evidence TEXT] [--date YYYY-MM-DD] [--root DIR]
  todo create --title TITLE [--evidence TEXT] [--next-check TEXT] [--root DIR]
  todo switch --to TARGET [--demote-status not_started|paused] [--root DIR]
  todo modify-current [--title TITLE] [--evidence TEXT] [--next-check TEXT] [--root DIR]
  todo delete --target TARGET [--root DIR]
  todo list [--root DIR]
  todo index [--root DIR]
  todo search [--query TEXT] [--status STATUS] [--limit N] [--root DIR]
  todo check-freshness [--root DIR]
  todo refresh-state [--root DIR]`)
}
