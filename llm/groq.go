package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GroqProvider struct{}

func (g *GroqProvider) Name() string { return "groq" }

// Complete sends the given prompt to Groq and returns the generated response text.
func (g *GroqProvider) Complete(prompt string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	model := os.Getenv("GROQ_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GROQ_API_KEY or GROQ_MODEL in env")
	}

	reqBody := groqRequest{
		Model: model,
		Messages: []groqMessage{
			{Role: "user", Content: prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var out groqResponse
	json.NewDecoder(res.Body).Decode(&out)

	if len(out.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}

	return out.Choices[0].Message.Content, nil
}

// -- internal types --

type groqRequest struct {
	Model    string        `json:"model"`
	Messages []groqMessage `json:"messages"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqResponse struct {
	Choices []struct {
		Message groqMessage `json:"message"`
	} `json:"choices"`
}
