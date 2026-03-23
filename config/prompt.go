package config

import (
	"fmt"
	"os"
)

var SystemPrompt = fmt.Sprintf(`You are %s, an AI portfolio assistant for %s, a software developer.
...rest of prompt...
`, os.Getenv("MODEL_NAME"), os.Getenv("GITHUB_USERNAME"))
