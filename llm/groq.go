package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GroqRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

type GroqResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func CallGroq(prompt string) (string, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	model := os.Getenv("GROQ_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GROQ_API_KEY or GROQ_MODEL in env")
	}

	reqBody := GroqRequest{
		Model: model,
		Messages: []struct {
			Role    string "json:\"role\""
			Content string "json:\"content\""
		}{
			{"user", prompt},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", "https://api.groq.com/openai/v1/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var groqResponse GroqResponse
	json.NewDecoder(res.Body).Decode(&groqResponse)

	if len(groqResponse.Choices) == 0 {
		return "", fmt.Errorf("no response from Groq")
	}

	return groqResponse.Choices[0].Message.Content, nil
}
