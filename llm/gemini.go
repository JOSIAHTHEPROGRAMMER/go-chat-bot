package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GeminiRequest struct {
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func CallGemini(prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	model := os.Getenv("GEMINI_MODEL")

	if apiKey == "" || model == "" {
		return "", fmt.Errorf("missing GEMINI_API_KEY or GEMINI_MODEL in env")
	}

	// Build request JSON
	reqBody := GeminiRequest{
		Contents: []struct {
			Role  string "json:\"role\""
			Parts []struct {
				Text string "json:\"text\""
			} "json:\"parts\""
		}{
			{
				Role: "user",
				Parts: []struct {
					Text string "json:\"text\""
				}{
					{Text: prompt},
				},
			},
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var geminiRes GeminiResponse
	json.NewDecoder(res.Body).Decode(&geminiRes)

	if len(geminiRes.Candidates) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	return geminiRes.Candidates[0].Content.Parts[0].Text, nil
}
