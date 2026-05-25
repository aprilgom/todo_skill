package todo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/index/scorch"
	"github.com/blevesearch/bleve/v2/mapping"
	searchquery "github.com/blevesearch/bleve/v2/search/query"
)

type SearchOptions struct {
	Query  string
	Status string
	Limit  int
}

type SearchResult struct {
	ID         string
	Title      string
	Status     string
	Rank       string
	Link       string
	DetailFile string
	Evidence   string
	NextCheck  string
	Score      float64
}

type indexedAction struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	Rank       string `json:"rank"`
	Link       string `json:"link"`
	DetailFile string `json:"detail_file"`
	Evidence   string `json:"evidence"`
	NextCheck  string `json:"next_check"`
	UpdatedAt  string `json:"updated_at"`
}

func TodoIndexPath(root string) string {
	paths, err := loadPlanPaths(root)
	if err == nil {
		return filepath.Join(paths.planDir, "todo.bleve")
	}
	return filepath.Join(root, ".living-plan", "todo.bleve")
}

func Reindex(root string) error {
	paths, err := loadPlanPaths(root)
	if err != nil {
		return err
	}
	return reindexPaths(paths)
}

func Search(root string, opts SearchOptions) ([]SearchResult, error) {
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	indexPath := TodoIndexPath(root)
	index, err := bleve.Open(indexPath)
	if errors.Is(err, bleve.ErrorIndexPathDoesNotExist) {
		if err := Reindex(root); err != nil {
			return nil, err
		}
		index, err = bleve.Open(indexPath)
	}
	if err != nil {
		return nil, err
	}
	defer index.Close()

	var query searchquery.Query = bleve.NewMatchAllQuery()
	if strings.TrimSpace(opts.Query) != "" {
		query = bleve.NewQueryStringQuery(opts.Query)
	}
	if strings.TrimSpace(opts.Status) != "" {
		statusQuery := bleve.NewTermQuery(opts.Status)
		statusQuery.SetField("status")
		query = bleve.NewConjunctionQuery(query, statusQuery)
	}

	request := bleve.NewSearchRequestOptions(query, opts.Limit, 0, false)
	request.Fields = []string{"id", "title", "status", "rank", "link", "detail_file", "evidence", "next_check"}
	response, err := index.Search(request)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(response.Hits))
	for _, hit := range response.Hits {
		results = append(results, SearchResult{
			ID:         fieldString(hit.Fields, "id"),
			Title:      fieldString(hit.Fields, "title"),
			Status:     fieldString(hit.Fields, "status"),
			Rank:       fieldString(hit.Fields, "rank"),
			Link:       fieldString(hit.Fields, "link"),
			DetailFile: fieldString(hit.Fields, "detail_file"),
			Evidence:   fieldString(hit.Fields, "evidence"),
			NextCheck:  fieldString(hit.Fields, "next_check"),
			Score:      hit.Score,
		})
	}
	return results, nil
}

func reindexPaths(paths planPaths) error {
	actions, err := collectIndexedActions(paths)
	if err != nil {
		return err
	}
	indexPath := TodoIndexPath(paths.root)
	tmpPath := indexPath + ".tmp"
	oldPath := indexPath + ".old"
	if err := os.RemoveAll(tmpPath); err != nil {
		return err
	}
	if err := os.RemoveAll(oldPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		return err
	}
	index, err := bleve.NewUsing(tmpPath, todoIndexMapping(), scorch.Name, scorch.Name, nil)
	if err != nil {
		return err
	}
	batch := index.NewBatch()
	for _, action := range actions {
		if err := batch.Index(action.ID, action); err != nil {
			index.Close()
			os.RemoveAll(tmpPath)
			return err
		}
	}
	if err := index.Batch(batch); err != nil {
		index.Close()
		os.RemoveAll(tmpPath)
		return err
	}
	if err := index.Close(); err != nil {
		os.RemoveAll(tmpPath)
		return err
	}
	if _, err := os.Stat(indexPath); err == nil {
		if err := os.Rename(indexPath, oldPath); err != nil {
			os.RemoveAll(tmpPath)
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		os.RemoveAll(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, indexPath); err != nil {
		if _, statErr := os.Stat(oldPath); statErr == nil {
			_ = os.Rename(oldPath, indexPath)
		}
		os.RemoveAll(tmpPath)
		return err
	}
	return os.RemoveAll(oldPath)
}

func todoIndexMapping() *mapping.IndexMappingImpl {
	mapping := bleve.NewIndexMapping()
	doc := bleve.NewDocumentMapping()
	for _, field := range []string{"title", "evidence", "next_check", "detail_file"} {
		doc.AddFieldMappingsAt(field, bleve.NewTextFieldMapping())
	}
	for _, field := range []string{"id", "status", "rank", "link"} {
		doc.AddFieldMappingsAt(field, bleve.NewKeywordFieldMapping())
	}
	mapping.DefaultMapping = doc
	mapping.DefaultAnalyzer = "standard"
	return mapping
}

func collectIndexedActions(paths planPaths) ([]indexedAction, error) {
	refs, err := allActionRefs(paths)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	actions := make([]indexedAction, 0, len(refs))
	for _, ref := range refs {
		if seen[ref.Link] {
			continue
		}
		seen[ref.Link] = true
		detail := detailPath(paths, ref)
		action, err := readIndexedAction(detail, ref)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func allActionRefs(paths planPaths) ([]actionRef, error) {
	var refs []actionRef
	for _, path := range []string{paths.current, paths.backlog, paths.completed} {
		fileRefs, err := readActionRefs(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		refs = append(refs, fileRefs...)
	}
	return refs, nil
}

func readIndexedAction(path string, ref actionRef) (indexedAction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return indexedAction{}, err
	}
	content := string(data)
	rank := detailLineValue(content, "- rank:")
	if rank == "" {
		rank = rankRE.FindString(strings.TrimSuffix(filepath.Base(ref.Link), ".md"))
	}
	status := ref.Status
	if status == "" {
		status = detailLineValue(content, "- status:")
	}
	info, err := os.Stat(path)
	if err != nil {
		return indexedAction{}, err
	}
	id := strings.TrimSuffix(filepath.Base(ref.Link), ".md")
	return indexedAction{
		ID:         id,
		Title:      ref.Title,
		Status:     status,
		Rank:       rank,
		Link:       ref.Link,
		DetailFile: path,
		Evidence:   detailSection(content, "## Evidence"),
		NextCheck:  detailSection(content, "## Next Check"),
		UpdatedAt:  info.ModTime().UTC().Format(time.RFC3339),
	}, nil
}

func detailLineValue(content, prefix string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}

func detailSection(content, heading string) string {
	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == heading {
			start = i + 1
			break
		}
	}
	if start == -1 {
		return ""
	}
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for i := start; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			end = i
			break
		}
	}
	return strings.TrimSpace(strings.Join(lines[start:end], "\n"))
}

func fieldString(fields map[string]any, key string) string {
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}
