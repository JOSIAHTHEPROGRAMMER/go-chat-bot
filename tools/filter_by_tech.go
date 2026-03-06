package tools

import (
	"fmt"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/portfolio-backend/rag"
)

// FilterByTechTool scans all docs and returns those that mention the given technology.
// Used for questions like "what projects use JavaScript?" or "show me Go projects".
type FilterByTechTool struct{}

func (t *FilterByTechTool) Name() string { return "filter_by_tech" }

func (t *FilterByTechTool) Run(input string) (string, error) {
	tech := strings.TrimSpace(strings.ToLower(input))

	var matches []string
	for _, doc := range rag.StoreAll() {
		if strings.Contains(strings.ToLower(doc.Content), tech) {
			matches = append(matches, fmt.Sprintf("File: %s\n%s", doc.Path, doc.Content))
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("no projects found mentioning %q", input)
	}

	return strings.Join(matches, "\n\n"), nil
}
