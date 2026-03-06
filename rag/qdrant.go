package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
)

// qdrantClient holds connection config for the Qdrant REST API.
type qdrantClient struct {
	url        string
	collection string
	apiKey     string
}

// newQdrantClient reads config from env and returns a client.
func newQdrantClient() *qdrantClient {
	return &qdrantClient{
		url:        os.Getenv("QDRANT_URL"),
		collection: os.Getenv("QDRANT_COLLECTION"),
		apiKey:     os.Getenv("QDRANT_API_KEY"),
	}
}

// do executes an HTTP request against the Qdrant API.
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

// EnsureCollection creates the collection if it does not already exist.
// Vector size 3072 matches gemini-embedding-001 output dimensions.
func (c *qdrantClient) EnsureCollection() error {
	path := fmt.Sprintf("/collections/%s", c.collection)

	// Check if collection exists
	res, err := c.do("GET", path, nil)
	if err != nil {
		return err
	}
	res.Body.Close()

	if res.StatusCode == http.StatusOK {
		return nil // already exists
	}

	// Create it with cosine distance to match our previous similarity metric
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

	return nil
}

// Upsert stores or updates a single doc as a Qdrant point.
// Uses a hash of the path as the point ID for stable, idempotent writes.
func (c *qdrantClient) Upsert(doc Doc) error {
	payload := map[string]any{
		"points": []map[string]any{
			{
				"id":      pathToID(doc.Path),
				"vector":  doc.Embedding,
				"payload": map[string]string{"path": doc.Path, "content": doc.Content},
			},
		},
	}

	path := fmt.Sprintf("/collections/%s/points", c.collection)
	res, err := c.do("PUT", path, payload)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant upsert failed: status %d", res.StatusCode)
	}

	return nil
}

// Search returns the top-k most similar docs to the query vector.
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
				Path    string `json:"path"`
				Content string `json:"content"`
			} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	docs := make([]Doc, 0, len(result.Result))
	for _, r := range result.Result {
		docs = append(docs, Doc{
			Path:    r.Payload.Path,
			Content: r.Payload.Content,
		})
	}

	return docs, nil
}

// Scroll retrieves all docs in the collection without a query vector.
// Used by StoreAll() to support the tools package.
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

	var result struct {
		Result struct {
			Points []struct {
				Payload struct {
					Path    string `json:"path"`
					Content string `json:"content"`
				} `json:"payload"`
			} `json:"points"`
		} `json:"result"`
	}

	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	docs := make([]Doc, 0, len(result.Result.Points))
	for _, p := range result.Result.Points {
		docs = append(docs, Doc{
			Path:    p.Payload.Path,
			Content: p.Payload.Content,
		})
	}

	return docs, nil
}

// pathToID hashes a repo path to a stable uint64 for use as a Qdrant point ID.
func pathToID(path string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(path))
	return h.Sum64()
}
