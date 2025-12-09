package config

var CurrentModel string

func GetCurrentModel() string {
	return CurrentModel
}

func SetCurrentModel(m string) {
	CurrentModel = m
}
