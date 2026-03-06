package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type GeminiEmbedder struct{}

func (g *GeminiEmbedder) Embed(text string) ([]float32, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("missing GEMINI_API_KEY in env")
	}

	// Request body - model goes in the URL only, not the body
	reqBody := geminiEmbedRequest{
		Content: geminiEmbedContent{
			Parts: []geminiEmbedPart{{Text: text}},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	// gemini-embedding-001 replaces text-embedding-004 and outputs 3072 dimensions
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/gemini-embedding-001:embedContent?key=%s",
		apiKey,
	)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var out geminiEmbedResponse
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}

	if len(out.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding returned from Gemini")
	}

	return out.Embedding.Values, nil
}

// -- internal types --

type geminiEmbedRequest struct {
	Content geminiEmbedContent `json:"content"`
}

type geminiEmbedContent struct {
	Parts []geminiEmbedPart `json:"parts"`
}

type geminiEmbedPart struct {
	Text string `json:"text"`
}

type geminiEmbedResponse struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}
