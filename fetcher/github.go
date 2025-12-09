package fetcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type Doc struct {
	Path    string
	Content string
}

// FetchAllReadmes fetches all README files from GitHub and returns them in memory
func FetchAllReadmes() ([]Doc, error) {
	username := os.Getenv("GITHUB_USERNAME")
	token := os.Getenv("GITHUB_TOKEN")

	if username == "" || token == "" {
		return nil, fmt.Errorf("missing GITHUB_USERNAME or GITHUB_TOKEN in .env")
	}

	req, _ := http.NewRequestWithContext(
		context.Background(),
		"GET",
		"https://api.github.com/users/"+username+"/repos",
		nil,
	)
	req.Header.Set("Authorization", "token "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}
	defer resp.Body.Close()

	var repos []struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return nil, fmt.Errorf("failed to decode repo list: %w", err)
	}

	var docs []Doc

	for _, repo := range repos {
		url := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/readme",
			username,
			repo.Name,
		)

		rReq, _ := http.NewRequestWithContext(
			context.Background(),
			"GET",
			url,
			nil,
		)
		rReq.Header.Set("Authorization", "token "+token)

		rResp, err := client.Do(rReq)
		if err != nil {
			fmt.Println("error fetching:", repo.Name, err)
			continue
		}
		defer rResp.Body.Close()

		if rResp.StatusCode != 200 {
			fmt.Println("no README found for", repo.Name)
			continue
		}

		var data struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(rResp.Body).Decode(&data); err != nil {
			fmt.Println("error decoding README for", repo.Name, err)
			continue
		}

		decoded, err := base64.StdEncoding.DecodeString(data.Content)
		if err != nil {
			fmt.Println("base64 decode error for", repo.Name, err)
			continue
		}

		docs = append(docs, Doc{
			Path:    repo.Name,
			Content: string(decoded),
		})
	}

	fmt.Printf("Fetched %d READMEs from GitHub\n", len(docs))
	return docs, nil
}
