package tools

import (
	"fmt"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/rag"
)

// techAliases maps a canonical technology name to GitHub language names that indicate its use.
var techAliases = map[string][]string{
	"javascript": {"JavaScript", "TypeScript", "JSX", "TSX", "JS"},
	"typescript": {"TypeScript", "TSX"},
	"python":     {"Python", "Jupyter Notebook"},
	"go":         {"Go"},
	"react":      {"JavaScript", "TypeScript", "JSX", "TSX"},
	"css":        {"CSS", "SCSS", "Sass"},
	"html":       {"HTML"},
	"sql":        {"PLpgSQL", "SQL"},
	"shell":      {"Shell", "Bash"},
	"java":       {"Java"},
	"kotlin":     {"Kotlin"},
	"swift":      {"Swift"},
	"rust":       {"Rust"},
	"c++":        {"C++"},
	"c":          {"C"},
}

// FilterByTechTool checks GitHub language stats first, then falls back to README keyword scan.
type FilterByTechTool struct{}

func (t *FilterByTechTool) Name() string { return "filter_by_tech" }

func (t *FilterByTechTool) Run(input string) (string, error) {
	tech := strings.TrimSpace(strings.ToLower(input))
	githubLangs := techAliases[tech]

	var matches []string

	for _, doc := range rag.StoreAll() {
		fmt.Printf("doc: %s languages: %v\n", doc.Path, doc.Languages)

		if matchesTech(doc, tech, githubLangs) {
			matches = append(matches, doc.Path)
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf(
			"No projects in the repository dataset were confirmed to use %s based on GitHub language data.",
			input,
		), nil
	}

	return fmt.Sprintf(
		"Projects confirmed to use %s based on GitHub language data (%d total):\n- %s\n\nIMPORTANT: Only reference these exact project names in your answer. Do not add any projects not on this list.",
		input,
		len(matches),
		strings.Join(matches, "\n- "),
	), nil
}

// matchesTech returns true if a doc uses the given technology.
// Checks GitHub language stats first, then falls back to README keyword scan.
func matchesTech(doc rag.Doc, tech string, githubLangs []string) bool {

	// Primary check: GitHub language stats
	if len(doc.Languages) > 0 {
		for repoLang := range doc.Languages {
			repoLangLower := strings.ToLower(repoLang)

			for _, lang := range githubLangs {
				if repoLangLower == strings.ToLower(lang) {
					return true
				}
			}
		}

		return false
	}

	// Fallback: scan README content if language data isn't available
	content := strings.ToLower(doc.Content)
	keywords := append(githubLangs, tech)

	for _, kw := range keywords {
		if strings.Contains(content, strings.ToLower(kw)) {
			return true
		}
	}

	return false
}
