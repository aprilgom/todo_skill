package todo

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type planConfig struct {
	Scope         string
	Kind          string
	PlanPath      string
	StatePath     string
	SensitivePath string
}

func loadPlanPaths(root string) (planPaths, error) {
	paths, _, err := loadPlanConfig(root)
	return paths, err
}

func loadPlanConfig(root string) (planPaths, planConfig, error) {
	if root == "" {
		root = "."
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return planPaths{}, planConfig{}, err
	}
	envPath := filepath.Join(absRoot, ".living-plan", "living-plan.env")
	cfg, err := readPlanConfig(envPath)
	if err != nil {
		return planPaths{}, planConfig{}, err
	}
	if !filepath.IsAbs(cfg.PlanPath) {
		cfg.PlanPath = filepath.Join(absRoot, cfg.PlanPath)
	}
	if !filepath.IsAbs(cfg.StatePath) {
		cfg.StatePath = filepath.Join(absRoot, cfg.StatePath)
	}
	if cfg.SensitivePath == "" {
		cfg.SensitivePath = "."
	}
	planDir := filepath.Dir(cfg.PlanPath)
	return planPaths{
		root:         absRoot,
		planDir:      planDir,
		current:      filepath.Join(planDir, "current.md"),
		backlog:      filepath.Join(planDir, "backlog.md"),
		completed:    filepath.Join(planDir, "completed.md"),
		completedDir: filepath.Join(planDir, "completed"),
		actionDir:    filepath.Join(planDir, "action"),
		checkFresh:   filepath.Join(absRoot, ".living-plan", "scripts", "check_plan_freshness.sh"),
		refresh:      filepath.Join(absRoot, ".living-plan", "scripts", "refresh_plan_state.sh"),
	}, cfg, nil
}

func readPlanConfig(envPath string) (planConfig, error) {
	file, err := os.Open(envPath)
	if err != nil {
		return planConfig{}, err
	}
	defer file.Close()

	values := map[string]string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if ok {
			values[strings.TrimSpace(key)] = strings.Trim(strings.TrimSpace(value), `"'`)
		}
	}
	if err := scanner.Err(); err != nil {
		return planConfig{}, err
	}
	cfg := planConfig{
		Scope:         values["PLAN_SCOPE"],
		Kind:          values["PLAN_KIND"],
		PlanPath:      values["PLAN_PATH"],
		StatePath:     values["STATE_PATH"],
		SensitivePath: values["SENSITIVE_PATH"],
	}
	if cfg.PlanPath == "" {
		return planConfig{}, fmt.Errorf("PLAN_PATH not found in %s", envPath)
	}
	if cfg.StatePath == "" {
		return planConfig{}, fmt.Errorf("STATE_PATH not found in %s", envPath)
	}
	return cfg, nil
}

func detailPath(paths planPaths, ref actionRef) string {
	return filepath.Join(paths.planDir, filepath.FromSlash(ref.Link))
}

func runFreshness(paths planPaths) error {
	_, err := CheckFreshness(paths.root)
	return err
}

func refreshAndCheck(paths planPaths) error {
	if _, err := RefreshPlanState(paths.root); err != nil {
		return err
	}
	if err := runFreshness(paths); err != nil {
		return err
	}
	return reindexPaths(paths)
}
