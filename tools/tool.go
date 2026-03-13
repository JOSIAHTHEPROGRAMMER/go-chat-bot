package tools

import "fmt"

// Tool is the contract every tool must implement.
type Tool interface {
	Name() string
	Run(input string) (string, error)
}

// registry holds all available tools keyed by name.
var registry = map[string]Tool{}

// Register adds or replaces a tool in the registry by its Name().
// Called at startup in main.go for each tool.
func Register(t Tool) {
	registry[t.Name()] = t
}

// Get returns the tool matching the given name.
func Get(name string) (Tool, error) {
	t, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %q", name)
	}
	return t, nil
}
