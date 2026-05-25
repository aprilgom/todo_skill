package todo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type planState struct {
	SchemaVersion   int    `json:"schema_version"`
	Scope           string `json:"scope"`
	PlanKind        string `json:"plan_kind"`
	PlanPath        string `json:"plan_path"`
	SensitivePath   string `json:"sensitive_path"`
	PlanUpdatedAt   string `json:"plan_updated_at"`
	PlanBranch      string `json:"plan_branch"`
	PlanBaseRef     string `json:"plan_base_ref"`
	TrackedTreeHash string `json:"tracked_tree_hash"`
	DirtyStatusHash string `json:"dirty_status_hash"`
	PlanHash        string `json:"plan_hash"`
}

func CheckFreshness(root string) (string, error) {
	paths, cfg, err := loadPlanConfig(root)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(cfg.PlanPath); errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("MISSING_PLAN: %s", cfg.PlanPath)
	} else if err != nil {
		return "", err
	}
	data, err := os.ReadFile(cfg.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("MISSING_STATE: %s", cfg.StatePath)
	}
	if err != nil {
		return "", err
	}
	var state planState
	if err := json.Unmarshal(data, &state); err != nil {
		return "", err
	}
	current, err := buildPlanState(paths.root, cfg)
	if err != nil {
		return "", err
	}
	var failures []string
	if state.PlanBranch != current.PlanBranch {
		failures = append(failures, fmt.Sprintf("STALE_BRANCH: state=%s current=%s", state.PlanBranch, current.PlanBranch))
	}
	if state.PlanBaseRef != current.PlanBaseRef {
		parent, _ := gitOutput(paths.root, "rev-parse", "HEAD^")
		if strings.TrimSpace(parent) != state.PlanBaseRef {
			failures = append(failures, fmt.Sprintf("STALE_HEAD: state=%s current=%s", state.PlanBaseRef, current.PlanBaseRef))
		}
	}
	if state.TrackedTreeHash != current.TrackedTreeHash {
		failures = append(failures, "STALE_TRACKED_TREE")
	}
	if state.DirtyStatusHash != current.DirtyStatusHash {
		failures = append(failures, "STALE_WORKTREE")
	}
	if state.PlanHash != current.PlanHash {
		failures = append(failures, "STALE_PLAN_HASH")
	}
	if len(failures) > 0 {
		return "", errors.New(strings.Join(failures, "\n"))
	}
	return fmt.Sprintf("FRESH: scope=%s plan=%s head=%s", state.Scope, cfg.PlanPath, shortRef(current.PlanBaseRef)), nil
}

func RefreshPlanState(root string) (string, error) {
	paths, cfg, err := loadPlanConfig(root)
	if err != nil {
		return "", err
	}
	state, err := buildPlanState(paths.root, cfg)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.StatePath), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')
	if err := os.WriteFile(cfg.StatePath, data, 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("refreshed %s for %s at %s", cfg.StatePath, cfg.Scope, shortRef(state.PlanBaseRef)), nil
}

func buildPlanState(repoRoot string, cfg planConfig) (planState, error) {
	branch, err := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		branch, err = gitOutput(repoRoot, "symbolic-ref", "--short", "HEAD")
		if err != nil {
			return planState{}, err
		}
	}
	head, err := gitOutput(repoRoot, "rev-parse", "HEAD")
	if err != nil {
		if isUnbornHead(repoRoot) {
			head = "UNBORN"
		} else {
			return planState{}, err
		}
	}
	tracked, err := gitOutput(repoRoot, "ls-files", "-s", "--", cfg.SensitivePath)
	if err != nil {
		return planState{}, err
	}
	status, err := gitOutput(repoRoot, "status", "--porcelain", "--", cfg.SensitivePath)
	if err != nil {
		return planState{}, err
	}
	planHash, err := fileSHA256(cfg.PlanPath)
	if err != nil {
		return planState{}, err
	}
	return planState{
		SchemaVersion:   1,
		Scope:           cfg.Scope,
		PlanKind:        cfg.Kind,
		PlanPath:        cfg.PlanPath,
		SensitivePath:   cfg.SensitivePath,
		PlanUpdatedAt:   time.Now().UTC().Format(time.RFC3339Nano),
		PlanBranch:      strings.TrimSpace(branch),
		PlanBaseRef:     strings.TrimSpace(head),
		TrackedTreeHash: shaText(filterGitLines(tracked, cfg)),
		DirtyStatusHash: shaText(filterGitLines(status, cfg)),
		PlanHash:        planHash,
	}, nil
}

func isUnbornHead(repoRoot string) bool {
	if _, err := gitOutput(repoRoot, "rev-parse", "--verify", "HEAD"); err == nil {
		return false
	}
	if _, err := gitOutput(repoRoot, "symbolic-ref", "--quiet", "HEAD"); err != nil {
		return false
	}
	return true
}

func gitOutput(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

func filterGitLines(value string, cfg planConfig) string {
	stateRel := filepath.ToSlash(cfg.StatePath)
	planRel := filepath.ToSlash(cfg.PlanPath)
	var lines []string
	for _, line := range strings.Split(value, "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(line, stateRel) || strings.Contains(line, planRel) {
			continue
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n") + "\n"
}

func shaText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func shortRef(ref string) string {
	if len(ref) <= 12 {
		return ref
	}
	return ref[:12]
}
