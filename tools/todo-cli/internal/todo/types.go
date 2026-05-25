package todo

type CreateOptions struct {
	Title     string
	Evidence  string
	NextCheck string
}

type CreateResult struct {
	DetailFile string
	Title      string
	Rank       string
}

type SwitchOptions struct {
	Target       string
	DemoteStatus string
}

type SwitchResult struct {
	PromotedTitle string
	DemotedTitle  string
	CurrentFile   string
	BacklogFile   string
}

type ModifyCurrentOptions struct {
	Title     string
	Evidence  string
	NextCheck string
}

type ModifyCurrentResult struct {
	DetailFile string
	Title      string
}

type DeleteOptions struct {
	Target string
}

type DeleteResult struct {
	DeletedTitle string
	DetailFile   string
}

type CurrentOptions struct {
	Complete bool
	Evidence string
	Date     string
}

type CurrentResult struct {
	Title      string
	DetailFile string
	Completed  bool
}

type ListResult struct {
	Current   []ListItem
	Backlog   []ListItem
	Completed []ListItem
	Freshness string
}

type ListItem struct {
	Title  string
	Link   string
	Status string
}

type InitOptions struct {
	Scope         string
	StoreRoot     string
	SensitivePath string
	AgentLinkPath string
	NoGitHook     bool
	NoCodexHook   bool
}

type InitResult struct {
	ConfigPath string
	PlanPath   string
	StatePath  string
}

type planPaths struct {
	root         string
	planDir      string
	current      string
	backlog      string
	completed    string
	completedDir string
	actionDir    string
	checkFresh   string
	refresh      string
}

type actionRef struct {
	Title  string
	Link   string
	Status string
	Line   string
}
