package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/oauth2"
)

func main() {
	githubToken := getEnv("INPUT_GITHUB_TOKEN", "INPUT_GITHUB-TOKEN", "GH_TOKEN")
	githubAPIBaseURL := getEnv("INPUT_GITHUB_API_BASE_URL", "INPUT_GITHUB-API-BASE-URL", "GH_API_BASE_URL")
	openaiAPIKey := getEnv("INPUT_OPENAI_API_KEY", "INPUT_OPENAI-API-KEY", "OPENAI_API_KEY")
	openaiModel := getEnv("INPUT_OPENAI_MODEL", "INPUT_OPENAI-MODEL", "OPENAI_MODEL")
	openaiBaseURL := getEnv("INPUT_OPENAI_BASE_URL", "INPUT_OPENAI-BASE-URL", "OPENAI_BASE_URL")
	repoFullName := os.Getenv("GITHUB_REPOSITORY")
	eventPath := os.Getenv("GITHUB_EVENT_PATH")

	if openaiModel == "" {
		openaiModel = openai.GPT3Dot5Turbo
	}

	if githubToken == "" || openaiAPIKey == "" || repoFullName == "" || eventPath == "" {
		fmt.Println("::error::Missing required environment variables")
		os.Exit(1)
	}

	repoParts := strings.Split(repoFullName, "/")
	if len(repoParts) != 2 {
		fmt.Printf("::error::Invalid GITHUB_REPOSITORY format: %s\n", repoFullName)
		os.Exit(1)
	}
	owner, repo := repoParts[0], repoParts[1]

	prNumber, err := getPrNumber(eventPath)
	if err != nil {
		fmt.Printf("::error::%v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	var ghClient *github.Client
	if githubAPIBaseURL != "" && githubAPIBaseURL != "https://api.github.com" {
		ghClient, _ = github.NewClient(tc).WithEnterpriseURLs(githubAPIBaseURL, githubAPIBaseURL)
	} else {
		ghClient = github.NewClient(tc)
	}

	// OpenAI Client
	config := openai.DefaultConfig(openaiAPIKey)
	if openaiBaseURL != "" {
		config.BaseURL = openaiBaseURL
	}
	aiClient := openai.NewClientWithConfig(config)

	// Check for label
	pr, _, err := ghClient.PullRequests.Get(ctx, owner, repo, prNumber)
	if err != nil {
		fmt.Printf("::error::Error fetching PR: %v\n", err)
		os.Exit(1)
	}

	hasLabel := false
	for _, l := range pr.Labels {
		if l.GetName() == "ai-describe" {
			hasLabel = true
			break
		}
	}

	if !hasLabel {
		fmt.Println("::info::Label 'ai-describe' not found. Skipping.")
		return
	}

	// Fetch changes
	files, _, err := ghClient.PullRequests.ListFiles(ctx, owner, repo, prNumber, nil)
	if err != nil {
		fmt.Printf("::error::Error fetching PR files: %v\n", err)
		os.Exit(1)
	}

	var changes strings.Builder
	for _, file := range files {
		changes.WriteString(fmt.Sprintf("File: %s\nChanges:\n%s\n", file.GetFilename(), file.GetPatch()))
	}

	// Fetch suggestions
	suggestions, err := fetchAiSuggestions(ctx, aiClient, changes.String(), openaiModel)
	if err != nil {
		fmt.Printf("::error::Error fetching AI suggestions: %v\n", err)
		os.Exit(1)
	}

	// Update PR Body
	newBody := fmt.Sprintf("%s\n\n## PR Auto Describe\n%s", pr.GetBody(), suggestions)
	_, _, err = ghClient.PullRequests.Edit(ctx, owner, repo, prNumber, &github.PullRequest{
		Body: github.String(newBody),
	})
	if err != nil {
		fmt.Printf("::error::Error updating PR body: %v\n", err)
		// Fallback to comment if body update fails
		postComment(ctx, ghClient, owner, repo, prNumber, suggestions)
	} else {
		fmt.Printf("::info::Successfully updated PR #%d description.\n", prNumber)
	}
}

func getPrNumber(eventPath string) (int, error) {
	data, err := os.ReadFile(eventPath)
	if err != nil {
		return 0, fmt.Errorf("error reading event file: %w", err)
	}

	var event struct {
		PullRequest struct {
			Number int `json:"number"`
		} `json:"pull_request"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return 0, fmt.Errorf("error unmarshaling event data: %w", err)
	}

	if event.PullRequest.Number == 0 {
		return 0, fmt.Errorf("pull request number not found in event data")
	}

	return event.PullRequest.Number, nil
}

func fetchAiSuggestions(ctx context.Context, client *openai.Client, prChanges string, model string) (string, error) {
	prompt := generatePrompt(prChanges)

	resp, err := client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			Temperature: 0.7,
			MaxTokens:   300,
		},
	)

	if err != nil {
		return "", err
	}

	return resp.Choices[0].Message.Content, nil
}

func generatePrompt(prChanges string) string {
	var prompt strings.Builder
	prompt.WriteString("We've made several updates in this pull request and would like to generate a list of changes. ")
	prompt.WriteString("The purpose is to summarize the pull request for easy understanding. ")
	prompt.WriteString("We are not interested in code reviews at this time.\n\n")

	prompt.WriteString("Below are the changes in this pull request:\n")
	prompt.WriteString("```\n")
	prompt.WriteString(prChanges)
	prompt.WriteString("```\n\n")

	prompt.WriteString("Please generate a concise list of changes, categorizing each by its type (e.g., 'Refactor', 'Bug Fix', 'Optimization').\n\n")

	prompt.WriteString("Structure your list of changes like this:\n")
	prompt.WriteString("[Short description of the whole changes]\n\n")
	prompt.WriteString("1. **Type**: Refactor\n   **Description**: [Short description of the change]\n")
	prompt.WriteString("2. **Type**: Optimization\n   **Description**: [Short description of the change]\n")
	prompt.WriteString("...")

	return prompt.String()
}

func postComment(ctx context.Context, client *github.Client, owner, repo string, prNumber int, suggestions string) {
	commentBody := fmt.Sprintf("**PR Auto Describe:**\n\n%s", suggestions)
	_, _, err := client.Issues.CreateComment(ctx, owner, repo, prNumber, &github.IssueComment{
		Body: github.String(commentBody),
	})
	if err != nil {
		fmt.Printf("::error::Error posting comment: %v\n", err)
	} else {
		fmt.Printf("::info::Successfully posted comment to PR #%d.\n", prNumber)
	}
}

func getEnv(keys ...string) string {
	for _, key := range keys {
		if val := os.Getenv(key); val != "" {
			return val
		}
	}
	return ""
}
