package todo

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var actionLineRE = regexp.MustCompile(`^\s*-\s+\[([^\]]+)\]\(([^)]+)\)\s+-\s+` + "`" + `([^` + "`" + `]+)` + "`" + `\s*$`)
var rankRE = regexp.MustCompile(`^(\d+)`)

func nextRank(actionDir string) (string, error) {
	entries, err := os.ReadDir(actionDir)
	if err != nil {
		return "", err
	}
	maxRank := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		match := rankRE.FindStringSubmatch(entry.Name())
		if len(match) == 0 {
			continue
		}
		rank, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if rank > maxRank {
			maxRank = rank
		}
	}
	return fmt.Sprintf("%03d", maxRank+1), nil
}

func slugify(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func appendBacklogRow(path, row string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	content := string(data)
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if !strings.HasSuffix(content, "\n\n") {
		content += "\n"
	}
	content += row
	return os.WriteFile(path, []byte(content), 0o644)
}

func readActionRefs(path string) ([]actionRef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var refs []actionRef
	for _, line := range strings.Split(string(data), "\n") {
		match := actionLineRE.FindStringSubmatch(line)
		if len(match) == 4 {
			refs = append(refs, actionRef{Title: match[1], Link: match[2], Status: match[3], Line: line})
		}
	}
	return refs, nil
}

func currentAction(path string) (actionRef, error) {
	refs, err := readActionRefs(path)
	if err != nil {
		return actionRef{}, err
	}
	if len(refs) != 1 {
		return actionRef{}, fmt.Errorf("current.md must contain exactly one action link, found %d", len(refs))
	}
	return refs[0], nil
}

func findAction(refs []actionRef, target string) (actionRef, bool) {
	target = strings.TrimSpace(strings.ToLower(target))
	for _, ref := range refs {
		linkBase := strings.TrimSuffix(filepath.Base(ref.Link), ".md")
		rank := rankRE.FindString(linkBase)
		candidates := []string{
			strings.ToLower(ref.Title),
			strings.ToLower(ref.Link),
			strings.ToLower(filepath.Base(ref.Link)),
			strings.ToLower(linkBase),
			strings.ToLower(rank),
		}
		sort.Strings(candidates)
		for _, candidate := range candidates {
			if candidate == target {
				return ref, true
			}
		}
	}
	return actionRef{}, false
}

func removeActionLink(path, link string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		refMatch := actionLineRE.FindStringSubmatch(line)
		if len(refMatch) == 4 && refMatch[2] == link {
			continue
		}
		lines = append(lines, line)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func setCurrent(path string, ref actionRef) error {
	content := fmt.Sprintf("# Current\n\n- [%s](%s) - `in_progress`\n", ref.Title, ref.Link)
	return os.WriteFile(path, []byte(content), 0o644)
}

func setCurrentEmpty(path string) error {
	return os.WriteFile(path, []byte("# Current\n\n"), 0o644)
}

func setDetailStatus(path, status string) error {
	return replaceDetailLine(path, "- status:", "- status: "+status)
}

func setDetailTitle(path, title string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "# ") {
		return fmt.Errorf("title line not found in %s", path)
	}
	lines[0] = "# " + title
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}

func replaceDetailSection(path, heading, body string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i
			break
		}
	}
	if start == -1 {
		return fmt.Errorf("%s section not found in %s", heading, path)
	}
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	replacement := []string{heading, "", body, ""}
	newLines := append([]string{}, lines[:start]...)
	newLines = append(newLines, replacement...)
	newLines = append(newLines, lines[end:]...)
	return os.WriteFile(path, []byte(strings.Join(newLines, "\n")), 0o644)
}

func replaceDetailLine(path, prefix, replacement string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), prefix) {
			lines[i] = replacement
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
		}
	}
	return fmt.Errorf("%s line not found in %s", prefix, path)
}
