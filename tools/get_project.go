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
	// input is the project name as extracted by the planner
	name := strings.TrimSpace(strings.ToLower(input))

	for _, doc := range rag.StoreAll() {
		if strings.ToLower(doc.Path) == name {
			return fmt.Sprintf("File: %s\n%s", doc.Path, doc.Content), nil
		}
	}

	return "", fmt.Errorf("no project found with name %q", input)
}
