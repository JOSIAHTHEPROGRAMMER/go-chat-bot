package tools

import (
	"fmt"
	"strings"

	"github.com/JOSIAHTHEPROGRAMMER/go-chat-bot/rag"
)

// GetProjectTool retrieves a single project's README by name.
// Faster and more precise than RAG when the user mentions a project by name.
type GetProjectTool struct{}

func (t *GetProjectTool) Name() string { return "get_project" }

func (t *GetProjectTool) Run(input string) (string, error) {
	input = strings.TrimSpace(input)

	// Primary check: exact match, case-insensitive
	for _, doc := range rag.StoreAll() {
		if strings.EqualFold(doc.Path, input) {
			return fmt.Sprintf("### %s\n\n%s", doc.Path, doc.Content), nil
		}
	}

	// Fuzzy fallback: check if input is a substring of a doc path or vice versa.
	// Handles cases like "Webora" matching "Webora-backend-server".
	inputLower := strings.ToLower(input)
	for _, doc := range rag.StoreAll() {
		if strings.Contains(strings.ToLower(doc.Path), inputLower) ||
			strings.Contains(inputLower, strings.ToLower(doc.Path)) {
			return fmt.Sprintf("### %s\n\n%s", doc.Path, doc.Content), nil
		}
	}

	return "", fmt.Errorf("no project found with name %q", input)
}
