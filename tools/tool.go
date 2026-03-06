package tools

import "fmt"

// Tool is the contract every tool must implement.
// Name() is used by the planner to dispatch to the right tool.
type Tool interface {
	Name() string
	Run(input string) (string, error)
}

// registry holds all available tools keyed by name.
var registry = map[string]Tool{
	"get_project":    &GetProjectTool{},
	"filter_by_tech": &FilterByTechTool{},
}

// Get returns the tool matching the given name.
func Get(name string) (Tool, error) {
	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	return t, nil
}
