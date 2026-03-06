package tools

import (
	"fmt"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/rag"
)

// techAliases maps a canonical technology name to GitHub language names that indicate its use.
// GitHub uses title case e.g. "JavaScript", "TypeScript", "Python".
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
// Returns project names with README snippets so the LLM can describe what each project does.
type FilterByTechTool struct{}

func (t *FilterByTechTool) Name() string { return "filter_by_tech" }

func (t *FilterByTechTool) Run(input string) (string, error) {
	tech := strings.TrimSpace(strings.ToLower(input))
	githubLangs := techAliases[tech]

	var matchedDocs []rag.Doc
	for _, doc := range rag.StoreAll() {
		if matchesTech(doc, tech, githubLangs) {
			matchedDocs = append(matchedDocs, doc)
		}
	}

	if len(matchedDocs) == 0 {
		return "", fmt.Errorf("no projects found using %q", input)
	}

	// Build context with name + first 800 chars of README per match.
	// 800 chars gives the LLM enough to describe what each project does
	// without blowing the context window when many projects match.
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Projects that use %s (%d total):\n\n", input, len(matchedDocs)))

	for _, doc := range matchedDocs {
		sb.WriteString(fmt.Sprintf("### %s\n", doc.Path))
		content := strings.TrimSpace(doc.Content)
		if len(content) > 800 {
			content = content[:800] + "..."
		}
		sb.WriteString(content)
		sb.WriteString("\n\n")
	}

	// Instruction injected directly into prompt context to reinforce accuracy
	sb.WriteString("IMPORTANT: Only reference the projects listed above. Describe each one using the context provided.")
	return sb.String(), nil
}

// matchesTech returns true if a doc uses the given technology.
// Checks GitHub language stats first (exact), then falls back to README keyword scan.
func matchesTech(doc rag.Doc, tech string, githubLangs []string) bool {
	// Primary check: GitHub language stats are authoritative
	if len(doc.Languages) > 0 {
		for repoLang := range doc.Languages {
			repoLangLower := strings.ToLower(repoLang)
			for _, lang := range githubLangs {
				if repoLangLower == strings.ToLower(lang) {
					return true
				}
			}
		}
		// If we have language data and nothing matched, trust it
		return false
	}

	// Fallback: scan README content if no language data is available
	content := strings.ToLower(doc.Content)
	keywords := append(githubLangs, tech)
	for _, kw := range keywords {
		if strings.Contains(content, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}
