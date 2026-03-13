package config

var CurrentModel string

func GetCurrentModel() string {
	return CurrentModel
}

func SetCurrentModel(m string) {
	CurrentModel = m
}

// SetActiveProvider sets the active LLM provider by name.
// Called at startup in main.go — e.g. SetActiveProvider("local").
func SetActiveProvider(name string) {
	CurrentModel = name
}
