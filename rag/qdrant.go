package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
	"os"
)

type qdrantClient struct {
	url        string
	collection string
	apiKey     string
}

func newQdrantClient() *qdrantClient {
	return &qdrantClient{
		url:        os.Getenv("QDRANT_URL"),
		collection: os.Getenv("QDRANT_COLLECTION"),
		apiKey:     os.Getenv("QDRANT_API_KEY"),
	}
}

func (c *qdrantClient) do(method, path string, body any) (*http.Response, error) {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, c.url+path, &buf)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("api-key", c.apiKey)
	}

	return http.DefaultClient.Do(req)
}

func (c *qdrantClient) EnsureCollection() error {
	path := fmt.Sprintf("/collections/%s", c.collection)
	//	fmt.Printf("Qdrant URL=%s Collection=%s\n", c.url, c.collection)

	res, err := c.do("GET", path, nil)
	if err != nil {
		return err
	}
	res.Body.Close()

	if res.StatusCode == http.StatusOK {
		fmt.Println("Qdrant collection already exists")
		return nil
	}

	payload := map[string]any{
		"vectors": map[string]any{
			"size":     3072,
			"distance": "Cosine",
		},
	}

	res, err = c.do("PUT", path, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create Qdrant collection: status %d", res.StatusCode)
	}

	fmt.Println("Qdrant collection created")
	return nil
}

func (c *qdrantClient) Upsert(doc Doc) error {
	payload := map[string]any{
		"points": []map[string]any{
			{
				"id":     pathToID(doc.Path),
				"vector": doc.Embedding,
				"payload": map[string]any{
					"path":      doc.Path,
					"content":   doc.Content,
					"languages": doc.Languages,
				},
			},
		},
	}

	path := fmt.Sprintf("/collections/%s/points?wait=true", c.collection)
	res, err := c.do("PUT", path, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant upsert failed: status %d body: %s", res.StatusCode, string(raw))
	}

	// Log first upsert response to confirm structure
	if doc.Path == "ai-study-extension-server" {
		fmt.Printf("Upsert response for %s: %s\n", doc.Path, string(raw))
	}

	return nil
}

func (c *qdrantClient) Search(vector []float32, k int) ([]Doc, error) {
	payload := map[string]any{
		"vector":       vector,
		"limit":        k,
		"with_payload": true,
	}

	path := fmt.Sprintf("/collections/%s/points/search", c.collection)
	res, err := c.do("POST", path, payload)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	var result struct {
		Result []struct {
			Payload struct {
				Path      string         `json:"path"`
				Content   string         `json:"content"`
				Languages map[string]int `json:"languages"`
			} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	docs := make([]Doc, 0, len(result.Result))
	for _, r := range result.Result {
		docs = append(docs, Doc{
			Path:      r.Payload.Path,
			Content:   r.Payload.Content,
			Languages: r.Payload.Languages,
		})
	}

	return docs, nil
}

func (c *qdrantClient) Scroll() ([]Doc, error) {
	payload := map[string]any{
		"limit":        1000,
		"with_payload": true,
		"with_vector":  false,
	}

	path := fmt.Sprintf("/collections/%s/points/scroll", c.collection)
	res, err := c.do("POST", path, payload)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("scroll read error: %w", err)
	}
	fmt.Printf("Scroll raw response (first 500 chars): %.500s\n", string(raw))

	var result struct {
		Result struct {
			Points []struct {
				Payload struct {
					Path      string         `json:"path"`
					Content   string         `json:"content"`
					Languages map[string]int `json:"languages"`
				} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}

	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("scroll decode error: %w", err)
	}

	docs := make([]Doc, 0, len(result.Result.Points))
	for _, p := range result.Result.Points {
		docs = append(docs, Doc{
			Path:      p.Payload.Path,
			Content:   p.Payload.Content,
			Languages: p.Payload.Languages,
		})
	}

	return docs, nil
}

func pathToID(path string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(path))
	return h.Sum64()
}
