package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-git/go-git/v5"
)

type GitHubResponse struct {
	Data struct {
		Organization struct {
			Repositories struct {
				Nodes []struct {
					NameWithOwner string `json:"nameWithOwner"`
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					StartCursor string `json:"startCursor"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"repositories"`
		} `json:"organization"`
		RateLimit struct {
			Limit     int    `json:"limit"`
			Remaining int    `json:"remaining"`
			ResetAt   string `json:"resetAt"`
		} `json:"rateLimit"`
	} `json:"data"`
}

func cloneRepositories(repoName string) {
	repoURL := fmt.Sprintf("https://github.com/%s.git", repoName)
	// TODO: Ensure the local path is provided and valid by using a root directory.
	localPath := ""

	// Clone the repository
	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:      repoURL,
		Progress: os.Stdout,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Fatalf("Failed to clone repository: %v", err)
	}

	// Print the repository's worktree path
	worktree, err := repo.Worktree()
	if err != nil {
		log.Fatalf("Failed to get worktree: %v", err)
	}
	fmt.Println("Cloned repository to:", worktree.Filesystem.Root())
}

func main() {
	githubToken := "your_github_token_here"
	orgs := []string{"18f", "gsa"}

	for _, org := range orgs {
		cursor := ""

		for {
			query := fmt.Sprintf(`{
            organization(login: "%s") {
                repositories(first: 100, after: "%s") {
                    nodes{
                        nameWithOwner
                    }
                    pageInfo {
                        hasNextPage
                        startCursor
                        endCursor
                    }
                }
            }
        }`, org, cursor)

			jsonQuery, _ := json.Marshal(map[string]string{"query": query})
			request, err := http.NewRequest("POST", "http://api.github.com/graphql", bytes.NewBuffer(jsonQuery))
			if err != nil {
				panic(err)
			}
			request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", githubToken))

			client := &http.Client{Timeout: time.Second * 10}
			response, err := client.Do(request)
			if err != nil {
				panic(err)
			}
			defer response.Body.Close()

			jsonResponseData, _ := io.ReadAll(response.Body)

			var data GitHubResponse
			err = json.Unmarshal([]byte(jsonResponseData), &data)
			if err != nil {
				panic(err)
			}

			for _, repo := range data.Data.Organization.Repositories.Nodes {
				fmt.Println(repo.NameWithOwner)
				// TODO: Take the repo names and clone them to a root folder. Clone must be with complete history.
				cloneRepositories(repo.NameWithOwner)
			}

			if !data.Data.Organization.Repositories.PageInfo.HasNextPage {
				break
			}

			cursor = data.Data.Organization.Repositories.PageInfo.EndCursor

			// TODO: Change this validation to use percentage instead of absolute value.
			if data.Data.RateLimit.Remaining <= 20 {
				time.Sleep(time.Until(time.Unix(int64(data.Data.RateLimit.ResetAt), 0)))
			}
			time.Sleep(time.Second * 2) // Wait before making the next request to avoid hitting the rate limit.
		}
	}
}
