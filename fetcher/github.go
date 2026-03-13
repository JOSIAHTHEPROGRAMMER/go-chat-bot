package fetcher

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Doc struct {
	Path      string
	Content   string
	Languages map[string]int
}

// buildIgnoreList builds the ignored repos set from the IGNORED_REPOS env var.
// Format: comma-separated repo names e.g. "my-repo,another-repo"
// Forks are always skipped regardless of this list.
func buildIgnoreList() map[string]bool {
	ignored := make(map[string]bool)
	if val := os.Getenv("IGNORED_REPOS"); val != "" {
		for _, name := range strings.Split(val, ",") {
			name = strings.TrimSpace(name)
			if name != "" {
				ignored[name] = true
			}
		}
	}
	return ignored
}

// FetchREADMEs is the primary entry point used by main.go.
func FetchREADMEs() ([]Doc, error) {
	return FetchAllReadmes()
}

// FetchAllReadmes fetches all README files and language stats from GitHub.
// Skips forks and any repos listed in IGNORED_REPOS.
func FetchAllReadmes() ([]Doc, error) {
	username := os.Getenv("GITHUB_USERNAME")
	token := os.Getenv("GITHUB_TOKEN")

	if username == "" || token == "" {
		return nil, fmt.Errorf("missing GITHUB_USERNAME or GITHUB_TOKEN in .env")
	}

	ignoredRepos := buildIgnoreList()
	client := &http.Client{}

	repos, err := fetchAllRepos(client, username, token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repos: %w", err)
	}

	fmt.Printf("Found %d repos total\n", len(repos))

	var docs []Doc
	for _, repo := range repos {
		if repo.Fork || ignoredRepos[repo.Name] {
			fmt.Printf("skipping %s\n", repo.Name)
			continue
		}

		readme, err := fetchReadme(client, username, repo.Name, token)
		if err != nil {
			fmt.Printf("no README for %s: %v\n", repo.Name, err)
			continue
		}

		languages, err := fetchLanguages(client, username, repo.Name, token)
		if err != nil {
			fmt.Printf("no languages for %s: %v\n", repo.Name, err)
			languages = map[string]int{}
		}

		docs = append(docs, Doc{
			Path:      repo.Name,
			Content:   readme,
			Languages: languages,
		})
	}

	fmt.Printf("Fetched %d READMEs from GitHub\n", len(docs))
	return docs, nil
}

// fetchAllRepos pages through the GitHub repo list until all repos are retrieved.
// GitHub caps each page at 100 so this handles accounts with more than 100 repos.
func fetchAllRepos(client *http.Client, username, token string) ([]struct {
	Name string `json:"name"`
	Fork bool   `json:"fork"`
}, error) {
	var all []struct {
		Name string `json:"name"`
		Fork bool   `json:"fork"`
	}

	page := 1
	for {
		url := fmt.Sprintf(
			"https://api.github.com/users/%s/repos?per_page=100&page=%d",
			username, page,
		)
		req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
		req.Header.Set("Authorization", "token "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		var repos []struct {
			Name string `json:"name"`
			Fork bool   `json:"fork"`
		}
		json.NewDecoder(resp.Body).Decode(&repos)
		resp.Body.Close()

		if len(repos) == 0 {
			break
		}
		all = append(all, repos...)
		page++
	}

	return all, nil
}

// fetchReadme retrieves and decodes the README for a single repo.
func fetchReadme(client *http.Client, username, repo, token string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", username, repo)

	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var data struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	raw := strings.ReplaceAll(data.Content, "\n", "")
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

// fetchLanguages retrieves the language breakdown for a single repo.
// Returns a map of language name to bytes e.g. {"JavaScript": 45123, "CSS": 12400}
func fetchLanguages(client *http.Client, username, repo, token string) (map[string]int, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/languages", username, repo)

	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	req.Header.Set("Authorization", "token "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var languages map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&languages); err != nil {
		return nil, err
	}

	return languages, nil
}
